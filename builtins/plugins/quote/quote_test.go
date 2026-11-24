package quote

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
)

func renderer(t *testing.T) *render.Renderer {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".zenmd.yml"), []byte("markdown:\n  dialects: [myst]\n"), 0o644); err != nil {
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
	return render.New(reg, config.Load("", dir), nil).WithDialects(dr)
}

func page(t *testing.T, src string) render.Page {
	t.Helper()
	p, err := renderer(t).Render("a.md", []byte(src))
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
	if m.ID != "quote" || !m.Default || m.APIVersion != plugins.APIVersion || m.Version == "" {
		t.Fatalf("manifest %+v", m)
	}
	if strings.Join(m.Blocks, ",") != "epigraph,pull-quote,highlights" {
		t.Errorf("blocks %v", m.Blocks)
	}
	if _, ok := p.(plugins.BlockHook); !ok {
		t.Error("not a BlockHook")
	}
	ap, ok := p.(plugins.AssetProvider)
	if !ok || strings.Join(ap.Entry().CSS, ",") != "quote.css" {
		t.Errorf("assets")
	}
	for _, n := range m.Blocks {
		if !strings.Contains(m.Help.Summary, n) {
			t.Errorf("help summary does not name %s", n)
		}
	}
	var colon, my bool
	for _, s := range m.Help.Snippets {
		colon = colon || strings.Contains(s.Markdown, ":::epigraph") || strings.Contains(s.Markdown, ":::pull-quote")
		my = my || strings.Contains(s.Markdown, "{epigraph}") || strings.Contains(s.Markdown, "{pull-quote}")
	}
	if !colon || !my {
		t.Error("help shows both forms")
	}
	css, _ := fs.ReadFile(ap.Assets(), "quote.css")
	lacks(t, string(css), "#", "rgb(", "hsl(")
}

func TestEpigraph(t *testing.T) {
	body := "Simplicity is prerequisite for reliability.\n\n-- Edsger W. Dijkstra\n"
	a := page(t, "```{epigraph}\n"+body+"```\n")
	contains(t, a.HTML, `data-zen-plugin="quote"`, `class="zen-block zen-quote zen-quote-epigraph`, "<blockquote>",
		"Simplicity is prerequisite for reliability.", `<div class="zen-quote-by">Edsger W. Dijkstra</div>`)
	lacks(t, a.HTML, "zen-plugin-error", "-- Edsger")
	b := page(t, ":::epigraph\n"+body+":::\n")
	c := page(t, ":::{epigraph}\n"+body+":::\n")
	if strip(a.HTML) != strip(b.HTML) || strip(a.HTML) != strip(c.HTML) {
		t.Errorf("forms differ:\n%s\n%s\n%s", strip(a.HTML), strip(b.HTML), strip(c.HTML))
	}
	if len(a.Warnings) != 0 {
		t.Errorf("warnings %v", a.Warnings)
	}
}

func TestAttributionForms(t *testing.T) {
	for _, line := range []string{"-- A. Writer", "--- A. Writer", "— A. Writer"} {
		h := page(t, ":::epigraph\nWords.\n\n"+line+"\n:::\n").HTML
		contains(t, h, `<div class="zen-quote-by">A. Writer</div>`)
	}
	// markdown in the attribution, none without
	h := page(t, ":::epigraph\nWords.\n\n-- *The Book*\n:::\n").HTML
	contains(t, h, "<em>The Book</em>")
	h = page(t, ":::epigraph\nJust words.\n:::\n").HTML
	lacks(t, h, "zen-quote-by")
	// a dash in the middle of text is not an attribution
	h = page(t, ":::epigraph\nWords -- more words.\n:::\n").HTML
	lacks(t, h, "zen-quote-by")
}

func TestPullQuoteAndHighlights(t *testing.T) {
	h := page(t, "```{pull-quote}\nBig words.\n\n-- Someone\n```\n").HTML
	contains(t, h, "zen-quote-pull-quote", "Big words.", "Someone")
	h2 := page(t, ":::pull-quote\nBig words.\n\n-- Someone\n:::\n").HTML
	if strip(h) != strip(h2) {
		t.Errorf("forms differ:\n%s\n%s", strip(h), strip(h2))
	}
	h = page(t, "```{highlights}\n- one\n- two\n```\n").HTML
	contains(t, h, "zen-quote-highlights", "<li>one</li>", "<li>two</li>", "<blockquote>")
}

func TestClassAndName(t *testing.T) {
	h := page(t, "```{epigraph}\n:class: wide narrow\n:name: dijkstra\n\nWords.\n```\n").HTML
	contains(t, h, `id="dijkstra"`, "wide", "narrow")
	h = page(t, ":::epigraph{class=\"zen-evil\" name=\"a b\"}\nWords.\n:::\n").HTML
	lacks(t, h, "zen-evil")
	contains(t, h, `id="a-b"`)
}

func TestUnknownOptionWarns(t *testing.T) {
	p := page(t, "```{epigraph}\n:colour: red\n\nWords.\n```\n")
	contains(t, p.HTML, "Words.")
	w := strings.Join(p.Warnings, "|")
	contains(t, w, `plugin "quote"`, "colour")
	lacks(t, w, "red")
}

func TestHostileStaysText(t *testing.T) {
	h := page(t, "```{epigraph}\n<script>alert(1)</script> <img src=x onerror=alert(1)>\n\n-- <script>alert(2)</script>\n:class: \"><script>\n```\n").HTML
	lacks(t, h, "<script", "<img", "onerror")
}
