package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunMovesOldConfigFolder(t *testing.T) {
	h := home(t)
	dir, err := os.UserConfigDir()
	if err != nil {
		t.Skip(err)
	}
	write(t, filepath.Join(dir, "zen", "config.yml"))
	r := run(t, h, nil, "--version")
	if r.code != 0 {
		t.Fatalf("code %d, stderr %q", r.code, r.stderr)
	}
	if _, err := os.Stat(filepath.Join(dir, "zenmark", "config.yml")); err != nil {
		t.Fatalf("config.yml not moved: %v", err)
	}
	if _, err := os.Lstat(filepath.Join(dir, "zen")); !os.IsNotExist(err) {
		t.Fatal("old folder still there")
	}
	if r.stderr != "" {
		t.Fatalf("stderr %q", r.stderr)
	}
}

func TestRunMigrationFailureWarns(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("root ignores directory permissions")
	}
	h := home(t)
	dir, err := os.UserConfigDir()
	if err != nil {
		t.Skip(err)
	}
	write(t, filepath.Join(dir, "zen", "config.yml"))
	if err := os.Chmod(dir, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(dir, 0o755) })
	r := run(t, h, nil, "--version")
	if r.code != 0 {
		t.Fatalf("code %d, want 0", r.code)
	}
	if n := strings.Count(strings.TrimSpace(r.stderr), "\n") + 1; !strings.Contains(r.stderr, "zenmark") || n != 1 {
		t.Fatalf("want one warning line, got %q", r.stderr)
	}
}
