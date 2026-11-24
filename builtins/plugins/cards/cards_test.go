package cards

import (
	"strings"
	"testing"

	"github.com/zenmark-org/zenmd/engine/plugins"
)

var colours = []string{"primary", "secondary", "success", "info", "warning", "danger", "light", "dark"}

func TestManifest(t *testing.T) {
	m := New().Manifest()
	if m.ID != "cards" || m.APIVersion != plugins.APIVersion || !m.Default {
		t.Errorf("manifest = %+v", m)
	}
	if got := strings.Join(m.Blocks, ","); got != "card,grid,grid-item,grid-item-card,button-link,button-ref" {
		t.Errorf("blocks = %v", m.Blocks)
	}
	want := []string{"bdg"}
	for _, c := range colours {
		want = append(want, "bdg-"+c, "bdg-"+c+"-line")
	}
	if strings.Join(m.Inlines, ",") != strings.Join(want, ",") {
		t.Errorf("inlines = %v", m.Inlines)
	}
	for _, n := range []string{"card", "grid", "grid-item-card", "button-link", "button-ref", "bdg-primary"} {
		has(t, m.Help.Summary, n)
	}
	has(t, m.Help.Summary, "MyST")
	var joined string
	for _, s := range m.Help.Snippets {
		joined += s.Markdown
	}
	has(t, joined, ":::card", "{card}", ":::grid", "{grid}", ":::button-link", "{button-link}", ":bdg-primary[")
	if err := plugins.NewRegistry().Register(New()); err != nil {
		t.Fatal(err)
	}
}

// ---- card ----

const plainCard = ":::card{title=\"Card title\"}\nHeader text\n^^^\nThe **body**.\n+++\nFooter text\n:::\n"

func TestCardHeaderBodyFooter(t *testing.T) {
	got := page(t, plain(t), plainCard)
	has(t, got, "zen-block", "zen-card", "zen-card-header", "zen-card-body", "zen-card-footer", "zen-card-title", "<strong>body</strong>")
	order(t, got, "zen-card-header", "Header text", "zen-card-body", "Card title", "The", "zen-card-footer", "Footer text")
	hasNot(t, got, "^^^", "+++")
}

func TestCardWithoutSeparators(t *testing.T) {
	got := page(t, plain(t), ":::card\nJust a body.\n:::\n")
	has(t, got, "zen-card-body", "Just a body.")
	hasNot(t, got, "zen-card-header", "zen-card-footer", "zen-card-title")
}

func TestCardFooterOnlyAndSeparatorsInCode(t *testing.T) {
	got := page(t, plain(t), ":::card\nBody\n+++\nOnly footer\n:::\n")
	has(t, got, "zen-card-footer", "Only footer")
	hasNot(t, got, "zen-card-header")
	// a ^^^ line inside a code fence is code, not a separator
	got = page(t, plain(t), ":::card\n~~~text\n^^^\n+++\n~~~\n:::\n")
	hasNot(t, got, "zen-card-header", "zen-card-footer")
	has(t, got, "^^^")
}

func TestCardMystFormsRenderLikeColons(t *testing.T) {
	want := bare(page(t, plain(t), plainCard))
	for name, src := range map[string]string{
		"colon":    ":::{card} Card title\nHeader text\n^^^\nThe **body**.\n+++\nFooter text\n:::\n",
		"backtick": "```{card} Card title\nHeader text\n^^^\nThe **body**.\n+++\nFooter text\n```\n",
	} {
		if got := bare(page(t, mystRenderer(t), src)); got != want {
			t.Errorf("%s form differs:\n got %s\nwant %s", name, got, want)
		}
	}
}

