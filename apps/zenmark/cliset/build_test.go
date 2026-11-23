package cliset

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/zenmark-org/zenmd/internal/cli"
)

// A folder's .zenmd.yml can't aim the site outside the folder: the build goes
// through the same session.WriteSite and config.Load as the app's, so the rule
// holds here too.
func TestOutputStaysInFolder(t *testing.T) {
	fakeHome(t)
	base := t.TempDir()
	dir := filepath.Join(base, "site src")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for name, text := range map[string]string{
		"index.md":   "# Home\n\nHello.\n",
		".zenmd.yml": "export:\n  out: ../elsewhere\n",
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	var out, errOut bytes.Buffer
	_ = BuildSite(cli.Build{Dir: dir}, &out, &errOut)
	if _, err := os.Stat(filepath.Join(base, "elsewhere")); err == nil {
		t.Errorf("the site was written outside the folder\nstdout: %s\nstderr: %s", out.String(), errOut.String())
	}
	// A path with spaces builds, inside the folder.
	if err := os.Remove(filepath.Join(dir, ".zenmd.yml")); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	errOut.Reset()
	if err := BuildSite(cli.Build{Dir: dir}, &out, &errOut); err != nil {
		t.Fatalf("%v\n%s", err, errOut.String())
	}
	if _, err := os.Stat(filepath.Join(dir, "exports", "html", "index.html")); err != nil {
		t.Errorf("no index.html in the default output: %v\n%s", err, out.String())
	}
}
