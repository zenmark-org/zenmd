// Package aside renders the side content directives of MyST and Sphinx
// (sidebar, margin and topic) and the same blocks written as :::sidebar,
// :::margin and :::topic. Each is an aside in the plugin kit's frame with an
// optional title, floated to the right of the text (sidebar and margin) when the page is wide
// enough and full width when it is narrow; a topic stays in the flow, full width. It is rendered in Go and needs no
// script.
package aside

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

//go:embed assets/aside.css
var assets embed.FS

var blockNames = []string{"sidebar", "margin", "topic"}

// knownOptions are read (the dialect adds title and arg itself).
var knownOptions = map[string]bool{"class": true, "name": true, "title": true, "arg": true, "subtitle": true}

var classRE = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_-]{0,63}$`)

type plugin struct{}

// New returns the aside plugin.
func New() plugins.Plugin { return plugin{} }

func (plugin) Manifest() plugins.Manifest {
	return plugins.Manifest{
		ID:          "aside",
		Version:     "0.1.0",
		APIVersion:  plugins.APIVersion,
		Description: "Side content: sidebar, margin and topic",
		Default:     true,
		Blocks:      blockNames,
		Allow:       []plugins.AllowRule{{Element: "aside"}},
		Help: plugins.Help{
			Summary: "Content beside the text. sidebar and topic take a title, margin has none; sidebar and margin float to the right when the page is wide enough and fill the width when it is narrow; topic stays in the flow of the text, full width. sidebar also takes a subtitle. They read the same in markdown written for MyST, Jupyter Book or Sphinx.",
			Snippets: []plugins.Snippet{
				{Title: "A sidebar", Markdown: ":::sidebar{title=\"Did you know?\"}\nThe body is markdown.\n:::\n"},
				{Title: "The MyST form", Markdown: "```{sidebar} Did you know?\n:subtitle: A side note\n\nThe body is markdown.\n```\n\n```{margin}\nA short note in the margin.\n```\n"},
				{Title: "A topic", Markdown: ":::topic{title=Background}\nA self-contained passage about one subject.\n:::\n\n```{topic} Background\nThe MyST form.\n```\n"},
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
		return "", fmt.Errorf("aside does not own %q", b.Name)
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
	classes := []string{"zen-block", "zen-aside", "zen-aside-" + b.Name}
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
	}
	var sb strings.Builder
	sb.WriteString(`<div` + idAttr + ` class="` + strings.Join(classes, " ") + `"><aside>`)
	if title != "" {
		sb.WriteString(`<div class="zen-block-head">` + html.EscapeString(title) + `</div>`)
	}
	if sub := strings.TrimSpace(b.Attrs["subtitle"]); sub != "" {
		sb.WriteString(`<div class="zen-block-status zen-aside-subtitle">` + html.EscapeString(sub) + `</div>`)
	}
	if body != "" {
		sb.WriteString(`<div class="zen-block-body">` + body + `</div>`)
	}
	sb.WriteString("</aside></div>")
	return sb.String(), nil
}

func (plugin) Assets() fs.FS {
	sub, err := fs.Sub(assets, "assets")
	if err != nil {
		panic(err)
	}
	return sub
}

func (plugin) Entry() plugins.Assets { return plugins.Assets{CSS: []string{"aside.css"}} }

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
