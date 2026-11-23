package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/zenmark-org/zenmd/engine/config"
	"github.com/zenmark-org/zenmd/engine/session"
)

// checkProblem is one line of `zenmd check`, and one entry of its JSON.
type checkProblem struct {
	Page     string `json:"page"`
	Line     int    `json:"line"`
	Severity string `json:"severity"`
	Rule     string `json:"rule"`
	Message  string `json:"message"`
	Fix      string `json:"fix"`
	Source   string `json:"source"`
}

var severityRank = map[string]int{"error": 0, "warning": 1, "suggestion": 2}

// runCheck is `zenmd check [folder] [--format text|json] [--severity error|warning|suggestion] [--template id]`.
// It checks every page of the folder against its page template and its layout's structure (the code behind
// Problems › Structure) and prints the problems by page. Text is one line per problem on stdout,
//
//	<page>:<line>: <severity>: <message>
//
// and nothing else (a CI step parses it); the summary goes to stderr. It exits with 1 when a problem is an
// error, 0 otherwise and 2 for a wrong request. --severity says how low to go: error shows only errors, warning
// (the default) adds warnings, suggestion shows everything. There is no `check:` config key yet: the flags are
// all the options there are.
func runCheck(args []string, cwd string, stdout, stderr io.Writer) int {
	format, severity, template := "text", "warning", ""
	var paths []string
	var perr error
	for i := 0; i < len(args) && perr == nil; i++ {
		arg := args[i]
		name, val, hasVal := arg, "", false
		if strings.HasPrefix(arg, "--") {
			name, val, hasVal = strings.Cut(arg, "=")
		}
		value := func() string {
			switch {
			case hasVal && val != "":
				return val
			case !hasVal && i+1 < len(args) && args[i+1] != "":
				i++
				return args[i]
			}
			perr = usageError(name + " needs a value")
			return ""
		}
		switch {
		case arg == "-h" || arg == "--help":
			fmt.Fprint(stdout, usage)
			return 0
		case name == "--format":
			format = value()
		case name == "--severity":
			severity = value()
		case name == "--template":
			template = value()
		case strings.HasPrefix(arg, "-") && arg != "-":
			perr = usageError("unknown option " + arg)
		default:
			paths = append(paths, arg)
		}
	}
	switch {
	case perr != nil:
		return fail(stderr, perr)
	case format != "text" && format != "json":
		return fail(stderr, usageError("--format is text or json, not "+format))
	case severity != "error" && severity != "warning" && severity != "suggestion":
		return fail(stderr, usageError("--severity is error, warning or suggestion, not "+severity))
	case len(paths) > 1:
		return fail(stderr, usageError("more than one folder given"))
	}
	dir := cwd
	if len(paths) == 1 {
		dir = paths[0]
		if !filepath.IsAbs(dir) {
			dir = filepath.Join(cwd, dir)
		}
	}
	dir = filepath.Clean(dir)
	switch fi, err := os.Stat(dir); {
	case err != nil:
		return fail(stderr, errors.New(paths0(paths)+": no such folder"))
	case !fi.IsDir():
		return fail(stderr, errors.New(paths0(paths)+": check takes a folder, not a file"))
	}

	loaded := loadConfig(dir)
	printNotices(stderr, loaded)
	globalPath, _ := config.GlobalPath()
	rep, err := session.CheckFolder(session.SiteEnv{GlobalPath: globalPath, Version: Version}, session.CheckOptions{Dir: dir, Template: template})
	if errors.Is(err, session.ErrNoPageTemplate) {
		return fail(stderr, usageError(err.Error()+"; zenmd templates list shows them"))
	}
	if err != nil {
		return fail(stderr, err)
	}

	limit := severityRank[severity]
	var shown []checkProblem
	counts := map[string]int{}
	for _, r := range rep.Results {
		var page []checkProblem
		for _, p := range r.Problems {
			if severityRank[p.Severity] > limit {
				continue
			}
			page = append(page, checkProblem{Page: r.Path, Line: p.Line, Severity: p.Severity, Rule: p.Rule, Message: oneLine(p.Message), Fix: p.Fix, Source: p.Source})
			counts[p.Severity]++
		}
		sort.SliceStable(page, func(i, j int) bool { return page[i].Line < page[j].Line })
		shown = append(shown, page...)
	}

	if format == "json" {
		out := struct {
			Pages    int            `json:"pages"`
			Problems []checkProblem `json:"problems"`
		}{rep.Pages, shown}
		if out.Problems == nil {
			out.Problems = []checkProblem{}
		}
		b, _ := json.MarshalIndent(out, "", "  ")
		fmt.Fprintln(stdout, string(b))
	} else {
		for _, p := range shown {
			fmt.Fprintf(stdout, "%s:%d: %s: %s\n", p.Page, p.Line, p.Severity, p.Message)
		}
	}
	fmt.Fprintf(stderr, "Checked %d %s: %s, %s, %s\n", rep.Pages, plural(rep.Pages, "page", "pages"),
		count(counts["error"], "error", "errors"), count(counts["warning"], "warning", "warnings"), count(counts["suggestion"], "suggestion", "suggestions"))
	if counts["error"] > 0 {
		return 1
	}
	return 0
}

func paths0(p []string) string {
	if len(p) == 1 {
		return p[0]
	}
	return "the current folder"
}

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}

func count(n int, one, many string) string { return fmt.Sprintf("%d %s", n, plural(n, one, many)) }

// oneLine keeps a message on its one line.
func oneLine(s string) string { return strings.Join(strings.Fields(s), " ") }
