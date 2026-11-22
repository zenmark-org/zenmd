package pagetemplates

import (
	"io/fs"
	"regexp"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestBuiltinPageTemplates(t *testing.T) {
	entries, err := fs.ReadDir(FS, ".")
	if err != nil {
		t.Fatal(err)
	}
	have := map[string]bool{}
	for _, e := range entries {
		id := strings.TrimSuffix(e.Name(), ".yml")
		have[id] = true
		data, _ := fs.ReadFile(FS, e.Name())
		var p struct {
			ID, Name, Description, Layout, Skeleton, Version string
			Fields                                           map[string]any
		}
		if err := yaml.Unmarshal(data, &p); err != nil {
			t.Fatalf("%s: %v", e.Name(), err)
		}
		if p.ID != id || p.Name == "" || p.Description == "" || p.Layout == "" || len(p.Fields) == 0 || strings.TrimSpace(p.Skeleton) == "" {
			t.Errorf("%s: id, name, description, layout, fields and skeleton are all needed: %+v", e.Name(), p)
		}
		if !regexp.MustCompile(`^\d+\.\d+\.\d+$`).MatchString(p.Version) {
			t.Errorf("%s: version %q is not a version such as 1.0.0 (installed copies are updated by it)", id, p.Version)
		}
		if _, ok := p.Fields["title"]; !ok && id != "term" { // a term's title is the H1 of its skeleton
			t.Errorf("%s: no title field", id)
		}
	}
	for _, id := range IDs {
		if !have[id] {
			t.Errorf("missing page template %s", id)
		}
	}
	if len(have) != len(IDs) {
		t.Errorf("IDs %v do not match the files %v", IDs, have)
	}
	// The ones the spec names: release notes need a version and a date; a term uses the term layout.
	data, _ := fs.ReadFile(FS, "release-notes.yml")
	for _, w := range []string{"version:", "required: true", "date:"} {
		if !strings.Contains(string(data), w) {
			t.Errorf("release-notes lacks %q", w)
		}
	}
	if d, _ := fs.ReadFile(FS, "term.yml"); !strings.Contains(string(d), "layout: term") {
		t.Error("term must use the term layout")
	} else if !strings.Contains(string(d), "skeleton: |\n  # {{title}}\n") || strings.Contains(string(d), "title:") {
		t.Error("a term starts with # {{title}} and has no title in its frontmatter")
	}
}

// The release-notes page template: the release layout, no title and no H1 (the
// version is the title); version and date are required, name is optional.
func TestReleaseNotesTemplate(t *testing.T) {
	data, _ := fs.ReadFile(FS, "release-notes.yml")
	src := string(data)
	for _, w := range []string{"layout: release", "version: { type: string, required: true }", "date:", "required: true", "name:", "skeleton: |\n  ## Added\n"} {
		if !strings.Contains(src, w) {
			t.Errorf("release-notes lacks %q", w)
		}
	}
	if strings.Contains(src, "# {{title}}") || strings.Contains(src, "\n  title:") || strings.Contains(src, "version: 1.0.0\nlayout") {
		t.Errorf("release-notes has a title or an H1 or the old version:\n%s", src)
	}
	if !strings.Contains(src, "\nversion: 1.1.0\n") {
		t.Errorf("release-notes version must be bumped to 1.1.0 (installed copies get the update):\n%s", src)
	}
}
