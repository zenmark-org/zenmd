package app

import "github.com/zenmark-org/zenmd/engine/session"

// FileTree is the Files view of the sidebar: the open folder's folders, pages and images as they are on disk,
// with each page marked by where the Pages tree puts it.
func (z *Zen) FileTree() (r0 session.FileTree, err error) {
	defer z.logCall("FileTree", &err)
	return z.s.FileTree()
}
