package layouts

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zenmark-org/zenmd/engine/config"
	"github.com/zenmark-org/zenmd/engine/structure"
)

func layoutStructure(t *testing.T, id string) *structure.Template {
	t.Helper()
	for _, l := range All() {
		if l.Manifest().ID == id {
			if len(l.Manifest().Structure) == 0 {
				return nil
			}
			tpl, err := structure.ParseLayout(id, l.Manifest().Structure)
			if err != nil {
				t.Fatalf("%s: %v", id, err)
			}
			return tpl
		}
	}
	t.Fatalf("no layout %s", id)
	return nil
}

// Every built-in layout's structure: block parses, and the ones the spec lists have one.
func TestLayoutStructuresParse(t *testing.T) {
	for _, id := range []string{"article", "docs", "post", "landing", "handout", "slides", "course", "term", "release"} {
		if layoutStructure(t, id) == nil {
			t.Errorf("%s has no structure:", id)
		}
	}
	for _, id := range IDs {
		layoutStructure(t, id) // parses, or fails the test
	}
}

// The demo's layout pages follow their layouts: not a single problem, suggestions included.
func TestDemoLayoutPagesAreClean(t *testing.T) {
	root := filepath.Join("..", "..", "examples", "demo")
	n := 0
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(p, ".md") {
			return err
		}
		src, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		meta, _, _ := config.SplitFrontmatter(src)
		id, _ := meta["layout"].(string)
		if id == "" {
			id = "article"
		}
		n++
		for _, pr := range structure.Check(structure.Analyse(src), nil, layoutStructure(t, id)) {
			rel, _ := filepath.Rel(root, p)
			t.Errorf("%s:%d: %s %s: %s", rel, pr.Line, pr.Severity, pr.Rule, pr.Message)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if n < 20 {
		t.Errorf("only %d demo pages were checked", n)
	}
}

// A broken page reports what the spec's table says for its layout.
func TestLayoutStructureReportsGaps(t *testing.T) {
	words := strings.Repeat("word ", 900)
	cases := []struct {
		layout, page string
		want         []string // "severity rule" each
	}{
		{"article", "No heading here.\n", []string{"suggestion heading"}},
		{"article", "# T\n\n#### Deep\n\n![](a.png)\n", []string{"warning heading-order", "warning image-alt"}},
		{"docs", "# T\n\n" + words + "\n", []string{"suggestion needs-sections"}},
		{"post", "# P\n\nSummary.\n", []string{"error fields", "warning fields"}},                                 // date, author
		{"post", "---\ndate: 2026-11-17\nauthor: A\ntags: x\n---\n# P\n\nSummary.\n", []string{"warning fields"}}, // tags is not a list
		{"post", "---\ndate: 2026-11-17\nauthor: A\n---\n# P\n\n" + words + "\n", []string{"warning paragraph"}},
		{"landing", "---\ntagline: x\n---\n# L\n\n" + words + "\n", []string{"suggestion max-words"}},
		{"landing", "# L\n", []string{"suggestion fields"}},
		{"handout", "# H\n\n" + strings.Repeat("word ", 700) + "\n", []string{"suggestion max-words"}},
		{"slides", "# One\n\n- a\n", []string{"suggestion slide"}},
		{"slides", "# One\n\n- 1\n- 2\n- 3\n- 4\n- 5\n- 6\n- 7\n\n---\n\n# Two\n", []string{"suggestion slide"}},
		{"course", "---\nlayout: course\n---\n# C\n\n## One\n\ntext\n", []string{"warning fields", "warning section"}},
		{"course", "---\nlayoutOptions: { id: c }\n---\n# C\n\n## One\n\ntext\n\n## Two\n\ntext\n", []string{"warning block"}},
		{"course", "---\nlayoutOptions: { id: c }\n---\n# C\n\n## One\n\n:::check\n- [ ] a\n:::\n\n## Two\n\n:::quiz{id=a}\n:::\n\n:::quiz{id=a}\n:::\n", []string{"warning block"}},
		{"term", "# Term\n\n" + strings.Repeat("def ", 70) + "\n", []string{"warning paragraph"}},
		{"release", "---\nlayout: release\n---\n## Added\n", []string{"error fields"}},
	}
	for i, c := range cases {
		got := structure.Check(structure.Analyse([]byte(c.page)), nil, layoutStructure(t, c.layout))
		have := map[string]bool{}
		for _, p := range got {
			have[p.Severity+" "+p.Rule] = true
			if p.Source != "layout "+c.layout {
				t.Errorf("case %d: source %q", i, p.Source)
			}
		}
		for _, w := range c.want {
			if !have[w] {
				t.Errorf("case %d (%s): want %q, got %+v", i, c.layout, w, got)
			}
		}
	}
}
