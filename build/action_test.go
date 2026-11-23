package build_test

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// The docs CI action (specs/packaging/docs-ci): action.yml, install.sh and
// run.sh. No test downloads: install.sh reads a file:// folder through
// ZENMD_TEST_RELEASE_BASE, which the action never passes on.

type actionFile struct {
	Inputs  map[string]struct{ Default any } `yaml:"inputs"`
	Outputs map[string]struct {
		Value string `yaml:"value"`
	} `yaml:"outputs"`
	Runs struct {
		Using string `yaml:"using"`
		Steps []struct {
			ID    string            `yaml:"id"`
			Run   string            `yaml:"run"`
			Shell string            `yaml:"shell"`
			Env   map[string]string `yaml:"env"`
			Uses  string            `yaml:"uses"`
		} `yaml:"steps"`
	} `yaml:"runs"`
}

func readAction(t *testing.T) actionFile {
	t.Helper()
	b, err := os.ReadFile("../action.yml")
	if err != nil {
		t.Fatal(err)
	}
	var a actionFile
	if err := yaml.Unmarshal(b, &a); err != nil {
		t.Fatal(err)
	}
	return a
}

func TestActionMetadata(t *testing.T) {
	a := readAction(t)
	if a.Runs.Using != "composite" {
		t.Fatalf("runs.using = %q, want composite", a.Runs.Using)
	}
	want := map[string]string{"folder": ".", "output": "_site", "version": "", "hosted": "true",
		"drafts": "false", "scheme": "", "clean": "true", "print": "true", "check": "false"}
	for k, d := range want {
		in, ok := a.Inputs[k]
		if !ok {
			t.Errorf("input %q missing", k)
			continue
		}
		if fmt.Sprint(in.Default) != d {
			t.Errorf("input %s default = %v, want %q", k, in.Default, d)
		}
	}
	for _, k := range []string{"output", "pages", "warnings"} {
		if !strings.Contains(a.Outputs[k].Value, "steps.") {
			t.Errorf("output %q has no step value: %q", k, a.Outputs[k].Value)
		}
	}
	if len(a.Runs.Steps) < 2 {
		t.Fatalf("want an install step and a run step, got %d steps", len(a.Runs.Steps))
	}
	var scripts string
	for i, s := range a.Runs.Steps {
		if s.Run == "" {
			continue
		}
		scripts += s.Run
		if s.Shell == "" {
			t.Errorf("step %d has no shell", i)
		}
		// Inputs reach the scripts as environment variables only.
		if strings.Contains(s.Run, "${{") {
			t.Errorf("step %d run text has an expression: %q", i, s.Run)
		}
		for k := range s.Env {
			if strings.HasPrefix(k, "ZENMD_TEST") {
				t.Errorf("step %d passes the test override %s", i, k)
			}
		}
	}
	if !strings.Contains(scripts, "install.sh") || !strings.Contains(scripts, "run.sh") {
		t.Errorf("steps do not call install.sh and run.sh: %q", scripts)
	}
	// The version comes from the action's ref.
	found := false
	for _, s := range a.Runs.Steps {
		for _, v := range s.Env {
			if strings.Contains(v, "github.action_ref") {
				found = true
			}
		}
	}
	if !found {
		t.Error("no step gets github.action_ref")
	}
}

// ---- install.sh ----

const testVersion = "0.8.0"

