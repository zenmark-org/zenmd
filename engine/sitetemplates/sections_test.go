package sitetemplates

import (
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
)

// pageTemplates is a fake set of page templates.
func pageTemplates() fstest.MapFS {
	return fstest.MapFS{
		"tutorial.yml":      {Data: []byte("id: tutorial\nname: Tutorial\ndescription: Learn by doing\nlayout: docs\nfields:\n  title: { required: true }\nskeleton: |\n  # {{title}}\n\n  By {{author}} for {{product}} on {{today}}.\n\n  ## Steps\n\n  1. First\n")},
		"how-to.yml":        {Data: []byte("id: how-to\nname: How-to\nlayout: docs\nfields:\n  title: { required: true }\nskeleton: |\n  # {{title}}\n\n  Steps.\n")},
		"overview.yml":      {Data: []byte("id: overview\nname: Overview\nlayout: docs\nfields:\n  title: { required: true }\nskeleton: |\n  # {{title}}\n\n  {{product}} is a thing.\n\n  ## Where to go next\n\n  {{sections}}\n")},
		"release-notes.yml": {Data: []byte("id: release-notes\nname: Release notes\nlayout: docs\nfields:\n  version: { required: true }\n  date: { type: date, required: true, default: \"{{today}}\" }\nskeleton: |\n  # {{title}} {{version}}\n\n  ## Added\n\n  - Everything\n")},
	}
}

const sectionsManifest = `id: docs
name: Docs
version: 1.0.0
ask:
  title:   { label: Site title, default: "{{folder}}" }
  author:  { label: Author, default: "the team" }
  product: { label: Product, default: "{{title}}" }
layout: docs
home:
  title: "{{title}}"
sections:
  - id: overview
    title: Overview
    folder: overview
    template: overview
    index: { template: overview }
    pages:
      - { file: pitch.md, title: "{{product}} in five slides", layout: slides }
  - id: getting-started
    title: Getting started
    folder: getting-started
    template: tutorial
    index: { layout: landing, tagline: "From nothing to a result" }
    pages:
      - { file: installation.md, title: Installation }
  - id: how-tos
    title: How-tos
    folder: how-tos
    template: how-to
  - id: about
    title: About us
    file: about.md
  - id: releases
    title: Releases
    folder: releases
    template: release-notes
    order: newest-first
    pages: [ { file: "{{today}}-first.md", title: "First release", set: { version: 0.1.0 } } ]
files: [CONTRIBUTING.md]
templates: [overview, tutorial, how-to, release-notes]
`

func buildSections(t *testing.T, manifest string, files map[string]string) (string, Result) {
	t.Helper()
	fixedDate(t)
	if files == nil {
		files = map[string]string{}
	}
	if _, ok := files["CONTRIBUTING.md"]; !ok {
		files["CONTRIBUTING.md"] = "Contribute to {{title}}.\n"
	}
	tp := tmpl(t, t.TempDir(), "docs", manifest, files)
	tp.PageSource = pageTemplates()
	target := filepath.Join(t.TempDir(), "My Docs")
	res, err := Create(tp, target, map[string]string{"title": "My Docs", "author": "Ana", "product": "Zed"})
	if err != nil {
		t.Fatal(err)
	}
	return target, res
}

func TestSectionsBuildFolders(t *testing.T) {
	target, _ := buildSections(t, sectionsManifest, nil)
	got := strings.Join(walk(t, target), "\n")
	for _, want := range []string{
		"README.md", "CONTRIBUTING.md", ".zenmd.yml",
		"overview/index.md", "overview/pitch.md",
		"getting-started/index.md", "getting-started/installation.md",
		"how-tos/index.md", "about.md",
		"releases/index.md", "releases/2026-11-17-first.md",
		".zen/templates/overview.yml", ".zen/templates/tutorial.yml", ".zen/templates/how-to.yml", ".zen/templates/release-notes.yml",
	} {
		if !strings.Contains("\n"+got+"\n", "\n"+want+"\n") {
			t.Errorf("missing %s in:\n%s", want, got)
		}
	}
	if strings.Contains(got, "blueprint.yml") || strings.Contains(got, "template.yml") {
		t.Error("the manifest was copied")
	}
}

func TestSectionsHomeLinksInOrder(t *testing.T) {
	target, _ := buildSections(t, sectionsManifest, nil)
	home := read(t, target, "README.md")
	if !strings.Contains(home, "title: My Docs") || !strings.Contains(home, "# My Docs") {
		t.Errorf("home title:\n%s", home)
	}
	order := []string{"[Overview](overview/index.md)", "[Getting started](getting-started/index.md)", "[How-tos](how-tos/index.md)", "[About us](about.md)", "[Releases](releases/index.md)"}
	last := -1
	for _, l := range order {
		i := strings.Index(home, l)
		if i < 0 || i < last {
			t.Fatalf("link %q missing or out of order (%d after %d):\n%s", l, i, last, home)
		}
		last = i
	}
}

