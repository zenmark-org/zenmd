package render

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html"
	"os"
	"sort"
	"strings"
	"unicode"

	"github.com/zenmark-org/zenmd/engine/dialects"
	"github.com/zenmark-org/zenmd/engine/plugins"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// maxNesting is how many levels of plugin containers with default content
// may nest.
const maxNesting = 8

// state is the per-render mutable state; one goroutine uses it.
type state struct {
	r           *Renderer
	set         *plugins.Set
	path        string
	nonce       string
	frags       []string // placeholder index -> HTML
	texts       []string // placeholder index -> plain text (for the title)
	used        map[string]bool
	warnings    []string
	denied      []string               // ids of not-allowed plugins the page uses, in order
	depth       int                    // nesting of default-content containers
	titleOnly   bool                   // only the title is wanted: run no hooks
	dialects    []dialects.Dialect     // the markdown dialects the folder turned on
	pluginWarns map[string]int         // plugin id -> Context.Warn warnings kept on this page
	warnSeen    map[string]bool        // warnings already added through Context.Warn
	closed      bool                   // the page is built: Context.Warn adds nothing
	subs        map[string]string      // the page's frontmatter substitutions (a dialect's {{ key }})
	exts        dialects.Extensions    // the extensions the folder turned on for the dialects; nil until asked
	notes       map[string]bool        // dialect warnings already given on this page
	titles      map[string]titleAnswer // Context.PageTitle answers, by path
	lookups     int                    // Context.PageTitle lookups made on this page
}

// titleAnswer is one answer of Context.PageTitle.
type titleAnswer struct {
	title string
	ok    bool
}

// maxTitleLookups is how many different pages one render may ask the title of.
const maxTitleLookups = 500

const (
	maxPluginWarns   = 20  // Context.Warn warnings per plugin per page
	maxPluginWarnLen = 200 // characters in one of them
)

// rooter is implemented by a Resolver that knows the open folder's root.
type rooter interface{ Root() string }

// ctx is the hook context of plugin id: the page's settings and a way to
// render markdown through this same render (Context.Markdown).
func (st *state) ctx(id string) plugins.Context {
	c := st.set.Context(id, st.path)
	c.Markdown = st.markdown
	if rt, ok := st.r.links.(rooter); ok {
		c.Workspace = rt.Root()
	}
	c.Warn = func(msg string) { st.pluginWarn(id, msg) }
	if _, ok := st.r.links.(pageReader); ok {
		c.PageTitle = st.pageTitle
	}
	return c
}

// pageReader is implemented by a Resolver that can read the folder's pages.
type pageReader interface {
	Read(rel string) ([]byte, error)
}

// pageTitle is Context.PageTitle: the title of the folder page that path
// names, relative to the page being rendered as a link is. It reads the
// page's source and runs no plugin, and never leaves the folder, because the
// path is resolved as a link is.
func (st *state) pageTitle(p string) (string, bool) {
	rd, ok := st.r.links.(pageReader)
	p = strings.TrimSpace(p)
	if !ok || p == "" || len(p) > 500 || strings.ContainsAny(p, "\\\x00\r\n") {
		return "", false
	}
	if a, seen := st.titles[p]; seen {
		return a.title, a.ok
	}
	if st.lookups >= maxTitleLookups {
		return "", false
	}
	st.lookups++
	var a titleAnswer
	if l := st.r.resolve(st.path, p); l.Kind == "internal" {
		if src, err := rd.Read(l.Path); err == nil {
			a = titleAnswer{st.r.Title(l.Path, src), true}
		}
	}
	if st.titles == nil {
		st.titles = map[string]titleAnswer{}
	}
	st.titles[p] = a
	return a.title, a.ok
}

// note adds a page warning once (a dialect's, which has no plugin id).
func (st *state) note(msg string) {
	if st.titleOnly || st.notes[msg] {
		return
	}
	if st.notes == nil {
		st.notes = map[string]bool{}
	}
	st.notes[msg] = true
	st.warnings = append(st.warnings, msg)
}

