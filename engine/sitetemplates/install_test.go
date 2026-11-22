package sitetemplates

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstall(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "templates") // does not exist yet
	b := builtinsFS("1.0.0")
	if err := Install(dir, b); err != nil {
		t.Fatal(err)
	}
	if got := walk(t, filepath.Join(dir, "blueprints")); strings.Join(got, ",") != ".zen-templates.yml,planner/.zenmd.yml,planner/blueprint.yml,techdocs/README.md,techdocs/blueprint.yml" {
		t.Fatalf("installed: %v", got)
	}
	rec := read(t, dir, "blueprints/.zen-templates.yml")
	if !strings.Contains(rec, "techdocs: 1.0.0") || !strings.Contains(rec, "planner: 2.0.0") {
		t.Errorf("record:\n%s", rec)
	}
	list, warns := ListWith(dir, b)
	if len(warns) != 0 || len(list) != 2 || list[0].ID != "techdocs" || !list[0].Builtin || !list[1].Builtin || list[0].Update != "" {
		t.Errorf("list: %+v %v", list, warns)
	}

	// An installed template is never overwritten, not even by a newer version.
	if err := os.WriteFile(filepath.Join(dir, "blueprints", "techdocs", "README.md"), []byte("mine"), 0o644); err != nil {
		t.Fatal(err)
	}
	b2 := builtinsFS("1.1.0")
	if err := Install(dir, b2); err != nil {
		t.Fatal(err)
	}
	if read(t, dir, "blueprints/techdocs/README.md") != "mine" {
		t.Error("overwritten")
	}
	// ... but it is listed as having an update.
	list, _ = ListWith(dir, b2)
	if list[0].Update != "1.1.0" || list[1].Update != "" {
		t.Errorf("update: %+v", list)
	}
	// Plain List (no bundle) knows the built-ins by the record, without update info.
	if plain, _ := List(dir); !plain[0].Builtin || plain[0].Update != "" {
		t.Errorf("plain list: %+v", plain[0])
	}

	// A preset the user deleted is not installed again.
	if err := os.RemoveAll(filepath.Join(dir, "blueprints", "planner")); err != nil {
		t.Fatal(err)
	}
	if err := Install(dir, b2); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "blueprints", "planner")); err == nil {
		t.Error("deleted preset came back")
	}
	if list, _ = ListWith(dir, b2); len(list) != 1 {
		t.Errorf("deleted preset listed: %+v", list)
	}

	// A folder the user made with the same name is theirs: not replaced, not recorded.
	dir2 := t.TempDir()
	put(t, dir2, map[string]string{"blueprints/techdocs/template.yml": "id: techdocs\nname: Mine\n", "blueprints/techdocs/x.md": "x"})
	if err := Install(dir2, b); err != nil {
		t.Fatal(err)
	}
	if read(t, dir2, "blueprints/techdocs/x.md") != "x" || strings.Contains(read(t, dir2, "blueprints/.zen-templates.yml"), "techdocs") {
		t.Error("user's own template touched or recorded")
	}
	if l, _ := ListWith(dir2, b); l[1].Builtin || l[1].Update != "" {
		t.Errorf("user's template flagged: %+v", l)
	}

	// Update writes the newer built-in next to the copy, once.
	res, err := Update(dir, b2, "techdocs")
	out := res.Path
	if err != nil || out != filepath.Join(dir, "blueprints", "techdocs.new") || res.InPlace {
		t.Fatalf("update: %+v %v", res, err)
	}
	if !strings.Contains(read(t, out, "blueprint.yml"), "1.1.0") || read(t, dir, "blueprints/techdocs/README.md") != "mine" {
		t.Error("update content or user's copy changed")
	}
	if l, _ := ListWith(dir, b2); len(l) != 1 || l[0].Update != "" {
		t.Errorf("after update: %+v", l) // the .new folder is not a template, and the record moved on
	}
	if _, err := Update(dir, b2, "techdocs"); err == nil {
		t.Error("second update must say it is up to date or that .new exists")
	}
	for _, id := range []string{"nosuch", "planner", ""} {
		if _, err := Update(dir, b2, id); err == nil {
			t.Errorf("update %q accepted", id)
		}
	}
}

func TestInstallReadOnly(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root writes everywhere")
	}
	base := t.TempDir()
	if err := os.Chmod(base, 0o500); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(base, 0o700)
	dir := filepath.Join(base, "templates")
	b := builtinsFS("1.0.0")
	if err := Install(dir, b); err == nil {
		t.Fatal("want an error for a folder that cannot be written")
	}
	// The bundle is listed straight from the app, and can create sites.
	list, warns := Builtins(b)
	if len(list) != 2 || len(warns) != 0 || list[0].ID != "techdocs" || !list[0].Builtin {
		t.Fatalf("builtins: %+v %v", list, warns)
	}
	target := filepath.Join(t.TempDir(), "My Docs")
	res, err := Create(list[0], target, map[string]string{"title": "T"})
	if err != nil || len(res.Files) != 1 || read(t, target, "README.md") != "# T\n" {
		t.Errorf("create from bundle: %+v %v", res, err)
	}
	// Dotfiles of a bundled template are copied as well.
	t2 := filepath.Join(t.TempDir(), "p")
	if _, err := Create(list[1], t2, nil); err != nil || read(t, t2, ".zenmd.yml") != "layout: wide\n" {
		t.Errorf("dotfile: %v", err)
	}
}

