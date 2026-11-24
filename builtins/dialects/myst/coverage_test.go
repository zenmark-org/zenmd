package myst_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/zenmark-org/zenmd/builtins"
)

// row is one MyST name of specs/plugins/myst-builtins and the built-in that owns it.
type row struct {
	phase int
	name  string
	id    string
	role  bool // an inline role, declared in Inlines; otherwise a directive, in Blocks
}

// coverage maps every MyST name of the spec's phases to its built-in. A row is
// enforced once its built-in is registered (builtins.PluginIDs): the registry
// must declare the name, in Blocks for a directive or Inlines for a role.
// Names of a phase that is not checked in yet stay pending (listed in the log).
// Later phases add their rows here as an Integration line.
var coverage = []row{
	// phase 1: callouts
	{1, "note", "callout", false}, {1, "tip", "callout", false}, {1, "hint", "callout", false},
	{1, "important", "callout", false}, {1, "seealso", "callout", false},
	{1, "warning", "callout", false}, {1, "caution", "callout", false}, {1, "attention", "callout", false},
	{1, "danger", "callout", false}, {1, "error", "callout", false}, {1, "admonition", "callout", false},
	{1, "versionadded", "callout", false}, {1, "versionchanged", "callout", false}, {1, "deprecated", "callout", false},
	// phase 2: figures and tables
	{2, "figure", "figure", false}, {2, "image", "figure", false},
	{2, "table", "table", false}, {2, "list-table", "table", false}, {2, "csv-table", "table", false},
	// phase 3: math
	{3, "math", "math", false},
	{5, "math", "math", true}, {5, "eq", "math", true},
	// phase 4: layout blocks
	{4, "tab-set", "tabs", false}, {4, "tab-item", "tabs", false}, {4, "dropdown", "dropdown", false},
	{4, "card", "cards", false}, {4, "grid", "cards", false}, {4, "grid-item", "cards", false},
	{4, "grid-item-card", "cards", false}, {4, "button-link", "cards", false}, {4, "button-ref", "cards", false},
	{4, "bdg", "cards", true}, {4, "bdg-line", "cards", true}, {4, "bdg-primary", "cards", true}, {4, "bdg-primary-line", "cards", true}, {4, "bdg-secondary", "cards", true}, {4, "bdg-secondary-line", "cards", true}, {4, "bdg-success", "cards", true}, {4, "bdg-success-line", "cards", true}, {4, "bdg-info", "cards", true}, {4, "bdg-info-line", "cards", true}, {4, "bdg-warning", "cards", true}, {4, "bdg-warning-line", "cards", true}, {4, "bdg-danger", "cards", true}, {4, "bdg-danger-line", "cards", true}, {4, "bdg-light", "cards", true}, {4, "bdg-light-line", "cards", true}, {4, "bdg-dark", "cards", true}, {4, "bdg-dark-line", "cards", true},
	// phase 5: roles and the rest
	{5, "kbd", "text", true}, {5, "abbr", "text", true}, {5, "sub", "text", true}, {5, "sup", "text", true},
	{5, "ref", "xref", true}, {5, "numref", "xref", true}, {5, "doc", "xref", true}, {5, "download", "xref", true},
	{5, "toctree", "xref", false}, {5, "term", "terms", true}, {5, "glossary", "terms", false},
	{5, "epigraph", "quote", false}, {5, "pull-quote", "quote", false}, {5, "highlights", "quote", false},
	{5, "sidebar", "aside", false}, {5, "margin", "aside", false}, {5, "topic", "aside", false},
	{5, "include", "include", false}, {5, "literalinclude", "include", false},
	{5, "mermaid", "diagram", false},
}

// leftAsCode are directives no built-in may own: they stay code with a warning.
var leftAsCode = []string{"code-cell", "eval", "raw", "only", "ifconfig"}

func declared(t *testing.T) (blocks, inlines map[string][]string) {
	t.Helper()
	blocks, inlines = map[string][]string{}, map[string][]string{}
	for _, m := range builtins.Plugins().All() {
		for _, b := range m.Blocks {
			blocks[b] = append(blocks[b], m.ID)
		}
		for _, i := range m.Inlines {
			inlines[i] = append(inlines[i], m.ID)
		}
	}
	return
}

func TestMystNamesMapToBuiltins(t *testing.T) {
	blocks, inlines := declared(t)
	if !slices.Contains(builtins.PluginIDs, "callout") {
		t.Skip("the callout built-in is not registered yet (the lead adds it to builtins.go at merge); phase 1 rows are enforced after that")
	}
	var pending []string
	for _, r := range coverage {
		if !slices.Contains(builtins.PluginIDs, r.id) {
			if r.phase == 1 {
				t.Errorf("phase 1 name %q: built-in %q is not registered", r.name, r.id)
			} else {
				pending = append(pending, r.name+" -> "+r.id)
			}
			continue
		}
		owners := blocks[r.name]
		if r.role {
			owners = inlines[r.name]
		}
		if !slices.Contains(owners, r.id) {
			t.Errorf("%q (phase %d) should be owned by %q, declared by %v", r.name, r.phase, r.id, owners)
		}
	}
	if len(pending) > 0 {
		t.Logf("pending, built-in not registered yet: %s", strings.Join(pending, ", "))
	}
}

// One owner per name (the registry refuses a clash when plugins register), and
// the names the spec leaves as code have none.
func TestMystLeftAsCodeHasNoOwner(t *testing.T) {
	blocks, _ := declared(t)
	for _, n := range leftAsCode {
		if len(blocks[n]) > 0 {
			t.Errorf("%q must stay code, but %v declare it", n, blocks[n])
		}
	}
	for _, n := range []string{"code-block", "code"} {
		if len(blocks[n]) > 0 {
			t.Errorf("%q is plain code in the dialect, but %v declare it", n, blocks[n])
		}
	}
}
