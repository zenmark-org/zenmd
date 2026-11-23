package math

import (
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/zenmark-org/zenmd/builtins/dialects/myst"
	"github.com/zenmark-org/zenmd/builtins/plugins/callout"
	"github.com/zenmark-org/zenmd/engine/config"
	"github.com/zenmark-org/zenmd/engine/dialects"
	"github.com/zenmark-org/zenmd/engine/plugins"
	"github.com/zenmark-org/zenmd/engine/render"
	"github.com/zenmark-org/zenmd/engine/workspace"
	xhtml "golang.org/x/net/html"
)

// renderer returns a renderer with the math plugin, and with the MyST dialect
// when withMyst is set.
func renderer(t *testing.T, withMyst bool) *render.Renderer {
	t.Helper()
	reg := plugins.NewRegistry()
	if err := reg.Register(New()); err != nil {
		t.Fatal(err)
	}
	if !withMyst {
		return render.New(reg, nil, nil)
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".zenmd.yml"), []byte("markdown:\n  dialects: [myst]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	ws, err := workspace.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	dr, err := dialects.NewRegistry(myst.New())
	if err != nil {
		t.Fatal(err)
	}
	return render.New(reg, config.Load("", dir), ws).WithDialects(dr)
}

func page(t *testing.T, r *render.Renderer, src string) render.Page {
	t.Helper()
	p, err := r.Render("page.md", []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	return p
}

// mystPage renders src with the MyST dialect switched on for the page.
func mystPage(t *testing.T, src string) render.Page {
	t.Helper()
	return page(t, renderer(t, true), src)
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

var attrsRE = regexp.MustCompile(` data-zen-attrs="[^"]*"`)

func stripAttrs(s string) string { return attrsRE.ReplaceAllString(s, "") }

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
			case "script", "svg", "iframe", "style", "object", "embed", "img":
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

func TestManifest(t *testing.T) {
	p := New()
	m := p.Manifest()
	if m.ID != "math" || !m.Default || m.APIVersion != plugins.APIVersion || m.Version == "" || m.Description == "" {
		t.Fatalf("manifest %+v", m)
	}
	if strings.Join(m.Blocks, ",") != "math" {
		t.Errorf("blocks %v", m.Blocks)
	}
	if len(m.Help.Snippets) < 2 || m.Help.Summary == "" {
		t.Errorf("help %+v", m.Help)
	}
	for _, s := range m.Help.Snippets {
		if s.Title == "" || s.Markdown == "" {
			t.Errorf("snippet %+v", s)
		}
	}
	if _, ok := p.(plugins.BlockHook); !ok {
		t.Error("not a BlockHook")
	}
	ap, ok := p.(plugins.AssetProvider)
	if !ok {
		t.Fatal("not an AssetProvider")
	}
	e := ap.Entry()
	if strings.Join(e.JS, ",") != "math.js" || strings.Join(e.CSS, ",") != "katex.min.css,math.css" {
		t.Errorf("entry %+v", e)
	}
	for _, f := range []string{"math.js", "math.css", "katex.min.js", "katex.min.css", "KATEX-LICENSE", "fonts/KaTeX_Main-Regular.woff2"} {
		if _, err := fs.Stat(ap.Assets(), f); err != nil {
			t.Errorf("asset %s: %v", f, err)
		}
	}
}

const mystEq = "```{math}\n:label: eq:energy\n\nE = mc^2\n```\n"

func TestMystForms(t *testing.T) {
	forms := map[string]string{
		"fence":           mystEq,
		"colon":           ":::{math}\n:label: eq:energy\n\nE = mc^2\n:::\n",
		"tilde":           "~~~{math}\n:label: eq:energy\nE = mc^2\n~~~\n",
		"dollar":          "$$\nE = mc^2\n$$ (eq:energy)\n",
		"dollar one line": "$$ E = mc^2 $$ (eq:energy)\n",
	}
	var first string
	for name, src := range forms {
		p := mystPage(t, src)
		contains(t, p.HTML, `data-zen-plugin="math"`, `class="zen-block zm"`, `id="eq:energy"`,
			`<pre class="zm-tex">E = mc^2</pre>`, `<span class="zm-num">(1)</span>`)
		if len(p.Warnings) != 0 {
			t.Errorf("%s: warnings %v", name, p.Warnings)
		}
		if first == "" {
			first = stripAttrs(p.HTML)
		} else if got := stripAttrs(p.HTML); got != first {
			t.Errorf("%s renders differently:\n%s\n%s", name, got, first)
		}
	}
}

func TestZenForm(t *testing.T) {
	// the same block without MyST: a :::math container
	p := page(t, renderer(t, false), ":::math{label=eq:energy}\nE = mc^2\n:::\n")
	contains(t, p.HTML, `class="zen-block zm"`, `id="eq:energy"`, `<pre class="zm-tex">E = mc^2</pre>`, `<span class="zm-num">(1)</span>`)
	if want := stripAttrs(mystPage(t, mystEq).HTML); stripAttrs(p.HTML) != want {
		t.Errorf("zen form differs:\n%s\n%s", stripAttrs(p.HTML), want)
	}
	// no label: no number and no id
	p = page(t, renderer(t, false), ":::math\nx^2\n:::\n")
	contains(t, p.HTML, `<pre class="zm-tex">x^2</pre>`)
	lacks(t, p.HTML, "zm-num", ` id=`)
}

func TestArgumentIsTheTeX(t *testing.T) {
	p := mystPage(t, "```{math} a^2 + b^2 = c^2\n```\n")
	contains(t, p.HTML, `<pre class="zm-tex">a^2 + b^2 = c^2</pre>`)
	lacks(t, p.HTML, "zm-num")
}

func TestNumbering(t *testing.T) {
	p := mystPage(t, "$$ a $$ (eq:a)\n\ntext\n\n$$ plain $$\n\n```{math}\n:label: eq:b\nb\n```\n\n:::{math}\n:label: Third One\nc\n:::\n")
	h := p.HTML
	contains(t, h, `id="eq:a"`, `id="eq:b"`, `id="third-one"`, `<span class="zm-num">(1)</span>`, `<span class="zm-num">(2)</span>`, `<span class="zm-num">(3)</span>`)
	lacks(t, h, `<span class="zm-num">(4)</span>`, "⟦")
	if n := strings.Count(h, `class="zm-num"`); n != 3 {
		t.Errorf("%d numbers, want 3 (the unlabelled equation has none)", n)
	}
	if strings.Index(h, `id="eq:a"`) > strings.Index(h, `id="eq:b"`) || strings.Index(h, `id="eq:b"`) > strings.Index(h, `id="third-one"`) {
		t.Errorf("numbers are in page order:\n%s", h)
	}
	// a new page starts again at 1
	r := renderer(t, true)
	src := "$$ a $$ (x)\n"
	a, b := page(t, r, src), page(t, r, src)
	contains(t, a.HTML, `(1)`)
	contains(t, b.HTML, `(1)`)
}

func TestNumberingInsideOtherBlocks(t *testing.T) {
	reg := plugins.NewRegistry()
	for _, pl := range []plugins.Plugin{New(), callout.New()} {
		if err := reg.Register(pl); err != nil {
			t.Fatal(err)
		}
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".zenmd.yml"), []byte("markdown:\n  dialects: [myst]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	ws, err := workspace.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	dr, _ := dialects.NewRegistry(myst.New())
	p := page(t, render.New(reg, config.Load("", dir), ws).WithDialects(dr), "$$ a $$ (eq:a)\n\n:::{note}\n$$ b $$ (eq:b)\n:::\n")
	contains(t, p.HTML, `id="eq:a"`, `id="eq:b"`, `(1)`, `(2)`)
}

func TestDuplicateLabelWarns(t *testing.T) {
	p := mystPage(t, "$$ a $$ (eq:x)\n\n$$ b $$ (eq:x)\n")
	w := strings.Join(p.Warnings, "|")
	contains(t, w, `plugin "math"`, "duplicate", "eq:x")
	if n := strings.Count(p.HTML, `id="eq:x"`); n != 1 {
		t.Errorf("%d elements with id eq:x, want 1:\n%s", n, p.HTML)
	}
	contains(t, p.HTML, `(1)`, `(2)`) // both are still numbered
}

func TestNameOption(t *testing.T) {
	p := mystPage(t, "```{math}\n:name: My Eq\nx\n```\n")
	contains(t, p.HTML, `id="my-eq"`, `<pre class="zm-tex">x</pre>`)
	lacks(t, p.HTML, "zm-num")
	if len(p.Warnings) != 0 {
		t.Errorf("warnings %v", p.Warnings)
	}
	// both: the label is the target, and the name is said to be ignored
	p = mystPage(t, "```{math}\n:label: l1\n:name: n1\nx\n```\n")
	contains(t, p.HTML, `id="l1"`, `(1)`)
	lacks(t, p.HTML, `id="n1"`)
	contains(t, strings.Join(p.Warnings, "|"), `plugin "math"`, "option :name:")
	// a value that makes no id
	p = mystPage(t, "```{math}\n:label: '''\nx\n```\n")
	contains(t, strings.Join(p.Warnings, "|"), "option :label:")
	contains(t, p.HTML, `<pre class="zm-tex">x</pre>`)
}

func TestClassAndNowrap(t *testing.T) {
	p := mystPage(t, "```{math}\n:class: wide narrow\n:nowrap:\nx\n```\n")
	contains(t, p.HTML, `class="zen-block zm zm-nowrap wide narrow"`)
	if len(p.Warnings) != 0 {
		t.Errorf("warnings %v", p.Warnings)
	}
	p = mystPage(t, "```{math}\n:class: zen-block zm-num ok\nx\n```\n")
	contains(t, p.HTML, `class="zen-block zm ok"`)
	contains(t, strings.Join(p.Warnings, "|"), "option :class:")
}

func TestUnknownOptionsWarn(t *testing.T) {
	p := mystPage(t, "```{math}\n:frobnicate: yes\n:label: l\nx\n```\n")
	contains(t, strings.Join(p.Warnings, "|"), `plugin "math"`, "option :frobnicate:")
	contains(t, p.HTML, `<pre class="zm-tex">x</pre>`, `(1)`) // the block still renders
}

func TestEmptyBlock(t *testing.T) {
	p := mystPage(t, "```{math}\n:label: l\n```\n\nafter\n")
	contains(t, p.HTML, "after")
	contains(t, strings.Join(p.Warnings, "|"), "math")
	lacks(t, p.HTML, "zm-tex")
}

func TestHostileText(t *testing.T) {
	tex := `</pre><script>alert(1)</script><img src=x onerror=alert(2)> \href{javascript:alert(3)}{x} &amp;`
	for _, src := range []string{
		"```{math}\n:label: x\" onclick=\"y\n" + tex + "\n```\n",
		":::math{label=\"x\\\" onclick=\\\"y\"}\n" + tex + "\n:::\n",
		"```{math} " + tex + "\n```\n",
	} {
		p := mystPage(t, src)
		inert(t, p.HTML)
		contains(t, p.HTML, "&lt;script&gt;alert(1)&lt;/script&gt;", "&amp;amp;")
		lacks(t, p.HTML, "<script")
	}
	p := mystPage(t, "$$ </pre><script>alert(1)</script> $$ (x)\n")
	inert(t, p.HTML)
}

func TestNoScriptOutput(t *testing.T) {
	p := mystPage(t, "```{math}\n:label: l\nE = mc^2\n```\n")
	// the TeX is readable text in a kit frame, with the number beside it
	contains(t, p.HTML, `class="zen-block zm"`, `<div class="zen-block-body">`, `<div class="zm-row">`, `<pre class="zm-tex">E = mc^2</pre>`)
	lacks(t, p.HTML, "<script", "katex")
}

func TestAssetsOnlyWhenUsed(t *testing.T) {
	r := renderer(t, true)
	p := page(t, r, "# Plain\n\n```go\nx\n```\n\n$ not math $\n")
	if len(p.Assets.JS)+len(p.Assets.CSS) != 0 {
		t.Errorf("assets on a plain page: %+v", p.Assets)
	}
	p = mystPage(t, mystEq)
	if strings.Join(p.Assets.JS, ",") != "/_plugins/math/math.js" ||
		strings.Join(p.Assets.CSS, ",") != "/_plugins/math/katex.min.css,/_plugins/math/math.css" {
		t.Errorf("assets %+v", p.Assets)
	}
}

// The stylesheet loads fonts as url(fonts/…): they must sit next to it, where
// both the app (/_plugins/math/) and a built site (_zen/plugins/math/) serve
// every file of the plugin's assets, subfolders included.
func TestFontsResolveNextToTheStylesheet(t *testing.T) {
	assets := New().(plugins.AssetProvider).Assets()
	css, err := fs.ReadFile(assets, "katex.min.css")
	if err != nil {
		t.Fatal(err)
	}
	urls := regexp.MustCompile(`url\(([^)]*)\)`).FindAllSubmatch(css, -1)
	if len(urls) < 20 {
		t.Fatalf("%d url() in katex.min.css", len(urls))
	}
	seen := map[string]bool{}
	for _, m := range urls {
		u := strings.Trim(string(m[1]), `"'`)
		if !strings.HasPrefix(u, "fonts/") || strings.Contains(u, "..") || strings.Contains(u, "//") {
			t.Errorf("url(%s) does not stay in fonts/", u)
			continue
		}
		seen[u] = true
		if _, err := fs.Stat(assets, u); err != nil {
			// the .woff and .ttf fallbacks are not vendored: only the first url() of each face is fetched
			if !strings.HasSuffix(u, ".woff") && !strings.HasSuffix(u, ".ttf") {
				t.Errorf("font %s: %v", u, err)
			}
		}
	}
	// and the walk a site build makes finds them
	var files []string
	_ = fs.WalkDir(assets, ".", func(p string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			files = append(files, p)
		}
		return nil
	})
	if !strings.Contains(strings.Join(files, " "), "fonts/KaTeX_Main-Regular.woff2") {
		t.Errorf("fonts are not in the asset walk: %v", files)
	}
}

func TestVendorChecksums(t *testing.T) {
	doc, err := os.ReadFile("VENDOR.md")
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{"katex.min.js", "katex.min.css"} {
		m := regexp.MustCompile("`assets/" + regexp.QuoteMeta(f) + "`.*`([0-9a-f]{64})`").FindSubmatch(doc)
		if m == nil {
			t.Fatalf("no checksum for %s in VENDOR.md", f)
		}
		b, err := fs.ReadFile(New().(plugins.AssetProvider).Assets(), f)
		if err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256(b)
		if got := hex.EncodeToString(sum[:]); got != string(m[1]) {
			t.Errorf("%s sha256 %s, VENDOR.md says %s", f, got, m[1])
		}
	}
}

func TestMathJS(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node not found")
	}
	out, err := exec.Command(node, "--test", "assets/math.test.mjs").CombinedOutput()
	if err != nil {
		t.Fatalf("node --test failed: %v\n%s", err, out)
	}
}
