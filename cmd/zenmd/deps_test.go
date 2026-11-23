package main

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

// The command-line zenmd is made without Wails and without cgo, for the four
// targets it ships or is built on.
func TestNoWailsNoCgo(t *testing.T) {
	for _, tg := range []struct{ goos, goarch string }{
		{"linux", "amd64"}, {"linux", "arm64"}, {"darwin", "amd64"}, {"darwin", "arm64"},
	} {
		t.Run(tg.goos+"_"+tg.goarch, func(t *testing.T) {
			env := append(os.Environ(), "CGO_ENABLED=0", "GOOS="+tg.goos, "GOARCH="+tg.goarch)
			cmd := exec.Command("go", "list", "-deps", "-f", "{{.ImportPath}}", ".")
			cmd.Env = env
			out, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("go list: %v\n%s", err, out)
			}
			for _, p := range strings.Fields(string(out)) {
				if strings.Contains(p, "wailsapp") || p == "github.com/zenmark-org/zenmd/kit/desktop" || p == "github.com/zenmark-org/zenmd/apps/zenmark" {
					t.Errorf("%s/%s imports %s", tg.goos, tg.goarch, p)
				}
			}
			if testing.Short() {
				return
			}
			// It also compiles (this is what catches the script helper's Dup2 on linux/arm64).
			b := exec.Command("go", "build", "-o", os.DevNull, ".")
			b.Env = env
			if out, err := b.CombinedOutput(); err != nil {
				t.Errorf("go build: %v\n%s", err, out)
			}
		})
	}
}
