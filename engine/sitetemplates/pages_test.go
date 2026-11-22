package sitetemplates

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
)

const howTo = `id: how-to
name: How-to guide
description: Steps to one result
version: VER
layout: docs
fields:
  title: { required: true }
  audience: { type: string, default: everyone }
structure:
  - heading: { level: 1, count: 1 }
  - section:
      heading: Steps
      contains: [ { list: ordered, min-items: 2 } ]
  - end: {}
skeleton: |
  # {{title}}

  ## Steps

  1. One
  2. Two
`

func pagesFS(ver string) fstest.MapFS {
	return fstest.MapFS{
		"how-to.yml": {Data: []byte(strings.Replace(howTo, "VER", ver, 1))},
		"term.yml":   {Data: []byte("id: term\nname: Term\ndescription: A term\nversion: " + ver + "\nlayout: term\nfields:\n  title: { required: true }\nskeleton: |\n  A definition.\n")},
	}
}

func TestInstallPages(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "templates")
	if err := InstallPages(dir, pagesFS("1.0.0")); err != nil {
		t.Fatal(err)
	}
	if got := walk(t, filepath.Join(dir, "pages")); strings.Join(got, ",") != ".zen-templates.yml,how-to.yml,term.yml" {
		t.Fatalf("installed: %v", got)
	}
	rec := read(t, dir, "pages/.zen-templates.yml")
	if !strings.Contains(rec, "how-to: 1.0.0") || !strings.Contains(rec, "term: 1.0.0") || strings.Count(rec, "sha256:") != 2 {
		t.Errorf("record:\n%s", rec)
	}
	// Installed once: an edit stays, a deleted one does not come back, even in a newer version.
	put(t, dir, map[string]string{"pages/how-to.yml": "id: how-to\nname: Mine\n"})
	if err := os.Remove(filepath.Join(dir, "pages", "term.yml")); err != nil {
		t.Fatal(err)
	}
	if err := InstallPages(dir, pagesFS("1.1.0")); err != nil {
		t.Fatal(err)
	}
	if read(t, dir, "pages/how-to.yml") != "id: how-to\nname: Mine\n" {
		t.Error("overwritten")
	}
	if _, err := os.Stat(filepath.Join(dir, "pages", "term.yml")); err == nil {
		t.Error("deleted template came back")
	}
	// A file the user put there first is theirs: not replaced, not recorded.
	dir2 := t.TempDir()
	put(t, dir2, map[string]string{"pages/term.yml": "id: term\nname: Mine\n"})
	if err := InstallPages(dir2, pagesFS("1.0.0")); err != nil {
		t.Fatal(err)
	}
	if read(t, dir2, "pages/term.yml") != "id: term\nname: Mine\n" || strings.Contains(read(t, dir2, "pages/.zen-templates.yml"), "term") {
		t.Error("user's own touched or recorded")
	}
}

func TestInstallPagesReadOnly(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root writes everywhere")
	}
	base := t.TempDir()
	if err := os.Chmod(base, 0o500); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(base, 0o700)
	if err := InstallPages(filepath.Join(base, "templates"), pagesFS("1.0.0")); err == nil {
		t.Fatal("want an error")
	}
	// The built-ins still list.
	l, _ := ListPages(filepath.Join(base, "templates"), pagesFS("1.0.0"), "")
	if len(l) != 2 || l[0].Source != SourceBuiltin {
		t.Errorf("list: %+v", l)
	}
}

