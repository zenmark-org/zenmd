// Package cli parses the zenmd command line and decides whether to print
// information or start the GUI. It uses the standard library only.
package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/zenmark-org/zenmd/engine/config"
	"github.com/zenmark-org/zenmd/kit/addons"
)

// Name is the command and executable: the name of the engine.
const Name = "zenmd"

// Alias is the old command name. It keeps working until 0.4.0 and says so.
const Alias = "zen" // removed in 0.4.0

// ProgramName tells which name the program was started as, from argv[0]:
// "zen" (the alias, matched by base name and ignoring case, as the disk may
// be case-insensitive) or "zenmd" for anything else (including a temporary
// binary from `go run .`). The link name counts, not the link's target.
func ProgramName(argv0 string) string {
	base := filepath.Base(filepath.FromSlash(argv0))
	if argv0 != "" && strings.EqualFold(base, Alias) {
		return Alias
	}
	return Name
}

// Notice prints the alias line for "zen" and nothing for any other name.
func Notice(w io.Writer, invokedAs string) {
	if invokedAs == Alias {
		fmt.Fprintln(w, "zen is now zenmd; the zen name stops working in 0.4.0")
	}
}

// Version is set at build time with
// -ldflags "-X github.com/zenmark-org/zenmd/internal/cli.Version=<v>".
var Version = "dev"

// addonList is the add-ons of this build (kit/addons); tests replace it.
var addonList = addons.List

// VersionLine is "zenmd 0.6.0", followed by the add-ons of the build when it has
// some: "zenmd 0.6.0 (add-ons: vale, voice)". The About panel shows the same.
func VersionLine() string {
	line := "zenmd " + Version
	var ids []string
	for _, a := range addonList() {
		ids = append(ids, a.ID)
	}
	if len(ids) > 0 {
		line += " (add-ons: " + strings.Join(ids, ", ") + ")"
	}
	return line
}

// BuildInfo is what an app's build is made of besides its add-ons (the
// assembly, kit/assembly); `zenmd version --verbose` prints it.
type BuildInfo struct {
	App                                   string
	Features, Plugins, Layouts, Templates []string
	Dialects                              []string // markdown dialects (engine/dialects)
	Theme                                 string
}

var buildInfo func() BuildInfo

// SetBuildInfo tells the command what the app is made of. The app calls it
// once, from its init; without it `version --verbose` prints the version line only.
func SetBuildInfo(f func() BuildInfo) { buildInfo = f }

// VerboseVersion is the version line followed by the assembly, one line each:
// app, features, add-ons, plugins, layouts, templates, dialects and theme.
func VerboseVersion() string {
	out := VersionLine() + "\n"
	if buildInfo == nil {
		return out
	}
	b := buildInfo()
	var addonIDs []string
	for _, a := range addonList() {
		addonIDs = append(addonIDs, a.ID)
	}
	list := func(ids []string) string {
		if len(ids) == 0 {
			return "none"
		}
		return strings.Join(ids, ", ")
	}
	return out + "app: " + b.App + "\n" +
		"features: " + list(b.Features) + "\n" +
		"add-ons: " + list(addonIDs) + "\n" +
		"plugins: " + list(b.Plugins) + "\n" +
		"layouts: " + list(b.Layouts) + "\n" +
		"templates: " + list(b.Templates) + "\n" +
		"dialects: " + list(b.Dialects) + "\n" +
		"theme: " + b.Theme + "\n"
}

// Launch describes what the GUI opens.
type Launch struct {
	Dir  string // absolute directory to open
	Page string // workspace-relative page, "" for default
	Mode string // "read" | "edit"
}

// Launcher starts the GUI and blocks until it exits.
type Launcher func(Launch) error

// Build describes what `zenmd build` does: no window, no questions.
type Build struct {
	Dir      string // absolute folder
	Out      string // absolute output folder; "" = from config / default
	Portable bool   // --portable: opens from disk (the default)
	Hosted   bool   // --hosted: scripts as files, for a web server
	Drafts   bool
	NoDrafts bool // --no-drafts: leave drafts out even when export.drafts is true
	Clean    bool
	NoPrint  bool // --no-print: no print.html
	// Scheme is --scheme: "auto", "light" or "dark"; "" leaves it to export.theme.scheme (auto without it).
	Scheme string
}

