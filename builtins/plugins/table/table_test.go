package table

import (
	"html"
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
	"github.com/zenmark-org/zenmd/engine/workspace"
)

func folder(t *testing.T) (*render.Renderer, string) {
	t.Helper()
	dir := t.TempDir()
	write := func(name, body string) {
		p := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(".zenmd.yml", "markdown:\n  dialects: [myst]\n")
	write("data/prices.csv", "Item,Price\nApple,1.20\n\"Pear, green\",2.50\n")
	write("docs/local.csv", "A,B\nlocal,yes\n")
	write("page.md", "# x\n")
	ws, err := workspace.Open(dir)
	if err != nil {
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
	return render.New(reg, config.Load("", dir), ws).WithDialects(dr), dir
}

func page(t *testing.T, r *render.Renderer, path, src string) render.Page {
	t.Helper()
	p, err := r.Render(path, []byte(src))
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

var attrsRE = regexp.MustCompile(` data-zen-attrs="[^"]*"| data-zen-name="[^"]*"`)

func stripAttrs(s string) string { return attrsRE.ReplaceAllString(s, "") }

func TestManifest(t *testing.T) {
	p := New()
	m := p.Manifest()
	if m.ID != "table" || !m.Default || m.APIVersion != plugins.APIVersion || m.Version == "" {
		t.Fatalf("manifest %+v", m)
	}
	if strings.Join(m.Blocks, ",") != "table,list-table,csv-table" {
		t.Errorf("blocks %v", m.Blocks)
	}
	if _, ok := p.(plugins.BlockHook); !ok {
		t.Error("not a BlockHook")
	}
	if _, ok := p.(plugins.HTMLHook); !ok {
		t.Error("not an HTMLHook (numbering)")
	}
	ap, ok := p.(plugins.AssetProvider)
	if !ok {
		t.Fatal("not an AssetProvider")
	}
	if e := ap.Entry(); strings.Join(e.CSS, ",") != "table.css" || len(e.JS) != 0 {
		t.Errorf("entry %+v", e)
	}
	var colon, myst bool
	for _, s := range m.Help.Snippets {
		colon = colon || strings.Contains(s.Markdown, ":::table") || strings.Contains(s.Markdown, ":::csv-table") || strings.Contains(s.Markdown, ":::list-table")
		myst = myst || strings.Contains(s.Markdown, "{table}") || strings.Contains(s.Markdown, "{csv-table}") || strings.Contains(s.Markdown, "{list-table}")
	}
	if !colon || !myst {
		t.Error("help shows both the ::: form and the MyST form")
	}
}

const gfm = "| Name | Qty |\n|------|-----|\n| Ink  | 3   |\n| Pen  | 10  |\n"

func TestTableDirective(t *testing.T) {
	r, _ := folder(t)
	h := page(t, r, "page.md", "```{table} Stock levels\n:name: stock\n:align: center\n"+gfm+"```\n").HTML
	contains(t, h, `data-zen-plugin="table"`, `class="zen-block`, `<table id="stock">`, `<caption>`, "Table ",
		"Stock levels", "<th>Name</th>", "<td>Ink</td>", "zt-center")
	lacks(t, h, "zen-plugin-error", "<script")
	a := stripAttrs(page(t, r, "page.md", "```{table} Stock levels\n:name: stock\n:align: center\n"+gfm+"```\n").HTML)
	b := stripAttrs(page(t, r, "page.md", ":::table{title=\"Stock levels\" name=stock align=center}\n"+gfm+":::\n").HTML)
	c := stripAttrs(page(t, r, "page.md", ":::{table} Stock levels\n:name: stock\n:align: center\n"+gfm+":::\n").HTML)
	if a != b || a != c {
		t.Errorf("forms differ:\n%s\n%s\n%s", a, b, c)
	}
}

const listTable = "* - Item\n  - Notes\n* - **Ink**\n  - black,\n    refillable\n* - Pen\n  - `blue`\n"

func TestListTable(t *testing.T) {
	r, _ := folder(t)
	src := "```{list-table} Supplies\n:header-rows: 1\n:stub-columns: 1\n:widths: 30 70\n:align: right\n\n" + listTable + "```\n"
	h := page(t, r, "page.md", src).HTML
	contains(t, h, "<caption>", "Supplies",
		"<thead>", `<th scope="col">Item</th>`, `<th scope="col">Notes</th>`,
		"<tbody>", `<th scope="row"><strong>Ink</strong></th>`, "<td>black, refillable</td>", "<td><code>blue</code></td>",
		`<col width="30%">`, `<col width="70%">`, "zt-right")
	if strings.Count(h, "<tr>") != 3 {
		t.Errorf("rows:\n%s", h)
	}
	lacks(t, h, "zen-plugin-error")
	colon := page(t, r, "page.md", ":::list-table{title=Supplies header-rows=1 stub-columns=1 widths=\"30 70\" align=right}\n"+listTable+":::\n").HTML
	if stripAttrs(h) != stripAttrs(colon) {
		t.Errorf("forms differ:\n%s\n%s", stripAttrs(h), stripAttrs(colon))
	}
}

func TestListTableDefaultsAndShortRows(t *testing.T) {
	r, _ := folder(t)
	h := page(t, r, "page.md", "```{list-table}\n* - a\n  - b\n  - c\n* - d\n```\n").HTML
	lacks(t, h, "<thead>", "<caption>", "Table ")
	if strings.Count(h, "<td>") != 6 {
		t.Errorf("short rows are padded to 3 columns:\n%s", h)
	}
}

const csvBody = "Item, Price\nApple, 1.20\n\"Pear, green\", 2.50\n\"Say \"\"hi\"\"\", 0\n"

func TestCSVTableBody(t *testing.T) {
	r, _ := folder(t)
	h := page(t, r, "page.md", "```{csv-table} Prices\n:header-rows: 1\n:widths: 1 1\n\n"+csvBody+"```\n").HTML
	contains(t, h, "Prices", `<th scope="col">Item</th>`, `<th scope="col">Price</th>`,
		"<td>Pear, green</td>", `<td>Say &#34;hi&#34;</td>`, "<td>1.20</td>", `<col width="50%">`)
	lacks(t, h, "zen-plugin-error")
	colon := page(t, r, "page.md", ":::csv-table{title=Prices header-rows=1 widths=\"1 1\"}\n"+csvBody+":::\n").HTML
	if stripAttrs(h) != stripAttrs(colon) {
		t.Errorf("forms differ:\n%s\n%s", stripAttrs(h), stripAttrs(colon))
	}
}

func TestCSVTableHeaderAndDelim(t *testing.T) {
	r, _ := folder(t)
	h := page(t, r, "page.md", "```{csv-table}\n:header: \"Name\", \"Score\"\n:delim: ;\n\nann;3\nbob;4\n```\n").HTML
	contains(t, h, "<thead>", `<th scope="col">Name</th>`, `<th scope="col">Score</th>`, "<td>ann</td>", "<td>4</td>")
	tab := page(t, r, "page.md", "```{csv-table}\n:delim: tab\n\na\tb\n```\n").HTML
	contains(t, tab, "<td>a</td>", "<td>b</td>")
	bad := page(t, r, "page.md", "```{csv-table}\n:delim: ab\n\na,b\n```\n").HTML
	contains(t, bad, "delim", "zen-block-status", "<td>a</td>")
}

func TestCSVTableFile(t *testing.T) {
	r, _ := folder(t)
	h := page(t, r, "page.md", "```{csv-table} From a file\n:file: data/prices.csv\n:header-rows: 1\n```\n").HTML
	contains(t, h, "<td>Apple</td>", "<td>Pear, green</td>", `<th scope="col">Price</th>`)
	// relative to the page's folder first, then the workspace root
	loc := page(t, r, "docs/page.md", "```{csv-table}\n:file: local.csv\n```\n").HTML
	contains(t, loc, "<td>local</td>")
	root := page(t, r, "docs/page.md", "```{csv-table}\n:file: data/prices.csv\n```\n").HTML
	contains(t, root, "<td>Apple</td>")
}

// :file: reads only what a page read may: a workspace file, no .., no
// absolute path, no symlink out, at most 1 MB.
func TestCSVFileConfinement(t *testing.T) {
	r, dir := folder(t)
	outside := t.TempDir()
	secret := filepath.Join(outside, "secret.csv")
	if err := os.WriteFile(secret, []byte("TOPSECRET,1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "sibling.csv"), []byte("INSIDE,1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(secret, filepath.Join(dir, "link.csv")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(dir, "linkdir")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(dir, "data", "prices.csv"), filepath.Join(dir, "inlink.csv")); err != nil {
		t.Fatal(err)
	}
	big := make([]byte, 1<<20+1)
	for i := range big {
		big[i] = 'a'
	}
	if err := os.WriteFile(filepath.Join(dir, "big.csv"), big, 0o644); err != nil {
		t.Fatal(err)
	}
	exact := append(make([]byte, 0, 1<<20), []byte("OK,1\n")...)
	exact = append(exact, make([]byte, 1<<20-len(exact))...)
	for i := 5; i < len(exact); i++ {
		exact[i] = ' '
	}
	if err := os.WriteFile(filepath.Join(dir, "exact.csv"), exact, 0o644); err != nil {
		t.Fatal(err)
	}
	refused := map[string]string{
		"dotdot":        "../" + filepath.Base(outside) + "/secret.csv",
		"dotdot inside": "data/../sibling.csv",
		"dotdot out":    "../../../../../../etc/passwd",
		"absolute":      secret,
		"absolute in":   filepath.Join(dir, "sibling.csv"),
		"slash":         "/etc/passwd",
		"symlink file":  "link.csv",
		"symlink dir":   "linkdir/secret.csv",
		"too big":       "big.csv",
		"missing":       "nope.csv",
		"directory":     "data",
		"backslash":     `..\secret.csv`,
		"nul":           "a\x00b.csv",
	}
	for name, f := range refused {
		h := page(t, r, "page.md", "```{csv-table}\n:file: "+f+"\n```\n").HTML
		lacks(t, h, "TOPSECRET", "INSIDE", "root:")
		if !strings.Contains(h, "zen-plugin-error") {
			t.Errorf("%s: %q was not refused:\n%s", name, f, h)
		}
		if strings.Contains(h, dir) || strings.Contains(h, outside) {
			t.Errorf("%s: the error names a real path:\n%s", name, h)
		}
	}
	// a symlink that stays inside the folder is a page read too
	in := page(t, r, "page.md", "```{csv-table}\n:file: inlink.csv\n```\n").HTML
	contains(t, in, "<td>Apple</td>")
	ok := page(t, r, "page.md", "```{csv-table}\n:file: exact.csv\n```\n").HTML
	contains(t, ok, "<td>OK</td>")
	lacks(t, ok, "zen-plugin-error")
}

func TestCSVFileWithoutAFolder(t *testing.T) {
	h := New().(plugins.BlockHook)
	_, err := h.RenderBlock(plugins.Context{Path: "a.md"}, plugins.Block{Name: "csv-table", Attrs: map[string]string{"file": "x.csv"}})
	if err == nil {
		t.Fatal("a file with no folder must fail")
	}
	// url is never fetched
	r, _ := folder(t)
	p := page(t, r, "page.md", "```{csv-table}\n:url: https://example.com/x.csv\n\na,b\n```\n").HTML
	contains(t, p, "url", "zen-block-status", "<td>a</td>")
}

func TestTableTooLarge(t *testing.T) {
	r, _ := folder(t)
	var sb strings.Builder
	for i := 0; i < 60000; i++ {
		sb.WriteString("a,b,c\n")
	}
	h := page(t, r, "page.md", "```{csv-table}\n\n"+sb.String()+"```\n").HTML
	contains(t, h, "zen-plugin-error")
}

func TestNumbering(t *testing.T) {
	r, _ := folder(t)
	src := "```{table} One\n" + gfm + "```\n\n```{table}\n" + gfm + "```\n\n```{list-table} Two\n" + listTable + "```\n\n```{csv-table} Three\na,b\n```\n\n```{table} Skipped\n:nonumber:\n" + gfm + "```\n"
	h := page(t, r, "page.md", src).HTML
	re := regexp.MustCompile(`Table (?:<span[^>]*>)?(\d+)`)
	var got []string
	for _, m := range re.FindAllStringSubmatch(h, -1) {
		got = append(got, m[1])
	}
	if strings.Join(got, ",") != "1,2,3" {
		t.Errorf("numbers %v, want 1,2,3:\n%s", got, h)
	}
	again := page(t, r, "other.md", "```{table} Solo\n"+gfm+"```\n").HTML
	if !regexp.MustCompile(`Table (?:<span[^>]*>)?1\b`).MatchString(again) {
		t.Errorf("second page does not start at 1:\n%s", again)
	}
}

func TestUnknownOptionsWarn(t *testing.T) {
	r, _ := folder(t)
	h := page(t, r, "page.md", "```{list-table} T\n:frobnicate: 1\n:align: up\n:widths: 1 x\n\n"+listTable+"```\n").HTML
	contains(t, h, "frobnicate", "align", "widths", "zen-block-status", "<td>")
	lacks(t, h, "zen-plugin-error")
	for _, n := range []string{"table", "csv-table"} {
		body := gfm
		if n == "csv-table" {
			body = "a,b\n"
		}
		h := page(t, r, "page.md", "```{"+n+"} T\n:bogus: 1\n\n"+body+"```\n").HTML
		contains(t, h, "bogus", "zen-block-status")
	}
}

func TestHostile(t *testing.T) {
	r, _ := folder(t)
	src := "```{table} <script>alert(1)</script> \"><img src=x onerror=alert(2)>\n:name: a\"><script>alert(3)</script>\n:class: c\"onmouseover=\"alert(4) \n" +
		"| <script>alert(5)</script> | <img src=x onerror=alert(6)> |\n|---|---|\n| [x](javascript:alert(7)) | b |\n```\n\n" +
		"```{csv-table} <b onclick=alert(8)>x</b>\n:header: \"<script>alert(9)</script>\"\n\n<script>alert(10)</script>,<img src=x onerror=alert(11)>\n\"\"\"><svg onload=alert(12)>\",2\n```\n\n" +
		"```{list-table}\n* - <script>alert(13)</script>\n  - <img src=x onerror=alert(14)>\n```\n"
	h := page(t, r, "page.md", src).HTML
	lacks(t, h, "<script", "onerror", "onmouseover", "onclick", "onload", "javascript:", "<svg")
}

func TestHostileDirectOutput(t *testing.T) {
	h := New().(plugins.BlockHook)
	md := func(s string) (string, error) { return "<p>" + html.EscapeString(s) + "</p>\n", nil }
	for _, n := range []string{"table", "list-table", "csv-table"} {
		out, err := h.RenderBlock(plugins.Context{Path: "p.md", Markdown: md}, plugins.Block{
			Name: n, Body: "* - <script>x</script>\n  - <b>b</b>\n",
			Attrs: map[string]string{"title": "<i>t</i>", "name": `n"><b>`, "class": `c"><u>`, "align": `x"><s>`, "widths": `<s> 1`, "foo\"><s>": "1"},
		})
		if err != nil {
			continue
		}
		lacks(t, out, "<script", "<b>", "<i>", "<u>", "<s>")
	}
}

func TestKitClassesAndNoScript(t *testing.T) {
	r, _ := folder(t)
	h := page(t, r, "page.md", "```{table} T\n"+gfm+"```\n").HTML
	contains(t, h, `class="zen-block`, "<table", "<caption>")
	lacks(t, h, "<script", "<button", "style=")
	if e := New().(plugins.AssetProvider).Entry(); len(e.JS) != 0 {
		t.Errorf("js %v", e.JS)
	}
}

func TestCSSUsesTokensOnly(t *testing.T) {
	b, err := os.ReadFile("assets/table.css")
	if err != nil {
		t.Fatal(err)
	}
	css := regexp.MustCompile(`(?s)/\*.*?\*/`).ReplaceAllString(string(b), "")
	for _, bad := range []string{"#", "rgb(", "rgba(", "hsl(", "--font-body", "outline: none", "button", "input"} {
		if strings.Contains(css, bad) {
			t.Errorf("table.css contains %q", bad)
		}
	}
	for _, want := range []string{"var(--rule", "var(--fg-muted)", "var(--font-ui)"} {
		if !strings.Contains(css, want) {
			t.Errorf("table.css does not use %s", want)
		}
	}
}

func TestWithoutMyst(t *testing.T) {
	reg := plugins.NewRegistry()
	if err := reg.Register(New()); err != nil {
		t.Fatal(err)
	}
	p, err := render.New(reg, nil, nil).Render("p.md", []byte(":::csv-table{title=T}\na,b\n:::\n"))
	if err != nil {
		t.Fatal(err)
	}
	contains(t, p.HTML, "<td>a</td>", "<caption>")
}