// extensions are the optional syntaxes the folder turned on for the page's
// dialects, read once per page and only when something asks.
func (st *state) extensions() dialects.Extensions {
	if st.exts != nil {
		return st.exts
	}
	st.exts = dialects.Extensions{}
	rt, ok := st.r.links.(rooter)
	if !ok || rt.Root() == "" {
		return st.exts
	}
	for _, d := range st.dialects {
		if e, ok := d.(dialects.Extensioner); ok {
			for k, v := range e.Extensions(os.DirFS(rt.Root())) {
				st.exts[k] = v
			}
		}
	}
	return st.exts
}

// markdown options for the dialects that need goldmark's extras.
func (st *state) newMarkdown(extra ...goldmark.Option) goldmark.Markdown {
	for _, d := range st.dialects {
		if dl, ok := d.(dialects.DefinitionLists); ok && dl.DefinitionLists() {
			extra = append(extra, goldmark.WithExtensions(extension.DefinitionList))
			break
		}
	}
	return newMarkdown(extra...)
}

// pluginWarn adds msg to the page's warnings as plugin id's: control
// characters removed, cut to maxPluginWarnLen characters, a message the page
// already has dropped, and at most maxPluginWarns per plugin.
func (st *state) pluginWarn(id, msg string) {
	if st.closed || st.titleOnly {
		return
	}
	msg = strings.TrimSpace(strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, msg))
	if rs := []rune(msg); len(rs) > maxPluginWarnLen {
		msg = string(rs[:maxPluginWarnLen])
	}
	if msg == "" {
		return
	}
	w := fmt.Sprintf("plugin %q: %s", id, msg)
	if st.warnSeen[w] || st.pluginWarns[id] >= maxPluginWarns {
		return
	}
	if st.warnSeen == nil {
		st.warnSeen, st.pluginWarns = map[string]bool{}, map[string]int{}
	}
	st.warnSeen[w] = true
	st.pluginWarns[id]++
	st.warnings = append(st.warnings, w)
}

// markdown renders src as a fragment of the page being rendered, with its
// plugins, nested at most maxNesting levels deep. The HTML is sanitised with
// the page.
func (st *state) markdown(src string) (string, error) {
	if st.titleOnly {
		return "", nil
	}
	if st.depth >= maxNesting {
		return "", fmt.Errorf("markdown nested more than %d levels deep", maxNesting)
	}
	st.depth++
	defer func() { st.depth-- }()
	return st.fragment(src), nil
}

func newNonce() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func (st *state) warn(id string, err error) {
	st.warnings = append(st.warnings, fmt.Sprintf("plugin %q: %v", id, err))
}

// deny records that the page uses a name of a plugin that is not allowed.
func (st *state) deny(id string) {
	for _, d := range st.denied {
		if d == id {
			return
		}
	}
	st.denied = append(st.denied, id)
}

func (st *state) token(i int) string { return fmt.Sprintf("zenph%sn%dz", st.nonce, i) }

// placeholder stores fragment HTML and returns its token.
func (st *state) placeholder(html, plain string) string {
	st.frags = append(st.frags, html)
	st.texts = append(st.texts, plain)
	return st.token(len(st.frags) - 1)
}

// attrsJSON returns the data-zen-attrs attribute for attrs ("" when empty).
func attrsJSON(attrs map[string]string) string {
	if len(attrs) == 0 {
		return ""
	}
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	if enc.Encode(attrs) != nil {
		return ""
	}
	return ` data-zen-attrs="` + html.EscapeString(strings.TrimSpace(b.String())) + `"`
}

func wrapper(tag, id, name string, attrs map[string]string, inner string) string {
	return "<" + tag + ` data-zen-plugin="` + html.EscapeString(id) + `" data-zen-name="` + html.EscapeString(name) + `"` +
		attrsJSON(attrs) + ">" + inner + "</" + tag + ">"
}

