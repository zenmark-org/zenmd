// Package xref draws MyST's cross-references and the table of contents block:
//
//	{ref}`label`, {numref}`label`  a link to a (label)= target or a :name: id of the page
//	{doc}`path`                    a link to a page of the folder, with its title
//	{download}`path`               a link to a file of the folder
//	toctree                        a list of links to pages, with their titles
//
// and their zen forms :ref[label], :numref[label], :doc[path], :download[path]
// and :::toctree. A role takes a title too: {ref}`the intro <label>`.
//
// A page link or a file link is written by the page's own link handling
// (Context.Markdown), so it is the same markup as a link in the text. Paths
// are relative to the page and never leave the folder: no .., no absolute
// path, no scheme. A {ref} is turned into a link when the page is finished
// (HTML), because the numbers of figures, tables and equations are known only
// then; xref is registered after them for that reason.
package xref

import (
	"embed"
	"errors"
	"html"
	"io/fs"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"github.com/zenmark-org/zenmd/engine/plugins"
	xhtml "golang.org/x/net/html"
)

//go:embed assets/xref.css
var assets embed.FS

type plugin struct{}

// New returns the xref plugin.
func New() plugins.Plugin { return plugin{} }

func (plugin) Manifest() plugins.Manifest {
	return plugins.Manifest{
		ID:          "xref",
		Version:     "0.1.0",
		APIVersion:  plugins.APIVersion,
		Description: "Cross-references: {ref}, {numref}, {doc}, {download} and toctree",
		Default:     true,
		Help: plugins.Help{
			Summary: "Links inside a page and inside the folder. The {ref} role (or :ref[label]) links to a (label)= target or a :name: of the page and shows its text; {numref} (or :numref[label]) shows the number of a figure, table or equation, and takes Fig. %s as its text. {doc} (or :doc[path]) links to a page of the folder and shows its title; {download} (or :download[path]) links to a file of the folder. All of them take a title as {doc}`the guide <guide/start>`. The {toctree} directive (or :::toctree) is a list of links to pages with their titles, options caption, maxdepth, hidden, name and class. Paths are relative to the page and stay in the folder. An unknown label, page or file shows a warning.",
			Snippets: []plugins.Snippet{
				{Title: "A link to a label", Markdown: "(setup)=\n## Setup\n\nSee :ref[setup], or :ref[the setup section <setup>].\n"},
				{Title: "A link to a label, MyST form", Markdown: "(setup)=\n## Setup\n\nSee {ref}`setup`, or {ref}`the setup section <setup>`.\n"},
				{Title: "The number of a figure", Markdown: "See :numref[sales] and :numref[Fig. %s <sales>].\n"},
				{Title: "The number of a figure, MyST form", Markdown: "See {numref}`sales` and {numref}`Fig. %s <sales>`.\n"},
				{Title: "A link to a page", Markdown: "Read :doc[guide/start], or :doc[the guide <guide/start>].\n"},
				{Title: "A link to a page, MyST form", Markdown: "Read {doc}`guide/start`, or {doc}`the guide <guide/start>`.\n"},
				{Title: "A link to a file", Markdown: "Get :download[images/chart.png].\n"},
				{Title: "A link to a file, MyST form", Markdown: "Get {download}`images/chart.png`.\n"},
				{Title: "A list of pages", Markdown: ":::toctree{caption=\"Guide\"}\nguide/start\nThe basics <guide/basics>\n:::\n"},
				{Title: "A list of pages, MyST form", Markdown: "```{toctree}\n:caption: Guide\n\nguide/start\nThe basics <guide/basics>\n```\n"},
			},
		},
		Blocks:  []string{"toctree"},
		Inlines: []string{"ref", "numref", "doc", "download"},
	}
}

func (plugin) Assets() fs.FS {
	sub, err := fs.Sub(assets, "assets")
	if err != nil {
		panic(err)
	}
	return sub
}

func (plugin) Entry() plugins.Assets { return plugins.Assets{CSS: []string{"xref.css"}} }

// ---- helpers ---------------------------------------------------------------

var titledRE = regexp.MustCompile(`^(.*?)\s*<([^<>]+)>$`)

// split reads `Title <target>` or `target`.
func split(s string) (title, target string) {
	s = strings.TrimSpace(s)
	if m := titledRE.FindStringSubmatch(s); m != nil {
		return strings.TrimSpace(m[1]), strings.TrimSpace(m[2])
	}
	return "", s
}

