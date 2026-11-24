package app

import "testing"

func TestFileTreeBinding(t *testing.T) {
	dir := tree(t, map[string]string{"README.md": "# Home\n\n[G](docs/g.md)\n", "docs/g.md": "# G\n", "docs/x.png": "png", "other.md": "# O\n"})
	z, _, _, _ := newTestZen(t, dir, "README.md", "read")
	ft, err := z.FileTree()
	if err != nil {
		t.Fatal(err)
	}
	want, _ := z.s.FileTree()
	if len(ft.Nodes) != len(want.Nodes) || len(ft.Nodes) != 3 || ft.Nodes[0].Path != "docs/" || ft.Nodes[0].Children[0].Nav != "tree" {
		t.Errorf("got %+v", ft)
	}
	if _, err := z.CloseFolder(); err != nil {
		t.Fatal(err)
	}
	if _, err := z.FileTree(); err == nil {
		t.Error("no folder is open: want an error")
	}
}
