package render

import (
	"strings"
	"testing"
)

func TestTitleRuleH1WinsOverFrontmatter(t *testing.T) {
	cases := []struct{ name, path, src, want string }{
		{"h1 wins", "a/b.md", "---\ntitle: Other\n---\n# The H1\n\nText\n", "The H1"},
		{"title only without an h1", "a/b.md", "---\ntitle: From front matter\n---\nText\n", "From front matter"},
		{"file name last", "a/my-page.md", "Text only\n", "my-page"},
		{"blank title falls to the file name", "a/x.md", "---\ntitle: \"  \"\n---\nText\n", "x"},
		{"h1 later in the page", "a/x.md", "---\ntitle: T\n---\nIntro\n\n# Late H1\n", "Late H1"},
	}
	r := New(nil, nil, nil)
	for _, c := range cases {
		p, err := r.Render(c.path, []byte(c.src))
		if err != nil {
			t.Fatal(err)
		}
		if p.Title != c.want {
			t.Errorf("%s: Render title %q, want %q", c.name, p.Title, c.want)
		}
		if got := r.Title(c.path, []byte(c.src)); got != c.want {
			t.Errorf("%s: Title %q, want %q", c.name, got, c.want)
		}
	}
}

const (
	noteIgnored = `title: is ignored because the page has an H1 ("The H1")`
	noteSame    = `title: is ignored because the page has an H1, and it says the same`
)

func warns(t *testing.T, src string) []string {
	t.Helper()
	p, err := New(nil, nil, nil).Render("x.md", []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, w := range p.Warnings {
		if strings.HasPrefix(w, "title:") {
			out = append(out, w)
		}
	}
	return out
}

func TestTitleNoteWhenBothAndDifferent(t *testing.T) {
	got := warns(t, "---\ntitle: Other\n---\n# The H1\n")
	if len(got) != 1 || got[0] != noteIgnored {
		t.Errorf("warnings %q, want [%q]", got, noteIgnored)
	}
}

func TestTitleNoteWhenBothAndEqual(t *testing.T) {
	got := warns(t, "---\ntitle: The H1\n---\n# The H1\n")
	if len(got) != 1 || got[0] != noteSame {
		t.Errorf("warnings %q, want [%q]", got, noteSame)
	}
}

func TestNoTitleNoteWithoutBoth(t *testing.T) {
	for _, src := range []string{"# Only H1\n", "---\ntitle: Only fm\n---\nText\n", "Text\n"} {
		if got := warns(t, src); len(got) != 0 {
			t.Errorf("%q: unexpected %q", src, got)
		}
	}
}

// Step 3 of the title rule: the field the page's layout names with title: in
// its layout.yml, shown as written. H1 and frontmatter title: come first, the
// file name last.
func TestTitleRuleLayoutField(t *testing.T) {
	rel := layoutFS(t, "release", "id: release\nversion: 1.0.0\ntitle: version\ntemplate: layout.html\nfields:\n  version: { type: string }\n", map[string]string{"layout.html": "<div>{{.Body}}</div>"})
	plain := layoutFS(t, "plain", "id: plain\nversion: 1.0.0\ntemplate: layout.html\nfields:\n  version: { type: string }\n", map[string]string{"layout.html": "<div>{{.Body}}</div>"})
	r := layoutRenderer(t, nil, rel, plain)
	cases := []struct{ name, path, src, want string }{
		{"h1 wins", "r/a.md", "---\nlayout: release\nversion: 0.1.0\n---\n# Big\n", "Big"},
		{"title: wins over the field", "r/a.md", "---\nlayout: release\nversion: 0.1.0\ntitle: Named\n---\n## Added\n", "Named"},
		{"the field", "r/a.md", "---\nlayout: release\nversion: 0.1.0\n---\n## Added\n", "0.1.0"},
		{"a number as written", "r/a.md", "---\nlayout: release\nversion: 2\n---\n## Added\n", "2"},
		{"field missing: file name", "r/a.md", "---\nlayout: release\n---\n## Added\n", "a"},
		{"layout names no field: file name", "r/b.md", "---\nlayout: plain\nversion: 0.1.0\n---\n## Added\n", "b"},
		{"no layout: file name", "r/c.md", "---\nversion: 0.1.0\n---\n## Added\n", "c"},
	}
	for _, c := range cases {
		p, err := r.Render(c.path, []byte(c.src))
		if err != nil {
			t.Fatal(err)
		}
		if p.Title != c.want {
			t.Errorf("%s: Render title %q, want %q", c.name, p.Title, c.want)
		}
		if got := r.Title(c.path, []byte(c.src)); got != c.want {
			t.Errorf("%s: Title %q, want %q", c.name, got, c.want)
		}
	}
}
