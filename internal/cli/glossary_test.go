package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func glossaryDir(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for rel, body := range files {
		p := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

const gIndex = "---\ntitle: Glossary\nlayout: glossary\n---\nWords.\n"

func TestGlossaryAddFindsTheOnlyGlossary(t *testing.T) {
	dir := glossaryDir(t, map[string]string{"README.md": "# Home\n", "glossary/index.md": gIndex})
	r := run(t, dir, nil, "glossary", "add", "Front matter")
	if r.code != 0 || !strings.Contains(r.stdout, "glossary/front-matter.md") {
		t.Fatalf("code %d out %q err %q", r.code, r.stdout, r.stderr)
	}
	b, err := os.ReadFile(filepath.Join(dir, "glossary", "front-matter.md"))
	if err != nil || !strings.HasPrefix(string(b), "---\nlayout: term\n---\n# Front matter\n") {
		t.Errorf("%v\n%s", err, b)
	}
	if r.launched {
		t.Error("the app opened without --open")
	}
	// Again: it exists, nothing is written over it, and the exit code says so.
	r = run(t, dir, nil, "glossary", "add", "front matter")
	if r.code != 1 || !strings.Contains(r.stderr, "already exists") || !strings.Contains(r.stderr, "glossary/front-matter.md") {
		t.Errorf("code %d err %q", r.code, r.stderr)
	}
}

func TestGlossaryAddDefinitionAndOpen(t *testing.T) {
	dir := glossaryDir(t, map[string]string{"glossary/index.md": gIndex})
	r := run(t, dir, nil, "glossary", "add", "Slug", "--definition", "A short name.", "--open")
	if r.code != 0 || !r.launched || r.l.Dir != dir || r.l.Page != "glossary/slug.md" || r.l.Mode != "edit" {
		t.Fatalf("code %d launched %v %+v err %q", r.code, r.launched, r.l, r.stderr)
	}
	b, _ := os.ReadFile(filepath.Join(dir, "glossary", "slug.md"))
	if !strings.Contains(string(b), "# Slug\n\nA short name.\n") {
		t.Errorf("%s", b)
	}
	// --definition=… works too.
	if r := run(t, dir, nil, "glossary", "add", "Layout", "--definition=How a page looks."); r.code != 0 {
		t.Fatalf("%d %q", r.code, r.stderr)
	}
}

func TestGlossaryAddNeedsAFolderWhenThereAreSeveral(t *testing.T) {
	dir := glossaryDir(t, map[string]string{"a/index.md": gIndex, "docs/b/index.md": gIndex, "docs/b/x.md": "---\nlayout: term\n---\n# X\n\nx.\n"})
	r := run(t, dir, nil, "glossary", "add", "One")
	if r.code != 1 || !strings.Contains(r.stderr, "--folder") || !strings.Contains(r.stderr, "a") || !strings.Contains(r.stderr, "docs/b") {
		t.Fatalf("code %d err %q", r.code, r.stderr)
	}
	r = run(t, dir, nil, "glossary", "add", "One", "--folder", "docs/b")
	if r.code != 0 {
		t.Fatalf("%d %q", r.code, r.stderr)
	}
	if _, err := os.Stat(filepath.Join(dir, "docs", "b", "one.md")); err != nil {
		t.Error(err)
	}
	// Run from inside one of them, the folder is that one.
	if r := run(t, filepath.Join(dir, "a"), nil, "glossary", "add", "Two"); r.code != 0 {
		t.Fatalf("%d %q", r.code, r.stderr)
	}
	if _, err := os.Stat(filepath.Join(dir, "a", "two.md")); err != nil {
		t.Error(err)
	}
}

func TestGlossaryAddNoGlossary(t *testing.T) {
	dir := glossaryDir(t, map[string]string{"README.md": "# Home\n"})
	r := run(t, dir, nil, "glossary", "add", "Term")
	if r.code != 1 || !strings.Contains(r.stderr, "no glossary") {
		t.Errorf("code %d err %q", r.code, r.stderr)
	}
}

func TestGlossaryList(t *testing.T) {
	dir := glossaryDir(t, map[string]string{
		"glossary/index.md": gIndex,
		"glossary/z.md":     "---\nlayout: term\n---\n# Zebra\n\nA striped animal.\n",
		"glossary/a.md":     "---\ntitle: Apple\nlayout: term\n---\nA fruit.\n",
		"glossary/n.md":     "# Notes\n\nNot a term.\n",
	})
	r := run(t, dir, nil, "glossary", "list")
	if r.code != 0 {
		t.Fatalf("%d %q", r.code, r.stderr)
	}
	lines := strings.Split(strings.TrimSpace(r.stdout), "\n")
	if len(lines) != 2 || !strings.HasPrefix(lines[0], "Apple") || !strings.Contains(lines[0], "A fruit.") || !strings.HasPrefix(lines[1], "Zebra") || !strings.Contains(lines[1], "A striped animal.") {
		t.Errorf("%q", r.stdout)
	}
	if r2 := run(t, dir, nil, "glossary", "list", "glossary"); r2.stdout != r.stdout {
		t.Errorf("with the folder: %q", r2.stdout)
	}
}

func TestGlossaryUsage(t *testing.T) {
	dir := glossaryDir(t, map[string]string{"glossary/index.md": gIndex})
	for _, args := range [][]string{
		{"glossary"}, {"glossary", "add"}, {"glossary", "add", "A", "B"}, {"glossary", "add", "A", "--nope"},
		{"glossary", "add", "A", "--folder"}, {"glossary", "list", "a", "b"}, {"glossary", "what"}, {"glossary", "list", "--open"},
	} {
		if r := run(t, dir, nil, args...); r.code != 2 {
			t.Errorf("%v: code %d err %q", args, r.code, r.stderr)
		}
	}
	if r := run(t, dir, nil, "glossary", "add", "???"); r.code == 0 {
		t.Error("a term with no letters or digits")
	}
}

func TestGlossaryHelp(t *testing.T) {
	for _, want := range []string{"zenmd glossary add \"<Term>\" [--folder <folder>] [--definition <text>] [--open]", "zenmd glossary list [folder]"} {
		if !strings.Contains(usage, want) {
			t.Errorf("usage lacks %q", want)
		}
	}
}
