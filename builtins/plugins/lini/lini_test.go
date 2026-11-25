package lini

import (
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"testing"

	"github.com/zenmark-org/zenmd/engine/plugins"
	"github.com/zenmark-org/zenmd/engine/render"
)

const on = "---\nplugins:\n  lini: {}\n---\n"

func newRenderer(t *testing.T) *render.Renderer {
	t.Helper()
	reg := plugins.NewRegistry()
	if err := reg.Register(New()); err != nil {
		t.Fatal(err)
	}
	return render.New(reg, nil, nil)
}

func renderPage(t *testing.T, src string) *render.Page {
	t.Helper()
	p, err := newRenderer(t).Render("a.md", []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestManifest(t *testing.T) {
	m := New().Manifest()
	if m.ID != "lini" || m.Default || m.APIVersion != plugins.APIVersion || m.Version == "" {
		t.Fatalf("manifest %+v", m)
	}
	if strings.Join(m.Blocks, ",") != "lini" {
		t.Errorf("blocks %v", m.Blocks)
	}
	if len(m.Config) != 0 {
		t.Errorf("no options expected: %v", m.Config)
	}
	if _, ok := New().(plugins.BlockHook); !ok {
		t.Error("not a BlockHook")
	}
	e := New().(plugins.AssetProvider).Entry()
	if strings.Join(e.JS, ",") != "lini.js" || strings.Join(e.CSS, ",") != "lini.css" {
		t.Errorf("entry %+v", e)
	}
}

func TestOffByDefault(t *testing.T) {
	p := renderPage(t, "```lini\n|box#a| \"A\"\n```\n")
	if strings.Contains(p.HTML, "zen-lini") || len(p.Assets.JS) != 0 {
		t.Errorf("rendered while off: %s %+v", p.HTML, p.Assets)
	}
}

func TestRenderBlockFencedAndContainer(t *testing.T) {
	h := New().(plugins.BlockHook)
	want := `<div class="zen-lini"><pre>|box#a| &#34;A&#34;</pre></div>`
	for _, fenced := range []bool{true, false} {
		out, err := h.RenderBlock(plugins.Context{}, plugins.Block{Name: "lini", Body: `|box#a| "A"`, Fenced: fenced})
		if err != nil || out != want {
			t.Errorf("fenced=%v: %q, %v", fenced, out, err)
		}
	}
	// through the renderer: a fence and a container
	for _, src := range []string{on + "```lini\n|box#a| \"A\"\n```\n", on + ":::lini\n|box#a| \"A\"\n:::\n"} {
		p := renderPage(t, src)
		if !strings.Contains(p.HTML, "zen-lini") || !strings.Contains(p.HTML, "|box#a|") {
			t.Errorf("not rendered: %s", p.HTML)
		}
		if strings.Join(p.Assets.JS, ",") != "/_plugins/lini/lini.js" || strings.Join(p.Assets.CSS, ",") != "/_plugins/lini/lini.css" {
			t.Errorf("assets %+v", p.Assets)
		}
	}
}

func TestRenderBlockEscapes(t *testing.T) {
	out, err := New().(plugins.BlockHook).RenderBlock(plugins.Context{}, plugins.Block{Name: "lini", Body: `|box| "<script>alert(1)</script> & "q""`})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "<script") || !strings.Contains(out, "&lt;script&gt;") || !strings.Contains(out, "&amp;") {
		t.Errorf("not escaped: %s", out)
	}
	// through the sanitiser too
	p := renderPage(t, on+"```lini\n|box| \"<script>alert(1)</script>\"\n```\n")
	if strings.Contains(p.HTML, "<script") {
		t.Errorf("script in page: %s", p.HTML)
	}
}

func TestEmptyBlock(t *testing.T) {
	out, err := New().(plugins.BlockHook).RenderBlock(plugins.Context{}, plugins.Block{Name: "lini"})
	if err != nil || out != `<div class="zen-lini"><pre></pre></div>` {
		t.Errorf("%q %v", out, err)
	}
}

func TestUnknownOptionWarns(t *testing.T) {
	p := renderPage(t, on+"```lini {bogus=1 file=a.lini}\n|box| \"A\"\n```\n")
	var got string
	for _, w := range p.Warnings {
		got += w + "|"
	}
	if !strings.Contains(got, "lini: unknown option :bogus:") {
		t.Errorf("warnings %v", p.Warnings)
	}
	if strings.Contains(got, ":file:") {
		t.Errorf("file is a known option: %v", p.Warnings)
	}
	if p := renderPage(t, on+"```lini\n|box| \"A\"\n```\n"); len(p.Warnings) != 0 {
		t.Errorf("warnings on a plain block: %v", p.Warnings)
	}
	// no Warn func: no panic
	if _, err := New().(plugins.BlockHook).RenderBlock(plugins.Context{}, plugins.Block{Name: "lini", Attrs: map[string]string{"x": "1"}}); err != nil {
		t.Fatal(err)
	}
}

// Behaviour 3a: needs plugins/diagram-files (FileRef, Context.ReadFile).
func TestFileThreeForms(t *testing.T) { t.Skip("needs plugins/diagram-files") }
func TestFileRootPath(t *testing.T)   { t.Skip("needs plugins/diagram-files") }
func TestFileMissing(t *testing.T)    { t.Skip("needs plugins/diagram-files") }
func TestFileAndBody(t *testing.T)    { t.Skip("needs plugins/diagram-files") }
func TestFileEmpty(t *testing.T)      { t.Skip("needs plugins/diagram-files") }

func TestAssetsListed(t *testing.T) {
	fsys := New().(plugins.AssetProvider).Assets()
	for _, f := range []string{"lini.js", "lini.css", "lini_wasm.js", "lini_wasm_bg.wasm"} {
		if _, err := fs.Stat(fsys, f); err != nil {
			t.Errorf("asset %s: %v", f, err)
		}
	}
	// every vendored file in the folder is shipped, so a built site copies it
	disk, err := os.ReadDir("assets")
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range disk {
		if strings.HasSuffix(e.Name(), ".test.ts") {
			continue
		}
		if _, err := fs.Stat(fsys, e.Name()); err != nil {
			t.Errorf("assets/%s is not in Assets(): %v", e.Name(), err)
		}
	}
}

func TestVendoredHashes(t *testing.T) {
	doc, err := os.ReadFile("VENDOR.md")
	if err != nil {
		t.Fatal(err)
	}
	rows := regexp.MustCompile("`assets/([^`]+)`[^\n]*`([0-9a-f]{64})`").FindAllSubmatch(doc, -1)
	if len(rows) < 5 {
		t.Fatalf("VENDOR.md lists %d files", len(rows))
	}
	fsys := New().(plugins.AssetProvider).Assets()
	for _, r := range rows {
		b, err := fs.ReadFile(fsys, string(r[1]))
		if err != nil {
			t.Errorf("%s: %v", r[1], err)
			continue
		}
		sum := sha256.Sum256(b)
		if got := hex.EncodeToString(sum[:]); got != string(r[2]) {
			t.Errorf("%s: sha256 %s, VENDOR.md says %s", r[1], got, r[2])
		}
	}
}

var snippetRE = regexp.MustCompile("(?s)```lini\n(.*?)```")

func TestHelpSnippets(t *testing.T) {
	h := New().Manifest().Help
	if err := plugins.CheckHelp(h); err != nil {
		t.Fatal(err)
	}
	if h.Summary == "" || len(h.Snippets) != 3 {
		t.Fatalf("help %+v", h)
	}
	for _, s := range h.Snippets {
		m := snippetRE.FindStringSubmatch(s.Markdown)
		if m == nil || strings.TrimSpace(m[1]) == "" {
			t.Errorf("%s: no lini block in %q", s.Title, s.Markdown)
			continue
		}
		p := renderPage(t, on+s.Markdown)
		if !strings.Contains(p.HTML, `class="zen-lini"`) || len(p.Warnings) != 0 {
			t.Errorf("%s: %s %v", s.Title, p.HTML, p.Warnings)
		}
	}
}

// The demo page shows the same three snippets as the help.
func TestDemoPageHasSnippets(t *testing.T) {
	b, err := os.ReadFile("../../../examples/demo/plugins/lini.md")
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range New().Manifest().Help.Snippets {
		m := snippetRE.FindStringSubmatch(s.Markdown)
		if m == nil || !strings.Contains(string(b), m[1]) {
			t.Errorf("demo page lacks the %q snippet", s.Title)
		}
	}
}
