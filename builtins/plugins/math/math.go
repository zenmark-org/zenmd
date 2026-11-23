// Package math renders display math: the MyST directive {math}, the same block
// written as :::math, and (through the MyST dialect) $$ … $$ blocks. A block
// with a label is numbered "(1)", "(2)" … per page and is a link target.
//
// Go writes the TeX as escaped text in a kit frame, so a page without scripts
// reads it, and assets/math.js draws it with the vendored KaTeX bundle (see
// VENDOR.md) on pages that have math. The inline {math} and {eq} roles and
// $ … $ belong to a later phase.
package math

import (
	"embed"
	"html"
	"io/fs"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/zenmark-org/zenmd/engine/plugins"
)

//go:embed assets/math.js assets/math.css assets/katex.min.js assets/katex.min.css assets/KATEX-LICENSE assets/fonts
var assets embed.FS

type plugin struct{}

// New returns the math plugin.
func New() plugins.Plugin { return plugin{} }

func (plugin) Manifest() plugins.Manifest {
	return plugins.Manifest{
		ID:          "math",
		Version:     "0.1.0",
		APIVersion:  plugins.APIVersion,
		Description: "Display math, drawn with KaTeX",
		Default:     true,
		Help: plugins.Help{
			Summary: "Draws TeX math as a centred equation. Write it as a math block, a MyST {math} directive, or (in a folder that uses the MyST dialect) between $$ lines. A label numbers the equation on its page, (1), (2) and so on, and makes it a link target; a name makes a target without a number. Options follow MyST: label, name, class and nowrap (do not put an equation with \\\\ in aligned). A wrong equation keeps its TeX visible, with KaTeX's message under it. KaTeX is loaded only on pages that have math.",
			Snippets: []plugins.Snippet{
				{Title: "An equation", Markdown: ":::math\nE = mc^2\n:::\n"},
				{Title: "A numbered equation", Markdown: ":::math{label=euler}\ne^{i\\pi} + 1 = 0\n:::\n"},
				{Title: "The MyST form", Markdown: "```{math}\n:label: eq:pythagoras\n\na^2 + b^2 = c^2\n```\n"},
				{Title: "Between $$ lines (MyST dialect)", Markdown: "$$\n\\int_0^1 x^2\\,dx = \\frac13\n$$ (eq:integral)\n"},
			},
		},
		Blocks: []string{"math"},
	}
}

func (plugin) Assets() fs.FS {
	sub, err := fs.Sub(assets, "assets")
	if err != nil {
		panic(err)
	}
	return sub
}

// Entry lists the files a page with math loads. katex.min.css names its fonts
// as url(fonts/…), which resolve next to it wherever the assets are served.
func (plugin) Entry() plugins.Assets {
	return plugins.Assets{CSS: []string{"katex.min.css", "math.css"}, JS: []string{"math.js"}}
}

const (
	numOpen  = `<span class="zm-num">⟦`
	numClose = `⟧</span>`
)

var numRE = regexp.MustCompile(`<span class="zm-num">⟦([^⟧]*)⟧</span>`)

// HTML numbers the labelled equations of the finished page: (1), (2) … in the
// order they appear. The numbers belong to one render, so a page starts at 1.
func (plugin) HTML(ctx plugins.Context, s string) (string, error) {
	if !strings.Contains(s, numOpen) {
		return s, nil
	}
	seen := map[string]bool{}
	var sb strings.Builder
	n, last := 0, 0
	for _, m := range numRE.FindAllStringSubmatchIndex(s, -1) {
		id := s[m[2]:m[3]]
		n++
		sb.WriteString(s[last:m[0]])
		if seen[id] {
			if ctx.Warn != nil {
				ctx.Warn("duplicate equation label " + id + ": the first one keeps the link target")
			}
			// the block's opening tag is the nearest one before its number
			out := sb.String()
			if i := strings.LastIndex(out, ` id="`+id+`"`); i >= 0 {
				sb.Reset()
				sb.WriteString(out[:i] + out[i+len(` id="`+id+`"`):])
			}
		}
		seen[id] = true
		sb.WriteString(`<span class="zm-num">(` + strconv.Itoa(n) + `)</span>`)
		last = m[1]
	}
	sb.WriteString(s[last:])
	return sb.String(), nil
}

