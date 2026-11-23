package session

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/zenmark-org/zenmd/engine/plugins"
)

// openFileZen opens one file the way the app does, with a private settings
// folder (nothing here reads the real one).
func openFileZen(t *testing.T, file, cfgDir string, confirm bool) (*Session, *sink, *fakeDialogs) {
	t.Helper()
	ev, dl := &sink{}, &fakeDialogs{Allow: confirm}
	z, err := newZen(zenOptions{Registry: plugins.NewRegistry(), ConfigDir: cfgDir, Events: ev, Dialogs: dl, DeferOpen: true})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(z.close)
	if err := z.OpenFile(file); err != nil {
		t.Fatal(err)
	}
	return z, ev, dl
}

func TestSingleFileLinks(t *testing.T) {
	dir := tree(t, map[string]string{
		"docs/notes.md":  "# Notes\n\n## First\n\n[b](b.md) [out](../x.md) [gone](nope.md) [up](../docs/b.md)\n",
		"docs/b.md":      "# B\n\n[back](notes.md)\n",
		"docs/other.md":  "# Other\n",
		"docs/sub/c.md":  "# C\n",
		"x.md":           "# Outside\n",
		"docs/img/a.png": "\x89PNG\r\n\x1a\n",
	})
	z, _, _ := openFileZen(t, filepath.Join(dir, "docs", "notes.md"), t.TempDir(), false)

	ws := z.Workspace()
	if ws == nil || !ws.Single || ws.File != "notes.md" || ws.Home != "notes.md" || ws.Name != "notes.md" || ws.Title != "Notes" {
		t.Fatalf("workspace: %+v", ws)
	}
	if len(ws.Tree) != 1 || ws.Tree[0].Path != "notes.md" {
		t.Fatalf("one entry, no scan of the folder: %+v", ws.Tree)
	}
	pg, err := z.Page("notes.md")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`data-zen-link="internal"`, `data-path="b.md"`, // a sibling opens in the same window
		`data-zen-link="broken"`, // above the folder, and missing
	} {
		if !strings.Contains(pg.HTML, want) {
			t.Errorf("page lacks %s:\n%s", want, pg.HTML)
		}
	}
	if strings.Count(pg.HTML, `data-zen-link="broken"`) != 3 {
		t.Errorf("want 3 broken links (two above the folder, one missing):\n%s", pg.HTML)
	}
	// Following the link: the sibling page renders, and the window stays single.
	b, err := z.Page("b.md")
	if err != nil || b.Title != "B" {
		t.Fatalf("sibling: %+v %v", b, err)
	}
	if sub, err := z.Page("sub/c.md"); err != nil || sub.Title != "C" {
		t.Fatalf("below the folder: %+v %v", sub, err)
	}
	if got := z.Workspace(); !got.Single || len(got.Tree) != 1 {
		t.Fatalf("a followed link changed the window: %+v", got)
	}
	// Above the folder: the usual outside notice, and no absolute path.
	if _, err := z.Page("../x.md"); err == nil || !strings.Contains(err.Error(), "outside") || strings.Contains(err.Error(), dir) {
		t.Fatalf("outside: %v", err)
	}
	if _, err := z.Source("../x.md"); err == nil {
		t.Fatal("source above the folder")
	}
	if _, err := z.Page("nope.md"); err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("missing: %v", err)
	}
	// Images next to the file still load.
	if !strings.Contains(pg.HTML, "/_ws/") && strings.Contains(pg.HTML, "img") {
		t.Errorf("asset: %s", pg.HTML)
	}

	// The newest open wins, file then folder and folder then file.
	if err := z.Open(filepath.Join(dir, "docs")); err != nil {
		t.Fatal(err)
	}
	if w := z.Workspace(); w.Single || w.File != "" || len(w.Tree) < 3 {
		t.Fatalf("folder after file: %+v", w)
	}
	if err := z.OpenFile(filepath.Join(dir, "docs", "other.md")); err != nil {
		t.Fatal(err)
	}
	if w := z.Workspace(); !w.Single || w.File != "other.md" {
		t.Fatalf("file after folder: %+v", w)
	}
	// A file that cannot open leaves what was open.
	if err := z.OpenFile(filepath.Join(dir, "docs", "img", "a.png")); err == nil {
		t.Fatal("a png opened as a page")
	}
	if err := z.OpenFile(filepath.Join(dir, "docs", "missing.md")); err == nil {
		t.Fatal("a missing file opened")
	}
	if w := z.Workspace(); w == nil || w.File != "other.md" {
		t.Fatalf("a failed open replaced the window: %+v", w)
	}
	// Open given a markdown file opens it alone.
	if err := z.Open(filepath.Join(dir, "docs", "b.md")); err != nil {
		t.Fatal(err)
	}
	if w := z.Workspace(); !w.Single || w.File != "b.md" {
		t.Fatalf("Open(file): %+v", w)
	}
}