// Builder builds a static website. It writes the summary to stdout and one
// line per warning to stderr, and returns an error when nothing usable was
// written.
type Builder func(Build, io.Writer, io.Writer) error

// Init describes `zenmd init` and `zenmd blueprint`: start a folder from a
// blueprint (formerly a site template), list the blueprints, update one, or
// make a new blueprint. It never asks anything, so it works in scripts.
type Init struct {
	Dir, Template string            // absolute target folder; blueprint id (or the id to update or make)
	Values        map[string]string // --title, --author and --set key=value
	List, Update  bool
	ShowDir       bool
	New           bool   // zenmd blueprint new <id>: write a blueprint into the blueprints folder
	From          string // zenmd blueprint new --from <id>: start from this blueprint
	// Pages is zenmd templates: the page templates instead of the blueprints. With it List, Show
	// (print the template Template) and Update (bring Template to its newer version) say what to do,
	// and ShowDir prints the whole templates folder.
	Pages, Show bool
}

// Initer does what `zenmd init` asks. It writes its result to stdout and notes
// to stderr. Return UsageError for a wrong request (exit 2) and any other
// error for a failure (exit 1).
type Initer func(Init, io.Writer, io.Writer) error

// UsageError is an error that makes zenmd print the usage and exit with 2.
func UsageError(msg string) error { return usageError(msg) }

// usage is the help text; the options of the exports come from their lists (options.go).
var usage = usageText()

func usageText() string {
	return `zenmd - the Zenmark engine: markdown folders as pages, sites and print

Usage:
  zenmd [path]          open a folder or markdown file (default: current folder)
  zenmd edit [path]     open in the editor; a new .md file is created on save
  zenmd export site [folder]
                        build a static website; options below
  zenmd export pdf [folder] [-o file.pdf]
                        write the whole folder as one PDF file; options below
  zenmd build [folder]  same as export site
  zenmd init --blueprint <id> [folder]
                        start a folder from a blueprint; options below
                        (zenmd init site --techdocs works too)
  zenmd blueprint list  list the blueprints
  zenmd blueprint new <id> [--from <id>]
                        write a new blueprint into the blueprints folder
  zenmd templates list  list the page templates, their source and version
  zenmd templates show <id>
                        print a page template's fields, structure, skeleton
  zenmd templates update <id>
                        bring a built-in page template to its newer version
                        (zenmd init --update <id> does it for a blueprint)
  zenmd templates --dir print the templates folder (blueprints/, pages/)
  zenmd glossary add "<Term>" [--folder <folder>] [--definition <text>] [--open]
                        add a term: <folder>/<slug>.md from the term template
                        (the folder is found when there is one glossary)
  zenmd glossary list [folder]
                        list the terms A to Z with their definitions
  zenmd help            show this help
  zenmd version         show the version

Options:
  -h, --help          show this help
  -v, --version       show the version
  --                  treat the following arguments as paths

Init options:
  --blueprint <id>    the blueprint (--template <id>, --<id> work too)
  --title <text>      the site title; --author <text>; --set <key=value>
  --list              list the blueprints; --update <id>; --dir (their folder)

` + optionHelp("Export site options", siteOptions) + `  (also for zenmd build)
  A flag wins over the folder's export: settings, then config.yml.

` + optionHelp("Export pdf options", pdfOptions) + `  A flag wins over the folder's export.pdf settings, then config.yml.
`
}

type usageError string

func (e usageError) Error() string { return string(e) }

