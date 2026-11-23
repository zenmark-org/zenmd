package cliset

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zenmark-org/zenmd/internal/cli"
)

// The build goes through the same session.WriteSite and config.Load as the
// app's, so its safety checks hold here too: --clean refuses a folder that
// holds files zenmd didn't write, and keeps them.
func TestOutputStaysInFolder(t *testing.T) {
	fakeHome(t)
	base := t.TempDir()
	dir := filepath.Join(base, "site src") // a folder with a space
	outDir := filepath.Join(base, "mine")
	for name, text := range map[string]string{
		filepath.Join(dir, "index.md"):   "# Home\n\nHello.\n",
		filepath.Join(outDir, "own.txt"): "mine",
	} {
		if err := os.MkdirAll(filepath.Dir(name), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(name, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	var out, errOut bytes.Buffer
	err := BuildSite(cli.Build{Dir: dir, Out: outDir, Clean: true}, &out, &errOut)
	if err == nil || !strings.Contains(err.Error(), "will not clean") {
		t.Fatalf("--clean on a folder of someone's own files: %v", err)
	}
	if b, _ := os.ReadFile(filepath.Join(outDir, "own.txt")); string(b) != "mine" {
		t.Error("the file in the output folder was touched")
	}
	// The default output is inside the folder, and a path with spaces builds.
	out.Reset()
	errOut.Reset()
	if err := BuildSite(cli.Build{Dir: dir}, &out, &errOut); err != nil {
		t.Fatalf("%v\n%s", err, errOut.String())
	}
	if _, err := os.Stat(filepath.Join(dir, "exports", "html", "index.html")); err != nil {
		t.Errorf("no index.html in the default output: %v\n%s", err, out.String())
	}
}
