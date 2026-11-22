package session_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zenmark-org/zenmd/engine/session"
	"github.com/zenmark-org/zenmd/engine/session/sessiontest"
)

// h1Term is a term as New Term writes it: no title in the frontmatter.
func h1Term(title, def, extra string) string {
	return "---\nlayout: term\n" + extra + "---\n# " + title + "\n\n" + def + "\n"
}

func h1Tree(t *testing.T) string {
	return sessiontest.Tree(t, map[string]string{
		"README.md":     "# Home\n\n[Glossary](g/index.md)\n",
		"g/index.md":    "---\ntitle: Glossary\nlayout: glossary\n---\nWords.\n",
		"g/zebra.md":    h1Term("Zebra", "An animal with stripes.", ""),
		"g/idem.md":     h1Term("Idempotent", "Doing it twice is doing it once.", "aliases: [idempotence]\n"),
		"g/old.md":      termSrc("Apple", "A fruit.", ""), // the earlier form: title: in the frontmatter
		"g/shown.md":    h1Term("Shown", "Stays in the sidebar.", "nav: { hidden: false }\n"),
		"other/idem.md": h1Term("Elsewhere", "Not here.", ""),
	})
}

func TestTermTitleIsTheHeading(t *testing.T) {
	s, _ := openGlossary(t, h1Tree(t))
	idx, err := s.Page("g/index.md")
	if err != nil || len(idx.Warnings) != 0 {
		t.Fatalf("%v %v", err, idx.Warnings)
	}
	// Sorted A to Z by the H1 (or the frontmatter title when there is no H1), with the definition after the H1.
	a, i, sh, z := strings.Index(idx.HTML, ">Apple<"), strings.Index(idx.HTML, ">Idempotent<"), strings.Index(idx.HTML, ">Shown<"), strings.Index(idx.HTML, ">Zebra<")
	if a < 0 || a > i || i > sh || sh > z {
		t.Errorf("order %d %d %d %d: %s", a, i, sh, z, idx.HTML)
	}
	for _, want := range []string{"4 terms", "Doing it twice is doing it once.", "An animal with stripes.", "A fruit.", "idempotence"} {
		if !strings.Contains(idx.HTML, want) {
			t.Errorf("index lacks %q", want)
		}
	}
	if strings.Contains(idx.HTML, "idem") && strings.Contains(idx.HTML, "Elsewhere") {
		t.Errorf("a page of another folder is listed")
	}
	// The New Term… button is in the page, hidden: the app shows it, an exported site never does.
	if !strings.Contains(idx.HTML, `data-zen-action="new-term" data-folder="g"`) {
		t.Errorf("no New Term button: %s", idx.HTML)
	}
	// The term page shows its H1 once and no warning.
	p, err := s.Page("g/idem.md")
	if err != nil || len(p.Warnings) != 0 {
		t.Fatalf("%v %v", err, p.Warnings)
	}
	if n := strings.Count(p.HTML, "Idempotent"); n != 1 {
		t.Errorf("the term shows %d times: %s", n, p.HTML)
	}
	if !strings.Contains(p.HTML, "Doing it twice is doing it once.") || !strings.Contains(p.HTML, "zl-term-prev") || !strings.Contains(p.HTML, "zl-term-next") {
		t.Errorf("%s", p.HTML)
	}
}

func TestDuplicateTermsUseTheHeading(t *testing.T) {
	dir := sessiontest.Tree(t, map[string]string{
		"g/index.md": "---\ntitle: Glossary\nlayout: glossary\n---\n",
		"g/a.md":     h1Term("Same", "One.", ""),
		"g/b.md":     h1Term("same", "Two.", ""),
	})
	s, _ := openGlossary(t, dir)
	p, _ := s.Page("g/index.md")
	if len(p.Warnings) == 0 || !strings.Contains(strings.Join(p.Warnings, "|"), "duplicate term") {
		t.Errorf("warnings %v", p.Warnings)
	}
}

func TestTermPagesAreHiddenFromTheSidebar(t *testing.T) {
	s, _ := openGlossary(t, h1Tree(t))
	var find func(ns []session.NavNode, p string) *session.NavNode
	find = func(ns []session.NavNode, p string) *session.NavNode {
		for i := range ns {
			if ns[i].Path == p {
				return &ns[i]
			}
			if n := find(ns[i].Children, p); n != nil {
				return n
			}
		}
		return nil
	}
	tree := s.Startup().Workspace.Tree
	g := find(tree, "g/index.md")
	if g == nil {
		t.Fatalf("no Glossary page in %+v", tree)
	}
	if find(g.Children, "g/idem.md") != nil || find(g.Children, "g/zebra.md") != nil || find(g.Children, "g/old.md") != nil {
		t.Errorf("terms are in the sidebar: %+v", g.Children)
	}
	if find(g.Children, "g/shown.md") == nil {
		t.Errorf("nav.hidden: false did not win: %+v", g.Children)
	}
	// Still reachable, and the breadcrumb is Glossary > Term.
	p, err := s.Page("g/zebra.md")
	if n := len(p.Breadcrumb); err != nil || n < 2 || p.Breadcrumb[n-2].Path != "g/index.md" || p.Breadcrumb[n-1].Path != "g/zebra.md" {
		t.Errorf("%v breadcrumb %+v", err, p.Breadcrumb)
	}
}