// Run executes the command line and returns the process exit code.
func Run(args []string, cwd string, stdout, stderr io.Writer, launch Launcher, build Builder, initer Initer, pdf PDFExporter) int {
	mode := "read"
	var paths []string
	flags, first := true, true
	for i, arg := range args {
		switch {
		case flags && strings.HasPrefix(arg, "-psn_"):
			continue
		case flags && arg == "--":
			flags = false
		case flags && (arg == "-h" || arg == "--help"):
			fmt.Fprint(stdout, usage)
			return 0
		case flags && (arg == "-v" || arg == "--version"):
			fmt.Fprintln(stdout, VersionLine())
			return 0
		case flags && strings.HasPrefix(arg, "-") && arg != "-":
			return fail(stderr, usageError("unknown option "+arg))
		case flags && first && arg == "help":
			fmt.Fprint(stdout, usage)
			return 0
		case flags && first && arg == "version":
			if slices.Contains(args[i+1:], "--verbose") {
				fmt.Fprint(stdout, VerboseVersion())
				return 0
			}
			fmt.Fprintln(stdout, VersionLine())
			return 0
		case flags && first && arg == "build":
			if code, ok := needBuilt(stderr, "export"); !ok { // build is export site
				return code
			}
			return runBuild(args[i+1:], cwd, stdout, stderr, build)
		case flags && first && arg == "export":
			return runExport(args[i+1:], cwd, stdout, stderr, build, pdf)
		case flags && first && arg == "init":
			return runInit(args[i+1:], cwd, stdout, stderr, initer)
		case flags && first && arg == "blueprint":
			return runBlueprint(args[i+1:], cwd, stdout, stderr, initer)
		case flags && first && arg == "templates":
			return runTemplates(args[i+1:], cwd, stdout, stderr, initer)
		case flags && first && arg == "glossary":
			return runGlossary(args[i+1:], cwd, stdout, stderr, launch)
		case flags && first && arg == "edit":
			mode = "edit"
		default:
			paths = append(paths, arg)
		}
		first = false
	}
	if len(paths) > 1 {
		return fail(stderr, usageError("more than one path given"))
	}

	l := Launch{Dir: cwd, Mode: mode}
	if len(paths) == 0 && finderLaunch(cwd) {
		l.Dir = "" // welcome screen
	}
	if len(paths) == 1 {
		var err error
		if l.Dir, l.Page, err = resolve(paths[0], cwd, mode == "edit"); err != nil {
			return fail(stderr, err)
		}
	}
	if l.Dir != "" {
		noticeLegacyConfig(stderr, l.Dir)
	}
	if err := launch(l); err != nil {
		return fail(stderr, err)
	}
	return 0
}

// noticeLegacyConfig says on stderr, once per run, that the folder still has
// the old settings file (or both files).
func noticeLegacyConfig(stderr io.Writer, dir string) {
	_, legacy, both := config.WorkspaceConfigPath(dir)
	if n := config.Notice(legacy, both); n != "" {
		fmt.Fprintln(stderr, n)
	}
}

// runBuild parses the arguments after `zenmd build` (or `export site`) and calls the builder.
func runBuild(args []string, cwd string, stdout, stderr io.Writer, build Builder) int {
	f, err := parseOptions(args, siteOptions)
	if err != nil {
		return fail(stderr, err)
	}
	if f.help {
		fmt.Fprint(stdout, usage)
		return 0
	}
	if f.on[flagDryRun] && !f.on[flagSaveDefaults] {
		return fail(stderr, usageError(flagDryRun+" goes with "+flagSaveDefaults))
	}
	if len(f.paths) > 1 {
		return fail(stderr, usageError("more than one path given"))
	}
	b := Build{
		Portable: f.on["--portable"], Hosted: f.off["--portable"],
		Drafts: f.on["--drafts"], NoDrafts: f.off["--drafts"],
		Clean: f.on["--clean"], NoPrint: f.off["--print"], Scheme: f.val["--scheme"],
	}
	b.Dir = cwd
	if len(f.paths) == 1 {
		if b.Dir, _, err = resolve(f.paths[0], cwd, false); err != nil {
			return fail(stderr, err)
		}
	}
	if f.on["--out"] {
		b.Out = f.val["--out"]
		if !filepath.IsAbs(b.Out) {
			b.Out = filepath.Join(cwd, b.Out)
		}
		b.Out = filepath.Clean(b.Out)
	}
	if build == nil {
		return fail(stderr, errors.New("this zenmd cannot build sites"))
	}
	loaded := loadConfig(b.Dir)
	printNotices(stderr, loaded)
	if f.on[flagSaveDefaults] {
		keys := []config.ExportKey{}
		if f.on["--out"] {
			keys = append(keys, config.StrKey("out", config.SiteOutValue(b.Dir, b.Out)))
		}
		if f.given("--portable") {
			keys = append(keys, config.BoolKey("portable", f.on["--portable"]))
		}
		if f.given("--drafts") {
			keys = append(keys, config.BoolKey("drafts", f.on["--drafts"]))
		}
		if f.on["--scheme"] {
			keys = append(keys, config.StrKey("theme.scheme", f.val["--scheme"]))
		}
		if done, err := saveDefaults(stdout, b.Dir, keys, f.on[flagDryRun]); err != nil {
			return fail(stderr, err)
		} else if done {
			return 0
		}
	}
	if err := build(b, stdout, stderr); err != nil {
		return fail(stderr, err)
	}
	return 0
}

