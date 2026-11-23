package config

import (
	"os"
	"path/filepath"
	"testing"
)

// fakeConfigDir points HOME and XDG_CONFIG_HOME into a temp dir and returns
// os.UserConfigDir().
func fakeConfigDir(t *testing.T) string {
	t.Helper()
	h := t.TempDir()
	t.Setenv("HOME", h)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(h, ".config"))
	dir, err := os.UserConfigDir()
	if err != nil {
		t.Skip("no user config dir:", err)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestMigrateMoves(t *testing.T) {
	dir := fakeConfigDir(t)
	write(t, filepath.Join(dir, "zen", "config.yml"), "theme: x\n")
	write(t, filepath.Join(dir, "zen", "plugins", "p", "a.txt"), "p")
	moved, err := Migrate()
	if err != nil || !moved {
		t.Fatalf("Migrate() = %v, %v; want true, nil", moved, err)
	}
	if b, err := os.ReadFile(filepath.Join(dir, "zenmark", "config.yml")); err != nil || string(b) != "theme: x\n" {
		t.Fatalf("config.yml: %q, %v", b, err)
	}
	if _, err := os.Stat(filepath.Join(dir, "zenmark", "plugins", "p", "a.txt")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(filepath.Join(dir, "zen")); !os.IsNotExist(err) {
		t.Fatalf("old folder still there: %v", err)
	}
}

func TestMigrateTrustFileCounts(t *testing.T) {
	dir := fakeConfigDir(t)
	write(t, filepath.Join(dir, "zen", "trusted-plugins.yml"), "a: b\n")
	if moved, err := Migrate(); err != nil || !moved {
		t.Fatalf("Migrate() = %v, %v", moved, err)
	}
}

func TestMigrateBothExist(t *testing.T) {
	dir := fakeConfigDir(t)
	write(t, filepath.Join(dir, "zen", "config.yml"), "old")
	write(t, filepath.Join(dir, "zenmark", "config.yml"), "new")
	moved, err := Migrate()
	if err != nil || moved {
		t.Fatalf("Migrate() = %v, %v; want false, nil", moved, err)
	}
	for f, want := range map[string]string{"zen": "old", "zenmark": "new"} {
		if b, _ := os.ReadFile(filepath.Join(dir, f, "config.yml")); string(b) != want {
			t.Fatalf("%s/config.yml = %q", f, b)
		}
	}
}

func TestMigrateForeignFolder(t *testing.T) {
	dir := fakeConfigDir(t)
	write(t, filepath.Join(dir, "zen", "Profiles", "x", "prefs.js"), "x")
	moved, err := Migrate()
	if err != nil || moved {
		t.Fatalf("Migrate() = %v, %v; want false, nil", moved, err)
	}
	if _, err := os.Stat(filepath.Join(dir, "zen", "Profiles", "x", "prefs.js")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(filepath.Join(dir, "zenmark")); !os.IsNotExist(err) {
		t.Fatal("zenmark created")
	}
}

func TestMigrateNeither(t *testing.T) {
	dir := fakeConfigDir(t)
	moved, err := Migrate()
	if err != nil || moved {
		t.Fatalf("Migrate() = %v, %v; want false, nil", moved, err)
	}
	for _, n := range []string{"zen", "zenmark"} {
		if _, err := os.Lstat(filepath.Join(dir, n)); !os.IsNotExist(err) {
			t.Fatalf("%s created", n)
		}
	}
}

func TestMigrateNotADir(t *testing.T) {
	t.Run("symlink", func(t *testing.T) {
		dir := fakeConfigDir(t)
		target := filepath.Join(t.TempDir(), "real")
		write(t, filepath.Join(target, "config.yml"), "x")
		if err := os.Symlink(target, filepath.Join(dir, "zen")); err != nil {
			t.Skip("no symlinks:", err)
		}
		moved, err := Migrate()
		if err != nil || moved {
			t.Fatalf("Migrate() = %v, %v", moved, err)
		}
		if fi, err := os.Lstat(filepath.Join(dir, "zen")); err != nil || fi.Mode()&os.ModeSymlink == 0 {
			t.Fatalf("symlink changed: %v", err)
		}
		if _, err := os.Lstat(filepath.Join(dir, "zenmark")); !os.IsNotExist(err) {
			t.Fatal("zenmark created")
		}
	})
	t.Run("file", func(t *testing.T) {
		dir := fakeConfigDir(t)
		write(t, filepath.Join(dir, "zen"), "just a file")
		moved, err := Migrate()
		if err != nil || moved {
			t.Fatalf("Migrate() = %v, %v", moved, err)
		}
		if b, _ := os.ReadFile(filepath.Join(dir, "zen")); string(b) != "just a file" {
			t.Fatal("file changed")
		}
	})
}