func TestListPagesSources(t *testing.T) {
	dir, root := t.TempDir(), t.TempDir()
	b := pagesFS("1.0.0")
	if err := InstallPages(dir, b); err != nil {
		t.Fatal(err)
	}
	put(t, dir, map[string]string{"pages/lesson.yml": "id: lesson\nname: Lesson\nlayout: docs\nfields:\n  title: {}\nskeleton: |\n  # {{title}}\n"})
	put(t, root, map[string]string{".zen/templates/term.yml": "id: term\nname: Term here\nlayout: term\nfields:\n  title: {}\nskeleton: x\n"})
	list, warns := ListPages(dir, b, root)
	if len(warns) != 0 {
		t.Fatal(warns)
	}
	by := map[string]PageTemplate{}
	for _, p := range list {
		by[p.ID] = p
	}
	if len(list) != 3 {
		t.Fatalf("list: %+v", list)
	}
	// The order on the left: this folder's own, then yours, then the built-ins, each by name.
	if list[0].ID != "term" || list[1].ID != "how-to" || list[2].ID != "lesson" {
		t.Errorf("order: %v %v %v", list[0].ID, list[1].ID, list[2].ID)
	}
	if by["term"].Source != SourceFolder || by["term"].Name != "Term here" || by["term"].Path != filepath.Join(root, ".zen", "templates", "term.yml") {
		t.Errorf("term: %+v", by["term"])
	}
	// An installed copy sits in templates/pages and is yours to edit, so it is Yours (Built-in is only what has no file).
	if by["how-to"].Source != SourceYours || by["how-to"].Path != filepath.Join(dir, "pages", "how-to.yml") || by["how-to"].Version != "1.0.0" {
		t.Errorf("how-to: %+v", by["how-to"])
	}
	if by["lesson"].Source != SourceYours || by["lesson"].Version != "" {
		t.Errorf("lesson: %+v", by["lesson"])
	}
}

func TestListPagesBuiltinWhenDeleted(t *testing.T) {
	dir := t.TempDir()
	b := pagesFS("1.0.0")
	if err := InstallPages(dir, b); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(dir, "pages", "term.yml")); err != nil {
		t.Fatal(err)
	}
	// A deleted copy is not installed again, but the built-in one still works (lookup falls through to it).
	list, _ := ListPages(dir, b, "")
	by := map[string]PageTemplate{}
	for _, p := range list {
		by[p.ID] = p
	}
	if by["term"].Source != SourceBuiltin || by["term"].Path != "" || by["how-to"].Source != SourceYours {
		t.Errorf("list: %+v", list)
	}
}

