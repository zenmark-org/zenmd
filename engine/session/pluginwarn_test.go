package session

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zenmark-org/zenmd/engine/plugins"
)

// ctxPlugin warns once per block and records the Workspace its hook got.
type ctxPlugin struct{ root *string }

func (ctxPlugin) Manifest() plugins.Manifest {
	return plugins.Manifest{ID: "ctxp", Version: "0.1.0", APIVersion: plugins.APIVersion, Default: true, Blocks: []string{"ctxp"}}
}

func (p ctxPlugin) RenderBlock(ctx plugins.Context, b plugins.Block) (string, error) {
	*p.root = ctx.Workspace
	if ctx.Warn != nil {
		ctx.Warn("careful: " + strings.TrimSpace(b.Body))
	}
	return "<p>ctx</p>", nil
}

func ctxRegistry(t *testing.T, root *string) *plugins.Registry {
	t.Helper()
	reg := plugins.NewRegistry()
	if err := reg.Register(ctxPlugin{root}); err != nil {
		t.Fatal(err)
	}
	return reg
}

func TestPluginWarnAndWorkspaceInSession(t *testing.T) {
	dir := tree(t, map[string]string{"README.md": "# Home\n\n:::ctxp\nthis\n:::\n"})
	var root string
	z, err := newZen(zenOptions{Registry: ctxRegistry(t, &root), ConfigDir: t.TempDir(), Dir: dir, Events: &sink{}, Dialogs: &fakeDialogs{Allow: true}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(z.close)
	p, err := z.Page("README.md")
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, w := range p.Warnings {
		if w == `plugin "ctxp": careful: this` {
			found = true
		}
	}
	if !found {
		t.Errorf("the plugin's warning did not reach the page (Problems): %q", p.Warnings)
	}
	if root == "" || root != z.ws.Root() {
		t.Errorf("Workspace %q, want %q", root, z.ws.Root())
	}
}

func TestPluginWorkspaceInSiteBuild(t *testing.T) {
	dir := tree(t, map[string]string{"README.md": "# Home\n\n:::ctxp\nthis\n:::\n"})
	var root string
	r := runBuild(t, dir, t.TempDir(), SiteOptions{}, func(e *SiteEnv) { e.Registry = ctxRegistry(t, &root) })
	if r.err != nil {
		t.Fatal(r.err)
	}
	want, _ := filepath.EvalSymlinks(dir)
	got, _ := filepath.EvalSymlinks(root)
	if root == "" || got != want {
		t.Errorf("Workspace %q in a build, want %q", root, dir)
	}
	if !hasLine(r.stderr, `plugin "ctxp"`, "careful: this") {
		t.Errorf("the build did not print the plugin's warning:\n%s", r.stderr)
	}
	_ = os.Stderr
}
