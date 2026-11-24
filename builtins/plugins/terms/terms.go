// Package terms renders the glossary of MyST and Sphinx and the links to its
// terms:
//
//   - glossary: a definition list. A line that is not indented is a term (a
//     term may have a classifier, written "term : classifier"); the indented
//     lines under it are its definition, which is markdown. Each term gets the
//     id term-<slug>.
//   - term: a link to a term of the same page, written :term[API] or, with the
//     text to show, :term[the API <API>] (the MyST role form is {term}`API`).
//     A link to a term that is not on the page is a page warning.
//
// A term on another page is found through the xref built-in ({ref}, {doc}).
// It is rendered in Go and needs no script.
package terms

import (
	"embed"
	"fmt"
	"html"
	"io/fs"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/zenmark-org/zenmd/engine/plugins"
)

//go:embed assets/terms.css
var assets embed.FS

// maxTerm is the longest term or link text.
const maxTerm = 200

// knownOptions are read (the dialect adds title and arg itself).
var knownOptions = map[string]bool{"class": true, "name": true, "sorted": true, "title": true, "arg": true}

var (
	classRE = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_-]{0,63}$`)
	// targetRE splits :term[the API <API>] into its text and its target.
	targetRE = regexp.MustCompile(`^(.*\S)\s*<([^<>]+)>$`)
	// markerRE is the mark the term inline leaves for HTML to check and remove.
	markerRE = regexp.MustCompile(` data-zen-term="([^"]*)"`)
)

type plugin struct{}

// New returns the terms plugin.
func New() plugins.Plugin { return plugin{} }

func (plugin) Manifest() plugins.Manifest {
	return plugins.Manifest{
		ID:          "terms",
		Version:     "0.1.0",
		APIVersion:  plugins.APIVersion,
		Description: "Glossaries: the glossary directive and links to its terms",
		Default:     true,
		Blocks:      []string{"glossary"},
		Inlines:     []string{"term"},
		Allow:       []plugins.AllowRule{{Element: "dl"}, {Element: "dt"}, {Element: "dd"}},
		Help: plugins.Help{
			Summary: "A glossary is a definition list: each term on its own line, its definition indented under it (the MyST directive glossary). Every term is a link target, and term links to it from the same page, written :term[API] or :term[the API <API>] (the MyST role {term}). A link to a term that is not on the page is a warning. The sorted option sorts the terms.",
			Snippets: []plugins.Snippet{
				{Title: "A glossary", Markdown: ":::glossary\nAPI\n  Application programming interface.\n\nREST : protocol\n  A style of API built on HTTP.\n:::\n\nThe :term[API] of a :term[REST service <REST>].\n"},
				{Title: "The MyST form", Markdown: "```{glossary}\n:sorted:\n\nAPI\n  Application programming interface.\n\nREST\n  A style of API built on HTTP.\n```\n\nThe {term}`API` and {term}`REST`.\n"},
			},
		},
	}
}

func (plugin) Assets() fs.FS {
	sub, err := fs.Sub(assets, "assets")
	if err != nil {
		panic(err)
	}
	return sub
}

func (plugin) Entry() plugins.Assets { return plugins.Assets{CSS: []string{"terms.css"}} }

type entry struct {
	terms []string // one or more terms
	class string   // the classifier of the first term, if any
	def   string
}

func (plugin) RenderBlock(ctx plugins.Context, b plugins.Block) (string, error) {
	warn := func(msg string) {
		if ctx.Warn != nil {
			ctx.Warn(msg)
		}
	}
	if b.Name != "glossary" {
		return "", fmt.Errorf("terms does not own %q", b.Name)
	}
	var unknown []string
	for k := range b.Attrs {
		if !knownOptions[k] {
			unknown = append(unknown, k)
		}
	}
	sort.Strings(unknown)
	for _, k := range unknown {
		warn("option :" + k + ": is not supported and was ignored")
	}
	entries := parse(b.Body)
	if v, ok := b.Attrs["sorted"]; ok && v != "false" {
		sort.SliceStable(entries, func(i, j int) bool {
			return strings.ToLower(entries[i].terms[0]) < strings.ToLower(entries[j].terms[0])
		})
	}
	classes := []string{"zen-block", "zen-glossary"}
	for _, tok := range strings.Fields(b.Attrs["class"]) {
		if classRE.MatchString(tok) && !strings.HasPrefix(tok, "zen-") {
			classes = append(classes, tok)
		}
	}
	idAttr := ""
	if n := b.Attrs["name"]; n != "" {
		if id := nameID(n); id != "" {
			idAttr = ` id="` + id + `"`
		} else {
			warn("option :name: has a value that is not supported and was ignored")
		}
	}
	seen := map[string]int{}
	var sb strings.Builder
	sb.WriteString(`<div` + idAttr + ` class="` + strings.Join(classes, " ") + `"><div class="zen-block-body"><dl>`)
	for _, e := range entries {
		for i, term := range e.terms {
			s := slug(term)
			id := ""
			if s == "" {
				warn("a term has no text that can be linked and has no link target")
			} else {
				seen[s]++
				if seen[s] > 1 {
					warn("a term is defined twice; the second one has its own link target")
					s += "-" + strconv.Itoa(seen[s])
				}
				id = ` id="term-` + s + `"`
			}
			sb.WriteString("<dt" + id + ">" + html.EscapeString(term))
			if i == 0 && e.class != "" {
				sb.WriteString(`<span class="zen-glossary-class">` + html.EscapeString(e.class) + `</span>`)
			}
			sb.WriteString("</dt>")
		}
		sb.WriteString("<dd>")
		if strings.TrimSpace(e.def) != "" {
			if ctx.Markdown != nil {
				h, err := ctx.Markdown(e.def)
				if err != nil {
					return "", err
				}
				sb.WriteString(h)
			} else {
				sb.WriteString("<p>" + html.EscapeString(strings.TrimSpace(e.def)) + "</p>")
			}
		}
		sb.WriteString("</dd>")
	}
	sb.WriteString("</dl></div></div>")
	return sb.String(), nil
}

