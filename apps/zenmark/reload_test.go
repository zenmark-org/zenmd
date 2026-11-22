package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReloadRescansTheFolder(t *testing.T) {
	dir := tree(t, map[string]string{"README.md": "# Home\n"})
	z, _, _, _ := newTestZen(t, dir, "README.md", "read")
	if err := os.WriteFile(filepath.Join(dir, "later.md"), []byte("# Later\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	ws, err := z.Reload()
	if err != nil || ws == nil {
		t.Fatalf("Reload = %v, %v", ws, err)
	}
	found := false
	for _, n := range ws.Tree {
		found = found || n.Path == "later.md"
		for _, c := range n.Children {
			found = found || c.Path == "later.md"
		}
	}
	if !found {
		t.Fatalf("later.md is not in the tree: %+v", ws.Tree)
	}
}

func TestReloadWithNoFolder(t *testing.T) {
	z, _, _, _ := newTestZen(t, "", "", "")
	ws, err := z.Reload()
	if err != nil || ws != nil {
		t.Fatalf("Reload = %v, %v", ws, err)
	}
}
