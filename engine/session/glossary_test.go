package session_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/zenmark-org/zenmd/engine/session"
	"github.com/zenmark-org/zenmd/engine/session/sessiontest"
)

func termSrc(title, def string, extra string) string {
	return "---\ntitle: " + title + "\nlayout: term\n" + extra + "---\n" + def + "\n"
}

func glossaryTree(t *testing.T) string {
	return sessiontest.Tree(t, map[string]string{
		"README.md":     "# Home\n\n[Glossary](g/index.md)\n",
		"g/index.md":    "---\ntitle: Glossary\nlayout: glossary\n---\nWords.\n",
		"g/alpha.md":    termSrc("Alpha", "The first letter.", "aliases: [first]\n"),
		"g/bravo.md":    termSrc("Bravo", "The second letter.", "abbr: B\nsee: [alpha.md]\n"),
		"g/notes.md":    "---\ntitle: Notes\n---\nNot a term.\n",
		"other/zulu.md": termSrc("Zulu", "Elsewhere.", ""),
	})
}

func openGlossary(t *testing.T, dir string) (*session.Session, *sessiontest.Sink) {
	t.Helper()
	ev := &sessiontest.Sink{}
	s := session.New(session.Options{ConfigDir: t.TempDir(), Events: ev, Watch: true})
	t.Cleanup(s.Close)
	if err := s.Open(dir); err != nil {
		t.Fatal(err)
	}
	return s, ev
}

func TestGlossaryIndexReadsSiblings(t *testing.T) {
	s, _ := openGlossary(t, glossaryTree(t))
	p, err := s.Page("g/index.md")
	if err != nil {
		t.Fatal(err)
	}
	if p.Layout.Name != "glossary" || len(p.Warnings) != 0 {
		t.Fatalf("layout %+v warnings %v", p.Layout, p.Warnings)
	}
	for _, want := range []string{"2 terms", `data-path="g/alpha.md"`, `data-path="g/bravo.md"`, "The first letter.", "first", "The second letter.", ">B<"} {
		if !strings.Contains(p.HTML, want) {
			t.Errorf("missing %q in %s", want, p.HTML)
		}
	}
	if strings.Contains(p.HTML, "Zulu") || strings.Contains(p.HTML, "Not a term") {
		t.Errorf("a page of another folder or a non-term is listed: %s", p.HTML)
	}
	// A term page links to its neighbours and its see-also terms.
	b, _ := s.Page("g/bravo.md")
	if len(b.Warnings) != 0 || !strings.Contains(b.HTML, "zl-term-prev") || strings.Contains(b.HTML, "zl-term-next") || !strings.Contains(b.HTML, "zl-term-see") {
		t.Errorf("term page: %v %s", b.Warnings, b.HTML)
	}
}

func TestGlossaryPreviewSeesUnsavedText(t *testing.T) {
	s, _ := openGlossary(t, glossaryTree(t))
	// The index being edited: new intro text, the terms still from the folder.
	p, err := s.Preview("g/index.md", "---\ntitle: Glossary\nlayout: glossary\n---\nNew intro.\n")
	if err != nil || !strings.Contains(p.HTML, "New intro.") || !strings.Contains(p.HTML, "2 terms") {
		t.Errorf("%v %s", err, p.HTML)
	}
	// A term being edited: its unsaved title decides where it sits.
	p, err = s.Preview("g/alpha.md", termSrc("Charlie", "Now third.", ""))
	if err != nil || !strings.Contains(p.HTML, `data-path="g/bravo.md"`) || !strings.Contains(p.HTML, "zl-term-prev") || strings.Contains(p.HTML, "zl-term-next") {
		t.Errorf("%v %s", err, p.HTML)
	}
}

