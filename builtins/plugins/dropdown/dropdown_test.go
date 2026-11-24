package dropdown

import (
	"strings"
	"testing"

	"github.com/zenmark-org/zenmd/engine/plugins"
)

func TestManifest(t *testing.T) {
	m := New().Manifest()
	if m.ID != "dropdown" || m.APIVersion != plugins.APIVersion || !m.Default {
		t.Errorf("manifest = %+v", m)
	}
	if strings.Join(m.Blocks, ",") != "dropdown" {
		t.Errorf("blocks = %v", m.Blocks)
	}
	has(t, m.Help.Summary, "dropdown", "MyST")
	var joined string
	for _, s := range m.Help.Snippets {
		joined += s.Markdown
	}
	has(t, joined, ":::dropdown", "{dropdown}")
}

func TestPlainDetailsInKitFrame(t *testing.T) {
	got := page(t, plain(t), ":::dropdown{title=\"More\"}\nHidden **text**.\n:::\n")
	has(t, got, "<details", "<summary", "zen-block", "zen-dropdown", "zen-block-head", ">More<", "zen-block-body", "<strong>text</strong>")
	hasNot(t, got, "<details open", "<script")
	order(t, got, "<summary", "More", "</summary>", "Hidden")
}

func TestMystFormsRenderLikeColons(t *testing.T) {
	want := bare(page(t, plain(t), ":::dropdown{title=\"More\"}\nHidden *text*.\n:::\n"))
	for name, src := range map[string]string{
		"colon":    ":::{dropdown} More\nHidden *text*.\n:::\n",
		"backtick": "```{dropdown} More\nHidden *text*.\n```\n",
	} {
		if got := bare(page(t, mystRenderer(t), src)); got != want {
			t.Errorf("%s form differs:\n got %s\nwant %s", name, got, want)
		}
	}
}

func TestOpenAndOptions(t *testing.T) {
	r := plain(t)
	has(t, page(t, r, ":::dropdown{title=A open}\nx\n:::\n"), "<details open")
	hasNot(t, page(t, r, ":::dropdown{title=A open=false}\nx\n:::\n"), "<details open")
	has(t, page(t, mystRenderer(t), "```{dropdown} A\n:open:\nx\n```\n"), "<details open")
	has(t, page(t, r, ":::dropdown{title=A name=Why.Not class=wide}\nx\n:::\n"), `id="why.not"`, "wide")
	if w := warnings(t, r, ":::dropdown{title=A open name=n class=c}\nx\n:::\n"); w != "" {
		t.Errorf("supported options warned: %s", w)
	}
}

func TestColorAndIconWarnAndAreIgnored(t *testing.T) {
	r := plain(t)
	src := ":::dropdown{title=A color=primary icon=star}\nx\n:::\n"
	w := warnings(t, r, src)
	has(t, w, "dropdown", ":color:", ":icon:", "ignored")
	got := bare(page(t, r, src))
	hasNot(t, got, "primary", "star")
	has(t, warnings(t, r, ":::dropdown{title=A animate=fade-in}\nx\n:::\n"), ":animate:")
}

func TestNoTitleUsesDefault(t *testing.T) {
	has(t, page(t, plain(t), ":::dropdown\nx\n:::\n"), ">Details<")
}

func TestHostileText(t *testing.T) {
	got := page(t, plain(t), ":::dropdown{title=\"<img src=x onerror=alert(1)>\"}\n<script>alert(1)</script>\n:::\n")
	hasNot(t, got, "<img", "<script")
	has(t, got, "&lt;img")
	hasNot(t, bare(page(t, plain(t), ":::dropdown{title=A class=\"x\\\" onclick=\\\"y\"}\nx\n:::\n")), "onclick")
}

func TestNestedDropdown(t *testing.T) {
	got := page(t, plain(t), ":::dropdown{title=Outer}\n:::dropdown{title=Inner}\ndeep\n:::\n:::\n")
	order(t, got, ">Outer<", ">Inner<", "deep")
}

func TestAssets(t *testing.T) {
	e := New().(plugins.AssetProvider).Entry()
	if len(e.CSS) != 1 || len(e.JS) != 0 {
		t.Errorf("entry = %+v", e)
	}
}

// Every snippet of the help renders without a warning or an error, and draws the plugin's own markup.
func TestHelpSnippetsRender(t *testing.T) {
	for _, s := range New().Manifest().Help.Snippets {
		r := mystRenderer(t)
		got := page(t, r, s.Markdown)
		if w := warnings(t, r, s.Markdown); w != "" {
			t.Errorf("snippet %q warns: %s", s.Title, w)
		}
		if !strings.Contains(got, `data-zen-plugin="`+New().Manifest().ID+`"`) {
			t.Errorf("snippet %q was not drawn by the plugin: %s", s.Title, got)
		}
		if strings.Contains(got, "<p>:::") || strings.Contains(got, "{") && strings.Contains(got, "```") {
			t.Errorf("snippet %q left its syntax as text: %s", s.Title, got)
		}
	}
}
