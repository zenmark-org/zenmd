package cliset

import (
	"io"

	"github.com/zenmark-org/zenmd/engine/config"
	"github.com/zenmark-org/zenmd/engine/dialects"
	enginelayouts "github.com/zenmark-org/zenmd/engine/layouts"
	"github.com/zenmark-org/zenmd/engine/plugins"
	"github.com/zenmark-org/zenmd/engine/session"
	"github.com/zenmark-org/zenmd/internal/cli"
)

// BuildSite is `zenmd build` and `zenmd export site`: it writes the static
// site of b.Dir and prints the summary to stdout and the warnings to stderr. It
// opens no window and asks nothing; trust decisions are only read.
func BuildSite(b cli.Build, stdout, stderr io.Writer) error {
	return BuildSiteWith(Plugins(), Layouts(cli.Version), Dialects())(b, stdout, stderr)
}

// BuildSiteWith is BuildSite with the registries given: the app passes its own
// (its tests replace them), the command line passes the assembly's.
func BuildSiteWith(reg *plugins.Registry, lay *enginelayouts.Registry, dia *dialects.Registry) cli.Builder {
	return func(b cli.Build, stdout, stderr io.Writer) error {
		globalPath, _ := config.GlobalPath()
		return session.WriteSite(session.SiteEnv{GlobalPath: globalPath, Registry: reg, Layouts: lay, Dialects: dia, Version: cli.Version}, siteOptions(b), stdout, stderr)
	}
}

func siteOptions(b cli.Build) session.SiteOptions {
	return session.SiteOptions{Dir: b.Dir, Out: b.Out, Portable: b.Portable, Hosted: b.Hosted, Drafts: b.Drafts, NoDrafts: b.NoDrafts, Clean: b.Clean, NoPrint: b.NoPrint}
}
