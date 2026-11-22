package session

import (
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"path/filepath"
	"sort"
	"strings"

	"github.com/zenmark-org/zenmd/engine/config"
	"github.com/zenmark-org/zenmd/engine/sitetemplates"
)

// SiteTemplate is a site template as the frontend lists it (core/init).
type SiteTemplate struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Version     string `json:"version"`
	Ask         []struct {
		Key     string `json:"key"`
		Label   string `json:"label"`
		Default string `json:"default"`
	} `json:"ask"`
	Builtin bool   `json:"builtin"`
	Update  string `json:"update"`
	Error   string `json:"error"`
}

// siteEnv is what the templates need from outside: the global config file
// (its folder holds the templates folder), the built-in blueprints and page
// templates, and where to report what happened to the folder (note, may be nil).
type siteEnv struct {
	globalPath string
	builtins   fs.FS
	pages      fs.FS
	note       func(string)
}

// TemplateListing is the result of listing the templates.
type TemplateListing struct {
	Templates []sitetemplates.Template
	Dir       string   // the templates folder, "" when there is none
	Warnings  []string // problems with the templates and the templates: key
	Note      string   // set when the built-in presets are used straight from zen
}

// dir is the templates folder: <config folder>/templates, or the global
// config's templates: dir. "" without a settings folder. The first use moves
// an old templates/sites to templates/blueprints (sitetemplates.MigrateSites)
// and reports it.
func (e siteEnv) dir() (string, []string) {
	if e.globalPath == "" {
		return "", nil
	}
	cfgDir := filepath.Dir(e.globalPath)
	dir, warns := config.Load(e.globalPath, "").TemplatesDir(cfgDir)
	if n := sitetemplates.MigrateSites(dir, cfgDir); n != "" && e.note != nil {
		e.note(n)
	}
	return dir, warns
}

// list installs the presets that are not there yet, and lists the templates.
// When the folder cannot be written to, the presets that are not in it are
// listed from zen itself, with a note.
func (e siteEnv) list() TemplateListing {
	l := e.list0()
	for i := range l.Templates {
		l.Templates[i].PageSource = e.pages
		if l.Dir != "" {
			l.Templates[i].PagesDir = sitetemplates.PagesDir(l.Dir)
		}
	}
	return l
}

func (e siteEnv) list0() TemplateListing {
	dir, warns := e.dir()
	out := TemplateListing{Dir: dir, Warnings: warns}
	if dir == "" {
		out.Templates, _ = sitetemplates.Builtins(e.builtins)
		out.Note = "There is no settings folder, so the built-in blueprints are used as they are."
		return out
	}
	err := sitetemplates.Install(dir, e.builtins)
	_ = sitetemplates.InstallPages(dir, e.pages) // the page templates work from zen itself when this fails
	ts, lw := sitetemplates.ListWith(dir, e.builtins)
	out.Templates, out.Warnings = ts, append(out.Warnings, lw...)
	if err != nil {
		have := map[string]bool{}
		for _, t := range ts {
			have[t.ID] = true
		}
		bundle, _ := sitetemplates.Builtins(e.builtins)
		for _, t := range bundle {
			if !have[t.ID] && t.Err == "" {
				out.Templates = append(out.Templates, t)
			}
		}
		sort.SliceStable(out.Templates, func(i, j int) bool {
			a, b := out.Templates[i], out.Templates[j]
			if (a.Err == "") != (b.Err == "") {
				return a.Err == ""
			}
			if a.Order != b.Order {
				return a.Order < b.Order
			}
			return a.Name < b.Name
		})
		out.Note = fmt.Sprintf("The blueprints folder (%s) cannot be written to, so the built-in blueprints are used as they are.", dir)
	}
	return out
}

func (e siteEnv) find(id string) (sitetemplates.Template, TemplateListing, bool) {
	l := e.list()
	for _, t := range l.Templates {
		if t.ID == id {
			return t, l, true
		}
	}
	return sitetemplates.Template{}, l, false
}

// TemplatesAPI is the session's templates: list the blueprints and the page
// templates, plan and create a site from a blueprint, and bring a newer
// built-in one to the user's copy.
type TemplatesAPI struct {
	env siteEnv
	s   *Session
}

// Templates returns the templates of this session: the templates folder
// next to its config file (blueprints/ and pages/) and the built-in ones.
func (s *Session) Templates() TemplatesAPI {
	return TemplatesAPI{env: siteEnv{globalPath: s.globalPath, builtins: s.presets, pages: s.pagePre, note: s.noteTemplates}, s: s}
}