// parse reads the glossary: unindented lines are terms, indented lines are the
// definition of the terms above them.
func parse(body string) []entry {
	var out []entry
	var cur *entry
	var def []string
	flush := func() {
		if cur != nil {
			cur.def = dedent(def)
			out = append(out, *cur)
		}
		cur, def = nil, nil
	}
	for _, line := range strings.Split(strings.ReplaceAll(body, "\r\n", "\n"), "\n") {
		line = strings.TrimRight(line, " \t")
		switch {
		case line == "":
			if cur != nil && len(def) > 0 {
				def = append(def, "")
			}
		case line[0] == ' ' || line[0] == '\t':
			if cur != nil {
				def = append(def, line)
			}
		default:
			if cur != nil && strings.TrimSpace(strings.Join(def, "")) != "" {
				flush()
			}
			term, class := line, ""
			if i := strings.Index(line, " : "); i > 0 {
				term, class = strings.TrimSpace(line[:i]), strings.TrimSpace(line[i+3:])
			}
			if utf8.RuneCountInString(term) > maxTerm {
				term = string([]rune(term)[:maxTerm])
			}
			if cur == nil {
				cur = &entry{class: class}
			}
			cur.terms = append(cur.terms, term)
			if len(cur.terms) == 1 {
				cur.class = class
			}
		}
	}
	flush()
	return out
}

// dedent removes the indent the definition lines share.
func dedent(lines []string) string {
	min := -1
	for _, l := range lines {
		if strings.TrimSpace(l) == "" {
			continue
		}
		n := len(l) - len(strings.TrimLeft(l, " \t"))
		if min < 0 || n < min {
			min = n
		}
	}
	var sb strings.Builder
	for _, l := range lines {
		if len(l) >= min && min > 0 {
			l = l[min:]
		} else if strings.TrimSpace(l) == "" {
			l = ""
		}
		sb.WriteString(l + "\n")
	}
	return strings.TrimSpace(sb.String())
}

func (plugin) RenderInline(ctx plugins.Context, in plugins.Inline) (string, error) {
	if in.Name != "term" {
		return "", fmt.Errorf("terms does not own %q", in.Name)
	}
	text := strings.TrimSpace(in.Text)
	target := text
	if m := targetRE.FindStringSubmatch(text); m != nil {
		text, target = strings.TrimSpace(m[1]), strings.TrimSpace(m[2])
	}
	if utf8.RuneCountInString(text) > maxTerm || utf8.RuneCountInString(target) > maxTerm {
		return html.EscapeString(text), nil
	}
	s := slug(target)
	if text == "" || s == "" {
		if ctx.Warn != nil {
			ctx.Warn("a term link has no text that can be linked")
		}
		return html.EscapeString(text), nil
	}
	return `<a href="#term-` + s + `" class="zen-term" data-zen-term="` + html.EscapeString(target) + `">` + html.EscapeString(text) + "</a>", nil
}

// HTML checks the term links of the finished page: each must point at a term
// of a glossary on the page. It then removes the marks.
func (plugin) HTML(ctx plugins.Context, s string) (string, error) {
	if !strings.Contains(s, ` data-zen-term="`) {
		return s, nil
	}
	reported := map[string]bool{}
	for _, m := range markerRE.FindAllStringSubmatch(s, -1) {
		target := html.UnescapeString(m[1])
		slug := slug(target)
		if reported[slug] {
			continue
		}
		reported[slug] = true
		if !strings.Contains(s, ` id="term-`+slug+`"`) && ctx.Warn != nil {
			if r := []rune(target); len(r) > 40 {
				target = string(r[:40]) + "…"
			}
			ctx.Warn(fmt.Sprintf("the term %q is not defined in a glossary on this page", target))
		}
	}
	return markerRE.ReplaceAllString(s, ""), nil
}

// slug turns a term into the text of its id: lower case, ASCII letters and
// digits, a dash for anything else; another letter or digit becomes u and its
// four hex digits, so every term has an id the sanitiser keeps.
func slug(s string) string {
	var sb strings.Builder
	dash := false
	for _, r := range strings.ToLower(strings.TrimSpace(s)) {
		switch {
		case r < 128 && (r >= 'a' && r <= 'z' || r >= '0' && r <= '9'):
			sb.WriteRune(r)
			dash = false
		case r >= 128 && (unicode.IsLetter(r) || unicode.IsDigit(r)):
			fmt.Fprintf(&sb, "u%04x", r)
			dash = false
		case !dash && sb.Len() > 0:
			sb.WriteByte('-')
			dash = true
		}
		if sb.Len() >= 80 {
			break
		}
	}
	return strings.Trim(sb.String(), "-")
}

// nameID turns a :name: into an id: lower case, letters, digits and . _ : -
// (the same as the callout and figure built-ins).
func nameID(s string) string {
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