func TestPageTemplateDetails(t *testing.T) {
	dir := t.TempDir()
	b := pagesFS("1.0.0")
	list, _ := ListPages(dir, b, "")
	var p PageTemplate
	for _, x := range list {
		if x.ID == "how-to" {
			p = x
		}
	}
	if p.Description != "Steps to one result" || p.Layout != "docs" || p.Version != "1.0.0" {
		t.Errorf("head: %+v", p)
	}
	if len(p.Fields) != 2 || p.Fields[0] != (PageField{"title", "string", true, ""}) || p.Fields[1] != (PageField{"audience", "string", false, "everyone"}) {
		t.Errorf("fields: %+v", p.Fields)
	}
	want := []string{
		"0 heading: level 1, count 1",
		"0 section: Steps",
		"1 list: ordered, min-items 2",
		"0 end",
	}
	var got []string
	for _, o := range p.Outline {
		got = append(got, string(rune('0'+o.Depth))+" "+o.Text)
	}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Errorf("outline:\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
	if !strings.HasPrefix(p.Skeleton, "# {{title}}") {
		t.Errorf("skeleton %q", p.Skeleton)
	}
	// A template with no structure has no outline.
	for _, x := range list {
		if x.ID == "term" && len(x.Outline) != 0 {
			t.Errorf("term outline: %+v", x.Outline)
		}
	}
}

func TestListPagesBadFile(t *testing.T) {
	dir := t.TempDir()
	put(t, dir, map[string]string{"pages/bad.yml": "id: [", "pages/notes.txt": "x", "pages/.hidden.yml": "id: h\n", "pages/how-to.yml.new": "x"})
	list, _ := ListPages(dir, pagesFS("1.0.0"), "")
	var bad *PageTemplate
	for i := range list {
		if list[i].ID == "bad" {
			bad = &list[i]
		}
		if list[i].ID == "notes" || list[i].ID == "h" || strings.Contains(list[i].ID, "new") {
			t.Errorf("listed %q", list[i].ID)
		}
	}
	if bad == nil || bad.Err == "" {
		t.Errorf("a broken file is listed with its problem: %+v", list)
	}
}

func TestPageUpdate(t *testing.T) {
	dir := t.TempDir()
	if err := InstallPages(dir, pagesFS("1.0.0")); err != nil {
		t.Fatal(err)
	}
	b2 := pagesFS("1.1.0")
	list, _ := ListPages(dir, b2, "")
	for _, p := range list {
		if p.Update != "1.1.0" {
			t.Errorf("%s update %q", p.ID, p.Update)
		}
	}
	// Untouched: in place.
	res, err := UpdatePage(dir, b2, "how-to")
	if err != nil || !res.InPlace || res.Path != filepath.Join(dir, "pages", "how-to.yml") || !strings.Contains(read(t, dir, "pages/how-to.yml"), "version: 1.1.0") {
		t.Fatalf("in place: %+v %v", res, err)
	}
	if _, err := UpdatePage(dir, b2, "how-to"); err == nil || !strings.Contains(err.Error(), "up to date") {
		t.Errorf("again: %v", err)
	}
	// Edited: <id>.yml.new, the copy is kept.
	put(t, dir, map[string]string{"pages/term.yml": "id: term\nname: Mine\nversion: 1.0.0\n"})
	res, err = UpdatePage(dir, b2, "term")
	if err != nil || res.InPlace || res.Path != filepath.Join(dir, "pages", "term.yml.new") {
		t.Fatalf("edited: %+v %v", res, err)
	}
	if read(t, dir, "pages/term.yml") != "id: term\nname: Mine\nversion: 1.0.0\n" || !strings.Contains(read(t, dir, "pages/term.yml.new"), "version: 1.1.0") {
		t.Error("contents")
	}
	if _, err := UpdatePage(dir, pagesFS("1.2.0"), "term"); err == nil || !strings.Contains(err.Error(), "term.yml.new") {
		t.Errorf("an existing .new is not replaced: %v", err)
	}
	for _, id := range []string{"nosuch", "", "../x"} {
		if _, err := UpdatePage(dir, b2, id); err == nil {
			t.Errorf("%q accepted", id)
		}
	}
	// A record without a fingerprint: treated as edited.
	dir3 := t.TempDir()
	put(t, dir3, map[string]string{"pages/term.yml": read(t, dir, "pages/term.yml.new"), "pages/.zen-templates.yml": "installed:\n    term: 1.0.0\n"})
	res, err = UpdatePage(dir3, b2, "term")
	if err != nil || res.InPlace {
		t.Errorf("no fingerprint: %+v %v", res, err)
	}
}

// The lookup of a page template: the folder's own, then templates/pages, then the built-in one.
func TestPageTemplateLookupOrder(t *testing.T) {
	dir := t.TempDir()
	tp := Template{PageSource: fstest.MapFS{"how-to.yml": {Data: []byte("id: how-to\nname: Built-in\nlayout: docs\nskeleton: b\n")}}}
	if p, err := tp.pageTemplate("how-to"); err != nil || p.Name != "Built-in" {
		t.Fatalf("builtin: %+v %v", p, err)
	}
	tp.PagesDir = filepath.Join(dir, "pages")
	put(t, dir, map[string]string{"pages/how-to.yml": "id: how-to\nname: Yours\nlayout: docs\nskeleton: y\n"})
	if p, err := tp.pageTemplate("how-to"); err != nil || p.Name != "Yours" {
		t.Fatalf("pages: %+v %v", p, err)
	}
	tp.fsys = fstest.MapFS{".zen/templates/how-to.yml": {Data: []byte("id: how-to\nname: Blueprint\nlayout: docs\nskeleton: f\n")}}
	if p, err := tp.pageTemplate("how-to"); err != nil || p.Name != "Blueprint" {
		t.Fatalf("blueprint's own: %+v %v", p, err)
	}
	// An id that tries to leave the folder finds nothing.
	if _, err := tp.pageTemplate("../x"); err == nil {
		t.Error("../x found")
	}
}

// An install pass brings untouched copies up to the built-in version, silently.
func TestInstallPagesUpdatesUntouched(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "templates")
	if err := InstallPages(dir, pagesFS("1.0.0")); err != nil {
		t.Fatal(err)
	}
	b2 := pagesFS("1.1.0")
	// Edited: left alone, update still available.
	edited := "id: term\nname: Mine\nversion: 1.0.0\n"
	put(t, dir, map[string]string{"pages/term.yml": edited})
	if err := InstallPages(dir, b2); err != nil {
		t.Fatal(err)
	}
	if got := read(t, dir, "pages/how-to.yml"); got != string(b2["how-to.yml"].Data) {
		t.Errorf("untouched copy not updated:\n%s", got)
	}
	if read(t, dir, "pages/term.yml") != edited {
		t.Error("edited copy changed")
	}
	if _, err := os.Stat(filepath.Join(dir, "pages", "term.yml.new")); err == nil {
		t.Error("the pass writes no .new")
	}
	rec := read(t, dir, "pages/.zen-templates.yml")
	if !strings.Contains(rec, "how-to: 1.1.0") || !strings.Contains(rec, "term: 1.0.0") {
		t.Errorf("record:\n%s", rec)
	}
	list, _ := ListPages(dir, b2, "")
	for _, p := range list {
		if want := map[string]string{"how-to": "", "term": "1.1.0"}[p.ID]; p.Update != want {
			t.Errorf("%s update %q, want %q", p.ID, p.Update, want)
		}
	}
	// The new fingerprint is recorded: the next version follows too.
	if err := InstallPages(dir, pagesFS("1.2.0")); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(read(t, dir, "pages/how-to.yml"), "version: 1.2.0") {
		t.Error("second update missing")
	}
}