// loadConfig reads the global config.yml and the folder's .zenmd.yml.
func loadConfig(dir string) *config.Config {
	gp, _ := config.GlobalPath()
	return config.Load(gp, dir)
}

// printNotices says on stderr what the config still names the old way: the folder's old settings file and a build:
// key, in the folder's file or the global one (the app shows the same under Problems).
func printNotices(stderr io.Writer, c *config.Config) {
	for _, n := range c.Notices() {
		fmt.Fprintln(stderr, n)
	}
}

// saveDefaults is --save-defaults: it writes the keys into the folder's .zenmd.yml (config.SaveExportKeys, the code
// behind the app's Save as folder defaults) and prints what changed. With dryRun it only prints it, and done is true:
// nothing is exported either.
func saveDefaults(stdout io.Writer, dir string, keys []config.ExportKey, dryRun bool) (done bool, err error) {
	if len(keys) == 0 {
		fmt.Fprintln(stdout, "Nothing to change: no option to save was given, such as --drafts.")
		return dryRun, nil
	}
	text, err := config.SaveExportKeys(dir, keys, "Nothing to change: .zenmd.yml already has these export defaults.", !dryRun)
	if err != nil {
		return false, err
	}
	fmt.Fprintln(stdout, text)
	return dryRun, nil
}

var initFlagRE = regexp.MustCompile(`^--[a-z][a-z0-9-]*$`)

// runInit parses the arguments after `zenmd init` and calls the initer.
func runInit(args []string, cwd string, stdout, stderr io.Writer, initer Initer) int {
	in := Init{Values: map[string]string{}}
	var paths []string
	set := map[string]string{}
	var title, author *string
	site, flags, noTemplate := false, true, false
	needs := func(i *int, name string) (string, error) {
		if *i+1 >= len(args) || args[*i+1] == "" {
			return "", usageError(name + " needs a value")
		}
		*i++
		return args[*i], nil
	}
	var perr error
	pickTemplate := func(id string) {
		if in.Template != "" && perr == nil {
			perr = usageError("more than one template given")
		}
		in.Template = id
	}
	for i := 0; i < len(args) && perr == nil; i++ {
		arg := args[i]
		name, val, hasVal := arg, "", false
		if flags && strings.HasPrefix(arg, "--") {
			name, val, hasVal = strings.Cut(arg, "=")
		}
		value := func() string {
			if hasVal {
				if val == "" {
					perr = usageError(name + " needs a value")
				}
				return val
			}
			v, err := needs(&i, name)
			if err != nil {
				perr = err
			}
			return v
		}
		switch {
		case flags && arg == "--":
			flags = false
		case flags && (arg == "-h" || arg == "--help"):
			fmt.Fprint(stdout, usage)
			return 0
		case flags && name == "--list":
			in.List = true
		case flags && name == "--dir":
			in.ShowDir = true
		case flags && name == "--update":
			in.Update = true
			pickTemplate(value())
			noTemplate = true
		case flags && (name == "--template" || name == "--blueprint"):
			pickTemplate(value())
		case flags && name == "--title":
			v := value()
			title = &v
		case flags && name == "--author":
			v := value()
			author = &v
		case flags && name == "--set":
			kv := value()
			k, v, ok := strings.Cut(kv, "=")
			if perr == nil && (!ok || k == "") {
				perr = usageError("--set needs key=value, not " + kv)
			}
			set[k] = v
		case flags && !hasVal && initFlagRE.MatchString(arg):
			pickTemplate(arg[2:]) // --techdocs, --learning, --planner, --<id>
		case flags && strings.HasPrefix(arg, "-") && arg != "-":
			perr = usageError("unknown option " + arg)
		case !site && len(paths) == 0 && arg == "site":
			site = true
		default:
			paths = append(paths, arg)
		}
	}
	modes := 0
	for _, b := range []bool{in.List, in.ShowDir, in.Update} {
		if b {
			modes++
		}
	}
	switch {
	case perr != nil:
		return fail(stderr, perr)
	case modes > 1:
		return fail(stderr, usageError("--list, --dir and --update cannot be used together"))
	case modes == 1 && (site || len(paths) > 0 || len(set) > 0 || title != nil || author != nil || (in.Template != "" && !noTemplate)):
		return fail(stderr, usageError("--list, --dir and --update take nothing else"))
	case modes == 0 && !site && in.Template == "":
		return fail(stderr, usageError("zenmd init what? try: zenmd init --blueprint techdocs"))
	case len(paths) > 1:
		return fail(stderr, usageError("more than one folder given"))
	}
	for k, v := range set {
		in.Values[k] = v
	}
	if title != nil {
		in.Values["title"] = *title
	}
	if author != nil {
		in.Values["author"] = *author
	}
	in.Dir = cwd
	if len(paths) == 1 {
		in.Dir = paths[0]
		if !filepath.IsAbs(in.Dir) {
			in.Dir = filepath.Join(cwd, in.Dir)
		}
		in.Dir = filepath.Clean(in.Dir)
	}
	if initer == nil {
		return fail(stderr, errors.New("zenmd init is not available in this build"))
	}
	missing := modes == 0 && in.Template == ""
	if missing {
		in.List = true // no template: show the choices
	}
	if err := initer(in, stdout, stderr); err != nil {
		return fail(stderr, err)
	}
	if missing {
		fmt.Fprintln(stderr, "zenmd: choose a blueprint, for example: zenmd init --blueprint techdocs")
		return 2
	}
	return 0
}