// block renders a block and returns the wrapped HTML. A plugin without a
// block hook gets default content: a container's body as markdown, a fence's
// body as text.
func (st *state) block(p plugins.Plugin, b plugins.Block) string {
	id := p.Manifest().ID
	if st.titleOnly {
		return ""
	}
	var out string
	var err error
	if h, ok := plugins.BlockHookOf(p); ok {
		err = guard(func() (e error) { out, e = h.RenderBlock(st.ctx(id), b); return })
	} else if b.Fenced {
		out = `<pre class="zen-plugin-source">` + html.EscapeString(b.Body) + "</pre>"
	} else if st.depth >= maxNesting {
		st.warnings = append(st.warnings, fmt.Sprintf("plugin %q: containers nested more than %d levels deep are shown as text", id, maxNesting))
		return `<pre class="zen-plugin-source">` + html.EscapeString(b.Body) + "</pre>"
	} else {
		st.depth++
		out = st.fragment(b.Body)
		st.depth--
	}
	if err != nil {
		st.warn(id, err)
		out = plugins.ErrorBox(id, err)
	} else {
		st.used[id] = true
	}
	return wrapper("div", id, b.Name, b.Attrs, out)
}

func (st *state) inline(p plugins.Plugin, in plugins.Inline) string {
	id := p.Manifest().ID
	if st.titleOnly {
		return ""
	}
	var out string
	var err error
	if h, ok := plugins.InlineHookOf(p); ok {
		err = guard(func() (e error) { out, e = h.RenderInline(st.ctx(id), in); return })
	} else {
		out = html.EscapeString(in.Text)
	}
	if err != nil {
		st.warn(id, err)
		out = plugins.ErrorBox(id, err)
	} else {
		st.used[id] = true
	}
	return wrapper("span", id, in.Name, in.Attrs, out)
}

// fragment renders markdown (a container's body) through the same pipeline
// and returns its HTML, placeholders filled.
func (st *state) fragment(src string) string {
	body := st.replaceContainers([]byte(src))
	body = st.replaceInlines(body)
	md := st.newMarkdown(goldmark.WithRendererOptions(
		renderer.WithNodeRenderers(util.Prioritized(&nodeRenderer{st: st, r: st.r}, 100)),
	))
	doc := md.Parser().Parse(text.NewReader(body))
	var buf bytes.Buffer
	if err := md.Renderer().Render(&buf, body, doc); err != nil {
		return html.EscapeString(src)
	}
	return st.fill(buf.String())
}

// replaceContainers renders the outermost containers owned by an enabled
// plugin and puts a placeholder paragraph in their place. With markdown
// dialects turned on, their directives (MyST's ```{note}) are containers too.
func (st *state) replaceContainers(src []byte) []byte {
	if len(st.dialects) > 0 {
		return st.rewrite(st.replaceWithDialects(src))
	}
	if !bytes.Contains(src, []byte(":::")) {
		return src
	}
	return st.replaceSpans(src, plugins.ScanContainers(src), nil)
}

// rewrite lets the dialects turn their syntax that is not a directive
// (targets, comments, breaks, substitutions) into markdown. It runs after the
// directives became placeholders, so a directive's body reaches its plugin as
// written and is rewritten when the plugin renders it as markdown.
func (st *state) rewrite(src []byte) []byte {
	for _, d := range st.dialects {
		if rw, ok := d.(dialects.Rewriter); ok {
			var warns []string
			src, warns = rw.Rewrite(src, dialects.Page{Substitutions: st.subs})
			for _, w := range warns {
				st.note(w)
			}
		}
	}
	return src
}

// replaceSpans replaces the outermost spans of an enabled plugin. dm holds the
// dialects' spans by start offset (their code flag and warnings); spans is
// already in source order.
func (st *state) replaceSpans(src []byte, spans []plugins.Span, dm map[int]dialects.Span) []byte {
	var out bytes.Buffer
	last := 0
	for _, sp := range spans {
		if sp.Start < last {
			continue // nested in a replaced container
		}
		ds, isDialect := dm[sp.Start]
		if isDialect && ds.Code {
			for _, w := range ds.Warnings {
				st.note(w)
			}
			out.Write(src[last:sp.Start])
			out.WriteString(codeFenceText(ds))
			last = sp.End
			continue
		}
		p := st.set.Block(sp.Name)
		if p == nil {
			if id := st.set.Denied("block", sp.Name); id != "" {
				st.deny(id)
			} else if isDialect && !st.titleOnly {
				// no plugin owns the directive: it stays code, never run or passed on
				st.note("myst: the directive {" + sp.Name + "} is not supported and stays as code")
				shown := ds
				shown.Lang = ""
				if b := bodyOf(src, sp); b != "" {
					shown.Body = b
				}
				out.Write(src[last:sp.Start])
				out.WriteString(codeFenceText(shown))
				last = sp.End
			}
			continue
		}
		for _, w := range ds.Warnings {
			st.note(w)
		}
		h := st.block(p, plugins.Block{Name: sp.Name, Body: sp.Body, Attrs: sp.Attrs})
		out.Write(src[last:sp.Start])
		out.WriteString("\n" + st.placeholder(h, "") + "\n\n")
		last = sp.End
	}
	if last == 0 {
		return src
	}
	out.Write(src[last:])
	return out.Bytes()
}

