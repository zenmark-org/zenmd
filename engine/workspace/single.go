package workspace

import (
	"fmt"
	"os"
	"path/filepath"
)

// OpenFile opens one markdown file on its own: a workspace rooted at the
// file's folder (the target's folder when the file is a symlink) whose only
// page is that file. The folder is not walked and its limits do not apply,
// so nothing else in it is scanned. Resolve, Read, Write and Asset keep their
// containment rules, rooted at the file's folder.
//
// It returns ErrNotMarkdown for any other kind of file, ErrTooLarge above
// 10 MB, and the operating system's error for a file that is not there.
func OpenFile(path string) (*Workspace, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	real, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return nil, err
	}
	fi, err := os.Stat(real)
	if err != nil {
		return nil, err
	}
	if fi.IsDir() {
		return nil, fmt.Errorf("workspace: %s is a directory", filepath.Base(path))
	}
	if !fi.Mode().IsRegular() {
		return nil, fmt.Errorf("workspace: %s is not a regular file", filepath.Base(path))
	}
	if !isMarkdown(real) {
		return nil, ErrNotMarkdown
	}
	if fi.Size() > maxFileSize {
		return nil, ErrTooLarge
	}
	w := &Workspace{root: filepath.Dir(real), single: true, file: filepath.Base(real), entries: 1}
	w.first = w.singleFiles()
	return w, nil
}

// Single reports whether the workspace is one file opened on its own.
func (w *Workspace) Single() bool { return w.single }

// File is the page's path of a one-file workspace ("" for a folder).
func (w *Workspace) File() string { return w.file }

// singleFiles is the one-entry listing of a one-file workspace.
func (w *Workspace) singleFiles() []string {
	w.mu.Lock()
	w.entries = 1
	w.mu.Unlock()
	return []string{w.file}
}
