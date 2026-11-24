package render

import (
	"html"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zenmark-org/zenmd/engine/config"
	"github.com/zenmark-org/zenmd/engine/plugins"
	"github.com/zenmark-org/zenmd/engine/workspace"
)

// roleEcho is an inline plugin that shows what it was given.
func roleEcho(id string, names ...string) *fakePlugin {
	m := man(id, true)
	m.Inlines = names
	return &fakePlugin{m: m, inline: func(_ plugins.Context, in plugins.Inline) (string, error) {
		return "<b>" + in.Name + ":" + html.EscapeString(in.Text) + "</b>", nil
	}}
}

func TestRolesMapToInlinePlugins(t *testing.T) {
	r := mystRenderer(t, mystOn, roleEcho("echo", "ref", "kbd"))
	p := render(t, r, "See {ref}`intro` and {kbd}`Ctrl+C`, `` {ref}`x` `` stays code.\n\n```\n{ref}`in a fence`\n```\n")
	contains(t, p.HTML, "<b>ref:intro</b>", "<b>kbd:Ctrl+C</b>", "<code>{ref}`x`</code>", "{ref}`in a fence`")
	lacks(t, p.HTML, "<b>ref:in a fence", "<b>ref:x")
	if len(p.Warnings) != 0 {
		t.Errorf("warnings: %v", p.Warnings)
	}
	// the zen form of the same inline plugin
	p = render(t, r, ":ref[intro]\n")
	contains(t, p.HTML, "<b>ref:intro</b>")
}

func TestUnownedRoleStaysTextWithOneWarning(t *testing.T) {
	r := mystRenderer(t, mystOn, roleEcho("echo", "ref"))
	p := render(t, r, "a {nobody}`one` b {nobody}`two` c {other}`<script>alert(1)</script>`\n")
	contains(t, p.HTML, "{nobody}", "one", "two", "{other}")
	lacks(t, p.HTML, "<script")
	n, o := 0, 0
	for _, w := range p.Warnings {
		if strings.Contains(w, "nobody") {
			n++
		}
		if strings.Contains(w, "other") {
			o++
		}
		if strings.Contains(w, "alert") {
			t.Errorf("the warning carries the role's text: %q", w)
		}
	}
	if n != 1 || o != 1 {
		t.Errorf("want one warning per role, got nobody=%d other=%d: %v", n, o, p.Warnings)
	}
	// without the dialect, roles are plain text and warn about nothing
	plainR := New(newReg(t, roleEcho("echo", "ref")), config.Default(), fakeResolver{}).WithDialects(mystReg(t))
	p = render(t, plainR, "{ref}`x`\n")
	contains(t, p.HTML, "{ref}")
	if len(p.Warnings) != 0 {
		t.Errorf("warnings without the dialect: %v", p.Warnings)
	}
}

