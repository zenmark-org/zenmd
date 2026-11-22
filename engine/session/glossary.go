package session

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/zenmark-org/zenmd/builtins/pagetemplates"
	"github.com/zenmark-org/zenmd/engine/config"
	"github.com/zenmark-org/zenmd/engine/index"
	"github.com/zenmark-org/zenmd/engine/layouts"
	"github.com/zenmark-org/zenmd/engine/workspace"
)

// NewTermResult is what adding a term answers: the term's page, and whether it
// was there already (then nothing was written, so a caller can offer to open it).
type NewTermResult struct {
	Path    string `json:"path"`
	Existed bool   `json:"existed"`
}

// TermEntry is one term of a glossary folder, for `zenmd glossary list`.
type TermEntry struct {
	Title      string `json:"title"`
	Path       string `json:"path"`
	Definition string `json:"definition"`
}

// GlossaryFiles works on the glossary folders of a folder on disk, with no
// window: what `zenmd glossary` does, and what the app's New Term does through
// the Session. A term is a page whose layout is term; a glossary folder has a
// page with the glossary layout (index.md or README.md).
type GlossaryFiles struct {
	ws  *workspace.Workspace
	cfg *config.Config
}

// OpenGlossaryFiles opens the folder root. globalPath is the global config
// file ("" for none).
func OpenGlossaryFiles(globalPath, root string) (*GlossaryFiles, error) {
	ws, err := workspace.Open(root)
	if err != nil {
		return nil, err
	}
	return &GlossaryFiles{ws: ws, cfg: config.Load(globalPath, ws.Root())}, nil
}

func (g *GlossaryFiles) layoutOf(p string, fm map[string]any) string {
	return layouts.Resolve(p, fm, g.cfg.ForPage(fm).Layouts()).Name
}

func (g *GlossaryFiles) frontmatter(p string) (fm map[string]any, body []byte, err error) {
	b, err := g.ws.Read(p)
	if err != nil {
		return nil, nil, err
	}
	fm, body, _ = config.SplitFrontmatter(b)
	if fm == nil {
		fm = map[string]any{}
	}
	return fm, body, nil
}

// Folders lists the folders (workspace-relative, "" for the top level) that
// have a glossary index, sorted.
func (g *GlossaryFiles) Folders() ([]string, error) {
	files, err := g.ws.Files()
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	out := []string{}
	for _, f := range files {
		if b := strings.ToLower(path.Base(f)); b != "index.md" && b != "readme.md" {
			continue
		}
		fm, _, err := g.frontmatter(f)
		if err != nil || g.layoutOf(f, fm) != "glossary" {
			continue
		}
		d := path.Dir(f)
		if d == "." {
			d = ""
		}
		if !seen[d] {
			seen[d] = true
			out = append(out, d)
		}
	}
	sort.Strings(out)
	return out, nil
}

// List returns the terms of folder, A to Z by title: the H1 of the page, else
// the title: of its frontmatter, else its file name.
func (g *GlossaryFiles) List(folder string) ([]TermEntry, error) {
	dir, err := cleanFolder(folder)
	if err != nil {
		return nil, err
	}
	files, err := g.ws.Files()
	if err != nil {
		return nil, err
	}
	var out []TermEntry
	for _, f := range files {
		if d := path.Dir(f); d == "." && dir != "" || d != "." && d != dir {
			continue
		}
		fm, body, err := g.frontmatter(f)
		if err != nil || g.layoutOf(f, fm) != "term" {
			continue
		}
		title := headingOf(body)
		if title == "" {
			title, _ = fm["title"].(string)
		}
		if title = strings.TrimSpace(title); title == "" {
			title = strings.TrimSuffix(path.Base(f), path.Ext(f))
		}
		out = append(out, TermEntry{Title: title, Path: f, Definition: definition(body)})
	}
	sort.SliceStable(out, func(i, j int) bool {
		a, b := index.Fold(out[i].Title), index.Fold(out[j].Title)
		if a != b {
			return a < b
		}
		return out[i].Path < out[j].Path
	})
	return out, nil
}

