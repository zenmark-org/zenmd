package layouts

import (
	"fmt"
	"html/template"
	"path"
	"sort"
	"strings"

	"github.com/zenmark-org/zenmd/engine/index"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// FolderPage is a page in the same folder as the page being laid out. Meta is
// its frontmatter and Definition the plain text of its first paragraph ("" when
// the text does not start with one).
type FolderPage struct {
	Title, Path string
	Meta        map[string]any
	Definition  string
}

// FolderSite is a Site that can also read the other pages of a page's
// folder, at render time (the term and glossary layouts use it). A Site
// without it sees no neighbours: the glossary is empty.
type FolderSite interface {
	// Folder returns the markdown pages beside path (not path itself).
	Folder(path string) []FolderPage
}

// termLayout is the layout id of the pages a glossary lists.
const termLayout = "term"

// maxDefinitionWords is the length a term's definition should not pass.
const maxDefinitionWords = 60

// Term is a page of the glossary folder.
type Term struct {
	Title, Path, Abbr string
	Aliases, See      []string
	Tags              []string
	Definition        string
	key               string // the title with case and accents ignored
}

// Letter is a button of the letter bar: A to Z, then #.
type Letter struct {
	Label, ID string
	Has       bool // a term starts with it
}

// Group is the terms under one letter.
type Group struct {
	Letter, ID string
	Terms      []Term
}

// Glossary is the terms of a folder, A to Z.
type Glossary struct {
	Terms   []Term
	Groups  []Group // only the letters that have terms
	Letters []Letter
	Count   int
}

// SeeLink is an entry of a term's see: list. A Missing one has no Path and
// Title is the text as written.
type SeeLink struct {
	Title, Path string
	Missing     bool
}

// TermPage is what a term page's template needs besides its fields: the terms
// before and after it in alphabetical order and its resolved see: links.
type TermPage struct {
	Prev, Next *Link
	See        []SeeLink
}

// Glossary reads the term pages of this page's folder: every page beside it
// whose layout is term. It reports duplicate titles, an alias that is another
// term's title, terms without a definition and see: links to missing terms as
// page warnings.
func (d *Data) Glossary() *Glossary {
	if d.gloss != nil {
		return d.gloss
	}
	terms := d.terms(false)
	g := buildGlossary(terms)
	d.gloss = g
	byPath := map[string]bool{}
	byKey := map[string][]Term{}
	for _, t := range terms {
		byPath[t.Path] = true
		byKey[t.key] = append(byKey[t.key], t)
	}
	for _, t := range terms {
		if t.Definition == "" {
			d.warnf("term %q has no definition (%s)", t.Title, t.Path)
		}
		for _, a := range t.Aliases {
			for _, o := range byKey[foldKey(a)] {
				if o.Path != t.Path {
					d.warnf("alias %q of %q is also the term %q", a, t.Title, o.Title)
				}
			}
		}
		for _, s := range t.See {
			if !byPath[seePath(d.Path, s)] {
				d.warnf("term %q: see: %q is not a term of this folder", t.Title, s)
			}
		}
	}
	keys := make([]string, 0, len(byKey))
	for k := range byKey {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if ts := byKey[k]; len(ts) > 1 {
			paths := make([]string, len(ts))
			for i, t := range ts {
				paths[i] = t.Path
			}
			d.warnf("duplicate term %q (%s)", ts[0].Title, strings.Join(paths, ", "))
		}
	}
	return g
}

// TermPage works out the neighbours and see: links of the term page being
// laid out, and warns about a missing or long definition and about a see:
// link to a page that is not a term. The page's own text and frontmatter are
// the ones being shown, which may be newer than the file.
func (d *Data) TermPage() *TermPage {
	if d.termPage != nil {
		return d.termPage
	}
	self := termFrom(d.Path, d.TermTitle(), d.frontmatter, definitionOfHTML(string(d.Content)))
	terms := append(d.terms(true), self)
	sortTerms(terms)
	tp := &TermPage{}
	byPath := map[string]Term{}
	for i, t := range terms {
		byPath[t.Path] = t
		if t.Path != d.Path {
			continue
		}
		if i > 0 {
			tp.Prev = &Link{Title: terms[i-1].Title, Path: terms[i-1].Path}
		}
		if i+1 < len(terms) {
			tp.Next = &Link{Title: terms[i+1].Title, Path: terms[i+1].Path}
		}
	}
	for _, s := range self.See {
		if t, ok := byPath[seePath(d.Path, s)]; ok && t.Path != d.Path {
			tp.See = append(tp.See, SeeLink{Title: t.Title, Path: t.Path})
			continue
		}
		tp.See = append(tp.See, SeeLink{Title: s, Missing: true})
		d.warnf("see: %q is not a term of this folder", s)
	}
	switch n := len(strings.Fields(self.Definition)); {
	case n == 0:
		d.warnf("term %q has no definition: start the text with a paragraph", self.Title)
	case n > maxDefinitionWords:
		d.warnf("the definition of %q has %d words; at most %d words are expected", self.Title, n, maxDefinitionWords)
	}
	d.termPage = tp
	return tp
}

// terms lists the term pages beside this one. self keeps the page itself out
// (it is then added from what is being shown).
func (d *Data) terms(self bool) []Term {
	fs, ok := d.site.(FolderSite)
	if !ok || d.layoutOf == nil {
		return nil
	}
	var out []Term
	for _, p := range fs.Folder(d.Path) {
		if p.Path == d.Path || d.layoutOf(p.Path, p.Meta) != termLayout {
			continue
		}
		out = append(out, termFrom(p.Path, p.Title, p.Meta, p.Definition))
	}
	sortTerms(out)
	return out
}

// termFrom makes the Term of a page. The title is the page's H1 (what the
// caller reads from the text); a title: in the frontmatter is used when there
// is none, and the file name when there is neither.
func termFrom(p, title string, meta map[string]any, def string) Term {
	if title = strings.TrimSpace(title); title == "" {
		if s, ok := meta["title"].(string); ok {
			title = strings.TrimSpace(s)
		}
	}
	if title == "" {
		title = strings.TrimSuffix(path.Base(p), path.Ext(p))
	}
	abbr, _ := meta["abbr"].(string)
	return Term{Title: title, Path: p, Abbr: strings.TrimSpace(abbr), Aliases: textList(meta["aliases"]),
		See: textList(meta["see"]), Tags: textList(meta["tags"]), Definition: def, key: foldKey(title)}
}

func textList(v any) []string {
	var out []string
	add := func(s string) {
		if s = strings.TrimSpace(s); s != "" {
			out = append(out, s)
		}
	}
	switch t := v.(type) {
	case string:
		add(t)
	case []string:
		for _, s := range t {
			add(s)
		}
	case []any:
		for _, e := range t {
			if s, ok := e.(string); ok {
				add(s)
			}
		}
	}
	return out
}

func sortTerms(ts []Term) {
	sort.SliceStable(ts, func(i, j int) bool {
		if ts[i].key != ts[j].key {
			return ts[i].key < ts[j].key
		}
		return ts[i].Path < ts[j].Path
	})
}

func buildGlossary(terms []Term) *Glossary {
	g := &Glossary{Terms: terms, Count: len(terms), Groups: []Group{}}
	has := map[string]bool{}
	for _, t := range terms {
		l := letterOf(t.Title)
		has[l] = true
		if n := len(g.Groups); n > 0 && g.Groups[n-1].Letter == l {
			g.Groups[n-1].Terms = append(g.Groups[n-1].Terms, t)
			continue
		}
		g.Groups = append(g.Groups, Group{Letter: l, ID: letterID(l), Terms: []Term{t}})
	}
	// "#" sorts before the letters in the bar and in the list, as digits and symbols do in the key order.
	for c := 'A'; c <= 'Z'; c++ {
		l := string(c)
		g.Letters = append(g.Letters, Letter{Label: l, ID: letterID(l), Has: has[l]})
	}
	g.Letters = append(g.Letters, Letter{Label: "#", ID: letterID("#"), Has: has["#"]})
	return g
}

func letterID(l string) string {
	if l == "#" {
		return "letter-other"
	}
	return "letter-" + strings.ToLower(l)
}

// letterOf is the letter a term is listed under: A to Z, or # for digits and
// symbols.
func letterOf(title string) string {
	k := foldKey(title)
	if k == "" || k[0] < 'a' || k[0] > 'z' {
		return "#"
	}
	return strings.ToUpper(k[:1])
}

// seePath resolves a see: entry, written relative to the term's folder.
func seePath(from, entry string) string {
	if i := strings.IndexAny(entry, "#?"); i >= 0 {
		entry = entry[:i]
	}
	return path.Join(path.Dir(from), entry)
}

// definitionOfHTML is the plain text of the first paragraph of a rendered
// page, when the page starts with one (a leading h1, which repeats the term,
// is passed over).
func definitionOfHTML(src string) string {
	nodes, err := html.ParseFragment(strings.NewReader(src), &html.Node{Type: html.ElementNode, Data: "div", DataAtom: atom.Div})
	if err != nil {
		return ""
	}
	skippedH1 := false
	for _, n := range nodes {
		switch {
		case n.Type == html.TextNode && strings.TrimSpace(n.Data) == "", n.Type == html.CommentNode:
			continue
		case n.Type == html.ElementNode && n.Data == "h1" && !skippedH1:
			skippedH1 = true
			continue
		case n.Type == html.ElementNode && n.Data == "p":
			return strings.Join(strings.Fields(nodeText(n)), " ")
		}
		return ""
	}
	return ""
}

func nodeText(n *html.Node) string {
	if n.Type == html.TextNode {
		return n.Data
	}
	var b strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		b.WriteString(nodeText(c))
	}
	return b.String()
}

