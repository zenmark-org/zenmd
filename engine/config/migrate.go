package config

import (
	"os"
	"path/filepath"
)

// oldFolderName is the settings folder before 0.8.0.
const oldFolderName = "zen"

// Migrate moves the old settings folder <UserConfigDir>/zen to
// <UserConfigDir>/zenmark, once. It moves only when zenmark does not exist and
// zen is a real directory (not a link) that holds our files: config.yml or
// trusted-plugins.yml directly inside. Anything else, such as the Zen
// browser's own folder, is left alone. The rename stays in one parent, so it
// is atomic and nothing is lost.
func Migrate() (moved bool, err error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return false, nil // no config dir: nothing to move
	}
	oldDir := filepath.Join(base, oldFolderName)
	newDir := filepath.Join(base, FolderName)
	if _, err := os.Lstat(newDir); err == nil {
		return false, nil
	} else if !os.IsNotExist(err) {
		return false, err
	}
	fi, err := os.Lstat(oldDir)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	if !fi.IsDir() { // a file or a symlink (Lstat does not follow it)
		return false, nil
	}
	ours := false
	for _, f := range []string{"config.yml", "trusted-plugins.yml"} {
		if st, err := os.Lstat(filepath.Join(oldDir, f)); err == nil && st.Mode().IsRegular() {
			ours = true
			break
		}
	}
	if !ours {
		return false, nil
	}
	if err := os.Rename(oldDir, newDir); err != nil {
		return false, err
	}
	return true, nil
}
