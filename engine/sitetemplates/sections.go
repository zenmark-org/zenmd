package sitetemplates

import (
	"errors"
	"fmt"
	"io/fs"
	"path"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// Home is the README.md start page: a short contents page that links each
// section in order (the sidebar is built from links).
type Home struct{ Title string }

// Section is one part of a blueprint: a folder with an index page (or one
// file), the page template its pages follow, and its starting pages.
type Section struct {
	ID, Title string
	Folder    string // a folder section: <folder>/index.md and the pages in it
	File      string // a file section: this one page
	Template  string // page template id: a path rule in .zenmd.yml, and the skeleton of the pages
	Index     Index
	Order     string // "", "newest-first" or "title": written as nav: { sort: ... } on the index
	Pages     []Page
}

// Index says how a folder section's index.md is made: a landing page with
// cards by default, a layout, or the body of a page template.
type Index struct {
	Layout, Template, Tagline, Title string
}

// Page is a starting page of a section.
type Page struct {
	File, Title, Layout string
	Set                 []KV // frontmatter fields, in file order
}

// KV is one frontmatter field.
type KV struct{ Key, Value string }

var orders = map[string]bool{"newest-first": true, "title": true}

func parseNames(key string, v *yaml.Node) ([]string, error) {
	if v.Kind == yaml.ScalarNode && v.Tag == "!!null" {
		return nil, nil
	}
	if v.Kind != yaml.SequenceNode {
		return nil, fmt.Errorf("%s must be a list", key)
	}
	var out []string
	for _, x := range v.Content {
		if x.Kind != yaml.ScalarNode || x.Value == "" {
			return nil, fmt.Errorf("%s must be a list of names", key)
		}
		out = append(out, x.Value)
	}
	return out, nil
}

func parseHome(v *yaml.Node) (*Home, error) {
	h := &Home{}
	if v.Kind == yaml.ScalarNode && v.Tag == "!!null" {
		return h, nil
	}
	if v.Kind != yaml.MappingNode {
		return nil, errors.New("home must be a mapping such as { title: \"{{title}}\" }")
	}
	for i := 0; i+1 < len(v.Content); i += 2 {
		k, x := v.Content[i].Value, v.Content[i+1]
		if k != "title" {
			return nil, fmt.Errorf("home: unknown key %q", k)
		}
		if x.Kind != yaml.ScalarNode {
			return nil, errors.New("home: title must be text")
		}
		h.Title = x.Value
	}
	return h, nil
}

// mapping walks a mapping node, giving each key's value to fn.
func mapping(v *yaml.Node, what string, fn func(k string, x *yaml.Node) error) error {
	if v.Kind != yaml.MappingNode {
		return fmt.Errorf("%s must be a mapping", what)
	}
	for i := 0; i+1 < len(v.Content); i += 2 {
		if err := fn(v.Content[i].Value, v.Content[i+1]); err != nil {
			return err
		}
	}
	return nil
}

func text(what string, x *yaml.Node) (string, error) {
	if x.Kind != yaml.ScalarNode {
		return "", fmt.Errorf("%s must be text", what)
	}
	return strings.TrimSpace(x.Value), nil
}

func parseSections(v *yaml.Node) ([]Section, error) {
	if v.Kind == yaml.ScalarNode && v.Tag == "!!null" {
		return nil, nil
	}
	if v.Kind != yaml.SequenceNode {
		return nil, errors.New("sections must be a list")
	}
	var out []Section
	ids := map[string]bool{}
	for n, item := range v.Content {
		var s Section
		what := fmt.Sprintf("sections[%d]", n+1)
		err := mapping(item, what, func(k string, x *yaml.Node) (err error) {
			switch k {
			case "id":
				s.ID, err = text(what+" id", x)
			case "title":
				s.Title, err = text(what+" title", x)
			case "folder":
				s.Folder, err = text(what+" folder", x)
			case "file":
				s.File, err = text(what+" file", x)
			case "template":
				s.Template, err = text(what+" template", x)
			case "order":
				s.Order, err = text(what+" order", x)
			case "index":
				err = mapping(x, what+" index", func(ik string, ix *yaml.Node) (err error) {
					switch ik {
					case "layout":
						s.Index.Layout, err = text("index layout", ix)
					case "template":
						s.Index.Template, err = text("index template", ix)
					case "tagline":
						s.Index.Tagline, err = text("index tagline", ix)
					case "title":
						s.Index.Title, err = text("index title", ix)
					default:
						err = fmt.Errorf("%s index: unknown key %q", what, ik)
					}
					return
				})
			case "pages":
				s.Pages, err = parsePages(what, x)
			default:
				err = fmt.Errorf("%s: unknown key %q", what, k)
			}
			return
		})
		if err != nil {
			return nil, err
		}
		if err := s.check(what, ids); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, nil
}

func parsePages(what string, v *yaml.Node) ([]Page, error) {
	if v.Kind == yaml.ScalarNode && v.Tag == "!!null" {
		return nil, nil
	}
	if v.Kind != yaml.SequenceNode {
		return nil, fmt.Errorf("%s pages must be a list", what)
	}
	var out []Page
	for _, item := range v.Content {
		var p Page
		err := mapping(item, what+" page", func(k string, x *yaml.Node) (err error) {
			switch k {
			case "file":
				p.File, err = text("page file", x)
			case "title":
				p.Title, err = text("page title", x)
			case "layout":
				p.Layout, err = text("page layout", x)
			case "set":
				err = mapping(x, "page set", func(sk string, sx *yaml.Node) error {
					val, err := text("set "+sk, sx)
					p.Set = append(p.Set, KV{sk, val})
					return err
				})
			default:
				err = fmt.Errorf("%s page: unknown key %q", what, k)
			}
			return
		})
		if err != nil {
			return nil, err
		}
		if p.File == "" {
			return nil, fmt.Errorf("%s: a page needs a file", what)
		}
		if p.Title == "" {
			p.Title = strings.TrimSuffix(path.Base(p.File), ".md")
		}
		out = append(out, p)
	}
	return out, nil
}

// safeRel reports whether p is a plain relative slash path inside the folder.
func safeRel(p string) bool {
	if p == "" || strings.ContainsAny(p, "\\\x00") || path.IsAbs(p) {
		return false
	}
	c := path.Clean(p)
	return c == p && c != "." && c != ".." && !strings.HasPrefix(c, "../")
}

func (s Section) check(what string, ids map[string]bool) error {
	switch {
	case !idRE.MatchString(s.ID):
		return fmt.Errorf("%s: id %q must use lower case letters, digits and -", what, s.ID)
	case ids[s.ID]:
		return fmt.Errorf("%s: id %q is used twice", what, s.ID)
	case s.Title == "":
		return fmt.Errorf("%s (%s): title is required", what, s.ID)
	case (s.Folder == "") == (s.File == ""):
		return fmt.Errorf("%s (%s): give either folder: or file:", what, s.ID)
	case s.Folder != "" && !safeRel(s.Folder), s.File != "" && (!safeRel(s.File) || !strings.HasSuffix(s.File, ".md")):
		return fmt.Errorf("%s (%s): folder and file must be plain paths inside the new folder (a file ends in .md)", what, s.ID)
	case s.Template != "" && !idRE.MatchString(s.Template), s.Index.Template != "" && !idRE.MatchString(s.Index.Template):
		return fmt.Errorf("%s (%s): template must be a page template id such as how-to", what, s.ID)
	case s.Index.Layout != "" && !idRE.MatchString(s.Index.Layout):
		return fmt.Errorf("%s (%s): index layout must be a layout id such as landing", what, s.ID)
	case s.Order != "" && !orders[s.Order]:
		return fmt.Errorf("%s (%s): order %q must be newest-first or title", what, s.ID, s.Order)
	case s.File != "" && (len(s.Pages) > 0 || s.Index != Index{}):
		return fmt.Errorf("%s (%s): a file section has no index or pages", what, s.ID)
	}
	for _, p := range s.Pages {
		if !safeRel(strings.ReplaceAll(strings.ReplaceAll(p.File, "{{", ""), "}}", "")) || !strings.HasSuffix(p.File, ".md") || s.Folder == "" {
			return fmt.Errorf("%s (%s): page %q must be a .md file name inside the section folder", what, s.ID, p.File)
		}
	}
	ids[s.ID] = true
	return nil
}

// pageTemplate is what a section needs of a page template file: its
// skeleton, its layout and the defaults of its fields. Checking comes later.
type pageTemplate struct {
	ID       string    `yaml:"id"`
	Layout   string    `yaml:"layout"`
	Skeleton string    `yaml:"skeleton"`
	Fields   yaml.Node `yaml:"fields"`
}

type field struct{ key, def string }

func (p pageTemplate) defaults() []field {
	var out []field
	if p.Fields.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(p.Fields.Content); i += 2 {
		var f struct {
			Default string `yaml:"default"`
		}
		_ = p.Fields.Content[i+1].Decode(&f)
		if f.Default != "" {
			out = append(out, field{p.Fields.Content[i].Value, f.Default})
		}
	}
	return out
}

// pageTemplateBytes finds a page template: the blueprint's own
// .zen/templates first, then the built-in ones.
func (t Template) pageTemplateBytes(id string) ([]byte, bool) {
	if t.fsys != nil {
		if b, err := fs.ReadFile(t.fsys, ".zen/templates/"+id+".yml"); err == nil {
			return b, true
		}
	}
	if t.PageSource != nil {
		if b, err := fs.ReadFile(t.PageSource, id+".yml"); err == nil {
			return b, true
		}
	}
	return nil, false
}

// pageTemplateOrNone is pageTemplate, and no error for an empty id.
func (t Template) pageTemplateOrNone(id string) (pageTemplate, error) {
	if id == "" {
		return pageTemplate{}, nil
	}
	return t.pageTemplate(id)
}

func (t Template) pageTemplate(id string) (pageTemplate, error) {
	b, ok := t.pageTemplateBytes(id)
	if !ok {
		return pageTemplate{}, fmt.Errorf("the page template %q is not available (it is not in the blueprint's .zen/templates or among the built-in ones)", id)
	}
	var p pageTemplate
	if err := yaml.Unmarshal(b, &p); err != nil {
		return p, fmt.Errorf("page template %s: %v", id, err)
	}
	return p, nil
}

var plainRE = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._/+-]*$`)

// scalar is s as a YAML value: plain when that is safe, quoted otherwise.
func scalar(s string) string {
	s = strings.NewReplacer("\r", " ", "\n", " ").Replace(s)
	switch strings.ToLower(s) {
	case "true", "false", "yes", "no", "on", "off", "null", "y", "n":
		return `"` + s + `"`
	}
	if plainRE.MatchString(s) {
		return s
	}
	b, err := yaml.Marshal(s)
	if err != nil {
		return `""`
	}
	return strings.TrimRight(string(b), "\n")
}

// frontmatter writes a --- block of fields.
func frontmatter(kvs []KV) string {
	var b strings.Builder
	b.WriteString("---\n")
	for _, kv := range kvs {
		b.WriteString(kv.Key + ": " + kv.Value + "\n")
	}
	b.WriteString("---\n")
	return b.String()
}

func setKV(kvs []KV, k, v string) []KV {
	for i := range kvs {
		if kvs[i].Key == k {
			kvs[i].Value = v
			return kvs
		}
	}
	return append(kvs, KV{k, v})
}

// sectionLinks is a markdown list of links to every section, relative to the
// page at dir ("" for the root), without the section whose folder is skip.
func sectionLinks(secs []Section, dir, skip string) string {
	var b strings.Builder
	for _, s := range secs {
		if skip != "" && s.Folder == skip {
			continue // an index does not link to itself
		}
		target := s.File
		if s.Folder != "" {
			target = s.Folder + "/index.md"
		}
		fmt.Fprintf(&b, "- [%s](%s)\n", s.Title, relTo(dir, target))
	}
	return strings.TrimRight(b.String(), "\n")
}

// fileLinks lists the other markdown files of files: as links after the
// sections, so the sidebar finds them.
func (t Template) fileLinks() string {
	var b strings.Builder
	for _, f := range t.Files {
		if strings.HasSuffix(strings.ToLower(f), ".md") && safeRel(f) {
			fmt.Fprintf(&b, "\n- [%s](%s)", strings.TrimSuffix(path.Base(f), path.Ext(f)), f)
		}
	}
	return b.String()
}

// relTo is target (a root-relative slash path) as a link from the folder dir.
func relTo(dir, target string) string {
	if dir == "" || dir == "." {
		return target
	}
	up := strings.Repeat("../", strings.Count(dir, "/")+1)
	return up + target
}

// generated is a file the sections write.
type generated struct{ dest, data, extra string }

// generate works out the files the sections make: README.md, each section's
// index and pages, the page templates and the .zenmd.yml fragment. vals are
// the values of the site (title, author, ...).
func (t Template) generate(vals map[string]string) ([]generated, string, error) {
	if len(t.Sections) == 0 && t.Layout == "" && len(t.PageTemplates) == 0 {
		return nil, "", nil
	}
	var out []generated
	var rules []string
	pageVals := func(dir, skip, title string, extra []KV) map[string]string {
		v := make(map[string]string, len(vals)+4)
		for k, x := range vals {
			v[k] = x
		}
		v["site"] = vals["title"]
		v["title"] = title
		v["sections"] = sectionLinks(t.Sections, dir, skip)
		for _, kv := range extra {
			v[kv.Key] = kv.Value
		}
		return v
	}
	// body renders a page template's skeleton for a page and returns the
	// frontmatter fields the template adds.
	body := func(id, dir, skip, title string, set []KV) (string, []KV, string, error) {
		pt, err := t.pageTemplate(id)
		if err != nil {
			return "", nil, "", err
		}
		var kvs []KV
		pv := pageVals(dir, skip, title, set)
		for _, d := range pt.defaults() {
			kvs = append(kvs, KV{d.key, scalar(replace(d.def, pv))})
		}
		for _, kv := range set {
			kvs = setKV(kvs, kv.Key, scalar(replace(kv.Value, pv)))
		}
		for _, kv := range kvs { // fields can be used as {{version}}
			pv[kv.Key] = strings.Trim(kv.Value, `"`)
		}
		return replace(pt.Skeleton, pv), kvs, pt.Layout, nil
	}

	if len(t.Sections) > 0 {
		ht := "{{title}}"
		if t.Home != nil && t.Home.Title != "" {
			ht = t.Home.Title
		}
		title := replace(ht, vals)
		out = append(out, generated{dest: "README.md", data: frontmatter([]KV{{"title", scalar(title)}}) +
			"# " + title + "\n\n" + "Start here. Each section below has its own folder.\n\n" + sectionLinks(t.Sections, "", "") + t.fileLinks() + "\n"})
	}
	for _, s := range t.Sections {
		for _, id := range []string{s.Template, s.Index.Template} {
			if _, err := t.pageTemplateOrNone(id); err != nil {
				return nil, "", fmt.Errorf("section %s: %w", s.ID, err)
			}
		}
		stitle := replace(s.Title, vals)
		if s.File != "" {
			content, err := t.sectionPage(s.Template, "", stitle, nil, body)
			if err != nil {
				return nil, "", err
			}
			out = append(out, generated{dest: replace(s.File, vals), data: content})
			if s.Template != "" {
				rules = append(rules, fmt.Sprintf("  %q: %s", replace(s.File, vals), s.Template))
			}
			continue
		}
		folder := replace(s.Folder, vals)
		// The index page.
		idx := []KV{{"title", scalar(stitle)}}
		var text string
		switch {
		case s.Index.Template != "":
			b, kvs, layout, err := body(s.Index.Template, folder, folder, stitle, nil)
			if err != nil {
				return nil, "", err
			}
			if s.Index.Layout != "" {
				layout = s.Index.Layout
			}
			if layout != "" && layout != t.Layout {
				idx = append(idx, KV{"layout", layout})
			}
			idx = append(idx, kvs...)
			text = b
		case s.Index.Layout == "glossary":
			idx = append(idx, KV{"layout", "glossary"})
			text = "# " + stitle + "\n\nThe terms of " + vals["title"] + ", from the pages in this folder.\n"
		default:
			layout := s.Index.Layout
			if layout == "" {
				layout = "landing"
			}
			idx = append(idx, KV{"layout", layout})
			if s.Index.Tagline != "" {
				idx = append(idx, KV{"tagline", scalar(replace(s.Index.Tagline, vals))})
			}
			text = "# " + stitle + "\n"
		}
		if len(s.Pages) > 0 && s.Index.Layout != "glossary" { // the sidebar is built from links
			var links []string
			for _, p := range s.Pages {
				links = append(links, fmt.Sprintf("- [%s](%s)", replace(p.Title, vals), replace(p.File, vals)))
			}
			if s.Index.Template != "" {
				text = strings.TrimRight(text, "\n") + "\n" + strings.Join(links, "\n") + "\n"
			} else {
				text += "\n" + strings.Join(links, "\n") + "\n"
			}
		}
		if s.Order != "" {
			idx = append(idx, KV{"nav", "{ children: folder, sort: " + s.Order + " }"})
		} else if s.Index.Layout == "glossary" {
			idx = append(idx, KV{"nav", "{ children: folder, sort: title }"})
		}
		out = append(out, generated{dest: folder + "/index.md", data: frontmatter(idx) + text})
		if s.Template != "" {
			rules = append(rules, fmt.Sprintf("  %q: %s", folder+"/**", s.Template))
		}
		// The starting pages.
		for _, p := range s.Pages {
			ptitle := replace(p.Title, vals)
			set := make([]KV, 0, len(p.Set))
			for _, kv := range p.Set {
				set = append(set, KV{kv.Key, replace(kv.Value, vals)})
			}
			var content string
			if p.Layout != "" { // its own layout: no skeleton, and it follows no template
				fm := []KV{{"title", scalar(ptitle)}, {"layout", scalar(p.Layout)}, {"template", "false"}}
				for _, kv := range set {
					fm = append(fm, KV{kv.Key, scalar(kv.Value)})
				}
				content = frontmatter(fm) + layoutBody(p.Layout, ptitle)
			} else {
				var err error
				if content, err = t.sectionPage(s.Template, folder, ptitle, set, body); err != nil {
					return nil, "", err
				}
				if s.Template != "" {
					// the layout comes from the folder's default unless the template differs (sectionPage)
				}
			}
			out = append(out, generated{dest: path.Join(folder, replace(p.File, vals)), data: content})
		}
	}
	// The page templates, copied so the folder carries its own structure.
	for _, id := range t.PageTemplates {
		if t.fsys != nil {
			if _, err := fs.Stat(t.fsys, ".zen/templates/"+id+".yml"); err == nil {
				continue // the blueprint ships its own file: it is copied as it is
			}
		}
		b, ok := t.pageTemplateBytes(id)
		if !ok {
			return nil, "", fmt.Errorf("templates: the page template %q is not available", id)
		}
		out = append(out, generated{dest: ".zen/templates/" + id + ".yml", data: string(b)})
	}
	// .zenmd.yml: the layout and the path rules.
	var frag strings.Builder
	if t.Layout != "" {
		frag.WriteString("layout: " + t.Layout + "\n")
	}
	if len(rules) > 0 {
		frag.WriteString("\n# Pages in these folders follow a page template (.zen/templates).\ntemplates:\n" + strings.Join(rules, "\n") + "\n")
	}
	return out, frag.String(), nil
}

