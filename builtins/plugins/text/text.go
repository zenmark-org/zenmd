// Package text renders the small inline marks of MyST and Sphinx, and the same
// marks written as :kbd[…], :abbr[…], :sub[…] and :sup[…] inlines:
//
//   - kbd: a key or a key combination, one <kbd> per key (split on + and -).
//   - abbr: <abbr title>, written HTML (HyperText Markup Language) or with a title option.
//   - sub and sup: subscript and superscript.
//
// The MyST role form ({kbd}`Ctrl+C`) comes from the dialect's role mapping,
// which hands the role's text to the inline hook of the plugin that owns the
// name. Everything is plain HTML that needs no script.
package text

import (
	"embed"
	"html"
	"io/fs"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/zenmark-org/zenmd/engine/plugins"
)

//go:embed assets/text.css
var assets embed.FS

// maxText is the longest text of one mark; more is shown as plain text.
const maxText = 200

type plugin struct{}

// New returns the text plugin.
func New() plugins.Plugin { return plugin{} }

func (plugin) Manifest() plugins.Manifest {
	return plugins.Manifest{
		ID:          "text",
		Version:     "0.1.0",
		APIVersion:  plugins.APIVersion,
		Description: "Small text marks: kbd, abbr, sub and sup",
		Default:     true,
		Inlines:     []string{"kbd", "abbr", "sub", "sup"},
		Allow: []plugins.AllowRule{
			{Element: "kbd"}, {Element: "sub"}, {Element: "sup"}, {Element: "abbr", Attrs: []string{"title"}},
		},
		Help: plugins.Help{
			Summary: "Small marks inside a line. kbd shows a key or a key combination, one key cap per key, split on + and - (the MyST role {kbd}). abbr shows an abbreviation with its explanation as a tooltip, written HTML (HyperText Markup Language) (the role {abbr}). sub and sup are subscript and superscript (the roles {sub} and {sup}).",
			Snippets: []plugins.Snippet{
				{Title: "Keys", Markdown: "Press :kbd[Ctrl+C] to copy, or :kbd[⌘+K] on a Mac.\n"},
				{Title: "An abbreviation", Markdown: "Pages are written in :abbr[HTML (HyperText Markup Language)].\n\nThe title can also be an option: :abbr[CSS]{title=\"Cascading Style Sheets\"}.\n"},
				{Title: "Subscript and superscript", Markdown: "Water is H :sub[2]O and E = mc :sup[2].\n"},
				{Title: "The MyST role form", Markdown: "Press {kbd}`Ctrl+C`. Written in {abbr}`HTML (HyperText Markup Language)`.\n\nH{sub}`2`O and E = mc{sup}`2`.\n"},
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

func (plugin) Entry() plugins.Assets { return plugins.Assets{CSS: []string{"text.css"}} }

// known are the options read: title for abbr (the rest is the text).
var known = map[string]map[string]bool{
	"kbd": {}, "sub": {}, "sup": {}, "abbr": {"title": true},
}

var abbrRE = regexp.MustCompile(`^(.*\S)\s*\(([^()]+)\)$`)

func (plugin) RenderInline(ctx plugins.Context, in plugins.Inline) (string, error) {
	warn := func(msg string) {
		if ctx.Warn != nil {
			ctx.Warn(msg)
		}
	}
	ok, found := known[in.Name]
	if !found {
		return "", &notOwned{in.Name}
	}
	var unknown []string
	for k := range in.Attrs {
		if !ok[k] {
			unknown = append(unknown, k)
		}
	}
	sort.Strings(unknown)
	for _, k := range unknown {
		warn(in.Name + ": option " + k + " is not supported and was ignored")
	}
	text := strings.TrimSpace(in.Text)
	if text == "" {
		return "", nil
	}
	if utf8.RuneCountInString(text) > maxText {
		return html.EscapeString(text), nil
	}
	switch in.Name {
	case "kbd":
		return kbd(text), nil
	case "sub", "sup":
		return "<" + in.Name + ">" + html.EscapeString(text) + "</" + in.Name + ">", nil
	}
	// abbr: HTML (HyperText Markup Language), or the title option
	title := strings.TrimSpace(in.Attrs["title"])
	if title == "" {
		if m := abbrRE.FindStringSubmatch(text); m != nil {
			text, title = strings.TrimSpace(m[1]), strings.TrimSpace(m[2])
		}
	}
	if title == "" {
		warn("abbr has no explanation; write it as HTML (HyperText Markup Language)")
		return "<abbr>" + html.EscapeString(text) + "</abbr>", nil
	}
	return `<abbr title="` + html.EscapeString(title) + `">` + html.EscapeString(text) + "</abbr>", nil
}

type notOwned struct{ name string }

func (e *notOwned) Error() string { return "text does not own " + e.name }

// kbd splits a key combination on + and -, one <kbd> per key, the separator
// shown between them. A + or - with no key before or after it is a key.
func kbd(s string) string {
	var sb strings.Builder
	var key strings.Builder
	flush := func() {
		k := strings.TrimSpace(key.String())
		key.Reset()
		if k != "" {
			sb.WriteString("<kbd>" + html.EscapeString(k) + "</kbd>")
		}
	}
	rs := []rune(s)
	for i, r := range rs {
		if (r == '+' || r == '-') && strings.TrimSpace(key.String()) != "" && i+1 < len(rs) {
			flush()
			sb.WriteRune(r)
			continue
		}
		key.WriteRune(r)
	}
	flush()
	return sb.String()
}
