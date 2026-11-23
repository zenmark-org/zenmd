package scriptplugins

import (
	"os"
	"path/filepath"
	"testing"
)

// redirectStderr makes the helper's stdout descriptor point at its stderr, so
// anything a script prints can't corrupt the protocol pipe.
func TestRedirectStderr(t *testing.T) {
	dir := t.TempDir()
	out, err := os.Create(filepath.Join(dir, "out"))
	if err != nil {
		t.Fatal(err)
	}
	defer out.Close()
	errf, err := os.Create(filepath.Join(dir, "err"))
	if err != nil {
		t.Fatal(err)
	}
	defer errf.Close()

	if err := redirectStderr(errf, out); err != nil {
		t.Fatal(err)
	}
	if _, err := out.WriteString("hello"); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(filepath.Join(dir, "err"))
	if string(got) != "hello" {
		t.Errorf("stderr file holds %q, want it to hold what was written to stdout", got)
	}
	if b, _ := os.ReadFile(filepath.Join(dir, "out")); len(b) != 0 {
		t.Errorf("stdout file holds %q, want nothing", b)
	}
}
