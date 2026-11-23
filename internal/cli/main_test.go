package cli

import (
	"os"
	"path/filepath"
	"testing"
)

// TestMain is a safety net: Run moves the old settings folder, so no test may
// see the real home, whether or not it calls home(t).
func TestMain(m *testing.M) {
	h, err := os.MkdirTemp("", "zenmd-home")
	if err != nil {
		panic(err)
	}
	os.Setenv("HOME", h)
	os.Setenv("XDG_CONFIG_HOME", filepath.Join(h, ".config"))
	code := m.Run()
	os.RemoveAll(h)
	os.Exit(code)
}
