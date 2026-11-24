package math

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zenmark-org/zenmd/builtins/dialects/myst"
	"github.com/zenmark-org/zenmd/engine/config"
	"github.com/zenmark-org/zenmd/engine/dialects"
	"github.com/zenmark-org/zenmd/engine/plugins"
	"github.com/zenmark-org/zenmd/engine/render"
	"github.com/zenmark-org/zenmd/engine/workspace"
)

// dollarRenderer is a MyST folder that turns on dollarmath.
func dollarRenderer(t *testing.T, dollar bool) *render.Renderer {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".zenmd.yml"), []byte("markdown:\n  dialects: [myst]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if dollar {
		if err := os.WriteFile(filepath.Join(dir, "_config.yml"), []byte("parse:\n  myst_enable_extensions:\n    - dollarmath\n"), 0o644); err != nil {
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
	dr, _ := dialects.NewRegistry(myst.New())
	return render.New(reg, config.Load("", dir), ws).WithDialects(dr)
}

func TestManifestOwnsTheRoles(t *testing.T) {
	m := New().Manifest()
	if strings.Join(m.Inlines, ",") != "math,eq" {
		t.Errorf("inlines %v", m.Inlines)
	}
	all := strings.Builder{}
	for _, s := range m.Help.Snippets {
		all.WriteString(s.Markdown)
	}
	for _, f := range []string{":math[", "{math}`", ":eq[", "{eq}`", "$"} {
		if !strings.Contains(all.String(), f) {
			t.Errorf("no help snippet shows %q", f)
		}
	}
	for _, n := range []string{"{math}", "{eq}", "dollarmath"} {
		if !strings.Contains(m.Help.Summary, n) {
			t.Errorf("the summary does not mention %s", n)
		}
	}
}

func TestInlineMathRole(t *testing.T) {
	r := dollarRenderer(t, false)
	for _, src := range []string{"A {math}`E = mc^2` here.\n", "A :math[E = mc^2] here.\n"} {
		p := page(t, r, src)
		contains(t, p.HTML, `<code class="zm-itex">E = mc^2</code>`, "zm-inline")
		lacks(t, p.HTML, "zm-tex", "<pre")
		if len(p.Warnings) != 0 {
			t.Errorf("warnings %v", p.Warnings)
		}
	}
	// the TeX is text, never markup
	p := page(t, r, "{math}`<script>alert(1)</script> \\alpha`\n")
	inert(t, p.HTML)
	contains(t, p.HTML, "&lt;script&gt;")
	// an empty role
	p = page(t, r, ":math[ ]\n")
	if len(p.Warnings) != 1 && !strings.Contains(p.HTML, "zen-plugin-error") {
		t.Errorf("empty inline math: %v / %s", p.Warnings, p.HTML)
	}
}

func TestDollarInlineMath(t *testing.T) {
	src := "Let $x^2$ and $y_1$ be; it costs $5 and $10.\n"
	p := page(t, dollarRenderer(t, true), src)
	contains(t, p.HTML, `<code class="zm-itex">x^2</code>`, `<code class="zm-itex">y_1</code>`, "$5 and $10")
	if strings.Count(p.HTML, "zm-itex") != 2 {
		t.Errorf("html: %s", p.HTML)
	}
	p = page(t, dollarRenderer(t, false), src)
	lacks(t, p.HTML, "zm-itex")
	contains(t, p.HTML, "$x^2$")
}

func TestEqRole(t *testing.T) {
	r := dollarRenderer(t, false)
	src := "See {eq}`pythag` and :eq[pythag].\n\n```{math}\n:label: pythag\n\na^2+b^2=c^2\n```\n\n```{math}\n:label: second\n\nx=1\n```\n\nand {eq}`second`.\n"
	p := page(t, r, src)
	contains(t, p.HTML, `href="#pythag"`, ">(1)</a>", `href="#second"`, ">(2)</a>")
	if strings.Count(p.HTML, `href="#pythag"`) != 2 {
		t.Errorf("html: %s", p.HTML)
	}
	lacks(t, p.HTML, "⟦", "⟧")
	if len(p.Warnings) != 0 {
		t.Errorf("warnings %v", p.Warnings)
	}
	// an unknown equation warns once and stays text; a name without a number is not an equation number
	p = page(t, r, "{eq}`nowhere` {eq}`nowhere` {eq}`<b>x</b>`\n\n```{math}\n:name: unnumbered\n\nx\n```\n\n{eq}`unnumbered`\n")
	lacks(t, p.HTML, `href="#nowhere`, "<b>x</b>", `href="#unnumbered"`)
	contains(t, p.HTML, "nowhere")
	n := 0
	for _, w := range p.Warnings {
		if strings.Contains(w, "nowhere") {
			n++
		}
	}
	if n != 1 {
		t.Errorf("want one warning for nowhere: %v", p.Warnings)
	}
}
