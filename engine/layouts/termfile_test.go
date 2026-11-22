package layouts

import (
	"strings"
	"testing"
)

func TestTermSlug(t *testing.T) {
	for in, want := range map[string]string{
		"Idempotent":        "idempotent",
		"Front matter":      "front-matter",
		"  Éclair au café ": "eclair-au-cafe",
		"C++ (language)":    "c-language",
		"A/B test":          "a-b-test",
		"42 things!":        "42-things",
		"already-slugged":   "already-slugged",
		"???":               "",
		"":                  "",
	} {
		if got := TermSlug(in); got != want {
			t.Errorf("TermSlug(%q) = %q, want %q", in, got, want)
		}
	}
}

const skeleton = "# {{title}}\n\nThe definition in one or two sentences.\n\n## Examples\n\nAn example.\n"

func TestTermText(t *testing.T) {
	got := TermText(skeleton, "Idempotent", "")
	if !strings.HasPrefix(got, "---\nlayout: term\n---\n# Idempotent\n\nThe definition in one or two sentences.") || strings.Contains(got, "title:") {
		t.Errorf("no definition:\n%s", got)
	}
	got = TermText(skeleton, "Idempotent", "Doing it twice is doing it once.")
	if !strings.Contains(got, "# Idempotent\n\nDoing it twice is doing it once.\n\n## Examples") || strings.Contains(got, "one or two sentences") {
		t.Errorf("with definition:\n%s", got)
	}
}
