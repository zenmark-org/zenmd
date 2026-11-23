package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/zenmark-org/zenmd/engine/scriptplugins"
)

// The test binary is its own script helper (`<test binary> __script-host`).
func TestMain(m *testing.M) {
	if len(os.Args) > 1 && os.Args[1] == scriptplugins.HostArg {
		os.Exit(scriptplugins.HostMain(os.Stdin, os.Stdout, os.Stderr))
	}
	// Whatever a test reaches through cli.Run (it moves the old settings
	// folder), it must never see the real home.
	home, err := os.MkdirTemp("", "zenmd-home")
	if err != nil {
		panic(err)
	}
	os.Setenv("HOME", home)
	os.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	useFullBuild() // features: all, whatever apps/zenmark/app-assembly.yml lists today
	code := m.Run()
	scriptplugins.Shutdown()
	os.RemoveAll(home)
	os.Exit(code)
}
