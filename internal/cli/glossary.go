package cli

import (
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/zenmark-org/zenmd/engine/session"
)

// runGlossary is `zenmd glossary add "<Term>" [--folder <folder>] [--definition <text>] [--open]`
// and `zenmd glossary list [folder]`. It shares the engine's GlossaryFiles with
// the app's New Term…, so both write the same file.
func runGlossary(args []string, cwd string, stdout, stderr io.Writer, launch Launcher) int {
	var rest []string
	var folder, def string
	var haveFolder, open bool
	var perr error
	value := func(i *int, name, val string, hasVal bool) (string, bool) {
		switch {
		case hasVal:
			return val, true
		case *i+1 < len(args):
			*i++
			return args[*i], true
		}
		perr = usageError(name + " needs a value")
		return "", false
	}
	for i := 0; i < len(args) && perr == nil; i++ {
		arg := args[i]
		name, val, hasVal := strings.Cut(arg, "=")
		switch {
		case arg == "-h" || arg == "--help":
			fmt.Fprint(stdout, usage)
			return 0
		case name == "--folder":
			if v, ok := value(&i, name, val, hasVal); ok {
				folder, haveFolder = v, true
			}
		case name == "--definition":
			def, _ = value(&i, name, val, hasVal)
		case arg == "--open":
			open = true
		case strings.HasPrefix(arg, "-") && arg != "-":
			perr = usageError("unknown option " + arg)
		default:
			rest = append(rest, arg)
		}
	}
	if perr != nil {
		return fail(stderr, perr)
	}
	if len(rest) == 0 {
		return fail(stderr, usageError(`zenmd glossary what? try: zenmd glossary add "<Term>" or zenmd glossary list`))
	}
	cmd, rest := rest[0], rest[1:]
	switch cmd {
	case "add":
		if len(rest) != 1 {
			return fail(stderr, usageError(`zenmd glossary add needs one term, for example: zenmd glossary add "Idempotent"`))
		}
		if def != "" && strings.TrimSpace(def) == "" {
			def = ""
		}
	case "list":
		if len(rest) > 1 {
			return fail(stderr, usageError("zenmd glossary list takes at most one folder"))
		}
		if haveFolder || def != "" || open {
			return fail(stderr, usageError("zenmd glossary list takes only a folder: zenmd glossary list [folder]"))
		}
		if len(rest) == 1 {
			folder, haveFolder = rest[0], true
		}
	default:
		return fail(stderr, usageError("unknown glossary command "+cmd+" (add and list)"))
	}
	if cmd != "add" && (def != "" || open) {
		return fail(stderr, usageError("--definition and --open go with zenmd glossary add"))
	}

	rootDir := root(cwd)
	g, err := session.OpenGlossaryFiles("", rootDir)
	if err != nil {
		return fail(stderr, err)
	}
	// The folder: as given (from where zenmd runs), else where we are, else the only glossary.
	var rel string
	if haveFolder {
		abs := folder
		if !filepath.IsAbs(abs) {
			abs = filepath.Join(cwd, abs)
		}
		r, err := filepath.Rel(rootDir, filepath.Clean(abs))
		if err != nil || r == ".." || strings.HasPrefix(r, ".."+string(filepath.Separator)) {
			return fail(stderr, fmt.Errorf("%s is not inside %s", folder, rootDir))
		}
		rel = filepath.ToSlash(r)
		if rel == "." {
			rel = ""
		}
	} else if rel, err = findGlossary(g, rootDir, cwd); err != nil {
		return fail(stderr, err)
	}

	if cmd == "list" {
		terms, err := g.List(rel)
		if err != nil {
			return fail(stderr, err)
		}
		for _, t := range terms {
			if t.Definition != "" {
				fmt.Fprintf(stdout, "%s\t%s\n", t.Title, t.Definition)
			} else {
				fmt.Fprintln(stdout, t.Title)
			}
		}
		return 0
	}

	added, err := g.Add(rel, rest[0], def)
	if err != nil {
		return fail(stderr, err)
	}
	if added.Existed {
		if !open {
			return fail(stderr, fmt.Errorf("%s already exists", added.Path))
		}
		fmt.Fprintf(stderr, "zenmd: %s already exists\n", added.Path)
	} else {
		fmt.Fprintf(stdout, "Added %s\n", added.Path)
	}
	if open {
		if launch == nil {
			return fail(stderr, errors.New("this zenmd cannot open the editor"))
		}
		if err := launch(Launch{Dir: rootDir, Page: added.Path, Mode: "edit"}); err != nil {
			return fail(stderr, err)
		}
	}
	return 0
}

// findGlossary picks the glossary folder when --folder is not given: the
// deepest one that holds the current folder, else the only one in the folder.
func findGlossary(g *session.GlossaryFiles, rootDir, cwd string) (string, error) {
	all, err := g.Folders()
	if err != nil {
		return "", err
	}
	here := ""
	if r, err := filepath.Rel(rootDir, cwd); err == nil && r != "." && !strings.HasPrefix(r, "..") {
		here = filepath.ToSlash(r)
	}
	best, found := "", false
	for _, f := range all {
		if (f == here || f == "" || strings.HasPrefix(here, f+"/")) && (!found || len(f) > len(best)) {
			best, found = f, true
		}
	}
	// Only a glossary that holds this folder counts, unless it is the one glossary there is.
	if found && (best != "" || len(all) == 1) {
		return best, nil
	}
	switch len(all) {
	case 0:
		return "", errors.New("there is no glossary in this folder (a page with layout: glossary, usually glossary/index.md)")
	case 1:
		return all[0], nil
	}
	names := make([]string, len(all))
	for i, f := range all {
		if names[i] = f; f == "" {
			names[i] = "."
		}
	}
	return "", fmt.Errorf("there are several glossaries (%s): say which with --folder <folder>", strings.Join(names, ", "))
}