// Every install records a SHA-256 fingerprint of what it wrote.
func TestInstallFingerprints(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "templates")
	if err := Install(dir, builtinsFS("1.0.0")); err != nil {
		t.Fatal(err)
	}
	rec := read(t, dir, "blueprints/.zen-templates.yml")
	for _, id := range []string{"techdocs", "planner"} {
		if !strings.Contains(rec, id+": ") || strings.Count(rec, "sha256:") != 2 {
			t.Errorf("fingerprints for %s in:\n%s", id, rec)
		}
	}
	// The fingerprint is stable: a copy that was not touched still matches it.
	fp, err := fingerprintDir(filepath.Join(dir, "blueprints", "techdocs"))
	if err != nil || !strings.Contains(rec, fp) {
		t.Errorf("fingerprint %q not in record (%v):\n%s", fp, err, rec)
	}
}

func TestUpdateInPlaceWhenUntouched(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "templates")
	if err := Install(dir, builtinsFS("1.0.0")); err != nil {
		t.Fatal(err)
	}
	b2 := builtinsFS("1.1.0")
	res, err := Update(dir, b2, "techdocs")
	if err != nil || !res.InPlace || res.Path != filepath.Join(dir, "blueprints", "techdocs") {
		t.Fatalf("update: %+v %v", res, err)
	}
	if !strings.Contains(read(t, dir, "blueprints/techdocs/blueprint.yml"), "1.1.0") {
		t.Error("the copy was not replaced")
	}
	if _, err := os.Stat(filepath.Join(dir, "blueprints", "techdocs.new")); err == nil {
		t.Error("no .new for an untouched copy")
	}
	if l, _ := ListWith(dir, b2); l[0].Update != "" {
		t.Errorf("still an update: %+v", l[0])
	}
	// The new fingerprint is recorded, so the next update is in place again.
	res, err = Update(dir, builtinsFS("1.2.0"), "techdocs")
	if err != nil || !res.InPlace {
		t.Errorf("second update: %+v %v", res, err)
	}
	// No leftovers beside the copies.
	if got := walk(t, filepath.Join(dir, "blueprints")); strings.Join(got, ",") != ".zen-templates.yml,planner/.zenmd.yml,planner/blueprint.yml,techdocs/README.md,techdocs/blueprint.yml" {
		t.Errorf("files: %v", got)
	}
}

func TestUpdateEditedCopyGetsNew(t *testing.T) {
	// Any change counts: an edited file, a new file or a deleted file.
	for i, edit := range []func(dir string){
		func(dir string) { put(t, dir, map[string]string{"blueprints/techdocs/README.md": "mine"}) },
		func(dir string) { put(t, dir, map[string]string{"blueprints/techdocs/extra.md": "x"}) },
		func(dir string) { _ = os.Remove(filepath.Join(dir, "blueprints", "techdocs", "README.md")) },
	} {
		d := filepath.Join(t.TempDir(), "templates")
		if err := Install(d, builtinsFS("1.0.0")); err != nil {
			t.Fatal(err)
		}
		edit(d)
		res, err := Update(d, builtinsFS("1.1.0"), "techdocs")
		if err != nil || res.InPlace || res.Path != filepath.Join(d, "blueprints", "techdocs.new") {
			t.Errorf("edit %d: %+v %v", i, res, err)
		}
	}
}

// A record from before fingerprints (the user's existing one) counts as edited: nothing is replaced silently.
func TestUpdateWithoutFingerprintGetsNew(t *testing.T) {
	dir := t.TempDir()
	put(t, dir, map[string]string{
		"blueprints/techdocs/blueprint.yml": "id: techdocs\nname: Tech docs\nversion: 1.0.0\n",
		"blueprints/techdocs/README.md":     "# {{title}}\n",
		"blueprints/.zen-templates.yml":     "installed:\n    techdocs: 1.0.0\n",
	})
	res, err := Update(dir, builtinsFS("1.1.0"), "techdocs")
	if err != nil || res.InPlace || res.Path != filepath.Join(dir, "blueprints", "techdocs.new") {
		t.Fatalf("update: %+v %v", res, err)
	}
	if read(t, dir, "blueprints/techdocs/README.md") != "# {{title}}\n" {
		t.Error("the copy changed")
	}
	// The record keeps no fingerprint it never had.
	if strings.Contains(read(t, dir, "blueprints/.zen-templates.yml"), "sha256:") {
		t.Errorf("record:\n%s", read(t, dir, "blueprints/.zen-templates.yml"))
	}
}
