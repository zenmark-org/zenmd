package index

import (
	"reflect"
	"testing"
)

func childPaths(ns []Node) []string {
	var out []string
	for _, n := range ns {
		out = append(out, n.Path)
	}
	return out
}

func find(ns []Node, p string) *Node {
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

func TestChildrenFolder(t *testing.T) {
	files := []string{"README.md", "g/index.md", "g/b.md", "g/a.md", "g/c.md", "g/hide.md", "g/sub/x.md", "other.md"}
	links := map[string][]string{"README.md": {"g/index.md", "g/c.md"}, "g/c.md": {"g/a.md"}}
	m := map[string]Meta{"g/index.md": {Children: "folder"}, "g/hide.md": {Hidden: true}}
	ix := BuildWithMeta(files, "README.md", graph(links), title, metas(m))
	g := find(ix.Tree(), "g/index.md")
	if g == nil {
		t.Fatalf("no index in %+v", ix.Tree())
	}
	// c is linked by README (earlier than the index's folder rule) and keeps its place; a stays under c.
	if got := childPaths(g.Children); !reflect.DeepEqual(got, []string{"g/b.md"}) {
		t.Fatalf("children %v", got)
	}
	if find(ix.Tree(), "g/hide.md") != nil {
		t.Fatal("hidden page shown")
	}
	if find(ix.Tree(), "g/sub/x.md") == nil {
		t.Fatal("subfolder page lost")
	}
	if p := find(g.Children, "g/sub/x.md"); p != nil {
		t.Fatal("subfolder page is a child of the index")
	}
	if bc := ix.Breadcrumb("g/b.md"); len(bc) != 3 || bc[2].Path != "g/b.md" || bc[1].Path != "g/index.md" {
		t.Fatalf("crumb %+v", bc)
	}
}

func TestChildrenFolderReadmeAndLinkedChild(t *testing.T) {
	files := []string{"README.md", "d/README.md", "d/x.md", "d/y.md"}
	links := map[string][]string{"README.md": {"d/README.md"}, "d/README.md": {"d/y.md"}}
	m := map[string]Meta{"d/README.md": {Children: "folder"}}
	d := find(BuildWithMeta(files, "README.md", graph(links), title, metas(m)).Tree(), "d/README.md")
	if got := childPaths(d.Children); !reflect.DeepEqual(got, []string{"d/y.md", "d/x.md"}) {
		t.Fatalf("children %v", got)
	}
}

func TestChildrenOffByDefault(t *testing.T) {
	files := []string{"README.md", "g/index.md", "g/b.md"}
	links := map[string][]string{"README.md": {"g/index.md"}}
	g := find(BuildWithMeta(files, "README.md", graph(links), title, metas(nil)).Tree(), "g/index.md")
	if len(g.Children) != 0 {
		t.Fatalf("children %+v", g.Children)
	}
}

func TestSortTitle(t *testing.T) {
	files := []string{"README.md", "g/index.md", "g/1.md", "g/2.md", "g/3.md", "g/4.md"}
	links := map[string][]string{"README.md": {"g/index.md"}}
	titles := map[string]string{"g/1.md": "zebra", "g/2.md": "Äpfel", "g/3.md": "apple", "g/4.md": "Éclair"}
	ti := func(p string) string {
		if s, ok := titles[p]; ok {
			return s
		}
		return "T:" + p
	}
	m := map[string]Meta{"g/index.md": {Children: "folder", Sort: "title"}, "g/1.md": {Order: ip(1)}}
	g := find(BuildWithMeta(files, "README.md", graph(links), ti, metas(m)).Tree(), "g/index.md")
	// apple and Äpfel fold to apple / apfel: apfel < apple.
	if got := childPaths(g.Children); !reflect.DeepEqual(got, []string{"g/2.md", "g/3.md", "g/4.md", "g/1.md"}) {
		t.Fatalf("children %v (sort decides over order)", got)
	}
}

func TestSortNewestFirst(t *testing.T) {
	files := []string{"README.md", "r/index.md", "r/a.md", "r/b.md", "r/c.md", "r/d.md", "r/e.md"}
	links := map[string][]string{"README.md": {"r/index.md"}}
	m := map[string]Meta{
		"r/index.md": {Children: "folder", Sort: "newest-first"},
		"r/a.md":     {Date: "2024-01-01", Version: "1.2.0"},
		"r/b.md":     {Date: "2024-01-01", Version: "1.10.0"},
		"r/c.md":     {Date: "2025-03-01", Version: "0.1.0"},
		"r/d.md":     {},
		"r/e.md":     {Date: "2024-01-01", Version: "1.10.0"},
	}
	r := find(BuildWithMeta(files, "README.md", graph(links), title, metas(m)).Tree(), "r/index.md")
	if got := childPaths(r.Children); !reflect.DeepEqual(got, []string{"r/c.md", "r/e.md", "r/b.md", "r/a.md", "r/d.md"}) {
		t.Fatalf("children %v", got)
	}
}

func TestParseMetaChildrenDateVersion(t *testing.T) {
	m, _, w := ParseMeta(map[string]any{"date": "2024-05-06", "version": "1.2.3", "nav": map[string]any{"children": "folder", "sort": "title"}})
	if len(w) != 0 || m.Children != "folder" || m.Sort != "title" || m.Date != "2024-05-06" || m.Version != "1.2.3" {
		t.Fatalf("%+v %v", m, w)
	}
	_, _, w = ParseMeta(map[string]any{"nav": map[string]any{"children": "all"}})
	if !hasWarning(w, "nav.children") {
		t.Fatalf("warnings %v", w)
	}
	// The glossary layout implies both keys unless the page sets them.
	m, _, _ = ParseMeta(map[string]any{"layout": "glossary"})
	if m.Children != "folder" || m.Sort != "title" {
		t.Fatalf("glossary %+v", m)
	}
	m, _, _ = ParseMeta(map[string]any{"layout": "glossary", "nav": map[string]any{"sort": "newest-first"}})
	if m.Children != "folder" || m.Sort != "newest-first" {
		t.Fatalf("glossary override %+v", m)
	}
}