// slug turns a label into the id the targets carry: lower case, letters,
// digits and . _ : -.
func slug(s string) string {
	var sb strings.Builder
	dash := false
	for _, r := range strings.ToLower(strings.TrimSpace(s)) {
		if r < 128 && (r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '_' || r == '.' || r == ':') {
			sb.WriteRune(r)
			dash = false
		} else if !dash && sb.Len() > 0 {
			sb.WriteByte('-')
			dash = true
		}
		if sb.Len() >= 100 {
			break
		}
	}
	return strings.Trim(sb.String(), "-")
}

// safePath reports whether p, written on the page at pagePath, may name a
// page or file of the folder: relative, no scheme, no characters that are not
// file-name text, and a path that stays inside the folder (a .. may climb out
// of the page's own folder, but never out of the open folder).
func safePath(pagePath, p string) bool {
	if p == "" || len(p) > 500 || strings.HasPrefix(p, "/") || strings.HasPrefix(p, "~") {
		return false
	}
	for _, r := range p {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
		case strings.ContainsRune(" _-./()+,@=", r):
		default:
			return false
		}
	}
	if strings.Contains(p, "//") {
		return false
	}
	joined := path.Join(path.Dir(pagePath), p)
	return joined != ".." && !strings.HasPrefix(joined, "../") && joined != "."
}

// fileName is the name a page or file shows without a title: the last
// segment, without .md.
func fileName(p string) string {
	base := path.Base(p)
	for _, ext := range []string{".md", ".markdown"} {
		base = strings.TrimSuffix(base, ext)
	}
	return base
}

var aRE = regexp.MustCompile(`(?s)<a [^>]*>.*?</a>`)

// escapePunct backslash-escapes ASCII punctuation, so text reaches the page's
// markdown as plain text.
func escapePunct(s string) string {
	var sb strings.Builder
	for _, r := range s {
		switch {
		case r == '\n' || r == '\r':
			sb.WriteByte(' ')
		case r < 128 && strings.ContainsRune("!\"#$%&'()*+,-./:;<=>?@[\\]^_`{|}~", r):
			sb.WriteByte('\\')
			sb.WriteRune(r)
		default:
			sb.WriteRune(r)
		}
	}
	return sb.String()
}