// replaceWithDialects merges the spans of the folder's dialects with those of
// `:::` containers in source order; the outermost wins. MyST's colon fences
// end on a bare `:::` line that ScanContainers would take for the end of an
// enclosing container, so those closing lines are blanked (same length) in the
// copy it scans, and the bodies it returns are read from the real source.
func (st *state) replaceWithDialects(src []byte) []byte {
	var ds []dialects.Span
	for _, d := range st.dialects {
		ds = append(ds, d.Scan(src)...)
	}
	if len(ds) == 0 {
		if !bytes.Contains(src, []byte(":::")) {
			return src
		}
		return st.replaceSpans(src, plugins.ScanContainers(src), nil)
	}
	masked, changed := src, false
	for _, sp := range ds {
		first := bytes.TrimLeft(src[sp.Start:], " ")
		ls := bytes.LastIndexByte(src[:sp.End], '\n') + 1
		if len(first) == 0 || first[0] != ':' || ls <= sp.Start {
			continue // not a colon fence
		}
		if !changed {
			masked, changed = append([]byte(nil), src...), true
		}
		for i := ls; i < sp.End; i++ {
			masked[i] = ' '
		}
	}
	var zen []plugins.Span
	if bytes.Contains(masked, []byte(":::")) {
		zen = plugins.ScanContainers(masked)
		if changed {
			for i := range zen {
				zen[i].Body = bodyOf(src, zen[i])
			}
		}
	}
	dm := make(map[int]dialects.Span, len(ds))
	all := make([]plugins.Span, 0, len(zen)+len(ds))
	all = append(all, zen...)
	for _, d := range ds {
		all = append(all, d.Span)
		dm[d.Start] = d
	}
	sort.SliceStable(all, func(i, j int) bool {
		if all[i].Start != all[j].Start {
			return all[i].Start < all[j].Start
		}
		return all[i].End > all[j].End // the outer one first
	})
	return st.replaceSpans(src, all, dm)
}

// bodyOf is the text between the opening and closing lines of a container.
func bodyOf(src []byte, sp plugins.Span) string {
	nl := bytes.IndexByte(src[sp.Start:], '\n')
	closeAt := bytes.LastIndexByte(src[:sp.End], '\n') + 1
	if nl < 0 || sp.Start+nl+1 > closeAt {
		return ""
	}
	return string(src[sp.Start+nl+1 : closeAt])
}

// codeFenceText is a code directive as an ordinary fenced block, long enough
// that its body cannot end it.
func codeFenceText(d dialects.Span) string {
	longest, cur := 0, 0
	for i := 0; i < len(d.Body); i++ {
		if d.Body[i] == '`' {
			cur++
			if cur > longest {
				longest = cur
			}
		} else {
			cur = 0
		}
	}
	fence := strings.Repeat("`", max(3, longest+1))
	body := d.Body
	if body != "" && !strings.HasSuffix(body, "\n") {
		body += "\n"
	}
	return fence + d.Lang + "\n" + body + fence
}

