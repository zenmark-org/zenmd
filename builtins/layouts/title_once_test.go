package layouts

import (
	"io/fs"
	"regexp"
	"strings"
	"testing"
)

// A layout that draws the page title as its own <h1> must not show the page's
// H1 again: it inserts .Body (the text without its leading H1), never .Content.
// The page's title is its H1 (spec engine/templates-blueprints, "Titles").
func TestTitleShownOnce(t *testing.T) {
	ownH1 := regexp.MustCompile(`<h1[^>]*>\s*\{\{\s*\.(Title|TermTitle)\s*\}\}`)
	files, _ := fs.Glob(files, "*/layout.html")
	if len(files) == 0 {
		t.Fatal("no layout templates found")
	}
	for _, f := range files {
		b, _ := fs.ReadFile(files, f)
		src := string(b)
		if ownH1.MatchString(src) && strings.Contains(src, "{{.Content}}") {
			t.Errorf("%s draws the title as an h1 and also inserts .Content (its H1 shows twice); use .Body", f)
		}
	}
}
