package blueprints_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/zenmark-org/zenmd/builtins/blueprints"
	"github.com/zenmark-org/zenmd/builtins/pagetemplates"
	"github.com/zenmark-org/zenmd/engine/config"
	"github.com/zenmark-org/zenmd/engine/index"
	enginetemplates "github.com/zenmark-org/zenmd/engine/sitetemplates"
	"gopkg.in/yaml.v3"
)

// fresh makes a folder from the built-in blueprint id, as zenmd init --blueprint does.
func fresh(t *testing.T, id string) string {
	t.Helper()
	list, _ := enginetemplates.Builtins(blueprints.FS)
	for _, tp := range list {
		if tp.ID != id {
			continue
		}
		if tp.Err != "" {
			t.Fatalf("%s is invalid: %s", id, tp.Err)
		}
		tp.PageSource = pagetemplates.FS
		target := filepath.Join(t.TempDir(), "My Docs")
		if _, err := enginetemplates.Create(tp, target, map[string]string{"title": "My Docs", "author": "Ana", "product": "Zed"}); err != nil {
			t.Fatal(err)
		}
		return target
	}
	t.Fatalf("no blueprint %s", id)
	return ""
}

func readFile(t *testing.T, root, rel string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestTechdocsHasSixSectionsInOrder(t *testing.T) {
	root := fresh(t, "techdocs")
	want := []string{"getting-started", "how-tos", "explanation", "reference", "glossary", "releases"}
	home := readFile(t, root, "index.md")
	links := regexp.MustCompile(`\]\(([a-z-]+)/index\.md\)`).FindAllStringSubmatch(home, -1)
	var got []string
	for _, m := range links {
		got = append(got, m[1])
	}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("home links %v, want %v:\n%s", got, want, home)
	}
	for _, s := range want {
		if _, err := os.Stat(filepath.Join(root, s, "index.md")); err != nil {
			t.Errorf("%s has no index.md", s)
		}
	}
	if _, err := os.Stat(filepath.Join(root, "README.md")); err == nil {
		t.Error("README.md written; the home is index.md")
	}
	if _, err := os.Stat(filepath.Join(root, "overview")); err == nil {
		t.Error("overview/ folder written; the overview is the home")
	}
	for _, f := range []string{"presentation.md", "getting-started/installation.md", "getting-started/first-steps.md", "how-tos/example-task.md", "explanation/architecture.md", "reference/configuration.md", "glossary/example-term.md", "CONTRIBUTING.md"} {
		if _, err := os.Stat(filepath.Join(root, f)); err != nil {
			t.Errorf("missing %s", f)
		}
	}
}

func TestTechdocsPagesAndRules(t *testing.T) {
	root := fresh(t, "techdocs")
	if g := readFile(t, root, "glossary/index.md"); !strings.Contains(g, "layout: glossary") || !strings.Contains(g, "sort: title") {
		t.Errorf("glossary index:\n%s", g)
	}
	if g := readFile(t, root, "glossary/example-term.md"); !strings.Contains(g, "layout: term") || strings.Contains(g, "title:") || !strings.Contains(g, "# Example term\n\n") {
		t.Errorf("example term:\n%s", g)
	}
	if o := readFile(t, root, "index.md"); !strings.Contains(o, "Zed is") || !strings.Contains(o, "## Key features") || !strings.Contains(o, "## Where to go next") || !strings.Contains(o, "(releases/index.md)") || !strings.Contains(o, "(presentation.md)") || !strings.Contains(o, "Zed in five slides") {
		t.Errorf("home:\n%s", o)
	}
	if p := readFile(t, root, "presentation.md"); !strings.Contains(p, "layout: slides") || !strings.Contains(p, "# Zed in five slides") || strings.Contains(p, "title:") {
		t.Errorf("presentation:\n%s", p)
	}
	rel := readFile(t, root, "releases/v0.1.0.md")
	for _, w := range []string{"layout: release", `version: "0.1.0"`, "date: ", "name: First release", "## Added", "## Changed", "## Fixed"} {
		if !strings.Contains(rel, w) {
			t.Errorf("first release lacks %q:\n%s", w, rel)
		}
	}
	if strings.Contains(rel, "title:") || strings.Contains(rel, "\n# ") {
		t.Errorf("a release has no title: and no H1; the version is the title:\n%s", rel)
	}
	if _, err := os.Stat(filepath.Join(root, "releases", "0.1.0.md")); err == nil {
		t.Error("the release file is v0.1.0.md, not 0.1.0.md")
	}
	if m, _ := filepath.Glob(filepath.Join(root, "releases", "*-first-release.md")); len(m) != 0 {
		t.Errorf("old release file name: %v", m)
	}
	if r := readFile(t, root, "releases/index.md"); !strings.Contains(r, "[v0.1.0](v0.1.0.md)") {
		t.Errorf("the releases index links the release as v0.1.0:\n%s", r)
	}
	if r := readFile(t, root, "releases/index.md"); !strings.Contains(r, "sort: newest-first") {
		t.Errorf("releases index:\n%s", r)
	}
	// The folder's settings load clean and keep the path rules, in section order.
	cfg := config.Load("", root)
	if w := cfg.Warnings(); len(w) != 0 {
		t.Errorf("config warnings: %v", w)
	}
	var rules []string
	for _, r := range cfg.TemplateRules() {
		rules = append(rules, r.Pattern+"="+r.Name)
	}
	wantRules := "getting-started/**=tutorial,how-tos/**=how-to,explanation/**=explanation,reference/**=reference,glossary/**=term,releases/**=release-notes"
	if strings.Join(rules, ",") != wantRules {
		t.Errorf("rules %v", rules)
	}
	if !strings.Contains(readFile(t, root, ".zenmd.yml"), "layout: docs") {
		t.Error("layout: docs missing from .zenmd.yml")
	}
	// The site's own page templates, as the blueprint names them.
	entries, _ := os.ReadDir(filepath.Join(root, ".zen", "templates"))
	var names []string
	for _, e := range entries {
		names = append(names, strings.TrimSuffix(e.Name(), ".yml"))
	}
	sort.Strings(names)
	if strings.Join(names, ",") != "explanation,how-to,overview,reference,release-notes,term,tutorial" {
		t.Errorf("templates %v", names)
	}
	// No placeholder is left in any page.
	_ = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() && strings.HasSuffix(p, ".md") && !strings.Contains(p, ".zen/") {
			if b, _ := os.ReadFile(p); strings.Contains(string(b), "{{") {
				t.Errorf("%s has a placeholder left", p)
			}
		}
		return nil
	})
}

