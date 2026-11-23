package workspace

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func write(t *testing.T, p, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// One entry: Files, Opened, Entries and Home all answer from the one file.
func TestOpenFileOneEntry(t *testing.T) {
	dir := t.TempDir()
	write(t, filepath.Join(dir, "notes.md"), "# Notes\n")
	write(t, filepath.Join(dir, "other.md"), "# Other\n")
	write(t, filepath.Join(dir, "sub", "deep.md"), "# Deep\n")

	w, err := OpenFile(filepath.Join(dir, "notes.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !w.Single() {
		t.Error("Single() = false")
	}
	if got := w.Root(); got != mustReal(t, dir) {
		t.Errorf("Root() = %q, want the file's folder", got)
	}
	if w.Name() != "notes.md" {
		t.Errorf("Name() = %q, want the file's name", w.Name())
	}
	if got := w.Opened(); len(got) != 1 || got[0] != "notes.md" {
		t.Errorf("Opened() = %v", got)
	}
	files, err := w.Files()
	if err != nil || len(files) != 1 || files[0] != "notes.md" {
		t.Errorf("Files() = %v, %v", files, err)
	}
	if w.Home() != "notes.md" {
		t.Errorf("Home() = %q", w.Home())
	}
	if w.Entries() != 1 {
		t.Errorf("Entries() = %d, want 1", w.Entries())
	}
	// A folder workspace is not single.
	fw, _ := Open(dir)
	if fw.Single() {
		t.Error("a folder workspace is single")
	}
	// The sibling page still reads: links to it open in the same window.
	if b, err := w.Read("other.md"); err != nil || string(b) != "# Other\n" {
		t.Errorf("Read sibling: %q %v", b, err)
	}
	if l := w.ResolveLink("notes.md", "other.md"); l.Kind != kindInternal || l.Path != "other.md" {
		t.Errorf("link to a sibling: %+v", l)
	}
	if l := w.ResolveLink("notes.md", "../x.md"); l.Kind != kindBroken {
		t.Errorf("link above the folder: %+v", l)
	}
}

func TestOpenFileErrors(t *testing.T) {
	dir := t.TempDir()
	write(t, filepath.Join(dir, "a.txt"), "x")
	if _, err := OpenFile(filepath.Join(dir, "a.txt")); !errors.Is(err, ErrNotMarkdown) {
		t.Errorf("a .txt file: %v", err)
	}
	if _, err := OpenFile(dir); err == nil {
		t.Error("a folder must not open as a file")
	}
	if _, err := OpenFile(filepath.Join(dir, "missing.md")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("missing: %v", err)
	}
	big := filepath.Join(dir, "big.md")
	f, err := os.Create(big)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.Truncate(maxFileSize + 1); err != nil {
		t.Fatal(err)
	}
	f.Close()
	if _, err := OpenFile(big); !errors.Is(err, ErrTooLarge) {
		t.Errorf("over the limit: %v", err)
	}
}

// A symlinked file: the root is the target's folder.
func TestOpenFileSymlink(t *testing.T) {
	real, links := t.TempDir(), t.TempDir()
	write(t, filepath.Join(real, "target.md"), "# T\n")
	if err := os.Symlink(filepath.Join(real, "target.md"), filepath.Join(links, "link.md")); err != nil {
		t.Skip("no symlinks:", err)
	}
	w, err := OpenFile(filepath.Join(links, "link.md"))
	if err != nil {
		t.Fatal(err)
	}
	if w.Root() != mustReal(t, real) || w.Home() != "target.md" {
		t.Errorf("root %q home %q", w.Root(), w.Home())
	}
}

// The folder is never walked: a sibling folder of 5000 files costs nothing.
func TestOpenFileNoWalk(t *testing.T) {
	dir := t.TempDir()
	write(t, filepath.Join(dir, "notes.md"), "# Notes\n")
	for i := 0; i < 5000; i++ {
		write(t, filepath.Join(dir, "many", fmt.Sprintf("f%d.md", i)), "x")
	}
	start := time.Now()
	w, err := OpenFile(filepath.Join(dir, "notes.md"))
	if err != nil {
		t.Fatal(err)
	}
	if took := time.Since(start); took > 50*time.Millisecond {
		t.Errorf("opening took %v, want under 50ms", took)
	}
	if w.Entries() != 1 {
		t.Errorf("Entries() = %d, want 1", w.Entries())
	}
	// The limits of a folder do not apply.
	f, _ := OpenWithLimits(dir, 0, 0)
	_ = f
	old := MaxFiles
	MaxFiles = 1
	defer func() { MaxFiles = old }()
	if _, err := OpenFile(filepath.Join(dir, "notes.md")); err != nil {
		t.Errorf("the folder limit applied to one file: %v", err)
	}
}

// Assets are read only inside the file's folder.
func TestOpenFileAssetContainment(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "docs")
	write(t, filepath.Join(dir, "notes.md"), "![a](a.png)\n")
	write(t, filepath.Join(dir, "a.png"), "\x89PNG")
	write(t, filepath.Join(dir, "img", "b.png"), "\x89PNG")
	write(t, filepath.Join(root, "secret.png"), "\x89PNG")
	if err := os.Symlink(filepath.Join(root, "secret.png"), filepath.Join(dir, "evil.png")); err != nil {
		t.Skip("no symlinks:", err)
	}

	w, err := OpenFile(filepath.Join(dir, "notes.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, ok := range []string{"a.png", "img/b.png"} {
		f, err := w.Asset(ok)
		if err != nil {
			t.Errorf("Asset(%q): %v", ok, err)
			continue
		}
		f.Close()
	}
	for _, bad := range []string{"../secret.png", "evil.png", "../docs/../secret.png"} {
		if f, err := w.Asset(bad); err == nil {
			f.Close()
			t.Errorf("Asset(%q) escaped the file's folder", bad)
		} else if !errors.Is(err, ErrOutside) {
			t.Errorf("Asset(%q): %v, want ErrOutside", bad, err)
		}
	}
	if _, err := w.Read("../secret.png"); err == nil || !strings.Contains(err.Error(), "outside") {
		t.Errorf("Read above the folder: %v", err)
	}
}

func mustReal(t *testing.T, p string) string {
	t.Helper()
	r, err := filepath.EvalSymlinks(p)
	if err != nil {
		t.Fatal(err)
	}
	return r
}
