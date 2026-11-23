package render

import (
	"fmt"
	"strings"
	"testing"

	"github.com/zenmark-org/zenmd/engine/config"
	"github.com/zenmark-org/zenmd/engine/plugins"
	"github.com/zenmark-org/zenmd/engine/workspace"
)

// rootResolver is a resolver that knows the folder root.
type rootResolver struct {
	fakeResolver
	root string
}

func (r rootResolver) Root() string { return r.root }

// warnPlugin calls Warn with each line of its block body, and keeps the context.
func warnPlugin(id string, seen *plugins.Context) *fakePlugin {
	m := man(id, true)
	m.Blocks = []string{id}
	m.Inlines = []string{id + "i"}
	return &fakePlugin{
		m: m,
		block: func(c plugins.Context, b plugins.Block) (string, error) {
			if seen != nil {
				*seen = c
			}
			if c.Warn != nil {
				for _, l := range strings.Split(strings.TrimSpace(b.Body), "\n") {
					c.Warn(l)
				}
			}
			return "<p>ok</p>", nil
		},
		inline: func(c plugins.Context, i plugins.Inline) (string, error) {
			if c.Warn != nil {
				c.Warn("inline " + i.Text)
			}
			return "<i>ok</i>", nil
		},
	}
}

func warnPage(t *testing.T, reg *plugins.Registry, src string) Page {
	t.Helper()
	return render(t, New(reg, config.Default(), fakeResolver{}), src)
}

func TestWarnReachesPageWarnings(t *testing.T) {
	p := warnPage(t, newReg(t, warnPlugin("w", nil)), ":::w\nunsupported option\n:::\n\nText :wi[x].\n")
	want := []string{`plugin "w": unsupported option`, `plugin "w": inline x`}
	if fmt.Sprint(p.Warnings) != fmt.Sprint(want) {
		t.Errorf("warnings %q, want %q", p.Warnings, want)
	}
	contains(t, p.HTML, "ok")
}

func TestWarnLimitPerPlugin(t *testing.T) {
	var lines []string
	for i := 0; i < 30; i++ {
		lines = append(lines, fmt.Sprintf("w%d", i))
	}
	p := warnPage(t, newReg(t, warnPlugin("w", nil), warnPlugin("v", nil)),
		":::w\n"+strings.Join(lines, "\n")+"\n:::\n\n:::v\nother\n:::\n")
	n := 0
	for _, w := range p.Warnings {
		if strings.HasPrefix(w, `plugin "w"`) {
			n++
		}
	}
	if n != 20 {
		t.Errorf("%d warnings from w, want 20: %q", n, p.Warnings)
	}
	if !strings.Contains(strings.Join(p.Warnings, "\n"), `plugin "v": other`) {
		t.Errorf("the limit is per plugin: %q", p.Warnings)
	}
	// A fresh page starts again.
	p = warnPage(t, newReg(t, warnPlugin("w", nil)), ":::w\nagain\n:::\n")
	if len(p.Warnings) != 1 {
		t.Errorf("second page: %q", p.Warnings)
	}
}

func TestWarnTrimAndClean(t *testing.T) {
	var got []string
	pl := warnPlugin("w", nil)
	pl.block = func(c plugins.Context, b plugins.Block) (string, error) {
		c.Warn(strings.Repeat("é", 300))
		c.Warn("a\x00b\x1b[31m\tc\nd\x7f")
		c.Warn("")
		c.Warn("\x01\x02")
		return "<p>ok</p>", nil
	}
	p := warnPage(t, newReg(t, pl), ":::w\nx\n:::\n")
	got = p.Warnings
	if len(got) < 1 {
		t.Fatalf("no warnings")
	}
	msg := strings.TrimPrefix(got[0], `plugin "w": `)
	if n := len([]rune(msg)); n != 200 {
		t.Errorf("message is %d characters, want 200", n)
	}
	for _, w := range got {
		for _, r := range w {
			if r < 0x20 || r == 0x7f {
				t.Errorf("control character %q in %q", r, w)
			}
		}
	}
	if !strings.Contains(strings.Join(got, "|"), `abc`) && !strings.Contains(strings.Join(got, "|"), "ab[31m") {
		t.Errorf("text around control characters lost: %q", got)
	}
	for _, w := range got {
		if w == `plugin "w": ` {
			t.Errorf("an empty message was added: %q", got)
		}
	}
}

