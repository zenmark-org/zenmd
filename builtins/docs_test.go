package builtins

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/zenmark-org/zenmd/engine/plugins"
	"gopkg.in/yaml.v3"
)

// pendingDocs lists built-ins whose reference and demo pages another branch of
// the same wave writes. Remove an id when its pages merge: the test fails for
// an id that is listed here and has its pages, so the entry cannot be left.
var pendingDocs = map[string]string{
	"math": "pages come with feature/myst-syntax",
}

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("no caller")
	}
	return filepath.Join(filepath.Dir(file), "..")
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Errorf("%v", err)
		return ""
	}
	return string(b)
}

func exists(path string) bool { _, err := os.Stat(path); return err == nil }

// TestEveryBuiltinIsDocumented: a built-in ships with a Help summary and
// snippets, a public reference page that names everything it owns, a row in
// builtins.md and, when Zenmark's assembly has it, a demo page with a row in the
// demo's plugin index (the demo shows only what the app's build has).
func TestEveryBuiltinIsDocumented(t *testing.T) {
	root := repoRoot(t)
	manifests := map[string]plugins.Manifest{}
	for _, m := range Plugins().All() {
		manifests[m.ID] = m
	}
	refDir := filepath.Join(root, "docs", "public", "zenmd", "reference")
	builtinsMd := readFile(t, filepath.Join(refDir, "builtins.md"))
	demoDir := filepath.Join(root, "examples", "demo", "plugins")
	demoIndex := readFile(t, filepath.Join(demoDir, "index.md"))

	var asm struct {
		Plugins yaml.Node `yaml:"plugins"`
	}
	if err := yaml.Unmarshal([]byte(readFile(t, filepath.Join(root, "apps", "zenmark", "app-assembly.yml"))), &asm); err != nil {
		t.Fatal(err)
	}
	inApp := map[string]bool{}
	if asm.Plugins.Kind == yaml.ScalarNode && asm.Plugins.Value == "all" {
		for _, id := range PluginIDs {
			inApp[id] = true
		}
	} else {
		var ids []string
		if err := asm.Plugins.Decode(&ids); err != nil {
			t.Fatal(err)
		}
		for _, id := range ids {
			inApp[id] = true
		}
	}

	for _, id := range PluginIDs {
		m, ok := manifests[id]
		if !ok {
			t.Errorf("%s: not in Plugins()", id)
			continue
		}
		if strings.TrimSpace(m.Help.Summary) == "" {
			t.Errorf("%s: Help.Summary is empty", id)
		}
		if len(m.Help.Snippets) == 0 {
			t.Errorf("%s: no Help snippets", id)
		}

		page := filepath.Join(refDir, "plugins", id+".md")
		demo := filepath.Join(demoDir, id+".md")
		if why, pending := pendingDocs[id]; pending {
			if exists(page) || exists(demo) {
				t.Errorf("%s: has its pages; remove it from pendingDocs (%s)", id, why)
			}
		} else {
			if text := readFile(t, page); text != "" {
				for _, n := range append(append([]string{}, m.Blocks...), m.Inlines...) {
					if !strings.Contains(text, n) {
						t.Errorf("%s: reference page does not name %q", id, n)
					}
				}
			}
			if inApp[id] {
				if !exists(demo) {
					t.Errorf("%s: examples/demo/plugins/%s.md is missing", id, id)
				}
				if !strings.Contains(demoIndex, "("+id+".md)") {
					t.Errorf("%s: examples/demo/plugins/index.md does not link %s.md", id, id)
				}
			}
		}

		if !strings.Contains(builtinsMd, "(plugins/"+id+".md)") {
			t.Errorf("%s: builtins.md has no row linking plugins/%s.md", id, id)
		}
	}
}
