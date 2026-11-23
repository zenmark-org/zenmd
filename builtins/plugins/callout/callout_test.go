package callout

import (
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

func plain(t *testing.T) *render.Renderer {
	t.Helper()
	reg := plugins.NewRegistry()
	if err := reg.Register(New()); err != nil {
		t.Fatal(err)
	}
	return render.New(reg, nil, nil)
}

func mystRenderer(t *testing.T) *render.Renderer {
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

func page(t *testing.T, r *render.Renderer, src string) string {
	t.Helper()
	p, err := r.Render("a.md", []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(p.HTML, "zen-plugin-error") {
		t.Fatalf("plugin error: %s", p.HTML)
	}
	return p.HTML
}

var attrsRE = regexp.MustCompile(` data-zen-attrs="[^"]*"`)

func bare(s string) string { return attrsRE.ReplaceAllString(s, "") }

func has(t *testing.T, got string, want ...string) {
	t.Helper()
	for _, w := range want {
		if !strings.Contains(got, w) {
			t.Errorf("missing %q in %s", w, got)
		}
	}
}

func TestManifest(t *testing.T) {
	m := New().Manifest()
	if m.ID != "callout" || !m.Default || m.APIVersion != plugins.APIVersion || m.Version == "" {
		t.Fatalf("manifest %+v", m)
	}
	want := "note,tip,hint,important,seealso,warning,caution,attention,danger,error,admonition,versionadded,versionchanged,deprecated"
	if strings.Join(m.Blocks, ",") != want {
		t.Errorf("blocks %v", m.Blocks)
	}
	if _, ok := New().(plugins.BlockHook); !ok {
		t.Error("not a BlockHook")
	}
	if _, ok := New().(plugins.AssetProvider); !ok {
		t.Error("no assets")
	}
	if len(m.Help.Snippets) < 2 || !strings.Contains(m.Help.Summary, "callout") {
		t.Errorf("help %+v", m.Help)
	}
	joined := ""
	for _, s := range m.Help.Snippets {
		joined += s.Markdown
	}
	has(t, joined, ":::note", "```{note}")
}

func TestKinds(t *testing.T) {
	cases := []struct{ name, title, icon, group string }{
		{"note", "Note", "N", "suggestion"}, {"tip", "Tip", "T", "suggestion"}, {"hint", "Hint", "H", "suggestion"},
		{"important", "Important", "!", "suggestion"}, {"seealso", "See also", "S", "suggestion"},
		{"warning", "Warning", "W", "warning"}, {"caution", "Caution", "C", "warning"}, {"attention", "Attention", "A", "warning"},
		{"danger", "Danger", "D", "error"}, {"error", "Error", "E", "error"},
	}
	r := plain(t)
	for _, c := range cases {
		got := page(t, r, ":::"+c.name+"\nSome **body**.\n:::\n")
		has(t, got, `zen-block`, `zen-callout-`+c.group, `<span class="zen-callout-icon" aria-hidden="true">`+c.icon+`</span>`,
			`<span class="zen-callout-title">`+c.title+`</span>`, `<strong>body</strong>`, `zen-block-head`, `zen-block-body`)
		if strings.Contains(got, "<details") {
			t.Errorf("%s: not collapsed by default: %s", c.name, got)
		}
	}
}

func TestMystFormsRenderLikeColons(t *testing.T) {
	plainR, mystR := plain(t), mystRenderer(t)
	for _, name := range []string{"note", "tip", "hint", "important", "seealso", "warning", "caution", "attention", "danger", "error"} {
		want := bare(page(t, plainR, ":::"+name+"{title=\"Remember\"}\nBody *text*.\n:::\n"))
		for form, src := range map[string]string{
			"fence": "```{" + name + "} Remember\nBody *text*.\n```\n",
			"colon": ":::{" + name + "} Remember\nBody *text*.\n:::\n",
		} {
			if got := bare(page(t, mystR, src)); got != want {
				t.Errorf("%s %s:\n got %s\nwant %s", name, form, got, want)
			}
		}
		has(t, want, "Remember")
	}
}

func TestTitleArgumentReplacesDefault(t *testing.T) {
	got := page(t, mystRenderer(t), "```{warning} Mind the gap\nx\n```\n")
	has(t, got, `<span class="zen-callout-title">Mind the gap</span>`, "zen-callout-warning")
	if strings.Contains(got, ">Warning<") {
		t.Errorf("default title kept: %s", got)
	}
}

func TestAdmonition(t *testing.T) {
	r := mystRenderer(t)
	got := page(t, r, "```{admonition} My own title\nBody.\n```\n")
	has(t, got, `<span class="zen-callout-title">My own title</span>`, "zen-callout-suggestion", "Body.")
	same := page(t, plain(t), ":::admonition{title=\"My own title\"}\nBody.\n:::\n")
	if bare(got) != bare(same) {
		t.Errorf("forms differ:\n%s\n%s", got, same)
	}
	// a class names the colour, as in MyST
	has(t, page(t, r, "```{admonition} Careful\n:class: danger\nx\n```\n"), "zen-callout-error")
	// a missing title warns and uses "Note"
	miss := page(t, r, "```{admonition}\nBody.\n```\n")
	has(t, miss, `<span class="zen-callout-title">Note</span>`, "zen-block-status", "needs a title")
}

func TestVersionKinds(t *testing.T) {
	r := mystRenderer(t)
	for name, title := range map[string]string{
		"versionadded": "New in version 1.2", "versionchanged": "Changed in version 1.2", "deprecated": "Deprecated since version 1.2",
	} {
		got := page(t, r, "```{"+name+"} 1.2\nUse the new flag.\n```\n")
		has(t, got, `<span class="zen-callout-title">`+title+`</span>`, "Use the new flag.", "zen-callout-version")
		same := page(t, plain(t), ":::"+name+"{version=1.2}\nUse the new flag.\n:::\n")
		if bare(got) != bare(same) {
			t.Errorf("%s forms differ:\n%s\n%s", name, got, same)
		}
		// no body is fine: one line
		has(t, page(t, r, "```{"+name+"} 1.2\n```\n"), title)
	}
	has(t, page(t, r, "```{versionadded}\nx\n```\n"), "needs a version", "zen-block-status")
}

func TestDropdown(t *testing.T) {
	r := mystRenderer(t)
	got := page(t, r, "```{note} Fold me\n:class: dropdown\nHidden **text**.\n```\n")
	has(t, got, "<details>", "<summary>", `zen-callout-title">Fold me</span>`, "<strong>text</strong>", "zen-callout-dropdown")
	if strings.Contains(got, "<details open") {
		t.Errorf("starts open: %s", got)
	}
	// summary comes first, so the title shows when collapsed and a page without scripts still reads
	if strings.Index(got, "<summary>") > strings.Index(got, "Hidden") {
		t.Errorf("summary after body: %s", got)
	}
	open := page(t, r, "```{note} Fold me\n:class: dropdown\n:open:\nx\n```\n")
	has(t, open, "<details open")
	if strings.Contains(open, "zen-callout-dropdown zen-callout-dropdown") {
		t.Error("duplicate class")
	}
	// the zen form
	has(t, page(t, plain(t), ":::tip{class=dropdown open}\nx\n:::\n"), "<details open", "zen-callout-suggestion")
}

func TestUnknownOptionWarns(t *testing.T) {
	got := page(t, mystRenderer(t), "```{note}\n:bogus: 1\n:class: dropdown\nBody\n```\n")
	has(t, got, "zen-block-status", "option :bogus: is not supported", "<details>", "Body")
	if strings.Contains(page(t, mystRenderer(t), "```{note}\n:class: wide\n:name: my-note\nBody\n```\n"), "not supported") {
		t.Error("class and name are supported")
	}
}

func TestClassTokens(t *testing.T) {
	got := page(t, mystRenderer(t), "```{note}\n:class: wide my-extra \"><script>\nBody\n```\n")
	has(t, got, "wide", "my-extra")
	if strings.Contains(got, "<script") {
		t.Errorf("hostile class: %s", got)
	}
}

func TestNameIsKept(t *testing.T) {
	// the wrapper carries the options, so :name: reaches the page
	has(t, page(t, mystRenderer(t), "```{note}\n:name: my-note\nBody\n```\n"), "my-note")
}

func TestHostileText(t *testing.T) {
	r := mystRenderer(t)
	got := page(t, r, "```{admonition} <img src=x onerror=alert(1)>\nBody <script>alert(1)</script> and <b onclick=x>b</b>\n```\n")
	for _, bad := range []string{"<img", "<script", " onclick", "<b onclick"} {
		if strings.Contains(got, bad) {
			t.Errorf("%q survived: %s", bad, got)
		}
	}
	has(t, got, "&lt;img")
	got = page(t, r, "```{versionadded} <script>x</script>\n```\n")
	if strings.Contains(got, "<script") {
		t.Errorf("version: %s", got)
	}
	got = page(t, plain(t), ":::note{title=\"<i>x</i>\" open}\n:::\n")
	has(t, got, "&lt;i&gt;x&lt;/i&gt;")
}

func TestNoScriptOutputReads(t *testing.T) {
	got := page(t, plain(t), ":::warning\nFirst.\n\n- one\n- two\n:::\n")
	// title, then body, in document order, no script needed
	if !(strings.Index(got, "Warning") < strings.Index(got, "First.") && strings.Contains(got, "<li>two</li>")) {
		t.Errorf("%s", got)
	}
	e := New().(plugins.AssetProvider).Entry()
	if len(e.JS) != 0 || len(e.CSS) != 1 || e.CSS[0] != "callout.css" {
		t.Errorf("entry %+v", e)
	}
}

func TestStylesheet(t *testing.T) {
	b, err := os.ReadFile("assets/callout.css")
	if err != nil {
		t.Fatal(err)
	}
	css := string(b)
	if regexp.MustCompile(`#[0-9a-fA-F]{3,8}\b|rgba?\(|hsla?\(`).MatchString(css) {
		t.Error("fixed colour in callout.css")
	}
	has(t, css, "var(--lint-suggestion)", "var(--lint-warning)", "var(--lint-error)", ":focus-visible")
	if strings.Contains(css, "outline: none") || strings.Contains(css, "font-family") {
		t.Error("restyles focus or font")
	}
	// the only hue is the left rule
	for _, l := range strings.Split(css, "\n") {
		if strings.Contains(l, "--lint-") && !strings.Contains(l, "border-left") && !strings.Contains(l, "border-inline-start") && !strings.Contains(l, "--zc-rule") {
			t.Errorf("a signal colour outside the left rule: %s", l)
		}
	}
}

func TestKitClasses(t *testing.T) {
	got := page(t, plain(t), ":::tip\nx\n:::\n")
	has(t, got, `class="zen-block zen-callout zen-callout-suggestion"`, `class="zen-block-head zen-callout-head"`)
}