func TestGlossaryFolders(t *testing.T) {
	g, err := session.OpenGlossaryFiles("", h1Tree(t))
	if err != nil {
		t.Fatal(err)
	}
	got, err := g.Folders()
	if err != nil || len(got) != 1 || got[0] != "g" {
		t.Fatalf("%v %v", got, err)
	}
	two := sessiontest.Tree(t, map[string]string{
		"a/index.md": "---\nlayout: glossary\n---\n", "b/sub/index.md": "---\nlayout: glossary\n---\n", "c/index.md": "# Plain\n",
	})
	g, _ = session.OpenGlossaryFiles("", two)
	got, _ = g.Folders()
	if strings.Join(got, ",") != "a,b/sub" {
		t.Errorf("folders %v", got)
	}
}

func TestAddTerm(t *testing.T) {
	dir := h1Tree(t)
	g, _ := session.OpenGlossaryFiles("", dir)
	added, err := g.Add("g", "Éclair au café", "")
	if err != nil || added.Path != "g/eclair-au-cafe.md" || added.Existed {
		t.Fatalf("%+v %v", added, err)
	}
	b, _ := os.ReadFile(filepath.Join(dir, "g", "eclair-au-cafe.md"))
	if !strings.HasPrefix(string(b), "---\nlayout: term\n---\n# Éclair au café\n\n") || strings.Contains(string(b), "title:") {
		t.Errorf("file:\n%s", b)
	}
	// An existing term is never overwritten: the answer says so, so the caller can offer to open it.
	if err := os.WriteFile(filepath.Join(dir, "g", "idem.md"), []byte(h1Term("Idempotent", "Kept.", "")), 0o644); err != nil {
		t.Fatal(err)
	}
	again, err := g.Add("g", "idempotent", "Replaced?")
	if err != nil || !again.Existed || again.Path != "g/idem.md" && again.Path != "g/idempotent.md" {
		t.Fatalf("%+v %v", again, err)
	}
	if b, _ := os.ReadFile(filepath.Join(dir, "g", "idem.md")); !strings.Contains(string(b), "Kept.") {
		t.Errorf("an existing term was changed:\n%s", b)
	}
	// The same slug as a file that exists.
	if err := os.WriteFile(filepath.Join(dir, "g", "same.md"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	same, err := g.Add("g", "Same", "")
	if err != nil || !same.Existed || same.Path != "g/same.md" {
		t.Fatalf("%+v %v", same, err)
	}
	// A definition replaces the skeleton's placeholder.
	d, err := g.Add("g", "Defined", "It is defined.")
	b, _ = os.ReadFile(filepath.Join(dir, "g", "defined.md"))
	if err != nil || !strings.Contains(string(b), "# Defined\n\nIt is defined.\n") {
		t.Errorf("%v\n%s", err, b)
	}
	_ = d
	for _, bad := range []string{"", "   ", "???"} {
		if _, err := g.Add("g", bad, ""); err == nil {
			t.Errorf("%q: no error", bad)
		}
	}
	if _, err := g.Add("../out", "X", ""); err == nil {
		t.Error("a folder outside the workspace is refused")
	}
}

func TestListTerms(t *testing.T) {
	g, _ := session.OpenGlossaryFiles("", h1Tree(t))
	got, err := g.List("g")
	if err != nil {
		t.Fatal(err)
	}
	var titles []string
	for _, e := range got {
		titles = append(titles, e.Title)
	}
	if strings.Join(titles, ",") != "Apple,Idempotent,Shown,Zebra" {
		t.Errorf("terms %v", titles)
	}
	if got[1].Definition != "Doing it twice is doing it once." || got[1].Path != "g/idem.md" {
		t.Errorf("%+v", got[1])
	}
}

// The session does the same through the open folder, and the new term shows in the index at once.
func TestSessionNewTerm(t *testing.T) {
	s, _ := openGlossary(t, h1Tree(t))
	r, err := s.NewTerm("g", "Yankee")
	if err != nil || r.Path != "g/yankee.md" || r.Existed {
		t.Fatalf("%+v %v", r, err)
	}
	if err := s.Save(r.Path, "---\nlayout: term\n---\n# Yankee\n\nA letter.\n"); err != nil {
		t.Fatal(err)
	}
	waitFor(t, "the new term in the index", func() bool {
		idx, _ := s.Page("g/index.md")
		return strings.Contains(idx.HTML, ">Yankee<")
	})
	if again, err := s.NewTerm("g", "Yankee"); err != nil || !again.Existed {
		t.Errorf("%+v %v", again, err)
	}
	folders, err := s.GlossaryFolders()
	if err != nil || len(folders) != 1 || folders[0] != "g" {
		t.Errorf("%v %v", folders, err)
	}
}
