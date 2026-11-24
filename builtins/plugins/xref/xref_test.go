package xref

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/zenmark-org/zenmd/builtins/dialects/myst"
	"github.com/zenmark-org/zenmd/builtins/plugins/figure"
	"github.com/zenmark-org/zenmd/builtins/plugins/math"
	"github.com/zenmark-org/zenmd/builtins/plugins/table"
	"github.com/zenmark-org/zenmd/engine/config"
	"github.com/zenmark-org/zenmd/engine/dialects"
	"github.com/zenmark-org/zenmd/engine/plugins"
	"github.com/zenmark-org/zenmd/engine/render"
	"github.com/zenmark-org/zenmd/engine/workspace"
	xhtml "golang.org/x/net/html"
)

// folder makes a workspace with a few pages and files, and a renderer with the
// built-ins that number things, xref last (as in the build), and MyST on.
func folder(t *testing.T) *render.Renderer {
	t.Helper()
	dir := t.TempDir()
	files := map[string]string{
		".zenmd.yml":        "markdown:\n  dialects: [myst]\n",
		"guide/start.md":    "# Getting started\n\ntext\n",
		"guide/advanced.md": "---\ntitle: Advanced use\n---\nno heading\n",
		"plain.md":          "just text\n",
		"pic.png":           "x",
		"data.zip":          "x",
		"img/a.png":         "x",
	}
	for n, c := range files {
		p := filepath.Join(dir, n)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(c), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	ws, err := workspace.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	reg := plugins.NewRegistry()
	for _, p := range []plugins.Plugin{figure.New(), table.New(), math.New(), New()} {
		if err := reg.Register(p); err != nil {
			t.Fatal(err)
		}
	}
	dr, err := dialects.NewRegistry(myst.New())
	if err != nil {
		t.Fatal(err)
	}
	return render.New(reg, config.Load("", dir), ws).WithDialects(dr)
}

func page(t *testing.T, src string) render.Page {
	t.Helper()
	p, err := folder(t).Render("page.md", []byte(src))
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

func warned(p render.Page, sub string) int {
	n := 0
	for _, w := range p.Warnings {
		if strings.Contains(w, sub) {
			n++
		}
	}
	return n
}

var attrsRE = regexp.MustCompile(` data-zen-attrs="[^"]*"`)

func inert(t *testing.T, h string) {
	t.Helper()
	doc, err := xhtml.Parse(strings.NewReader(h))
	if err != nil {
		t.Fatal(err)
	}
	var walk func(n *xhtml.Node)
	walk = func(n *xhtml.Node) {
		if n.Type == xhtml.ElementNode {
			switch n.Data {
			case "script", "svg", "iframe", "style", "object", "embed", "img":
				t.Errorf("hostile element <%s> in\n%s", n.Data, h)
			}
			for _, a := range n.Attr {
				if strings.HasPrefix(strings.ToLower(a.Key), "on") || strings.HasPrefix(strings.ToLower(a.Val), "javascript:") {
					t.Errorf("hostile attribute %s=%q in\n%s", a.Key, a.Val, h)
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
}

func TestManifest(t *testing.T) {
	m := New().Manifest()
	if m.ID != "xref" || !m.Default || m.APIVersion != plugins.APIVersion || m.Version == "" || m.Description == "" {
		t.Fatalf("manifest %+v", m)
	}
	if strings.Join(m.Blocks, ",") != "toctree" {
		t.Errorf("blocks %v", m.Blocks)
	}
	if strings.Join(m.Inlines, ",") != "ref,numref,doc,download" {
		t.Errorf("inlines %v", m.Inlines)
	}
	if m.Help.Summary == "" || len(m.Help.Snippets) < 4 {
		t.Errorf("help %+v", m.Help)
	}
	for _, name := range []string{"ref", "numref", "doc", "download", "toctree"} {
		if !strings.Contains(m.Help.Summary, "{"+name+"}") {
			t.Errorf("the summary does not name {%s}", name)
		}
	}
	forms := strings.Builder{}
	for _, s := range m.Help.Snippets {
		forms.WriteString(s.Markdown)
	}
	for _, f := range []string{":ref[", ":numref[", ":doc[", ":download[", ":::toctree", "{ref}`", "{numref}`", "{doc}`", "{download}`", "```{toctree}"} {
		if !strings.Contains(forms.String(), f) {
			t.Errorf("no help snippet shows %q", f)
		}
	}
}

func TestRefToATargetBeforeAHeading(t *testing.T) {
	p := page(t, "(sec)=\n## A section\n\nSee {ref}`sec`, {ref}`go there <sec>` and :ref[sec].\n")
	contains(t, p.HTML, `id="sec"`, `href="#sec"`, ">A section</a>", ">go there</a>")
	if strings.Count(p.HTML, `href="#sec"`) != 3 {
		t.Errorf("want 3 links:\n%s", p.HTML)
	}
	if len(p.Warnings) != 0 {
		t.Errorf("warnings %v", p.Warnings)
	}
}

func TestNumrefToFiguresTablesAndEquations(t *testing.T) {
	src := "```{figure} img/a.png\n:name: fig-a\n\nA cap.\n```\n\n" +
		"```{table} Sales\n:name: tab-s\n\n| a | b |\n|---|---|\n| 1 | 2 |\n```\n\n" +
		"```{math}\n:label: eq-e\n\na=b\n```\n\n" +
		"{numref}`fig-a`, {numref}`tab-s`, {numref}`Fig. %s <fig-a>`, :numref[fig-a], {eq}`eq-e`, {numref}`eq-e`.\n"
	p := page(t, src)
	contains(t, p.HTML, `href="#fig-a"`, ">Figure 1</a>", `href="#tab-s"`, ">Table 1</a>", ">Fig. 1</a>", `href="#eq-e"`, ">(1)</a>")
	if strings.Count(p.HTML, ">Figure 1</a>") != 2 {
		t.Errorf("the zen form should read the same:\n%s", p.HTML)
	}
	lacks(t, p.HTML, "⟦", "⟧")
	if len(p.Warnings) != 0 {
		t.Errorf("warnings %v", p.Warnings)
	}
	// a numref to a target that has no number reads as a plain ref, with a warning
	p = page(t, "(sec)=\n# S\n\n{numref}`sec`\n")
	contains(t, p.HTML, `href="#sec"`)
	if warned(p, "sec") != 1 {
		t.Errorf("warnings %v", p.Warnings)
	}
}

func TestUnknownLabelWarnsAndStaysText(t *testing.T) {
	p := page(t, "A {ref}`nowhere`, {ref}`Some text <nowhere>`, {numref}`nowhere2`.\n")
	lacks(t, p.HTML, `href="#nowhere`, "⟦")
	contains(t, p.HTML, "nowhere", "Some text")
	if warned(p, "nowhere,") != 1 || warned(p, "nowhere2,") != 1 {
		t.Errorf("want one warning per label: %v", p.Warnings)
	}
}

func TestHostileText(t *testing.T) {
	p := page(t, "{ref}`<script>alert(1)</script>` {ref}`<img src=x onerror=alert(1)> <(sec)>` {doc}`<script>x</script>` {download}`javascript:alert(1)` :ref[\"><b onclick=x>]\n")
	inert(t, p.HTML)
	lacks(t, p.HTML, "<script", `href="javascript:`, "<b ")
	for _, w := range p.Warnings {
		if strings.ContainsAny(w, "<>") {
			t.Errorf("a warning carries hostile text: %q", w)
		}
	}
}

func TestDocLinksAndTitles(t *testing.T) {
	p := page(t, "{doc}`guide/start`, {doc}`guide/advanced.md`, {doc}`Read this <guide/start>`, {doc}`plain`, :doc[guide/start].\n")
	contains(t, p.HTML, `data-path="guide/start.md"`, ">Getting started</a>", `data-path="guide/advanced.md"`, ">Advanced use</a>", ">Read this</a>", ">plain</a>")
	if strings.Count(p.HTML, ">Getting started</a>") != 2 {
		t.Errorf("zen form: %s", p.HTML)
	}
	if len(p.Warnings) != 0 {
		t.Errorf("warnings %v", p.Warnings)
	}
	// a page that is not there warns, and still reads as text
	p = page(t, "{doc}`guide/missing`\n")
	contains(t, p.HTML, "missing")
	if warned(p, "guide/missing") != 1 {
		t.Errorf("warnings %v", p.Warnings)
	}
}

func TestDocWithoutAFolderUsesTheFileName(t *testing.T) {
	reg := plugins.NewRegistry()
	if err := reg.Register(New()); err != nil {
		t.Fatal(err)
	}
	dr, _ := dialects.NewRegistry(myst.New())
	cfg := config.Default()
	r := render.New(reg, cfg, nil).WithDialects(dr)
	p, err := r.Render("page.md", []byte(":doc[guide/start-here.md]\n"))
	if err != nil {
		t.Fatal(err)
	}
	contains(t, p.HTML, "start-here")
	lacks(t, p.HTML, "<script")
}

func TestDownload(t *testing.T) {
	p := page(t, "{download}`pic.png`, {download}`Get the picture <pic.png>`, :download[pic.png].\n")
	contains(t, p.HTML, `href="/_ws/pic.png"`, ">pic.png</a>", ">Get the picture</a>")
	if len(p.Warnings) != 0 {
		t.Errorf("warnings %v", p.Warnings)
	}
	p = page(t, "{download}`nothing.png`\n")
	lacks(t, p.HTML, `href="/_ws`)
	if warned(p, "nothing.png") != 1 {
		t.Errorf("warnings %v", p.Warnings)
	}
}

func TestOutsideTheFolderIsRefused(t *testing.T) {
	for _, path := range []string{"../secret.md", "guide/../../secret.md", "/etc/passwd", "/guide/start", "C:\\x.md", "https://example.com/a.md", "guide\\start", "a\x00b", "file:///etc/passwd"} {
		for _, form := range []string{"{doc}`%s`", "{download}`%s`", ":doc[%s]", ":download[%s]"} {
			p := page(t, strings.Replace(form, "%s", path, 1)+"\n")
			lacks(t, p.HTML, `data-zen-link="internal"`, `href="/_ws`, `href="https`, `href="file`, `data-zen-link="external"`)
			if len(p.Warnings) == 0 {
				t.Errorf("%s %q: no warning", form, path)
			}
		}
	}
	p := page(t, ":::toctree\n../secret\n/etc/passwd\nguide/start\n:::\n")
	contains(t, p.HTML, "Getting started")
	lacks(t, p.HTML, "secret", "passwd")
	if len(p.Warnings) < 2 {
		t.Errorf("warnings %v", p.Warnings)
	}
}

func TestToctree(t *testing.T) {
	for name, src := range map[string]string{
		"myst": "```{toctree}\n:caption: Contents\n:maxdepth: 2\n\nguide/start\nCustom name <guide/advanced>\nplain\n```\n",
		"zen":  ":::toctree{caption=Contents maxdepth=2}\nguide/start\nCustom name <guide/advanced>\nplain\n:::\n",
	} {
		p := page(t, src)
		contains(t, p.HTML, "Contents", "<ul>", `data-path="guide/start.md"`, ">Getting started</a>", ">Custom name</a>", `data-path="guide/advanced.md"`, `data-path="plain.md"`, ">plain</a>")
		if len(p.Warnings) != 0 {
			t.Errorf("%s: warnings %v", name, p.Warnings)
		}
	}
	// a link to a site, an unknown entry, a glob and an unknown option
	p := page(t, "```{toctree}\n:glob:\n:titlesonly:\n:bogus: 1\n\nhttps://example.com\nSite <https://example.org/x>\nmissing/page\nguide/*\n```\n")
	contains(t, p.HTML, `href="https://example.com"`, ">Site</a>")
	for _, sub := range []string{"missing/page", "guide/*", "bogus"} {
		if warned(p, sub) != 1 {
			t.Errorf("want one warning for %q: %v", sub, p.Warnings)
		}
	}
	// hidden draws nothing
	p = page(t, "before\n\n```{toctree}\n:hidden:\n\nguide/start\n```\n\nafter\n")
	lacks(t, p.HTML, "Getting started")
	// hostile entries
	p = page(t, "```{toctree}\n<script>alert(1)</script> <guide/start>\n<img src=x onerror=alert(1)>\n```\n")
	inert(t, p.HTML)
	// an empty toctree warns
	p = page(t, ":::toctree\n:::\n")
	if len(p.Warnings) != 1 {
		t.Errorf("empty toctree: %v", p.Warnings)
	}
}

func TestKitClasses(t *testing.T) {
	p := page(t, "```{toctree}\nguide/start\n```\n")
	contains(t, p.HTML, "zen-block")
}

func TestParentFolderStaysInsideTheFolder(t *testing.T) {
	r := folder(t)
	p, err := r.Render("guide/p.md", []byte("{doc}`../plain` {doc}`start` {doc}`../../plain`\n"))
	if err != nil {
		t.Fatal(err)
	}
	contains(t, p.HTML, `data-path="plain.md"`, `data-path="guide/start.md"`)
	if strings.Count(p.HTML, `data-zen-link="internal"`) != 2 {
		t.Errorf("only the two inside the folder link:\n%s", p.HTML)
	}
	if warned(p, "{doc} needs") != 1 {
		t.Errorf("warnings %v", p.Warnings)
	}
}
