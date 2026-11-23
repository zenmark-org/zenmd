package app

import (
	"os"
	"path/filepath"
	"testing"
)

// pathDialogs is the dialogs of the tests with a ⌘O panel that answers one path.
type pathDialogs struct {
	fakeDialogs
	path string
	err  error
}

func (d *pathDialogs) PickPath() (string, error) { return d.path, d.err }

// openedEvents are the workspace:opened events the sink got, as Startup.
func openedEvents(ev *sink) []Startup {
	var out []Startup
	for _, e := range ev.all() {
		if e.Name == EventWorkspaceOpened {
			if s, ok := e.Data.(Startup); ok {
				out = append(out, s)
			}
		}
	}
	return out
}

// homeTree is a fake home with files below it (the marker search stops at home, and never sees the real one).
func homeTree(t *testing.T, files map[string]string) string {
	t.Helper()
	h := t.TempDir()
	t.Setenv("HOME", h)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(h, ".config"))
	for p, c := range files {
		full := filepath.Join(h, filepath.FromSlash(p))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(c), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return h
}

// ⌘O takes a folder or a markdown file; a file opens alone unless it is in a project folder; Open With uses the
// same rules.
func TestOpenPicksMode(t *testing.T) {
	h := homeTree(t, map[string]string{
		"Downloads/notes.md": "# Notes\n\n## Part\n",
		"Downloads/other.md": "# Other\n",
		"Downloads/pic.png":  "\x89PNG",
		"proj/.zenmd.yml":    "version: 1\n",
		"proj/README.md":     "# Project\n",
		"proj/docs/guide.md": "# Guide\n",
		"folder/index.md":    "# Folder\n",
		"folder/sub/page.md": "# Page\n",
	})
	z, ev, dl, _ := newTestZen(t, "", "", "")
	pd := &pathDialogs{fakeDialogs: *dl}
	z.dialogs = pd

	// A file alone.
	pd.path = filepath.Join(h, "Downloads", "notes.md")
	if err := z.openPicked(); err != nil {
		t.Fatal(err)
	}
	got := openedEvents(ev)
	if len(got) != 1 || got[0].Workspace == nil || !got[0].Workspace.Single || got[0].Workspace.File != "notes.md" || got[0].Page != "notes.md" {
		t.Fatalf("a file opens alone: %+v", got)
	}
	if st := z.Startup(); st.Workspace == nil || !st.Workspace.Single || len(st.Workspace.Tree) != 1 {
		t.Fatalf("startup: %+v", st)
	}

	// A folder.
	ev.reset()
	pd.path = filepath.Join(h, "folder")
	if err := z.openPicked(); err != nil {
		t.Fatal(err)
	}
	got = openedEvents(ev)
	if len(got) != 1 || got[0].Workspace.Single || got[0].Page != "index.md" || len(got[0].Workspace.Tree) == 0 {
		t.Fatalf("a folder opens as a folder: %+v", got)
	}

	// A file inside a project folder: the folder at that page, as zenmd does.
	ev.reset()
	pd.path = filepath.Join(h, "proj", "docs", "guide.md")
	if err := z.openPicked(); err != nil {
		t.Fatal(err)
	}
	got = openedEvents(ev)
	if len(got) != 1 || got[0].Workspace.Single || got[0].Workspace.Name != "proj" || got[0].Page != "docs/guide.md" {
		t.Fatalf("a file of a project opens the project: %+v", got)
	}

	// Cancel changes nothing; neither does a file that is not markdown.
	ev.reset()
	pd.path = ""
	if err := z.openPicked(); err != nil || len(openedEvents(ev)) != 0 {
		t.Fatalf("cancel: %v %+v", err, openedEvents(ev))
	}
	pd.path = filepath.Join(h, "Downloads", "pic.png")
	if err := z.openPicked(); err == nil || len(openedEvents(ev)) != 0 {
		t.Fatalf("a png: %v", err)
	}
	if w := z.Startup().Workspace; w == nil || w.Name != "proj" {
		t.Fatalf("a refused file replaced the window: %+v", w)
	}

	// The bound Open takes a path too; it must be absolute and exist.
	ev.reset()
	if err := z.Open(filepath.Join(h, "Downloads", "other.md")); err != nil {
		t.Fatal(err)
	}
	if got := openedEvents(ev); len(got) != 1 || !got[0].Workspace.Single || got[0].Page != "other.md" {
		t.Fatalf("Open(file): %+v", got)
	}
	for _, bad := range []string{"Downloads/notes.md", filepath.Join(h, "nope.md"), filepath.Join(h, "Downloads", "pic.png"), ""} {
		if err := z.Open(bad); err == nil {
			t.Errorf("Open(%q) did not fail", bad)
		}
	}
	// A cancelled-and-replaced open is not an error: the dialog's folder when the panel is the old one.
	z.dialogs = dl // no PickPath: the folder panel
	dl.folder = filepath.Join(h, "folder")
	ev.reset()
	if err := z.openPicked(); err != nil || len(openedEvents(ev)) != 1 {
		t.Fatalf("a panel with folders only: %v %+v", err, openedEvents(ev))
	}

	// Finder's Open With: the same rules, the answer is the Startup the window shows.
	st, err := z.openAny(filepath.Join(h, "Downloads", "notes.md"))
	if err != nil || st.Workspace == nil || !st.Workspace.Single || st.Page != "notes.md" {
		t.Fatalf("Open With a file: %+v %v", st, err)
	}
	st, err = z.openAny(filepath.Join(h, "proj", "docs", "guide.md"))
	if err != nil || st.Workspace == nil || st.Workspace.Single || st.Page != "docs/guide.md" {
		t.Fatalf("Open With a project's file: %+v %v", st, err)
	}
	st, err = z.openAny(filepath.Join(h, "folder"))
	if err != nil || st.Workspace == nil || st.Workspace.Single {
		t.Fatalf("Open With a folder: %+v %v", st, err)
	}
	if _, err := z.openAny(filepath.Join(h, "Downloads", "pic.png")); err == nil {
		t.Fatal("Open With a png")
	}
}