// noteTemplates reports what happened to the templates folder (the sites move):
// it shows under Problems on every page for the rest of the session, and goes
// to the log once.
func (s *Session) noteTemplates(n string) {
	s.mu.Lock()
	for _, have := range s.tmplNotes {
		if have == n {
			s.mu.Unlock()
			return
		}
	}
	s.tmplNotes = append(s.tmplNotes, n)
	s.mu.Unlock()
	logApp(slog.LevelWarn, n, "source", "templates")
}

// Notes are what happened to the templates folder in this session, such as the move of an old
// templates/sites (Problems shows the same).
func (t TemplatesAPI) Notes() []string {
	if t.s == nil {
		return nil
	}
	t.s.mu.Lock()
	defer t.s.mu.Unlock()
	return append([]string(nil), t.s.tmplNotes...)
}

// templateContext is the open folder's root and its templates: path rules, "" and none without a folder.
func (s *Session) templateContext() (string, []config.TemplateRule) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ws == nil {
		return "", nil
	}
	return s.ws.Root(), s.cfg.TemplateRules()
}

// List lists the site templates for the New Site dialog and the welcome
// screen, installing the built-in presets on first use.
func (t TemplatesAPI) List() []SiteTemplate {
	out := []SiteTemplate{}
	for _, tpl := range t.env.list().Templates {
		st := SiteTemplate{ID: tpl.ID, Name: tpl.Name, Description: tpl.Description, Version: tpl.Version, Builtin: tpl.Builtin, Update: tpl.Update, Error: tpl.Err}
		for _, a := range tpl.Ask {
			st.Ask = append(st.Ask, struct {
				Key     string `json:"key"`
				Label   string `json:"label"`
				Default string `json:"default"`
			}{a.Key, a.Label, a.Default})
		}
		out = append(out, st)
	}
	return out
}

// Listing is List with everything the templates package knows: warnings and
// notes included.
func (t TemplatesAPI) Listing() TemplateListing { return t.env.list() }

// Dir is the templates folder (templates.dir in config.yml, or the default
// next to it), "" when there is none. It installs the built-in presets first.
func (t TemplatesAPI) Dir() string { return t.env.list().Dir }

// Folder is Dir without installing anything, with the problems found.
func (t TemplatesAPI) Folder() (string, []string) { return t.env.dir() }

// New starts blueprint id in the blueprints folder, from blueprint from when
// given, and returns the new folder (zenmd blueprint new).
func (t TemplatesAPI) New(id, from string) (string, error) {
	dir, _ := t.env.dir()
	if dir == "" {
		return "", errors.New("there is no blueprints folder")
	}
	t.env.list() // install the built-ins, so --from can find them in the folder
	return sitetemplates.New(dir, t.env.builtins, id, from)
}

// Update brings the newer built-in blueprint id to the user's copy: in place
// when the copy was never edited, else beside it as <id>.new.
func (t TemplatesAPI) Update(id string) (sitetemplates.UpdateResult, error) {
	dir, _ := t.env.dir()
	if dir == "" {
		return sitetemplates.UpdateResult{}, errors.New("there is no blueprints folder")
	}
	t.env.list() // install what is missing, so there is a record
	return sitetemplates.Update(dir, t.env.builtins, id)
}

// Plan is a checked request to make a site: the template exists and dir is
// not inside the templates folder.
type Plan struct {
	t   sitetemplates.Template
	dir string
}

// Plan checks that a site can be made from template id in dir (new or
// existing, absolute) and returns the plan; nothing is written yet.
func (t TemplatesAPI) Plan(id, dir string) (Plan, error) {
	if !filepath.IsAbs(dir) {
		return Plan{}, errors.New("choose a folder")
	}
	tpl, l, ok := t.env.find(id)
	if !ok {
		return Plan{}, fmt.Errorf("there is no blueprint %q", id)
	}
	if l.Holds(dir) {
		return Plan{}, errors.New("choose a folder outside the blueprints folder")
	}
	return Plan{t: tpl, dir: dir}, nil
}

// Create writes the site; nothing is overwritten.
func (p Plan) Create(values map[string]string) error {
	_, err := sitetemplates.Create(p.t, p.dir, values)
	return err
}

// Holds reports whether dir is the templates folder or inside it.
func (l TemplateListing) Holds(dir string) bool { return l.Dir != "" && insideDir(dir, l.Dir) }

// insideDir reports whether p is dir or inside it.
func insideDir(p, dir string) bool {
	rel, err := filepath.Rel(dir, p)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
