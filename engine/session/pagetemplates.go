package session

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/zenmark-org/zenmd/engine/config"
	"github.com/zenmark-org/zenmd/engine/sitetemplates"
)

// PageTemplateField is one field of a page template, for the Templates view.
type PageTemplateField struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Required bool   `json:"required"`
	Default  string `json:"default"`
}

// PageTemplateLine is one line of a page template's structure outline.
type PageTemplateLine struct {
	Depth int    `json:"depth"`
	Text  string `json:"text"`
}

// PageTemplateInfo is a page template as Settings › Templates shows it.
type PageTemplateInfo struct {
	ID          string              `json:"id"`
	Name        string              `json:"name"`
	Description string              `json:"description"`
	Version     string              `json:"version"`
	Layout      string              `json:"layout"`
	Source      string              `json:"source"` // "builtin" (also an installed copy), "custom" (your file in templates/pages) or "folder" (.zen/templates)
	Edited      bool                `json:"edited"` // an installed copy of a built-in one that you changed
	Path        string              `json:"path"`   // the file, "" for a built-in one
	Fields      []PageTemplateField `json:"fields"`
	Outline     []PageTemplateLine  `json:"outline"`
	Skeleton    string              `json:"skeleton"`
	Update      string              `json:"update"` // a newer built-in version, "" for none
	Error       string              `json:"error"`
	Rules       []string            `json:"rules"` // the open folder's path rules that use it
}

// PageListing is the page templates for the open folder (or none), with what went wrong.
type PageListing struct {
	Templates []PageTemplateInfo
	Dir       string   // the templates folder, "" when there is none
	Warnings  []string // problems with the templates: key
	Note      string   // set when the folder cannot be written to, so the built-in ones are used as they are
}

// Pages lists the page templates: the open folder's own, those in
// templates/pages, and the built-in ones. It installs the built-in ones into
// templates/pages first (once).
func (t TemplatesAPI) Pages() PageListing {
	root, rules := t.context()
	return t.pagesIn(root, rules)
}

// PagesIn is Pages for the folder root, which need not be open: its own .zen/templates and its
// .zenmd.yml path rules (zenmd templates, run in a folder).
func (t TemplatesAPI) PagesIn(root string) PageListing {
	var rules []config.TemplateRule
	if root != "" {
		rules = config.Load(t.env.globalPath, root).TemplateRules()
	}
	return t.pagesIn(root, rules)
}

func (t TemplatesAPI) pagesIn(root string, rules []config.TemplateRule) PageListing {
	dir, warns := t.env.dir()
	out := PageListing{Dir: dir, Warnings: warns}
	if dir != "" {
		if err := sitetemplates.InstallPages(dir, t.env.pages); err != nil {
			out.Note = fmt.Sprintf("The templates folder (%s) cannot be written to, so the built-in page templates are used as they are.", dir)
		}
	}
	ruleList := func(id string) []string {
		var out []string
		for _, r := range rules {
			if r.Name == id {
				out = append(out, r.Pattern)
			}
		}
		return out
	}
	list, _ := sitetemplates.ListPages(dir, t.env.pages, root)
	out.Templates = make([]PageTemplateInfo, 0, len(list))
	for _, p := range list {
		info := PageTemplateInfo{
			ID: p.ID, Name: p.Name, Description: p.Description, Version: p.Version, Layout: p.Layout,
			Source: p.Source, Edited: p.Edited, Path: p.Path, Skeleton: p.Skeleton, Update: p.Update, Error: p.Err,
			Fields: []PageTemplateField{}, Outline: []PageTemplateLine{}, Rules: []string{},
		}
		for _, f := range p.Fields {
			info.Fields = append(info.Fields, PageTemplateField{f.Name, f.Type, f.Required, f.Default})
		}
		for _, o := range p.Outline {
			info.Outline = append(info.Outline, PageTemplateLine{o.Depth, o.Text})
		}
		if r := ruleList(p.ID); r != nil {
			info.Rules = r
		}
		out.Templates = append(out.Templates, info)
	}
	return out
}

func (t TemplatesAPI) context() (string, []config.TemplateRule) {
	if t.s == nil {
		return "", nil
	}
	return t.s.templateContext()
}

// PagesDir is the folder of the page templates (templates/pages), "" when
// there is none. It installs the built-in ones first.
func (t TemplatesAPI) PagesDir() string {
	dir, _ := t.env.dir()
	if dir == "" {
		return ""
	}
	_ = sitetemplates.InstallPages(dir, t.env.pages)
	return sitetemplates.PagesDir(dir)
}

