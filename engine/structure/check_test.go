package structure

import (
	"fmt"
	"strings"
	"testing"
)

// want is "severity rule line" (line 0 for "any"), then text the message must contain.
type want struct {
	sev, rule string
	line      int
	msg       string
}

func render(ps []Problem) string {
	var b strings.Builder
	for _, p := range ps {
		fmt.Fprintf(&b, "\n  %s %s line %d: %s", p.Severity, p.Rule, p.Line, p.Message)
	}
	return b.String()
}

func mustTemplate(t *testing.T, y string) *Template {
	t.Helper()
	tpl, err := Parse("t", []byte(y))
	if err != nil {
		t.Fatalf("template does not parse: %v\n%s", err, y)
	}
	return tpl
}

func match(ps []Problem, w want) bool {
	for _, p := range ps {
		if p.Severity == w.sev && p.Rule == w.rule && (w.line == 0 || p.Line == w.line) && strings.Contains(p.Message, w.msg) {
			return true
		}
	}
	return false
}

func TestCheckTable(t *testing.T) {
	cases := []struct {
		name string
		tpl  string
		page string
		want []want // each must be reported; len(want)==0 means clean; exact says the count
		n    int    // the exact number of problems
	}{
		{
			name: "clean how-to",
			tpl: `
structure:
  - heading: { level: 1, count: 1 }
  - paragraph: { min: 1, max-words: 20, label: Summary }
  - section: { heading: "Steps", contains: [ { list: ordered, min-items: 2 } ] }
  - end: {}
`,
			page: "# Title\n\nShort summary.\n\n## Steps\n\n1. a\n2. b\n",
		},
		{
			name: "missing section is a warning marked at the heading before the gap",
			tpl:  "structure:\n  - heading: { level: 1 }\n  - section: { heading: Steps }\n",
			page: "# Title\n\nText.\n",
			want: []want{{"warning", "section", 1, "Missing section 'Steps'"}}, n: 1,
		},
		{
			name: "optional section may be absent",
			tpl:  "structure:\n  - section: { heading: Steps, optional: true }\n",
			page: "# Title\n",
		},
		{
			name: "severity on an element",
			tpl:  "structure:\n  - section: { heading: Steps, severity: error }\n",
			page: "# Title\n",
			want: []want{{"error", "section", 0, "Steps"}}, n: 1,
		},
		{
			name: "suggestion severity",
			tpl:  "structure:\n  - section: { heading: Steps, severity: suggestion }\n",
			page: "# Title\n",
			want: []want{{"suggestion", "section", 0, "Steps"}}, n: 1,
		},
		{
			name: "no H1 and no title field",
			tpl:  "structure:\n  - heading: { level: 1, count: 1 }\n",
			page: "Just text.\n",
			want: []want{{"warning", "heading", 1, "H1"}}, n: 1,
		},
		{
			name: "title field stands for the H1",
			tpl:  "structure:\n  - heading: { level: 1, count: 1 }\n",
			page: "---\ntitle: Hello\n---\nJust text.\n",
		},
		{
			name: "two H1s",
			tpl:  "structure:\n  - heading: { level: 1, count: 1 }\n",
			page: "# One\n\ntext\n\n# Two\n",
			want: []want{{"warning", "heading", 5, "2 H1"}}, n: 1,
		},
		{
			name: "heading by text, case-insensitive",
			tpl:  "structure:\n  - heading: { level: 2, text: \"summary\" }\n",
			page: "# T\n\n## Summary\n",
		},
		{
			name: "section by regex and repeat",
			tpl:  "structure:\n  - section: { heading: \"/^(Troubleshooting|Next steps)$/\", repeat: true, min: 2 }\n",
			page: "# T\n\n## Troubleshooting\n\nx\n\n## Next steps\n\ny\n",
		},
		{
			name: "section min not reached",
			tpl:  "structure:\n  - section: { min: 2 }\n",
			page: "# T\n\n## Only\n",
			want: []want{{"warning", "section", 0, "at least 2"}}, n: 1,
		},
		{
			name: "section max exceeded",
			tpl:  "structure:\n  - section: { max: 1 }\n",
			page: "# T\n\n## A\n\n## B\n",
			want: []want{{"warning", "section", 5, "at most 1"}}, n: 1,
		},
		{
			name: "section out of order",
			tpl:  "structure:\n  - section: { heading: B }\n  - section: { heading: A }\n",
			page: "# T\n\n## A\n\n## B\n",
			want: []want{{"warning", "section", 0, "order"}}, n: 1,
		},
		{
			name: "section contains: the list is missing inside, not elsewhere",
			tpl:  "structure:\n  - section: { heading: Steps, contains: [ { list: ordered } ] }\n  - section: { heading: Other, optional: true }\n",
			page: "# T\n\n## Steps\n\ntext\n\n## Other\n\n1. a\n",
			want: []want{{"warning", "list", 3, "ordered list"}}, n: 1,
		},
		{
			name: "list min-items and max-items",
			tpl:  "structure:\n  - list: ordered\n    min-items: 3\n",
			page: "# T\n\n1. a\n2. b\n",
			want: []want{{"warning", "list", 3, "at least 3"}}, n: 1,
		},
		{
			name: "list max-items",
			tpl:  "structure:\n  - list: { type: unordered, max-items: 1 }\n",
			page: "- a\n- b\n",
			want: []want{{"warning", "list", 1, "at most 1"}}, n: 1,
		},
		{
			name: "task list is not an ordered list",
			tpl:  "structure:\n  - list: task\n",
			page: "- [ ] a\n- [x] b\n",
		},
		{
			name: "task list missing",
			tpl:  "structure:\n  - list: task\n",
			page: "- a\n- b\n",
			want: []want{{"warning", "list", 0, "task list"}}, n: 1,
		},
		{
			name: "paragraph too long",
			tpl:  "structure:\n  - heading: { level: 1 }\n  - paragraph: { max-words: 3, label: Summary }\n",
			page: "# T\n\none two three four five\n",
			want: []want{{"warning", "paragraph", 3, "Summary has 5 words"}}, n: 1,
		},
		{
			name: "paragraph must come before the next heading",
			tpl:  "structure:\n  - heading: { level: 1 }\n  - paragraph: { label: Summary }\n",
			page: "# T\n\n## Later\n\nA paragraph in a section.\n",
			want: []want{{"warning", "paragraph", 1, "Summary"}}, n: 1,
		},
		{
			name: "paragraph max",
			tpl:  "structure:\n  - paragraph: { max: 1 }\n",
			page: "one\n\ntwo\n",
			want: []want{{"warning", "paragraph", 3, "at most 1"}}, n: 1,
		},
		{
			name: "block by container name",
			tpl:  "structure:\n  - section: { heading: Step, contains: [ { block: check } ] }\n",
			page: "## Step\n\n:::check\n- [ ] a\n:::\n",
		},
		{
			name: "block missing",
			tpl:  "structure:\n  - section: { heading: Step, contains: [ { block: check } ] }\n",
			page: "## Step\n\ntext\n",
			want: []want{{"warning", "block", 1, "check block"}}, n: 1,
		},
		{
			name: "block as a fenced code block",
			tpl:  "structure:\n  - block: { name: mermaid, min: 1, max: 1 }\n",
			page: "```mermaid\ngraph TD\n```\n",
		},
		{
			name: "block max",
			tpl:  "structure:\n  - block: { name: mermaid, max: 1 }\n",
			page: "```mermaid\na\n```\n\n```mermaid\nb\n```\n",
			want: []want{{"warning", "block", 0, "at most 1"}}, n: 1,
		},
		{
			name: "every step but the last ends with a check block",
			tpl:  "structure:\n  - section: { min: 2, contains: [ { block: { name: check, except-last: true } } ] }\n",
			page: "# C\n\n## One\n\n:::check\n- [ ] a\n:::\n\n## Two\n\ntext\n",
		},
		{
			name: "a step but the last has no check",
			tpl:  "structure:\n  - section: { min: 2, contains: [ { block: { name: check, except-last: true } } ] }\n",
			page: "# C\n\n## One\n\ntext\n\n## Two\n\ntext\n",
			want: []want{{"warning", "block", 3, "check"}}, n: 1,
		},
		{
			name: "block ids unique page-wide",
			tpl:  "structure:\n  - block: { name: quiz, min: 0, ids: unique }\n",
			page: ":::quiz{id=a}\nx\n:::\n\n:::quiz{id=a}\ny\n:::\n",
			want: []want{{"warning", "block", 5, "id 'a'"}}, n: 1,
		},
		{
			name: "end: nothing after",
			tpl:  "structure:\n  - heading: { level: 1 }\n  - section: { heading: Steps }\n  - end: {}\n",
			page: "# T\n\n## Steps\n\nx\n\n## Extra\n\ny\n",
			want: []want{{"warning", "end", 7, "Unexpected"}}, n: 1,
		},
		{
			name: "end is satisfied",
			tpl:  "structure:\n  - heading: { level: 1 }\n  - end: {}\n",
			page: "# T\n",
		},
		{
			name: "code with a language",
			tpl:  "structure:\n  - code: { lang: required }\n",
			page: "```go\nx\n```\n",
		},
		{
			name: "code without a language",
			tpl:  "structure:\n  - code: { lang: required }\n",
			page: "text\n\n```\nx\n```\n",
			want: []want{{"warning", "code", 3, "language"}}, n: 1,
		},
		{
			name: "table columns",
			tpl:  "structure:\n  - table: { columns: [Name, Type], min-rows: 1 }\n",
			page: "| Name | Type |\n|---|---|\n| a | b |\n",
		},
		{
			name: "table column missing",
			tpl:  "structure:\n  - table: { columns: [Name, Type] }\n",
			page: "| Name | Size |\n|---|---|\n| a | b |\n",
			want: []want{{"warning", "table", 1, "Type"}}, n: 1,
		},
		{
			name: "image with alt",
			tpl:  "structure:\n  - image: { alt: required }\n",
			page: "![A cat](cat.png)\n",
		},
		{
			name: "image without alt",
			tpl:  "structure:\n  - image: { alt: required }\n",
			page: "![](cat.png)\n",
			want: []want{{"warning", "image", 1, "alt"}}, n: 1,
		},
		{
			name: "slides: each has a heading and few bullets",
			tpl:  "structure:\n  - slide: { min: 2, heading: required, max-bullets: 2, severity: suggestion }\n",
			page: "# One\n\n- a\n- b\n\n---\n\n# Two\n\n- c\n",
		},
		{
			name: "slides: too few, no heading, too many bullets",
			tpl:  "structure:\n  - slide: { min: 3, heading: required, max-bullets: 1, severity: suggestion }\n",
			page: "# One\n\n- a\n- b\n\n---\n\njust text\n",
			want: []want{{"suggestion", "slide", 0, "at least 3"}, {"suggestion", "slide", 0, "no heading"}, {"suggestion", "slide", 3, "2 bullets"}}, n: 3,
		},
		{
			name: "any: free text, at most",
			tpl:  "structure:\n  - heading: { level: 1 }\n  - any: { max: 1 }\n",
			page: "# T\n\none\n\ntwo\n",
			want: []want{{"warning", "any", 5, "at most 1"}}, n: 1,
		},
		{
			name: "rule heading-order",
			tpl:  "rules:\n  heading-order: true\n",
			page: "# T\n\n## A\n\n#### Deep\n",
			want: []want{{"warning", "heading-order", 5, "H2 to H4"}}, n: 1,
		},
		{
			name: "rule heading-order ok going up",
			tpl:  "rules:\n  heading-order: true\n",
			page: "# T\n\n## A\n\n### B\n\n## C\n",
		},
		{
			name: "rule image-alt required is an error",
			tpl:  "rules:\n  image-alt: required\n",
			page: "![](a.png)\n\n![ok](b.png)\n",
			want: []want{{"error", "image-alt", 1, "alt"}}, n: 1,
		},
		{
			name: "rule max-words with severity",
			tpl:  "rules:\n  max-words: { words: 4, severity: suggestion }\n",
			page: "# Title here\n\none two three\n",
			want: []want{{"suggestion", "max-words", 1, "5 words"}}, n: 1,
		},
		{
			name: "rule code-lang",
			tpl:  "rules:\n  code-lang: suggestion\n",
			page: "```\nx\n```\n",
			want: []want{{"suggestion", "code-lang", 1, "language"}}, n: 1,
		},
		{
			name: "fields: required and types",
			tpl: `
fields:
  date:   { type: date, required: true }
  tags:   { type: list }
  draft:  { type: bool }
  rating: { type: number }
`,
			page: "---\ndate: 2026-11-17\ntags: [a]\ndraft: true\nrating: 3\n---\n# T\n",
		},
		{
			name: "fields: missing required, wrong types",
			tpl: `
fields:
  date:   { type: date, required: true }
  tags:   { type: list }
  rating: { type: number }
  audience: { type: string, required: true, severity: suggestion }
`,
			page: "---\ntags: one\nrating: high\n---\n# T\n",
			want: []want{
				{"warning", "fields", 1, "date is required"},
				{"warning", "fields", 2, "tags must be a list"},
				{"warning", "fields", 3, "rating must be a number"},
				{"suggestion", "fields", 1, "audience is required"},
			}, n: 4,
		},
		{
			name: "fields: a bad date, enum and pattern",
			tpl: `
fields:
  date:   { type: date }
  kind:   { enum: [a, b] }
  slug:   { pattern: "^[a-z]+$" }
`,
			page: "---\ndate: someday\nkind: c\nslug: Bad1\n---\n# T\n",
			want: []want{{"warning", "fields", 2, "date must be a date"}, {"warning", "fields", 3, "one of a, b"}, {"warning", "fields", 4, "slug"}}, n: 3,
		},
		{
			name: "fields: title is met by an H1",
			tpl:  "fields:\n  title: { required: true }\n",
			page: "# Hello\n",
		},
		{
			name: "fields: title missing everywhere",
			tpl:  "fields:\n  title: { required: true }\n",
			page: "text\n",
			want: []want{{"warning", "fields", 1, "title is required"}}, n: 1,
		},
		{
			name: "lines count the frontmatter",
			tpl:  "structure:\n  - section: { heading: A }\n  - paragraph: { max-words: 1, label: Lead }\n",
			page: "---\na: 1\nb: 2\n---\n\n## A\n\ntoo many words here\n",
			want: []want{{"warning", "paragraph", 8, "Lead has 4 words"}}, n: 1,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			tpl := mustTemplate(t, c.tpl)
			got := Check(Analyse([]byte(c.page)), tpl, nil)
			for _, w := range c.want {
				if !match(got, w) {
					t.Errorf("want %+v in:%s", w, render(got))
				}
			}
			if len(got) != c.n {
				t.Errorf("want %d problems, got %d:%s", c.n, len(got), render(got))
			}
			for _, p := range got {
				if p.Message == "" || p.Fix == "" || p.Line < 1 {
					t.Errorf("a problem needs a message, a fix and a line: %+v", p)
				}
			}
		})
	}
}