// runBlueprint parses the arguments after `zenmd blueprint`: list, or new <id>
// [--from <id>]. The check, save and add-section commands come later.
func runBlueprint(args []string, cwd string, stdout, stderr io.Writer, initer Initer) int {
	in := Init{Values: map[string]string{}, Dir: cwd}
	var rest []string
	var perr error
	for i := 0; i < len(args) && perr == nil; i++ {
		arg := args[i]
		name, val, hasVal := strings.Cut(arg, "=")
		switch {
		case arg == "-h" || arg == "--help":
			fmt.Fprint(stdout, usage)
			return 0
		case name == "--from":
			if in.From != "" {
				perr = usageError("more than one --from given")
			} else if hasVal && val != "" {
				in.From = val
			} else if !hasVal && i+1 < len(args) && args[i+1] != "" {
				i++
				in.From = args[i]
			} else {
				perr = usageError("--from needs a blueprint id")
			}
		case strings.HasPrefix(arg, "-") && arg != "-":
			perr = usageError("unknown option " + arg)
		default:
			rest = append(rest, arg)
		}
	}
	switch {
	case perr != nil:
		return fail(stderr, perr)
	case len(rest) == 0:
		return fail(stderr, usageError("zenmd blueprint what? try: zenmd blueprint list"))
	case rest[0] == "list" && len(rest) == 1 && in.From == "":
		in.List = true
	case rest[0] == "new" && len(rest) == 2:
		in.New, in.Template = true, rest[1]
	case rest[0] == "new":
		return fail(stderr, usageError("zenmd blueprint new needs one id, for example: zenmd blueprint new mine --from techdocs"))
	case rest[0] == "list":
		return fail(stderr, usageError("zenmd blueprint list takes nothing else"))
	default:
		return fail(stderr, usageError("unknown blueprint command "+rest[0]+" (list and new are there; check, save and add-section come later)"))
	}
	if in.From != "" && !in.New {
		return fail(stderr, usageError("--from goes with zenmd blueprint new"))
	}
	if initer == nil {
		return fail(stderr, errors.New("zenmd blueprint is not available in this build"))
	}
	if err := initer(in, stdout, stderr); err != nil {
		return fail(stderr, err)
	}
	return 0
}

