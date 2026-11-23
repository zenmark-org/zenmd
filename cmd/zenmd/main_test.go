package main

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

var zenmdBin string

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "zenmd-bin")
	if err != nil {
		panic(err)
	}
	zenmdBin = filepath.Join(dir, "zenmd")
	b := exec.Command("go", "build", "-ldflags", "-X github.com/zenmark-org/zenmd/internal/cli.Version=0.8.0", "-o", zenmdBin, ".")
	b.Env = append(os.Environ(), "CGO_ENABLED=0")
	if out, err := b.CombinedOutput(); err != nil {
		os.Stderr.Write(out)
		os.RemoveAll(dir)
		os.Exit(1)
	}
	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}

type ran struct {
	code           int
	stdout, stderr string
}

// zenmd runs the built binary in cwd with a throwaway HOME, so it never
// touches the real settings folder.
func zenmd(t *testing.T, home, cwd string, args ...string) ran {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, zenmdBin, args...)
	cmd.Dir = cwd
	cmd.Env = []string{"HOME=" + home, "XDG_CONFIG_HOME=" + filepath.Join(home, ".config"), "PATH=" + os.Getenv("PATH"), "TMPDIR=" + home}
	var out, errOut bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errOut
	start := time.Now()
	err := cmd.Run()
	code := 0
	if err != nil {
		ee, ok := err.(*exec.ExitError)
		if !ok {
			t.Fatalf("%v", err)
		}
		code = ee.ExitCode()
	}
	if d := time.Since(start); d > 50*time.Second {
		t.Errorf("zenmd %v took %v", args, d)
	}
	return ran{code, out.String(), errOut.String()}
}

func firstLine(s string) string { return strings.SplitN(s, "\n", 2)[0] }

func TestCommands(t *testing.T) {
	home := t.TempDir()
	work := t.TempDir()
	// A folder with spaces in its name.
	demo := filepath.Join(work, "my demo")
	if err := os.CopyFS(demo, os.DirFS("../../examples/demo")); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(work, "site out")

	r := zenmd(t, home, work, "export", "site", demo, "-o", out)
	if r.code != 0 {
		t.Fatalf("export site: %+v", r)
	}
	if _, err := os.Stat(filepath.Join(out, "index.html")); err != nil {
		t.Errorf("no index.html: %v\n%s", err, r.stdout)
	}
	if r.stdout == "" {
		t.Error("export site printed nothing")
	}

	for _, args := range [][]string{{"init", "--list"}, {"templates", "list"}, {"blueprint", "list"}} {
		r = zenmd(t, home, work, args...)
		if r.code != 0 || r.stdout == "" {
			t.Errorf("%v: %+v", args, r)
		}
	}
	if r = zenmd(t, home, work, "init", "--list"); !strings.Contains(r.stdout, "techdocs") {
		t.Errorf("init --list: %q", r.stdout)
	}

	r = zenmd(t, home, work, "help")
	if r.code != 0 || !strings.HasPrefix(r.stdout, "zenmd - the Zenmark engine") {
		t.Errorf("help: %+v", r)
	}
	for _, a := range []string{"version", "--version", "-v"} {
		r = zenmd(t, home, work, a)
		if r.code != 0 || r.stdout != "zenmd 0.8.0 (command line)\n" {
			t.Errorf("%s: %+v", a, r)
		}
	}

	// A usage error: exit 2 with the hint.
	r = zenmd(t, home, work, "--nonsense")
	if r.code != 2 || firstLine(r.stderr) != "zenmd: unknown option --nonsense" || !strings.Contains(r.stderr, "zenmd help") {
		t.Errorf("usage error: %+v", r)
	}
	r = zenmd(t, home, work, "export", "frob")
	if r.code != 2 || !strings.Contains(r.stderr, "unknown export target frob") {
		t.Errorf("export frob: %+v", r)
	}
	if _, err := os.Stat(filepath.Join(home, ".config", "zen")); err == nil {
		t.Error("the run made the old settings folder")
	}
}

// The old zen name is only an alias: the notice comes on stderr, the command runs.
func TestAliasNotice(t *testing.T) {
	home, work := t.TempDir(), t.TempDir()
	link := filepath.Join(work, "zen")
	if err := os.Symlink(zenmdBin, link); err != nil {
		t.Skip(err)
	}
	cmd := exec.Command(link, "version")
	cmd.Env = []string{"HOME=" + home, "XDG_CONFIG_HOME=" + filepath.Join(home, ".config")}
	var o, e bytes.Buffer
	cmd.Stdout, cmd.Stderr = &o, &e
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}
	if o.String() != "zenmd 0.8.0 (command line)\n" || !strings.Contains(e.String(), "zen is now zenmd") {
		t.Errorf("stdout %q stderr %q", o.String(), e.String())
	}
}

// What needs the app says so in one line, exits 1 and never waits.
func TestAppOnly(t *testing.T) {
	home, work := t.TempDir(), t.TempDir()
	if err := os.WriteFile(filepath.Join(work, "page.md"), []byte("# Page\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	const noApp = "zenmd: the Zenmark app isn't part of this build; zenmd export site <folder> makes a website\n"
	const noPDF = "zenmd: export pdf needs the Zenmark app on macOS\n"
	for _, c := range []struct {
		args []string
		want string
	}{
		{nil, noApp},
		{[]string{work}, noApp},
		{[]string{filepath.Join(work, "page.md")}, noApp},
		{[]string{"edit", work}, noApp},
		{[]string{"export", "pdf", work}, noPDF},
		{[]string{"export", "pdf"}, noPDF},
	} {
		start := time.Now()
		r := zenmd(t, home, work, c.args...)
		if r.code != 1 || r.stderr != c.want || r.stdout != "" {
			t.Errorf("%v: %+v", c.args, r)
		}
		if d := time.Since(start); d > 2*time.Second {
			t.Errorf("%v took %v", c.args, d)
		}
	}
}

// templates --dir and init --dir print the folder, under the fake home's XDG config.
func TestDirsPrint(t *testing.T) {
	home, work := t.TempDir(), t.TempDir()
	cfg := filepath.Join(home, ".config", "zenmark")
	r := zenmd(t, home, work, "templates", "--dir")
	if runtimeIsLinux() && (r.code != 0 || r.stdout != filepath.Join(cfg, "templates")+"\n") {
		t.Errorf("templates --dir: %+v", r)
	}
	if r.code != 0 || !strings.HasSuffix(r.stdout, "templates\n") {
		t.Errorf("templates --dir: %+v", r)
	}
	r = zenmd(t, home, work, "init", "--dir")
	if r.code != 0 || !strings.HasSuffix(r.stdout, filepath.Join("templates", "blueprints")+"\n") {
		t.Errorf("init --dir: %+v", r)
	}
}

func runtimeIsLinux() bool { return runtime.GOOS == "linux" }