// A file given on the command line opens alone too.
func TestLaunchFileOpensAlone(t *testing.T) {
	h := homeTree(t, map[string]string{"Downloads/notes.md": "# Notes\n", "Downloads/other.md": "# Other\n"})
	z, _, _, _ := newTestZen(t, filepath.Join(h, "Downloads", "notes.md"), "", "read")
	st := z.Startup()
	if st.Workspace == nil || !st.Workspace.Single || st.Page != "notes.md" || len(st.Workspace.Tree) != 1 {
		t.Fatalf("startup: %+v", st)
	}
}

// Open Enclosing Folder opens the file's project folder at this page: the nearest marker folder, else the
// file's own folder.
func TestOpenEnclosingFolder(t *testing.T) {
	h := homeTree(t, map[string]string{
		"proj/.zenmd.yml":    "version: 1\n",
		"proj/docs/guide.md": "# Guide\n",
		"proj/docs/more.md":  "# More\n",
		"Downloads/notes.md": "# Notes\n",
		"Downloads/other.md": "# Other\n",
	})
	z, ev, _, _ := newTestZen(t, "", "", "")

	// No file open on its own: nothing to do.
	if err := z.OpenEnclosingFolder(); err == nil {
		t.Fatal("no window")
	}
	if err := z.s.Open(filepath.Join(h, "Downloads")); err != nil {
		t.Fatal(err)
	}
	if err := z.OpenEnclosingFolder(); err == nil {
		t.Fatal("a folder is already a folder")
	}

	// A file in a project (forced alone, as zenmd --file): the project, at that page.
	if err := z.s.OpenFile(filepath.Join(h, "proj", "docs", "guide.md")); err != nil {
		t.Fatal(err)
	}
	ev.reset()
	if err := z.OpenEnclosingFolder(); err != nil {
		t.Fatal(err)
	}
	got := openedEvents(ev)
	if len(got) != 1 || got[0].Workspace.Single || got[0].Workspace.Name != "proj" || got[0].Page != "docs/guide.md" {
		t.Fatalf("the project: %+v", got)
	}
	if w := z.s.Workspace(); w.Single || w.Name != "proj" {
		t.Fatalf("session: %+v", w)
	}

	// No marker: the file's own folder, at the file.
	if err := z.s.OpenFile(filepath.Join(h, "Downloads", "notes.md")); err != nil {
		t.Fatal(err)
	}
	ev.reset()
	if err := z.OpenEnclosingFolder(); err != nil {
		t.Fatal(err)
	}
	got = openedEvents(ev)
	if len(got) != 1 || got[0].Workspace.Single || got[0].Workspace.Name != "Downloads" || got[0].Page != "notes.md" {
		t.Fatalf("the file's folder: %+v", got)
	}
	// The same as zenmd <file> --folder: a page of the folder, not the file alone.
	if files := len(z.s.Workspace().Tree); files != 2 {
		t.Fatalf("the folder was not scanned: %d pages", files)
	}
}