// foldKey is a title with case and accents ignored (see index.Fold).
func foldKey(s string) string { return index.Fold(s) }

func (d *Data) warnf(format string, args ...any) {
	if d.warn != nil {
		d.warn(fmt.Sprintf(format, args...))
	}
}

// TermTitle is the term a page names: its H1 when the text starts with one,
// else the page's title (the title: of an older file, or its file name).
func (d *Data) TermTitle() string {
	if h := leadingH1(string(d.Content)); h != "" {
		return h
	}
	return d.Title
}

// TermBody is the page's text without its leading H1, which TermTitle shows
// as the big term.
func (d *Data) TermBody() template.HTML {
	c := string(d.Content)
	t := strings.TrimLeft(c, " \t\r\n")
	if !strings.HasPrefix(t, "<h1") {
		return d.Content
	}
	end := strings.Index(t, "</h1>")
	if end < 0 {
		return d.Content
	}
	return template.HTML(strings.TrimLeft(t[end+len("</h1>"):], "\r\n"))
}

// leadingH1 is the plain text of the h1 a rendered page starts with.
func leadingH1(src string) string {
	t := strings.TrimLeft(src, " \t\r\n")
	if !strings.HasPrefix(t, "<h1") {
		return ""
	}
	end := strings.Index(t, "</h1>")
	if end < 0 {
		return ""
	}
	nodes, err := html.ParseFragment(strings.NewReader(t[:end+len("</h1>")]), &html.Node{Type: html.ElementNode, Data: "div", DataAtom: atom.Div})
	if err != nil || len(nodes) == 0 {
		return ""
	}
	return strings.Join(strings.Fields(nodeText(nodes[0])), " ")
}

// Folder is the folder of the page being laid out, "" for the top level (the
// New Term button asks for it).
func (d *Data) Folder() string {
	if dir := path.Dir(d.Path); dir != "." {
		return dir
	}
	return ""
}
