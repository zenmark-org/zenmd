package figure

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/zenmark-org/zenmd/builtins/dialects/myst"
	"github.com/zenmark-org/zenmd/engine/config"
	"github.com/zenmark-org/zenmd/engine/dialects"
	"github.com/zenmark-org/zenmd/engine/plugins"
	"github.com/zenmark-org/zenmd/engine/render"
	"github.com/zenmark-org/zenmd/engine/workspace"
	xhtml "golang.org/x/net/html"
)

// folder makes a workspace with a few images and returns a renderer with the
// MyST dialect on, the way a MyST folder renders.
func folder(t *testing.T) (*render.Renderer, string) {
	t.Helper()
	dir := t.TempDir()
	for _, f := range []string{"img/a.png", "img/b.png", "docs/c.png", "other.md"} {
		p := filepath.Join(dir, f)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, ".zenmd.yml"), []byte("markdown:\n  dialects: [myst]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	ws, err := workspace.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	reg := plugins.NewRegistry()
	if err := reg.Register(New()); err != nil {
		t.Fatal(err)
	}
	dr, err := dialects.NewRegistry(myst.New())
	if err != nil {
		t.Fatal(err)
	}
	return render.New(reg, config.Load("", dir), ws).WithDialects(dr), dir
}

func page(t *testing.T, r *render.Renderer, path, src string) render.Page {
	t.Helper()
	p, err := r.Render(path, []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func contains(t *testing.T, s string, want ...string) {
	t.Helper()
	for _, w := range want {
		if !strings.Contains(s, w) {
			t.Errorf("missing %q in\n%s", w, s)
		}
	}
}

func lacks(t *testing.T, s string, bad ...string) {
	t.Helper()
	for _, b := range bad {
		if strings.Contains(s, b) {
			t.Errorf("unexpected %q in\n%s", b, s)
		}
	}
}

// inert fails the test when h has a script element or an event-handler attribute.
func inert(t *testing.T, h string) {
	t.Helper()
	doc, err := xhtml.Parse(strings.NewReader(h))
	if err != nil {
		t.Fatal(err)
	}
	var walk func(n *xhtml.Node)
	walk = func(n *xhtml.Node) {
		if n.Type == xhtml.ElementNode {
			switch n.Data {
			case "script", "svg", "iframe", "style", "object", "embed":
				t.Errorf("hostile element <%s> in\n%s", n.Data, h)
			}
			for _, a := range n.Attr {
				if strings.HasPrefix(strings.ToLower(a.Key), "on") || strings.HasPrefix(strings.ToLower(a.Val), "javascript:") {
					t.Errorf("hostile attribute %s=%q in\n%s", a.Key, a.Val, h)
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
}

var attrsRE = regexp.MustCompile(` data-zen-attrs="[^"]*"`)

func stripAttrs(s string) string { return attrsRE.ReplaceAllString(s, "") }

const mystFigure = "```{figure} img/a.png\n:alt: A chart\n:width: 50%\n:align: left\n:name: my-fig\n\nThe caption with **bold**.\n\nThe legend paragraph.\n```\n"

const colonFigure = ":::figure{src=img/a.png alt=\"A chart\" width=50% align=left name=my-fig}\nThe caption with **bold**.\n\nThe legend paragraph.\n:::\n"

func TestManifest(t *testing.T) {
	p := New()
	m := p.Manifest()
	if m.ID != "figure" || !m.Default || m.APIVersion != plugins.APIVersion || m.Version == "" {
		t.Fatalf("manifest %+v", m)
	}
	if strings.Join(m.Blocks, ",") != "figure,image" {
		t.Errorf("blocks %v", m.Blocks)
	}
	if _, ok := p.(plugins.BlockHook); !ok {
		t.Error("not a BlockHook")
	}
	if _, ok := p.(plugins.HTMLHook); !ok {
		t.Error("not an HTMLHook (numbering)")
	}
	ap, ok := p.(plugins.AssetProvider)
	if !ok {
		t.Fatal("not an AssetProvider")
	}
	if e := ap.Entry(); strings.Join(e.CSS, ",") != "figure.css" || len(e.JS) != 0 {
		t.Errorf("entry %+v", e)
	}
	if len(m.Help.Snippets) == 0 || !strings.Contains(m.Help.Summary, "figure") {
		t.Errorf("help %+v", m.Help)
	}
	var colon, myst bool
	for _, s := range m.Help.Snippets {
		colon = colon || strings.Contains(s.Markdown, ":::figure")
		myst = myst || strings.Contains(s.Markdown, "{figure}")
	}
	if !colon || !myst {
		t.Error("help shows both the ::: form and the MyST form")
	}
}

func TestFigure(t *testing.T) {
	r, _ := folder(t)
	p := page(t, r, "page.md", mystFigure)
	contains(t, p.HTML,
		`data-zen-plugin="figure"`, `class="zen-block`, `<figure id="my-fig">`,
		`<img src="/_ws/img/a.png" alt="A chart" width="50%">`,
		`<figcaption>`, `Figure `, `The caption with <strong>bold</strong>.`, `The legend paragraph.`,
		`zf-left`)
	lacks(t, p.HTML, "zen-plugin-error", "<script")
	if len(p.Warnings) != 0 {
		t.Errorf("warnings %v", p.Warnings)
	}
	if !strings.Contains(p.HTML, `Figure <span class="zf-num">1</span>`) && !strings.Contains(p.HTML, `Figure 1`) {
		t.Errorf("no Figure 1 in %s", p.HTML)
	}
}

// The MyST form and the ::: form render the same.
func TestFormsRenderTheSame(t *testing.T) {
	r, _ := folder(t)
	a := stripAttrs(page(t, r, "page.md", mystFigure).HTML)
	b := stripAttrs(page(t, r, "page.md", colonFigure).HTML)
	if a != b {
		t.Errorf("MyST form and ::: form differ:\n%s\n%s", a, b)
	}
	// the colon fence of MyST too
	c := stripAttrs(page(t, r, "page.md", ":::{figure} img/a.png\n:alt: A chart\n:width: 50%\n:align: left\n:name: my-fig\n\nThe caption with **bold**.\n\nThe legend paragraph.\n:::\n").HTML)
	if a != c {
		t.Errorf("colon fence differs:\n%s\n%s", a, c)
	}
}

// Images resolve the way page images do: folder-relative, the same URL.
func TestImageResolvesLikePageImages(t *testing.T) {
	r, _ := folder(t)
	plain := page(t, r, "docs/p.md", "![x](c.png)\n").HTML
	fig := page(t, r, "docs/p.md", "```{figure} c.png\nCap\n```\n").HTML
	if !strings.Contains(plain, `src="/_ws/docs/c.png"`) || !strings.Contains(fig, `src="/_ws/docs/c.png"`) {
		t.Errorf("plain %s\nfigure %s", plain, fig)
	}
	root := page(t, r, "docs/p.md", "```{figure} /img/a.png\nCap\n```\n").HTML
	contains(t, root, `src="/_ws/img/a.png"`)
	up := page(t, r, "docs/p.md", "```{figure} ../img/a.png\nCap\n```\n").HTML
	contains(t, up, `src="/_ws/img/a.png"`)
}

func TestMissingImage(t *testing.T) {
	r, _ := folder(t)
	p := page(t, r, "page.md", "```{figure} img/nope.png\nA caption\n```\n")
	lacks(t, p.HTML, "<img")
	contains(t, p.HTML, "img/nope.png", "not found", "A caption", "zen-block-status") // the reader sees it
	contains(t, strings.Join(p.Warnings, "|"), `plugin "figure"`, "image not found: img/nope.png")
	out := page(t, r, "page.md", "```{figure} ../../outside.png\nA caption\n```\n").HTML
	lacks(t, out, "<img")
}

func TestExternalAndRefusedSources(t *testing.T) {
	r, _ := folder(t)
	ok := page(t, r, "page.md", "```{image} https://example.com/p.png\n:alt: remote\n```\n").HTML
	contains(t, ok, `<img src="https://example.com/p.png" alt="remote">`)
	for _, bad := range []string{"javascript:alert(1)", "data:image/png;base64,AAAA", "http://example.com/p.png", "file:///etc/passwd"} {
		out := page(t, r, "page.md", "```{image} "+bad+"\n```\n").HTML
		lacks(t, out, "<img")
		inert(t, out)
	}
}

func TestNumbering(t *testing.T) {
	r, _ := folder(t)
	src := "```{figure} img/a.png\nFirst\n```\n\n```{image} img/b.png\n```\n\n```{figure} img/b.png\nSecond\n```\n\n```{figure} img/b.png\n:nonumber:\nThird\n```\n"
	h := page(t, r, "page.md", src).HTML
	if n := strings.Count(h, "Figure "); n != 2 {
		t.Errorf("%d numbered figures, want 2:\n%s", n, h)
	}
	re := regexp.MustCompile(`Figure (?:<span[^>]*>)?(\d+)`)
	var got []string
	for _, m := range re.FindAllStringSubmatch(h, -1) {
		got = append(got, m[1])
	}
	if strings.Join(got, ",") != "1,2" {
		t.Errorf("numbers %v, want 1,2:\n%s", got, h)
	}
	// numbering is per page: a second render starts again, and another page too
	again := page(t, r, "other.md", "```{figure} img/a.png\nOnly\n```\n").HTML
	if !regexp.MustCompile(`Figure (?:<span[^>]*>)?1\b`).MatchString(again) {
		t.Errorf("second page does not start at 1:\n%s", again)
	}
	again = page(t, r, "page.md", src).HTML
	if !strings.Contains(again, ">2<") && !strings.Contains(again, "Figure 2") {
		t.Errorf("re-render lost the numbers:\n%s", again)
	}
	if strings.Contains(again, ">3<") || strings.Contains(again, "Figure 3") {
		t.Errorf("numbers carried over between renders:\n%s", again)
	}
}

func TestImageDirective(t *testing.T) {
	r, _ := folder(t)
	h := page(t, r, "page.md", "```{image} img/a.png\n:alt: Plain\n:height: 40px\n:width: 120\n:align: right\n:name: pic\n```\n").HTML
	contains(t, h, `<img src="/_ws/img/a.png" alt="Plain" width="120" height="40">`, `id="pic"`, "zf-right")
	lacks(t, h, "<figcaption", "<figure", "Figure")
}

func TestTarget(t *testing.T) {
	r, _ := folder(t)
	h := page(t, r, "page.md", "```{figure} img/a.png\n:target: https://example.com/big\nCap\n```\n").HTML
	contains(t, h, `<a href="https://example.com/big"`, `<img `)
	if !regexp.MustCompile(`<a [^>]*><img `).MatchString(h) {
		t.Errorf("the image is not inside the link:\n%s", h)
	}
	in := page(t, r, "page.md", "```{image} img/a.png\n:target: other.md\n```\n").HTML
	contains(t, in, `data-zen-link="internal"`, `data-path="other.md"`)
	bad := page(t, r, "page.md", "```{image} img/a.png\n:target: javascript:alert(1)\n```\n").HTML
	inert(t, bad)
	lacks(t, bad, "href=\"javascript")
}

func TestClassAndName(t *testing.T) {
	r, _ := folder(t)
	h := page(t, r, "page.md", "```{figure} img/a.png\n:class: wide dark\n:name: Fig One\nCap\n```\n").HTML
	contains(t, h, `wide`, `dark`, `id="fig-one"`)
}

func TestOptionsInYAMLBlock(t *testing.T) {
	r, _ := folder(t)
	h := page(t, r, "page.md", "```{figure} img/a.png\n---\nalt: From YAML\nname: yaml-fig\nwidth: 30%\n---\nCap\n```\n").HTML
	contains(t, h, `alt="From YAML"`, `id="yaml-fig"`, `width="30%"`, "Cap")
}

// An option the built-in does not support is shown on the page and ignored.
func TestUnknownOptionsWarn(t *testing.T) {
	r, _ := folder(t)
	pg := page(t, r, "page.md", "```{figure} img/a.png\n:frobnicate: yes\n:align: sideways\n:width: 3cm\nCap\n```\n")
	h, w := pg.HTML, strings.Join(pg.Warnings, "|")
	contains(t, w, "option :frobnicate:", "option :align:", "option :width:")
	lacks(t, w, "sideways", "3cm", "yes")
	lacks(t, h, "zen-block-status", "zen-plugin-error", `width="3cm"`, "sideways\"")
	if !strings.Contains(h, "<img") || !strings.Contains(h, "Cap") {
		t.Errorf("the figure still renders:\n%s", h)
	}
}

func TestNeedsAnImage(t *testing.T) {
	r, _ := folder(t)
	p := page(t, r, "page.md", "```{figure}\nCap\n```\n")
	contains(t, p.HTML, "zen-plugin-error")
	if len(p.Warnings) == 0 {
		t.Error("no warning")
	}
}

// Hostile text stays text.
func TestHostile(t *testing.T) {
	r, _ := folder(t)
	src := "```{figure} img/a.png\n:alt: \"><script>alert(1)</script>\n:name: x\"><script>alert(2)</script>\n:class: a\"onmouseover=\"alert(3) zen-block\n:width: 1\" onerror=\"alert(4)\n:target: \"><script>alert(5)</script>\n<script>alert(6)</script> <img src=x onerror=alert(7)> caption\n```\n\n" +
		"```{image} img/a\" onerror=\"alert(8).png\n```\n\n```{image} <script>alert(9)</script>\n```\n"
	h := page(t, r, "page.md", src).HTML
	inert(t, h)
}

func TestHostileDirectOutput(t *testing.T) {
	h := New().(plugins.BlockHook)
	md := func(s string) (string, error) {
		return "<p>" + strings.ReplaceAll(strings.ReplaceAll(s, "&", "&amp;"), "<", "&lt;") + "</p>\n", nil
	}
	out, err := h.RenderBlock(plugins.Context{Path: "p.md", Markdown: md}, plugins.Block{
		Name: "figure", Body: "<script>x</script>",
		Attrs: map[string]string{"arg": `a" onerror="x`, "name": `n"><b>`, "alt": `"><i>`, "class": `c"><u>`, "width": `1"><s>`, "align": `x"><s>`, "foo\"><s>": "1"},
	})
	if err != nil {
		return // refusing is fine
	}
	inert(t, out)
	lacks(t, out, "<b>", "<i>", "<u>", "<s>")
}

func TestKitClassesAndNoScript(t *testing.T) {
	r, _ := folder(t)
	h := page(t, r, "page.md", mystFigure).HTML
	contains(t, h, `class="zen-block`, `zen-block-body`)
	lacks(t, h, "<script", "<button", "style=")
	// figure and image need no JS
	e := New().(plugins.AssetProvider).Entry()
	if len(e.JS) != 0 {
		t.Errorf("js %v", e.JS)
	}
}

func TestCSSUsesTokensOnly(t *testing.T) {
	b, err := os.ReadFile("assets/figure.css")
	if err != nil {
		t.Fatal(err)
	}
	css := string(b)
	for _, bad := range []string{"#", "rgb(", "rgba(", "hsl(", "--font-body", "outline: none", "button", "input"} {
		if strings.Contains(stripComments(css), bad) {
			t.Errorf("figure.css contains %q", bad)
		}
	}
	for _, want := range []string{"var(--rule", "var(--fg-muted)", "var(--font-ui)"} {
		if !strings.Contains(css, want) {
			t.Errorf("figure.css does not use %s", want)
		}
	}
	if strings.Contains(css, "animation") || strings.Contains(css, "transition") {
		if !strings.Contains(css, "prefers-reduced-motion") {
			t.Error("motion without prefers-reduced-motion")
		}
	}
}

func stripComments(s string) string {
	return regexp.MustCompile(`(?s)/\*.*?\*/`).ReplaceAllString(s, "")
}

// Without the dialect the figure is still there as :::figure.
func TestWithoutMyst(t *testing.T) {
	r, _ := folder(t)
	plainR := render.New(registered(t), nil, nil)
	_ = r
	h, err := plainR.Render("p.md", []byte(":::figure{src=https://example.com/p.png}\nCaption\n:::\n"))
	if err != nil {
		t.Fatal(err)
	}
	contains(t, h.HTML, `<img src="https://example.com/p.png"`, "Caption", "<figcaption>")
}

func registered(t *testing.T) *plugins.Registry {
	t.Helper()
	reg := plugins.NewRegistry()
	if err := reg.Register(New()); err != nil {
		t.Fatal(err)
	}
	return reg
}
