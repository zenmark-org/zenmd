package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// BlueprintsDir is where blueprints live (formerly site templates): the
// global config's
//
//	blueprints:
//	  dir: ~/Documents/zen blueprints
//
// "~" is the home folder and a relative path is relative to configDir, the
// folder of config.yml. Without the key it is <configDir>/blueprints, or the
// old <configDir>/templates when only that exists. The old key templates: is
// read as blueprints: (and noted, see Notices). A folder's .zenmd.yml cannot
// set it (that is a warning when the file loads). The strings returned are
// problems with the key; the default is used then.
func (c *Config) BlueprintsDir(configDir string) (string, []string) {
	def := filepath.Join(configDir, "blueprints")
	if _, err := os.Stat(def); err != nil {
		if old := filepath.Join(configDir, "templates"); isDir(old) {
			def = old
		}
	}
	key, raw := "blueprints", any(nil)
	if c != nil {
		raw = c.blueprints
		if raw == nil && c.templates != nil {
			key, raw = "templates", c.templates
		}
	}
	if raw == nil {
		return def, nil
	}
	m, ok := raw.(map[string]any)
	if !ok {
		return def, []string{key + " must be a mapping such as { dir: ~/Documents/zen blueprints }"}
	}
	var warns []string
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	dir := ""
	for _, k := range keys {
		if k != "dir" {
			warns = append(warns, fmt.Sprintf("%s: unknown key %q", key, k))
			continue
		}
		s, ok := m[k].(string)
		if !ok {
			warns = append(warns, key+": dir must be a folder path")
			continue
		}
		dir = strings.TrimSpace(s)
	}
	if len(warns) > 0 {
		return def, warns[:1]
	}
	if dir == "" {
		return def, nil
	}
	switch {
	case dir == "~" || strings.HasPrefix(dir, "~/"):
		if home, err := os.UserHomeDir(); err == nil && home != "" {
			dir = filepath.Join(home, strings.TrimPrefix(dir, "~"))
		}
	}
	if !filepath.IsAbs(dir) {
		dir = filepath.Join(configDir, dir)
	}
	return filepath.Clean(dir), nil
}

func isDir(p string) bool { fi, err := os.Stat(p); return err == nil && fi.IsDir() }

// TemplateRule maps a path pattern to the id of a page template: the folder's
//
//	templates:
//	  "how-tos/**": how-to
//
// Pages that match follow that template. Rules are tried in order and the first
// match wins. The checker that reads them comes later; they are kept here.
type TemplateRule struct{ Pattern, Name string }

// TemplateRules are the path rules of the folder's .zenmd.yml templates: key.
func (c *Config) TemplateRules() []TemplateRule {
	if c == nil {
		return nil
	}
	return append([]TemplateRule(nil), c.templateRules...)
}
