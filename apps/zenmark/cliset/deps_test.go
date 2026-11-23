package cliset

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

// cliset is what the command-line-only zenmd is made of: it must never reach
// Wails, kit/desktop or the app, or the CLI build would need cgo again.
func TestClisetImports(t *testing.T) {
	cmd := exec.Command("go", "list", "-deps", "-f", "{{.ImportPath}}", ".")
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go list: %v\n%s", err, out)
	}
	for _, p := range strings.Fields(string(out)) {
		if strings.Contains(p, "wailsapp") || p == "github.com/zenmark-org/zenmd/kit/desktop" || p == "github.com/zenmark-org/zenmd/apps/zenmark" {
			t.Errorf("cliset imports %s", p)
		}
	}
}
