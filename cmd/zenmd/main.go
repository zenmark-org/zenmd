// Command zenmd is the command-line-only zenmd: the engine's commands (export
// site, init, templates, ...) without the Zenmark app. It imports no Wails and
// needs no cgo, so it builds for Linux and macOS alike
// (specs/packaging/linux-cli). The app's zenmd is the root main.go.
package main

import (
	"fmt"
	"os"

	"github.com/zenmark-org/zenmd/apps/zenmark/cliset"
	"github.com/zenmark-org/zenmd/engine/scriptplugins"
	"github.com/zenmark-org/zenmd/internal/cli"
)

func main() {
	// The hidden script-helper mode: before the command line is parsed.
	if len(os.Args) > 1 && os.Args[1] == scriptplugins.HostArg {
		os.Exit(scriptplugins.HostMain(os.Stdin, os.Stdout, os.Stderr))
	}
	cli.BuildKind = "command line"
	// The alias notice: only for the old name, on stderr, after the helper
	// mode above (which must print nothing).
	cli.Notice(os.Stderr, cli.ProgramName(os.Args[0]))
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "zenmd: %v\n", err)
		os.Exit(1)
	}
	os.Exit(cli.Run(os.Args[1:], cwd, os.Stdout, os.Stderr, cliset.NoApp, cliset.BuildSite, cliset.InitSite, cliset.NoPDF))
}
