package diagram

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

// folderRenderer opens a folder with files and returns a renderer for it. With
// dialect it turns MyST on.
func folderRenderer(t *testing.T, files map[string]string, dialect bool) *render.Renderer {
	t.Helper()
	dir := t.TempDir()
	if dialect {
		files[".zenmd.yml"] = "markdown:\n  dialects: [myst]\n"
	}
	for name, body := range files {
		p := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
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
	r := render.New(reg, config.Load("", dir), ws)
	if dialect {
		dr, err := dialects.NewRegistry(myst.New())
		if err != nil {
			t.Fatal(err)
		}
		r = r.WithDialects(dr)
	}
	return r
}

func renderAt(t *testing.T, r *render.Renderer, page, src string) render.Page {
	t.Helper()
	p, err := r.Render(page, []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func pre(html string) string {
	i := strings.Index(html, "<pre>")
	j := strings.Index(html, "</pre>")
	if i < 0 || j < i {
		return ""
	}
	return html[i+5 : j]
}

func TestFileThreeForms(t *testing.T) {
	r := folderRenderer(t, map[string]string{"diagrams/flow.mmd": "graph LR\n  A --> B\n"}, false)
	want := "graph LR\n  A --&gt; B\n"
	for name, src := range map[string]string{
		"braces":  "```mermaid {file=diagrams/flow.mmd}\n```\n",
		"quoted":  "```mermaid {file=\"diagrams/flow.mmd\"}\n```\n",
		"arg":     "```mermaid diagrams/flow.mmd\n```\n",
		"line":    "```mermaid\n:file: diagrams/flow.mmd\n```\n",
		"blank":   "```mermaid\n\n:file: diagrams/flow.mmd\n```\n",
		"colon":   ":::diagram{file=diagrams/flow.mmd}\n:::\n",
		"colonln": ":::diagram\n:file: diagrams/flow.mmd\n:::\n",
	} {
		p := renderAt(t, r, "page.md", src)
		if got := pre(p.HTML); got != want {
			t.Errorf("%s: pre %q, want %q in %s", name, got, want, p.HTML)
		}
		if !strings.Contains(p.HTML, `class="zen-diagram"`) || strings.Contains(p.HTML, "zen-plugin-error") || len(p.Warnings) != 0 {
			t.Errorf("%s: %s %v", name, p.HTML, p.Warnings)
		}
		if n := strings.Count(p.HTML, `data-zen-plugin="diagram"`); n != 1 {
			t.Errorf("%s: %d wrappers", name, n)
		}
	}
}

func TestFileRootPath(t *testing.T) {
	r := folderRenderer(t, map[string]string{
		"diagrams/flow.mmd":      "root file",
		"a/b/diagrams/x.MMD":     "near file",
		"a/b/my flows/é.mermaid": "spaced",
	}, false)
	// two folders down: "/" is the root, a plain path is the page's folder
	for src, want := range map[string]string{
		"```mermaid /diagrams/flow.mmd\n```\n":            "root file",
		"```mermaid diagrams/x.MMD\n```\n":                "near file",
		"```mermaid ../../diagrams/flow.mmd\n```\n":       "root file",
		"```mermaid {file=\"my flows/é.mermaid\"}\n```\n": "spaced",
		"```mermaid my flows/é.mermaid\n```\n":            "spaced",
	} {
		p := renderAt(t, r, "a/b/page.md", src)
		if got := pre(p.HTML); got != want {
			t.Errorf("%q: %q, want %q (%v)", src, got, want, p.Warnings)
		}
	}
	// a page at the root
	p := renderAt(t, r, "page.md", "```mermaid diagrams/flow.mmd\n```\n")
	if got := pre(p.HTML); got != "root file" {
		t.Errorf("root page: %q", got)
	}
}

func TestFileEscaped(t *testing.T) {
	r := folderRenderer(t, map[string]string{"a.mmd": "A-->B<script>alert(1)</script>&\"x\"\r\nC\r\n"}, false)
	p := renderAt(t, r, "p.md", "```mermaid a.mmd\n```\n")
	if strings.Contains(p.HTML, "<script>") {
		t.Errorf("not escaped: %s", p.HTML)
	}
	if got := pre(p.HTML); got != "A--&gt;B&lt;script&gt;alert(1)&lt;/script&gt;&amp;&#34;x&#34;\nC\n" {
		t.Errorf("pre %q", got)
	}
}

func TestFileMyST(t *testing.T) {
	r := folderRenderer(t, map[string]string{"diagrams/flow.mmd": "graph TD\n  A --> B\n"}, true)
	for name, src := range map[string]string{
		"option":   "```{mermaid}\n:file: diagrams/flow.mmd\n```\n",
		"argument": "```{mermaid} diagrams/flow.mmd\n```\n",
		"diagram":  "```{diagram}\n:file: /diagrams/flow.mmd\n```\n",
	} {
		p := renderAt(t, r, "page.md", src)
		if got := pre(p.HTML); got != "graph TD\n  A --&gt; B\n" || len(p.Warnings) != 0 {
			t.Errorf("%s: %q %v in %s", name, got, p.Warnings, p.HTML)
		}
	}
}

func TestFileMissing(t *testing.T) {
	r := folderRenderer(t, map[string]string{"x.md": ""}, false)
	for src, msg := range map[string]string{
		"```mermaid gone.mmd\n```\n":           "gone.mmd was not found in the folder",
		"```mermaid ../../etc/passwd\n```\n":   "../../etc/passwd is outside the folder",
		"```mermaid {file=/}\n```\n":           "/ is not a file",
		"```mermaid\n:file: nope/x.mmd\n```\n": "nope/x.mmd was not found in the folder",
	} {
		p := renderAt(t, r, "page.md", src)
		if !strings.Contains(p.HTML, `class="zen-plugin-error"`) || !strings.Contains(p.HTML, "diagram</strong>: "+msg) {
			t.Errorf("%q: %s", src, p.HTML)
		}
		if len(p.Warnings) != 1 || !strings.Contains(p.Warnings[0], msg) {
			t.Errorf("%q: warnings %v", src, p.Warnings)
		}
		if strings.Contains(p.HTML, "<div class=\"zen-diagram\"") {
			t.Errorf("%q: drew a diagram", src)
		}
	}
	// no folder: a lone file
	reg := plugins.NewRegistry()
	_ = reg.Register(New())
	p := renderAt(t, render.New(reg, nil, nil), "a.md", "```mermaid a.mmd\n```\n")
	if !strings.Contains(p.HTML, "needs an open folder") || len(p.Warnings) != 1 {
		t.Errorf("no folder: %s %v", p.HTML, p.Warnings)
	}
}

func TestFileAndBody(t *testing.T) {
	r := folderRenderer(t, map[string]string{"a.mmd": "FROM FILE"}, false)
	const note = "the block's own text is ignored because it names a file"
	for name, src := range map[string]string{
		"line":   "```mermaid\n:file: a.mmd\ngraph TD\n  A --> B\n```\n",
		"braces": "```mermaid {file=a.mmd}\ngraph TD\n```\n",
		"arg":    "```mermaid a.mmd\ngraph TD\n```\n",
	} {
		p := renderAt(t, r, "page.md", src)
		if got := pre(p.HTML); got != "FROM FILE" {
			t.Errorf("%s: file must win, got %q", name, got)
		}
		n := 0
		for _, w := range p.Warnings {
			if strings.Contains(w, note) {
				n++
			}
		}
		if n != 1 || len(p.Warnings) != 1 {
			t.Errorf("%s: warnings %v", name, p.Warnings)
		}
	}
	// blank own text does not warn
	p := renderAt(t, r, "page.md", "```mermaid a.mmd\n\n  \n```\n")
	if len(p.Warnings) != 0 {
		t.Errorf("blank body warned: %v", p.Warnings)
	}
}

func TestFileEmpty(t *testing.T) {
	r := folderRenderer(t, map[string]string{"e.mmd": "", "w.mmd": " \n\r\n"}, false)
	for _, f := range []string{"e.mmd", "w.mmd"} {
		p := renderAt(t, r, "page.md", "```mermaid "+f+"\n```\n")
		if strings.Contains(p.HTML, "zen-diagram") || strings.Contains(p.HTML, "<pre>") {
			t.Errorf("%s: drew something: %s", f, p.HTML)
		}
		if len(p.Warnings) != 1 || !strings.Contains(p.Warnings[0], f+" is empty") {
			t.Errorf("%s: warnings %v", f, p.Warnings)
		}
	}
}

func TestFileNoneKeepsInline(t *testing.T) {
	r := folderRenderer(t, nil, false)
	p := renderAt(t, r, "page.md", "```mermaid\ngraph TD\n:file: not-first.mmd\n```\n")
	if !strings.Contains(pre(p.HTML), "graph TD") || len(p.Warnings) != 0 {
		t.Errorf("%s %v", p.HTML, p.Warnings)
	}
	// ":file:" with no path draws the block's own text
	p = renderAt(t, r, "page.md", "```mermaid\n:file:\ngraph TD\n```\n")
	if len(p.Warnings) != 0 || !strings.Contains(p.HTML, "zen-diagram") {
		t.Errorf("%s %v", p.HTML, p.Warnings)
	}
}

func TestHelpSnippets(t *testing.T) {
	files := map[string]string{"diagrams/flow.mmd": "graph LR\n  A --> B\n"}
	snips := New().Manifest().Help.Snippets
	var fileSnippet bool
	for _, s := range snips {
		r := folderRenderer(t, files, false)
		p := renderAt(t, r, "page.md", s.Markdown)
		if len(p.Warnings) != 0 || strings.Contains(p.HTML, "zen-plugin-error") || !strings.Contains(p.HTML, `class="zen-diagram"`) {
			t.Errorf("snippet %q: %v %s", s.Title, p.Warnings, p.HTML)
		}
		if s.Title == "A diagram from a file" {
			fileSnippet = true
			if !strings.Contains(s.Markdown, ":file:") || !strings.Contains(s.Markdown, "/") {
				t.Errorf("file snippet: %q", s.Markdown)
			}
		}
	}
	if !fileSnippet {
		t.Error("no snippet \"A diagram from a file\"")
	}
	if err := plugins.CheckHelp(New().Manifest().Help); err != nil {
		t.Error(err)
	}
}
