package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/zenmark-org/zenmd/engine/config"
	"github.com/zenmark-org/zenmd/engine/session"
)

const treeUsage = `Usage: zenmd tree [folder] [--files] [--json]

Print the sidebar's trees of a folder (the current folder by default).

  zenmd tree            the Pages tree: titles, then each path in parentheses;
                        pages no link reaches are under "Other pages"
  zenmd tree --files    the Files listing: the folders, pages and images as they
                        are on disk, with (other) or (hidden) after a page the
                        Pages tree puts under Other pages or leaves out
  --json                print either tree as JSON
`

// runTree is `zenmd tree [folder] [--files] [--json]`: the Pages tree, or with --files the Files listing, of
// the folder, the same trees as the app's sidebar (Pages | Files). It has no config key: the view is a choice
// of the person looking, not of the folder.
func runTree(args []string, cwd string, stdout, stderr io.Writer) int {
	var files, asJSON bool
	var paths []string
	var perr error
	for _, arg := range args {
		switch {
		case arg == "-h" || arg == "--help":
			fmt.Fprint(stdout, treeUsage)
			return 0
		case arg == "--files":
			files = true
		case arg == "--json":
			asJSON = true
		case strings.HasPrefix(arg, "-") && arg != "-":
			perr = usageError("unknown option " + arg)
		default:
			paths = append(paths, arg)
		}
		if perr != nil {
			return fail(stderr, perr)
		}
	}
	if len(paths) > 1 {
		return fail(stderr, usageError("more than one folder given"))
	}
	dir := cwd
	if len(paths) == 1 {
		dir = paths[0]
		if !filepath.IsAbs(dir) {
			dir = filepath.Join(cwd, dir)
		}
	}
	dir = filepath.Clean(dir)
	switch fi, err := os.Stat(dir); {
	case err != nil:
		return fail(stderr, errors.New(paths0(paths)+": no such folder"))
	case !fi.IsDir():
		return fail(stderr, errors.New(paths0(paths)+": tree takes a folder, not a file"))
	}

	printNotices(stderr, loadConfig(dir))
	globalPath, _ := config.GlobalPath()
	s := session.New(session.Options{GlobalPath: globalPath, Version: Version})
	defer s.Close()
	if err := s.Open(dir); err != nil {
		return fail(stderr, err)
	}

	if files {
		ft, err := s.FileTree()
		if err != nil {
			return fail(stderr, err)
		}
		if asJSON {
			return printJSON(stdout, ft)
		}
		printFiles(stdout, ft.Nodes, 0)
		if ft.Omitted > 0 {
			fmt.Fprintf(stdout, "… %d more files not shown\n", ft.Omitted)
		}
		return 0
	}
	nodes := s.Workspace().Tree
	if asJSON {
		return printJSON(stdout, nodes)
	}
	printPages(stdout, nodes, 0)
	return 0
}

func printJSON(w io.Writer, v any) int {
	b, _ := json.MarshalIndent(v, "", "  ")
	fmt.Fprintln(w, string(b))
	return 0
}

// printPages prints the Pages tree, two spaces per level; a group (Other pages, a folder in it) is its title.
func printPages(w io.Writer, nodes []session.NavNode, depth int) {
	for _, n := range nodes {
		indent := strings.Repeat("  ", depth)
		if n.Kind == "page" {
			fmt.Fprintf(w, "%s%s (%s)\n", indent, n.Title, n.Path)
		} else {
			fmt.Fprintf(w, "%s%s\n", indent, n.Title)
		}
		printPages(w, n.Children, depth+1)
	}
}

// printFiles prints the Files listing, two spaces per level: folders with a trailing "/", a page's mark after
// it when the Pages tree puts it under Other pages or leaves it out.
func printFiles(w io.Writer, nodes []session.FileNode, depth int) {
	for _, n := range nodes {
		line := strings.Repeat("  ", depth) + n.Name
		switch {
		case n.Kind == "folder":
			line += "/"
		case n.Nav == "other" || n.Nav == "hidden":
			line += " (" + n.Nav + ")"
		}
		fmt.Fprintln(w, line)
		printFiles(w, n.Children, depth+1)
	}
}
