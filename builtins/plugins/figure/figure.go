// Package figure renders the MyST directives {figure} and {image}, and the
// same blocks written as :::figure and :::image. A figure is an image with a
// caption (the body's first paragraph) and a legend (the paragraphs after it),
// numbered "Figure 1", "Figure 2" … per page; an image is the picture alone.
//
// The image is not resolved here. It goes through the page's own markdown
// (Context.Markdown), so a figure finds its file exactly as ![](path) does:
// relative to the page's folder, https for a remote one, a missing one shown as
// text instead of a picture. The output is plain HTML that reads without
// scripts; assets/figure.css only styles it.
package figure

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

//go:embed assets/figure.css
var assets embed.FS

type plugin struct{}

// New returns the figure plugin.
func New() plugins.Plugin { return plugin{} }

func (plugin) Manifest() plugins.Manifest {
	return plugins.Manifest{
		ID:          "figure",
		Version:     "0.1.0",
		APIVersion:  plugins.APIVersion,
		Description: "Figures and images, as MyST writes them",
		Default:     true,
		Help: plugins.Help{
			Summary: "A figure is a picture with a caption and an optional legend, numbered per page (Figure 1, Figure 2). An image is the picture alone. The file is found the way an ordinary page image is: relative to the page's folder. Options follow MyST: alt, width, height, align (left, center, right), name (a link target), target (a link) and class.",
			Snippets: []plugins.Snippet{
				{Title: "A figure", Markdown: ":::figure{src=images/chart.png alt=\"Sales by month\" width=60% name=sales}\nSales rose every month.\n\nThe legend is the paragraphs after the caption.\n:::\n"},
				{Title: "A figure, MyST form", Markdown: "```{figure} images/chart.png\n:alt: Sales by month\n:width: 60%\n:name: sales\n\nSales rose every month.\n\nThe legend is the paragraphs after the caption.\n```\n"},
				{Title: "An image, MyST form", Markdown: "```{image} images/logo.png\n:alt: The logo\n:align: right\n```\n"},
			},
		},
		Blocks: []string{"figure", "image"},
	}
}

func (plugin) Assets() fs.FS {
	sub, err := fs.Sub(assets, "assets")
	if err != nil {
		panic(err)
	}
	return sub
}

func (plugin) Entry() plugins.Assets { return plugins.Assets{CSS: []string{"figure.css"}} }

// numMarker stands in the caption until HTML numbers the figures of the page,
// in the order they appear.
const numMarker = `<span class="zf-num"></span>`

// HTML numbers the figures of the finished page: Figure 1, Figure 2 … The
// numbers belong to one render, so a page always starts again at 1.
func (plugin) HTML(_ plugins.Context, s string) (string, error) {
	if !strings.Contains(s, numMarker) {
		return s, nil
	}
	var sb strings.Builder
	n := 0
	for {
		i := strings.Index(s, numMarker)
		if i < 0 {
			break
		}
		n++
		sb.WriteString(s[:i])
		sb.WriteString(`<span class="zf-num">` + strconv.Itoa(n) + `</span>`)
		s = s[i+len(numMarker):]
	}
	sb.WriteString(s)
	return sb.String(), nil
}

// supported options; anything else is shown as ignored.
var known = map[string]bool{
	"src": true, "arg": true, "title": true, "alt": true, "width": true, "height": true, "align": true,
	"name": true, "target": true, "class": true, "figclass": true, "loading": true,
	"nonumber": true, "numbered": true,
}

var (
	imgRE   = regexp.MustCompile(`<img [^>]*>`)
	linkRE  = regexp.MustCompile(`<a [^>]*>zenfigimg</a>`)
	dimRE   = regexp.MustCompile(`^([0-9]{1,5})(px)?$`)
	pctRE   = regexp.MustCompile(`^([0-9]{1,3})%$`)
	classRE = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_-]{0,40}$`)
	keyRE   = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_-]*$`)
	blankRE = regexp.MustCompile(`\n[ \t]*\n`)
)

