// Package cliset is what the command-line zenmd and the Zenmark app share: the
// generated assembly (what the build is made of), the site export and the init
// hooks. It never imports Wails, kit/desktop or apps/zenmark, so cmd/zenmd
// builds without cgo (specs/packaging/linux-cli).
package cliset

import (
	"io/fs"

	"github.com/zenmark-org/zenmd/engine/dialects"
	enginelayouts "github.com/zenmark-org/zenmd/engine/layouts"
	"github.com/zenmark-org/zenmd/engine/plugins"
)

// Plugins is the registry of the built-in plugins the assembly includes.
func Plugins() *plugins.Registry { return assemblyPlugins() }

// Layouts is the registry of the built-in layouts the assembly includes.
func Layouts(version string) *enginelayouts.Registry { return assemblyLayouts(version) }

// Templates holds the built-in blueprints the assembly includes.
func Templates() fs.FS { return assemblyTemplates() }

// Dialects is the registry of the markdown dialects the assembly includes.
func Dialects() *dialects.Registry { return assemblyDialects() }

// Info is what the build is made of, as app-assembly.yml says.
var Info = assemblyInfo