func TestLayoutAndTemplate(t *testing.T) {
	layout, err := ParseLayout("post", []byte(`
fields:
  date: { type: date, required: true }
  author: { type: string, severity: warning, required: true }
rules:
  image-alt: true
  heading-order: true
`))
	if err != nil {
		t.Fatal(err)
	}
	if layout.Kind != "layout" || layout.ID != "post" {
		t.Errorf("layout = %+v", layout)
	}
	d := Analyse([]byte("---\nauthor: Ada\n---\n# T\n\n![](x.png)\n"))
	got := Check(d, nil, layout)
	if !match(got, want{"warning", "fields", 0, "date is required by the post layout"}) || !match(got, want{"warning", "image-alt", 0, ""}) {
		t.Errorf("layout alone:%s", render(got))
	}
	for _, p := range got {
		if p.Source != "layout post" {
			t.Errorf("source = %q", p.Source)
		}
	}

	// A template that declares the field and the rule wins over the layout's.
	tpl := mustTemplate(t, "fields:\n  date: { type: date }\nrules:\n  image-alt: suggestion\n")
	tpl.ID = "note"
	got = Check(d, tpl, layout)
	if match(got, want{"warning", "fields", 0, "date is required"}) {
		t.Errorf("the template says date is optional:%s", render(got))
	}
	if !match(got, want{"suggestion", "image-alt", 0, ""}) || match(got, want{"warning", "image-alt", 0, ""}) {
		t.Errorf("the template's image-alt wins:%s", render(got))
	}
	for _, p := range got {
		if p.Rule == "image-alt" && p.Source != "template t" && p.Source != "template note" {
			t.Errorf("source = %q", p.Source)
		}
	}
}

