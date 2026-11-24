package myst

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/zenmark-org/zenmd/engine/dialects"
)

func roles(src string, ext dialects.Extensions) (out []string) {
	for _, sp := range New().(dialects.InlineScanner).ScanInline([]byte(src), ext) {
		out = append(out, sp.Name+"="+sp.Body)
	}
	return
}

func TestScanInlineRoles(t *testing.T) {
	src := "See {ref}`intro`, then {doc}`Guide <guide/start>` and {download}`a.zip`."
	got := strings.Join(roles(src, nil), "|")
	if got != "ref=intro|doc=Guide <guide/start>|download=a.zip" {
		t.Errorf("roles: %q", got)
	}
	sps := New().(dialects.InlineScanner).ScanInline([]byte(src), nil)
	if src[sps[0].Start:sps[0].End] != "{ref}`intro`" {
		t.Errorf("range %q", src[sps[0].Start:sps[0].End])
	}
	// a double-backtick role keeps single backticks inside
	if got := roles("{math}``a`b``", nil); len(got) != 1 || got[0] != "math=a`b" {
		t.Errorf("double: %v", got)
	}
}

func TestScanInlineNotRoles(t *testing.T) {
	for _, src := range []string{
		"`{ref}`x``",             // inside a code span
		"\\{ref}`x`",             // escaped
		"{Ref}`x`",               // not a role name
		"{ref} `x`",              // a space
		"{ref}`unclosed",         // no closing backtick
		"{ref}``",                // empty
		"text {not a role} here", // plain braces
		"{ref}`a\nb`",            // roles are one line
	} {
		if got := roles(src, nil); len(got) != 0 {
			t.Errorf("%q: found %v", src, got)
		}
	}
	// hostile content stays in the span body for the plugin to escape; the scan never rewrites it
	if got := roles("{ref}`<script>alert(1)</script>`", nil); len(got) != 1 {
		t.Errorf("hostile: %v", got)
	}
}

func TestScanInlineDollarMath(t *testing.T) {
	on := dialects.Extensions{"dollarmath": true}
	if got := roles("so $x^2$ and $a+b$.", nil); len(got) != 0 {
		t.Errorf("without dollarmath: %v", got)
	}
	if got := strings.Join(roles("so $x^2$ and $a+b$.", on), "|"); got != "math=x^2|math=a+b" {
		t.Errorf("with dollarmath: %q", got)
	}
	for _, src := range []string{
		"it costs $5 and $10 today", // closing $ followed by a digit
		"a $ b $ c",                 // spaces inside
		"$$display$$",               // blocks are Scan's
		"\\$x$",                     // escaped
		"`$x$`",                     // a code span
		"$unclosed",                 // no closer
		"{ref}`$a$` stays one role", // inside a role
	} {
		got := roles(src, on)
		if src[0] == '{' {
			if len(got) != 1 || got[0] != "ref=$a$" {
				t.Errorf("%q: %v", src, got)
			}
			continue
		}
		if len(got) != 0 {
			t.Errorf("%q: found %v", src, got)
		}
	}
}

func TestExtensions(t *testing.T) {
	d := New().(dialects.Extensioner)
	for name, files := range map[string]map[string]string{
		"jupyter book list": {"_config.yml": "parse:\n  myst_enable_extensions:\n    - amsmath\n    - dollarmath\n"},
		"jupyter book flow": {"_config.yml": "parse:\n  myst_enable_extensions: [colon_fence, dollarmath]\n"},
		"sphinx":            {"conf.py": "myst_enable_extensions = [\n    \"dollarmath\",\n]\n"},
		"myst.yml":          {"myst.yml": "project:\n  enable_extensions:\n    - dollarmath\n"},
	} {
		fsys := fstest.MapFS{}
		for n, c := range files {
			fsys[n] = &fstest.MapFile{Data: []byte(c)}
		}
		if !d.Extensions(fsys)["dollarmath"] {
			t.Errorf("%s: dollarmath not found", name)
		}
	}
	off := fstest.MapFS{"_config.yml": {Data: []byte("parse:\n  myst_enable_extensions:\n    - amsmath\ntitle: dollarmath notes\n")}}
	if d.Extensions(off)["dollarmath"] || d.Extensions(nil)["dollarmath"] {
		t.Error("dollarmath found where it is not enabled")
	}
}

