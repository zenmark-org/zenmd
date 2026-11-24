// Package tabs renders the tab sets of sphinx-design as written in MyST,
// Jupyter Book and Sphinx ({tab-set} holding {tab-item}s) and the same blocks
// written as :::tab-set and :::tab-item containers.
//
// Go renders every item as a title over its panel, in order, so a page without
// scripts reads well. assets/tabs.js then turns a set into ARIA tabs (arrow
// keys, Home and End), keeps tabs that share a :sync: key in step across the
// page, and remembers the last synced choice per folder (ctx.storage). The
// look is the plugin kit's frame (docs/wiki/PLUGIN-UI.md); tabs.css holds only
// the tab row.
package tabs

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

//go:embed assets/tabs.css assets/tabs.js
var assets embed.FS

// knownOptions are the options each block reads (the dialect adds title and arg itself).
var knownOptions = map[string]map[string]bool{
	"tab-set":  {"class": true, "name": true, "sync-group": true, "title": true, "arg": true, "id": true},
	"tab-item": {"class": true, "name": true, "sync": true, "selected": true, "title": true, "arg": true, "id": true},
}

var classRE = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_-]{0,63}$`)

type plugin struct{}

// New returns the tabs plugin.
func New() plugins.Plugin { return plugin{} }

func (plugin) Manifest() plugins.Manifest {
	return plugins.Manifest{
		ID:          "tabs",
		Version:     "0.1.0",
		APIVersion:  plugins.APIVersion,
		Description: "Tabs: tab-set and tab-item",
		Default:     true,
		Blocks:      []string{"tab-set", "tab-item"},
		Help: plugins.Help{
			Summary: "Tabs show one panel at a time. A tab-set holds tab-items, each with a title (the MyST names are tab-set and tab-item, from sphinx-design). Use the arrow keys, Home and End to move between tabs. Give tab-items the same sync key and they change together across the page; the last synced choice is kept for the folder. Without scripts every panel shows in order under its title. Add selected to a tab-item to open it first.",
			Snippets: []plugins.Snippet{
				{Title: "Tabs", Markdown: ":::tab-set\n:::tab-item{title=\"macOS\"}\nInstall with Homebrew.\n:::\n:::tab-item{title=\"Linux\"}\nInstall with your package manager.\n:::\n:::\n"},
				{Title: "The MyST form", Markdown: "::::{tab-set}\n:::{tab-item} macOS\nInstall with Homebrew.\n:::\n:::{tab-item} Linux\nInstall with your package manager.\n:::\n::::\n"},
				{Title: "Tabs that change together", Markdown: ":::tab-set\n:::tab-item{title=Python sync=py}\nprint(1)\n:::\n:::tab-item{title=Go sync=go}\nfmt.Println(1)\n:::\n:::\n\n:::tab-set\n:::tab-item{title=Python sync=py}\nInstall with pip.\n:::\n:::tab-item{title=Go sync=go}\nInstall with go get.\n:::\n:::\n"},
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

func (plugin) Entry() plugins.Assets {
	return plugins.Assets{CSS: []string{"tabs.css"}, JS: []string{"tabs.js"}}
}

func (plugin) RenderBlock(ctx plugins.Context, b plugins.Block) (string, error) {
	known, ok := knownOptions[b.Name]
	if !ok {
		return "", fmt.Errorf("tabs does not own %q", b.Name)
	}
	warn := func(msg string) {
		if ctx.Warn != nil {
			ctx.Warn(msg)
		}
	}
	var unknown []string
	for k := range b.Attrs {
		if !known[k] {
			unknown = append(unknown, k)
		}
	}
	sort.Strings(unknown)
	for _, k := range unknown {
		warn("option :" + k + ": is not supported and was ignored")
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

	classes := []string{"zen-block"}
	idAttr := ""
	if n := b.Attrs["name"]; n != "" {
		if id := slug(n); id != "" {
			idAttr = ` id="` + id + `"`
		} else {
			warn("option :name: has a value that is not supported and was ignored")
		}
	}
	var user []string
	for _, tok := range strings.Fields(b.Attrs["class"]) {
		if classRE.MatchString(tok) && !strings.HasPrefix(tok, "zen-") {
			user = append(user, tok)
		}
	}

	if b.Name == "tab-set" {
		classes = append(classes, "zen-tabs")
		classes = append(classes, user...)
		return `<div` + idAttr + ` class="` + strings.Join(classes, " ") + `"><div class="zen-tabs-body">` + body + `</div></div>`, nil
	}

	title := strings.TrimSpace(b.Attrs["title"])
	if title == "" {
		title = strings.TrimSpace(b.Attrs["arg"])
	}
	if title == "" {
		title = "Tab"
		warn("tab-item needs a title; it is shown as Tab")
	}
	itemClasses := []string{"zen-tab-item"}
	if v, ok := b.Attrs["selected"]; ok && v != "false" {
		itemClasses = append(itemClasses, "zen-tab-selected")
	}
	itemClasses = append(itemClasses, user...)
	return `<div` + idAttr + ` class="` + strings.Join(itemClasses, " ") + `">` +
		`<div class="zen-block-head zen-tab-title">` + html.EscapeString(title) + `</div>` +
		`<div class="zen-block-body zen-tab-panel">` + body + `</div></div>`, nil
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