func TestDollarMathNeedsTheExtension(t *testing.T) {
	m := roleEcho("m", "math")
	dir := t.TempDir()
	write := func(name, s string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(s), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(".zenmd.yml", mystOn)
	ws, err := workspace.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	r := New(newReg(t, m), config.Load("", dir), ws).WithDialects(mystReg(t))
	if p := render(t, r, "so $x^2$ here\n"); strings.Contains(p.HTML, "<b>math") {
		t.Errorf("dollar math without the extension: %s", p.HTML)
	}
	write("_config.yml", "parse:\n  myst_enable_extensions: [dollarmath]\n")
	ws, _ = workspace.Open(dir)
	r = New(newReg(t, m), config.Load("", dir), ws).WithDialects(mystReg(t))
	contains(t, render(t, r, "so $x^2$ here\n").HTML, "<b>math:x^2</b>")
}

func TestUnsupportedDirectivesStayCode(t *testing.T) {
	r := mystRenderer(t, mystOn, admonitions("note", "note"))
	src := "```{raw} html\n<script>alert(1)</script>\n```\n\n:::{eval}\n1+1\n:::\n\n```{code-cell} python\nprint(1)\n```\n\n:::{only} html\ntext\n:::\n\n```{ifconfig} x\ny\n```\n\n```{nobody}\nbody\n```\n"
	p := render(t, r, src)
	lacks(t, p.HTML, "<script", "data-zen-plugin")
	contains(t, p.HTML, "&lt;script&gt;alert(1)&lt;/script&gt;", "print(1)", "body")
	for _, name := range []string{"raw", "eval", "code-cell", "only", "ifconfig", "nobody"} {
		n := 0
		for _, w := range p.Warnings {
			if strings.Contains(w, "{"+name+"}") {
				n++
			}
		}
		if n != 1 {
			t.Errorf("%s: %d warnings, want 1: %v", name, n, p.Warnings)
		}
	}
	// the same directive twice warns once
	p = render(t, r, "```{raw} html\na\n```\n\n```{raw} html\nb\n```\n")
	n := 0
	for _, w := range p.Warnings {
		if strings.Contains(w, "{raw}") {
			n++
		}
	}
	if n != 1 {
		t.Errorf("raw twice: %d warnings %v", n, p.Warnings)
	}
}

func TestNonDirectiveSyntax(t *testing.T) {
	r := mystRenderer(t, mystOn, admonitions("note", "note"))
	src := "---\nsubstitutions:\n  product: Zenmark\n---\n(intro)=\n# Intro\n\n% a comment\nWelcome to {{ product }}.\n\n+++\n\nTerm one\n: The first definition.\n\nTerm two\n: The second.\n\n```{note}\n% kept? {{ product }}\n```\n"
	p := render(t, r, src)
	contains(t, p.HTML, `id="intro"`, "Welcome to Zenmark.", "<hr", "<dl>", "<dt>Term one</dt>", "<dd>", "The first definition.", "Zenmark")
	lacks(t, p.HTML, "a comment", "(intro)=", "+++", "{{")
	if p.Title != "Intro" {
		t.Errorf("title %q", p.Title)
	}
	// the cards-style +++ separator inside a directive body is the directive's, not a break
	var got string
	cap := &fakePlugin{m: func() plugins.Manifest { m := man("cap", true); m.Blocks = []string{"cap"}; return m }(),
		block: func(_ plugins.Context, b plugins.Block) (string, error) { got = b.Body; return "<i>x</i>", nil }}
	r = mystRenderer(t, mystOn, cap)
	render(t, r, "```{cap}\nhead\n+++\nfoot\n% c\n```\n")
	if !strings.Contains(got, "+++") || !strings.Contains(got, "% c") {
		t.Errorf("a directive body was rewritten before its plugin read it: %q", got)
	}
}

func TestFieldListsAreFrontmatter(t *testing.T) {
	r := mystRenderer(t, mystOn)
	p := render(t, r, "---\nauthor: Front\n---\n:author: Field\n:status: draft\n\n# Title\n\nBody\n")
	lacks(t, p.HTML, ":status:", "draft")
	if p.Frontmatter["status"] != "draft" {
		t.Errorf("frontmatter %v", p.Frontmatter)
	}
	if p.Frontmatter["author"] != "Front" {
		t.Errorf("a field must not override frontmatter: %v", p.Frontmatter)
	}
	// a field list cannot set what frontmatter controls
	p = render(t, r, ":layout: slides\n:plugins: x\n\n# T\n")
	if p.Layout.Name != "article" && p.Layout.Name != "" {
		t.Errorf("layout %v", p.Layout)
	}
}

func TestPageTitleContext(t *testing.T) {
	dir := t.TempDir()
	files := map[string]string{
		".zenmd.yml":     mystOn,
		"guide/start.md": "# Getting started\n\ntext\n",
		"fm.md":          "---\ntitle: From front matter\n---\nno heading\n",
		"both.md":        "---\ntitle: Ignored\n---\n# The H1\n",
		"plain.md":       "just text\n",
	}
	for n, c := range files {
		_ = os.MkdirAll(filepath.Dir(filepath.Join(dir, n)), 0o755)
		if err := os.WriteFile(filepath.Join(dir, n), []byte(c), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Symlink("/etc", filepath.Join(dir, "etc")); err != nil {
		t.Fatal(err)
	}
	ws, err := workspace.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	var title func(string) (string, bool)
	pl := &fakePlugin{m: func() plugins.Manifest { m := man("pt", true); m.Inlines = []string{"pt"}; return m }(),
		inline: func(c plugins.Context, in plugins.Inline) (string, error) { title = c.PageTitle; return "x", nil }}
	r := New(newReg(t, pl), config.Load("", dir), ws).WithDialects(mystReg(t))
	if _, err := r.Render("docs/page.md", []byte(":pt[x]\n")); err != nil {
		t.Fatal(err)
	}
	if title == nil {
		t.Fatal("PageTitle is nil with a folder")
	}
	// paths are relative to the page, as links are
	for path, want := range map[string]string{"../guide/start": "Getting started", "../guide/start.md": "Getting started", "../fm": "From front matter", "../both": "The H1", "../plain": "plain"} {
		if got, ok := title(path); !ok || got != want {
			t.Errorf("%s: %q %v, want %q", path, got, ok, want)
		}
	}
	for _, path := range []string{"../nothing", "", "../../outside", "../etc/passwd", "https://example.com"} {
		if got, ok := title(path); ok {
			t.Errorf("%q should not resolve, got %q", path, got)
		}
	}
	// no folder: nil
	title = func(string) (string, bool) { return "set", true }
	r = New(newReg(t, pl), config.Default(), nil)
	if _, err := r.Render("a.md", []byte(":pt[x]\n")); err != nil {
		t.Fatal(err)
	}
	if title != nil {
		t.Error("PageTitle should be nil without a folder")
	}
}

func TestCardFooterBreakIsTheCards(t *testing.T) {
	var got string
	m := man("cards", true)
	m.Blocks = []string{"card"}
	card := &fakePlugin{m: m, block: func(_ plugins.Context, b plugins.Block) (string, error) { got = b.Body; return "<i>card</i>", nil }}
	r := mystRenderer(t, mystOn, card)
	p := render(t, r, "```{card} Title\nbody\n+++\nfooter\n```\n\n+++\n\nafter\n")
	if !strings.Contains(got, "\n+++\nfooter") {
		t.Errorf("the card lost its +++ line: %q", got)
	}
	if strings.Count(p.HTML, "<hr") != 1 {
		t.Errorf("only the break outside the card is a rule: %s", p.HTML)
	}
}
