package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// TemplatesDir is the folder that holds both kinds of template, blueprints/
// and pages/: the global config's
//
//	templates:
//	  dir: ~/Documents/zen templates
//
// "~" is the home folder and a relative path is relative to configDir, the
// folder of config.yml. Without the key it is <configDir>/templates. In a
// folder's .zenmd.yml the same key holds the path rules (TemplateRules), a
// different thing, so a folder cannot set this one. The strings returned are
// problems with the key; the default is used then.
func (c *Config) TemplatesDir(configDir string) (string, []string) {
	def := filepath.Join(configDir, "templates")
	var raw any
	if c != nil {
		raw = c.templates
	}
	if raw == nil {
		return def, nil
	}
	m, ok := raw.(map[string]any)
	if !ok {
		return def, []string{"templates must be a mapping such as { dir: ~/Documents/zen templates }"}
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
			warns = append(warns, fmt.Sprintf("templates: unknown key %q", k))
			continue
		}
		s, ok := m[k].(string)
		if !ok {
			warns = append(warns, "templates: dir must be a folder path")
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
