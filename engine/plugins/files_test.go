package plugins

import (
	"strings"
	"testing"
)

func TestResolvePath(t *testing.T) {
	cases := []struct {
		page, p, want string
		bad           bool
	}{
		{"docs/p.md", "/a.mmd", "a.mmd", false},
		{"docs/p.md", "/x/y/a.mmd", "x/y/a.mmd", false},
		{"docs/p.md", "a.mmd", "docs/a.mmd", false},
		{"p.md", "a.mmd", "a.mmd", false},
		{"a/b/p.md", "d/a.mmd", "a/b/d/a.mmd", false},
		{"docs/p.md", "../a.mmd", "a.mmd", false},
		{"docs/p.md", "./x/../a.mmd", "docs/a.mmd", false},
		{"docs/p.md", "../../a.mmd", "", true},
		{"p.md", "../a.mmd", "", true},
		{"docs/p.md", "/../a", "", true},
		{"docs/p.md", "/a/../../b", "", true},
		{"docs/p.md", "a\\b", "", true},
		{"docs/p.md", "a\x00b", "", true},
		{"docs/p.md", "a\nb", "", true},
		{"docs/p.md", "a\rb", "", true},
		{"docs/p.md", strings.Repeat("a", 1025), "", true},
		{"docs/p.md", strings.Repeat("a", 1024), "docs/" + strings.Repeat("a", 1024), false},
		{"docs/p.md", "", "", true},
		{"docs/p.md", "   ", "", true},
		{"docs/p.md", "/", ".", false},
		{"docs/p.md", "my flows/é.MMD", "docs/my flows/é.MMD", false},
	}
	for _, c := range cases {
		got, err := ResolvePath(c.page, c.p)
		if c.bad {
			if err == nil {
				t.Errorf("ResolvePath(%q, %q) = %q, want an error", c.page, c.p, got)
			}
			continue
		}
		if err != nil || got != c.want {
			t.Errorf("ResolvePath(%q, %q) = %q, %v; want %q", c.page, c.p, got, err, c.want)
		}
	}
}

func TestFileRefOrder(t *testing.T) {
	b := Block{Name: "mermaid", Body: ":file: line.mmd\n", Attrs: map[string]string{"file": "attr.mmd", "arg": "arg.mmd"}}
	if p, _, ok := FileRef(b); !ok || p != "attr.mmd" {
		t.Errorf("attr first: %q %v", p, ok)
	}
	delete(b.Attrs, "file")
	if p, _, ok := FileRef(b); !ok || p != "line.mmd" {
		t.Errorf("line second: %q %v", p, ok)
	}
	b.Body = "graph TD\n"
	if p, body, ok := FileRef(b); !ok || p != "arg.mmd" || body != "graph TD\n" {
		t.Errorf("arg last: %q %q %v", p, body, ok)
	}
	// more than one word in the argument stays whole
	b.Attrs["arg"] = "my flows/a.mmd"
	if p, _, _ := FileRef(b); p != "my flows/a.mmd" {
		t.Errorf("arg with a space: %q", p)
	}
}

func TestFileRefLineStripped(t *testing.T) {
	p, body, ok := FileRef(Block{Body: ":file: diagrams/flow.mmd\nA --> B\n"})
	if !ok || p != "diagrams/flow.mmd" || body != "A --> B\n" {
		t.Errorf("got %q %q %v", p, body, ok)
	}
	p, body, ok = FileRef(Block{Body: ":file: a.mmd\r\n"})
	if !ok || p != "a.mmd" || strings.TrimSpace(body) != "" {
		t.Errorf("crlf: %q %q %v", p, body, ok)
	}
}

func TestFileRefBlankLinesBefore(t *testing.T) {
	p, body, ok := FileRef(Block{Body: "\n  \n:file: a.mmd\nrest\n"})
	if !ok || p != "a.mmd" || body != "rest\n" {
		t.Errorf("got %q %q %v", p, body, ok)
	}
}

func TestFileRefNone(t *testing.T) {
	for _, b := range []Block{
		{Body: "graph TD\nA-->B\n"},
		{Body: "graph TD\n:file: not-first.mmd\n"},
		{Body: ":file:\n"},
		{Body: ":file:   \n"},
		{Attrs: map[string]string{"file": "  "}},
		{},
	} {
		if p, _, ok := FileRef(b); ok {
			t.Errorf("%+v: got %q, want none", b, p)
		}
	}
	// an empty :file: line is dropped from the body
	if _, body, _ := FileRef(Block{Body: ":file:\nA\n"}); body != "A\n" {
		t.Errorf("body %q", body)
	}
}