func TestParseErrors(t *testing.T) {
	long := strings.Repeat("a", 201)
	for name, y := range map[string]string{
		"unknown element":     "structure:\n  - paragrah: {}\n",
		"unknown option":      "structure:\n  - paragraph: { colour: red }\n",
		"bad severity":        "structure:\n  - paragraph: { severity: awful }\n",
		"bad regex":           "structure:\n  - section: { heading: \"/(/\" }\n",
		"pattern too long":    "structure:\n  - section: { heading: \"/" + long + "/\" }\n",
		"field pattern long":  "fields:\n  a: { pattern: \"" + long + "\" }\n",
		"field bad pattern":   "fields:\n  a: { pattern: \"(\" }\n",
		"field bad type":      "fields:\n  a: { type: blob }\n",
		"list kind":           "structure:\n  - list: sideways\n",
		"structure not list":  "structure: nope\n",
		"rule unknown":        "rules:\n  shout: true\n",
		"rule bad value":      "rules:\n  heading-order: maybe\n",
		"contains not a list": "structure:\n  - section: { contains: 3 }\n",
	} {
		if _, err := Parse("t", []byte(y)); err == nil {
			t.Errorf("%s: want an error", name)
		}
	}
	if _, err := Parse("t", []byte("structure:\n  - section: { heading: \"/" + strings.Repeat("a", 200) + "/\" }\n")); err != nil {
		t.Errorf("a pattern of 200 characters is allowed: %v", err)
	}
	// A page template file as the built-ins write it: no structure at all.
	if _, err := Parse("how-to", []byte("id: how-to\nname: How\nlayout: docs\nfields:\n  title: { required: true }\nskeleton: |\n  # {{title}}\n")); err != nil {
		t.Errorf("a template without structure parses: %v", err)
	}
}

