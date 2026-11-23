package session

import (
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"

	"github.com/zenmark-org/zenmd/engine/config"
	"github.com/zenmark-org/zenmd/engine/layouts"
	"github.com/zenmark-org/zenmd/engine/sitetemplates"
	"github.com/zenmark-org/zenmd/engine/structure"
)

// StructureProblem is one thing a page lacks or does wrong, as Problems › Structure and zenmd check
// show it (engine/structure). Line is 1-based and never 0; Anchor is the id of the heading to mark
// ("" for the top of the page); Source is "template how-to" or "layout post".
type StructureProblem struct {
	Rule     string `json:"rule"`
	Message  string `json:"message"`
	Fix      string `json:"fix"`
	Severity string `json:"severity"`
	Line     int    `json:"line"`
	Anchor   string `json:"anchor"`
	Source   string `json:"source"`
}

// ErrNoPageTemplate is returned by CheckFolder for a Template that is not a page template.
var ErrNoPageTemplate = errors.New("no such page template")

// structEnv is what choosing and reading a page's template needs, taken from the session under its lock.
type structEnv struct {
	cfg     *config.Config
	layouts *layouts.Registry
	root    string // the open folder
	dir     string // the templates folder, "" for none
	pages   fs.FS  // the built-in page templates
	force   string // a template id that applies to every page (zenmd check --template)
}

func (s *Session) structEnvLocked() structEnv {
	e := structEnv{cfg: s.cfg, layouts: s.lreg, pages: s.pagePre}
	if s.ws != nil {
		e.root = s.ws.Root()
	}
	if s.globalPath != "" && s.configDir != "" {
		e.dir, _ = config.Load(s.globalPath, "").TemplatesDir(s.configDir)
	}
	return e
}

// check reads the page's template (frontmatter template:, else the folder's path rule for it) and
// its layout's structure, and checks the page source against both. A page with neither has none.
func (e structEnv) check(path string, src []byte) []StructureProblem {
	out := []StructureProblem{}
	if e.cfg == nil && e.layouts == nil && e.pages == nil {
		return out
	}
	meta, _, _ := config.SplitFrontmatter(src)
	var tpl, lay *structure.Template
	problem := func(msg, fix string) {
		out = append(out, StructureProblem{Rule: "template", Message: msg, Fix: fix, Severity: structure.SevWarning, Line: 1})
	}

	id, optOut := e.force, false
	if id == "" {
		switch v := meta["template"].(type) {
		case bool:
			optOut = !v
		case string:
			id = strings.TrimSpace(v)
		}
	}
	// A path rule such as releases/** is for the pages of the folder; the folder's own index.md (or the
	// README.md that stands in for it) is its home, a list page, and follows no entry template unless
	// its frontmatter says template:.
	if base := strings.ToLower(path[strings.LastIndex(path, "/")+1:]); id == "" && !optOut && e.cfg != nil && base != "index.md" && base != "readme.md" {
		for _, r := range e.cfg.TemplateRules() {
			if layouts.Match(r.Pattern, path) {
				id = r.Name
				break
			}
		}
	}
	if id != "" {
		if data, _, ok := sitetemplates.FindPage(e.dir, e.pages, e.root, id); !ok {
			problem(fmt.Sprintf("Page template %q is not found", id), "Check template: in the frontmatter (or the folder's templates: rule), or Settings › Templates")
		} else if t, err := structure.Parse(id, data); err != nil {
			problem(fmt.Sprintf("Page template %q is not valid: %s", id, strings.TrimPrefix(err.Error(), "template "+id+": ")), "Fix the template file")
		} else {
			tpl = t
		}
	}

	if e.layouts != nil && e.cfg != nil {
		choice := layouts.Resolve(path, meta, e.cfg.ForPage(meta).Layouts())
		if l := e.layouts.Get(choice.Name); l != nil && len(l.Manifest().Structure) > 0 {
			if t, err := structure.ParseLayout(choice.Name, l.Manifest().Structure); err != nil {
				problem(fmt.Sprintf("The structure of the %s layout is not valid: %s", choice.Name, err), "Fix the structure: block in the layout's layout.yml")
			} else {
				lay = t
			}
		}
	}
	if tpl == nil && lay == nil {
		return out
	}
	for _, p := range structure.Check(structure.Analyse(src), tpl, lay) {
		out = append(out, StructureProblem{Rule: p.Rule, Message: p.Message, Fix: p.Fix, Severity: p.Severity, Line: p.Line, Anchor: p.Anchor, Source: p.Source})
	}
	return out
}