func TestCardLinkMakesTheWholeCardALink(t *testing.T) {
	got := page(t, plain(t), ":::card{title=\"Go on\" link=\"https://example.com/x\"}\nBody\n:::\n")
	has(t, got, "zen-card-link", `href="https://example.com/x"`, "zen-card-stretch", ">Go on<")
	if n := strings.Count(got, "<a "); n != 1 {
		t.Errorf("want one link, got %d in %s", n, got)
	}
	// an internal page goes through the page's own link handling
	got = page(t, plain(t), ":::card{title=Next link=other.md link-type=doc}\nBody\n:::\n")
	has(t, got, "zen-card-stretch", "other.md")
	got = page(t, plain(t), ":::card{title=Next link=other link-type=doc}\nBody\n:::\n")
	has(t, got, "other.md")
	got = page(t, plain(t), ":::card{title=Next link=my-label link-type=ref}\nBody\n:::\n")
	has(t, got, `href="#my-label"`)
	// link-alt names the link when there is no title
	got = page(t, plain(t), ":::card{link=\"https://example.com\" link-alt=\"Read more\"}\nBody\n:::\n")
	has(t, got, "Read more", "zen-card-stretch")
}

func TestCardLinkIsSafe(t *testing.T) {
	r := plain(t)
	got := page(t, r, ":::card{title=X link=\"javascript:alert(1)\"}\nBody\n:::\n")
	hasNot(t, got, "javascript:")
	got = page(t, r, ":::card{title=\"A [b](http://evil.example) c\" link=\"https://example.com\"}\nBody\n:::\n")
	if n := strings.Count(got, "<a "); n != 1 {
		t.Errorf("a title must not add links: %s", got)
	}
}

func TestCardOptions(t *testing.T) {
	r := plain(t)
	got := page(t, r, ":::card{title=T name=My.Card class=wide text-align=center}\nBody\n:::\n")
	has(t, got, `id="my.card"`, "wide", "zen-card-center")
	if w := warnings(t, r, ":::card{title=T name=n class=c text-align=left link=\"https://x.org\" link-type=url link-alt=a}\nBody\n:::\n"); w != "" {
		t.Errorf("supported options warned: %s", w)
	}
	w := warnings(t, r, ":::card{shadow=md img-top=a.png class-card=x}\nBody\n:::\n")
	has(t, w, "cards", ":shadow:", ":img-top:", ":class-card:", "ignored")
	has(t, warnings(t, r, ":::card{text-align=sideways}\nBody\n:::\n"), ":text-align:")
	has(t, warnings(t, r, ":::card{link=x link-type=nope}\nBody\n:::\n"), ":link-type:")
}

func TestCardHostileText(t *testing.T) {
	got := page(t, plain(t), ":::card{title=\"<img src=x onerror=alert(1)>\"}\n<script>alert(1)</script>\n^^^\n<b onclick=\"x\">h</b>\n:::\n")
	hasNot(t, got, "<img", "<script", "onclick")
	has(t, got, "&lt;img")
}

func TestCardInsideCard(t *testing.T) {
	got := page(t, plain(t), ":::card{title=Outer}\n:::card{title=Inner}\ndeep\n:::\n:::\n")
	order(t, got, "Outer", "Inner", "deep")
}

// ---- grid ----

const plainGrid = ":::grid{columns=\"1 2 3 4\"}\n:::grid-item-card{title=One}\nFirst\n:::\n:::grid-item-card{title=Two}\nSecond\n:::\n:::grid-item\nPlain *item*.\n:::\n:::\n"

func TestGridColumnsPerWidth(t *testing.T) {
	got := page(t, plain(t), plainGrid)
	has(t, got, "zen-grid", "zen-grid-row", "zen-grid-xs-1", "zen-grid-sm-2", "zen-grid-md-3", "zen-grid-lg-4", "zen-grid-item", "zen-card", "<em>item</em>")
	order(t, got, "One", "First", "Two", "Second", "item")
	// fewer numbers: the last one carries up
	got = page(t, plain(t), ":::grid{columns=\"2\"}\n:::grid-item\nx\n:::\n:::\n")
	has(t, got, "zen-grid-xs-2", "zen-grid-sm-2", "zen-grid-md-2", "zen-grid-lg-2")
	got = page(t, plain(t), ":::grid{columns=\"1 1 2\"}\n:::grid-item\nx\n:::\n:::\n")
	has(t, got, "zen-grid-xs-1", "zen-grid-sm-1", "zen-grid-md-2", "zen-grid-lg-2")
	// no argument: one column
	got = page(t, plain(t), ":::grid\n:::grid-item\nx\n:::\n:::\n")
	has(t, got, "zen-grid-xs-1", "zen-grid-lg-1")
}