// known options; anything else is reported and ignored.
var known = map[string]bool{
	"arg": true, "title": true, "label": true, "name": true, "class": true, "nowrap": true,
}

var (
	classRE = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_-]{0,40}$`)
	keyRE   = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_-]*$`)
)

func (plugin) RenderBlock(ctx plugins.Context, b plugins.Block) (string, error) {
	attrs, body := yamlOptions(b.Attrs, b.Body)
	note := func(msg string) {
		if ctx.Warn != nil {
			ctx.Warn(msg)
		}
	}
	keys := make([]string, 0, len(attrs))
	for k := range attrs {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if !known[k] {
			note("option :" + k + ": is not supported and was ignored")
		}
	}

	tex := strings.TrimSpace(body)
	if tex == "" { // {math} x^2 puts the TeX in the argument
		tex = strings.TrimSpace(attrs["arg"])
	}
	if tex == "" {
		return "", &optionError{"the equation is empty: write TeX in the block's body"}
	}

	// The target: a label (numbered) wins over a name (not numbered).
	id, numbered := "", false
	if l := strings.TrimSpace(attrs["label"]); l != "" {
		if id = slug(l); id == "" {
			note("option :label: has a value that is not supported and was ignored")
		} else {
			numbered = true
		}
	}
	if n := strings.TrimSpace(attrs["name"]); n != "" {
		switch {
		case numbered:
			note("option :name: was ignored: :label: already names the equation")
		case slug(n) == "":
			note("option :name: has a value that is not supported and was ignored")
		default:
			id = slug(n)
		}
	}

	classes := []string{"zen-block", "zm"}
	if v, ok := attrs["nowrap"]; ok && !strings.EqualFold(v, "false") {
		classes = append(classes, "zm-nowrap")
	}
	for _, c := range strings.Fields(attrs["class"]) {
		if classRE.MatchString(c) && !reserved(c) {
			classes = append(classes, c)
		} else {
			note("option :class: has a class name that is not supported and was ignored")
		}
	}
	idAttr := ""
	if id != "" {
		idAttr = ` id="` + id + `"`
	}
	num := ""
	if numbered {
		num = numOpen + id + numClose
	}
	return `<div class="` + strings.Join(classes, " ") + `"` + idAttr + `><div class="zen-block-body"><div class="zm-row">` +
		`<pre class="zm-tex">` + html.EscapeString(tex) + `</pre>` + num + `</div></div></div>`, nil
}

type optionError struct{ msg string }

func (e *optionError) Error() string { return e.msg }

func reserved(c string) bool {
	return strings.HasPrefix(c, "zen") || strings.HasPrefix(c, "zm-") || c == "zm"
}

// slug turns a label into an id: lower case, letters, digits and . _ : -.
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

// yamlOptions reads a body that starts with a --- block of `key: value` lines,
// MyST's other way of writing options. Options given as :key: lines win.
func yamlOptions(in map[string]string, body string) (map[string]string, string) {
	out := make(map[string]string, len(in)+4)
	for k, v := range in {
		out[k] = v
	}
	t := strings.TrimLeft(body, "\r\n")
	if !strings.HasPrefix(t, "---\n") && !strings.HasPrefix(t, "---\r\n") {
		return out, body
	}
	lines := strings.Split(t, "\n")
	end := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimRight(lines[i], " \t\r") == "---" {
			end = i
			break
		}
	}
	if end < 0 {
		return out, body
	}
	for _, l := range lines[1:end] {
		k, v, ok := strings.Cut(l, ":")
		k = strings.TrimSpace(k)
		if !ok || !keyRE.MatchString(k) {
			continue
		}
		v = strings.TrimSpace(v)
		if len(v) >= 2 && (v[0] == '"' && v[len(v)-1] == '"' || v[0] == '\'' && v[len(v)-1] == '\'') {
			v = v[1 : len(v)-1]
		}
		if v == "" {
			v = "true"
		}
		if _, set := out[k]; !set {
			out[k] = v
		}
	}
	return out, strings.Join(lines[end+1:], "\n")
}
