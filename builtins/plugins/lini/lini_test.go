package lini

import (
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/zenmark-org/zenmd/engine/config"
	"github.com/zenmark-org/zenmd/engine/plugins"
	"github.com/zenmark-org/zenmd/engine/render"
	"github.com/zenmark-org/zenmd/engine/workspace"
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

func renderPage(t *testing.T, src string) render.Page {
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
	if !strings.Contains(got, "unknown option :bogus:") {
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

// Behaviour 3a: a figure from a file of the folder (plugins/diagram-files).

func folderRenderer(t *testing.T, files map[string]string) *render.Renderer {
	t.Helper()
	dir := t.TempDir()
	files[".zenmd.yml"] = "plugins:\n  lini: {}\n"
	for name, body := range files {
		p := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	ws, err := workspace.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	reg := plugins.NewRegistry()
	if err := reg.Register(New()); err != nil {
		t.Fatal(err)
	}
	return render.New(reg, config.Load("", dir), ws)
}

func renderAt(t *testing.T, r *render.Renderer, page, src string) render.Page {
	t.Helper()
	p, err := r.Render(page, []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func pre(h string) string {
	i, j := strings.Index(h, "<pre>"), strings.Index(h, "</pre>")
	if i < 0 || j < i {
		return ""
	}
	return h[i+5 : j]
}

func TestFileThreeForms(t *testing.T) {
	r := folderRenderer(t, map[string]string{"figures/a.lini": "|box#a| \"A\"\r\na -> a\r\n"})
	want := "|box#a| &#34;A&#34;\na -&gt; a\n"
	for name, src := range map[string]string{
		"braces":  "```lini {file=figures/a.lini}\n```\n",
		"quoted":  "```lini {file=\"figures/a.lini\"}\n```\n",
		"arg":     "```lini figures/a.lini\n```\n",
		"line":    "```lini\n:file: figures/a.lini\n```\n",
		"colon":   ":::lini{file=figures/a.lini}\n:::\n",
		"colonln": ":::lini\n:file: figures/a.lini\n:::\n",
	} {
		p := renderAt(t, r, "page.md", src)
		if got := pre(p.HTML); got != want {
			t.Errorf("%s: pre %q, want %q in %s", name, got, want, p.HTML)
		}
		if !strings.Contains(p.HTML, `class="zen-lini"`) || strings.Contains(p.HTML, "zen-plugin-error") || len(p.Warnings) != 0 {
			t.Errorf("%s: %s %v", name, p.HTML, p.Warnings)
		}
	}
}

func TestFileRootPath(t *testing.T) {
	r := folderRenderer(t, map[string]string{"figures/a.lini": "root file", "a/b/figures/x.lini": "near file"})
	for src, want := range map[string]string{
		"```lini /figures/a.lini\n```\n":        "root file",
		"```lini figures/x.lini\n```\n":         "near file",
		"```lini ../../figures/a.lini\n```\n":   "root file",
		"```lini {file=/figures/a.lini}\n```\n": "root file",
	} {
		p := renderAt(t, r, "a/b/page.md", src)
		if got := pre(p.HTML); got != want {
			t.Errorf("%q: %q, want %q (%v)", src, got, want, p.Warnings)
		}
	}
}

func TestFileMissing(t *testing.T) {
	r := folderRenderer(t, map[string]string{"x.md": ""})
	for src, msg := range map[string]string{
		"```lini gone.lini\n```\n":           "gone.lini was not found in the folder",
		"```lini ../../etc/passwd\n```\n":    "../../etc/passwd is outside the folder",
		"```lini\n:file: nope/x.lini\n```\n": "nope/x.lini was not found in the folder",
	} {
		p := renderAt(t, r, "page.md", src)
		if !strings.Contains(p.HTML, `class="zen-plugin-error"`) || !strings.Contains(p.HTML, "lini</strong>: "+msg) {
			t.Errorf("%q: %s", src, p.HTML)
		}
		if len(p.Warnings) != 1 || !strings.Contains(p.Warnings[0], msg) {
			t.Errorf("%q: warnings %v", src, p.Warnings)
		}
		if strings.Contains(p.HTML, `<div class="zen-lini"`) {
			t.Errorf("%q: drew a figure", src)
		}
	}
	// no folder
	reg := plugins.NewRegistry()
	_ = reg.Register(New())
	p := renderAt(t, render.New(reg, nil, nil), "a.md", on+"```lini a.lini\n```\n")
	if !strings.Contains(p.HTML, "needs an open folder") || len(p.Warnings) != 1 {
		t.Errorf("no folder: %s %v", p.HTML, p.Warnings)
	}
}

func TestFileAndBody(t *testing.T) {
	r := folderRenderer(t, map[string]string{"a.lini": "FROM FILE"})
	const note = "the block's own text is ignored because it names a file"
	for name, src := range map[string]string{
		"line":   "```lini\n:file: a.lini\n|box| \"x\"\n```\n",
		"braces": "```lini {file=a.lini}\n|box| \"x\"\n```\n",
		"arg":    "```lini a.lini\n|box| \"x\"\n```\n",
	} {
		p := renderAt(t, r, "page.md", src)
		if got := pre(p.HTML); got != "FROM FILE" {
			t.Errorf("%s: file must win, got %q", name, got)
		}
		if len(p.Warnings) != 1 || !strings.Contains(p.Warnings[0], note) {
			t.Errorf("%s: warnings %v", name, p.Warnings)
		}
	}
	if p := renderAt(t, r, "page.md", "```lini a.lini\n\n  \n```\n"); len(p.Warnings) != 0 {
		t.Errorf("blank body warned: %v", p.Warnings)
	}
}

func TestFileEmpty(t *testing.T) {
	r := folderRenderer(t, map[string]string{"e.lini": "", "w.lini": " \n\r\n"})
	for _, f := range []string{"e.lini", "w.lini"} {
		p := renderAt(t, r, "page.md", "```lini "+f+"\n```\n")
		if strings.Contains(p.HTML, "zen-lini") || strings.Contains(p.HTML, "<pre>") {
			t.Errorf("%s: drew something: %s", f, p.HTML)
		}
		if len(p.Warnings) != 1 || !strings.Contains(p.Warnings[0], f+" is empty") {
			t.Errorf("%s: warnings %v", f, p.Warnings)
		}
	}
}

func TestArgAndTitleAreKnown(t *testing.T) {
	r := folderRenderer(t, map[string]string{"a.lini": "x"})
	for _, src := range []string{"```lini a.lini\n```\n", "```lini {title=\"T\" file=a.lini}\n```\n"} {
		if p := renderAt(t, r, "page.md", src); len(p.Warnings) != 0 {
			t.Errorf("%q: %v", src, p.Warnings)
		}
	}
}

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
