package sitetemplates

import (
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/zenmark-org/zenmd/internal/atomicfile"
	"gopkg.in/yaml.v3"
)

// Where a page template comes from.
const (
	SourceBuiltin = "builtin" // Zenmark's own: no copy on this Mac, or an installed copy in templates/pages (see Edited)
	SourceCustom  = "custom"  // a file in templates/pages that you added, not in the install record
	SourceFolder  = "folder"  // the open folder's own .zen/templates/<id>.yml
)

const maxPageTemplate = 256 * 1024

// PageField is one frontmatter field of a page template.
type PageField struct {
	Name, Type string
	Required   bool
	Default    string
}

// OutlineItem is one line of a template's structure: Depth 0 for the top level
// and one more inside a section's contains:.
type OutlineItem struct {
	Depth int
	Text  string
}

// PageTemplate is a page template as Settings and zenmd templates show it.
type PageTemplate struct {
	ID, Name, Description, Version, Layout string
	Source                                 string // SourceBuiltin, SourceCustom or SourceFolder
	Edited                                 bool   // an installed copy of a built-in one that you changed (or cannot be shown unchanged)
	Path                                   string // the file: the installed copy, "" for a built-in one with none
	Fields                                 []PageField
	Outline                                []OutlineItem // its structure:, empty without one
	Skeleton                               string
	Update                                 string // a newer built-in version than the installed copy ("" for none)
	Err                                    string // the file cannot be read: why
}

type pageDoc struct {
	ID          string    `yaml:"id"`
	Name        string    `yaml:"name"`
	Description string    `yaml:"description"`
	Version     string    `yaml:"version"`
	Layout      string    `yaml:"layout"`
	Skeleton    string    `yaml:"skeleton"`
	Fields      yaml.Node `yaml:"fields"`
	Structure   yaml.Node `yaml:"structure"`
}

// readPageFile reads a regular file of at most maxPageTemplate bytes; a link is not followed.
func readPageFile(p string) ([]byte, bool) {
	fi, err := os.Lstat(p)
	if err != nil || !fi.Mode().IsRegular() || fi.Size() > maxPageTemplate {
		return nil, false
	}
	b, err := os.ReadFile(p)
	return b, err == nil
}

func parsePage(id string, data []byte) PageTemplate {
	var d pageDoc
	if err := yaml.Unmarshal(data, &d); err != nil {
		return PageTemplate{ID: id, Name: id, Err: strings.TrimPrefix(err.Error(), "yaml: ")}
	}
	p := PageTemplate{ID: id, Name: d.Name, Description: d.Description, Version: d.Version, Layout: d.Layout, Skeleton: d.Skeleton}
	if p.Name == "" {
		p.Name = id
	}
	if d.Fields.Kind == yaml.MappingNode {
		for i := 0; i+1 < len(d.Fields.Content); i += 2 {
			var f struct {
				Type     string `yaml:"type"`
				Required bool   `yaml:"required"`
				Default  string `yaml:"default"`
			}
			_ = d.Fields.Content[i+1].Decode(&f)
			if f.Type == "" {
				f.Type = "string"
			}
			p.Fields = append(p.Fields, PageField{d.Fields.Content[i].Value, f.Type, f.Required, f.Default})
		}
	}
	p.Outline = outline(&d.Structure, 0, nil)
	return p
}

// outline lists the elements of a structure: sequence, one line each. An
// element is a mapping whose first key names it; its value (a mapping of
// options, or a plain value) and any further keys are its options.
func outline(n *yaml.Node, depth int, out []OutlineItem) []OutlineItem {
	if n.Kind != yaml.SequenceNode {
		return out
	}
	for _, item := range n.Content {
		if item.Kind != yaml.MappingNode || len(item.Content) < 2 {
			continue
		}
		name := item.Content[0].Value
		var parts []string
		var contains *yaml.Node
		addOpts := func(m *yaml.Node, label bool) {
			for i := 0; i+1 < len(m.Content); i += 2 {
				k, v := m.Content[i].Value, m.Content[i+1]
				switch {
				case k == "contains" && v.Kind == yaml.SequenceNode:
					contains = v
				case label && name == "section" && k == "heading":
					parts = append([]string{scalarText(v)}, parts...)
				case v.Kind == yaml.ScalarNode && v.Value == "true":
					parts = append(parts, k)
				default:
					parts = append(parts, k+" "+scalarText(v))
				}
			}
		}
		switch first := item.Content[1]; first.Kind {
		case yaml.MappingNode:
			addOpts(first, true)
		case yaml.ScalarNode:
			if first.Value != "" && first.Tag != "!!null" {
				parts = append(parts, first.Value)
			}
		}
		if len(item.Content) > 2 { // { list: ordered, min-items: 2 }: the rest are options
			rest := &yaml.Node{Kind: yaml.MappingNode, Content: item.Content[2:]}
			addOpts(rest, false)
		}
		text := name
		if len(parts) > 0 {
			text += ": " + strings.Join(parts, ", ")
		}
		out = append(out, OutlineItem{depth, text})
		if contains != nil {
			out = outline(contains, depth+1, out)
		}
	}
	return out
}