func rewrite(src string, subs map[string]string) (string, []string) {
	out, w := New().(dialects.Rewriter).Rewrite([]byte(src), dialects.Page{Substitutions: subs})
	return string(out), w
}

func TestRewriteCommentsAndBreaks(t *testing.T) {
	got, _ := rewrite("one\n% a comment\n%another\ntwo\n\n+++\n\n+++ {\"k\": 1}\nthree\n", nil)
	if strings.Contains(got, "%") || strings.Contains(got, "+++") {
		t.Errorf("comment or break left: %q", got)
	}
	if strings.Count(got, "***") != 2 || !strings.Contains(got, "one\ntwo") {
		t.Errorf("rewrite: %q", got)
	}
	// code is left alone
	code := "```\n% not a comment\n+++\n(x)=\n{{ k }}\n```\n\n    % indented code\n\n`+++` and `{{ k }}`\n"
	if got, _ := rewrite(code, map[string]string{"k": "V"}); got != code {
		t.Errorf("code changed:\n%q", got)
	}
}

func TestRewriteTargets(t *testing.T) {
	got, _ := rewrite("(my-label)=\n# Heading\n\ntext\n\n(Other Label)=\nmore\n", nil)
	if !strings.Contains(got, `<span id="my-label"></span>`) || !strings.Contains(got, `<span id="other-label"></span>`) {
		t.Errorf("targets: %q", got)
	}
	if strings.Contains(got, ")=") || !strings.Contains(got, "# Heading") {
		t.Errorf("target line or heading: %q", got)
	}
	// a hostile label never reaches the HTML
	got, _ = rewrite("(a\"><script>alert1</script>)=\n# H\n", nil)
	if strings.Contains(got, "<script") {
		t.Errorf("hostile label: %q", got)
	}
}

func TestRewriteSubstitutions(t *testing.T) {
	subs := map[string]string{"name": "Zenmark", "v": "0.8"}
	got, w := rewrite("Hi {{ name }} {{v}} and {{ missing }}.\n", subs)
	if got != "Hi Zenmark 0.8 and {{ missing }}.\n" {
		t.Errorf("subst: %q", got)
	}
	if len(w) != 1 || !strings.Contains(w[0], "missing") {
		t.Errorf("warnings: %v", w)
	}
	// no substitutions defined: the text is left, no warning
	got, w = rewrite("a {{ x }} b\n", nil)
	if got != "a {{ x }} b\n" || len(w) != 0 {
		t.Errorf("no subs: %q %v", got, w)
	}
	// a long value is cut, and substitutions are not recursive
	got, _ = rewrite("{{ a }}\n", map[string]string{"a": "{{ b }}", "b": "no"})
	if got != "{{ b }}\n" {
		t.Errorf("recursive: %q", got)
	}
}

func TestFields(t *testing.T) {
	f := New().(dialects.FieldLister)
	fields, rest := f.Fields([]byte(":author: Ann Lee\n:status:   draft\n:tags: a, b\n  continued\n\n# Title\n"))
	if fields["author"] != "Ann Lee" || fields["status"] != "draft" || fields["tags"] != "a, b continued" {
		t.Errorf("fields: %v", fields)
	}
	if string(rest) != "# Title\n" {
		t.Errorf("rest %q", rest)
	}
	for _, src := range []string{"# Title\n:author: not at top\n", "text\n\n:a: b\n", ":::{note}\nx\n:::\n", ":kbd[x] is inline\n", ""} {
		if fields, rest := f.Fields([]byte(src)); len(fields) != 0 || string(rest) != src {
			t.Errorf("%q: fields %v rest %q", src, fields, rest)
		}
	}
}

func TestDefinitionListsFeature(t *testing.T) {
	if dl, ok := New().(dialects.DefinitionLists); !ok || !dl.DefinitionLists() {
		t.Error("the dialect should ask for definition lists")
	}
}