func (plugin) RenderBlock(ctx plugins.Context, b plugins.Block) (string, error) {
	attrs, body := yamlOptions(b.Attrs, b.Body)
	var notes []string
	note := func(key, why string) {
		notes = append(notes, "Option :"+key+": "+why+" and was ignored.")
	}
	keys := make([]string, 0, len(attrs))
	for k := range attrs {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if !known[k] {
			note(k, "is not supported")
		}
	}

	src := strings.TrimSpace(attrs["arg"])
	if src == "" {
		src = strings.TrimSpace(attrs["src"])
	}
	if src == "" {
		return "", errMissing(b.Name)
	}
	isImage := b.Name == "image"

	imgHTML := ""
	if ok := safeDest(src); ok {
		imgHTML = pageImage(ctx, src, attrs["alt"])
	}
	if imgHTML != "" {
		var extra strings.Builder
		if v, ok := dimension(attrs["width"], true); ok {
			extra.WriteString(` width="` + v + `"`)
		} else if attrs["width"] != "" {
			note("width", "has a value that is not supported (use px or %)")
		}
		if v, ok := dimension(attrs["height"], false); ok {
			extra.WriteString(` height="` + v + `"`)
		} else if attrs["height"] != "" {
			note("height", "has a value that is not supported (use px)")
		}
		if attrs["arg"] == "" && attrs["title"] != "" {
			extra.WriteString(` title="` + html.EscapeString(attrs["title"]) + `"`)
		}
		imgHTML = strings.TrimSuffix(imgHTML, ">") + extra.String() + ">"
		if t := strings.TrimSpace(attrs["target"]); t != "" {
			imgHTML = linked(ctx, t, imgHTML, func() { note("target", "has a value that is not supported") })
		}
	}

	classes := []string{"zf"}
	align := strings.ToLower(strings.TrimSpace(attrs["align"]))
	switch align {
	case "left", "center", "right":
	case "":
		align = "center"
		if isImage {
			align = "left"
		}
	default:
		note("align", "has a value that is not supported (left, center or right)")
		align = "center"
		if isImage {
			align = "left"
		}
	}
	classes = append(classes, "zf-"+align)
	for _, k := range []string{"class", "figclass"} {
		for _, c := range strings.Fields(attrs[k]) {
			if classRE.MatchString(c) && !reserved(c) {
				classes = append(classes, c)
			} else {
				note(k, "has a class name that is not supported")
			}
		}
	}
	id := ""
	if n := attrs["name"]; n != "" {
		if id = slug(n); id == "" {
			note("name", "has a value that is not supported")
		}
	}
	idAttr := ""
	if id != "" {
		idAttr = ` id="` + id + `"`
	}

	missing := ""
	if imgHTML == "" {
		missing = `<div class="zen-block-status zf-missing">Image not found: ` + html.EscapeString(clip(src, 200)) + `</div>`
	}
	var notesHTML strings.Builder
	for _, n := range notes {
		notesHTML.WriteString(`<div class="zen-block-status zf-note">` + html.EscapeString(n) + `</div>`)
	}

	if isImage {
		classes = append(classes, "zf-image")
		return `<div class="` + strings.Join(classes, " ") + `"` + idAttr + `>` + imgHTML + missing + notesHTML.String() + `</div>`, nil
	}

	caption, legend := splitCaption(body)
	var fc strings.Builder
	if caption != "" {
		fc.WriteString(`<figcaption>`)
		if numbered(attrs) {
			fc.WriteString(`<span class="zf-label">Figure ` + numMarker + `</span>: `)
		}
		fc.WriteString(inline(ctx, caption))
		if legend != "" {
			fc.WriteString(`<div class="zf-legend">` + markdown(ctx, legend) + `</div>`)
		}
		fc.WriteString(`</figcaption>`)
	} else if legend != "" {
		fc.WriteString(`<figcaption><div class="zf-legend">` + markdown(ctx, legend) + `</div></figcaption>`)
	}
	return `<div class="zen-block ` + strings.Join(classes, " ") + `"><div class="zen-block-body"><figure` + idAttr + `>` +
		imgHTML + missing + fc.String() + `</figure>` + notesHTML.String() + `</div></div>`, nil
}

