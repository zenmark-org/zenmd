// Package cards renders the cards, grids, link buttons and badges of
// sphinx-design as written in MyST, Jupyter Book and Sphinx, and the same
// blocks written as :::card, :::grid, ... containers and :bdg-primary[text]
// inlines.
//
//   - card: a framed card; "^^^" ends its header and "+++" starts its footer;
//     :link: makes the whole card a link.
//   - grid, grid-item, grid-item-card: a responsive grid; the argument gives
//     the columns for each width ("1 2 3 4").
//   - button-link, button-ref: a plugin kit button that is a link.
//   - bdg, bdg-<colour> and bdg-<colour>-line: a small text badge. The colour
//     name is accepted and every badge is drawn the same, in the kit's muted
//     black and white.
//
// All of it is rendered in Go and reads without scripts. Links go through the
// page's own markdown pipeline, so a link to another page of the folder works
// as every page link does. The look is the plugin kit's (docs/wiki/PLUGIN-UI.md);
// cards.css holds only the grid, the card parts and the badge.
package cards

import (
	"embed"
	"fmt"
	"html"
	"io/fs"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/zenmark-org/zenmd/engine/plugins"
)

//go:embed assets/cards.css
var assets embed.FS

var colours = []string{"primary", "secondary", "success", "info", "warning", "danger", "light", "dark"}

var blockNames = []string{"card", "grid", "grid-item", "grid-item-card", "button-link", "button-ref"}

func inlineNames() []string {
	out := []string{"bdg"}
	for _, c := range colours {
		out = append(out, "bdg-"+c, "bdg-"+c+"-line")
	}
	return out
}

func set(k ...string) map[string]bool {
	m := make(map[string]bool, len(k))
	for _, s := range k {
		m[s] = true
	}
	return m
}

// knownOptions are the options each block reads (the dialect adds title and arg itself).
var knownOptions = map[string]map[string]bool{
	"card":           set("class", "name", "id", "title", "arg", "link", "link-type", "link-alt", "text-align"),
	"grid-item-card": set("class", "name", "id", "title", "arg", "link", "link-type", "link-alt", "text-align"),
	"grid":           set("class", "name", "id", "title", "arg", "columns", "gutter"),
	"grid-item":      set("class", "name", "id", "title", "arg"),
	"button-link":    set("class", "name", "id", "title", "arg", "href", "ref", "align", "expand"),
	"button-ref":     set("class", "name", "id", "title", "arg", "href", "ref", "ref-type", "align", "expand"),
}

var (
	classRE  = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_-]{0,63}$`)
	schemeRE = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9+.-]*:`)
	labelRE  = regexp.MustCompile(`^[A-Za-z0-9_.:-]{1,100}$`)
)

type plugin struct{}

// New returns the cards plugin.
func New() plugins.Plugin { return plugin{} }