// fixture writes a release folder: the archive for arch and SHA256SUMS.
// badSum writes a wrong hash.
func fixture(t *testing.T, arch string, badSum bool) string {
	t.Helper()
	dir := t.TempDir()
	name := fmt.Sprintf("zenmd_%s_linux_%s", testVersion, arch)
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	add := func(n, body string, mode int64) {
		if err := tw.WriteHeader(&tar.Header{Name: name + "/" + n, Mode: mode, Size: int64(len(body)), Typeflag: tar.TypeReg}); err != nil {
			t.Fatal(err)
		}
		tw.Write([]byte(body))
	}
	add("zenmd", "#!/bin/sh\necho zenmd-fixture-"+arch+"\n", 0o755)
	add("LICENSE", "license", 0o644)
	add("README.md", "readme", 0o644)
	tw.Close()
	gz.Close()
	rel := filepath.Join(dir, "v"+testVersion)
	if err := os.MkdirAll(rel, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(rel, name+".tar.gz"), buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	sum := fmt.Sprintf("%x", sha256.Sum256(buf.Bytes()))
	if badSum {
		sum = strings.Repeat("0", 64)
	}
	// SHA256SUMS lines: <hash>  <archive>, with another archive's line too.
	sums := fmt.Sprintf("%s  %s.tar.gz\n%s  zenmd_%s_linux_other.tar.gz\n", sum, name, strings.Repeat("1", 64), testVersion)
	if err := os.WriteFile(filepath.Join(rel, "SHA256SUMS"), []byte(sums), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func runScript(t *testing.T, script string, env map[string]string, dir string) (string, string, int) {
	t.Helper()
	home := t.TempDir()
	cmd := exec.Command("sh", script)
	cmd.Dir = dir
	cmd.Env = []string{"PATH=" + os.Getenv("PATH"), "HOME=" + home, "XDG_CONFIG_HOME=" + filepath.Join(home, ".config")}
	for k, v := range env {
		cmd.Env = append(cmd.Env, k+"="+v)
	}
	var so, se bytes.Buffer
	cmd.Stdout, cmd.Stderr = &so, &se
	err := cmd.Run()
	code := 0
	if ee, ok := err.(*exec.ExitError); ok {
		code = ee.ExitCode()
	} else if err != nil {
		t.Fatal(err)
	}
	return so.String(), se.String(), code
}

func abs(t *testing.T, p string) string {
	t.Helper()
	a, err := filepath.Abs(p)
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func installEnv(t *testing.T, base string, extra map[string]string) (map[string]string, string, string) {
	t.Helper()
	runner := filepath.Join(t.TempDir(), "runner temp") // a space on purpose
	if err := os.MkdirAll(runner, 0o755); err != nil {
		t.Fatal(err)
	}
	ghPath := filepath.Join(t.TempDir(), "github_path")
	env := map[string]string{
		"RUNNER_TEMP": runner, "GITHUB_PATH": ghPath,
		"ZENMD_TEST_RELEASE_BASE": "file://" + base,
		"ZENMD_TEST_OS":           "Linux", "ZENMD_TEST_ARCH": "x86_64",
		"ZM_VERSION": testVersion,
	}
	for k, v := range extra {
		env[k] = v
	}
	return env, runner, ghPath
}

func TestInstallScript(t *testing.T) {
	script := abs(t, "../.github/actions/install.sh")
	src, err := os.ReadFile(script)
	if err != nil {
		t.Fatal(err)
	}
	// Only the release URL, https only, no redirect to another scheme.
	for _, w := range []string{"https://github.com/zenmark-org/zenmd/releases/download", "--proto =https", "--proto-redir =https", "--fail"} {
		if !strings.Contains(string(src), w) {
			t.Errorf("install.sh lacks %q", w)
		}
	}
	if n := strings.Count(string(src), "https://"); n != 1 {
		t.Errorf("install.sh names %d https URLs, want only the release URL", n)
	}

	t.Run("installs and checks", func(t *testing.T) {
		for _, c := range []struct{ uname, arch string }{{"x86_64", "amd64"}, {"aarch64", "arm64"}, {"arm64", "arm64"}} {
			base := fixture(t, c.arch, false)
			env, runner, ghPath := installEnv(t, base, map[string]string{"ZENMD_TEST_ARCH": c.uname})
			_, se, code := runScript(t, script, env, t.TempDir())
			if code != 0 {
				t.Fatalf("%s: exit %d: %s", c.uname, code, se)
			}
			bin := filepath.Join(runner, "zenmd", "bin")
			out, err := exec.Command(filepath.Join(bin, "zenmd")).Output()
			if err != nil || !strings.Contains(string(out), "zenmd-fixture-"+c.arch) {
				t.Errorf("%s: installed binary = %q, %v", c.uname, out, err)
			}
			gp, _ := os.ReadFile(ghPath)
			if strings.TrimSpace(string(gp)) != bin {
				t.Errorf("GITHUB_PATH = %q, want %q", gp, bin)
			}
		}
	})
	t.Run("bad checksum stops", func(t *testing.T) {
		env, runner, ghPath := installEnv(t, fixture(t, "amd64", true), nil)
		_, se, code := runScript(t, script, env, t.TempDir())
		if code != 1 || !strings.Contains(se, "checksum") {
			t.Fatalf("exit %d, stderr %q", code, se)
		}
		if _, err := os.Stat(filepath.Join(runner, "zenmd", "bin", "zenmd")); err == nil {
			t.Error("zenmd was installed after a bad checksum")
		}
		if _, err := os.Stat(ghPath); err == nil {
			t.Error("PATH was changed after a bad checksum")
		}
	})
	t.Run("not on the sums list", func(t *testing.T) {
		base := fixture(t, "amd64", false)
		if err := os.WriteFile(filepath.Join(base, "v"+testVersion, "SHA256SUMS"), []byte(strings.Repeat("1", 64)+"  other.tar.gz\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		env, _, _ := installEnv(t, base, nil)
		if _, se, code := runScript(t, script, env, t.TempDir()); code != 1 || !strings.Contains(se, "SHA256SUMS") {
			t.Fatalf("exit %d, stderr %q", code, se)
		}
	})
	t.Run("exact versions only", func(t *testing.T) {
		for _, v := range []string{"latest", "0.8", "v0", "0.8.x", "../0.8.0", "0.8.0/../../x", "0.8.0 "} {
			env, _, _ := installEnv(t, fixture(t, "amd64", false), map[string]string{"ZM_VERSION": v})
			_, se, code := runScript(t, script, env, t.TempDir())
			if code != 1 || !strings.Contains(se, "exact version") {
				t.Errorf("version %q: exit %d, stderr %q", v, code, se)
			}
		}
	})
	t.Run("a leading v is fine", func(t *testing.T) {
		env, _, _ := installEnv(t, fixture(t, "amd64", false), map[string]string{"ZM_VERSION": "v" + testVersion})
		if _, se, code := runScript(t, script, env, t.TempDir()); code != 0 {
			t.Fatalf("exit %d: %s", code, se)
		}
	})
	t.Run("version from the action ref", func(t *testing.T) {
		env, _, _ := installEnv(t, fixture(t, "amd64", false), map[string]string{"ZM_VERSION": "", "ACTION_REF": "v" + testVersion})
		if _, se, code := runScript(t, script, env, t.TempDir()); code != 0 {
			t.Fatalf("exit %d: %s", code, se)
		}
		env, _, _ = installEnv(t, fixture(t, "amd64", false), map[string]string{"ZM_VERSION": "", "ACTION_REF": "v0"})
		if _, se, code := runScript(t, script, env, t.TempDir()); code != 1 || !strings.Contains(se, "exact version") {
			t.Fatalf("v0: exit %d: %s", code, se)
		}
	})
	t.Run("other runners", func(t *testing.T) {
		env, _, _ := installEnv(t, fixture(t, "amd64", false), map[string]string{"ZENMD_TEST_OS": "Darwin"})
		_, se, code := runScript(t, script, env, t.TempDir())
		if code != 1 || strings.TrimSpace(se) != "zenmd's action runs on Linux runners" {
			t.Fatalf("exit %d, stderr %q", code, se)
		}
		env, _, _ = installEnv(t, fixture(t, "amd64", false), map[string]string{"ZENMD_TEST_ARCH": "riscv64"})
		if _, se, code := runScript(t, script, env, t.TempDir()); code != 1 || !strings.Contains(se, "riscv64") {
			t.Fatalf("arch: exit %d, stderr %q", code, se)
		}
	})
	t.Run("no Linux archives before 0.8.0", func(t *testing.T) {
		env, _, _ := installEnv(t, t.TempDir(), map[string]string{"ZM_VERSION": "0.7.0"})
		_, se, code := runScript(t, script, env, t.TempDir())
		if code != 1 || !strings.Contains(se, "0.8.0") || strings.Count(strings.TrimSpace(se), "\n") != 0 {
			t.Fatalf("exit %d, stderr %q (want one line naming 0.8.0)", code, se)
		}
	})
}

// ---- run.sh ----

// fakeZenmd writes a zenmd on PATH. FAKE_* variables steer it; it records its
// arguments, one per line, in $FAKE_ARGS (appended).
func fakeZenmd(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	body := `#!/bin/sh
{ printf 'CMD'; for a in "$@"; do printf '\n%s' "$a"; done; printf '\n'; } >> "$FAKE_ARGS"
case "$1" in
help)
  echo "zenmd - the Zenmark engine"
  echo "  zenmd export site [folder]"
  if [ "$FAKE_HAS_CHECK" = 1 ]; then echo "  zenmd check [folder]"; fi
  ;;
check)
  printf '%s' "$FAKE_CHECK_ERR" >&2
  exit "${FAKE_CHECK_RC:-0}"
  ;;
export)
  printf '%s' "$FAKE_EXPORT_ERR" >&2
  if [ "${FAKE_EXPORT_RC:-0}" != 0 ]; then exit "$FAKE_EXPORT_RC"; fi
  echo "Built ${FAKE_PAGES:-3} pages and 7 files into out"
  ;;
esac
`
	if err := os.WriteFile(filepath.Join(dir, "zenmd"), []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	return dir
}

type runResult struct {
	stdout, stderr string
	code           int
	outputs        map[string]string
	calls          [][]string // each zenmd call, "CMD" first removed
}

func runRun(t *testing.T, env map[string]string) runResult {
	t.Helper()
	script := abs(t, "../.github/actions/run.sh")
	fake := fakeZenmd(t)
	work := t.TempDir()
	argsFile := filepath.Join(work, "args.txt")
	ghOut := filepath.Join(work, "github_output")
	e := map[string]string{
		"PATH": fake + ":" + os.Getenv("PATH"), "FAKE_ARGS": argsFile, "GITHUB_OUTPUT": ghOut,
		"ZM_FOLDER": ".", "ZM_OUTPUT": "_site", "ZM_HOSTED": "true", "ZM_DRAFTS": "false",
		"ZM_SCHEME": "", "ZM_CLEAN": "true", "ZM_PRINT": "true", "ZM_CHECK": "false",
	}
	for k, v := range env {
		e[k] = v
	}
	home := t.TempDir()
	cmd := exec.Command("sh", script)
	cmd.Dir = work
	cmd.Env = []string{"HOME=" + home, "XDG_CONFIG_HOME=" + filepath.Join(home, ".config")}
	for k, v := range e {
		cmd.Env = append(cmd.Env, k+"="+v)
	}
	var so, se bytes.Buffer
	cmd.Stdout, cmd.Stderr = &so, &se
	code := 0
	if err := cmd.Run(); err != nil {
		ee, ok := err.(*exec.ExitError)
		if !ok {
			t.Fatal(err)
		}
		code = ee.ExitCode()
	}
	r := runResult{stdout: so.String(), stderr: se.String(), code: code, outputs: map[string]string{}}
	if b, err := os.ReadFile(ghOut); err == nil {
		for _, l := range strings.Split(string(b), "\n") {
			if k, v, ok := strings.Cut(l, "="); ok {
				r.outputs[k] = v
			}
		}
	}
	if b, err := os.ReadFile(argsFile); err == nil {
		for _, c := range strings.Split(strings.TrimSpace(string(b)), "\nCMD") {
			c = strings.TrimPrefix(c, "CMD")
			c = strings.TrimPrefix(c, "\n")
			r.calls = append(r.calls, strings.Split(c, "\n"))
		}
	}
	return r
}

func (r runResult) call(sub string) []string {
	for _, c := range r.calls {
		if len(c) > 0 && c[0] == sub {
			return c
		}
	}
	return nil
}

func has(args []string, want ...string) bool {
	return strings.Contains("\x00"+strings.Join(args, "\x00")+"\x00", "\x00"+strings.Join(want, "\x00")+"\x00")
}

func TestRunScript(t *testing.T) {
	t.Run("defaults", func(t *testing.T) {
		r := runRun(t, nil)
		if r.code != 0 {
			t.Fatalf("exit %d: %s", r.code, r.stderr)
		}
		ex := r.call("export")
		want := []string{"export", "site", ".", "-o", "_site", "--hosted", "--clean"}
		if strings.Join(ex, " ") != strings.Join(want, " ") {
			t.Errorf("export args = %q, want %q", ex, want)
		}
		if r.call("check") != nil {
			t.Error("check ran with check=false")
		}
		if r.outputs["output"] != "_site" || r.outputs["pages"] != "3" || r.outputs["warnings"] != "0" {
			t.Errorf("outputs = %v", r.outputs)
		}
	})
	t.Run("every input is a flag", func(t *testing.T) {
		r := runRun(t, map[string]string{"ZM_HOSTED": "false", "ZM_DRAFTS": "true", "ZM_SCHEME": "dark", "ZM_CLEAN": "false", "ZM_PRINT": "false", "ZM_FOLDER": "docs", "ZM_OUTPUT": "out"})
		ex := r.call("export")
		for _, w := range []string{"--portable", "--drafts", "--no-print"} {
			if !has(ex, w) {
				t.Errorf("missing %s in %q", w, ex)
			}
		}
		if !has(ex, "--scheme", "dark") || !has(ex, "-o", "out") || has(ex, "--clean") || has(ex, "--hosted") {
			t.Errorf("export args = %q", ex)
		}
	})
	t.Run("warnings become annotations", func(t *testing.T) {
		stderr := "guide/setup.md: heading level jumps\nsite: no logo, 100% sure, a:b,c\nguide/x.md: line 12: unknown plugin \"zz\"\n"
		r := runRun(t, map[string]string{"FAKE_EXPORT_ERR": stderr, "FAKE_PAGES": "9", "ZM_FOLDER": "docs"})
		if r.code != 0 {
			t.Fatalf("exit %d: %s", r.code, r.stderr)
		}
		for _, w := range []string{
			"::warning file=docs/guide/setup.md,title=zenmd::heading level jumps",
			"::warning title=zenmd::site: no logo, 100%25 sure, a:b,c",
			"::warning file=docs/guide/x.md,line=12,title=zenmd::unknown plugin \"zz\"",
		} {
			if !strings.Contains(r.stdout, w+"\n") {
				t.Errorf("missing annotation %q in\n%s", w, r.stdout)
			}
		}
		if r.outputs["warnings"] != "3" || r.outputs["pages"] != "9" || r.outputs["output"] != "_site" {
			t.Errorf("outputs = %v", r.outputs)
		}
	})
	t.Run("check missing", func(t *testing.T) {
		r := runRun(t, map[string]string{"ZM_CHECK": "true"})
		if r.code != 1 || r.call("check") != nil || r.call("export") != nil {
			t.Fatalf("exit %d, calls %v", r.code, r.calls)
		}
		if strings.Count(strings.TrimSpace(r.stderr), "\n") != 0 || !strings.Contains(r.stderr, "0.8.0") {
			t.Errorf("stderr = %q, want one line naming 0.8.0", r.stderr)
		}
	})
	t.Run("check passes with warnings", func(t *testing.T) {
		r := runRun(t, map[string]string{"ZM_CHECK": "true", "FAKE_HAS_CHECK": "1", "FAKE_CHECK_ERR": "a.md: broken link\n", "ZM_FOLDER": "d"})
		if r.code != 0 || !strings.Contains(r.stdout, "::warning file=d/a.md,title=zenmd::broken link\n") {
			t.Fatalf("exit %d\n%s\n%s", r.code, r.stdout, r.stderr)
		}
		c := r.call("check")
		if len(c) < 2 || c[1] != "d" {
			t.Errorf("check args = %q", c)
		}
		if r.call("export") == nil {
			t.Error("export did not run after a passing check")
		}
	})
	t.Run("check fails the job", func(t *testing.T) {
		r := runRun(t, map[string]string{"ZM_CHECK": "true", "FAKE_HAS_CHECK": "1", "FAKE_CHECK_RC": "1", "FAKE_CHECK_ERR": "a.md: 2 errors\n"})
		if r.code != 1 || r.call("export") != nil {
			t.Fatalf("exit %d, calls %v", r.code, r.calls)
		}
	})
	t.Run("export failure passes through", func(t *testing.T) {
		r := runRun(t, map[string]string{"FAKE_EXPORT_RC": "1", "FAKE_EXPORT_ERR": "zenmd: nothing to build\n"})
		if r.code != 1 || !strings.Contains(r.stderr, "zenmd: nothing to build") || strings.Contains(r.stdout, "::warning") {
			t.Fatalf("exit %d\nstdout %q\nstderr %q", r.code, r.stdout, r.stderr)
		}
	})
	t.Run("injection", func(t *testing.T) {
		for _, name := range []string{`a"; touch PWNED #`, "my docs", "$(touch PWNED)", "`touch PWNED`"} {
			r := runRun(t, map[string]string{"ZM_FOLDER": name, "ZM_OUTPUT": name + "/out", "ZM_SCHEME": name})
			ex := r.call("export")
			if len(ex) < 3 || ex[2] != name || !has(ex, "-o", name+"/out") {
				t.Errorf("%q: export args = %q", name, ex)
			}
			if r.code != 0 {
				t.Errorf("%q: exit %d: %s", name, r.code, r.stderr)
			}
		}
	})
	t.Run("annotations carry only the warning text", func(t *testing.T) {
		r := runRun(t, map[string]string{"FAKE_EXPORT_ERR": "p.md: bad\n"})
		for _, l := range strings.Split(r.stdout, "\n") {
			if l != "" && !strings.HasPrefix(l, "::") && !strings.HasPrefix(l, "Built ") {
				t.Errorf("unexpected stdout line %q", l)
			}
		}
	})
}

// ---- inputs against export-options.json ----

// Site options the action leaves out on purpose.
var leftOut = map[string]string{
	"--save-defaults": "writes the folder's .zenmd.yml; not for CI",
	"--dry-run":       "goes with --save-defaults",
}

// The site flag each input drives.
var inputFlag = map[string]string{
	"output": "--out", "hosted": "--portable", "drafts": "--drafts",
	"scheme": "--scheme", "clean": "--clean", "print": "--print",
}

// Inputs that are not site options.
var otherInputs = map[string]bool{"folder": true, "version": true, "check": true}

func TestInputsMatchExportOptions(t *testing.T) {
	b, err := os.ReadFile("../apps/zenmark/export-options.json")
	if err != nil {
		t.Fatal(err)
	}
	var opts struct {
		Site []struct {
			Flag string `json:"flag"`
		} `json:"site"`
	}
	if err := json.Unmarshal(b, &opts); err != nil {
		t.Fatal(err)
	}
	a := readAction(t)
	covered := map[string]bool{}
	for in := range a.Inputs {
		if otherInputs[in] {
			continue
		}
		f, ok := inputFlag[in]
		if !ok {
			t.Errorf("input %q is not a zenmd export site flag", in)
			continue
		}
		covered[f] = true
	}
	var missing []string
	for _, o := range opts.Site {
		if !covered[o.Flag] && leftOut[o.Flag] == "" {
			missing = append(missing, o.Flag)
		}
	}
	sort.Strings(missing)
	if len(missing) > 0 {
		t.Errorf("site options with no action input and not left out on purpose: %v", missing)
	}
	for _, f := range inputFlag {
		found := false
		for _, o := range opts.Site {
			if o.Flag == f {
				found = true
			}
		}
		if !found {
			t.Errorf("input maps to %s, which export-options.json no longer has", f)
		}
	}
}

// ---- example workflows ----

type exWorkflow struct {
	On          map[string]any    `yaml:"on"`
	Permissions map[string]string `yaml:"permissions"`
	Jobs        map[string]struct {
		If          string            `yaml:"if"`
		Permissions map[string]string `yaml:"permissions"`
		Steps       []struct {
			Uses string         `yaml:"uses"`
			With map[string]any `yaml:"with"`
			If   string         `yaml:"if"`
		} `yaml:"steps"`
	} `yaml:"jobs"`
}

func TestExampleWorkflows(t *testing.T) {
	pinned := regexp.MustCompile(`^[\w.-]+/[\w./-]+@v\d+(\.\d+){0,2}$`)
	load := func(name string) exWorkflow {
		b, err := os.ReadFile("../examples/ci/" + name)
		if err != nil {
			t.Fatal(err)
		}
		var w exWorkflow
		if err := yaml.Unmarshal(b, &w); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		for jn, j := range w.Jobs {
			for _, s := range j.Steps {
				if s.Uses != "" && !pinned.MatchString(s.Uses) {
					t.Errorf("%s job %s: %q is not pinned to a version", name, jn, s.Uses)
				}
			}
		}
		return w
	}
	usesAction := func(w exWorkflow) bool {
		for _, j := range w.Jobs {
			for _, s := range j.Steps {
				if strings.HasPrefix(s.Uses, "zenmark-org/zenmd@v") {
					return true
				}
			}
		}
		return false
	}

	check := load("check-docs.yml")
	if _, ok := check.On["pull_request"]; !ok || !usesAction(check) {
		t.Errorf("check-docs.yml: want a pull_request trigger and the action: %v", check.On)
	}
	if _, ok := check.On["push"]; ok {
		t.Error("check-docs.yml runs on push")
	}

	pub := load("publish-pages.yml")
	if !usesAction(pub) {
		t.Error("publish-pages.yml does not use the action")
	}
	if _, ok := pub.On["pull_request"]; ok {
		t.Error("publish-pages.yml runs for pull requests: a fork could publish")
	}
	push, ok := pub.On["push"].(map[string]any)
	if !ok || fmt.Sprint(push["branches"]) != "[main]" {
		t.Errorf("publish-pages.yml on.push = %v, want branches: [main]", pub.On["push"])
	}
	if len(pub.Permissions) > 0 && (pub.Permissions["pages"] != "" || pub.Permissions["id-token"] != "") {
		t.Errorf("publish-pages.yml: pages permission at the top level: %v", pub.Permissions)
	}
	var upload, deploy bool
	for jn, j := range pub.Jobs {
		for _, s := range j.Steps {
			if strings.HasPrefix(s.Uses, "actions/upload-pages-artifact@") {
				upload = true
				if j.Permissions["pages"] == "write" {
					t.Errorf("job %s uploads and has pages: write; only the deploy job may", jn)
				}
			}
			if strings.HasPrefix(s.Uses, "actions/deploy-pages@") {
				deploy = true
				if j.Permissions["pages"] != "write" {
					t.Errorf("deploy job %s lacks pages: write", jn)
				}
			}
		}
	}
	if !upload || !deploy {
		t.Errorf("publish-pages.yml: upload-pages-artifact %v, deploy-pages %v", upload, deploy)
	}
}