// Add writes <folder>/<slug>.md for the term from the term page template: the
// term layout, an H1 with the term and the definition placeholder (or
// definition). Nothing is overwritten: when the file, or a term with the same
// title, is there already, the answer has Existed set and its path.
func (g *GlossaryFiles) Add(folder, term, definition string) (NewTermResult, error) {
	term = strings.Join(strings.Fields(term), " ")
	if term == "" {
		return NewTermResult{}, errors.New("give the term a name")
	}
	slug := layouts.TermSlug(term)
	if slug == "" {
		return NewTermResult{}, errors.New("the term needs a letter or a digit to name its file")
	}
	dir, err := cleanFolder(folder)
	if err != nil {
		return NewTermResult{}, err
	}
	rel, err := newPagePath(path.Join(dir, slug+".md"))
	if err != nil {
		return NewTermResult{}, err
	}
	p, err := g.ws.Resolve(rel)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return NewTermResult{}, err
	}
	if _, err := os.Lstat(p); err == nil {
		return NewTermResult{Path: rel, Existed: true}, nil
	}
	if terms, err := g.List(dir); err == nil {
		for _, t := range terms {
			if index.Fold(t.Title) == index.Fold(term) {
				return NewTermResult{Path: t.Path, Existed: true}, nil
			}
		}
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return NewTermResult{}, err
	}
	if err := g.ws.Write(rel, []byte(layouts.TermText(g.skeleton(), term, definition))); err != nil {
		return NewTermResult{}, err
	}
	return NewTermResult{Path: rel}, nil
}

// skeleton is the skeleton of the term page template: the folder's own
// (.zen/templates/term.yml), else the built-in one.
func (g *GlossaryFiles) skeleton() string {
	read := func(b []byte) string {
		var t struct {
			Skeleton string `yaml:"skeleton"`
		}
		if yaml.Unmarshal(b, &t) != nil {
			return ""
		}
		return t.Skeleton
	}
	if b, err := os.ReadFile(filepath.Join(g.ws.Root(), ".zen", "templates", "term.yml")); err == nil {
		if s := read(b); s != "" {
			return s
		}
	}
	b, _ := fs.ReadFile(pagetemplates.FS, "term.yml")
	return read(b)
}

// cleanFolder turns a folder argument into a workspace-relative folder ("" for
// the top level); it must be inside the folder.
func cleanFolder(folder string) (string, error) {
	f := strings.Trim(strings.ReplaceAll(strings.TrimSpace(folder), `\`, "/"), "/")
	if f == "" || f == "." {
		return "", nil
	}
	c := path.Clean(f)
	if c == ".." || strings.HasPrefix(c, "../") {
		return "", fmt.Errorf("%s: the folder must be inside the open folder", folder)
	}
	return c, nil
}

// glossaryFiles is the open folder as GlossaryFiles.
func (s *Session) glossaryFiles() (*GlossaryFiles, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ws == nil {
		return nil, errNoWorkspace
	}
	return &GlossaryFiles{ws: s.ws, cfg: s.cfg}, nil
}

// GlossaryFolders lists the folders of the open folder that have a glossary
// index ("" is the top level): the sidebar offers New Term… on their index.
func (s *Session) GlossaryFolders() (r0 []string, err error) {
	defer s.logFail("GlossaryFolders", &err)
	g, err := s.glossaryFiles()
	if err != nil {
		return nil, err
	}
	return g.Folders()
}

// NewTerm adds a term to the glossary folder: <folder>/<slug>.md, made from the
// term page template. An existing term is not touched; the answer says so.
func (s *Session) NewTerm(folder, term string) (r0 NewTermResult, err error) {
	defer s.logFail("NewTerm", &err)
	if err := s.writable(); err != nil {
		return NewTermResult{}, err
	}
	g, err := s.glossaryFiles()
	if err != nil {
		return NewTermResult{}, err
	}
	r, err := g.Add(folder, term, "")
	if err != nil {
		s.mu.Lock()
		defer s.mu.Unlock()
		return NewTermResult{}, s.cleanErr(path.Join(folder, "x"), err)
	}
	return r, nil
}