func errMissing(name string) error {
	return &optionError{"the image is missing: write its path after {" + name + "} (or src=)"}
}

type optionError struct{ msg string }

func (e *optionError) Error() string { return e.msg }

// pageImage renders ![alt](src) with the page's markdown and returns its <img>
// element; "" when the page would not show a picture for it (a missing file, a
// path outside the folder, a scheme other than https).
func pageImage(ctx plugins.Context, src, alt string) string {
	if ctx.Markdown == nil {
		return ""
	}
	out, err := ctx.Markdown("![" + escapePunct(alt) + "](<" + escapeDest(src) + ">)")
	if err != nil {
		return ""
	}
	return imgRE.FindString(out)
}

// linked wraps img in the link the page would make for target.
func linked(ctx plugins.Context, target, img string, bad func()) string {
	if ctx.Markdown == nil || !safeDest(target) {
		bad()
		return img
	}
	out, err := ctx.Markdown("[zenfigimg](<" + escapeDest(target) + ">)")
	if err != nil {
		bad()
		return img
	}
	a := linkRE.FindString(out)
	if a == "" {
		bad()
		return img
	}
	return strings.TrimSuffix(a, "zenfigimg</a>") + img + "</a>"
}

// safeDest reports whether s can sit between < > as a markdown link target.
func safeDest(s string) bool {
	if s == "" || len(s) > 2000 || strings.ContainsAny(s, "<>\n\r\x00") {
		return false
	}
	return true
}

func escapeDest(s string) string {
	return strings.NewReplacer(`\`, `\\`, `&`, `\&`).Replace(s)
}

// escapePunct backslash-escapes every ASCII punctuation mark, so text reaches
// the page's markdown as plain text.
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

func dimension(v string, percent bool) (string, bool) {
	v = strings.TrimSpace(v)
	if m := dimRE.FindStringSubmatch(v); m != nil {
		return m[1], true
	}
	if m := pctRE.FindStringSubmatch(v); m != nil && percent {
		return m[1] + "%", true
	}
	return "", false
}

func reserved(c string) bool {
	return strings.HasPrefix(c, "zen") || strings.HasPrefix(c, "zf-") || c == "zf"
}

// slug turns a :name: into an id: lower case, letters, digits and . _ : -.
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

func clip(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

func numbered(attrs map[string]string) bool {
	if v, ok := attrs["nonumber"]; ok && v != "false" {
		return false
	}
	switch strings.ToLower(attrs["numbered"]) {
	case "false", "no", "0", "off":
		return false
	}
	return true
}

// splitCaption cuts the body into the caption (its first paragraph) and the
// legend (the rest).
func splitCaption(body string) (caption, legend string) {
	body = strings.TrimSpace(body)
	if body == "" {
		return "", ""
	}
	if loc := blankRE.FindStringIndex(body); loc != nil {
		return strings.TrimSpace(body[:loc[0]]), strings.TrimSpace(body[loc[1]:])
	}
	return body, ""
}

func markdown(ctx plugins.Context, s string) string {
	if ctx.Markdown == nil {
		return "<p>" + html.EscapeString(s) + "</p>"
	}
	out, err := ctx.Markdown(s)
	if err != nil {
		return "<p>" + html.EscapeString(s) + "</p>"
	}
	return out
}

// inline renders s and drops the paragraph around it when it is only one.
func inline(ctx plugins.Context, s string) string {
	out := strings.TrimSpace(markdown(ctx, s))
	if strings.HasPrefix(out, "<p>") && strings.HasSuffix(out, "</p>") && strings.Count(out, "<p>") == 1 {
		return out[3 : len(out)-4]
	}
	return out
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