func TestGridMystFormsRenderLikeColons(t *testing.T) {
	want := bare(page(t, plain(t), plainGrid))
	forms := map[string]string{
		"colon":    "::::{grid} 1 2 3 4\n:::{grid-item-card} One\nFirst\n:::\n:::{grid-item-card} Two\nSecond\n:::\n:::{grid-item}\nPlain *item*.\n:::\n::::\n",
		"backtick": "````{grid} 1 2 3 4\n```{grid-item-card} One\nFirst\n```\n```{grid-item-card} Two\nSecond\n```\n```{grid-item}\nPlain *item*.\n```\n````\n",
	}
	for name, src := range forms {
		if got := bare(page(t, mystRenderer(t), src)); got != want {
			t.Errorf("%s form differs:\n got %s\nwant %s", name, got, want)
		}
	}
}

func TestGridOptionsAndWarnings(t *testing.T) {
	r := plain(t)
	got := page(t, r, ":::grid{columns=\"1 2\" gutter=3 class=wide name=G}\n:::grid-item\nx\n:::\n:::\n")
	has(t, got, "zen-gutter-3", "wide", `id="g"`)
	if w := warnings(t, r, ":::grid{columns=\"1 2\" gutter=\"1 2 3 4\" class=c name=n}\n:::grid-item{class=c}\nx\n:::\n:::\n"); w != "" {
		t.Errorf("supported options warned: %s", w)
	}
	w := warnings(t, r, ":::grid{columns=\"1 2\" reverse}\n:::grid-item{columns=6 child-direction=row}\nx\n:::\n:::\n")
	has(t, w, ":reverse:", ":columns:", ":child-direction:", "ignored")
	has(t, warnings(t, r, ":::grid{columns=\"1 two 3\"}\n:::grid-item\nx\n:::\n:::\n"), "grid", "columns")
	has(t, warnings(t, r, ":::grid{columns=\"1 2 3 4 5\"}\n:::grid-item\nx\n:::\n:::\n"), "grid", "columns")
	has(t, warnings(t, r, ":::grid{columns=\"13\"}\n:::grid-item\nx\n:::\n:::\n"), "grid", "columns")
	// a bad number falls back to one column
	has(t, page(t, r, ":::grid{columns=\"zzz\"}\n:::grid-item\nx\n:::\n:::\n"), "zen-grid-xs-1")
}

func TestGridItemCardOptionsLikeCard(t *testing.T) {
	r := plain(t)
	got := page(t, r, ":::grid{columns=2}\n:::grid-item-card{title=T link=\"https://example.com\"}\nBody\n^^^\nHead\n:::\n:::\n")
	has(t, got, "zen-card-link", "zen-card-header", "zen-grid-item")
	has(t, warnings(t, r, ":::grid{columns=2}\n:::grid-item-card{title=T shadow=md}\nB\n:::\n:::\n"), ":shadow:")
}

func TestGridHostileText(t *testing.T) {
	got := page(t, plain(t), ":::grid{columns=\"1\"}\n:::grid-item\n<script>alert(1)</script>\n:::\n:::\n")
	hasNot(t, got, "<script")
	hasNot(t, page(t, plain(t), ":::grid{columns=\"1\" class=\"a\\\" onclick=\\\"b\"}\n:::grid-item\nx\n:::\n:::\n"), "onclick")
}

// ---- buttons ----