func TestWarnDuplicatesDropped(t *testing.T) {
	p := warnPage(t, newReg(t, warnPlugin("w", nil)), ":::w\nsame\nsame\nother\nsame\n:::\n\n:::w\nsame\n:::\n")
	if len(p.Warnings) != 2 {
		t.Errorf("warnings %q, want same and other once each", p.Warnings)
	}
}

func TestWarnFromNestedMarkdown(t *testing.T) {
	outer := &fakePlugin{m: man("outer", true)}
	outer.m.Blocks = []string{"outer"}
	outer.block = func(c plugins.Context, b plugins.Block) (string, error) {
		return c.Markdown(b.Body)
	}
	p := warnPage(t, newReg(t, outer, warnPlugin("w", nil)), ":::outer\n:::w\ninside\n:::\n\nand :wi[y]\n:::\n")
	got := strings.Join(p.Warnings, "|")
	if !strings.Contains(got, `plugin "w": inside`) || !strings.Contains(got, `plugin "w": inline y`) {
		t.Errorf("nested warnings lost: %q", p.Warnings)
	}
	// The outer plugin's own Warn, called after it nested markdown, works too.
	outer.block = func(c plugins.Context, b plugins.Block) (string, error) {
		h, err := c.Markdown(b.Body)
		c.Warn("outer after")
		return h, err
	}
	p = warnPage(t, newReg(t, outer, warnPlugin("w", nil)), ":::outer\n:::w\ninside\n:::\n:::\n")
	got = strings.Join(p.Warnings, "|")
	if !strings.Contains(got, `plugin "outer": outer after`) || !strings.Contains(got, `plugin "w": inside`) {
		t.Errorf("warnings %q", p.Warnings)
	}
}

func TestWarnNilWithoutPage(t *testing.T) {
	reg := newReg(t, warnPlugin("w", nil))
	set, _ := reg.Enabled(func(string) (bool, map[string]any, bool) { return true, nil, true }, []string{"w"})
	if c := set.Context("w", "a.md"); c.Warn != nil || c.Workspace != "" {
		t.Errorf("a context from the set alone has Warn or Workspace: %+v", c)
	}
}

func TestWorkspaceInBlockHooks(t *testing.T) {
	var seen plugins.Context
	reg := newReg(t, warnPlugin("w", &seen))
	r := New(reg, config.Default(), rootResolver{fakeResolver{}, "/the/root"})
	render(t, r, ":::w\nx\n:::\n")
	if seen.Workspace != "/the/root" {
		t.Errorf("Workspace %q", seen.Workspace)
	}

	// An inline hook gets it too.
	var inl plugins.Context
	pl := warnPlugin("w", nil)
	pl.inline = func(c plugins.Context, i plugins.Inline) (string, error) { inl = c; return "<i>x</i>", nil }
	render(t, New(newReg(t, pl), config.Default(), rootResolver{fakeResolver{}, "/the/root"}), "a :wi[x] b\n")
	if inl.Workspace != "/the/root" {
		t.Errorf("inline Workspace %q", inl.Workspace)
	}
}

func TestWorkspaceEmptyWithoutRoot(t *testing.T) {
	var seen plugins.Context
	seen.Workspace = "unset"
	reg := newReg(t, warnPlugin("w", &seen))
	render(t, New(reg, config.Default(), fakeResolver{}), ":::w\nx\n:::\n")
	if seen.Workspace != "" {
		t.Errorf("Workspace %q with a resolver that has no root", seen.Workspace)
	}
	seen.Workspace = "unset"
	render(t, New(reg, config.Default(), nil), ":::w\nx\n:::\n")
	if seen.Workspace != "" {
		t.Errorf("Workspace %q with no resolver", seen.Workspace)
	}
}

var _ = workspace.Link{}
