package index

import "testing"

func TestTermLayoutHidesFromTheSidebarByDefault(t *testing.T) {
	m, _, _ := ParseMeta(map[string]any{"layout": "term"})
	if !m.Hidden {
		t.Error("a term page is hidden from the sidebar by default")
	}
	m, _, _ = ParseMeta(map[string]any{"layout": "term", "nav": map[string]any{"hidden": false}})
	if m.Hidden {
		t.Error("nav.hidden: false wins")
	}
	m, _, _ = ParseMeta(map[string]any{"layout": "article"})
	if m.Hidden {
		t.Error("other pages stay shown")
	}
}

func TestHiddenTermsKeepTheirBreadcrumb(t *testing.T) {
	files := []string{"README.md", "g/index.md", "g/a.md", "g/b.md"}
	links := map[string][]string{"README.md": {"g/index.md"}}
	m := map[string]Meta{"g/index.md": {Children: "folder", Sort: "title"}, "g/a.md": {Hidden: true}, "g/b.md": {Hidden: false}}
	ix := BuildWithMeta(files, "README.md", graph(links), title, metas(m))
	g := find(ix.Tree(), "g/index.md")
	if g == nil || find(g.Children, "g/a.md") != nil || find(g.Children, "g/b.md") == nil {
		t.Fatalf("tree %+v", ix.Tree())
	}
	var got []string
	for _, c := range ix.Breadcrumb("g/a.md") {
		got = append(got, c.Path)
	}
	if len(got) < 2 || got[len(got)-2] != "g/index.md" {
		t.Errorf("breadcrumb %v", got)
	}
}
