package dropdown

import "github.com/zenmark-org/zenmd/engine/plugins"

type plugin struct{}

// New returns the dropdown plugin.
func New() plugins.Plugin { return plugin{} }

func (plugin) Manifest() plugins.Manifest {
	return plugins.Manifest{ID: "dropdown", Version: "0.1.0", APIVersion: plugins.APIVersion, Default: true}
}