func scalarText(n *yaml.Node) string {
	switch n.Kind {
	case yaml.SequenceNode:
		var xs []string
		for _, c := range n.Content {
			xs = append(xs, scalarText(c))
		}
		return strings.Join(xs, " / ")
	case yaml.MappingNode:
		return "…"
	}
	return n.Value
}

// pageFiles lists the *.yml files of a folder (ids, sorted), skipping hidden
// files, update copies (<id>.yml.new) and anything that is not a regular file.
func pageFiles(folder string) []string {
	entries, err := os.ReadDir(folder)
	if err != nil {
		return nil
	}
	var ids []string
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, ".") || !strings.HasSuffix(name, ".yml") || !e.Type().IsRegular() {
			continue
		}
		ids = append(ids, strings.TrimSuffix(name, ".yml"))
	}
	return ids
}

func builtinPageIDs(builtins fs.FS) []string {
	if builtins == nil {
		return nil
	}
	entries, err := fs.ReadDir(builtins, ".")
	if err != nil {
		return nil
	}
	var ids []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".yml") {
			ids = append(ids, strings.TrimSuffix(e.Name(), ".yml"))
		}
	}
	return ids
}

func builtinPage(builtins fs.FS, id string) (PageTemplate, []byte, bool) {
	data, err := fs.ReadFile(builtins, id+".yml")
	if err != nil {
		return PageTemplate{}, nil, false
	}
	p := parsePage(id, data)
	p.Source = SourceBuiltin
	return p, data, p.Err == ""
}

// InstallPages copies the built-in page templates that were never installed
// into <dir>/pages, as editable files, and records them with their version and
// fingerprint. A template that is recorded is never installed again (deleting
// one is final), and a file you put there first is yours and left alone. An
// error means the folder cannot be written; the built-in ones still work.
func InstallPages(dir string, builtins fs.FS) error {
	pages := PagesDir(dir)
	if err := os.MkdirAll(pages, 0o755); err != nil {
		return err
	}
	rec := readRecord(pages)
	changed := false
	for _, id := range builtinPageIDs(builtins) {
		p, data, ok := builtinPage(builtins, id)
		if !ok {
			continue
		}
		dest := filepath.Join(pages, id+".yml")
		if have, done := rec.Installed[id]; done {
			// A copy nobody edited follows the built-in; an edited one, one
			// without a fingerprint and a deleted one are left alone.
			if want, ok := rec.Fingerprints[id]; ok && newer(p.Version, have) {
				if cur, ok := readPageFile(dest); ok && fingerprintBytes(cur) == want {
					if atomicfile.Write(dest, data) == nil {
						rec.Installed[id], rec.Fingerprints[id] = p.Version, fingerprintBytes(data)
						slog.Info("page template "+id+" updated to "+p.Version+" (your copy was not edited)", "source", "app")
						changed = true
					}
				}
			}
			continue
		}
		if _, err := os.Lstat(dest); err == nil {
			continue // the user's own
		}
		if err := atomicfile.Write(dest, data); err != nil {
			return fmt.Errorf("installing %s: %w", id, err)
		}
		rec.Installed[id], rec.Fingerprints[id] = p.Version, fingerprintBytes(data)
		changed = true
	}
	if changed || !fileExists(filepath.Join(pages, RecordFile)) {
		return writeRecord(pages, rec)
	}
	return nil
}

