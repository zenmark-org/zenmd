package session

import (
	"os"
	"path/filepath"
	"testing"
)

func hasPage(ns []NavNode, path string) bool {
	for _, n := range ns {
		if n.Path == path || hasPage(n.Children, path) {
			return true
		}
	}
	return false
}

// Reload scans the folder again without the watcher: a file added on disk is in the tree, and
// workspace:changed is sent for it.
func TestReloadRescansTheFolder(t *testing.T) {
	dir := tree(t, map[string]string{"README.md": "# Home\n"})
	z, ev, _, _ := newTestZen(t, dir, "", "")
	if err := os.WriteFile(filepath.Join(dir, "new.md"), []byte("# New\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	ev.reset()
	ws := z.Reload()
	if ws == nil || !hasPage(ws.Tree, "new.md") {
		t.Fatalf("Reload tree lacks new.md: %+v", ws)
	}
	if !hasPage(z.Workspace().Tree, "new.md") {
		t.Fatal("the session still has the old tree")
	}
	if changedEvents(ev) != 1 {
		t.Fatalf("workspace:changed sent %d times, want 1", changedEvents(ev))
	}
	// Nothing changed since: the same tree, no second event.
	ev.reset()
	if ws := z.Reload(); ws == nil || !hasPage(ws.Tree, "new.md") {
		t.Fatalf("second Reload: %+v", ws)
	}
	if changedEvents(ev) != 0 {
		t.Fatalf("an unchanged tree sent workspace:changed %d times", changedEvents(ev))
	}
}

func TestReloadWithoutFolder(t *testing.T) {
	z, _, _, _ := newTestZen(t, "", "", "")
	if ws := z.Reload(); ws != nil {
		t.Fatalf("Reload with no folder = %+v", ws)
	}
}
