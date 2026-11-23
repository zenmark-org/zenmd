package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// -out writes the generated file into another folder, in that folder's
// package; without it the file goes into the app folder.
func TestOutFlag(t *testing.T) {
	dir := appDir(t, full)
	out := filepath.Join(t.TempDir(), "cliset")
	if err := os.MkdirAll(out, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(out, "x.go"), []byte("package cliset\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if code, _, errOut := gen("-out", out, dir); code != 0 {
		t.Fatalf("code %d: %s", code, errOut)
	}
	if _, err := os.Stat(filepath.Join(dir, "assembly_gen.go")); err == nil {
		t.Error("-out: the app folder must stay untouched")
	}
	src, err := os.ReadFile(filepath.Join(out, "assembly_gen.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(src), "// Code generated") || !strings.Contains(string(src), "package cliset\n") {
		t.Errorf("generated file:\n%.200s", src)
	}
	// Without -out: the app folder.
	if code, _, errOut := gen(dir); code != 0 {
		t.Fatalf("code %d: %s", code, errOut)
	}
	if _, err := os.Stat(filepath.Join(dir, "assembly_gen.go")); err != nil {
		t.Error(err)
	}
	// An -out folder that is not there is an error, nothing is made.
	if code, _, _ := gen("-out", filepath.Join(t.TempDir(), "nope"), dir); code == 0 {
		t.Error("a missing -out folder should fail")
	}
	// An app with a cliset folder generates into it when -out is not given.
	dir = appDir(t, full)
	if err := os.MkdirAll(filepath.Join(dir, "cliset"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "cliset", "x.go"), []byte("package cliset\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if code, _, errOut := gen(dir); code != 0 {
		t.Fatalf("code %d: %s", code, errOut)
	}
	if _, err := os.Stat(filepath.Join(dir, "cliset", "assembly_gen.go")); err != nil {
		t.Error(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "assembly_gen.go")); err == nil {
		t.Error("the app folder must stay untouched when it has a cliset folder")
	}
}
