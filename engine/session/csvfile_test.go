package session

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/zenmark-org/zenmd/builtins"
)

var attrsRE = regexp.MustCompile(` data-zen-attrs="[^"]*"`)

// A csv-table :file: reads a file of the open folder through the real render
// path (Context.Workspace), and every refused path warns and reads nothing.
func TestCSVTableFileThroughTheSession(t *testing.T) {
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "secret.csv"), []byte("TOPSECRET,1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	dir := tree(t, map[string]string{
		".zenmd.yml":     "markdown:\n  dialects: [myst]\n",
		"README.md":      "# Home\n\n```{csv-table}\n:file: data/x.csv\n:header-rows: 1\n```\n",
		"data/x.csv":     "Item,Qty\nPen,3\n",
		"docs/page.md":   "# Docs\n",
		"docs/local.csv": "L,1\n",
	})
	if err := os.Symlink(filepath.Join(outside, "secret.csv"), filepath.Join(dir, "link.csv")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "big.csv"), []byte(strings.Repeat("a", 1<<20+1)), 0o644); err != nil {
		t.Fatal(err)
	}
	s := New(Options{GlobalPath: filepath.Join(t.TempDir(), "config.yml"), Registry: builtins.Plugins(), Dialects: builtins.Dialects(), Events: &sink{}, Runtime: stubAssets()})
	t.Cleanup(s.Close)
	if err := s.Open(dir); err != nil {
		t.Fatal(err)
	}

	pg, err := s.Page("README.md")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"<td>Pen</td>", "<td>3</td>", `<th scope="col">Qty</th>`} {
		if !strings.Contains(pg.HTML, want) {
			t.Errorf("missing %q in %s", want, pg.HTML)
		}
	}
	if len(pg.Warnings) != 0 {
		t.Errorf("warnings: %v", pg.Warnings)
	}
	// next to the page first
	loc, err := s.Preview("docs/page.md", "```{csv-table}\n:file: local.csv\n```\n")
	if err != nil || !strings.Contains(loc.HTML, "<td>L</td>") {
		t.Errorf("local file: %v %s", err, loc.HTML)
	}

	for name, f := range map[string]string{
		"dotdot":   "../" + filepath.Base(outside) + "/secret.csv",
		"absolute": filepath.Join(outside, "secret.csv"),
		"symlink":  "link.csv",
		"big":      "big.csv",
		"missing":  "nope.csv",
	} {
		p, err := s.Preview("README.md", "```{csv-table}\n:file: "+f+"\n```\n")
		if err != nil {
			t.Fatal(err)
		}
		shown := attrsRE.ReplaceAllString(p.HTML, "") // the wrapper echoes the page's own option
		if strings.Contains(shown, "TOPSECRET") || strings.Contains(shown, outside) {
			t.Errorf("%s: read or named the outside file: %s", name, p.HTML)
		}
		w := strings.Join(p.Warnings, "|")
		if !strings.Contains(w, `plugin "table"`) || strings.Contains(w, outside) || strings.Contains(w, dir) {
			t.Errorf("%s: want a table warning that names no real path, got %q", name, w)
		}
	}
}