// The menu: New Page and Export › Website are off for one file, Open Enclosing Folder is on only then.
func TestMenuStateSingle(t *testing.T) {
	for _, l := range []string{labelNewPage, labelExportWebsite} {
		if !folderOnly(l) || !realFolderOnly(l) {
			t.Errorf("%q needs a real folder", l)
		}
	}
	for _, l := range []string{labelExportPDF, labelPrintPage, labelFind, labelCloseFolder, labelExport} {
		if realFolderOnly(l) {
			t.Errorf("%q works for one file", l)
		}
	}
	if !singleOnly(labelOpenEnclosing) || folderOnly(labelOpenEnclosing) || singleOnly(labelOpen) {
		t.Error("only Open Enclosing Folder is for one file alone")
	}
	st := newMenuState(false)
	any, real, one := &fakeItem{enabled: true}, &fakeItem{enabled: true}, &fakeItem{enabled: true}
	st.add(any.SetEnabled)
	st.addReal(real.SetEnabled)
	st.addSingle(one.SetEnabled)
	if any.enabled || real.enabled || one.enabled {
		t.Fatal("all start off with nothing open")
	}
	st.setOpen(true, false)
	if !any.enabled || !real.enabled || one.enabled {
		t.Fatalf("a folder: %v %v %v", any.enabled, real.enabled, one.enabled)
	}
	st.setOpen(true, true)
	if !any.enabled || real.enabled || !one.enabled {
		t.Fatalf("one file: %v %v %v", any.enabled, real.enabled, one.enabled)
	}
	st.setOpen(false, false)
	if any.enabled || real.enabled || one.enabled {
		t.Fatal("closed")
	}

	// folderOpened reads the session: the items follow what is open.
	h := homeTree(t, map[string]string{"a/n.md": "# N\n"})
	z, _, _, _ := newTestZen(t, "", "", "")
	z.menu = newMenuState(false)
	a, b, c := &fakeItem{}, &fakeItem{}, &fakeItem{}
	z.menu.add(a.SetEnabled)
	z.menu.addReal(b.SetEnabled)
	z.menu.addSingle(c.SetEnabled)
	if err := z.s.OpenFile(filepath.Join(h, "a", "n.md")); err != nil {
		t.Fatal(err)
	}
	z.folderOpened(true)
	if !a.enabled || b.enabled || !c.enabled {
		t.Fatalf("after a file: %v %v %v", a.enabled, b.enabled, c.enabled)
	}
	if err := z.s.Open(filepath.Join(h, "a")); err != nil {
		t.Fatal(err)
	}
	z.folderOpened(true)
	if !a.enabled || !b.enabled || c.enabled {
		t.Fatalf("after a folder: %v %v %v", a.enabled, b.enabled, c.enabled)
	}
}