// CheckOptions are the options of `zenmd check`.
type CheckOptions struct {
	Dir      string // the folder
	Template string // check every page against this page template ("" for each page's own)
}

// PageCheck is the problems of one page: its structure and the notes its render gave.
type PageCheck struct {
	Path     string
	Problems []StructureProblem
}

// CheckReport is the result of CheckFolder. Results has the pages that have problems, in path order.
type CheckReport struct {
	Pages   int
	Results []PageCheck
}

// CheckFolder is `zenmd check`: it checks every markdown page of the folder against its page template
// and its layout's structure, and adds what the page's render says (such as a title: the H1 makes
// useless) as problems with the rule page-warning. It opens no window, asks nothing and writes nothing.
func CheckFolder(env SiteEnv, o CheckOptions) (CheckReport, error) {
	root, err := filepath.Abs(o.Dir)
	if err != nil {
		return CheckReport{}, err
	}
	st := New(Options{GlobalPath: env.GlobalPath, Registry: env.Registry, Layouts: env.Layouts, Dialects: env.Dialects, Version: env.Version})
	defer st.Close()
	if err := st.Open(root); err != nil {
		return CheckReport{}, st.cleanErr(".", err)
	}
	st.mu.Lock()
	se := st.structEnvLocked()
	ws := st.ws
	st.mu.Unlock()
	if o.Template != "" {
		if _, _, ok := sitetemplates.FindPage(se.dir, se.pages, se.root, o.Template); !ok {
			return CheckReport{}, fmt.Errorf("%w %q", ErrNoPageTemplate, o.Template)
		}
		se.force = o.Template
	}
	files, err := ws.Files()
	if err != nil {
		return CheckReport{}, err
	}
	sort.Strings(files)
	var rep CheckReport
	for _, p := range files {
		if !isMarkdownPath(p) {
			continue
		}
		rep.Pages++
		st.mu.Lock()
		src, err := ws.Read(p)
		var sn renderSnap
		if err == nil {
			sn = st.snapLocked(p)
		}
		st.mu.Unlock()
		if err != nil {
			rep.Results = append(rep.Results, PageCheck{Path: p, Problems: []StructureProblem{{Rule: "page", Message: "The page could not be read: " + st.cleanErr(p, err).Error(), Fix: "Check the file", Severity: structure.SevError, Line: 1}}})
			continue
		}
		sn.extWarns, sn.tmplWarns = nil, nil // reported once for the folder, not on every page
		sn.structure = se
		page, err := sn.render(p, src)
		problems := []StructureProblem{}
		if err != nil {
			problems = append(problems, StructureProblem{Rule: "page", Message: err.Error(), Fix: "Check the page", Severity: structure.SevError, Line: 1})
		} else {
			problems = append(problems, page.Structure...)
			for _, w := range page.Warnings {
				sev := structure.SevWarning
				if strings.HasSuffix(w, "and it says the same") {
					sev = structure.SevSuggestion
				}
				problems = append(problems, StructureProblem{Rule: "page-warning", Message: w, Fix: warningFix(w), Severity: sev, Line: 1})
			}
		}
		if len(problems) > 0 {
			rep.Results = append(rep.Results, PageCheck{Path: p, Problems: problems})
		}
	}
	return rep, nil
}

// warningFix is the Fix line of the render warnings the checker knows (the Inspector's warningFix has the same).
func warningFix(w string) string {
	switch {
	case strings.HasPrefix(w, "title: is ignored because the page has an H1"):
		return "Remove title: from the frontmatter"
	case strings.HasPrefix(w, "version must be a semantic version"):
		return `Write the version without a v, quoted, such as version: "0.2.0"`
	case strings.HasPrefix(w, "unknown layout"):
		return "Settings › Layouts, or change layout: in the front matter"
	}
	return ""
}