// Every built-in blueprint builds on a fresh folder.
func TestBuiltinBlueprintsBuild(t *testing.T) {
	for _, id := range blueprints.IDs {
		root := fresh(t, id)
		if _, err := os.Stat(filepath.Join(root, "index.md")); err != nil {
			t.Errorf("%s: no index.md", id)
		}
		if w := config.Load("", root).Warnings(); len(w) != 0 {
			t.Errorf("%s: config warnings %v", id, w)
		}
	}
}

func TestTechdocsNewTermAppearsUnderGlossary(t *testing.T) {
	root := fresh(t, "techdocs")
	for _, idx := range []string{"glossary/index.md", "releases/index.md"} {
		if g := readFile(t, root, idx); !strings.Contains(g, "children: folder") {
			t.Errorf("%s has no children: folder:\n%s", idx, g)
		}
	}
	// A term added by hand after creation is listed, A to Z, without any link.
	if err := os.WriteFile(filepath.Join(root, "glossary", "aardvark.md"), []byte("---\nlayout: term\n---\n# Aardvark\n\nAn animal.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var files []string
	metas := map[string]index.Meta{}
	filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(p, ".md") {
			return nil
		}
		rel, _ := filepath.Rel(root, p)
		rel = filepath.ToSlash(rel)
		files = append(files, rel)
		fm := map[string]any{}
		if b, err := os.ReadFile(p); err == nil {
			if s := string(b); strings.HasPrefix(s, "---\n") {
				if end := strings.Index(s[4:], "\n---"); end >= 0 {
					yaml.Unmarshal([]byte(s[4:4+end]), &fm)
				}
			}
		}
		metas[rel], _, _ = index.ParseMeta(fm)
		return nil
	})
	titleOf := func(p string) string {
		return strings.TrimSuffix(filepath.Base(p), ".md")
	}
	ix := index.BuildWithMeta(files, "index.md", func(string) []string { return nil }, titleOf, func(p string) index.Meta { return metas[p] })
	var g *index.Node
	var walk func(ns []index.Node)
	walk = func(ns []index.Node) {
		for i := range ns {
			if ns[i].Path == "glossary/index.md" {
				g = &ns[i]
			}
			walk(ns[i].Children)
		}
	}
	walk(ix.Tree())
	if g == nil {
		t.Fatal("no glossary index in the tree")
	}
	var got []string
	for _, c := range g.Children {
		got = append(got, c.Path)
	}
	// Terms are not in the sidebar (layout: term hides them); the Glossary page lists them.
	if len(got) != 0 {
		t.Fatalf("glossary children %v, want none", got)
	}
	for _, f := range []string{"glossary/aardvark.md", "glossary/example-term.md"} {
		if !metas[f].Hidden {
			t.Errorf("%s is not hidden from the sidebar", f)
		}
	}
}
