// Package callout renders the admonitions of MyST, Jupyter Book and Sphinx
// (note, tip, warning, admonition, versionadded ...) and the same blocks
// written as :::note containers. It is rendered in Go and needs no script: a
// collapsed callout is a plain details element, which the browser opens by
// click or Enter. The look is the plugin kit's frame (docs/wiki/PLUGIN-UI.md)
// with a left rule in one of the lint tokens, from assets/callout.css.
package callout

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

//go:embed assets/callout.css
var assets embed.FS

// Colour groups: the lint token that paints the left rule.
const (
	suggestion = "suggestion"
	warning    = "warning"
	errorGroup = "error"
)

type kind struct{ title, icon, group string }

// kinds maps each directive name to its default title, icon and colour.
var kinds = map[string]kind{
	"note":      {"Note", "N", suggestion},
	"tip":       {"Tip", "T", suggestion},
	"hint":      {"Hint", "H", suggestion},
	"important": {"Important", "!", suggestion},
	"seealso":   {"See also", "S", suggestion},
	"warning":   {"Warning", "W", warning},
	"caution":   {"Caution", "C", warning},
	"attention": {"Attention", "A", warning},
	"danger":    {"Danger", "D", errorGroup},
	"error":     {"Error", "E", errorGroup},
}

// versionKinds are the one-line version notes: the text before the version.
var versionKinds = map[string]string{
	"versionadded":   "New in version",
	"versionchanged": "Changed in version",
	"deprecated":     "Deprecated since version",
}

var blockNames = []string{"note", "tip", "hint", "important", "seealso", "warning", "caution", "attention",
	"danger", "error", "admonition", "versionadded", "versionchanged", "deprecated"}

// knownOptions are read (the dialect adds title and arg itself).
var knownOptions = map[string]bool{"class": true, "name": true, "open": true, "title": true, "arg": true, "id": true, "version": true}

var classRE = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_-]{0,63}$`)

type plugin struct{}

// New returns the callout plugin.
func New() plugins.Plugin { return plugin{} }

func (plugin) Manifest() plugins.Manifest {
	return plugins.Manifest{
		ID:          "callout",
		Version:     "0.1.0",
		APIVersion:  plugins.APIVersion,
		Description: "Callouts: note, tip, warning, admonition and version notes",
		Default:     true,
		Blocks:      blockNames,
		Allow: []plugins.AllowRule{
			{Element: "details", Attrs: []string{"open"}},
			{Element: "span", Attrs: []string{"aria-hidden"}},
		},
		Help: plugins.Help{
			Summary: "A callout sets a note apart from the text: note, tip, hint, important, seealso, warning, caution, attention, danger and error each have a title and a coloured left rule, and admonition takes any title. versionadded, versionchanged and deprecated are one-line version notes. Add class dropdown to fold a callout, and open to start it unfolded. They read the same in markdown written for MyST, Jupyter Book or Sphinx.",
			Snippets: []plugins.Snippet{
				{Title: "A callout", Markdown: ":::note\nThe body is markdown.\n:::\n\n:::warning{title=\"Heads up\"}\nA title replaces the default one.\n:::\n"},
				{Title: "The MyST form", Markdown: "```{note} Remember\nThe text after the name is the title.\n```\n\n```{admonition} My own title\nAny title you like.\n```\n"},
				{Title: "Fold it", Markdown: ":::tip{class=dropdown open}\nA dropdown opens by click or Enter.\n:::\n\n```{tip} Closed at first\n:class: dropdown\nHidden until opened.\n```\n"},
				{Title: "Version notes", Markdown: "```{versionadded} 1.2\nThe --fast flag.\n```\n\n:::deprecated{version=2.0}\nUse --quick instead.\n:::\n"},
			},
		},
	}
}

func (plugin) RenderBlock(ctx plugins.Context, b plugins.Block) (string, error) {
	var warns []string
	group, icon, title := suggestion, "N", ""
	extra := []string{}
	dropdown := false

	// the class option: dropdown folds, a kind name picks an admonition's colour, the rest are kept
	var userClass []string
	for _, tok := range strings.Fields(b.Attrs["class"]) {
		switch {
		case tok == "dropdown":
			dropdown = true
		case b.Name == "admonition" && kinds[tok].title != "":
			group, icon = kinds[tok].group, kinds[tok].icon
		case classRE.MatchString(tok) && !strings.HasPrefix(tok, "zen-"):
			userClass = append(userClass, tok)
		}
	}

	arg := strings.TrimSpace(b.Attrs["title"])
	if arg == "" {
		arg = strings.TrimSpace(b.Attrs["arg"])
	}
	switch {
	case b.Name == "admonition":
		title = arg
		if title == "" {
			title = "Note"
			warns = append(warns, "the admonition needs a title, so it shows as Note")
		}
	case versionKinds[b.Name] != "":
		v := strings.TrimSpace(b.Attrs["version"])
		if v == "" {
			v = arg
		}
		title = versionKinds[b.Name]
		if v == "" {
			warns = append(warns, "the "+b.Name+" note needs a version")
		} else {
			title += " " + v
		}
		extra = append(extra, "zen-callout-version")
	default:
		k, ok := kinds[b.Name]
		if !ok {
			return "", fmt.Errorf("callout does not own %q", b.Name)
		}
		group, icon, title = k.group, k.icon, k.title
		if arg != "" {
			title = arg
		}
	}
	if v := versionKinds[b.Name]; v != "" {
		icon = "V"
	}

	var unknown []string
	for k := range b.Attrs {
		if !knownOptions[k] {
			unknown = append(unknown, k)
		}
	}
	sort.Strings(unknown)
	for _, k := range unknown {
		warns = append(warns, "the option :"+k+": is not supported and was ignored")
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
	}
	idAttr := ""
	if n := b.Attrs["name"]; n != "" {
		if id := slug(n); id != "" {
			idAttr = ` id="` + id + `"`
		} else {
			warns = append(warns, "the option :name: has a value that is not supported")
		}
	}
	for _, w := range warns {
		body = `<div class="zen-block-status">` + html.EscapeString(w) + `</div>` + body
	}
	if body != "" {
		body = `<div class="zen-block-body">` + body + `</div>`
	}

	classes := append([]string{"zen-block", "zen-callout", "zen-callout-" + group}, extra...)
	if dropdown {
		classes = append(classes, "zen-callout-dropdown")
	}
	classes = append(classes, userClass...)
	head := `<span class="zen-callout-icon" aria-hidden="true">` + html.EscapeString(icon) + `</span>` +
		`<span class="zen-callout-title">` + html.EscapeString(title) + `</span>`
	var sb strings.Builder
	sb.WriteString(`<div` + idAttr + ` class="` + strings.Join(classes, " ") + `">`)
	if dropdown {
		sb.WriteString("<details")
		if open {
			sb.WriteString(" open")
		}
		sb.WriteString(`><summary><span class="zen-block-head zen-callout-head">` + head + `</span></summary>` + body + `</details>`)
	} else {
		sb.WriteString(`<div class="zen-block-head zen-callout-head">` + head + `</div>` + body)
	}
	sb.WriteString(`</div>`)
	return sb.String(), nil
}

func (plugin) Assets() fs.FS {
	sub, err := fs.Sub(assets, "assets")
	if err != nil {
		panic(err)
	}
	return sub
}

func (plugin) Entry() plugins.Assets { return plugins.Assets{CSS: []string{"callout.css"}} }

// slug turns a :name: into an id: lower case, letters, digits and . _ : -
// (the same as the figure built-in).
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
