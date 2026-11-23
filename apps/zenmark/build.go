package app

import (
	"io"

	"github.com/zenmark-org/zenmd/apps/zenmark/cliset"
	"github.com/zenmark-org/zenmd/internal/cli"
)

// BuildSite is `zenmd build`: it writes the static site of b.Dir and prints the
// summary to stdout and the warnings to stderr. It opens no window and asks
// nothing; trust decisions are only read. The work is cliset's, shared with the
// command-line zenmd, with this app's registries.
func BuildSite(b cli.Build, stdout, stderr io.Writer) error {
	return cliset.BuildSiteWith(builtinPlugins(), builtinLayouts(cli.Version), builtinDialects())(b, stdout, stderr)
}
