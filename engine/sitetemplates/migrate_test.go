package sitetemplates

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// cfgWith makes a config folder with templates/sites holding a blueprint and its record.
func cfgWith(t *testing.T) (cfg, dir string) {
	t.Helper()
	cfg = filepath.Join(t.TempDir(), "zen")
	dir = filepath.Join(cfg, "templates")
	put(t, dir, map[string]string{
		"sites/techdocs/blueprint.yml": "id: techdocs\nname: T\n",
		"sites/mine/blueprint.yml":     "id: mine\nname: Mine\n",
		"sites/.zen-templates.yml":     "installed:\n    techdocs: 2.0.0\n",
	})
	return cfg, dir
}

func TestMigrateSitesMoves(t *testing.T) {
	cfg, dir := cfgWith(t)
	if n := MigrateSites(dir, cfg); n != "templates/sites is now templates/blueprints" {
		t.Fatalf("notice %q", n)
	}
	if got := walk(t, dir); strings.Join(got, ",") != "blueprints/.zen-templates.yml,blueprints/mine/blueprint.yml,blueprints/techdocs/blueprint.yml" {
		t.Errorf("files: %v", got)
	}
	if _, err := os.Stat(filepath.Join(dir, "sites")); err == nil {
		t.Error("sites is still there")
	}
	// Once: the second call finds nothing to move, and says nothing.
	if n := MigrateSites(dir, cfg); n != "" {
		t.Errorf("second notice %q", n)
	}
	// The record travelled: listing sees techdocs as installed.
	if l, _ := List(dir); len(l) != 2 || !l[1].Builtin && !l[0].Builtin {
		t.Errorf("list: %+v", l)
	}
}

func TestMigrateSitesNothingToDo(t *testing.T) {
	cfg := t.TempDir()
	if n := MigrateSites(filepath.Join(cfg, "templates"), cfg); n != "" {
		t.Errorf("no folder: %q", n)
	}
	// A file called sites is not a folder.
	dir := filepath.Join(cfg, "templates")
	put(t, dir, map[string]string{"sites": "x"})
	if n := MigrateSites(dir, cfg); n != "" {
		t.Errorf("file: %q", n)
	}
}

func TestMigrateSitesKeepsExistingBlueprints(t *testing.T) {
	cfg, dir := cfgWith(t)
	put(t, dir, map[string]string{"blueprints/own/blueprint.yml": "id: own\nname: Own\n"})
	n := MigrateSites(dir, cfg)
	if !strings.Contains(n, "templates/sites is ignored") || !strings.Contains(n, "templates/blueprints") {
		t.Errorf("notice %q", n)
	}
	if got := walk(t, dir); !strings.Contains(strings.Join(got, ","), "sites/techdocs/blueprint.yml") || !strings.Contains(strings.Join(got, ","), "blueprints/own/blueprint.yml") || strings.Contains(strings.Join(got, ","), "blueprints/techdocs") {
		t.Errorf("files: %v", got)
	}
}

func TestMigrateSitesNeverFollowsLinks(t *testing.T) {
	// sites is a link
	cfg := filepath.Join(t.TempDir(), "zen")
	dir := filepath.Join(cfg, "templates")
	elsewhere := t.TempDir()
	put(t, elsewhere, map[string]string{"techdocs/blueprint.yml": "id: techdocs\nname: T\n"})
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(elsewhere, filepath.Join(dir, "sites")); err != nil {
		t.Fatal(err)
	}
	if n := MigrateSites(dir, cfg); !strings.Contains(n, "link") {
		t.Errorf("link notice %q", n)
	}
	if fi, err := os.Lstat(filepath.Join(dir, "sites")); err != nil || fi.Mode()&os.ModeSymlink == 0 {
		t.Error("the link was moved")
	}
	if _, err := os.Lstat(filepath.Join(dir, "blueprints")); err == nil {
		t.Error("blueprints appeared")
	}
	// blueprints is a link (so it exists): sites is ignored
	cfg2, dir2 := cfgWith(t)
	if err := os.Symlink(elsewhere, filepath.Join(dir2, "blueprints")); err != nil {
		t.Fatal(err)
	}
	if n := MigrateSites(dir2, cfg2); n == "" {
		t.Error("want a notice")
	}
	if _, err := os.Stat(filepath.Join(dir2, "sites", "techdocs")); err != nil {
		t.Error("sites moved into a link")
	}
	// the templates folder is a link out of the config folder
	cfg3 := filepath.Join(t.TempDir(), "zen")
	if err := os.MkdirAll(cfg3, 0o755); err != nil {
		t.Fatal(err)
	}
	out := t.TempDir()
	put(t, out, map[string]string{"sites/techdocs/blueprint.yml": "x"})
	if err := os.Symlink(out, filepath.Join(cfg3, "templates")); err != nil {
		t.Fatal(err)
	}
	if n := MigrateSites(filepath.Join(cfg3, "templates"), cfg3); !strings.Contains(n, "outside") {
		t.Errorf("outside notice %q", n)
	}
	if _, err := os.Stat(filepath.Join(out, "sites", "techdocs", "blueprint.yml")); err != nil {
		t.Error("moved out of the config folder")
	}
}

func TestMigrateSitesOutsideConfigFolder(t *testing.T) {
	cfg := filepath.Join(t.TempDir(), "zen")
	if err := os.MkdirAll(cfg, 0o755); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(t.TempDir(), "Documents", "templates") // templates: { dir: ... } elsewhere
	put(t, dir, map[string]string{"sites/techdocs/blueprint.yml": "x"})
	if n := MigrateSites(dir, cfg); !strings.Contains(n, "outside") {
		t.Errorf("notice %q", n)
	}
	if _, err := os.Stat(filepath.Join(dir, "sites", "techdocs")); err != nil {
		t.Error("moved")
	}
}
