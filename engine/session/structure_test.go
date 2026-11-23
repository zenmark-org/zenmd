package session

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/zenmark-org/zenmd/engine/plugins"
)

const guideTpl = `id: guide
name: Guide
structure:
  - heading: { level: 1, count: 1 }
  - section: { heading: Steps, severity: error }
`

const otherTpl = `id: other
name: Other
structure:
  - section: { heading: Extra }
`

func structureFolder(t *testing.T) *Session {
	t.Helper()
	dir := tree(t, map[string]string{
		".zen/templates/guide.yml": guideTpl,
		".zen/templates/other.yml": otherTpl,
		".zenmd.yml":               "templates:\n  \"guides/**\": guide\n",
		"guides/a.md":              "# A\n\ntext\n",
		"guides/b.md":              "---\ntemplate: other\n---\n# B\n\n## Steps\n",
		"guides/c.md":              "---\ntemplate: false\n---\n# C\n",
		"guides/d.md":              "---\ntemplate: nope\n---\n# D\n",
		"guides/e.md":              "---\ntemplate: ../../x\n---\n# E\n",
		"guides/ok.md":             "# Ok\n\n## Steps\n\n1. a\n",
		"blog/x.md":                "---\nlayout: post\n---\n# X\n\nSummary.\n",
		"plain.md":                 "# Plain\n\ntext\n",
	})
	z, err := newZen(zenOptions{Registry: plugins.NewRegistry(), ConfigDir: t.TempDir(), Dir: dir, Events: &sink{}, Dialogs: &fakeDialogs{}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(z.close)
	return z
}

func sources(p Page) map[string]bool {
	out := map[string]bool{}
	for _, s := range p.Structure {
		out[s.Source+"|"+s.Rule+"|"+s.Severity] = true
	}
	return out
}

func TestStructureTemplateChoice(t *testing.T) {
	z := structureFolder(t)
	page := func(path string) Page {
		t.Helper()
		p, err := z.Page(path)
		if err != nil {
			t.Fatal(err)
		}
		if p.Structure == nil {
			t.Fatalf("%s: Structure must be a list, never null", path)
		}
		return p
	}
	// The folder's path rule.
	if s := sources(page("guides/a.md")); !s["template guide|section|error"] {
		t.Errorf("path rule: %v", s)
	}
	// The page's own template: wins over the path rule.
	b := page("guides/b.md")
	if s := sources(b); !s["template other|section|warning"] || s["template guide|section|error"] {
		t.Errorf("frontmatter template: %v", s)
	}
	// template: false opts out of the page template.
	if p := page("guides/c.md"); len(p.Structure) != 0 {
		t.Errorf("template: false: %+v", p.Structure)
	}
	// An unknown template, and an id that is not a template id at all.
	for _, path := range []string{"guides/d.md", "guides/e.md"} {
		p := page(path)
		if len(p.Structure) != 1 || p.Structure[0].Rule != "template" || !strings.Contains(p.Structure[0].Message, "not found") || p.Structure[0].Fix == "" {
			t.Errorf("%s: %+v", path, p.Structure)
		}
	}
	// A page that follows its template is clean.
	if p := page("guides/ok.md"); len(p.Structure) != 0 {
		t.Errorf("ok: %+v", p.Structure)
	}
	// With no template, the layout's own structure.
	if s := sources(page("blog/x.md")); !s["layout post|fields|error"] {
		t.Errorf("layout structure: %v", s)
	}
	// A page with neither is checked against the article layout.
	if p := page("plain.md"); len(p.Structure) != 0 {
		t.Errorf("plain: %+v", p.Structure)
	}
}

func TestStructureInPagePayloadJSON(t *testing.T) {
	z := structureFolder(t)
	p, err := z.Page("guides/a.md")
	if err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(p)
	var m map[string]any
	_ = json.Unmarshal(b, &m)
	rows, _ := m["structure"].([]any)
	if len(rows) == 0 {
		t.Fatalf("no structure in %s", b)
	}
	row := rows[0].(map[string]any)
	for _, k := range []string{"rule", "message", "fix", "severity", "line", "anchor", "source"} {
		if _, ok := row[k]; !ok {
			t.Errorf("row has no %q: %v", k, row)
		}
	}
}

func TestStructureForPreview(t *testing.T) {
	z := structureFolder(t)
	p, err := z.Preview("guides/a.md", "# A\n\n## Steps\n")
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Structure) != 0 {
		t.Errorf("the preview is checked with the editor's text: %+v", p.Structure)
	}
}

func TestCheckFolder(t *testing.T) {
	dir := tree(t, map[string]string{
		".zen/templates/guide.yml": guideTpl,
		".zenmd.yml":               "templates:\n  \"guides/**\": guide\n",
		"guides/a.md":              "# A\n",
		"guides/ok.md":             "# Ok\n\n## Steps\n",
		"blog/x.md":                "---\nlayout: post\n---\n# X\n",
		"notes.txt":                "not a page",
	})
	cfg := t.TempDir()
	rep, err := CheckFolder(SiteEnv{GlobalPath: cfg + "/config.yml", Registry: plugins.NewRegistry()}, CheckOptions{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	if rep.Pages != 3 {
		t.Errorf("pages = %d", rep.Pages)
	}
	by := map[string][]StructureProblem{}
	for _, r := range rep.Results {
		by[r.Path] = r.Problems
	}
	if len(by["guides/a.md"]) == 0 || len(by["blog/x.md"]) == 0 || len(by["guides/ok.md"]) != 0 {
		t.Errorf("results = %+v", by)
	}
	// --template: every page is checked against it.
	rep, err = CheckFolder(SiteEnv{GlobalPath: cfg + "/config.yml", Registry: plugins.NewRegistry()}, CheckOptions{Dir: dir, Template: "guide"})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, r := range rep.Results {
		if r.Path == "blog/x.md" {
			for _, p := range r.Problems {
				found = found || p.Source == "template guide"
			}
		}
	}
	if !found {
		t.Errorf("--template guide did not apply to blog/x.md: %+v", rep.Results)
	}
	_, err = CheckFolder(SiteEnv{GlobalPath: cfg + "/config.yml", Registry: plugins.NewRegistry()}, CheckOptions{Dir: dir, Template: "nope"})
	if !errors.Is(err, ErrNoPageTemplate) {
		t.Errorf("an unknown --template: %v", err)
	}
}
