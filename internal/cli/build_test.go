package cli

import (
	"testing"

	"github.com/zenmark-org/zenmd/kit/addons"
)

// The command-line-only zenmd says so: "zenmd 0.8.0 (command line)"; the
// app's zenmd keeps "zenmd 0.8.0".
func TestVersionBuild(t *testing.T) {
	oldV, oldK, oldA := Version, BuildKind, addonList
	t.Cleanup(func() { Version, BuildKind, addonList = oldV, oldK, oldA })
	addonList = func() []addons.Addon { return nil }
	Version = "0.8.0"

	BuildKind = ""
	if got := VersionLine(); got != "zenmd 0.8.0" {
		t.Errorf("app: %q", got)
	}
	BuildKind = "command line"
	if got := VersionLine(); got != "zenmd 0.8.0 (command line)" {
		t.Errorf("command line: %q", got)
	}
	r := run(t, t.TempDir(), nil, "version")
	if r.stdout != "zenmd 0.8.0 (command line)\n" {
		t.Errorf("zenmd version: %q", r.stdout)
	}
	r = run(t, t.TempDir(), nil, "--version")
	if r.stdout != "zenmd 0.8.0 (command line)\n" {
		t.Errorf("zenmd --version: %q", r.stdout)
	}
}
