package aside

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

func TestManifest(t *testing.T) {
	p := New()
	m := p.Manifest()
	if m.ID != "aside" || !m.Default || m.APIVersion != plugins.APIVersion || m.Version == "" {
		t.Fatalf("manifest %+v", m)
	}
	if strings.Join(m.Blocks, ",") != "sidebar,margin,topic" {
		t.Errorf("blocks %v", m.Blocks)
	}
	ap, ok := p.(plugins.AssetProvider)
	if !ok || strings.Join(ap.Entry().CSS, ",") != "aside.css" {
		t.Fatal("assets")
	}
	for _, n := range m.Blocks {
		if !strings.Contains(m.Help.Summary, n) {
			t.Errorf("help summary does not name %s", n)
		}
	}
	var colon, my bool
	for _, s := range m.Help.Snippets {
		colon = colon || strings.Contains(s.Markdown, ":::sidebar") || strings.Contains(s.Markdown, ":::margin") || strings.Contains(s.Markdown, ":::topic")
		my = my || strings.Contains(s.Markdown, "{sidebar}") || strings.Contains(s.Markdown, "{margin}") || strings.Contains(s.Markdown, "{topic}")
	}
	if !colon || !my {
		t.Error("help shows both forms")
	}
	css, _ := fs.ReadFile(ap.Assets(), "aside.css")
	contains(t, string(css), "float: right", "@media", "var(--rule)")
	lacks(t, string(css), "#", "rgb(", "hsl(")
}

func TestSidebar(t *testing.T) {
	a := page(t, "```{sidebar} Did you know?\n:subtitle: A side note\n\nThe **body** is markdown.\n```\n")
	contains(t, a.HTML, `data-zen-plugin="aside"`, `class="zen-block zen-aside zen-aside-sidebar`, "<aside>",
		`<div class="zen-block-head">Did you know?</div>`, `zen-aside-subtitle`, "A side note", "<strong>body</strong>")
	lacks(t, a.HTML, "zen-plugin-error")
	b := page(t, ":::sidebar{title=\"Did you know?\" subtitle=\"A side note\"}\nThe **body** is markdown.\n:::\n")
	c := page(t, ":::{sidebar} Did you know?\n:subtitle: A side note\n\nThe **body** is markdown.\n:::\n")
	if strip(a.HTML) != strip(b.HTML) || strip(a.HTML) != strip(c.HTML) {
		t.Errorf("forms differ:\n%s\n%s\n%s", strip(a.HTML), strip(b.HTML), strip(c.HTML))
	}
	if len(a.Warnings) != 0 {
		t.Errorf("warnings %v", a.Warnings)
	}
}

func TestMarginAndTopic(t *testing.T) {
	h := page(t, "```{margin}\nA note in the margin.\n```\n").HTML
	contains(t, h, "zen-aside-margin", "A note in the margin.", "<aside>")
	lacks(t, h, "zen-block-head")
	h = page(t, ":::margin\nA note in the margin.\n:::\n").HTML
	contains(t, h, "zen-aside-margin")
	h = page(t, "```{topic} Background\nText.\n```\n").HTML
	contains(t, h, "zen-aside-topic", `<div class="zen-block-head">Background</div>`, "Text.")
	h2 := page(t, ":::topic{title=Background}\nText.\n:::\n").HTML
	if strip(h) != strip(h2) {
		t.Errorf("forms differ:\n%s\n%s", strip(h), strip(h2))
	}
}

func TestClassAndName(t *testing.T) {
	h := page(t, "```{sidebar} T\n:class: tall\n:name: side-one\n\nBody.\n```\n").HTML
	contains(t, h, `id="side-one"`, "tall")
	h = page(t, ":::sidebar{class=zen-bad}\nBody.\n:::\n").HTML
	lacks(t, h, "zen-bad")
}

func TestUnknownOptionWarns(t *testing.T) {
	p := page(t, "```{sidebar} T\n:width: 40%\n\nBody.\n```\n")
	contains(t, p.HTML, "Body.")
	w := strings.Join(p.Warnings, "|")
	contains(t, w, `plugin "aside"`, "width")
	lacks(t, w, "40%")
}

func TestHostileStaysText(t *testing.T) {
	h := page(t, "```{sidebar} <img src=x onerror=alert(1)>\n:subtitle: <script>alert(1)</script>\n\n<script>alert(2)</script>\n```\n").HTML
	lacks(t, h, "<script", "<img", "onerror")
	contains(t, h, "&lt;img")
}
