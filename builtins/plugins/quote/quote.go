// Package quote renders the quote directives of MyST and Sphinx (epigraph,
// pull-quote and highlights) and the same blocks written as :::epigraph,
// :::pull-quote and :::highlights. Each is a blockquote in the plugin kit's
// frame; a last line that starts with -- (or ---, or a dash) is the
// attribution and is shown under the quote. It is rendered in Go and needs no
// script.
package quote

import (
	"embed"
	"fmt"
	"html"
	"io/fs"
	"regexp"
	"sort"
	"strings"

	"github.com/zenmark-org/zenmd/engine/plugins"
)

//go:embed assets/quote.css
var assets embed.FS

var blockNames = []string{"epigraph", "pull-quote", "highlights"}

// knownOptions are read (the dialect adds title and arg itself; neither is used).
var knownOptions = map[string]bool{"class": true, "name": true, "title": true, "arg": true}

var (
	classRE = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_-]{0,63}$`)
	// attribRE is the attribution line: -- Name, --- Name, or a dash.
	attribRE = regexp.MustCompile(`^(?:---?|—|–)[ \t]+(\S.*)$`)
)

type plugin struct{}

// New returns the quote plugin.
func New() plugins.Plugin { return plugin{} }

func (plugin) Manifest() plugins.Manifest {
	return plugins.Manifest{
		ID:          "quote",
		Version:     "0.1.0",
		APIVersion:  plugins.APIVersion,
		Description: "Quotes: epigraph, pull-quote and highlights",
		Default:     true,
		Blocks:      blockNames,
		Allow:       []plugins.AllowRule{{Element: "blockquote"}},
		Help: plugins.Help{
			Summary: "A quote set apart from the text. epigraph is a short quote that opens a page or a section, pull-quote repeats a line from the text in larger type, and highlights lists the key points. A last line that starts with -- is the attribution. They read the same in markdown written for MyST, Jupyter Book or Sphinx.",
			Snippets: []plugins.Snippet{
				{Title: "An epigraph", Markdown: ":::epigraph\nSimplicity is prerequisite for reliability.\n\n-- Edsger W. Dijkstra\n:::\n"},
				{Title: "The MyST form", Markdown: "```{epigraph}\nSimplicity is prerequisite for reliability.\n\n-- Edsger W. Dijkstra\n```\n\n```{pull-quote}\nA line worth repeating.\n```\n"},
				{Title: "Highlights", Markdown: ":::highlights\n- The first key point\n- The second key point\n:::\n"},
			},
		},
	}
}

func (plugin) RenderBlock(ctx plugins.Context, b plugins.Block) (string, error) {
	warn := func(msg string) {
		if ctx.Warn != nil {
			ctx.Warn(msg)
		}
	}
	owned := false
	for _, n := range blockNames {
		owned = owned || n == b.Name
	}
	if !owned {
		return "", fmt.Errorf("quote does not own %q", b.Name)
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
	classes := []string{"zen-block", "zen-quote", "zen-quote-" + b.Name}
	for _, tok := range strings.Fields(b.Attrs["class"]) {
		if classRE.MatchString(tok) && !strings.HasPrefix(tok, "zen-") {
			classes = append(classes, tok)
		}
	}
	idAttr := ""
	if n := b.Attrs["name"]; n != "" {
		if id := slug(n); id != "" {
			idAttr = ` id="` + id + `"`
		} else {
			warn("option :name: has a value that is not supported and was ignored")
		}
	}

	quote, by := split(b.Body)
	md := func(src string) (string, error) {
		if ctx.Markdown != nil {
			return ctx.Markdown(src)
		}
		return "<p>" + html.EscapeString(strings.TrimSpace(src)) + "</p>", nil
	}
	body := ""
	if strings.TrimSpace(quote) != "" {
		h, err := md(quote)
		if err != nil {
			return "", err
		}
		body = h
	}
	var sb strings.Builder
	sb.WriteString(`<div` + idAttr + ` class="` + strings.Join(classes, " ") + `"><div class="zen-block-body">`)
	sb.WriteString("<blockquote>" + body + "</blockquote>")
	if by != "" {
		h, err := md(by)
		if err != nil {
			return "", err
		}
		sb.WriteString(`<div class="zen-quote-by">` + inline(h, by) + `</div>`)
	}
	sb.WriteString("</div></div>")
	return sb.String(), nil
}

// split separates the quote from its attribution: the last line of the body
// when it starts with -- (or --- or a dash).
func split(body string) (quote, by string) {
	body = strings.TrimRight(strings.ReplaceAll(body, "\r\n", "\n"), " \t\n")
	i := strings.LastIndex(body, "\n")
	last := strings.TrimSpace(body[i+1:])
	if m := attribRE.FindStringSubmatch(last); m != nil {
		if i < 0 {
			return "", strings.TrimSpace(m[1])
		}
		return strings.TrimRight(body[:i], " \t\n"), strings.TrimSpace(m[1])
	}
	return body, ""
}

// inline turns the page's rendering of a one-line text into inline HTML: a
// single paragraph loses its <p>; anything else is shown as plain text.
func inline(h, src string) string {
	h = strings.TrimSpace(h)
	if strings.HasPrefix(h, "<p>") && strings.HasSuffix(h, "</p>") && strings.Count(h, "<p>") == 1 {
		return strings.TrimSuffix(strings.TrimPrefix(h, "<p>"), "</p>")
	}
	return html.EscapeString(src)
}

func (plugin) Assets() fs.FS {
	sub, err := fs.Sub(assets, "assets")
	if err != nil {
		panic(err)
	}
	return sub
}

func (plugin) Entry() plugins.Assets { return plugins.Assets{CSS: []string{"quote.css"}} }

// slug turns a :name: into an id: lower case, letters, digits and . _ : -
// (the same as the callout and figure built-ins).
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
