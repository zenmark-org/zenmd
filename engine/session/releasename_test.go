package session

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNewPagePathDigitDots(t *testing.T) {
	for in, want := range map[string]string{
		"releases/0.2.0":      "releases/0.2.0.md",
		"releases/v0.2.0":     "releases/v0.2.0.md",
		"2026.11":             "2026.11.md",
		"x.md":                "x.md",
		"notes/Plan.markdown": "notes/Plan.markdown",
		"ideas":               "ideas.md",
	} {
		got, err := newPagePath(in)
		if err != nil || got != want {
			t.Errorf("newPagePath(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	for _, bad := range []string{"notes.txt", "v1.2.x", "a.v2"} {
		if got, err := newPagePath(bad); err == nil {
			t.Errorf("newPagePath(%q) = %q, want an error", bad, got)
		}
	}
}

const releaseRules = "version: 1\ntemplates:\n  \"releases/**\": release-notes\n"

func releaseZen(t *testing.T, files map[string]string) (*Session, string) {
	t.Helper()
	files[".zenmd.yml"] = releaseRules
	dir := tree(t, files)
	z, _, _, _ := newTestZen(t, dir, "", "")
	return z, dir
}

func TestSuggestPageName(t *testing.T) {
	z, _ := releaseZen(t, map[string]string{"README.md": "# Home\n"})
	if got := z.SuggestPageName("releases/"); got != "releases/v0.1.0" {
		t.Errorf("no release yet: %q", got)
	}
	if got := z.SuggestPageName(""); got != "" {
		t.Errorf("a folder without the rule: %q", got)
	}
	z, _ = releaseZen(t, map[string]string{
		"releases/v0.1.0.md": "---\nlayout: release\nversion: \"0.1.0\"\n---\n",
		"releases/v0.3.0.md": "---\nlayout: release\nversion: \"0.3.0\"\n---\n",
		"releases/index.md":  "# Releases\n",
	})
	if got := z.SuggestPageName("releases/"); got != "releases/v0.4.0" {
		t.Errorf("after v0.3.0: %q", got)
	}
	z, _ = releaseZen(t, map[string]string{"releases/v0.9.0.md": "x", "releases/v0.10.0.md": "x"})
	if got := z.SuggestPageName("releases/"); got != "releases/v0.11.0" {
		t.Errorf("semver compare, not text: %q", got)
	}
}

func TestCreateReleasePage(t *testing.T) {
	z, dir := releaseZen(t, map[string]string{"releases/v0.3.0.md": "x"})
	rel, err := z.CreatePage("releases/v0.4.0", "")
	if err != nil || rel != "releases/v0.4.0.md" {
		t.Fatalf("CreatePage = %q, %v", rel, err)
	}
	b, _ := os.ReadFile(filepath.Join(dir, filepath.FromSlash(rel)))
	got := string(b)
	today := time.Now().Format("2006-01-02")
	for _, w := range []string{"layout: release\n", `version: "0.4.0"` + "\n", "date: " + today + "\n", "## Added"} {
		if !strings.Contains(got, w) {
			t.Errorf("lacks %q:\n%s", w, got)
		}
	}
	if strings.Contains(got, "title:") || strings.Contains(got, "\n# ") {
		t.Errorf("a release has no title: and no H1:\n%s", got)
	}
	if pg, err := z.Page(rel); err != nil || pg.Title != "v0.4.0" {
		t.Errorf("title %q, %v", pg.Title, err)
	}
	// A name that is not v<semver> still makes a release page; the version is then left empty.
	if _, err := z.CreatePage("releases/next", ""); err != nil {
		t.Errorf("releases/next: %v", err)
	}
	// Outside the rule's folder a page stays plain.
	rel, _ = z.CreatePage("notes/a", "")
	if b, _ := os.ReadFile(filepath.Join(dir, rel)); !strings.HasPrefix(string(b), "# A") {
		t.Errorf("plain page:\n%s", b)
	}
}