func TestButtonLink(t *testing.T) {
	got := page(t, plain(t), ":::button-link{href=\"https://example.com\"}\nGo there\n:::\n")
	has(t, got, "zen-btn", "zen-button-link", `href="https://example.com"`, ">Go there<")
	hasNot(t, got, "<button", "zen-btn-primary")
	// MyST: the argument is the URL, the body the text
	want := bare(got)
	for name, src := range map[string]string{
		"colon":    ":::{button-link} https://example.com\nGo there\n:::\n",
		"backtick": "```{button-link} https://example.com\nGo there\n```\n",
	} {
		if g := bare(page(t, mystRenderer(t), src)); g != want {
			t.Errorf("%s form differs:\n got %s\nwant %s", name, g, want)
		}
	}
	// without text, the link shows itself
	has(t, page(t, plain(t), ":::button-link{href=\"https://example.com/a\"}\n:::\n"), ">https://example.com/a<")
}

func TestButtonRef(t *testing.T) {
	got := page(t, mystRenderer(t), "```{button-ref} my-label\n:ref-type: ref\nSee it\n```\n")
	has(t, got, "zen-btn", `href="#my-label"`, ">See it<")
	got = page(t, mystRenderer(t), "```{button-ref} other\n:ref-type: doc\nOther page\n```\n")
	has(t, got, "other.md", ">Other page<")
	got = page(t, plain(t), ":::button-ref{ref=lab}\nText\n:::\n")
	has(t, got, `href="#lab"`)
}

func TestButtonOptions(t *testing.T) {
	r := plain(t)
	got := page(t, r, ":::button-link{href=\"https://example.com\" align=center expand class=wide name=B}\nHi\n:::\n")
	has(t, got, "zen-button-center", "zen-button-expand", "wide", `id="b"`)
	if w := warnings(t, r, ":::button-link{href=\"https://example.com\" align=right expand class=c name=n}\nHi\n:::\n"); w != "" {
		t.Errorf("supported options warned: %s", w)
	}
	w := warnings(t, r, ":::button-link{href=\"https://example.com\" color=primary outline shadow tooltip=x}\nHi\n:::\n")
	has(t, w, ":color:", ":outline:", ":shadow:", ":tooltip:", "ignored")
	has(t, warnings(t, r, ":::button-link\nNo target\n:::\n"), "button-link", "link")
}

func TestButtonIsSafeAndEscaped(t *testing.T) {
	r := plain(t)
	got := page(t, r, ":::button-link{href=\"javascript:alert(1)\"}\nClick\n:::\n")
	hasNot(t, got, "javascript:")
	got = page(t, r, ":::button-link{href=\"https://example.com\"}\n<img src=x onerror=alert(1)> [x](http://evil.example) *em*\n:::\n")
	hasNot(t, got, "<img", "evil.example", "<em>")
	has(t, got, "&lt;img")
}

// ---- badges ----

func TestBadges(t *testing.T) {
	for _, c := range colours {
		for _, suffix := range []string{"", "-line"} {
			name := "bdg-" + c + suffix
			got := page(t, plain(t), "See :"+name+"[Beta] here.\n")
			has(t, got, `class="zen-badge`, ">Beta<", "See", "here.")
			if suffix == "-line" {
				has(t, got, "zen-badge-line")
			} else {
				hasNot(t, got, "zen-badge-line")
			}
			// the colour name never reaches the page: every badge is drawn the same
			hasNot(t, got, "zen-badge-"+c, ">"+c+"<")
		}
	}
	has(t, page(t, plain(t), ":bdg[Plain]\n"), "zen-badge", ">Plain<")
}

func TestBadgeTextIsText(t *testing.T) {
	got := page(t, plain(t), ":bdg-info[<img src=x onerror=alert(1)>]\n")
	hasNot(t, got, "<img")
	has(t, got, "&lt;img")
}

func TestBadgeInCodeStaysCode(t *testing.T) {
	got := page(t, plain(t), "Write `:bdg-primary[x]` to get one.\n")
	hasNot(t, got, "zen-badge")
}

func TestBadgeOptionsWarn(t *testing.T) {
	r := plain(t)
	has(t, warnings(t, r, ":bdg-primary[x]{foo=bar}\n"), "cards", ":foo:", "ignored")
}

func TestAssets(t *testing.T) {
	e := New().(plugins.AssetProvider).Entry()
	if len(e.CSS) != 1 || len(e.JS) != 0 {
		t.Errorf("entry = %+v", e)
	}
}
