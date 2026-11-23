package app

import (
	"bytes"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/zenmark-org/zenmd/apps/zenmark/cliset"
	"github.com/zenmark-org/zenmd/internal/cli"
)

// The command-line-only zenmd and the app's zenmd write the same site and
// print the same lines for the same folder: both are built from the one
// generated assembly and the same session.WriteSite.
func TestCLIBuildMatchesApp(t *testing.T) {
	// The package's tests run on the full build; the CLI uses the generated assembly, so compare on that.
	oldP, oldL, oldD := builtinPlugins, builtinLayouts, builtinDialects
	builtinPlugins, builtinLayouts, builtinDialects = generatedPlugins, generatedLayouts, generatedDialects
	t.Cleanup(func() { builtinPlugins, builtinLayouts, builtinDialects = oldP, oldL, oldD })
	h := t.TempDir()
	t.Setenv("HOME", h)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(h, ".config"))

	src := filepath.Join(h, "my demo")
	if err := os.CopyFS(src, os.DirFS("../../examples/demo")); err != nil {
		t.Fatal(err)
	}
	build := func(f func(cli.Build, io.Writer, io.Writer) error, out string) (string, string) {
		var so, se bytes.Buffer
		if err := f(cli.Build{Dir: src, Out: out}, &so, &se); err != nil {
			t.Fatalf("%v\n%s", err, se.String())
		}
		return so.String(), se.String()
	}
	outA, outC := filepath.Join(h, "a"), filepath.Join(h, "c")
	soA, seA := build(BuildSite, outA)
	soC, seC := build(cliset.BuildSite, outC)
	// The summary names the output folder; compare with it taken out.
	if got, want := bytes.ReplaceAll([]byte(soC), []byte(outC), []byte("OUT")), bytes.ReplaceAll([]byte(soA), []byte(outA), []byte("OUT")); !bytes.Equal(got, want) {
		t.Errorf("stdout differs:\napp: %s\ncli: %s", soA, soC)
	}
	if got, want := bytes.ReplaceAll([]byte(seC), []byte(outC), []byte("OUT")), bytes.ReplaceAll([]byte(seA), []byte(outA), []byte("OUT")); !bytes.Equal(got, want) {
		t.Errorf("stderr differs:\napp: %s\ncli: %s", seA, seC)
	}
	files := func(root string) map[string][]byte {
		m := map[string][]byte{}
		err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return err
			}
			b, err := os.ReadFile(p)
			rel, _ := filepath.Rel(root, p)
			m[rel] = b
			return err
		})
		if err != nil {
			t.Fatal(err)
		}
		return m
	}
	a, c := files(outA), files(outC)
	if len(a) == 0 {
		t.Fatal("the app wrote nothing")
	}
	if len(a) != len(c) {
		t.Errorf("%d files from the app, %d from the CLI", len(a), len(c))
	}
	for name, want := range a {
		got, ok := c[name]
		if !ok {
			t.Errorf("%s: missing from the CLI's site", name)
		} else if !bytes.Equal(got, want) {
			t.Errorf("%s differs", name)
		}
	}
}
