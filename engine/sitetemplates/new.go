package sitetemplates

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// starter is the commented blueprint.yml `zenmd blueprint new` writes.
const starter = `# A blueprint: a folder you start new sites from. This file says what a new
# folder gets. Edit it, then try it: zenmd init --blueprint {{id}} some-folder
id: {{id}}
name: {{name}}
description: What a folder made from this blueprint is for
version: 1.0.0
order: 100                     # where it shows in the list: lower first

# The layouts and plugins the pages use (checked against the app).
requires: { layouts: [docs, landing], plugins: [] }

# Questions asked when a folder is made. {{title}}, {{author}} and any key you
# add here can be used in every file and page of the blueprint.
ask:
  title:  { label: Site title, default: "{{folder}}" }
  author: { label: Author, default: "the team" }

layout: docs                   # the folder's default layout (.zenmd.yml layout:)

# README.md: the start page. It links each section below, in order, which is
# also their order in the sidebar.
home:
  title: "{{title}}"

# Each section is a folder with an index.md. Add as many as you like.
#   template: the page template its pages follow (a path rule in .zenmd.yml)
#   index:    { layout: landing } is the default (cards); { template: x } or
#             { layout: glossary } make the index from that instead
#   order:    newest-first sorts the pages by date or version, newest on top
#   pages:    starting pages, written from the template's skeleton
sections:
  - id: getting-started
    title: Getting started
    folder: getting-started
    template: tutorial
    index: { layout: landing, tagline: "From nothing to your first result" }
    pages:
      - { file: first-steps.md, title: Your first steps }

# Page templates copied into the new folder's .zen/templates. You can also keep
# your own in this folder's .zen/templates/<id>.yml.
templates: [tutorial]

# Other files in this folder are copied as they are. Names and text can use
# {{title}}, {{author}} and {{date}}.
`

var newIDRE = regexp.MustCompile(`^[a-z][a-z0-9-]{0,63}$`)

// New starts a blueprint in the blueprints folder dir: <dir>/sites/<id> with a
// commented blueprint.yml, or, with from (a blueprint of the folder or of
// builtins), a copy of that one under the new id. It never overwrites and
// returns the new folder.
func New(dir string, builtins fs.FS, id, from string) (string, error) {
	if !newIDRE.MatchString(id) {
		return "", fmt.Errorf("the id %q must use lower case letters, digits and -, and start with a letter", id)
	}
	if dir == "" {
		return "", errors.New("there is no blueprints folder")
	}
	sites := SitesDir(dir)
	dest := filepath.Join(sites, id)
	if _, err := os.Lstat(dest); err == nil {
		return "", fmt.Errorf("%s already exists in the blueprints folder", id)
	}
	if err := os.MkdirAll(sites, 0o755); err != nil {
		return "", err
	}
	if from == "" {
		name := strings.ToUpper(id[:1]) + strings.ReplaceAll(id[1:], "-", " ")
		text := strings.NewReplacer("{{id}}", id, "{{name}}", name).Replace(starter)
		tmp, err := os.MkdirTemp(sites, ".tmp-new-")
		if err != nil {
			return "", err
		}
		defer os.RemoveAll(tmp)
		if err := os.WriteFile(filepath.Join(tmp, ManifestFile), []byte(text), 0o644); err != nil {
			return "", err
		}
		if err := os.Chmod(tmp, 0o755); err != nil {
			return "", err
		}
		return dest, os.Rename(tmp, dest)
	}
	// From another blueprint: this folder's, else a built-in one.
	var src fs.FS
	list, _ := List(dir)
	for _, t := range list {
		if t.ID == from && t.Err == "" {
			src = t.fsys
		}
	}
	if src == nil {
		b, _ := Builtins(builtins)
		for _, t := range b {
			if t.ID == from && t.Err == "" {
				src = t.fsys
			}
		}
	}
	if src == nil {
		return "", fmt.Errorf("there is no blueprint %q to start from", from)
	}
	if err := copyTree(src, dest); err != nil {
		return "", err
	}
	// The copy has its own id (and the new name), and is read as a blueprint.yml.
	name := ManifestFile
	data, err := os.ReadFile(filepath.Join(dest, ManifestFile))
	if err != nil {
		name = LegacyManifestFile
		data, err = os.ReadFile(filepath.Join(dest, LegacyManifestFile))
	}
	if err != nil {
		return "", err
	}
	text := regexp.MustCompile(`(?m)^id:.*$`).ReplaceAllString(string(data), "id: "+id)
	text = regexp.MustCompile(`(?m)^name:.*$`).ReplaceAllString(text, "name: "+scalar(strings.ToUpper(id[:1])+strings.ReplaceAll(id[1:], "-", " ")))
	if err := os.WriteFile(filepath.Join(dest, ManifestFile), []byte(text), 0o644); err != nil {
		return "", err
	}
	if name == LegacyManifestFile {
		_ = os.Remove(filepath.Join(dest, LegacyManifestFile))
	}
	return dest, nil
}
