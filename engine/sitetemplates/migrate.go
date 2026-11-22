package sitetemplates

import (
	"os"
	"path/filepath"
)

// MovedNotice is the text under Problems when the old sites folder moved.
const MovedNotice = "templates/sites is now templates/blueprints"

// MigrateSites renames the old <dir>/sites to <dir>/blueprints, with its
// .zen-templates.yml, and returns the notice to show (once: the move is done
// only once). The folder is left alone, and the notice says why, when
//
//   - <dir>/blueprints already exists: sites is ignored;
//   - sites or blueprints is a link, or <dir> is;
//   - <dir> is outside the config folder, which is the only place zen moves things in.
//
// It returns "" when there is no sites folder.
func MigrateSites(dir, configDir string) string {
	sites, blueprints := filepath.Join(dir, "sites"), BlueprintsDir(dir)
	fi, err := os.Lstat(sites)
	if err != nil {
		return ""
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		return "templates/sites was not moved to templates/blueprints because it is a link; move it yourself"
	}
	if !fi.IsDir() {
		return ""
	}
	if _, err := os.Lstat(blueprints); err == nil {
		return "templates/sites is ignored because templates/blueprints exists; move what you need into it and delete sites"
	}
	if d, err := os.Lstat(dir); err != nil || d.Mode()&os.ModeSymlink != 0 || !within(real(dir), real(configDir)) {
		return "templates/sites was not moved to templates/blueprints because the templates folder is outside the settings folder; move it yourself"
	}
	if err := os.Rename(sites, blueprints); err != nil {
		return "templates/sites could not be moved to templates/blueprints: " + err.Error()
	}
	return MovedNotice
}