func TestSectionsIndexPages(t *testing.T) {
	target, _ := buildSections(t, sectionsManifest, nil)
	// A landing index with a tagline.
	gs := read(t, target, "getting-started/index.md")
	if !strings.Contains(gs, "layout: landing") || !strings.Contains(gs, "tagline: From nothing to a result") || !strings.Contains(gs, "# Getting started") {
		t.Errorf("landing index:\n%s", gs)
	}
	// No index: setting means a landing page with cards.
	if h := read(t, target, "how-tos/index.md"); !strings.Contains(h, "layout: landing") || !strings.Contains(h, "# How-tos") {
		t.Errorf("default index:\n%s", h)
	}
	// An index from a page template: the skeleton, with a link to every section filled in.
	ov := read(t, target, "overview/index.md")
	for _, w := range []string{"# Overview", "Zed is a thing.", "[Getting started](../getting-started/index.md)", "[Releases](../releases/index.md)"} {
		if !strings.Contains(ov, w) {
			t.Errorf("overview index lacks %q:\n%s", w, ov)
		}
	}
	if strings.Contains(ov, "[Overview]") {
		t.Errorf("the overview links to itself:\n%s", ov)
	}
	// The file section is one page.
	if a := read(t, target, "about.md"); !strings.Contains(a, "title: About us") || !strings.Contains(a, "# About us") {
		t.Errorf("file section:\n%s", a)
	}
}

func TestSectionsPathRulesAndConfig(t *testing.T) {
	target, _ := buildSections(t, sectionsManifest, nil)
	cfg := read(t, target, ".zenmd.yml")
	for _, w := range []string{"layout: docs", "templates:", `"overview/**": overview`, `"getting-started/**": tutorial`, `"how-tos/**": how-to`, `"releases/**": release-notes`} {
		if !strings.Contains(cfg, w) {
			t.Errorf(".zenmd.yml lacks %q:\n%s", w, cfg)
		}
	}
	// The generated file must load without warnings (templates: is a known key in a folder file).
	// (checked in engine/config; here the order of rules follows the sections)
	if strings.Index(cfg, "overview/**") > strings.Index(cfg, "releases/**") {
		t.Error("rules out of section order")
	}
}

func TestSectionsMergeWithOwnZenmd(t *testing.T) {
	target, _ := buildSections(t, sectionsManifest, map[string]string{".zenmd.yml": "version: 1\nplugins:\n  codeblock: true\n"})
	cfg := read(t, target, ".zenmd.yml")
	if !strings.Contains(cfg, "plugins:") || !strings.Contains(cfg, "layout: docs") || !strings.Contains(cfg, `"how-tos/**": how-to`) {
		t.Errorf("merge:\n%s", cfg)
	}
	if strings.Count(cfg, "version: 1") != 1 {
		t.Errorf("version repeated:\n%s", cfg)
	}
}

func TestSectionsStartingPages(t *testing.T) {
	target, _ := buildSections(t, sectionsManifest, nil)
	// Written from the section template's skeleton, placeholders filled.
	in := read(t, target, "getting-started/installation.md")
	for _, w := range []string{"title: Installation", "# Installation", "By Ana for Zed on 2026-11-17.", "## Steps"} {
		if !strings.Contains(in, w) {
			t.Errorf("installation lacks %q:\n%s", w, in)
		}
	}
	if strings.Contains(in, "{{") {
		t.Errorf("placeholder left:\n%s", in)
	}
	// A page with its own layout keeps it and skips the section's template.
	pitch := read(t, target, "overview/pitch.md")
	if !strings.Contains(pitch, "title: Zed in five slides") || !strings.Contains(pitch, "layout: slides") || !strings.Contains(pitch, "template: false") {
		t.Errorf("slides page:\n%s", pitch)
	}
	if strings.Contains(pitch, "Zed is a thing") || !strings.Contains(pitch, "\n---\n") {
		t.Errorf("slides page body:\n%s", pitch)
	}
}

func TestSectionsNewestFirstAndSet(t *testing.T) {
	target, _ := buildSections(t, sectionsManifest, nil)
	idx := read(t, target, "releases/index.md")
	if !strings.Contains(idx, "nav:") || !strings.Contains(idx, "sort: newest-first") {
		t.Errorf("releases index:\n%s", idx)
	}
	rel := read(t, target, "releases/2026-11-17-first.md")
	for _, w := range []string{"title: First release", "version: 0.1.0", "date: 2026-11-17", "# First release 0.1.0", "## Added"} {
		if !strings.Contains(rel, w) {
			t.Errorf("release lacks %q:\n%s", w, rel)
		}
	}
	// Other indexes have no nav.
	if strings.Contains(read(t, target, "how-tos/index.md"), "nav:") {
		t.Error("nav on a section without order")
	}
}

func TestSectionsCopyPageTemplates(t *testing.T) {
	target, _ := buildSections(t, sectionsManifest, nil)
	if got := read(t, target, ".zen/templates/tutorial.yml"); !strings.Contains(got, "id: tutorial") || !strings.Contains(got, "{{title}}") {
		t.Errorf("template copied wrong (it keeps its own placeholders):\n%s", got)
	}
	// The blueprint's own copy wins over the built-in one.
	target, _ = buildSections(t, sectionsManifest, map[string]string{".zen/templates/how-to.yml": "id: how-to\nname: Mine\nlayout: docs\nskeleton: |\n  # {{title}}\n\n  MINE\n"})
	if got := read(t, target, ".zen/templates/how-to.yml"); !strings.Contains(got, "name: Mine") {
		t.Errorf("own template replaced:\n%s", got)
	}
}

