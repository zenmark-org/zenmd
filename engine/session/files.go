package session

import (
	"github.com/zenmark-org/zenmd/engine/workspace"
)

// FileNode is one row of the Files view: a folder, a markdown page or an image as it is on disk.
type FileNode struct {
	Name     string     `json:"name"`
	Title    string     `json:"title"`         // the page's title in the Pages tree, else Name
	Path     string     `json:"path"`          // folders end in "/"
	Kind     string     `json:"kind"`          // "folder" | "page" | "image"
	Nav      string     `json:"nav,omitempty"` // pages: "tree" | "other" | "hidden"
	Children []FileNode `json:"children,omitempty"`
}

// FileTree is the Files view: the folder's listing with the Pages tree's view of each page. Omitted counts the
// files left out when the folder has more than workspace.MaxFileTreeEntries.
type FileTree struct {
	Nodes   []FileNode `json:"nodes"`
	Omitted int        `json:"omitted"`
}

// FileTree lists the open folder as it is on disk (workspace.FileTree) and marks each page by where the Pages
// tree puts it: "tree" in the tree, "other" under Other pages, "hidden" in neither (nav.hidden, or under a
// hidden page). Titles come from the same tree. It reads the index the session already built and never
// renders or reads a page again; it keeps nothing, so every call walks the folder.
func (s *Session) FileTree() (FileTree, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ws == nil {
		return FileTree{}, errNoWorkspace
	}
	entries, omitted, err := s.ws.FileTree(s.life)
	if err != nil {
		return FileTree{}, s.cleanErr(".", err)
	}
	type info struct{ title, nav string }
	pages := map[string]info{}
	var collect func(ns []NavNode, other bool)
	collect = func(ns []NavNode, other bool) {
		for _, n := range ns {
			if n.Kind == "page" {
				nav := "tree"
				if other {
					nav = "other"
				}
				pages[n.Path] = info{n.Title, nav}
			}
			collect(n.Children, other)
		}
	}
	for _, n := range s.tree {
		collect([]NavNode{n}, n.Kind == "group" && n.Title == "Other pages")
	}
	var conv func([]workspace.FileEntry) []FileNode
	conv = func(es []workspace.FileEntry) []FileNode {
		out := make([]FileNode, 0, len(es))
		for _, e := range es {
			n := FileNode{Name: e.Name, Title: e.Name, Path: e.Path, Kind: e.Kind}
			if e.Kind == "page" {
				if p, ok := pages[e.Path]; ok {
					n.Title, n.Nav = p.title, p.nav
					if n.Title == "" {
						n.Title = e.Name
					}
				} else {
					n.Nav = "hidden"
				}
			}
			if len(e.Children) > 0 {
				n.Children = conv(e.Children)
			}
			out = append(out, n)
		}
		return out
	}
	return FileTree{Nodes: conv(entries), Omitted: omitted}, nil
}