func TestSingleFileConfig(t *testing.T) {
	dir := tree(t, map[string]string{
		".zenmd.yml": "version: 1\nlayout: docs\nlogo: logo.svg\n",
		"logo.svg":   `<svg xmlns="http://www.w3.org/2000/svg"/>`,
		"plain.md":   "# Plain\n",
		"slides.md":  "---\nlayout: slides\n---\n# Slides\n",
		"other.md":   "# Other\n",
	})
	// As a folder, the folder's settings apply.
	f, _, _, _ := newTestZen(t, dir, "", "read")
	if pg, _ := f.Page("plain.md"); pg.Layout.Name != "docs" {
		t.Fatalf("the folder's layout: %+v", pg.Layout)
	}
	if w := f.Workspace(); w.Logo == "" || w.ConfigFile != ".zenmd.yml" {
		t.Fatalf("the folder's logo and settings file: %+v", w)
	}

	// Alone, only the global config.yml does: the .zenmd.yml next to it is ignored.
	cfg := t.TempDir()
	z, _, _ := openFileZen(t, filepath.Join(dir, "plain.md"), cfg, false)
	pg, err := z.Page("plain.md")
	if err != nil {
		t.Fatal(err)
	}
	if pg.Layout.Name != "article" || pg.Layout.Source != "builtin" {
		t.Errorf("no layout: is article, got %+v", pg.Layout)
	}
	if w := z.Workspace(); w.Logo != "" || w.ConfigFile != "" {
		t.Errorf("the folder's settings leaked in: %+v", w)
	}
	// The page's own frontmatter still applies, to the file and to a followed link.
	sl, err := z.Page("slides.md")
	if err != nil || sl.Layout.Name != "slides" || sl.Layout.Source != "builtin" {
		t.Errorf("frontmatter layout: %+v %v", sl.Layout, err)
	}
	// The global config does apply.
	global := filepath.Join(t.TempDir(), "config.yml")
	if err := os.WriteFile(global, []byte("version: 1\nlayout: wide\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	g := New(Options{GlobalPath: global, Registry: plugins.NewRegistry()})
	t.Cleanup(g.Close)
	if err := g.OpenFile(filepath.Join(dir, "plain.md")); err != nil {
		t.Fatal(err)
	}
	if pg, _ := g.Page("plain.md"); pg.Layout.Name != "wide" {
		t.Errorf("the global layout: %+v", pg.Layout)
	}
	// The folder's own settings are not written from here either.
	if _, _, err := z.ConfigFile("workspace"); err == nil {
		t.Error("the folder's settings file is offered for one file")
	}
	if _, err := z.BuildOptions(); err == nil {
		t.Error("Export › Website for one file")
	}
}

func TestSingleFileSaveAndWatch(t *testing.T) {
	dir := tree(t, map[string]string{"notes.md": "# Notes\n", "other.md": "# Other\n"})
	file := filepath.Join(dir, "notes.md")
	z, ev, _ := openFileZen(t, file, t.TempDir(), false)

	// Saving writes the one file, atomically: no temp file stays.
	if err := z.Save("notes.md", "# Saved\n"); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(file); string(b) != "# Saved\n" {
		t.Fatalf("saved: %q", b)
	}
	left, _ := filepath.Glob(filepath.Join(dir, ".*.tmp"))
	if len(left) != 0 {
		t.Fatalf("temp files left: %v", left)
	}
	// New Page is off.
	if _, err := z.CreatePage("fresh", "article"); err == nil {
		t.Error("New Page in a one-file window")
	}
	if _, err := os.Stat(filepath.Join(dir, "fresh.md")); err == nil {
		t.Error("a page was created next to the file")
	}

	// The watcher watches the file alone.
	time.Sleep(300 * time.Millisecond)
	ev.reset()
	if err := os.WriteFile(filepath.Join(dir, "other.md"), []byte("# Other 2\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "new.md"), []byte("# New\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	time.Sleep(600 * time.Millisecond)
	if got := ev.all(); len(got) != 0 {
		t.Fatalf("a sibling made events: %+v", got)
	}
	// An edit from outside (an atomic rename, as editors save) reloads the page.
	tmp := filepath.Join(dir, "notes.md.edit")
	if err := os.WriteFile(tmp, []byte("# Edited outside\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(tmp, file); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(3 * time.Second)
	for {
		changed, titled := false, false
		for _, e := range ev.all() {
			if e.Name == "page:changed" && e.Data == (PageChanged{Path: "notes.md"}) {
				changed = true
			}
			if e.Name == "workspace:changed" {
				if w, ok := e.Data.(*Workspace); ok && w.Single && w.Title == "Edited outside" {
					titled = true
				}
			}
		}
		if changed && titled {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("no reload after an outside edit: %+v", ev.all())
		}
		time.Sleep(20 * time.Millisecond)
	}
	// The file is deleted while open: the existing "not found" answer.
	if err := os.Remove(file); err != nil {
		t.Fatal(err)
	}
	if _, err := z.Page("notes.md"); err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("a deleted file: %v", err)
	}
}

// The plugins of the file's folder are asked about exactly as for a folder,
// and the answer is remembered by the folder.
func TestSingleFileTrust(t *testing.T) {
	files := pluginDocs()
	dir := pluginTree(t, files)
	cfg := t.TempDir()
	ran := filepath.Join(dir, "ran.txt")

	z, _, dl := openFileZen(t, filepath.Join(dir, "README.md"), cfg, false)
	if len(dl.Asked) != 1 || !strings.Contains(dl.Asked[0], "hello") || !strings.Contains(dl.Asked[0], filepath.Join(".zen", "plugins", "hello")) {
		t.Fatalf("asked: %q", dl.Asked)
	}
	pg, err := z.Page("README.md")
	if err != nil {
		t.Fatal(err)
	}
	if exists(ran) || !hasWarning(pg, notAllowed) {
		t.Fatalf("a declined plugin ran or was not reported: %v", pg.Warnings)
	}
	if got := infoOf(t, z, "hello"); got.Status != "blocked" || got.Source != "folder" {
		t.Errorf("info: %+v", got)
	}

	// The same folder opened as a folder: the answer was remembered by the folder.
	f, _, dl2 := openWith(t, dir, cfg, true)
	if len(dl2.Asked) != 0 {
		t.Fatalf("asked again for the folder: %q", dl2.Asked)
	}
	if got := infoOf(t, f, "hello"); got.Status != "blocked" {
		t.Errorf("the folder lost the decision: %+v", got)
	}
	// A file in another folder is not covered by it.
	other := pluginTree(t, files)
	_, _, dl3 := openFileZen(t, filepath.Join(other, "README.md"), cfg, true)
	if len(dl3.Asked) != 1 {
		t.Fatalf("another folder's plugin was not asked about: %q", dl3.Asked)
	}
}

// issue #50: closing a window leaves nothing to refresh. plugins:changed and layouts:changed make the open page render
// again (runtime/src/plugins.ts), so one sent while closing asks Go for a page of a folder that is gone.
func TestCloseSendsNoRefreshEvents(t *testing.T) {
	for _, single := range []bool{true, false} {
		dir := tree(t, map[string]string{"notes.md": "# Notes\n", "b.md": "# B\n"})
		var z *Session
		var ev *sink
		if single {
			z, ev, _ = openFileZen(t, filepath.Join(dir, "notes.md"), t.TempDir(), false)
		} else {
			ev = &sink{}
			var err error
			z, err = newZen(zenOptions{Registry: plugins.NewRegistry(), ConfigDir: t.TempDir(), Events: ev, Dialogs: &fakeDialogs{}, DeferOpen: true})
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(z.close)
			if err := z.Open(dir); err != nil {
				t.Fatal(err)
			}
		}
		if _, err := z.Page("notes.md"); err != nil {
			t.Fatal(err)
		}
		ev.reset()
		if err := z.Open(""); err != nil {
			t.Fatal(err)
		}
		for _, e := range ev.all() {
			if e.name == EventPluginsChanged || e.name == EventLayoutsChanged || e.name == EventPageChanged {
				t.Errorf("single=%v: closing sent %s", single, e.name)
			}
		}
		if _, err := z.Page("notes.md"); err == nil {
			t.Errorf("single=%v: a page of a closed folder", single)
		}
	}
}

// issue #50: the log says "file" for one file: opened, switched, closed.
func TestFileLogging(t *testing.T) {
	h := capture(t)
	dir := tree(t, map[string]string{"notes.md": "# Notes\n", "other.md": "# Other\n", "f/index.md": "# F\n"})
	z, _, _ := openFileZen(t, filepath.Join(dir, "notes.md"), t.TempDir(), false)
	mustContain(t, h.all(), "INFO file opened", "name=notes.md", "pages=1")
	if strings.Contains(h.all(), "folder opened") {
		t.Errorf("a file logged as a folder:\n%s", h.all())
	}

	if err := z.OpenFile(filepath.Join(dir, "other.md")); err != nil {
		t.Fatal(err)
	}
	mustContain(t, h.all(), "INFO file switched", "name=other.md", "from=notes.md")

	if err := z.Open(filepath.Join(dir, "f")); err != nil { // a file, then a folder
		t.Fatal(err)
	}
	mustContain(t, h.all(), "INFO folder switched", "name=f", "from=other.md")

	if err := z.OpenFile(filepath.Join(dir, "notes.md")); err != nil { // a folder, then a file
		t.Fatal(err)
	}
	mustContain(t, h.all(), "INFO file switched", "name=notes.md", "from=f")

	if err := z.Open(""); err != nil {
		t.Fatal(err)
	}
	mustContain(t, h.all(), "INFO file closed", "name=notes.md")
	if strings.Contains(h.all(), "folder closed") {
		t.Errorf("a file closed as a folder:\n%s", h.all())
	}
}
