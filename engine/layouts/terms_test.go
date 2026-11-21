package layouts

import (
	"strings"
	"testing"
)

type folderSite struct {
	fakeSite
	pages []FolderPage
	asked []string
}

func (f *folderSite) Folder(path string) []FolderPage {
	f.asked = append(f.asked, path)
	return f.pages
}

func termPage(title, file, def string, meta map[string]any) FolderPage {
	if meta == nil {
		meta = map[string]any{}
	}
	meta["title"] = title
	meta["layout"] = "term"
	return FolderPage{Title: title, Path: "g/" + file, Meta: meta, Definition: def}
}

func layoutOf(path string, meta map[string]any) string {
	if s, ok := meta["layout"].(string); ok {
		return s
	}
	return "article"
}

const glossTpl = `{{with .Glossary}}{{.Count}}|{{range .Letters}}{{.Label}}{{if .Has}}+{{end}} {{end}}|{{range .Groups}}{{.ID}}={{.Letter}}:{{range .Terms}}{{.Title}}({{.Path}}){{end}};{{end}}{{end}}`

func glossary(t *testing.T, pages []FolderPage) (string, []string) {
	t.Helper()
	l := mustLoad(t, "glossary", map[string]string{"layout.yml": "id: glossary\nversion: 1.0.0\ntemplate: t.html\n", "t.html": glossTpl})
	res, err := l.Apply(Input{Title: "Glossary", Path: "g/index.md", Site: &folderSite{pages: pages}, LayoutOf: layoutOf})
	if err != nil {
		t.Fatal(err)
	}
	return res.HTML, res.Warnings
}

func TestGlossarySortsGroupsAndCounts(t *testing.T) {
	html, warns := glossary(t, []FolderPage{
		termPage("zebra", "z.md", "An animal.", nil),
		termPage("Éclair", "e.md", "A cake.", nil),
		termPage("apple", "a.md", "A fruit.", nil),
		termPage("42", "n.md", "A number.", nil),
		termPage("Äpfel", "ae.md", "Fruits.", nil),
		{Title: "Notes", Path: "g/notes.md", Meta: map[string]any{"layout": "article"}, Definition: "Not a term."},
	})
	if len(warns) != 0 {
		t.Errorf("warnings %v", warns)
	}
	// Case and accents ignored: apple, Äpfel (apfel), Éclair, zebra; digits under #.
	for _, want := range []string{"5|", "A+ B C ", "E+ ", "Z+ ", "#+", "letter-a=A:apple(g/a.md)Äpfel(g/ae.md);", "letter-e=E:Éclair(g/e.md);", "letter-z=Z:zebra(g/z.md);", "letter-other=#:42(g/n.md);"} {
		if !strings.Contains(html, want) {
			t.Errorf("missing %q in %s", want, html)
		}
	}
	if strings.Contains(html, "Notes") {
		t.Errorf("a non-term page is listed: %s", html)
	}
	if strings.Contains(html, "B+") || strings.Contains(html, "Q+") {
		t.Errorf("empty letters marked: %s", html)
	}
}

func TestGlossaryProblems(t *testing.T) {
	_, warns := glossary(t, []FolderPage{
		termPage("Idempotent", "i.md", "Twice is once.", map[string]any{"aliases": []any{"retry"}}),
		termPage("idempotent", "i2.md", "Again.", nil),
		termPage("Retry", "r.md", "Try again.", map[string]any{"see": []any{"missing.md", "i.md"}}),
		termPage("Empty", "e.md", "", nil),
	})
	all := strings.Join(warns, "\n")
	for _, want := range []string{
		`duplicate term "Idempotent"`, `alias "retry" of "Idempotent" is also the term "Retry"`,
		`"Empty" has no definition`, `see: "missing.md" is not a term`,
	} {
		if !strings.Contains(all, want) {
			t.Errorf("no warning %q in\n%s", want, all)
		}
	}
	if strings.Contains(all, `"i.md" is not a term`) {
		t.Errorf("a real term reported missing:\n%s", all)
	}
}

func TestGlossaryAskedOnlyWhenUsed(t *testing.T) {
	site := &folderSite{}
	l := mustLoad(t, "glossary", map[string]string{"layout.yml": "id: glossary\nversion: 1.0.0\ntemplate: t.html\n", "t.html": "<p>x</p>"})
	if _, err := l.Apply(Input{Path: "g/index.md", Site: site, LayoutOf: layoutOf}); err != nil {
		t.Fatal(err)
	}
	if len(site.asked) != 0 {
		t.Errorf("folder read although the template did not ask: %v", site.asked)
	}
}

func TestGlossaryWithoutFolderSite(t *testing.T) {
	l := mustLoad(t, "glossary", map[string]string{"layout.yml": "id: glossary\nversion: 1.0.0\ntemplate: t.html\n", "t.html": glossTpl})
	res, err := l.Apply(Input{Path: "g/index.md", Site: &fakeSite{}, LayoutOf: layoutOf})
	if err != nil || !strings.Contains(res.HTML, "0|") {
		t.Errorf("%v %s", err, res.HTML)
	}
}

