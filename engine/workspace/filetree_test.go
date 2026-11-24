package workspace

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// specsShape mirrors the repo's own specs/ folder: scope folders holding feature folders with a spec.md,
// a README at the top, a template folder, a page with the same name as a folder and images in a subfolder.
var specsShape = map[string]string{
	"README.md":                 "# Specs\n",
	"_template/spec.md":         "# Template\n",
	"engine/index/spec.md":      "# Index\n",
	"engine/session/spec.md":    "# Session\n",
	"ui/files-view/spec.md":     "# Files\n",
	"ui/files-view/shots/a.png": "png",
	"ui/files-view/shots/b.PNG": "png",
	"ui.md":                     "# UI\n",
	"plugins/10-setup/spec.md":  "# Setup\n",
	"plugins/2-intro/spec.md":   "# Intro\n",
}

// paths flattens a listing to "kind path" lines in display order.
func flat(es []FileEntry) []string {
	var out []string
	for _, e := range es {
		out = append(out, e.Kind+" "+e.Path)
		out = append(out, flat(e.Children)...)
	}
	return out
}

func tree(t *testing.T, w *Workspace) ([]FileEntry, int) {
	t.Helper()
	es, omitted, err := w.FileTree(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	return es, omitted
}

func TestSkipDirMatchesScan(t *testing.T) {
	cases := []struct {
		name   string
		atRoot bool
		skip   bool
	}{
		{".git", false, true}, {".hidden", true, true}, {"node_modules", false, true}, {"vendor", true, true},
		{"exports", true, true}, {"exports", false, false}, {"docs", true, false}, {"Vendor", false, false},
	}
	for _, c := range cases {
		if got := skipDir(c.name, c.atRoot); got != c.skip {
			t.Errorf("skipDir(%q, %v) = %v, want %v", c.name, c.atRoot, got, c.skip)
		}
	}
	// The page scan and the file tree agree on what is skipped.
	w, _ := fixture(t, map[string]string{
		"a.md": "x", ".git/g.md": "x", "node_modules/n.md": "x", "vendor/v.md": "x", "exports/html/e.md": "x",
		"sub/exports/e.md": "x", ".hid/h.md": "x", "sub/.git/s.md": "x",
	})
	files, err := w.Files()
	if err != nil {
		t.Fatal(err)
	}
	es, _ := tree(t, w)
	var got []string
	var walk func([]FileEntry)
	walk = func(l []FileEntry) {
		for _, e := range l {
			if e.Kind == "page" {
				got = append(got, e.Path)
			}
			walk(e.Children)
		}
	}
	walk(es)
	if !reflect.DeepEqual(got, files) {
		t.Errorf("tree pages %v, scan %v", got, files)
	}
}

func TestFileTreeSkips(t *testing.T) {
	w, _ := fixture(t, map[string]string{
		"a.md": "x", ".git/g.md": "x", ".env/e.png": "x", "node_modules/n.md": "x", "vendor/v.png": "x", "exports/html/e.md": "x",
		"docs/exports/ok.md": "x",
	})
	es, _ := tree(t, w)
	want := []string{"folder docs/", "folder docs/exports/", "page docs/exports/ok.md", "page a.md"}
	if got := flat(es); !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestFileTreeSymlinkOutside(t *testing.T) {
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "o.png"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(outside, "o.md"), []byte("# o"), 0o644); err != nil {
		t.Fatal(err)
	}
	w, root := fixture(t, map[string]string{"a.md": "x", "img/in.png": "x"})
	for name, target := range map[string]string{"out.png": filepath.Join(outside, "o.png"), "out.md": filepath.Join(outside, "o.md"), "dir": outside} {
		if err := os.Symlink(target, filepath.Join(root, name)); err != nil {
			t.Skip("no symlinks:", err)
		}
	}
	if err := os.Symlink(filepath.Join(root, "img", "in.png"), filepath.Join(root, "inside.png")); err != nil {
		t.Fatal(err)
	}
	es, _ := tree(t, w)
	want := []string{"folder img/", "image img/in.png", "page a.md", "image inside.png"}
	if got := flat(es); !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestFileTreeEntryLimit(t *testing.T) {
	withLimits(t, 100, 5)
	dir := t.TempDir()
	// Open itself would refuse; lift the limit for Open and lower it again.
	fill(t, dir, 3, ".md")
	w, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	fill(t, dir, 10, ".png")
	_, _, err = w.FileTree(context.Background())
	if !errors.Is(err, ErrTooManyFiles) {
		t.Fatalf("got %v", err)
	}
}

func TestFileTreeCancel(t *testing.T) {
	w, _ := fixture(t, map[string]string{"a.md": "x", "b/c.md": "x"})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, _, err := w.FileTree(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v", err)
	}
}

func TestFileTreeKinds(t *testing.T) {
	w, _ := fixture(t, map[string]string{
		"a.md": "x", "b.markdown": "x", "C.MD": "x", "d.txt": "x", "e.pdf": "x", "f.go": "x",
		"i.png": "x", "j.jpg": "x", "k.jpeg": "x", "l.gif": "x", "m.webp": "x", "n.svg": "x", "o.avif": "x", "p.bmp": "x",
	})
	es, _ := tree(t, w)
	want := []string{"page a.md", "page b.markdown", "page C.MD", "image i.png", "image j.jpg", "image k.jpeg", "image l.gif", "image m.webp", "image n.svg", "image o.avif"}
	got := flat(es)
	if len(got) != len(want) {
		t.Fatalf("got %v", got)
	}
	have := map[string]bool{}
	for _, g := range got {
		have[g] = true
	}
	for _, x := range want {
		if !have[x] {
			t.Errorf("missing %q in %v", x, got)
		}
	}
}

func TestFileTreeDropsEmptyFolders(t *testing.T) {
	w, root := fixture(t, map[string]string{
		"a.md": "x", "empty/readme.txt": "x", "deep/er/est/data.csv": "x", "pics/p.png": "x", "deep/er/keep.md": "x",
	})
	if err := os.MkdirAll(filepath.Join(root, "void"), 0o755); err != nil {
		t.Fatal(err)
	}
	es, _ := tree(t, w)
	want := []string{"folder deep/", "folder deep/er/", "page deep/er/keep.md", "folder pics/", "image pics/p.png", "page a.md"}
	if got := flat(es); !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestFileTreeOrder(t *testing.T) {
	w, _ := fixture(t, specsShape)
	es, _ := tree(t, w)
	want := []string{
		"folder _template/", "page _template/spec.md",
		"folder engine/", "folder engine/index/", "page engine/index/spec.md", "folder engine/session/", "page engine/session/spec.md",
		"folder plugins/", "folder plugins/2-intro/", "page plugins/2-intro/spec.md", "folder plugins/10-setup/", "page plugins/10-setup/spec.md",
		"folder ui/", "folder ui/files-view/", "folder ui/files-view/shots/", "image ui/files-view/shots/a.png", "image ui/files-view/shots/b.PNG",
		"page ui/files-view/spec.md",
		"page README.md", "page ui.md",
	}
	if got := flat(es); !reflect.DeepEqual(got, want) {
		t.Errorf("got\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
	// The folder ui/ and the page ui.md do not clash.
	paths := map[string]bool{}
	for _, l := range flat(es) {
		p := strings.SplitN(l, " ", 2)[1]
		if paths[p] {
			t.Errorf("duplicate path %q", p)
		}
		paths[p] = true
	}
}

func TestFileTreeOrderCaseAndNames(t *testing.T) {
	w, _ := fixture(t, map[string]string{"b.md": "x", "A.md": "x", "Ä café.md": "x", "my page #1.md": "x", "z/x.md": "x", "Y/x.md": "x"})
	es, _ := tree(t, w)
	want := []string{"folder Y/", "page Y/x.md", "folder z/", "page z/x.md", "page A.md", "page b.md", "page my page #1.md", "page Ä café.md"}
	if got := flat(es); !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestFileTreeImagesOnlyFolder(t *testing.T) {
	w, _ := fixture(t, map[string]string{"img/a.png": "x", "img/b.svg": "x", "README.md": "x", "index.md": "x"})
	es, _ := tree(t, w)
	want := []string{"folder img/", "image img/a.png", "image img/b.svg", "page index.md", "page README.md"}
	if got := flat(es); !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestFileTreeSameNameAsFolder(t *testing.T) {
	w, _ := fixture(t, map[string]string{"guides.md": "x", "guides/a.md": "x"})
	es, _ := tree(t, w)
	want := []string{"folder guides/", "page guides/a.md", "page guides.md"}
	if got := flat(es); !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
	if es[0].Name != "guides" {
		t.Errorf("folder name %q", es[0].Name)
	}
}

func withCap(t *testing.T, n int) {
	t.Helper()
	old := MaxFileTreeEntries
	MaxFileTreeEntries = n
	t.Cleanup(func() { MaxFileTreeEntries = old })
}

func TestFileTreeCap(t *testing.T) {
	// folder d/ (1) + pages a1..a3 (3) + images i1..i4 (4) = 8 entries; cap 5 drops 3 images (the last ones).
	w, _ := fixture(t, map[string]string{
		"d/a1.md": "x", "d/a2.md": "x", "a3.md": "x", "d/i1.png": "x", "d/i2.png": "x", "i3.png": "x", "i4.png": "x",
	})
	withCap(t, 5)
	es, omitted := tree(t, w)
	if omitted != 3 {
		t.Errorf("omitted = %d", omitted)
	}
	want := []string{"folder d/", "page d/a1.md", "page d/a2.md", "image d/i1.png", "page a3.md"}
	if got := flat(es); !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
	// Pages are cut only when no image is left; the folder that held them stays.
	withCap(t, 2)
	es, omitted = tree(t, w)
	if got := flat(es); len(got) != 2 || got[0] != "folder d/" || omitted != 6 {
		t.Errorf("got %v omitted %d", got, omitted)
	}
	// Under the cap nothing is left out.
	withCap(t, 100)
	if _, omitted = tree(t, w); omitted != 0 {
		t.Errorf("omitted = %d", omitted)
	}
}

func TestFileTree5001Files(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	withLimits(t, 6000, 50000)
	dir := t.TempDir()
	for i := 0; i < 5001; i++ {
		if err := os.WriteFile(filepath.Join(dir, fmt.Sprintf("p%04d.md", i)), nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	w, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	es, omitted, err := w.FileTree(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(es) != 5000 || omitted != 1 {
		t.Errorf("listed %d, omitted %d", len(es), omitted)
	}
}

func TestFileTreeSingle(t *testing.T) {
	dir := t.TempDir()
	for _, f := range []string{"one.md", "two.md"} {
		if err := os.WriteFile(filepath.Join(dir, f), []byte("# x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(filepath.Join(dir, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	w, err := OpenFile(filepath.Join(dir, "one.md"))
	if err != nil {
		t.Fatal(err)
	}
	es, omitted, err := w.FileTree(context.Background())
	if err != nil || omitted != 0 {
		t.Fatal(err, omitted)
	}
	if want := []FileEntry{{Name: "one.md", Path: "one.md", Kind: "page"}}; !reflect.DeepEqual(es, want) {
		t.Errorf("got %+v", es)
	}
}
