package render

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zenmark-org/zenmd/engine/config"
	"github.com/zenmark-org/zenmd/engine/plugins"
)

// folder makes an open folder with files and returns its root.
func folder(t *testing.T, files map[string]string) string {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for name, body := range files {
		p := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

// ctxOf renders a page that holds a block of plugin "f" and returns the context
// that plugin was given.
func ctxOf(t *testing.T, root, page string) (plugins.Context, plugins.Block) {
	t.Helper()
	var seen plugins.Context
	var blk plugins.Block
	m := man("f", true)
	m.Blocks = []string{"f"}
	p := &fakePlugin{m: m, block: func(c plugins.Context, b plugins.Block) (string, error) {
		seen, blk = c, b
		return "<p>ok</p>", nil
	}}
	var res Resolver = fakeResolver{}
	if root != "" {
		res = rootResolver{fakeResolver{}, root}
	}
	r := New(newReg(t, p), config.Default(), res)
	if _, err := r.Render(page, []byte(":::f\nx\n:::\n")); err != nil {
		t.Fatal(err)
	}
	return seen, blk
}

func readOf(t *testing.T, root, page string) func(string) ([]byte, error) {
	t.Helper()
	c, _ := ctxOf(t, root, page)
	if c.ReadFile == nil {
		t.Fatal("ReadFile is nil")
	}
	return c.ReadFile
}

func TestReadFileNilWithoutFolder(t *testing.T) {
	if c, _ := ctxOf(t, "", "a.md"); c.ReadFile != nil {
		t.Error("ReadFile set without a folder")
	}
}

func TestReadFileFromRootAndPage(t *testing.T) {
	root := folder(t, map[string]string{
		"top.mmd":           "top",
		"docs/a.mmd":        "docs a",
		"docs/deep/b.mmd":   "deep b",
		"my flows/é.MMD":    "unicode",
		"docs/crlf.mmd":     "x\r\ny\r\n",
		"docs/deep/up.mmd":  "up",
		"docs/deep/more/.k": "",
	})
	rf := readOf(t, root, "docs/deep/page.md")
	for p, want := range map[string]string{
		"b.mmd":                "deep b",
		"/top.mmd":             "top",
		"/docs/a.mmd":          "docs a",
		"../a.mmd":             "docs a",
		"../../top.mmd":        "top",
		"/my flows/é.MMD":      "unicode",
		"../../my flows/é.MMD": "unicode",
		"../crlf.mmd":          "x\r\ny\r\n",
	} {
		got, err := rf(p)
		if err != nil || string(got) != want {
			t.Errorf("ReadFile(%q) = %q, %v; want %q", p, got, err, want)
		}
	}
	// a page at the root
	rf = readOf(t, root, "page.md")
	if got, err := rf("top.mmd"); err != nil || string(got) != "top" {
		t.Errorf("root page: %q %v", got, err)
	}
	if _, err := rf("../top.mmd"); err == nil {
		t.Error("root page climbed out")
	}
}

func TestReadFileMissingAndOutside(t *testing.T) {
	root := folder(t, map[string]string{"a.mmd": "a", "docs/x.md": ""})
	rf := readOf(t, root, "docs/page.md")
	_, err := rf("nope.mmd")
	if err == nil || err.Error() != "nope.mmd was not found in the folder" {
		t.Errorf("missing: %v", err)
	}
	for _, p := range []string{"../../a.mmd", "/../a.mmd", "/etc/passwd/../../../x"} {
		_, err := rf(p)
		if err == nil || err.Error() != p+" is outside the folder" {
			t.Errorf("%q: %v", p, err)
		}
	}
	for _, p := range []string{"a\\b", "a\x00b", "a\nb", strings.Repeat("a", 1025)} {
		if _, err := rf(p); err == nil {
			t.Errorf("%q accepted", p)
		}
	}
}

func TestReadFileOutsideByLink(t *testing.T) {
	outside := folder(t, map[string]string{"secret.txt": "secret", "dir/s.txt": "s"})
	root := folder(t, map[string]string{"ok.mmd": "ok"})
	for from, to := range map[string]string{
		"link.mmd": filepath.Join(outside, "secret.txt"),
		"dirlink":  filepath.Join(outside, "dir"),
	} {
		if err := os.Symlink(to, filepath.Join(root, from)); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Symlink("ok.mmd", filepath.Join(root, "inner.mmd")); err != nil {
		t.Fatal(err)
	}
	rf := readOf(t, root, "page.md")
	for _, p := range []string{"link.mmd", "/link.mmd", "dirlink/s.txt"} {
		_, err := rf(p)
		if err == nil || err.Error() != p+" is outside the folder" {
			t.Errorf("%q: %v", p, err)
		}
	}
	if got, err := rf("inner.mmd"); err != nil || string(got) != "ok" {
		t.Errorf("a link inside the folder: %q %v", got, err)
	}
}

func TestReadFileDanglingLink(t *testing.T) {
	root := folder(t, nil)
	if err := os.Symlink(filepath.Join(root, "..", "gone-"+filepath.Base(root)), filepath.Join(root, "d.mmd")); err != nil {
		t.Fatal(err)
	}
	rf := readOf(t, root, "page.md")
	_, err := rf("d.mmd")
	if err == nil || err.Error() != "d.mmd is outside the folder" {
		t.Errorf("dangling: %v", err)
	}
}

func TestReadFileNotRegular(t *testing.T) {
	root := folder(t, map[string]string{"dir/x.mmd": "x"})
	rf := readOf(t, root, "page.md")
	for _, p := range []string{"dir", "/", "/dir"} {
		_, err := rf(p)
		if err == nil || err.Error() != p+" is not a file" {
			t.Errorf("%q: %v", p, err)
		}
	}
}

func TestReadFileTooLarge(t *testing.T) {
	root := folder(t, map[string]string{
		"big.mmd":  strings.Repeat("a", plugins.MaxReadFile+1),
		"edge.mmd": strings.Repeat("a", plugins.MaxReadFile),
	})
	rf := readOf(t, root, "page.md")
	_, err := rf("big.mmd")
	if err == nil || err.Error() != "big.mmd is larger than 1 MB" {
		t.Errorf("big: %v", err)
	}
	if got, err := rf("edge.mmd"); err != nil || len(got) != plugins.MaxReadFile {
		t.Errorf("edge: %d %v", len(got), err)
	}
}

func TestReadFileBinary(t *testing.T) {
	root := folder(t, map[string]string{"b.mmd": "ab\x00cd"})
	rf := readOf(t, root, "page.md")
	_, err := rf("b.mmd")
	if err == nil || err.Error() != "b.mmd is not a text file" {
		t.Errorf("binary: %v", err)
	}
}

func TestReadFileMessagesNoAbsolutePath(t *testing.T) {
	outside := folder(t, map[string]string{"s.txt": "s"})
	root := folder(t, map[string]string{"dir/x": "x", "b": "\x00", "big": strings.Repeat("a", plugins.MaxReadFile+1)})
	_ = os.Symlink(filepath.Join(outside, "s.txt"), filepath.Join(root, "l"))
	rf := readOf(t, root, "docs/page.md")
	for _, p := range []string{"nope", "../../x", "/l", "/dir", "/b", "/big", "/l/../zz"} {
		_, err := rf(p)
		if err == nil {
			t.Errorf("%q: no error", p)
			continue
		}
		for _, bad := range []string{root, outside, os.TempDir(), "/private", "/var/"} {
			if strings.Contains(err.Error(), bad) {
				t.Errorf("%q: message %q names %q", p, err, bad)
			}
		}
		if !strings.HasPrefix(err.Error(), p) {
			t.Errorf("%q: message %q does not start with the path as written", p, err)
		}
	}
}

func TestFenceArgument(t *testing.T) {
	for src, want := range map[string]string{
		"```f diagrams/flow.mmd\nx\n```\n":         "diagrams/flow.mmd",
		"```f my flows/a b.mmd\nx\n```\n":          "my flows/a b.mmd",
		"```f   spaced.mmd  \nx\n```\n":            "spaced.mmd",
		"```f\nx\n```\n":                           "",
		"```f {file=a.mmd}\nx\n```\n":              "",
		"```f {file=\"my flows/a.mmd\"}\nx\n```\n": "",
	} {
		_, b := ctxOf2(t, src)
		if b.Attrs["arg"] != want {
			t.Errorf("%q: arg %q, want %q", src, b.Attrs["arg"], want)
		}
	}
}

func TestFenceBracesUnchanged(t *testing.T) {
	_, b := ctxOf2(t, "```f {file=\"my flows/a.mmd\" x=1}\nbody\n```\n")
	if b.Attrs["file"] != "my flows/a.mmd" || b.Attrs["x"] != "1" || b.Attrs["arg"] != "" || b.Body != "body\n" {
		t.Errorf("%+v", b)
	}
}

// ctxOf2 renders src (a fence of plugin "f") and returns the block it was given.
func ctxOf2(t *testing.T, src string) (plugins.Context, plugins.Block) {
	t.Helper()
	var seen plugins.Context
	var blk plugins.Block
	m := man("f", true)
	m.Blocks = []string{"f"}
	p := &fakePlugin{m: m, block: func(c plugins.Context, b plugins.Block) (string, error) {
		seen, blk = c, b
		return "<p>ok</p>", nil
	}}
	if _, err := New(newReg(t, p), config.Default(), fakeResolver{}).Render("a.md", []byte(src)); err != nil {
		t.Fatal(err)
	}
	return seen, blk
}
