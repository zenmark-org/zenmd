package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func templatesOf(t *testing.T, global, folder string) (string, []string, []string) {
	t.Helper()
	dir := t.TempDir()
	cfgDir := filepath.Join(dir, "cfg")
	g, root := "", ""
	if global != "" {
		g = write(t, filepath.Join(cfgDir, "config.yml"), global)
	}
	if folder != "" {
		root = filepath.Join(dir, "ws")
		write(t, filepath.Join(root, WorkspaceFile), folder)
	}
	c := Load(g, root)
	d, w := c.BlueprintsDir(cfgDir)
	return filepath.ToSlash(strings.TrimPrefix(d, dir)), w, c.Warnings()
}

func TestBlueprintsDir(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	// Default: the folder next to config.yml.
	if d, w, lw := templatesOf(t, "", ""); d != "/cfg/blueprints" || len(w) != 0 || len(lw) != 0 {
		t.Errorf("default: %q %v %v", d, w, lw)
	}
	if d, _, _ := templatesOf(t, "theme: dark\n", ""); d != "/cfg/blueprints" {
		t.Errorf("no key: %q", d)
	}
	// ~ expands to the home folder; an absolute path is kept as it is.
	c := Load("", "")
	_ = c
	dir := t.TempDir()
	g := write(t, filepath.Join(dir, "config.yml"), "blueprints:\n  dir: ~/Documents/zen templates\n")
	got, warns := Load(g, "").BlueprintsDir(dir)
	if got != filepath.Join(home, "Documents", "zen templates") || len(warns) != 0 {
		t.Errorf("tilde: %q %v", got, warns)
	}
	g = write(t, filepath.Join(dir, "config.yml"), "blueprints:\n  dir: /srv/zen/t\n")
	if got, _ = Load(g, "").BlueprintsDir(dir); got != "/srv/zen/t" {
		t.Errorf("absolute: %q", got)
	}
	// A relative path is relative to the config folder, and is cleaned.
	g = write(t, filepath.Join(dir, "config.yml"), "blueprints:\n  dir: ../shared/./site templates\n")
	if got, _ = Load(g, "").BlueprintsDir(dir); got != filepath.Join(filepath.Dir(dir), "shared", "site templates") {
		t.Errorf("relative: %q", got)
	}
	// "~user" is not expanded (a relative name), "~" alone is home.
	g = write(t, filepath.Join(dir, "config.yml"), "blueprints:\n  dir: \"~\"\n")
	if got, _ = Load(g, "").BlueprintsDir(dir); got != home {
		t.Errorf("home alone: %q", got)
	}

	// Wrong shapes: the default is used, and TemplatesDir says why.
	for _, src := range []string{"blueprints: yes\n", "blueprints:\n  dir: 3\n", "blueprints:\n  dir: [a]\n", "blueprints:\n  folder: x\n"} {
		d, w, lw := templatesOf(t, src, "")
		if d != "/cfg/blueprints" || len(w) != 1 || !strings.Contains(w[0], "blueprints") || len(lw) != 0 {
			t.Errorf("%q: %q %v %v", src, d, w, lw)
		}
	}
	// An empty dir means the default.
	if d, w, _ := templatesOf(t, "blueprints:\n  dir: \"\"\n", ""); d != "/cfg/blueprints" || len(w) != 0 {
		t.Errorf("empty: %q %v", d, w)
	}

	// A folder's .zenmd.yml cannot set it: the key is a warning, the global value stays.
	d, w, lw := templatesOf(t, "blueprints:\n  dir: /srv/g\n", "blueprints:\n  dir: /tmp/evil\n")
	if d != "/srv/g" || len(w) != 0 || len(lw) != 1 || !strings.Contains(lw[0], "blueprints can only be set in the global config") {
		t.Errorf("workspace key: %q %v %v", d, w, lw)
	}
	if d, _, _ := templatesOf(t, "", "blueprints:\n  dir: /tmp/evil\n"); d != "/cfg/blueprints" {
		t.Errorf("workspace only: %q", d)
	}
	// A nil config answers with the default.
	var nilCfg *Config
	if got, _ := nilCfg.BlueprintsDir("/x"); got != "/x/blueprints" {
		t.Errorf("nil: %q", got)
	}
}

func TestLegacyTemplatesKey(t *testing.T) {
	cfgDir := t.TempDir()
	g := write(t, filepath.Join(cfgDir, "config.yml"), "templates:\n  dir: /srv/old\n")
	c := Load(g, "")
	if d, w := c.BlueprintsDir(cfgDir); d != "/srv/old" || len(w) != 0 {
		t.Errorf("legacy templates: key is read: %q %v", d, w)
	}
	if n := c.Notice(); !strings.Contains(n, "templates: in config.yml is now blueprints:") || len(c.Warnings()) != 0 {
		t.Errorf("notice %q, warnings %v", n, c.Warnings())
	}
	// Both keys: blueprints: wins, templates: is noted as ignored.
	g = write(t, filepath.Join(cfgDir, "config.yml"), "templates:\n  dir: /srv/old\nblueprints:\n  dir: /srv/new\n")
	c = Load(g, "")
	if d, _ := c.BlueprintsDir(cfgDir); d != "/srv/new" || !strings.Contains(c.Notice(), "ignored because blueprints: is set") {
		t.Errorf("both: %q %q", d, c.Notice())
	}
	// The old default folder is used when only it exists.
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "templates", "sites"), 0o755); err != nil {
		t.Fatal(err)
	}
	if d, _ := Load("", "").BlueprintsDir(dir); d != filepath.Join(dir, "templates") {
		t.Errorf("old default: %q", d)
	}
}

func TestFolderTemplateRules(t *testing.T) {
	root := t.TempDir()
	write(t, filepath.Join(root, WorkspaceFile), "layout: docs\ntemplates:\n  \"how-tos/**\": how-to\n  \"getting-started/**\": tutorial\n  nav-like: [x]\n")
	c := Load("", root)
	rules := c.TemplateRules()
	if len(rules) != 2 || rules[0] != (TemplateRule{"how-tos/**", "how-to"}) || rules[1] != (TemplateRule{"getting-started/**", "tutorial"}) {
		t.Errorf("rules %v", rules)
	}
	// Only the bad entry warns; the key is known in a folder file.
	w := c.Warnings()
	if len(w) != 1 || strings.Contains(w[0], "unknown key") || !strings.Contains(w[0], "nav-like") {
		t.Errorf("warnings %v", w)
	}
	write(t, filepath.Join(root, WorkspaceFile), "templates:\n  \"a/**\": how-to\n")
	if w := Load("", root).Warnings(); len(w) != 0 {
		t.Errorf("clean rules warn: %v", w)
	}
}
