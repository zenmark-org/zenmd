package cli

import (
	"encoding/json"
	"regexp"
	"strings"
	"testing"
)

// checkFolder has a template of its own and a path rule: guides/a.md lacks the section (an error),
// blog/x.md lacks the post layout's date (an error), guides/w.md only has a warning, and ok.md is clean.
func checkFolder(t *testing.T) string {
	t.Helper()
	home(t)
	return glossaryDir(t, map[string]string{
		".zen/templates/guide.yml": "structure:\n  - heading: { level: 1, count: 1 }\n  - section: { heading: Steps, severity: error }\n  - section: { heading: Next, severity: suggestion }\n",
		".zenmd.yml":               "templates:\n  \"guides/**\": guide\n",
		"guides/a.md":              "# A\n\ntext\n",
		"guides/w.md":              "# W\n\n## Steps\n\n## Later\n",
		"blog/x.md":                "---\nlayout: post\nauthor: Ada\n---\n# X\n\nSummary.\n",
		"ok.md":                    "# Ok\n\ntext\n",
	})
}

// A CI action parses these lines (#31): <page>:<line>: <severity>: <message>.
var checkLine = regexp.MustCompile(`^[^:\s][^:]*:[0-9]+: (error|warning|suggestion): .+$`)

func TestCheckTextFormatAndExitCode(t *testing.T) {
	dir := checkFolder(t)
	r := run(t, dir, nil, "check")
	if r.code != 1 {
		t.Fatalf("errors give exit code 1, got %d\nout %q err %q", r.code, r.stdout, r.stderr)
	}
	lines := strings.Split(strings.TrimRight(r.stdout, "\n"), "\n")
	var problems []string
	for _, l := range lines {
		if checkLine.MatchString(l) {
			problems = append(problems, l)
		}
	}
	if len(problems) == 0 {
		t.Fatalf("no problem lines in %q", r.stdout)
	}
	want := []string{
		"guides/a.md:1: error: Missing section 'Steps'",
		"blog/x.md:1: error: date is required by the post layout",
	}
	for _, w := range want {
		found := false
		for _, l := range problems {
			found = found || strings.HasPrefix(l, strings.SplitN(w, ":", 3)[0]+":") && strings.Contains(l, w[strings.LastIndex(w, ": ")+2:])
		}
		if !found {
			t.Errorf("want a line like %q in:\n%s", w, r.stdout)
		}
	}
	// One problem per line, by page: the lines of a page stay together, in page order, and a clean page has none.
	if strings.Contains(r.stdout, "ok.md") {
		t.Errorf("a clean page is not listed:\n%s", r.stdout)
	}
	for _, l := range problems {
		if strings.Contains(l, "\t") {
			t.Errorf("one plain line per problem: %q", l)
		}
	}
	// The Fix line is not part of the text lines a parser reads: it follows indented, or not at all.
	for _, l := range lines {
		if !checkLine.MatchString(l) && !strings.HasPrefix(l, " ") && l != "" && !strings.Contains(l, "problem") && !strings.Contains(l, "checked") {
			t.Errorf("an unexpected line %q", l)
		}
	}
}

func TestCheckExactLineFormat(t *testing.T) {
	dir := glossaryDir(t, map[string]string{"a.md": "# A\n\n![](x.png)\n"})
	home(t)
	r := run(t, dir, nil, "check", "--format", "text")
	// The article layout asks for alt text: a warning at line 3. Warnings do not fail the build.
	if r.code != 0 {
		t.Fatalf("code %d %q %q", r.code, r.stdout, r.stderr)
	}
	if !strings.Contains(r.stdout, "\na.md:3: warning: Image has no alt text\n") && !strings.HasPrefix(r.stdout, "a.md:3: warning: Image has no alt text\n") {
		t.Errorf("format: %q", r.stdout)
	}
}

func TestCheckSeverityFilter(t *testing.T) {
	dir := checkFolder(t)
	only := func(args ...string) string {
		r := run(t, dir, nil, append([]string{"check"}, args...)...)
		return r.stdout
	}
	if out := only(); strings.Contains(out, ": suggestion: ") {
		t.Errorf("suggestions are hidden by default:\n%s", out)
	}
	if out := only("--severity", "suggestion"); !strings.Contains(out, ": suggestion: ") {
		t.Errorf("--severity suggestion shows them:\n%s", out)
	}
	out := only("--severity", "error")
	if strings.Contains(out, ": warning: ") || strings.Contains(out, ": suggestion: ") || !strings.Contains(out, ": error: ") {
		t.Errorf("--severity error:\n%s", out)
	}
}

func TestCheckCleanFolderExitsZero(t *testing.T) {
	home(t)
	dir := glossaryDir(t, map[string]string{"README.md": "# Home\n\nText.\n"})
	r := run(t, dir, nil, "check")
	if r.code != 0 || strings.TrimSpace(r.stdout) == "" || strings.Contains(r.stdout, ": error: ") {
		t.Errorf("code %d out %q err %q", r.code, r.stdout, r.stderr)
	}
}

func TestCheckFolderArgumentAndTemplate(t *testing.T) {
	dir := checkFolder(t)
	r := run(t, t.TempDir(), nil, "check", dir, "--template", "guide")
	if r.code != 1 || !strings.Contains(r.stdout, "ok.md:1: error: Missing section 'Steps'") {
		t.Errorf("--template applies to every page: code %d\n%s", r.code, r.stdout)
	}
	r = run(t, dir, nil, "check", "--template", "nope")
	if r.code != 2 || !strings.Contains(r.stderr, "nope") {
		t.Errorf("an unknown template is a usage error: code %d %q", r.code, r.stderr)
	}
}

func TestCheckJSON(t *testing.T) {
	dir := checkFolder(t)
	r := run(t, dir, nil, "check", "--format", "json")
	if r.code != 1 {
		t.Fatalf("code %d", r.code)
	}
	var got struct {
		Pages    int `json:"pages"`
		Problems []struct {
			Page, Rule, Message, Fix, Severity, Source string
			Line                                       int
		} `json:"problems"`
	}
	if err := json.Unmarshal([]byte(r.stdout), &got); err != nil {
		t.Fatalf("%v\n%s", err, r.stdout)
	}
	if got.Pages != 4 || len(got.Problems) == 0 {
		t.Fatalf("%+v", got)
	}
	for _, p := range got.Problems {
		if p.Page == "" || p.Line < 1 || p.Message == "" || p.Severity == "" {
			t.Errorf("%+v", p)
		}
	}
}

func TestCheckUsageErrors(t *testing.T) {
	dir := checkFolder(t)
	for _, args := range [][]string{
		{"check", "--format", "yaml"},
		{"check", "--severity", "loud"},
		{"check", "--format"},
		{"check", "--nope"},
		{"check", dir, dir},
	} {
		if r := run(t, dir, nil, args...); r.code != 2 {
			t.Errorf("%v: code %d %q", args, r.code, r.stderr)
		}
	}
	if r := run(t, dir, nil, "check", dir+"/nothing"); r.code != 1 {
		t.Errorf("a missing folder: code %d", r.code)
	}
	if r := run(t, dir, nil, "check", "--help"); r.code != 0 || !strings.Contains(r.stdout, "zenmd check") {
		t.Errorf("help: %d %q", r.code, r.stdout)
	}
}

// Every command is in the usage text.
func TestCheckInUsage(t *testing.T) {
	if !strings.Contains(usage, "zenmd check [folder]") || !strings.Contains(usage, "--severity") {
		t.Error("the help text does not describe zenmd check")
	}
}
