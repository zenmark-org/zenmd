package sitetemplates

import (
	"strings"
	"testing"
)

// frontOf is the frontmatter block of a page ("" when there is none).
func frontOf(s string) string {
	if !strings.HasPrefix(s, "---\n") {
		return ""
	}
	if i := strings.Index(s[4:], "\n---"); i >= 0 {
		return s[4 : 4+i]
	}
	return ""
}

func TestSectionsWriteNoTitleWhereThereIsAnH1(t *testing.T) {
	target, _ := buildSections(t, sectionsManifest, nil)
	for _, rel := range []string{
		"index.md", "getting-started/index.md", "getting-started/installation.md",
		"how-tos/index.md", "about.md", "releases/index.md", "pitch.md",
	} {
		s := read(t, target, rel)
		if !strings.Contains(s, "\n# ") && !strings.HasPrefix(s, "# ") {
			continue // no H1: nothing to say
		}
		for _, line := range strings.Split(frontOf(s), "\n") {
			if strings.HasPrefix(line, "title:") {
				t.Errorf("%s has an H1 and a title:\n%s", rel, s)
			}
		}
	}
	if in := read(t, target, "getting-started/installation.md"); !strings.Contains(in, "# Installation") {
		t.Errorf("installation lacks its H1:\n%s", in)
	}
	if pitch := read(t, target, "pitch.md"); !strings.Contains(pitch, "layout: slides") || !strings.Contains(pitch, "# Zed in five slides") {
		t.Errorf("slides page:\n%s", pitch)
	}
}
