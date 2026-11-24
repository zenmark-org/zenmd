// Package dropdown renders the dropdown of sphinx-design as written in MyST,
// Jupyter Book and Sphinx ({dropdown} Title) and the same block written as a
// :::dropdown container. It is a plain details element in the plugin kit's
// frame (docs/wiki/PLUGIN-UI.md), so it opens by click or Enter and Space with
// no script, and a page without scripts reads the same. :open: starts it
// unfolded. :color: and :icon: warn and are ignored: every block is black and
// white.
package dropdown

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

//go:embed assets/dropdown.css
var assets embed.FS

// knownOptions are read (the dialect adds title and arg itself).
var knownOptions = map[string]bool{"class": true, "name": true, "open": true, "title": true, "arg": true, "id": true}

var classRE = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_-]{0,63}$`)

type plugin struct{}

// New returns the dropdown plugin.
func New() plugins.Plugin { return plugin{} }

func (plugin) Manifest() plugins.Manifest {
	return plugins.Manifest{
		ID:          "dropdown",
		Version:     "0.1.0",
		APIVersion:  plugins.APIVersion,
		Description: "Dropdowns: a title that opens and closes its text",
		Default:     true,
		Blocks:      []string{"dropdown"},
		Allow:       []plugins.AllowRule{{Element: "details", Attrs: []string{"open"}}},
		Help: plugins.Help{
			Summary: "A dropdown is a title that opens and closes its text, by click or Enter (the MyST name is dropdown, from sphinx-design). The body is markdown. Add open to start it unfolded. The colour and icon options of Sphinx are accepted and ignored, because every block is black and white.",
			Snippets: []plugins.Snippet{
				{Title: "A dropdown", Markdown: ":::dropdown{title=\"More detail\"}\nThe body is markdown.\n:::\n\n:::dropdown{title=\"Open at first\" open}\nShown unfolded.\n:::\n"},
				{Title: "The MyST form", Markdown: "```{dropdown} More detail\nThe text after the name is the title.\n```\n\n:::{dropdown} Open at first\n:open:\nShown unfolded.\n:::\n"},
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

func (plugin) Entry() plugins.Assets { return plugins.Assets{CSS: []string{"dropdown.css"}} }

func (plugin) RenderBlock(ctx plugins.Context, b plugins.Block) (string, error) {
	if b.Name != "dropdown" {
		return "", fmt.Errorf("dropdown does not own %q", b.Name)
	}
	warn := func(msg string) {
		if ctx.Warn != nil {
			ctx.Warn(msg)
		}
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

	title := strings.TrimSpace(b.Attrs["title"])
	if title == "" {
		title = strings.TrimSpace(b.Attrs["arg"])
	}
	if title == "" {
		title = "Details"
	}
	open := false
	if v, ok := b.Attrs["open"]; ok && v != "false" {
		open = true
	}
	body := ""
	if strings.TrimSpace(b.Body) != "" {
		if ctx.Markdown != nil {
			h, err := ctx.Markdown(b.Body)
			if err != nil {
				return "", err
			}
			body = h
		} else {
			body = "<p>" + html.EscapeString(strings.TrimSpace(b.Body)) + "</p>"
		}
		body = `<div class="zen-block-body">` + body + `</div>`
	}
	idAttr := ""
	if n := b.Attrs["name"]; n != "" {
		if id := slug(n); id != "" {
			idAttr = ` id="` + id + `"`
		} else {
			warn("option :name: has a value that is not supported and was ignored")
		}
	}
	classes := []string{"zen-block", "zen-dropdown"}
	for _, tok := range strings.Fields(b.Attrs["class"]) {
		if classRE.MatchString(tok) && !strings.HasPrefix(tok, "zen-") {
			classes = append(classes, tok)
		}
	}
	var sb strings.Builder
	sb.WriteString(`<div` + idAttr + ` class="` + strings.Join(classes, " ") + `"><details`)
	if open {
		sb.WriteString(" open")
	}
	sb.WriteString(`><summary><span class="zen-block-head zen-dropdown-head"><span class="zen-dropdown-title">` + html.EscapeString(title) + `</span></span></summary>` + body + `</details></div>`)
	return sb.String(), nil
}

// slug turns a :name: into an id: lower case, letters, digits and . _ : -
// (the same as the callout built-in).
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