// ListPages lists the page templates a folder can use: its own
// .zen/templates (folder is its root, "" for none), then those in
// <dir>/pages, then the built-in ones, each id once, where the first source
// wins. They are sorted in that order and then by name.
func ListPages(dir string, builtins fs.FS, folder string) ([]PageTemplate, []string) {
	by := map[string]PageTemplate{}
	for _, id := range builtinPageIDs(builtins) {
		if p, _, ok := builtinPage(builtins, id); ok {
			by[id] = p
		}
	}
	pages := PagesDir(dir)
	if dir == "" {
		pages = "" // no templates folder: only the built-in ones and the open folder's own
	}
	rec := readRecord(pages)
	for _, id := range pageFiles(pages) {
		file := filepath.Join(pages, id+".yml")
		data, ok := readPageFile(file)
		if !ok {
			continue
		}
		p := parsePage(id, data)
		p.Source, p.Path = SourceCustom, file
		if have, ok := rec.Installed[id]; ok {
			if b, _, ok := builtinPage(builtins, id); ok {
				p.Source = SourceBuiltin
				// Unedited: the fingerprint of the file is the one recorded at install.
				want, has := rec.Fingerprints[id]
				p.Edited = !has || fingerprintBytes(data) != want
				if newer(b.Version, have) {
					p.Update = b.Version
				}
			}
		}
		by[id] = p
	}
	if folder != "" {
		own := filepath.Join(folder, ".zen", "templates")
		for _, id := range pageFiles(own) {
			file := filepath.Join(own, id+".yml")
			if data, ok := readPageFile(file); ok {
				p := parsePage(id, data)
				p.Source, p.Path = SourceFolder, file
				by[id] = p
			}
		}
	}
	rank := func(p PageTemplate) int {
		switch {
		case p.Source == SourceFolder:
			return 0
		case p.Path != "": // a file in templates/pages: an installed copy or your own
			return 1
		}
		return 2 // built-in with no copy
	}
	out := make([]PageTemplate, 0, len(by))
	for _, p := range by {
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if ra, rb := rank(a), rank(b); ra != rb {
			return ra < rb
		}
		if a.Name != b.Name {
			return a.Name < b.Name
		}
		return a.ID < b.ID
	})
	return out, nil
}

// UpdatePage brings the built-in page template id to its newer version. A copy
// that still matches the fingerprint of its install is replaced in place. An
// edited copy (or one whose install recorded no fingerprint) is left as it is
// and the new version is written beside it as <id>.yml.new.
func UpdatePage(dir string, builtins fs.FS, id string) (UpdateResult, error) {
	pages := PagesDir(dir)
	rec := readRecord(pages)
	have, ok := rec.Installed[id]
	if id == "" || strings.ContainsAny(id, `/\`) || !ok {
		return UpdateResult{}, fmt.Errorf("%q is not a built-in page template that was installed here", id)
	}
	b, data, ok := builtinPage(builtins, id)
	if !ok {
		return UpdateResult{}, fmt.Errorf("this zen has no built-in page template %q", id)
	}
	if !newer(b.Version, have) {
		return UpdateResult{}, fmt.Errorf("%s is up to date (version %s)", id, have)
	}
	dest := filepath.Join(pages, id+".yml")
	if want, ok := rec.Fingerprints[id]; ok {
		if cur, ok := readPageFile(dest); ok && fingerprintBytes(cur) == want {
			if err := atomicfile.Write(dest, data); err != nil {
				return UpdateResult{}, fmt.Errorf("updating %s: %w", id, err)
			}
			rec.Installed[id], rec.Fingerprints[id] = b.Version, fingerprintBytes(data)
			return UpdateResult{Path: dest, InPlace: true}, writeRecord(pages, rec)
		}
	}
	newPath := dest + ".new"
	if fileExists(newPath) {
		return UpdateResult{}, fmt.Errorf("%s already exists; compare it with your copy and delete it first", filepath.Base(newPath))
	}
	if err := atomicfile.Write(newPath, data); err != nil {
		return UpdateResult{}, err
	}
	rec.Installed[id] = b.Version
	if err := writeRecord(pages, rec); err != nil {
		return UpdateResult{Path: newPath}, err
	}
	return UpdateResult{Path: newPath}, nil
}

var pageIDRE = regexp.MustCompile(`^[a-z][a-z0-9-]{0,63}$`)

// FindPage reads the file of the page template id, for checking a page against it: the open
// folder's own .zen/templates/<id>.yml (folder is its root, "" for none), then <dir>/pages/<id>.yml
// (dir "" for none), then the built-in one. source is SourceFolder, SourceCustom or SourceBuiltin.
// It reads three files at most, installs nothing and never follows a link. An id that is not a
// plain template id (it may come from a page's frontmatter) finds nothing.
func FindPage(dir string, builtins fs.FS, folder, id string) (data []byte, source string, ok bool) {
	if !pageIDRE.MatchString(id) {
		return nil, "", false
	}
	if folder != "" {
		if b, ok := readPageFile(filepath.Join(folder, ".zen", "templates", id+".yml")); ok {
			return b, SourceFolder, true
		}
	}
	if dir != "" {
		if b, ok := readPageFile(filepath.Join(PagesDir(dir), id+".yml")); ok {
			return b, SourceCustom, true
		}
	}
	if builtins != nil {
		if b, err := fs.ReadFile(builtins, id+".yml"); err == nil {
			return b, SourceBuiltin, true
		}
	}
	return nil, "", false
}
