package text

import (
	"io/fs"

	"github.com/zenmark-org/zenmd/engine/plugins"
)

func fsRead(ap plugins.AssetProvider, name string) (string, error) {
	b, err := fs.ReadFile(ap.Assets(), name)
	return string(b), err
}
