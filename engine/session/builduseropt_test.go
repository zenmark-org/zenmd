package session

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zenmark-org/zenmd/engine/plugins"
)

func userOptBuild(t *testing.T, files map[string]string) buildRun {
	t.Helper()
	dir := tree(t, files)
	cfg := t.TempDir()
	if err := os.WriteFile(filepath.Join(cfg, "config.yml"), []byte("version: 1\nplugins:\n  opt:\n    token: SECRET-TOKEN\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	reg := plugins.NewRegistry()
	if err := reg.Register(optPlugin{}); err != nil {
		t.Fatal(err)
	}
	r := runBuild(t, dir, cfg, SiteOptions{}, func(e *SiteEnv) { e.Registry = reg })
	if r.err != nil {
		t.Fatal(r.err)
	}
	return r
}

func countLines(s, sub string) int {
	n := 0
	for _, l := range strings.Split(s, "\n") {
		if strings.Contains(l, sub) {
			n++
		}
	}
	return n
}

// A user-scoped option of the global config is noted once, not on every page.
func TestBuildGlobalUserOptionWarnsOnce(t *testing.T) {
	r := userOptBuild(t, map[string]string{
		"README.md": "# Home\n\n:::opt\nx\n:::\n",
		"a.md":      "# A\n\n:::opt\nx\n:::\n",
		"b.md":      "# B\n\n:::opt\nx\n:::\n",
	})
	if n := countLines(r.stderr, `option "token"`); n != 1 {
		t.Errorf("want one note about the token, got %d:\n%s", n, r.stderr)
	}
	if strings.Contains(r.stderr, "can only be set in the global config") {
		t.Errorf("per-page warning for a global option:\n%s", r.stderr)
	}
}

// A user-scoped option set by a folder still warns.
func TestBuildFolderUserOptionStillWarns(t *testing.T) {
	r := userOptBuild(t, map[string]string{
		"README.md":  "# Home\n\n:::opt\nx\n:::\n",
		".zenmd.yml": "version: 1\nplugins:\n  opt:\n    token: FROM-FOLDER\n",
	})
	if !strings.Contains(r.stderr, "can only be set in the global config") {
		t.Errorf("no warning for a folder-set user option:\n%s", r.stderr)
	}
}