// replaceInlines replaces inline directives of enabled plugins outside
// fenced code blocks with placeholder tokens.
func (st *state) replaceInlines(src []byte) []byte {
	known := func(n string) bool { return st.set.Inline(n) != nil || st.set.Denied("inline", n) != "" }
	var scanners []dialects.InlineScanner
	for _, d := range st.dialects {
		if sc, ok := d.(dialects.InlineScanner); ok {
			scanners = append(scanners, sc)
		}
	}
	if bytes.IndexByte(src, ':') < 0 && (len(scanners) == 0 || bytes.IndexAny(src, "{$") < 0) {
		return src
	}
	var out bytes.Buffer
	var fenceChar byte
	fenceLen := 0
	for pos := 0; pos < len(src); {
		next := len(src)
		if nl := bytes.IndexByte(src[pos:], '\n'); nl >= 0 {
			next = pos + nl + 1
		}
		line := src[pos:next]
		pos = next
		if c, n, rest := codeFence(bytes.TrimRight(line, " \t\r\n")); n > 0 {
			if fenceChar == 0 && (c != '`' || !bytes.Contains(rest, []byte("`"))) {
				fenceChar, fenceLen = c, n
			} else if fenceChar == c && n >= fenceLen && len(rest) == 0 {
				fenceChar = 0
			}
			out.Write(line)
			continue
		}
		if fenceChar != 0 {
			out.Write(line)
			continue
		}
		at := 0
		spans, roles := st.inlineSpans(line, known, scanners)
		for _, sp := range spans {
			if sp.Start < at {
				continue // inside the one before
			}
			ip := st.set.Inline(sp.Name)
			if ip == nil { // a plugin that is not allowed or a role nobody owns: the text stays literal
				if id := st.set.Denied("inline", sp.Name); id != "" {
					st.deny(id)
				} else if roles[sp.Start] && line[sp.Start] == '{' && !st.titleOnly {
					st.note("myst: the role {" + sp.Name + "} is not supported (no plugin owns it) and stays as text")
				}
				continue
			}
			h := st.inline(ip, plugins.Inline{Name: sp.Name, Text: sp.Body, Attrs: sp.Attrs})
			out.Write(line[at:sp.Start])
			out.WriteString(st.placeholder(h, sp.Body))
			at = sp.End
		}
		out.Write(line[at:])
	}
	return out.Bytes()
}

// inlineSpans are the inline directives of one line: zen's :name[text] and
// the dialects' roles, in order. roles holds the starts of the dialects' spans.
func (st *state) inlineSpans(line []byte, known func(string) bool, scanners []dialects.InlineScanner) ([]plugins.Span, map[int]bool) {
	spans := plugins.ScanInlines(line, known)
	if len(scanners) == 0 {
		return spans, nil
	}
	roles := map[int]bool{}
	var ext dialects.Extensions
	if bytes.IndexByte(line, '$') >= 0 {
		ext = st.extensions()
	}
	for _, sc := range scanners {
		for _, sp := range sc.ScanInline(line, ext) {
			roles[sp.Start] = true
			spans = append(spans, sp)
		}
	}
	sort.SliceStable(spans, func(i, j int) bool { return spans[i].Start < spans[j].Start })
	return spans, roles
}

// codeFence mirrors plugins' fence detection: up to three spaces, then three
// or more backticks or tildes.
func codeFence(line []byte) (c byte, n int, rest []byte) {
	i := 0
	for i < len(line) && i < 3 && line[i] == ' ' {
		i++
	}
	if i >= len(line) || (line[i] != '`' && line[i] != '~') {
		return 0, 0, nil
	}
	c = line[i]
	j := i
	for j < len(line) && line[j] == c {
		j++
	}
	if j-i < 3 {
		return 0, 0, nil
	}
	return c, j - i, bytes.TrimSpace(line[j:])
}

// fill swaps placeholder tokens for their fragments.
func (st *state) fill(s string) string {
	if len(st.frags) == 0 {
		return s
	}
	pairs := make([]string, 0, 4*len(st.frags))
	for i, f := range st.frags {
		t := st.token(i)
		pairs = append(pairs, "<p>"+t+"</p>", f, t, f)
	}
	return strings.NewReplacer(pairs...).Replace(s)
}

// plain swaps placeholder tokens for their plain text.
func (st *state) plain(s string) string {
	if len(st.frags) == 0 {
		return s
	}
	pairs := make([]string, 0, 2*len(st.frags))
	for i, t := range st.texts {
		pairs = append(pairs, st.token(i), t)
	}
	return strings.NewReplacer(pairs...).Replace(s)
}