func TestInstallPagesDeletedStaysDeletedAndNoFingerprint(t *testing.T) {
	dir := t.TempDir()
	if err := InstallPages(dir, pagesFS("1.0.0")); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(dir, "pages", "term.yml")); err != nil {
		t.Fatal(err)
	}
	if err := InstallPages(dir, pagesFS("1.1.0")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "pages", "term.yml")); err == nil {
		t.Error("deleted copy came back")
	}
	// A record without fingerprints: nothing is replaced.
	d2 := t.TempDir()
	old := "id: term\nname: Term\nversion: 1.0.0\n"
	put(t, d2, map[string]string{"pages/term.yml": old, "pages/.zen-templates.yml": "installed:\n    term: 1.0.0\n"})
	if err := InstallPages(d2, pagesFS("1.1.0")); err != nil {
		t.Fatal(err)
	}
	if read(t, d2, "pages/term.yml") != old {
		t.Error("copy without a fingerprint was replaced")
	}
}

// pages/release-notes.yml installed at 1.0.0 with a fingerprint follows the built-in.
func TestInstallPagesReleaseNotes(t *testing.T) {
	mk := func(ver, extra string) fstest.MapFS {
		return fstest.MapFS{"release-notes.yml": {Data: []byte("id: release-notes\nname: Release notes\nversion: " + ver + "\nlayout: docs\nfields:\n  title: { required: true }\n" + extra + "skeleton: |\n  # {{title}}\n")}}
	}
	dir := t.TempDir()
	if err := InstallPages(dir, mk("1.0.0", "")); err != nil {
		t.Fatal(err)
	}
	b := mk("1.1.0", "  version: { required: true }\n  date: { required: true }\n")
	if err := InstallPages(dir, b); err != nil {
		t.Fatal(err)
	}
	if read(t, dir, "pages/release-notes.yml") != string(b["release-notes.yml"].Data) {
		t.Errorf("not the built-in's content:\n%s", read(t, dir, "pages/release-notes.yml"))
	}
}
