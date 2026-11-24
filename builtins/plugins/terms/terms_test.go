package terms

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

func page(t *testing.T, src string) render.Page {
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
	p, err := render.New(reg, config.Load("", dir), nil).WithDialects(dr).Render("a.md", []byte(src))
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

const gloss = "API\n  Application programming interface.\n\nREST : protocol\n  A **style** of API.\n  Second line.\n\n  A second paragraph.\n"

func TestManifest(t *testing.T) {
	p := New()
	m := p.Manifest()
	if m.ID != "terms" || !m.Default || m.APIVersion != plugins.APIVersion || m.Version == "" {
		t.Fatalf("manifest %+v", m)
	}
	if strings.Join(m.Blocks, ",") != "glossary" || strings.Join(m.Inlines, ",") != "term" {
		t.Errorf("blocks %v inlines %v", m.Blocks, m.Inlines)
	}
	if _, ok := p.(plugins.HTMLHook); !ok {
		t.Error("not an HTMLHook (checks the links)")
	}
	ap, ok := p.(plugins.AssetProvider)
	if !ok || strings.Join(ap.Entry().CSS, ",") != "terms.css" {
		t.Fatal("assets")
	}
	for _, n := range []string{"glossary", "term"} {
		if !strings.Contains(m.Help.Summary, n) {
			t.Errorf("help summary does not name %s", n)
		}
	}
	var colon, my, inl, role bool
	for _, s := range m.Help.Snippets {
		colon = colon || strings.Contains(s.Markdown, ":::glossary")
		my = my || strings.Contains(s.Markdown, "{glossary}")
		inl = inl || strings.Contains(s.Markdown, ":term[")
		role = role || strings.Contains(s.Markdown, "{term}`")
	}
	if !colon || !my || !inl || !role {
		t.Error("help shows both forms of the directive and of the term")
	}
	css, _ := fs.ReadFile(ap.Assets(), "terms.css")
	lacks(t, string(css), "#", "rgb(", "hsl(")
}

func TestGlossary(t *testing.T) {
	a := page(t, "```{glossary}\n"+gloss+"```\n")
	contains(t, a.HTML, `data-zen-plugin="terms"`, `class="zen-block zen-glossary`, "<dl>",
		`<dt id="term-api">API</dt>`, `<dt id="term-rest">REST`, "protocol", "Application programming interface.",
		"<strong>style</strong>", "A second paragraph.")
	lacks(t, a.HTML, "zen-plugin-error")
	if len(a.Warnings) != 0 {
		t.Errorf("warnings %v", a.Warnings)
	}
	b := page(t, ":::glossary\n"+gloss+":::\n")
	c := page(t, ":::{glossary}\n"+gloss+":::\n")
	if strip(a.HTML) != strip(b.HTML) || strip(a.HTML) != strip(c.HTML) {
		t.Errorf("forms differ:\n%s\n%s\n%s", strip(a.HTML), strip(b.HTML), strip(c.HTML))
	}
}

func TestGlossaryTermForms(t *testing.T) {
	// two terms for one definition
	h := page(t, ":::glossary\nfoo\nbar\n  Both mean the same.\n:::\n").HTML
	contains(t, h, `<dt id="term-foo">foo</dt>`, `<dt id="term-bar">bar</dt>`, "Both mean the same.")
	if strings.Count(h, "<dd>") != 1 {
		t.Errorf("one definition:\n%s", h)
	}
	// slugs: case, spaces, punctuation, non-ASCII
	h = page(t, ":::glossary\nDead Letter Queue!\n  x\n\nÄpfel\n  y\n:::\n").HTML
	contains(t, h, `id="term-dead-letter-queue"`, `id="term-u00e4pfel"`)
	// sorted
	h = page(t, ":::{glossary}\n:sorted:\n\nbanana\n  b\n\napple\n  a\n:::\n").HTML
	if strings.Index(h, "apple") > strings.Index(h, "banana") {
		t.Errorf("not sorted:\n%s", h)
	}
	// a term defined twice warns and the second gets its own id
	p := page(t, ":::glossary\nAPI\n  one\n\napi\n  two\n:::\n")
	contains(t, p.HTML, `id="term-api"`, `id="term-api-2"`)
	contains(t, strings.Join(p.Warnings, "|"), `plugin "terms"`, "twice")
}

func TestGlossaryOptions(t *testing.T) {
	h := page(t, "```{glossary}\n:class: wide\n:name: g1\n\nA\n  b\n```\n").HTML
	contains(t, h, `id="g1"`, "wide")
	p := page(t, "```{glossary}\n:colour: crimson\n\nA\n  b\n```\n")
	contains(t, p.HTML, `id="term-a"`)
	w := strings.Join(p.Warnings, "|")
	contains(t, w, `plugin "terms"`, "colour")
	lacks(t, w, "crimson")
}

func TestTermLink(t *testing.T) {
	h := page(t, ":::glossary\nAPI\n  x\n:::\n\nSee :term[API] and :term[the API <API>] and :term[api].\n").HTML
	contains(t, h, `<a href="#term-api"`, ">API</a>", ">the API</a>", ">api</a>")
	if strings.Count(h, `href="#term-api"`) != 3 {
		t.Errorf("links:\n%s", h)
	}
	lacks(t, h, "data-zen-term")
	// the same term written before the glossary
	p := page(t, "See :term[API].\n\n:::glossary\nAPI\n  x\n:::\n")
	contains(t, p.HTML, `href="#term-api"`)
	if len(p.Warnings) != 0 {
		t.Errorf("warnings %v", p.Warnings)
	}
	// the role form, as the dialect's role mapping calls the inline hook
	got, err := New().(plugins.InlineHook).RenderInline(plugins.Context{}, plugins.Inline{Name: "term", Text: "API"})
	if err != nil || !strings.Contains(got, `href="#term-api"`) {
		t.Errorf("RenderInline = %q, %v", got, err)
	}
}

func TestTermNotOnThePageWarns(t *testing.T) {
	p := page(t, "See :term[Missing] here.\n")
	contains(t, p.HTML, "Missing")
	contains(t, strings.Join(p.Warnings, "|"), `plugin "terms"`, "Missing")
	lacks(t, p.HTML, "data-zen-term")
	// a glossary on the page but without that term
	p = page(t, ":::glossary\nAPI\n  x\n:::\n\n:term[Other]\n")
	contains(t, strings.Join(p.Warnings, "|"), "Other")
}

func TestHostileStaysText(t *testing.T) {
	h := page(t, ":::glossary\n<script>alert(1)</script>\n  <img src=x onerror=alert(1)>\n:::\n\n:term[<script>x</script>] :term[a <\"><script>>]\n").HTML
	lacks(t, h, "<script", "onerror")
	h = page(t, ":::glossary\n:class: \"><script>\n\nA\n  b\n:::\n").HTML
	lacks(t, h, "<script")
}