// A hostile pattern runs in linear time (RE2).
func TestRegexLinear(t *testing.T) {
	tpl := mustTemplate(t, "structure:\n  - section: { heading: \"/^(a+)+$/\" }\n")
	page := "## " + strings.Repeat("a", 5000) + "!\n"
	got := Check(Analyse([]byte(page)), tpl, nil)
	if len(got) != 1 {
		t.Errorf("got %d problems", len(got))
	}
}

func TestAnchorIsTheHeadingBeforeTheGap(t *testing.T) {
	tpl := mustTemplate(t, "structure:\n  - heading: { level: 1 }\n  - section: { heading: Intro }\n  - section: { heading: Steps }\n")
	got := Check(Analyse([]byte("# Title\n\n## Intro\n\ntext\n")), tpl, nil)
	if len(got) != 1 || got[0].Anchor != "intro" || got[0].Line != 3 {
		t.Fatalf("got%s anchor=%v", render(got), got)
	}
	got = Check(Analyse([]byte("text\n")), mustTemplate(t, "structure:\n  - section: { heading: Intro }\n"), nil)
	if len(got) != 1 || got[0].Anchor != "" || got[0].Line != 1 {
		t.Fatalf("top of the page: got%s", render(got))
	}
}

func TestFixLine(t *testing.T) {
	tpl := mustTemplate(t, "structure:\n  - heading: { level: 1 }\n  - section: { heading: Steps }\n")
	got := Check(Analyse([]byte("# T\n")), tpl, nil)
	if len(got) != 1 || !strings.Contains(got[0].Fix, "## Steps") {
		t.Errorf("fix = %+v", got)
	}
}
