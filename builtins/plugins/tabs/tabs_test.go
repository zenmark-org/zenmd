package tabs

import (
	"strings"
	"testing"

	"github.com/zenmark-org/zenmd/engine/plugins"
)

const plainSet = ":::tab-set\n:::tab-item{title=One}\nFirst **body**.\n:::\n:::tab-item{title=Two}\nSecond body.\n:::\n:::\n"

func TestManifest(t *testing.T) {
	m := New().Manifest()
	if m.ID != "tabs" || m.APIVersion != plugins.APIVersion || !m.Default {
		t.Errorf("manifest = %+v", m)
	}
	if strings.Join(m.Blocks, ",") != "tab-set,tab-item" {
		t.Errorf("blocks = %v", m.Blocks)
	}
	if err := plugins.NewRegistry().Register(New()); err != nil {
		t.Fatal(err)
	}
	has(t, m.Help.Summary, "tab-set", "tab-item", "MyST")
	var joined string
	for _, s := range m.Help.Snippets {
		joined += s.Markdown
	}
	has(t, joined, ":::tab-set", ":::tab-item", "{tab-set}", "{tab-item}")
}

func TestKitClassesAndNoScriptOrder(t *testing.T) {
	got := page(t, plain(t), plainSet)
	has(t, got, "zen-block", "zen-tabs", "zen-tab-item", "zen-tab-title", "zen-block-body", "<strong>body</strong>")
	// every panel shows, in order, under its title
	order(t, got, ">One<", "First", ">Two<", "Second")
	hasNot(t, got, "hidden", "<script", "<button")
}

func TestMystFormsRenderLikeColons(t *testing.T) {
	want := bare(page(t, plain(t), plainSet))
	forms := map[string]string{
		"colon":    "::::{tab-set}\n:::{tab-item} One\nFirst **body**.\n:::\n:::{tab-item} Two\nSecond body.\n:::\n::::\n",
		"backtick": "````{tab-set}\n```{tab-item} One\nFirst **body**.\n```\n```{tab-item} Two\nSecond body.\n```\n````\n",
	}
	for name, src := range forms {
		if got := bare(page(t, mystRenderer(t), src)); got != want {
			t.Errorf("%s form differs:\n got %s\nwant %s", name, got, want)
		}
	}
}

func TestTabItemAloneStillRenders(t *testing.T) {
	got := page(t, plain(t), ":::tab-item{title=Solo}\nBody.\n:::\n")
	has(t, got, ">Solo<", "Body.", "zen-tab-item")
}

func TestOptions(t *testing.T) {
	r := plain(t)
	got := page(t, r, ":::tab-set{class=wide name=Lang.Tabs sync-group=lang}\n:::tab-item{title=Py sync=py selected}\np\n:::\n:::tab-item{title=Go sync=go}\ng\n:::\n:::\n")
	has(t, got, `id="lang.tabs"`, "wide", "zen-tab-selected", "sync")
	if w := warnings(t, r, ":::tab-set{class=wide name=x sync-group=g}\n:::tab-item{title=A sync=a selected class=c}\nx\n:::\n:::\n"); w != "" {
		t.Errorf("supported options warned: %s", w)
	}
	w := warnings(t, r, ":::tab-set{color=red}\n:::tab-item{title=A class-label=x}\nx\n:::\n:::\n")
	has(t, w, "tabs", ":color:", ":class-label:", "ignored")
	// a selected flag only marks the one item
	got = page(t, r, plainSet)
	if strings.Contains(got, "zen-tab-selected") {
		t.Errorf("nothing was selected: %s", got)
	}
	// a user class cannot take a zen- name
	hasNot(t, bare(page(t, r, ":::tab-set{class=zen-evil}\n:::tab-item{title=A}\nx\n:::\n:::\n")), "zen-evil")
}

func TestMissingTitleWarns(t *testing.T) {
	r := plain(t)
	w := warnings(t, r, ":::tab-set\n:::tab-item\nx\n:::\n:::\n")
	has(t, w, "tab-item", "title")
	has(t, page(t, r, ":::tab-set\n:::tab-item\nx\n:::\n:::\n"), ">Tab<")
}

func TestHostileText(t *testing.T) {
	got := page(t, plain(t), ":::tab-set\n:::tab-item{title=\"<img src=x onerror=alert(1)>\"}\n<script>alert(1)</script>\n<b onclick=\"x\">raw</b>\n:::\n:::\n")
	hasNot(t, got, "<img", "<script", "onclick")
	has(t, got, "&lt;img")
	// names are not markup either
	hasNot(t, page(t, plain(t), ":::tab-set{name=\"a b<c\"}\n:::tab-item{title=A}\nx\n:::\n:::\n"), "<c")
}

func TestOtherMarkdownInTabs(t *testing.T) {
	got := page(t, plain(t), ":::tab-set\n:::tab-item{title=Code}\n```go\nfmt.Println(1)\n```\n\n- a\n- b\n:::\n:::\n")
	has(t, got, "<pre", "<li>a</li>")
}

func TestAssets(t *testing.T) {
	e := New().(plugins.AssetProvider).Entry()
	if len(e.CSS) != 1 || len(e.JS) != 1 {
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