// UpdatePage brings the newer built-in page template id to the user's copy: in
// place when the copy was never edited, else beside it as <id>.yml.new.
func (t TemplatesAPI) UpdatePage(id string) (sitetemplates.UpdateResult, error) {
	dir, _ := t.env.dir()
	if dir == "" {
		return sitetemplates.UpdateResult{}, errors.New("there is no templates folder")
	}
	_ = sitetemplates.InstallPages(dir, t.env.pages) // so there is a record
	return sitetemplates.UpdatePage(dir, t.env.pages, id)
}

var pageIDRE = regexp.MustCompile(`^[a-z][a-z0-9-]{0,63}$`)

// CopyPage writes the page template id to the open folder's .zen/templates/<id>.yml,
// to edit there, and returns that path inside the folder. An existing file is never replaced.
func (t TemplatesAPI) CopyPage(id string) (string, error) {
	if t.s == nil {
		return "", errors.New("there is no open folder")
	}
	if err := t.s.writable(); err != nil {
		return "", err
	}
	if !pageIDRE.MatchString(id) {
		return "", fmt.Errorf("%q is not a page template id", id)
	}
	if root, _ := t.context(); root == "" {
		return "", errNoWorkspace
	}
	var from *PageTemplateInfo
	list := t.Pages().Templates
	if i := slices.IndexFunc(list, func(p PageTemplateInfo) bool { return p.ID == id }); i >= 0 {
		from = &list[i]
	}
	if from == nil {
		return "", fmt.Errorf("there is no page template %q", id)
	}
	rel := ".zen/templates/" + id + ".yml"
	if from.Source == sitetemplates.SourceFolder {
		return "", fmt.Errorf("this folder already has %s", rel)
	}
	var data []byte
	if from.Path != "" {
		b, err := os.ReadFile(from.Path)
		if err != nil {
			return "", fmt.Errorf("could not read %s: %w", id, err)
		}
		data = b
	} else {
		b, err := fs.ReadFile(t.env.pages, id+".yml")
		if err != nil {
			return "", fmt.Errorf("there is no built-in page template %q", id)
		}
		data = b
	}
	if err := t.s.writeInFolder(rel, data); err != nil {
		return "", err
	}
	return rel, nil
}

// TemplateUpdate is what an update did: Path is the file or folder that has the new version, and
// InPlace tells that it replaced the user's untouched copy (else it is beside an edited one, as .new).
type TemplateUpdate struct {
	Path    string `json:"path"`
	InPlace bool   `json:"inPlace"`
}

// PagePath is the file of the page template id for opening it in an editor: the copy in the
// templates folder or the open folder's own. It fails for one that is only built in.
func (t TemplatesAPI) PagePath(id string) (string, error) {
	for _, p := range t.Pages().Templates {
		if p.ID == id {
			if p.Path == "" {
				return "", fmt.Errorf("%s is built in and has no file to open; copy it to this folder first", id)
			}
			return p.Path, nil
		}
	}
	return "", fmt.Errorf("there is no page template %q", id)
}

// writeInFolder writes a new file at the slash path rel inside the open folder, for the hidden
// .zen/templates (the folder's page templates). It creates folders, never follows a link out of the
// folder and never replaces a file.
func (s *Session) writeInFolder(rel string, data []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ws == nil {
		return errNoWorkspace
	}
	root, err := filepath.EvalSymlinks(s.ws.Root())
	if err != nil {
		return err
	}
	cur := root
	parts := strings.Split(rel, "/")
	for _, part := range parts[:len(parts)-1] {
		cur = filepath.Join(cur, part)
		switch fi, err := os.Lstat(cur); {
		case err == nil && !fi.IsDir(): // a file or a link where a folder goes
			return fmt.Errorf("%s is not a folder; cannot write %s", filepath.Base(cur), rel)
		case errors.Is(err, fs.ErrNotExist):
			if err := os.Mkdir(cur, 0o755); err != nil {
				return err
			}
		case err != nil:
			return err
		}
	}
	f, err := os.OpenFile(filepath.Join(cur, parts[len(parts)-1]), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if errors.Is(err, fs.ErrExist) {
		return fmt.Errorf("this folder already has %s", rel)
	}
	if err != nil {
		return err
	}
	if _, err := f.Write(data); err != nil {
		f.Close()
		os.Remove(f.Name())
		return err
	}
	return f.Close()
}