func TestSectionsErrors(t *testing.T) {
	fixedDate(t)
	cases := map[string]string{
		"unknown page template": strings.Replace(sectionsManifest, "template: how-to", "template: nosuch", 1),
		"both folder and file":  "id: d\nname: D\nsections:\n  - { id: a, title: A, folder: a, file: a.md }\n",
		"neither":               "id: d\nname: D\nsections:\n  - { id: a, title: A }\n",
		"path escape":           "id: d\nname: D\nsections:\n  - { id: a, title: A, folder: ../a }\n",
		"duplicate id":          "id: d\nname: D\nsections:\n  - { id: a, title: A, folder: a }\n  - { id: a, title: B, folder: b }\n",
		"bad order":             "id: d\nname: D\nsections:\n  - { id: a, title: A, folder: a, order: sideways }\n",
		"unknown key":           "id: d\nname: D\nsections:\n  - { id: a, title: A, folder: a, colour: red }\n",
		"page escapes":          "id: d\nname: D\nsections:\n  - { id: a, title: A, folder: a, pages: [ { file: ../x.md, title: X } ] }\n",
	}
	for name, m := range cases {
		files := map[string]string{"CONTRIBUTING.md": "x"}
		dir := t.TempDir()
		root := filepath.Join(dir, "sites", "docs")
		put(t, root, map[string]string{"blueprint.yml": m, "CONTRIBUTING.md": files["CONTRIBUTING.md"]})
		list, _ := List(dir)
		if len(list) != 1 {
			t.Fatalf("%s: %v", name, list)
		}
		tp := list[0]
		tp.PageSource = pageTemplates()
		if tp.Err == "" { // the error may come at create time
			if _, err := Create(tp, filepath.Join(t.TempDir(), "x"), nil); err == nil {
				t.Errorf("%s: accepted", name)
			}
		}
	}
}

func TestLegacyTemplateYml(t *testing.T) {
	dir := t.TempDir()
	root := filepath.Join(dir, "sites", "old")
	put(t, root, map[string]string{"template.yml": "id: old\nname: Old\n", "README.md": "# {{title}}\n"})
	list, _ := List(dir)
	if len(list) != 1 || list[0].Err != "" || list[0].ID != "old" {
		t.Fatalf("legacy template.yml is read: %+v", list)
	}
	target := filepath.Join(t.TempDir(), "x")
	if _, err := Create(list[0], target, nil); err != nil {
		t.Fatal(err)
	}
	if got := walk(t, target); len(got) != 1 || got[0] != "README.md" {
		t.Errorf("files %v", got)
	}
	// blueprint.yml wins when both exist.
	put(t, root, map[string]string{"blueprint.yml": "id: old\nname: New name\n"})
	if list, _ = List(dir); list[0].Name != "New name" {
		t.Errorf("blueprint.yml should win: %+v", list[0])
	}
}

func TestNewBlueprint(t *testing.T) {
	dir := t.TempDir()
	out, err := New(dir, nil, "mine", "")
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(out) != "mine" {
		t.Errorf("folder %s", out)
	}
	text := read(t, out, "blueprint.yml")
	for _, w := range []string{"id: mine", "name:", "sections:", "# "} {
		if !strings.Contains(text, w) {
			t.Errorf("blueprint.yml lacks %q:\n%s", w, text)
		}
	}
	// The file is valid and listed, and builds.
	list, _ := List(dir)
	if len(list) != 1 || list[0].Err != "" || list[0].ID != "mine" {
		t.Fatalf("listed: %+v", list)
	}
	list[0].PageSource = pageTemplates()
	if _, err := Create(list[0], filepath.Join(t.TempDir(), "site"), nil); err != nil {
		t.Errorf("the starter blueprint builds: %v", err)
	}
	// Never overwrites; bad ids are refused.
	if _, err := New(dir, nil, "mine", ""); err == nil {
		t.Error("overwrote")
	}
	if _, err := New(dir, nil, "../x", ""); err == nil {
		t.Error("bad id accepted")
	}
	// --from copies another blueprint under a new id.
	b := builtinsFS("1.0.0")
	out, err = New(dir, b, "tech2", "techdocs")
	if err != nil {
		t.Fatal(err)
	}
	if got := read(t, out, "blueprint.yml"); !strings.Contains(got, "id: tech2") || strings.Contains(got, "id: techdocs") {
		t.Errorf("from: %s", got)
	}
	if read(t, out, "README.md") != "# {{title}}\n" {
		t.Error("from: files not copied")
	}
	if _, err := New(dir, b, "x2", "nosuch"); err == nil || !strings.Contains(err.Error(), "nosuch") {
		t.Errorf("unknown --from: %v", err)
	}
}