// waitFor polls until ok, up to 4 seconds.
func waitFor(t *testing.T, what string, ok func() bool) {
	t.Helper()
	deadline := time.Now().Add(4 * time.Second)
	for time.Now().Before(deadline) {
		if ok() {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("timed out: %s", what)
}

func changedPaths(ev *sessiontest.Sink) map[string]session.PageChanged {
	out := map[string]session.PageChanged{}
	for _, e := range ev.All() {
		if pc, ok := e.Data.(session.PageChanged); ok && e.Name == session.EventPageChanged {
			out[pc.Path] = pc
		}
	}
	return out
}

func write(t *testing.T, dir, rel, content string) {
	t.Helper()
	full := filepath.Join(dir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// A term file that is added, edited, renamed or deleted reloads the open index of its folder.
func TestGlossaryIndexReloadsWhenATermChanges(t *testing.T) {
	dir := glossaryTree(t)
	s, ev := openGlossary(t, dir)
	if _, err := s.Page("g/index.md"); err != nil { // the index is the open page
		t.Fatal(err)
	}
	steps := []struct {
		name string
		do   func()
		term string // the file that changed
		want string // what the index then shows ("" = nothing more to check)
	}{
		{"added", func() { write(t, dir, "g/charlie.md", termSrc("Charlie", "The third letter.", "")) }, "g/charlie.md", "3 terms"},
		{"edited", func() { write(t, dir, "g/charlie.md", termSrc("Charlie", "A changed definition.", "")) }, "g/charlie.md", "A changed definition."},
		{"deleted", func() { os.Remove(filepath.Join(dir, "g/charlie.md")) }, "g/charlie.md", "2 terms"},
	}
	for _, st := range steps {
		ev.Reset()
		st.do()
		waitFor(t, st.name+": index reload", func() bool { _, ok := changedPaths(ev)["g/index.md"]; return ok })
		pc := changedPaths(ev)["g/index.md"]
		if !pc.Dependent {
			t.Errorf("%s: the index reload is not marked as dependent: %+v", st.name, pc)
		}
		if _, ok := changedPaths(ev)[st.term]; !ok {
			t.Errorf("%s: the changed file's own event is gone: %v", st.name, changedPaths(ev))
		}
		p, err := s.Page("g/index.md")
		if err != nil || !strings.Contains(p.HTML, st.want) {
			t.Errorf("%s: %v lacks %q: %s", st.name, err, st.want, p.HTML)
		}
	}
}

func TestGlossaryIndexNotReloadedByOtherFolders(t *testing.T) {
	dir := glossaryTree(t)
	s, ev := openGlossary(t, dir)
	if _, err := s.Page("g/index.md"); err != nil {
		t.Fatal(err)
	}
	ev.Reset()
	write(t, dir, "other/yankee.md", termSrc("Yankee", "Elsewhere too.", ""))
	waitFor(t, "the event of the new file", func() bool { _, ok := changedPaths(ev)["other/yankee.md"]; return ok })
	time.Sleep(150 * time.Millisecond)
	if _, ok := changedPaths(ev)["g/index.md"]; ok {
		t.Error("the index reloaded for a page of another folder")
	}
}

func TestGlossaryIndexNotReloadedWhenNotOpen(t *testing.T) {
	dir := glossaryTree(t)
	s, ev := openGlossary(t, dir)
	if _, err := s.Page("README.md"); err != nil {
		t.Fatal(err)
	}
	ev.Reset()
	write(t, dir, "g/charlie.md", termSrc("Charlie", "Third.", ""))
	waitFor(t, "the event of the new file", func() bool { _, ok := changedPaths(ev)["g/charlie.md"]; return ok })
	time.Sleep(150 * time.Millisecond)
	if _, ok := changedPaths(ev)["g/index.md"]; ok {
		t.Error("the index reloaded although another page is open")
	}
}

// A term page shows its neighbours, so it reloads when a sibling term comes or goes.
func TestTermPageReloadsWhenASiblingChanges(t *testing.T) {
	dir := glossaryTree(t)
	s, ev := openGlossary(t, dir)
	if _, err := s.Page("g/bravo.md"); err != nil {
		t.Fatal(err)
	}
	write(t, dir, "g/charlie.md", termSrc("Charlie", "Third.", ""))
	waitFor(t, "the term page reload", func() bool { _, ok := changedPaths(ev)["g/bravo.md"]; return ok })
	p, _ := s.Page("g/bravo.md")
	if !strings.Contains(p.HTML, `data-path="g/charlie.md"`) || !strings.Contains(p.HTML, "zl-term-next") {
		t.Errorf("no link to the new next term: %s", p.HTML)
	}
}
