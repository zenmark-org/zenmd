package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zenmark-org/zenmd/engine/workspace"
)

// treeFolder is shaped like the repo's specs/ folder: scope folders with feature folders holding a spec.md,
// a README at the top, a page with the name of a folder, an image folder, and pages that are not in the tree.
func treeFolder(t *testing.T) string {
	t.Helper()
	home(t)
	return glossaryDir(t, map[string]string{
		"README.md":              "# Specs\n\n[Index](engine/index/spec.md) [Session](engine/session/spec.md) [Files](ui/files-view/spec.md)\n",
		"engine/index/spec.md":   "# Index\n",
		"engine/session/spec.md": "# Session\n",
		"ui/files-view/spec.md":  "# Files view\n",
		"ui/files-view/a b.png":  "png",
		"ui/img/shot.png":        "png",
		"ui.md":                  "# UI notes\n",
		"café/niño #1.md":        "# Niño\n",
		"secret.md":              "---\nnav:\n  hidden: true\n---\n# Secret\n",
		"data/rows.csv":          "a,b",
	})
}

func runTreeCmd(dir string, args ...string) (int, string, string) {
	var out, errOut bytes.Buffer
	code := runTree(args, dir, &out, &errOut)
	return code, out.String(), errOut.String()
}

func TestTreePages(t *testing.T) {
	dir := treeFolder(t)
	code, out, errOut := runTreeCmd(dir)
	if code != 0 {
		t.Fatalf("code %d, err %q", code, errOut)
	}
	for _, want := range []string{
		"Specs (README.md)\n",
		"  Index (engine/index/spec.md)\n",
		"  Session (engine/session/spec.md)\n",
		"  Files view (ui/files-view/spec.md)\n",
		"Other pages\n",
		"  UI notes (ui.md)\n",
		"Niño (café/niño #1.md)\n",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
	if strings.Contains(out, "Secret") || strings.Contains(out, "png") {
		t.Errorf("hidden pages and images are not in the Pages tree:\n%s", out)
	}
	// A folder argument.
	code, out2, _ := runTreeCmd(t.TempDir(), dir)
	if code != 0 || out2 != out {
		t.Errorf("folder argument: code %d\n%s", code, out2)
	}
}

func TestTreeFiles(t *testing.T) {
	dir := treeFolder(t)
	code, out, errOut := runTreeCmd(dir, "--files")
	if code != 0 {
		t.Fatalf("code %d, err %q", code, errOut)
	}
	want := `café/
  niño #1.md (other)
engine/
  index/
    spec.md
  session/
    spec.md
ui/
  files-view/
    a b.png
    spec.md
  img/
    shot.png
README.md
secret.md (hidden)
ui.md (other)
`
	if out != want {
		t.Errorf("got:\n%s\nwant:\n%s", out, want)
	}
	if strings.Contains(out, "more files") {
		t.Errorf("nothing was left out:\n%s", out)
	}
}

func TestTreeFilesOmitted(t *testing.T) {
	dir := treeFolder(t)
	old := workspace.MaxFileTreeEntries
	t.Cleanup(func() { workspace.MaxFileTreeEntries = old })
	workspace.MaxFileTreeEntries = 6
	code, out, _ := runTreeCmd(dir, "--files")
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if code != 0 || !strings.HasSuffix(lines[len(lines)-1], "more files not shown") || !strings.HasPrefix(lines[len(lines)-1], "… ") {
		t.Errorf("code %d:\n%s", code, out)
	}
}

func TestTreeJSON(t *testing.T) {
	dir := treeFolder(t)
	code, out, _ := runTreeCmd(dir, "--json")
	var nodes []struct {
		Title    string
		Path     string
		Kind     string
		Children []any
	}
	if code != 0 || json.Unmarshal([]byte(out), &nodes) != nil || len(nodes) < 2 || nodes[0].Path != "README.md" {
		t.Fatalf("code %d:\n%s", code, out)
	}
	code, out, _ = runTreeCmd(dir, "--files", "--json")
	var ft struct {
		Nodes   []struct{ Name, Title, Path, Kind, Nav string }
		Omitted int
	}
	if code != 0 || json.Unmarshal([]byte(out), &ft) != nil || len(ft.Nodes) == 0 || ft.Nodes[0].Path != "café/" || ft.Nodes[0].Kind != "folder" {
		t.Fatalf("code %d:\n%s", code, out)
	}
	if !strings.Contains(out, `"omitted": 0`) {
		t.Errorf("omitted missing:\n%s", out)
	}
}

func TestTreeErrors(t *testing.T) {
	dir := treeFolder(t)
	code, _, errOut := runTreeCmd(dir, filepath.Join(dir, "nope"))
	if code != 1 || !strings.HasPrefix(errOut, "zenmd: ") {
		t.Errorf("missing folder: code %d err %q", code, errOut)
	}
	file := filepath.Join(dir, "README.md")
	if code, _, errOut = runTreeCmd(dir, file); code != 1 || !strings.HasPrefix(errOut, "zenmd: ") {
		t.Errorf("a file: code %d err %q", code, errOut)
	}
	if code, _, errOut = runTreeCmd(dir, "--nope"); code != 2 || !strings.Contains(errOut, "--nope") {
		t.Errorf("unknown flag: code %d err %q", code, errOut)
	}
	if code, _, _ = runTreeCmd(dir, "a", "b"); code != 2 {
		t.Errorf("two folders: code %d", code)
	}
}

// There is no config key and no export option: tree takes --files, --json and -h, and nothing else.
func TestTreeFlags(t *testing.T) {
	dir := treeFolder(t)
	for _, f := range []string{"-o", "--out", "--theme", "--format", "--config"} {
		if code, _, _ := runTreeCmd(dir, f, "x"); code != 2 {
			t.Errorf("%s: code %d, want 2", f, code)
		}
	}
	if code, out, _ := runTreeCmd(dir, "-h"); code != 0 || !strings.Contains(out, "tree") {
		t.Errorf("-h: code %d", code)
	}
	if _, err := os.Stat(filepath.Join(dir, ".zenmd.yml")); err == nil {
		t.Error("tree wrote a config file")
	}
}
