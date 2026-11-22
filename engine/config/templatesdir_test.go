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
	d, w := c.TemplatesDir(cfgDir)
	return filepath.ToSlash(strings.TrimPrefix(d, dir)), w, c.Warnings()
}

func TestTemplatesDir(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	// Default: the templates folder next to config.yml.
	if d, w, lw := templatesOf(t, "", ""); d != "/cfg/templates" || len(w) != 0 || len(lw) != 0 {
		t.Errorf("default: %q %v %v", d, w, lw)
	}
	if d, _, _ := templatesOf(t, "theme: dark\n", ""); d != "/cfg/templates" {
		t.Errorf("no key: %q", d)
	}
	// ~ expands to the home folder; an absolute path is kept as it is.
	dir := t.TempDir()
	g := write(t, filepath.Join(dir, "config.yml"), "templates:\n  dir: ~/Documents/zen templates\n")
	got, warns := Load(g, "").TemplatesDir(dir)
	if got != filepath.Join(home, "Documents", "zen templates") || len(warns) != 0 {
		t.Errorf("tilde: %q %v", got, warns)
	}
	g = write(t, filepath.Join(dir, "config.yml"), "templates:\n  dir: /srv/zen/t\n")
	if got, _ = Load(g, "").TemplatesDir(dir); got != "/srv/zen/t" {
		t.Errorf("absolute: %q", got)
	}
	// A relative path is relative to the config folder, and is cleaned.
	g = write(t, filepath.Join(dir, "config.yml"), "templates:\n  dir: ../shared/./site templates\n")
	if got, _ = Load(g, "").TemplatesDir(dir); got != filepath.Join(filepath.Dir(dir), "shared", "site templates") {
		t.Errorf("relative: %q", got)
	}
	g = write(t, filepath.Join(dir, "config.yml"), "templates:\n  dir: \"~\"\n")
	if got, _ = Load(g, "").TemplatesDir(dir); got != home {
		t.Errorf("home alone: %q", got)
	}

	// Wrong shapes: the default is used, and TemplatesDir says why.
	for _, src := range []string{"templates: yes\n", "templates:\n  dir: 3\n", "templates:\n  dir: [a]\n", "templates:\n  folder: x\n"} {
		d, w, lw := templatesOf(t, src, "")
		if d != "/cfg/templates" || len(w) != 1 || !strings.Contains(w[0], "templates") || len(lw) != 0 {
			t.Errorf("%q: %q %v %v", src, d, w, lw)
		}
	}
	// An empty dir means the default.
	if d, w, _ := templatesOf(t, "templates:\n  dir: \"\"\n", ""); d != "/cfg/templates" || len(w) != 0 {
		t.Errorf("empty: %q %v", d, w)
	}

	// A folder's .zenmd.yml templates: is the path rules, never the folder: the global value stays.
	d, w, lw := templatesOf(t, "templates:\n  dir: /srv/g\n", "templates:\n  \"a/**\": how-to\n")
	if d != "/srv/g" || len(w) != 0 || len(lw) != 0 {
		t.Errorf("workspace rules: %q %v %v", d, w, lw)
	}
	if d, _, _ := templatesOf(t, "", "templates:\n  dir: /tmp/evil\n"); d != "/cfg/templates" {
		t.Errorf("workspace dir: %q", d)
	}
	var nilCfg *Config
	if got, _ := nilCfg.TemplatesDir("/x"); got != "/x/templates" {
		t.Errorf("nil: %q", got)
	}
}

// The global blueprints: key was never released: it is an unknown key now, with no notice and no effect.
func TestBlueprintsKeyIsGone(t *testing.T) {
	cfgDir := t.TempDir()
	g := write(t, filepath.Join(cfgDir, "config.yml"), "blueprints:\n  dir: /srv/old\n")
	c := Load(g, "")
	if d, w := c.TemplatesDir(cfgDir); d != filepath.Join(cfgDir, "templates") || len(w) != 0 {
		t.Errorf("blueprints: must not set the folder: %q %v", d, w)
	}
	if c.Notice() != "" {
		t.Errorf("no notice for a key that never shipped: %q", c.Notice())
	}
	if w := c.Warnings(); len(w) != 1 || !strings.Contains(w[0], "blueprints") {
		t.Errorf("unknown key warning: %v", w)
	}
	// templates: in the global file is the real key now, with no notice.
	g = write(t, filepath.Join(cfgDir, "config.yml"), "templates:\n  dir: /srv/t\n")
	c = Load(g, "")
	if d, _ := c.TemplatesDir(cfgDir); d != "/srv/t" || c.Notice() != "" || len(c.Warnings()) != 0 {
		t.Errorf("templates: %q %q %v", d, c.Notice(), c.Warnings())
	}
	_ = os.Stdout
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