const termTpl = `{{with .TermPage}}{{with .Prev}}<{{.Title}}({{.Path}}){{end}}|{{with .Next}}>{{.Title}}({{.Path}}){{end}}|{{range .See}}[{{.Title}}:{{.Path}}:{{.Missing}}]{{end}}{{end}}`

func termApply(t *testing.T, self string, content string, fm map[string]any, pages []FolderPage) (string, []string) {
	t.Helper()
	l := mustLoad(t, "term", map[string]string{"layout.yml": "id: term\nversion: 1.0.0\ntemplate: t.html\n", "t.html": termTpl})
	res, err := l.Apply(Input{Title: fm["title"].(string), Path: self, Content: content, Frontmatter: fm,
		Site: &folderSite{pages: pages}, LayoutOf: layoutOf})
	if err != nil {
		t.Fatal(err)
	}
	return res.HTML, res.Warnings
}

func TestTermPageNeighboursAndSee(t *testing.T) {
	pages := []FolderPage{
		termPage("Apple", "a.md", "A fruit.", nil),
		termPage("cherry", "c.md", "A fruit.", nil),
		termPage("Banana", "b.md", "A fruit.", nil), // the page being shown, as on disk (stale)
	}
	html, warns := termApply(t, "g/b.md", "<p>A yellow fruit.</p>",
		map[string]any{"title": "Banana", "layout": "term", "see": []any{"c.md", "./a.md", "gone.md"}}, pages)
	for _, want := range []string{"<Apple(g/a.md)", ">cherry(g/c.md)", "[cherry:g/c.md:false]", "[Apple:g/a.md:false]", "[gone.md::true]"} {
		if !strings.Contains(html, want) {
			t.Errorf("missing %q in %s", want, html)
		}
	}
	if len(warns) != 1 || !strings.Contains(warns[0], `see: "gone.md" is not a term`) {
		t.Errorf("warnings %v", warns)
	}
	// The first term has no previous, the last no next.
	html, _ = termApply(t, "g/a.md", "<p>A fruit.</p>", map[string]any{"title": "Apple", "layout": "term"}, pages)
	if !strings.HasPrefix(html, "|>Banana(g/b.md)|") {
		t.Errorf("first term: %s", html)
	}
}

func TestTermPageUsesTheShownText(t *testing.T) {
	// The unsaved title and text of the page itself win over the file on disk.
	pages := []FolderPage{termPage("Old title", "b.md", "old", nil), termPage("Mango", "m.md", "A fruit.", nil)}
	html, _ := termApply(t, "g/b.md", "<p>New.</p>", map[string]any{"title": "Zucchini", "layout": "term"}, pages)
	if !strings.HasPrefix(html, "<Mango(g/m.md)||") {
		t.Errorf("%s", html)
	}
}

func TestTermPageDefinitionChecks(t *testing.T) {
	_, warns := termApply(t, "g/b.md", "<h2>Examples</h2><p>x</p>", map[string]any{"title": "B", "layout": "term"}, nil)
	if len(warns) != 1 || !strings.Contains(warns[0], "has no definition") {
		t.Errorf("a missing definition: %v", warns)
	}
	_, warns = termApply(t, "g/b.md", "", map[string]any{"title": "B", "layout": "term"}, nil)
	if len(warns) != 1 || !strings.Contains(warns[0], "has no definition") {
		t.Errorf("an empty page: %v", warns)
	}
	_, warns = termApply(t, "g/b.md", "<p>"+strings.Repeat("word ", 61)+"</p>", map[string]any{"title": "B", "layout": "term"}, nil)
	if len(warns) != 1 || !strings.Contains(warns[0], "at most 60 words") {
		t.Errorf("a long definition: %v", warns)
	}
	_, warns = termApply(t, "g/b.md", "<p>"+strings.Repeat("word ", 60)+"</p>", map[string]any{"title": "B", "layout": "term"}, nil)
	if len(warns) != 0 {
		t.Errorf("60 words are fine: %v", warns)
	}
}

func TestFold(t *testing.T) {
	for in, want := range map[string]string{"Éclair": "eclair", "ÄÖÜß": "aouss", "Ñandú": "nandu", "Ørsted": "orsted", "Zażółć": "zazolc", "İstanbul": "istanbul", "abc": "abc"} {
		if got := foldKey(in); got != want {
			t.Errorf("foldKey(%q) = %q, want %q", in, got, want)
		}
	}
	for in, want := range map[string]string{"apple": "A", "Éclair": "E", "42": "#", "_x": "#", "": "#", "ß": "S"} {
		if got := letterOf(in); got != want {
			t.Errorf("letterOf(%q) = %q, want %q", in, got, want)
		}
	}
}