// sectionPage writes one page from a page template (or a bare one without).
func (t Template) sectionPage(tmplID, dir, title string, set []KV, body func(id, dir, skip, title string, set []KV) (string, []KV, string, error)) (string, error) {
	fm := []KV{{"title", scalar(title)}}
	if tmplID == "" {
		for _, kv := range set {
			fm = append(fm, KV{kv.Key, scalar(kv.Value)})
		}
		return frontmatter(fm) + "# " + title + "\n", nil
	}
	b, kvs, layout, err := body(tmplID, dir, "", title, set)
	if err != nil {
		return "", err
	}
	if layout != "" && layout != t.Layout {
		fm = append(fm, KV{"layout", layout})
	}
	return frontmatter(append(fm, kvs...)) + b, nil
}

// layoutBody is a starting body for a page with its own layout.
func layoutBody(layout, title string) string {
	if layout == "slides" {
		return "# " + title + "\n\nWhat this is, in one line.\n\nNote: Say hello and why this matters.\n\n---\n\n## The problem\n\n- What is hard today\n- Who feels it\n\n---\n\n## The idea\n\n- What it does\n- Why it works\n\n---\n\n## What you get\n\n- The first result\n- The next step\n\n---\n\n## Try it\n\nWhere to start.\n"
	}
	return "# " + title + "\n"
}
