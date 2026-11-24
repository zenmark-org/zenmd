package session

import (
	"os"
	"path/filepath"
	"testing"
)

// filesDocs: README links a.md and sub/s.md (the Pages tree), loose.md is linked by nothing (Other pages),
// secret.md and under.md are hidden (under.md because its parent is).
var filesDocs = map[string]string{
	"README.md":      "# Home\n\n[A](a.md) [S](sub/s.md) [Secret](secret.md)\n",
	"a.md":           "---\nnav:\n  title: Alpha\n---\n# A page\n",
	"sub/s.md":       "# Sub page\n",
	"loose.md":       "# Loose\n",
	"loose/x.md":     "# X\n",
	"secret.md":      "---\nnav:\n  hidden: true\n---\n# Secret\n\n[U](under.md)\n",
	"under.md":       "# Under\n",
	"img/a.png":      "png",
	"img/b.png":      "png",
	"notes/data.txt": "no",
}

func nodeAt(ns []FileNode, path string) *FileNode {
	for i := range ns {
		if ns[i].Path == path {
			return &ns[i]
		}
		if r := nodeAt(ns[i].Children, path); r != nil {
			return r
		}
	}
	return nil
}

func TestFileTreeMarks(t *testing.T) {
	dir := tree(t, filesDocs)
	z, _, _, _ := newTestZen(t, dir, "", "")
	ft, err := z.FileTree()
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{
		"README.md": "tree", "a.md": "tree", "sub/s.md": "tree",
		"loose.md": "other", "loose/x.md": "other",
		"secret.md": "hidden", "under.md": "hidden",
	}
	for p, nav := range want {
		n := nodeAt(ft.Nodes, p)
		if n == nil || n.Kind != "page" || n.Nav != nav {
			t.Errorf("%s: %+v, want nav %s", p, n, nav)
		}
	}
	for _, p := range []string{"img/", "img/a.png", "sub/", "notes/"} {
		n := nodeAt(ft.Nodes, p)
		if p == "notes/" {
			if n != nil {
				t.Errorf("notes/ has no page or image, got %+v", n)
			}
			continue
		}
		if n == nil || n.Nav != "" {
			t.Errorf("%s: %+v, want no mark", p, n)
		}
	}
	if n := nodeAt(ft.Nodes, "img/a.png"); n.Kind != "image" {
		t.Errorf("kind %q", n.Kind)
	}
	if n := nodeAt(ft.Nodes, "img/"); n.Kind != "folder" || len(n.Children) != 2 {
		t.Errorf("folder %+v", n)
	}
}

func TestFileTreeTitles(t *testing.T) {
	dir := tree(t, filesDocs)
	z, _, _, _ := newTestZen(t, dir, "", "")
	ft, err := z.FileTree()
	if err != nil {
		t.Fatal(err)
	}
	// A page's title is the one the Pages tree shows; a hidden page, an image and a folder show their name.
	titleOf := map[string]string{}
	var walk func([]NavNode)
	walk = func(ns []NavNode) {
		for _, n := range ns {
			if n.Kind == "page" {
				titleOf[n.Path] = n.Title
			}
			walk(n.Children)
		}
	}
	walk(z.Workspace().Tree)
	if titleOf["a.md"] == "" || titleOf["a.md"] != nodeAt(ft.Nodes, "a.md").Title {
		t.Errorf("a.md: tree %q, files %q", titleOf["a.md"], nodeAt(ft.Nodes, "a.md").Title)
	}
	for p, title := range titleOf {
		if got := nodeAt(ft.Nodes, p).Title; got != title {
			t.Errorf("%s: title %q, tree %q", p, got, title)
		}
	}
	for p, name := range map[string]string{"secret.md": "secret.md", "img/a.png": "a.png", "img/": "img"} {
		n := nodeAt(ft.Nodes, p)
		if n.Title != name || n.Name != name {
			t.Errorf("%s: name %q title %q, want %q", p, n.Name, n.Title, name)
		}
	}
}

// The listing reads the index the session built: a page edited on disk afterwards keeps the title and the mark
// the index gave it, so nothing is rendered or read again.
func TestFileTreeUsesIndex(t *testing.T) {
	dir := tree(t, filesDocs)
	z, _, _, _ := newTestZen(t, dir, "", "")
	before, err := z.FileTree()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "a.md"), []byte("---\nnav:\n  hidden: true\n---\n# Renamed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	after, err := z.FileTree()
	if err != nil {
		t.Fatal(err)
	}
	b, a := nodeAt(before.Nodes, "a.md"), nodeAt(after.Nodes, "a.md")
	if a.Title != b.Title || a.Nav != "tree" {
		t.Errorf("a.md was read again: %+v", a)
	}
}

func TestFileTreeNoWorkspace(t *testing.T) {
	s := New(Options{})
	t.Cleanup(s.Close)
	if _, err := s.FileTree(); err != ErrNoWorkspace {
		t.Errorf("got %v", err)
	}
}

func TestFileTreeSingle(t *testing.T) {
	dir := tree(t, map[string]string{"one.md": "# One\n", "two.md": "# Two\n"})
	s := New(Options{})
	t.Cleanup(s.Close)
	if err := s.Open(filepath.Join(dir, "one.md")); err != nil {
		t.Fatal(err)
	}
	ft, err := s.FileTree()
	if err != nil || len(ft.Nodes) != 1 || ft.Nodes[0].Path != "one.md" || ft.Nodes[0].Nav != "tree" {
		t.Errorf("got %+v, %v", ft, err)
	}
}
