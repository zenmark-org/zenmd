package dropdown

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/zenmark-org/zenmd/builtins/dialects/myst"
	"github.com/zenmark-org/zenmd/engine/config"
	"github.com/zenmark-org/zenmd/engine/dialects"
	"github.com/zenmark-org/zenmd/engine/plugins"
	"github.com/zenmark-org/zenmd/engine/render"
)

func plain(t *testing.T) *render.Renderer {
	t.Helper()
	reg := plugins.NewRegistry()
	if err := reg.Register(New()); err != nil {
		t.Fatal(err)
	}
	return render.New(reg, nil, nil)
}

func mystRenderer(t *testing.T) *render.Renderer {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".zenmd.yml"), []byte("markdown:\n  dialects: [myst]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	reg := plugins.NewRegistry()
	if err := reg.Register(New()); err != nil {
		t.Fatal(err)
	}
	dr, err := dialects.NewRegistry(myst.New())
	if err != nil {
		t.Fatal(err)
	}
	return render.New(reg, config.Load("", dir), nil).WithDialects(dr)
}

func page(t *testing.T, r *render.Renderer, src string) string {
	t.Helper()
	p, err := r.Render("a.md", []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(p.HTML, "zen-plugin-error") {
		t.Fatalf("plugin error: %s", p.HTML)
	}
	return p.HTML
}

// warnings renders src and returns the page's warnings, one string. A
// non-fatal problem is a page warning (ctx.Warn), never a line in the block.
func warnings(t *testing.T, r *render.Renderer, src string) string {
	t.Helper()
	p, err := r.Render("a.md", []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(p.HTML, "zen-block-status") {
		t.Errorf("a status line in the block: %s", p.HTML)
	}
	return strings.Join(p.Warnings, " | ")
}

var attrsRE = regexp.MustCompile(` data-zen-attrs="[^"]*"`)

func bare(s string) string { return attrsRE.ReplaceAllString(s, "") }

func has(t *testing.T, got string, want ...string) {
	t.Helper()
	for _, w := range want {
		if !strings.Contains(got, w) {
			t.Errorf("missing %q in %s", w, got)
		}
	}
}

func hasNot(t *testing.T, got string, bad ...string) {
	t.Helper()
	for _, w := range bad {
		if strings.Contains(got, w) {
			t.Errorf("unexpected %q in %s", w, got)
		}
	}
}

// order checks that the parts appear in the output in this order.
func order(t *testing.T, got string, parts ...string) {
	t.Helper()
	at := 0
	for _, p := range parts {
		i := strings.Index(got[at:], p)
		if i < 0 {
			t.Errorf("%q is missing or out of order in %s", p, got)
			return
		}
		at += i + len(p)
	}
}
