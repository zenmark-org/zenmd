package workspace

import (
	"context"
	"io/fs"
	"os"
	"path"
	"sort"
	"strings"
)

// MaxFileTreeEntries is the most entries (folders, pages and images) FileTree lists. A variable so tests can
// lower it.
var MaxFileTreeEntries = 5000

// FileEntry is one folder, markdown page or image of the Files listing. Folder paths end in "/", so a folder
// never clashes with a page of the same name (guides/ and guides.md).
type FileEntry struct {
	Name     string      `json:"name"` // "intro.md", "guides"
	Path     string      `json:"path"` // "guides/intro.md"; folders end in "/"
	Kind     string      `json:"kind"` // "folder" | "page" | "image"
	Children []FileEntry `json:"children,omitempty"`
}

// FileTree lists the folder as it is on disk: its folders, markdown pages and images, folders first, names in
// natural case-insensitive order. It skips what the page scan skips (skipDir), counts a symlink only when it
// resolves to a regular file inside the root, and leaves out folders that hold nothing listed. At most
// MaxFileTreeEntries entries come back; omitted counts the files left out beyond that (images go first, then
// the last pages, and the folders that held them stay). A one-file workspace lists its one file.
func (w *Workspace) FileTree(ctx context.Context) (entries []FileEntry, omitted int, err error) {
	if w.single {
		return []FileEntry{{Name: w.file, Path: w.file, Kind: "page"}}, 0, nil
	}
	if err := ctx.Err(); err != nil {
		return nil, 0, err
	}
	maxEntries := MaxEntries
	if w.maxEntries > 0 {
		maxEntries = w.maxEntries
	}
	visited := 0
	var walk func(dir string, atRoot bool) ([]FileEntry, error)
	walk = func(dir string, atRoot bool) ([]FileEntry, error) {
		abs, err := w.Resolve(dir)
		if err != nil {
			return nil, err
		}
		list, err := os.ReadDir(abs)
		if err != nil {
			if atRoot {
				return nil, err
			}
			return nil, nil // unreadable below the root: leave it out
		}
		var folders, files []FileEntry
		for _, d := range list {
			if visited++; visited > maxEntries {
				return nil, &TooManyFilesError{Limit: maxEntries}
			}
			if visited&255 == 0 {
				if err := ctx.Err(); err != nil {
					return nil, err
				}
			}
			name := d.Name()
			rel := path.Join(dir, name)
			if rel == "." {
				rel = name
			}
			switch {
			case d.IsDir():
				if skipDir(name, atRoot) {
					continue
				}
				kids, err := walk(rel, false)
				if err != nil {
					return nil, err
				}
				if len(kids) > 0 {
					folders = append(folders, FileEntry{Name: name, Path: rel + "/", Kind: "folder", Children: kids})
				}
			case isMarkdown(name) || isAsset(name):
				ok := d.Type().IsRegular()
				if d.Type()&fs.ModeSymlink != 0 {
					ok = w.isFile(rel)
				}
				if !ok {
					continue
				}
				kind := "page"
				if !isMarkdown(name) {
					kind = "image"
				}
				files = append(files, FileEntry{Name: name, Path: rel, Kind: kind})
			}
		}
		sortEntries(folders)
		sortEntries(files)
		return append(folders, files...), nil
	}
	if entries, err = walk(".", true); err != nil {
		return nil, 0, err
	}
	if entries == nil {
		entries = []FileEntry{}
	}
	return capEntries(entries, MaxFileTreeEntries)
}

// capEntries leaves at most limit entries: when there are more, the last images are dropped first, then the
// last pages. Folders are kept.
func capEntries(entries []FileEntry, limit int) ([]FileEntry, int, error) {
	var images, pages []string
	total := 0
	var count func([]FileEntry)
	count = func(l []FileEntry) {
		for _, e := range l {
			total++
			switch e.Kind {
			case "image":
				images = append(images, e.Path)
			case "page":
				pages = append(pages, e.Path)
			}
			count(e.Children)
		}
	}
	count(entries)
	need := total - limit
	if need <= 0 {
		return entries, 0, nil
	}
	if need > len(images)+len(pages) {
		need = len(images) + len(pages)
	}
	drop := map[string]bool{}
	for i := len(images) - 1; i >= 0 && len(drop) < need; i-- {
		drop[images[i]] = true
	}
	for i := len(pages) - 1; i >= 0 && len(drop) < need; i-- {
		drop[pages[i]] = true
	}
	var prune func([]FileEntry) []FileEntry
	prune = func(l []FileEntry) []FileEntry {
		out := make([]FileEntry, 0, len(l))
		for _, e := range l {
			if drop[e.Path] {
				continue
			}
			if e.Kind == "folder" {
				e.Children = prune(e.Children)
			}
			out = append(out, e)
		}
		return out
	}
	return prune(entries), len(drop), nil
}

func sortEntries(l []FileEntry) {
	sort.SliceStable(l, func(i, j int) bool { return naturalLess(l[i].Name, l[j].Name) })
}

// naturalLess orders names case-insensitively with runs of digits compared as numbers ("2-intro" before
// "10-setup"); ties fall back to the plain comparison so the order is stable.
func naturalLess(a, b string) bool {
	la, lb := strings.ToLower(a), strings.ToLower(b)
	i, j := 0, 0
	for i < len(la) && j < len(lb) {
		if isDigit(la[i]) && isDigit(lb[j]) {
			si := i
			for i < len(la) && isDigit(la[i]) {
				i++
			}
			sj := j
			for j < len(lb) && isDigit(lb[j]) {
				j++
			}
			na, nb := strings.TrimLeft(la[si:i], "0"), strings.TrimLeft(lb[sj:j], "0")
			if len(na) != len(nb) {
				return len(na) < len(nb)
			}
			if na != nb {
				return na < nb
			}
			continue
		}
		if la[i] != lb[j] {
			return la[i] < lb[j]
		}
		i++
		j++
	}
	if len(la)-i != len(lb)-j {
		return len(la)-i < len(lb)-j
	}
	return a < b
}

func isDigit(c byte) bool { return c >= '0' && c <= '9' }
