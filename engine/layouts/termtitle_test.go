package layouts

import (
	"strings"
	"testing"
)

// The term is the page's H1; a title: in the frontmatter is read only when
// there is no H1.
func TestTermFromHeadingFirst(t *testing.T) {
	site := &folderSite{pages: []FolderPage{
		{Title: "Zebra", Path: "g/z.md", Meta: map[string]any{"layout": "term"}, Definition: "An animal."},
		{Title: "", Path: "g/old.md", Meta: map[string]any{"layout": "term", "title": "Apple"}, Definition: "A fruit."},
		{Title: "", Path: "g/bare-name.md", Meta: map[string]any{"layout": "term"}, Definition: "No title anywhere."},
	}}
	l := mustLoad(t, "glossary", map[string]string{"layout.yml": "id: glossary\nversion: 1.0.0\ntemplate: t.html\n", "t.html": glossTpl})
	res, err := l.Apply(Input{Title: "Glossary", Path: "g/index.md", Site: site, LayoutOf: layoutOf})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"letter-a=A:Apple(g/old.md);", "letter-b=B:bare-name(g/bare-name.md);", "letter-z=Z:Zebra(g/z.md);"} {
		if !strings.Contains(res.HTML, want) {
			t.Errorf("missing %q in %s", want, res.HTML)
		}
	}
}

// A term page shows its H1 once, as the big term, and the definition follows.
func TestTermPageShowsTheHeadingOnce(t *testing.T) {
	l := mustLoad(t, "term", map[string]string{
		"layout.yml": "id: term\nversion: 1.0.0\ntemplate: t.html\n",
		"t.html":     `<h1 class="big">{{.TermTitle}}</h1>{{.TermBody}}`,
	})
	res, err := l.Apply(Input{Title: "Idempotent", Path: "g/i.md", Content: `<h1 id="idempotent">Idempotent</h1>` + "\n<p>Twice is once.</p>", Site: &folderSite{}, LayoutOf: layoutOf})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(res.HTML, "Idempotent") != 1 || !strings.Contains(res.HTML, "<p>Twice is once.</p>") {
		t.Errorf("%s", res.HTML)
	}
	// No H1 in the text (an old file with title: in its frontmatter): the title is the term, the text is whole.
	res, _ = l.Apply(Input{Title: "Idempotent", Path: "g/i.md", Content: "<p>Twice is once.</p>", Site: &folderSite{}, LayoutOf: layoutOf})
	if !strings.Contains(res.HTML, `<h1 class="big">Idempotent</h1><p>Twice is once.</p>`) {
		t.Errorf("%s", res.HTML)
	}
}