func (plugin) Manifest() plugins.Manifest {
	return plugins.Manifest{
		ID:          "cards",
		Version:     "0.1.0",
		APIVersion:  plugins.APIVersion,
		Description: "Cards, grids, link buttons and badges",
		Default:     true,
		Blocks:      blockNames,
		Inlines:     inlineNames(),
		Help: plugins.Help{
			Summary: "Cards, grids, buttons that are links, and badges, as written for MyST, Jupyter Book and Sphinx (sphinx-design). card is a framed block with an optional header and footer (^^^ ends the header, +++ starts the footer), and link makes the whole card a link. grid holds grid-item and grid-item-card blocks; its argument, like 1 2 3 4, is the number of columns at each width. button-link and button-ref are buttons that open a link or a target. bdg and bdg-primary, bdg-secondary, bdg-success, bdg-info, bdg-warning, bdg-danger, bdg-light and bdg-dark (and their -line forms) are small badges; they are written :bdg-primary[text], and every one is drawn in the same black and white.",
			Snippets: []plugins.Snippet{
				{Title: "A card", Markdown: ":::card{title=\"A card\"}\nHeader text\n^^^\nThe body is markdown.\n+++\nFooter text\n:::\n"},
				{Title: "A card that is a link, MyST form", Markdown: "```{card} Read the guide\n:link: https://example.com/guide\nThe whole card opens the link.\n```\n"},
				{Title: "A grid of cards", Markdown: ":::grid{columns=\"1 2 2 3\"}\n:::grid-item-card{title=\"One\"}\nFirst card.\n:::\n:::grid-item-card{title=\"Two\"}\nSecond card.\n:::\n:::grid-item-card{title=\"Three\"}\nThird card.\n:::\n:::\n"},
				{Title: "A grid, MyST form", Markdown: "::::{grid} 1 2 2 3\n:::{grid-item-card} One\nFirst card.\n:::\n:::{grid-item-card} Two\nSecond card.\n:::\n::::\n"},
				{Title: "Buttons", Markdown: ":::button-link{href=\"https://example.com\"}\nOpen the site\n:::\n"},
				{Title: "Buttons, MyST form", Markdown: "```{button-link} https://example.com\nOpen the site\n```\n\n```{button-ref} my-label\n:ref-type: ref\nGo to the section\n```\n"},
				{Title: "Badges", Markdown: "Status: :bdg-primary[new] :bdg-success[stable] :bdg-warning-line[beta] :bdg[plain]\n"},
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

func (plugin) Entry() plugins.Assets { return plugins.Assets{CSS: []string{"cards.css"}} }

// RenderInline draws a badge.
func (plugin) RenderInline(ctx plugins.Context, in plugins.Inline) (string, error) {
	if in.Name != "bdg" && !strings.HasPrefix(in.Name, "bdg-") {
		return "", fmt.Errorf("cards does not own %q", in.Name)
	}
	keys := make([]string, 0, len(in.Attrs))
	for k := range in.Attrs {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		warn(ctx, "option :"+k+": is not supported and was ignored")
	}
	class := "zen-badge"
	if strings.HasSuffix(in.Name, "-line") {
		class += " zen-badge-line"
	}
	return `<span class="` + class + `">` + html.EscapeString(strings.TrimSpace(in.Text)) + `</span>`, nil
}

func warn(ctx plugins.Context, msg string) {
	if ctx.Warn != nil {
		ctx.Warn(msg)
	}
}

// opts reports the unsupported options of b once each, in order.
func opts(ctx plugins.Context, b plugins.Block) {
	var unknown []string
	for k := range b.Attrs {
		if !knownOptions[b.Name][k] {
			unknown = append(unknown, k)
		}
	}
	sort.Strings(unknown)
	for _, k := range unknown {
		warn(ctx, "option :"+k+": is not supported and was ignored")
	}
}

func (plugin) RenderBlock(ctx plugins.Context, b plugins.Block) (string, error) {
	if knownOptions[b.Name] == nil {
		return "", fmt.Errorf("cards does not own %q", b.Name)
	}
	opts(ctx, b)
	switch b.Name {
	case "card", "grid-item-card":
		return renderCard(ctx, b)
	case "grid":
		return renderGrid(ctx, b)
	case "grid-item":
		return renderItem(ctx, b)
	}
	return renderButton(ctx, b)
}

// markdown renders a piece of the body ("" when it is empty).
func markdown(ctx plugins.Context, src string) (string, error) {
	if strings.TrimSpace(src) == "" {
		return "", nil
	}
	if ctx.Markdown == nil {
		return "<p>" + html.EscapeString(strings.TrimSpace(src)) + "</p>", nil
	}
	return ctx.Markdown(src)
}

// idAndClasses reads :name: and :class: (a user class never takes a zen- name).
func idAndClasses(ctx plugins.Context, b plugins.Block) (idAttr string, user []string) {
	if n := b.Attrs["name"]; n != "" {
		if id := slug(n); id != "" {
			idAttr = ` id="` + id + `"`
		} else {
			warn(ctx, "option :name: has a value that is not supported and was ignored")
		}
	}
	for _, tok := range strings.Fields(b.Attrs["class"]) {
		if classRE.MatchString(tok) && !strings.HasPrefix(tok, "zen-") {
			user = append(user, tok)
		}
	}
	return idAttr, user
}

func titleOf(b plugins.Block) string {
	t := strings.TrimSpace(b.Attrs["title"])
	if t == "" {
		t = strings.TrimSpace(b.Attrs["arg"])
	}
	return strings.Join(strings.Fields(t), " ")
}

// ---- card ----

// splitCard cuts a card body at its first "^^^" line (the header ends) and its
// first "+++" line (the footer starts), ignoring lines inside code fences.
func splitCard(body string) (header, main, footer string) {
	lines := strings.Split(body, "\n")
	var fence byte
	fenceLen := 0
	hdrEnd, ftrStart := -1, -1
	for i, l := range lines {
		t := strings.TrimRight(l, " \t\r")
		if c, n := fenceOf(t); n > 0 {
			if fence == 0 {
				fence, fenceLen = c, n
			} else if c == fence && n >= fenceLen && strings.TrimSpace(strings.TrimLeft(t, string(c))) == "" {
				fence = 0
			}
			continue
		}
		if fence != 0 {
			continue
		}
		switch strings.TrimSpace(t) {
		case "^^^":
			if hdrEnd < 0 && ftrStart < 0 {
				hdrEnd = i
			}
		case "+++":
			if ftrStart < 0 {
				ftrStart = i
			}
		}
	}
	start, end := 0, len(lines)
	if ftrStart >= 0 {
		end = ftrStart
		footer = strings.Join(lines[ftrStart+1:], "\n")
	}
	if hdrEnd >= 0 {
		header = strings.Join(lines[:hdrEnd], "\n")
		start = hdrEnd + 1
	}
	return header, strings.Join(lines[start:end], "\n"), footer
}

// fenceOf reports a code fence line: up to three spaces, then three or more
// backticks or tildes.
func fenceOf(line string) (byte, int) {
	i := 0
	for i < len(line) && i < 3 && line[i] == ' ' {
		i++
	}
	if i >= len(line) || (line[i] != '`' && line[i] != '~') {
		return 0, 0
	}
	c := line[i]
	j := i
	for j < len(line) && line[j] == c {
		j++
	}
	if j-i < 3 {
		return 0, 0
	}
	return c, j - i
}

func renderCard(ctx plugins.Context, b plugins.Block) (string, error) {
	idAttr, user := idAndClasses(ctx, b)
	classes := []string{"zen-block", "zen-card"}
	if b.Name == "grid-item-card" {
		classes = append(classes, "zen-grid-item")
	}
	switch a := b.Attrs["text-align"]; a {
	case "":
	case "left", "center", "right":
		classes = append(classes, "zen-card-"+a)
	default:
		warn(ctx, "option :text-align: has a value that is not supported and was ignored")
	}
	title := titleOf(b)

	// the link: the whole card is the stretched link of its title
	linkHTML := ""
	if dest := strings.TrimSpace(b.Attrs["link"]); dest != "" {
		d, ok := destination(ctx, dest, b.Attrs["link-type"], "link-type")
		if ok {
			text := title
			if text == "" {
				text = strings.TrimSpace(b.Attrs["link-alt"])
			}
			if text == "" {
				text = dest
			}
			a, ok := link(ctx, text, d, "zen-card-stretch")
			if ok {
				classes = append(classes, "zen-card-link")
				linkHTML = a
			}
		}
	} else if b.Attrs["link-type"] != "" || b.Attrs["link-alt"] != "" {
		warn(ctx, "option :link-type: and :link-alt: need :link: and were ignored")
	}
	classes = append(classes, user...)

	headerSrc, mainSrc, footerSrc := splitCard(b.Body)
	hdr, err := markdown(ctx, headerSrc)
	if err != nil {
		return "", err
	}
	main, err := markdown(ctx, mainSrc)
	if err != nil {
		return "", err
	}
	ftr, err := markdown(ctx, footerSrc)
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	sb.WriteString(`<div` + idAttr + ` class="` + strings.Join(classes, " ") + `">`)
	if hdr != "" {
		sb.WriteString(`<div class="zen-card-header">` + hdr + `</div>`)
	}
	var inner strings.Builder
	switch {
	case title != "" && linkHTML != "":
		inner.WriteString(`<div class="zen-card-title">` + linkHTML + `</div>`)
	case title != "":
		inner.WriteString(`<div class="zen-card-title">` + html.EscapeString(title) + `</div>`)
	case linkHTML != "":
		inner.WriteString(`<div class="zen-card-linktext">` + linkHTML + `</div>`)
	}
	inner.WriteString(main)
	if inner.Len() > 0 || (hdr == "" && ftr == "") {
		sb.WriteString(`<div class="zen-block-body zen-card-body">` + inner.String() + `</div>`)
	}
	if ftr != "" {
		sb.WriteString(`<div class="zen-card-footer">` + ftr + `</div>`)
	}
	sb.WriteString(`</div>`)
	return sb.String(), nil
}

// ---- grid ----

var widths = []string{"xs", "sm", "md", "lg"}

// numbers reads up to four whole numbers in lo..hi; missing ones repeat the
// last. ok is false when there is something else or too many.
func numbers(s string, lo, hi int) (out [4]int, ok bool) {
	f := strings.Fields(s)
	if len(f) == 0 || len(f) > 4 {
		return out, false
	}
	for i := range out {
		if i < len(f) {
			n, err := strconv.Atoi(f[i])
			if err != nil || n < lo || n > hi {
				return out, false
			}
			out[i] = n
		} else {
			out[i] = out[i-1]
		}
	}
	return out, true
}

func renderGrid(ctx plugins.Context, b plugins.Block) (string, error) {
	idAttr, user := idAndClasses(ctx, b)
	cols := [4]int{1, 1, 1, 1}
	spec := strings.TrimSpace(b.Attrs["columns"])
	if spec == "" {
		spec = strings.TrimSpace(b.Attrs["arg"])
	}
	if spec == "" {
		spec = strings.TrimSpace(b.Attrs["title"])
	}
	if spec != "" {
		if n, ok := numbers(spec, 1, 12); ok {
			cols = n
		} else {
			warn(ctx, "grid columns must be one to four numbers from 1 to 12, like 1 2 3 4; one column is used")
		}
	}
	classes := []string{"zen-grid"}
	for i, w := range widths {
		classes = append(classes, "zen-grid-"+w+"-"+strconv.Itoa(cols[i]))
	}
	classes = append(classes, user...)

	rowClasses := []string{"zen-grid-row"}
	if g := strings.TrimSpace(b.Attrs["gutter"]); g != "" {
		gut, ok := numbers(g, 0, 5)
		switch {
		case !ok:
			warn(ctx, "option :gutter: must be one to four numbers from 0 to 5 and was ignored")
		case gut[0] == gut[1] && gut[1] == gut[2] && gut[2] == gut[3]:
			rowClasses = append(rowClasses, "zen-gutter-"+strconv.Itoa(gut[0]))
		default:
			for i, w := range widths {
				rowClasses = append(rowClasses, "zen-gutter-"+w+"-"+strconv.Itoa(gut[i]))
			}
		}
	}
	body, err := markdown(ctx, b.Body)
	if err != nil {
		return "", err
	}
	return `<div` + idAttr + ` class="` + strings.Join(classes, " ") + `"><div class="` + strings.Join(rowClasses, " ") + `">` + body + `</div></div>`, nil
}

func renderItem(ctx plugins.Context, b plugins.Block) (string, error) {
	idAttr, user := idAndClasses(ctx, b)
	body, err := markdown(ctx, b.Body)
	if err != nil {
		return "", err
	}
	classes := append([]string{"zen-grid-item"}, user...)
	return `<div` + idAttr + ` class="` + strings.Join(classes, " ") + `">` + body + `</div>`, nil
}

// ---- buttons ----

func renderButton(ctx plugins.Context, b plugins.Block) (string, error) {
	idAttr, user := idAndClasses(ctx, b)
	target := ""
	for _, k := range []string{"href", "ref", "arg", "title"} {
		if v := strings.TrimSpace(b.Attrs[k]); v != "" {
			target = v
			break
		}
	}
	if target == "" {
		warn(ctx, b.Name+" needs a link target; it is shown as plain text")
	}
	text := strings.Join(strings.Fields(b.Body), " ")
	if len(text) > 200 {
		text = text[:200]
	}
	if text == "" {
		text = target
	}
	if text == "" {
		text = "Link"
	}
	rowClasses := []string{"zen-linkbtn-row"}
	switch a := b.Attrs["align"]; a {
	case "":
	case "left", "center", "right":
		rowClasses = append(rowClasses, "zen-linkbtn-"+a)
	default:
		warn(ctx, "option :align: has a value that is not supported and was ignored")
	}
	btn := []string{"zen-btn", "zen-linkbtn"}
	if v, ok := b.Attrs["expand"]; ok && v != "false" {
		rowClasses = append(rowClasses, "zen-linkbtn-expand")
	}
	rowClasses = append(rowClasses, user...)

	linkType := "url"
	if b.Name == "button-ref" {
		linkType = "any"
		if t := b.Attrs["ref-type"]; t != "" {
			linkType = t
		}
	}
	inner := html.EscapeString(text)
	if target != "" {
		if d, ok := destination(ctx, target, linkType, "ref-type"); ok {
			if a, ok := link(ctx, text, d, strings.Join(btn, " ")); ok {
				inner = a
			}
		}
	}
	return `<div` + idAttr + ` class="` + strings.Join(rowClasses, " ") + `">` + inner + `</div>`, nil
}

// ---- links ----

// destination turns a target and its type (url, ref, doc, any; "myst" is any)
// into what a markdown link points at. A scheme other than http, https or
// mailto is refused with a warning.
func destination(ctx plugins.Context, target, typ, option string) (string, bool) {
	target = strings.TrimSpace(target)
	if strings.ContainsAny(target, "<>\r\n\x00") || len([]rune(target)) > 2000 {
		warn(ctx, "a link target is not supported and was ignored")
		return "", false
	}
	switch typ {
	case "", "url", "ref", "doc", "any":
	case "myst":
		typ = "any"
	default:
		warn(ctx, "option :"+option+": has a value that is not supported and was ignored")
		typ = "any"
	}
	if typ == "" {
		typ = "url"
	}
	if typ == "any" {
		switch {
		case schemeRE.MatchString(target):
			typ = "url"
		case strings.Contains(target, "/") || path.Ext(target) == ".md":
			typ = "doc"
		default:
			typ = "ref"
		}
	}
	switch typ {
	case "ref":
		if !labelRE.MatchString(target) {
			warn(ctx, "a ref target has characters that are not supported and was ignored")
			return "", false
		}
		return "#" + target, true
	case "doc":
		if path.Ext(target) == "" {
			target += ".md"
		}
	}
	if m := schemeRE.FindString(target); m != "" {
		switch strings.ToLower(strings.TrimSuffix(m, ":")) {
		case "http", "https", "mailto":
		default:
			warn(ctx, "a link target with the scheme "+strings.ToLower(m)+" is not supported and was ignored")
			return "", false
		}
	}
	return target, true
}

// link renders [text](dest) through the page's markdown pipeline, so internal
// pages, anchors and external links behave as every page link does, and puts
// class on the anchor. The text is escaped: it is never markdown.
func link(ctx plugins.Context, text, dest, class string) (string, bool) {
	if ctx.Markdown == nil {
		return `<a class="` + class + `" href="` + html.EscapeString(dest) + `">` + html.EscapeString(text) + `</a>`, true
	}
	h, err := ctx.Markdown("[" + escapeMD(text) + "](<" + dest + ">)")
	if err != nil {
		return "", false
	}
	h = strings.TrimSpace(h)
	h = strings.TrimSuffix(strings.TrimPrefix(h, "<p>"), "</p>")
	h = strings.TrimSpace(h)
	if !strings.HasPrefix(h, "<a ") || strings.Count(h, "<a ") != 1 || !strings.HasSuffix(h, "</a>") {
		return "", false
	}
	return `<a class="` + class + `" ` + h[len("<a "):], true
}

// escapeMD makes s plain text in markdown: every ASCII punctuation mark is
// backslash-escaped and line breaks become spaces.
func escapeMD(s string) string {
	var sb strings.Builder
	for _, r := range strings.Join(strings.Fields(s), " ") {
		if r < 128 && strings.ContainsRune("!\"#$%&'()*+,-./:;<=>?@[\\]^_`{|}~", r) {
			sb.WriteByte('\\')
		}
		sb.WriteRune(r)
	}
	return sb.String()
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
