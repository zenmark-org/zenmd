package include

import (
	"io/fs"
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
)

const sample = "line zero\nline one\nline two\nline three\nline four\nline five\n"

func folder(t *testing.T) (*render.Renderer, string) {
	t.Helper()
	dir := t.TempDir()
	write := func(name, body string) {
		p := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(".zenmd.yml", "markdown:\n  dialects: [myst]\n")
	write("snippets/part.md", "## Included heading\n\nSome **bold** words and a [link](other.md).\n")
	write("snippets/front.md", "---\ntitle: Hidden\n---\nAfter the frontmatter.\n")
	write("docs/local.md", "Local to docs.\n")
	write("sample.txt", sample)
	write("code/hello.go", "package main\n\nfunc main() {\n\tprintln(\"<hi>\")\n}\n")
	write("code/notes.xyz", "<script>alert(1)</script>\n")
	write("hostile.md", "<script>alert(1)</script>\n\n<img src=x onerror=alert(1)>\n")
	write("self.md", "Self text.\n\n```{include} self.md\n```\n")
	write("ping.md", "Ping.\n\n:::include{file=pong.md}\n:::\n")
	write("pong.md", "Pong.\n\n:::include{file=ping.md}\n:::\n")
	write("page.md", "# x\n")
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

var attrsRE = regexp.MustCompile(` data-zen-attrs="[^"]*"| data-zen-name="[^"]*"`)

func strip(s string) string { return attrsRE.ReplaceAllString(s, "") }

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

func TestManifest(t *testing.T) {
	p := New()
	m := p.Manifest()
	if m.ID != "include" || !m.Default || m.APIVersion != plugins.APIVersion || m.Version == "" {
		t.Fatalf("manifest %+v", m)
	}
	if strings.Join(m.Blocks, ",") != "include,literalinclude" {
		t.Errorf("blocks %v", m.Blocks)
	}
	ap, ok := p.(plugins.AssetProvider)
	if !ok || strings.Join(ap.Entry().CSS, ",") != "include.css" {
		t.Fatal("assets")
	}
	for _, n := range m.Blocks {
		if !strings.Contains(m.Help.Summary, n) {
			t.Errorf("help summary does not name %s", n)
		}
	}
	var colon, my bool
	for _, s := range m.Help.Snippets {
		colon = colon || strings.Contains(s.Markdown, ":::include") || strings.Contains(s.Markdown, ":::literalinclude")
		my = my || strings.Contains(s.Markdown, "{include}") || strings.Contains(s.Markdown, "{literalinclude}")
	}
	if !colon || !my {
		t.Error("help shows both forms")
	}
	css, _ := fs.ReadFile(ap.Assets(), "include.css")
	lacks(t, string(css), "#", "rgb(", "hsl(")
}

func TestInclude(t *testing.T) {
	r, _ := folder(t)
	a := page(t, r, "page.md", "Before.\n\n```{include} snippets/part.md\n```\n\nAfter.\n")
	contains(t, a.HTML, `data-zen-plugin="include"`, "<h2", "Included heading", "<strong>bold</strong>", "Before.", "After.")
	lacks(t, a.HTML, "zen-plugin-error")
	if len(a.Warnings) != 0 {
		t.Errorf("warnings %v", a.Warnings)
	}
	b := page(t, r, "page.md", "Before.\n\n:::include{file=snippets/part.md}\n:::\n\nAfter.\n")
	c := page(t, r, "page.md", "Before.\n\n:::{include} snippets/part.md\n:::\n\nAfter.\n")
	if strip(a.HTML) != strip(b.HTML) || strip(a.HTML) != strip(c.HTML) {
		t.Errorf("forms differ:\n%s\n%s\n%s", strip(a.HTML), strip(b.HTML), strip(c.HTML))
	}
	// frontmatter of the included file is not shown
	h := page(t, r, "page.md", "```{include} snippets/front.md\n```\n").HTML
	contains(t, h, "After the frontmatter.")
	lacks(t, h, "Hidden", "title:")
	// next to the page first, then from the root
	h = page(t, r, "docs/page.md", "```{include} local.md\n```\n").HTML
	contains(t, h, "Local to docs.")
	h = page(t, r, "docs/page.md", "```{include} snippets/part.md\n```\n").HTML
	contains(t, h, "Included heading")
}

func TestIncludeLines(t *testing.T) {
	r, _ := folder(t)
	h := page(t, r, "page.md", "```{include} sample.txt\n:start-line: 1\n:end-line: 3\n```\n").HTML
	contains(t, h, "line one", "line two")
	lacks(t, h, "line zero", "line three")
	h = page(t, r, "page.md", "```{include} sample.txt\n:lines: 1-2,5\n```\n").HTML
	contains(t, h, "line zero", "line one", "line four")
	lacks(t, h, "line two", "line three", "line five")
	h = page(t, r, "page.md", "```{include} sample.txt\n:start-after: line one\n:end-before: line four\n```\n").HTML
	contains(t, h, "line two", "line three")
	lacks(t, h, "line one<", "line four", "line zero")
}

func TestIncludeOptions(t *testing.T) {
	r, _ := folder(t)
	h := page(t, r, "page.md", "```{include} snippets/part.md\n:class: wide\n:name: part\n```\n").HTML
	contains(t, h, `id="part"`, "wide")
	p := page(t, r, "page.md", "```{include} snippets/part.md\n:heading-offset: 1\n```\n")
	contains(t, p.HTML, "Included heading")
	w := strings.Join(p.Warnings, "|")
	contains(t, w, `plugin "include"`, "heading-offset")
	// a bad :lines: warns and includes everything
	p = page(t, r, "page.md", "```{include} sample.txt\n:lines: x-y\n```\n")
	contains(t, p.HTML, "line zero", "line five")
	contains(t, strings.Join(p.Warnings, "|"), "lines")
	// no file named
	h = page(t, r, "page.md", "```{include}\n```\n").HTML
	contains(t, h, "zen-plugin-error")
}

func TestHostileIncludeStaysText(t *testing.T) {
	r, _ := folder(t)
	h := page(t, r, "page.md", "```{include} hostile.md\n```\n").HTML
	lacks(t, h, "<script", "<img", "onerror")
	h = page(t, r, "page.md", "```{include} snippets/part.md\n:class: \"><script>\n```\n").HTML
	lacks(t, h, "<script")
}

func TestLiteralInclude(t *testing.T) {
	r, _ := folder(t)
	a := page(t, r, "page.md", "```{literalinclude} code/hello.go\n```\n")
	contains(t, a.HTML, `data-zen-plugin="include"`, `class="zen-block zen-include zen-include-code`, `<code class="language-go">`,
		"package main", "func main()", "&lt;hi&gt;")
	lacks(t, a.HTML, "<hi>", "zen-plugin-error")
	b := page(t, r, "page.md", ":::literalinclude{file=code/hello.go}\n:::\n")
	c := page(t, r, "page.md", ":::{literalinclude} code/hello.go\n:::\n")
	if strip(a.HTML) != strip(b.HTML) || strip(a.HTML) != strip(c.HTML) {
		t.Errorf("forms differ:\n%s\n%s\n%s", strip(a.HTML), strip(b.HTML), strip(c.HTML))
	}
	// :language: wins over the extension; none known gives no language class
	h := page(t, r, "page.md", "```{literalinclude} code/hello.go\n:language: text\n```\n").HTML
	contains(t, h, `<code class="language-text">`)
	h = page(t, r, "page.md", "```{literalinclude} code/notes.xyz\n```\n").HTML
	contains(t, h, "&lt;script&gt;alert(1)&lt;/script&gt;")
	lacks(t, h, "<script", "language-")
	// a language that is not a plain word is dropped with a warning
	p := page(t, r, "page.md", "```{literalinclude} code/hello.go\n:language: \"><x\n```\n")
	lacks(t, p.HTML, "language-\"", "<x")
	contains(t, strings.Join(p.Warnings, "|"), "language")
	// caption
	h = page(t, r, "page.md", "```{literalinclude} code/hello.go\n:caption: hello.go\n```\n").HTML
	contains(t, h, `<div class="zen-block-head">hello.go</div>`)
}

func TestLiteralIncludeLines(t *testing.T) {
	r, _ := folder(t)
	h := page(t, r, "page.md", "```{literalinclude} sample.txt\n:start-line: 1\n:end-line: 3\n```\n").HTML
	contains(t, h, "line one", "line two")
	lacks(t, h, "line zero", "line three")
	h = page(t, r, "page.md", "```{literalinclude} sample.txt\n:lines: 1-2,5\n```\n").HTML
	contains(t, h, "line zero", "line one", "line four")
	lacks(t, h, "line two", "line three", "line five")
	h = page(t, r, "page.md", "```{literalinclude} sample.txt\n:lines: 3-\n```\n").HTML
	contains(t, h, "line two", "line five")
	lacks(t, h, "line one")
	// line numbers: 1.. of the shown lines, :lineno-start: moves them, a gutter that is not text
	h = page(t, r, "page.md", "```{literalinclude} sample.txt\n:lines: 2-3\n:linenos:\n```\n").HTML
	contains(t, h, "zen-include-numbered", `<span class="zen-include-n" aria-hidden="true">1</span>`, `<span class="zen-include-n" aria-hidden="true">2</span>`)
	lacks(t, h, ">3</span>")
	h = page(t, r, "page.md", "```{literalinclude} sample.txt\n:lines: 2-3\n:linenos:\n:lineno-start: 10\n```\n").HTML
	contains(t, h, `aria-hidden="true">10</span>`, `aria-hidden="true">11</span>`)
	h = page(t, r, "page.md", "```{literalinclude} sample.txt\n:lines: 2-3\n```\n").HTML
	lacks(t, h, "zen-include-n", "zen-include-numbered")
	p := page(t, r, "page.md", "```{literalinclude} sample.txt\n:emphasize-lines: 1\n```\n")
	contains(t, p.HTML, "line zero")
	contains(t, strings.Join(p.Warnings, "|"), `plugin "include"`, "emphasize-lines")
}

// The folder read is the same as the table's: workspace-relative, no .., no
// absolute path, no symlink out (a dangling one included), at most 1 MB, and
// no message ever names a path.
func TestConfinement(t *testing.T) {
	r, dir := folder(t)
	outside := t.TempDir()
	secret := filepath.Join(outside, "secret.md")
	if err := os.WriteFile(secret, []byte("TOPSECRET\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "sibling.md"), []byte("INSIDE\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for name, target := range map[string]string{"link.md": secret, "linkdir": outside, "dangling.md": filepath.Join(outside, "gone.md"), "inlink.md": filepath.Join(dir, "sibling.md")} {
		if err := os.Symlink(target, filepath.Join(dir, name)); err != nil {
			t.Fatal(err)
		}
	}
	big := []byte(strings.Repeat("a", 1<<20+1))
	if err := os.WriteFile(filepath.Join(dir, "big.md"), big, 0o644); err != nil {
		t.Fatal(err)
	}
	exact := []byte("OKFILE" + strings.Repeat(" ", 1<<20-6))
	if err := os.WriteFile(filepath.Join(dir, "exact.md"), exact, 0o644); err != nil {
		t.Fatal(err)
	}
	refused := map[string]string{
		"dotdot":        "../" + filepath.Base(outside) + "/secret.md",
		"dotdot inside": "snippets/../sibling.md",
		"dotdot far":    "../../../../../../etc/passwd",
		"absolute":      secret,
		"absolute in":   filepath.Join(dir, "sibling.md"),
		"slash":         "/etc/passwd",
		"symlink file":  "link.md",
		"symlink dir":   "linkdir/secret.md",
		"dangling":      "dangling.md",
		"too big":       "big.md",
		"missing":       "nope.md",
		"directory":     "snippets",
		"backslash":     `..\secret.md`,
		"nul":           "a\x00b.md",
	}
	for _, form := range []string{"include", "literalinclude"} {
		for name, f := range refused {
			src := "```{" + form + "} " + f + "\n```\n"
			p := page(t, r, "page.md", src)
			lacks(t, p.HTML, "TOPSECRET", "INSIDE", "root:")
			if !strings.Contains(p.HTML, "zen-plugin-error") {
				t.Errorf("%s %s: %q was not refused:\n%s", form, name, f, p.HTML)
			}
			w := strings.Join(p.Warnings, "|")
			if !strings.Contains(w, `plugin "include"`) || strings.Contains(w, dir) || strings.Contains(w, outside) {
				t.Errorf("%s %s: want a warning without a real path, got %q", form, name, w)
			}
			if sh := strip(p.HTML); strings.Contains(sh, dir) || strings.Contains(sh, outside) {
				t.Errorf("%s %s: the error names a real path:\n%s", form, name, p.HTML)
			}
		}
		// :file: is no way round it
		p := page(t, r, "page.md", ":::"+form+"{file=../"+filepath.Base(outside)+"/secret.md}\n:::\n")
		lacks(t, p.HTML, "TOPSECRET")
		contains(t, p.HTML, "zen-plugin-error")
	}
	// a symlink that stays inside the folder is a page read too, and exactly 1 MB is fine
	contains(t, page(t, r, "page.md", "```{include} inlink.md\n```\n").HTML, "INSIDE")
	ok := page(t, r, "page.md", "```{include} exact.md\n```\n").HTML
	contains(t, ok, "OKFILE")
	lacks(t, ok, "zen-plugin-error")
}

func TestWithoutAFolder(t *testing.T) {
	reg := plugins.NewRegistry()
	if err := reg.Register(New()); err != nil {
		t.Fatal(err)
	}
	pg, err := render.New(reg, nil, nil).Render("a.md", []byte(":::include{file=x.md}\n:::\n"))
	if err != nil {
		t.Fatal(err)
	}
	contains(t, strings.Join(pg.Warnings, "|"), `plugin "include"`, "open folder")
	h := New().(plugins.BlockHook)
	if _, err := h.RenderBlock(plugins.Context{Path: "a.md"}, plugins.Block{Name: "include", Attrs: map[string]string{"file": "x.md"}}); err == nil {
		t.Fatal("a file with no folder must fail")
	}
}

// A page that includes itself, and files that include each other, stop with a
// warning instead of running away.
func TestSelfInclusion(t *testing.T) {
	r, _ := folder(t)
	p := page(t, r, "self.md", "Self text.\n\n```{include} self.md\n```\n")
	contains(t, p.HTML, "Self text.")
	contains(t, strings.Join(p.Warnings, "|"), `plugin "include"`, "itself")
	// a file that includes itself, reached from another page, stops at the depth limit
	p = page(t, r, "page.md", "```{include} self.md\n```\n")
	contains(t, p.HTML, "Self text.")
	if len(p.Warnings) == 0 {
		t.Error("an include of itself must warn")
	}
	if n := strings.Count(p.HTML, "Self text."); n > 10 {
		t.Errorf("self text repeated %d times", n)
	}
	// a ring of two
	p = page(t, r, "page.md", "```{include} ping.md\n```\n")
	contains(t, p.HTML, "Ping.", "Pong.")
	if len(p.Warnings) == 0 {
		t.Error("a ring of includes must warn")
	}
}