// runTemplates parses the arguments after `zenmd templates`: list, show <id>, update <id> or --dir.
func runTemplates(args []string, cwd string, stdout, stderr io.Writer, initer Initer) int {
	in := Init{Values: map[string]string{}, Dir: cwd, Pages: true}
	var rest []string
	for _, arg := range args {
		switch {
		case arg == "-h" || arg == "--help":
			fmt.Fprint(stdout, usage)
			return 0
		case arg == "--dir":
			in.ShowDir = true
		case strings.HasPrefix(arg, "-") && arg != "-":
			return fail(stderr, usageError("unknown option "+arg))
		default:
			rest = append(rest, arg)
		}
	}
	switch {
	case in.ShowDir && len(rest) == 0:
	case in.ShowDir:
		return fail(stderr, usageError("zenmd templates --dir takes nothing else"))
	case len(rest) == 0:
		return fail(stderr, usageError("zenmd templates what? try: zenmd templates list"))
	case rest[0] == "list" && len(rest) == 1:
		in.List = true
	case (rest[0] == "show" || rest[0] == "update") && len(rest) == 2:
		in.Show, in.Update, in.Template = rest[0] == "show", rest[0] == "update", rest[1]
	case rest[0] == "show" || rest[0] == "update":
		return fail(stderr, usageError("zenmd templates "+rest[0]+" needs one id, for example: zenmd templates "+rest[0]+" how-to"))
	case rest[0] == "list":
		return fail(stderr, usageError("zenmd templates list takes nothing else"))
	default:
		return fail(stderr, usageError("unknown templates command "+rest[0]+" (list, show, update and --dir are there)"))
	}
	if initer == nil {
		return fail(stderr, errors.New("zenmd templates is not available in this build"))
	}
	if err := initer(in, stdout, stderr); err != nil {
		return fail(stderr, err)
	}
	return 0
}

// finderLaunch reports whether zen should start without a folder: Finder,
// the Dock and launchd start apps with cwd "/". Checking for a terminal does
// not work (they attach /dev/null, which is a character device), and opening
// the whole disk is never what anyone wants, so cwd "/" alone decides.
func finderLaunch(cwd string) bool {
	return cwd == string(filepath.Separator)
}

// helpHint follows a usage error; the whole help is for zenmd help.
const helpHint = "Run zenmd help for the commands and options."

func fail(stderr io.Writer, err error) int {
	var u usageError
	if errors.As(err, &u) {
		fmt.Fprintf(stderr, "zenmd: %v\n\n%s\n", err, helpHint)
		return 2
	}
	fmt.Fprintf(stderr, "zenmd: %v\n", err)
	return 1
}

func isMarkdown(p string) bool {
	switch strings.ToLower(filepath.Ext(p)) {
	case ".md", ".markdown":
		return true
	}
	return false
}

// resolve maps a path argument to a workspace directory and page.
func resolve(arg, cwd string, allowNew bool) (dir, page string, err error) {
	p := arg
	if !filepath.IsAbs(p) {
		p = filepath.Join(cwd, p)
	}
	p = filepath.Clean(p)

	info, statErr := os.Stat(p)
	switch {
	case statErr == nil && info.IsDir():
		return p, "", nil
	case statErr != nil && errors.Is(statErr, os.ErrNotExist) && !(allowNew && isMarkdown(p)):
		return "", "", usageError("no such folder or file: " + p)
	case !isMarkdown(p):
		return "", "", usageError(p + " is neither a directory nor a markdown file")
	case statErr == nil:
	case allowNew && errors.Is(statErr, os.ErrNotExist):
		if pi, err := os.Stat(filepath.Dir(p)); err != nil || !pi.IsDir() {
			return "", "", usageError(p + ": parent directory does not exist")
		}
	default:
		return "", "", usageError("no such folder or file: " + p)
	}

	dir = root(filepath.Dir(p))
	rel, err := filepath.Rel(dir, p)
	if err != nil {
		return "", "", err
	}
	return dir, filepath.ToSlash(rel), nil
}

// root returns the nearest ancestor of start (inclusive) containing a folder
// settings file (either name) or .git, searching only strictly below the
// user's home directory. Without a marker, start itself is the root.
func root(start string) string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return start
	}
	home = filepath.Clean(home)
	for d := start; within(d, home); d = filepath.Dir(d) {
		for _, m := range []string{config.WorkspaceFile, config.LegacyWorkspaceFile, ".git"} {
			if _, err := os.Stat(filepath.Join(d, m)); err == nil {
				return d
			}
		}
	}
	return start
}

// within reports whether d is strictly inside home.
func within(d, home string) bool {
	rel, err := filepath.Rel(home, d)
	return err == nil && rel != "." && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
