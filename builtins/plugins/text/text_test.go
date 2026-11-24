package text

import (
	"strings"
	"testing"

	"github.com/zenmark-org/zenmd/engine/plugins"
	"github.com/zenmark-org/zenmd/engine/render"
)

func page(t *testing.T, src string) render.Page {
	t.Helper()
	reg := plugins.NewRegistry()
	if err := reg.Register(New()); err != nil {
		t.Fatal(err)
	}
	p, err := render.New(reg, nil, nil).Render("a.md", []byte(src))
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

func TestManifest(t *testing.T) {
	p := New()
	m := p.Manifest()
	if m.ID != "text" || !m.Default || m.APIVersion != plugins.APIVersion || m.Version == "" {
		t.Fatalf("manifest %+v", m)
	}
	if strings.Join(m.Inlines, ",") != "kbd,abbr,sub,sup" {
		t.Errorf("inlines %v", m.Inlines)
	}
	if len(m.Blocks) != 0 {
		t.Errorf("blocks %v", m.Blocks)
	}
	if _, ok := p.(plugins.InlineHook); !ok {
		t.Error("not an InlineHook")
	}
	ap, ok := p.(plugins.AssetProvider)
	if !ok || strings.Join(ap.Entry().CSS, ",") != "text.css" {
		t.Errorf("assets %+v", ok)
	}
	for _, n := range []string{"kbd", "abbr", "sub", "sup"} {
		if !strings.Contains(m.Help.Summary, n) {
			t.Errorf("help summary does not name %s", n)
		}
	}
	var zen, myst bool
	for _, s := range m.Help.Snippets {
		zen = zen || strings.Contains(s.Markdown, ":kbd[")
		myst = myst || strings.Contains(s.Markdown, "{kbd}`")
	}
	if !zen || !myst {
		t.Error("help shows the :kbd[] form and the MyST role form")
	}
}

func TestKbd(t *testing.T) {
	for in, want := range map[string]string{
		"Ctrl+C":       "<kbd>Ctrl</kbd>+<kbd>C</kbd>",
		"⌘+K":          "<kbd>⌘</kbd>+<kbd>K</kbd>",
		"Ctrl-Shift-P": "<kbd>Ctrl</kbd>-<kbd>Shift</kbd>-<kbd>P</kbd>",
		"Enter":        "<kbd>Enter</kbd>",
		"Ctrl+-":       "<kbd>Ctrl</kbd>+<kbd>-</kbd>",
		"+":            "<kbd>+</kbd>",
	} {
		h := page(t, ":kbd["+in+"]\n").HTML
		contains(t, h, `data-zen-plugin="text"`, want)
		lacks(t, h, "zen-plugin-error")
	}
	// the inline hook alone, as the dialect's role mapping calls it
	got, err := New().(plugins.InlineHook).RenderInline(plugins.Context{}, plugins.Inline{Name: "kbd", Text: "Ctrl+C"})
	if err != nil || got != "<kbd>Ctrl</kbd>+<kbd>C</kbd>" {
		t.Errorf("RenderInline = %q, %v", got, err)
	}
	if got, _ := New().(plugins.InlineHook).RenderInline(plugins.Context{}, plugins.Inline{Name: "kbd", Text: "  "}); got != "" {
		t.Errorf("empty kbd = %q", got)
	}
}

func TestAbbr(t *testing.T) {
	h := page(t, ":abbr[HTML (HyperText Markup Language)]\n").HTML
	contains(t, h, `<abbr title="HyperText Markup Language">HTML</abbr>`)
	h = page(t, `:abbr[CSS]{title="Cascading Style Sheets"}`+"\n").HTML
	contains(t, h, `<abbr title="Cascading Style Sheets">CSS</abbr>`)
	p := page(t, ":abbr[NASA]\n")
	contains(t, p.HTML, "<abbr>NASA</abbr>")
	contains(t, strings.Join(p.Warnings, "|"), `plugin "text"`, "abbr")
	// hostile
	h = page(t, `:abbr[X (a "quoted" <b>title</b>)]`+"\n").HTML
	lacks(t, h, "<b>", `title="a "quoted"`)
}

func TestSubSup(t *testing.T) {
	h := page(t, "E = mc :sup[2] and H :sub[2]\n").HTML
	contains(t, h, "<sup>2</sup>", "<sub>2</sub>")
	h = page(t, ":sup[<script>alert(1)</script>]\n").HTML
	lacks(t, h, "<script")
}

func TestHostileStaysText(t *testing.T) {
	for _, src := range []string{
		":kbd[<img src=x onerror=alert(1)>]\n",
		":kbd[<script>alert(1)</script>]\n",
		":sub[<img src=x onerror=alert(1)>]\n",
		":abbr[x (<img src=x onerror=alert(1)>)]\n",
		":abbr[x]{title=\"\\\" onmouseover=alert(1) x=\\\"\"}\n",
	} {
		h := page(t, src).HTML
		lacks(t, h, "<img", "<script", `" onmouseover`)
	}
}

func TestUnknownOptionWarns(t *testing.T) {
	p := page(t, ":kbd[X]{color=red}\n")
	contains(t, p.HTML, "<kbd>X</kbd>")
	w := strings.Join(p.Warnings, "|")
	contains(t, w, `plugin "text"`, "color")
	lacks(t, w, "=red", " red")
}

func TestCSS(t *testing.T) {
	ap := New().(plugins.AssetProvider)
	b, err := fsRead(ap, "text.css")
	if err != nil {
		t.Fatal(err)
	}
	contains(t, b, "kbd", "var(--rule)")
	lacks(t, b, "#", "rgb(", "hsl(")
}
