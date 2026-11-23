package cliset

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zenmark-org/zenmd/engine/config"
	"github.com/zenmark-org/zenmd/internal/cli"
)

// fakeHome points HOME and XDG_CONFIG_HOME into a temp folder and returns the
// settings folder (<config dir>/zenmark).
func fakeHome(t *testing.T) string {
	t.Helper()
	h := t.TempDir()
	t.Setenv("HOME", h)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(h, ".config"))
	g, err := config.GlobalPath()
	if err != nil {
		t.Skip("no user config dir:", err)
	}
	return filepath.Dir(g)
}

// `zenmd init --dir` and `zenmd templates --dir` print the folder and open
// nothing: the reveal is never called.
func TestDirPrintsPath(t *testing.T) {
	cfg := fakeHome(t)
	var revealed []string
	run := InitSiteWith(func(dir string) error { revealed = append(revealed, dir); return nil })

	var out, errOut bytes.Buffer
	if err := run(cli.Init{ShowDir: true, Values: map[string]string{}}, &out, &errOut); err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(cfg, "templates", "blueprints") + "\n"; out.String() != want {
		t.Errorf("init --dir: %q, want %q", out.String(), want)
	}
	out.Reset()
	if err := run(cli.Init{Pages: true, ShowDir: true, Values: map[string]string{}}, &out, &errOut); err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(cfg, "templates") + "\n"; out.String() != want {
		t.Errorf("templates --dir: %q, want %q", out.String(), want)
	}
	if len(revealed) != 0 {
		t.Errorf("reveal was called with %v", revealed)
	}
	// And the plain InitSite (reveal = print the path) does the same.
	out.Reset()
	if err := InitSite(cli.Init{ShowDir: true, Values: map[string]string{}}, &out, &errOut); err != nil || !strings.HasSuffix(strings.TrimSpace(out.String()), filepath.Join("templates", "blueprints")) {
		t.Errorf("InitSite --dir: %q, %v", out.String(), err)
	}
}

func TestNoAppAndNoPDF(t *testing.T) {
	err := NoApp(cli.Launch{Dir: "/x"})
	if err == nil || err.Error() != "the Zenmark app isn't part of this build; zenmd export site <folder> makes a website" {
		t.Errorf("NoApp: %v", err)
	}
	err = NoPDF(cli.ExportPDF{Dir: "/x"}, os.Stdout, os.Stderr)
	if err == nil || err.Error() != "export pdf needs the Zenmark app on macOS" {
		t.Errorf("NoPDF: %v", err)
	}
}