// link is the page's own link to dest, showing text; broken when dest does
// not lead anywhere.
func link(ctx plugins.Context, text, dest string) (string, bool) {
	if ctx.Markdown == nil {
		return "", false
	}
	out, err := ctx.Markdown("[" + escapePunct(text) + "](<" + strings.NewReplacer(`\`, `\\`, `&`, `\&`).Replace(dest) + ">)")
	if err != nil {
		return "", false
	}
	a := aRE.FindString(out)
	if a == "" || strings.Contains(a, `data-zen-link="broken"`) {
		return "", false
	}
	return a, true
}

func warn(ctx plugins.Context, msg string) {
	if ctx.Warn != nil {
		ctx.Warn(msg)
	}
}

// ---- inline roles -----------------------------------------------------------

const (
	markOpen = "⟦zx|"
	markEnd  = "⟧"
)

var markRE = regexp.MustCompile(`⟦zx\|(ref|numref)\|([^|⟧]*)\|([^⟧]*)⟧`)

var delimiters = strings.NewReplacer("⟦", "", "⟧", "", "|", "")

func (plugin) RenderInline(ctx plugins.Context, in plugins.Inline) (string, error) {
	for k := range in.Attrs {
		warn(ctx, "option "+k+" is not supported on :"+in.Name+"[…] and was ignored")
	}
	title, target := split(in.Text)
	switch in.Name {
	case "ref", "numref":
		id := slug(target)
		if id == "" {
			return "", errors.New("{" + in.Name + "} needs the label of a target")
		}
		return markOpen + in.Name + "|" + id + "|" + html.EscapeString(delimiters.Replace(title)) + markEnd, nil
	case "doc":
		return docLink(ctx, title, target)
	case "download":
		return downloadLink(ctx, title, target)
	}
	return html.EscapeString(in.Text), nil
}

func docLink(ctx plugins.Context, title, target string) (string, error) {
	if !safePath(ctx.Path, target) {
		warn(ctx, "{doc} needs the path of a page of this folder, relative to this page and inside the folder")
		return html.EscapeString(firstNonEmpty(title, clip(target))), nil
	}
	shown := title
	if ctx.PageTitle != nil {
		t, ok := ctx.PageTitle(target)
		if !ok {
			warn(ctx, "{doc} points to "+clip(target)+", which is not a page of this folder")
			return html.EscapeString(firstNonEmpty(title, fileName(target))), nil
		}
		shown = firstNonEmpty(title, t)
	}
	shown = firstNonEmpty(shown, fileName(target))
	a, ok := link(ctx, shown, target)
	if !ok {
		warn(ctx, "{doc} points to "+clip(target)+", which is not a page of this folder")
		return html.EscapeString(shown), nil
	}
	return a, nil
}

func downloadLink(ctx plugins.Context, title, target string) (string, error) {
	if !safePath(ctx.Path, target) {
		warn(ctx, "{download} needs the path of a file of this folder, relative to this page and inside the folder")
		return html.EscapeString(firstNonEmpty(title, clip(target))), nil
	}
	shown := firstNonEmpty(title, path.Base(target))
	a, ok := link(ctx, shown, target)
	if !ok {
		warn(ctx, "{download} cannot link to "+clip(target)+": it is not in this folder, or it is not a page or an image")
		return html.EscapeString(shown), nil
	}
	return a, nil
}

func clip(s string) string {
	if r := []rune(s); len(r) > 80 {
		return string(r[:80]) + "…"
	}
	return s
}

func firstNonEmpty(a, b string) string {
	if strings.TrimSpace(a) != "" {
		return a
	}
	return b
}

// ---- resolving labels when the page is finished -------------------------------

type target struct {
	text string // the heading's text, or ""
	num  string // "Figure 1", "Table 2", "(3)", or ""
}

var numberRE = regexp.MustCompile(`[0-9]+(?:\.[0-9]+)*`)

// HTML turns the marks of {ref} and {numref} into links, now that the
// figures, tables and equations of the page are numbered.
func (plugin) HTML(ctx plugins.Context, s string) (string, error) {
	if !strings.Contains(s, markOpen) {
		return s, nil
	}
	ids := indexTargets(s)
	return markRE.ReplaceAllStringFunc(s, func(m string) string {
		sub := markRE.FindStringSubmatch(m)
		kind, id, title := sub[1], sub[2], sub[3] // title is already escaped
		t, found := ids[id]
		if !found {
			warn(ctx, "{"+kind+"} points to the label "+id+", which this page does not have")
			return firstNonEmpty(title, html.EscapeString(id))
		}
		text := title
		switch {
		case kind == "numref" && t.num == "":
			warn(ctx, "{numref} points to "+id+", which has no number (only figures, tables and equations with a label or name are numbered)")
			text = firstNonEmpty(title, html.EscapeString(firstNonEmpty(t.text, id)))
		case kind == "numref":
			n := numberRE.FindString(t.num)
			if title == "" {
				text = html.EscapeString(t.num)
			} else {
				text = strings.NewReplacer("%s", n, "{number}", n, "{name}", html.EscapeString(t.text)).Replace(title)
			}
		case title == "":
			text = html.EscapeString(firstNonEmpty(t.text, firstNonEmpty(t.num, id)))
		}
		return `<a class="zx-ref" href="#` + id + `">` + text + `</a>`
	}), nil
}

// indexTargets finds, for every id of the page, the heading text it names
// and the number its figure, table or equation has.
func indexTargets(page string) map[string]target {
	out := map[string]target{}
	doc, err := xhtml.Parse(strings.NewReader(page))
	if err != nil {
		return out
	}
	var walk func(n *xhtml.Node)
	walk = func(n *xhtml.Node) {
		if n.Type == xhtml.ElementNode {
			if id := attr(n, "id"); id != "" {
				if _, seen := out[id]; !seen {
					out[id] = describe(n)
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	return out
}

func attr(n *xhtml.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}

func isHeading(n *xhtml.Node) bool {
	return n != nil && n.Type == xhtml.ElementNode && len(n.Data) == 2 && n.Data[0] == 'h' && n.Data[1] >= '1' && n.Data[1] <= '6'
}

func text(n *xhtml.Node) string {
	var sb strings.Builder
	var walk func(n *xhtml.Node)
	walk = func(n *xhtml.Node) {
		if n.Type == xhtml.TextNode {
			sb.WriteString(n.Data)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return strings.Join(strings.Fields(sb.String()), " ")
}

func hasClass(n *xhtml.Node, c string) bool {
	for _, f := range strings.Fields(attr(n, "class")) {
		if f == c {
			return true
		}
	}
	return false
}

// describe reads the text and number of the element that carries an id.
func describe(n *xhtml.Node) target {
	var t target
	switch {
	case isHeading(n):
		t.text = text(n)
	case n.Data == "span" && n.FirstChild == nil && n.Parent != nil && n.Parent.Data == "p":
		// a (label)= target: the heading after its paragraph
		p := n.Parent
		for s := p.NextSibling; s != nil; s = s.NextSibling {
			if s.Type == xhtml.TextNode && strings.TrimSpace(s.Data) == "" {
				continue
			}
			if isHeading(s) {
				t.text = text(s)
			}
			break
		}
	}
	var find func(n *xhtml.Node) bool
	find = func(n *xhtml.Node) bool {
		if n.Type == xhtml.ElementNode && (hasClass(n, "zf-label") || hasClass(n, "zt-label") || hasClass(n, "zm-num")) {
			t.num = text(n)
			return true
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if find(c) {
				return true
			}
		}
		return false
	}
	find(n)
	return t
}

// ---- toctree -------------------------------------------------------------------

var knownOptions = map[string]bool{
	"arg": true, "title": true, "caption": true, "maxdepth": true, "hidden": true, "name": true, "class": true,
	"glob": true, "numbered": true, "titlesonly": true, "reversed": true, "includehidden": true,
}

var classRE = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_-]{0,40}$`)

func (plugin) RenderBlock(ctx plugins.Context, b plugins.Block) (string, error) {
	keys := make([]string, 0, len(b.Attrs))
	for k := range b.Attrs {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if !knownOptions[k] {
			warn(ctx, "option :"+k+": is not supported on toctree and was ignored")
		}
	}
	if v, ok := b.Attrs["hidden"]; ok && !strings.EqualFold(v, "false") {
		return "", nil // MyST keeps a hidden toctree out of the page
	}

	var md strings.Builder
	n, seen := 0, 0
	for _, line := range strings.Split(b.Body, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		seen++
		title, dest := split(line)
		if strings.ContainsAny(dest, "*?[") {
			warn(ctx, "toctree: the pattern "+clip(dest)+" is not supported (glob entries are not expanded)")
			continue
		}
		if external(dest) {
			md.WriteString("- [" + escapePunct(firstNonEmpty(title, dest)) + "](<" + dest + ">)\n")
			n++
			continue
		}
		if !safePath(ctx.Path, dest) {
			warn(ctx, "toctree: entry "+strconv.Itoa(seen)+" is not a page of this folder (paths are relative to the page and stay in the folder)")
			continue
		}
		shown := title
		if ctx.PageTitle != nil {
			t, ok := ctx.PageTitle(dest)
			if !ok {
				warn(ctx, "toctree: "+clip(dest)+" is not a page of this folder")
				continue
			}
			shown = firstNonEmpty(title, t)
		}
		shown = firstNonEmpty(shown, fileName(dest))
		md.WriteString("- [" + escapePunct(shown) + "](<" + strings.NewReplacer(`\`, `\\`, `&`, `\&`).Replace(dest) + ">)\n")
		n++
	}
	if n == 0 {
		warn(ctx, "toctree has no entries to list")
		return "", nil
	}
	list, err := ctx.Markdown(md.String())
	if err != nil {
		return "", err
	}

	classes := []string{"zen-block", "zx-toctree"}
	for _, c := range strings.Fields(b.Attrs["class"]) {
		if classRE.MatchString(c) && !strings.HasPrefix(c, "zen") && !strings.HasPrefix(c, "zx-") {
			classes = append(classes, c)
		} else {
			warn(ctx, "option :class: has a class name that is not supported and was ignored")
		}
	}
	idAttr := ""
	if v := b.Attrs["name"]; v != "" {
		if id := slug(v); id != "" {
			idAttr = ` id="` + id + `"`
		} else {
			warn(ctx, "option :name: has a value that is not supported and was ignored")
		}
	}
	caption := ""
	if c := strings.TrimSpace(b.Attrs["caption"]); c != "" {
		caption = `<p class="zx-caption">` + html.EscapeString(c) + `</p>`
	}
	return `<div class="` + strings.Join(classes, " ") + `"` + idAttr + `><div class="zen-block-body">` + caption + list + `</div></div>`, nil
}

func external(s string) bool {
	l := strings.ToLower(s)
	return (strings.HasPrefix(l, "https://") || strings.HasPrefix(l, "http://")) && !strings.ContainsAny(s, "<>\"'\\ \n\r\x00") && len(s) <= 2000
}
