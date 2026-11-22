package session

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zenmark-org/zenmd/engine/config"
)

// TestNoTitleNextToAnH1 keeps the shipped content to one title per page: no
// page of the demo, the showcase or the built-in blueprints has both a title:
// and an H1.
func TestNoTitleNextToAnH1(t *testing.T) {
	for _, root := range []string{"../../examples/demo", "../../examples/showcase", "../../builtins/blueprints"} {
		filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() || !strings.HasSuffix(p, ".md") {
				return nil
			}
			b, _ := os.ReadFile(p)
			fm, body, _ := config.SplitFrontmatter(b)
			if _, has := fm["title"]; !has {
				return nil
			}
			fence := false
			for _, l := range strings.Split(string(body), "\n") {
				if strings.HasPrefix(strings.TrimSpace(l), "```") {
					fence = !fence
				}
				if !fence && strings.HasPrefix(l, "# ") {
					t.Errorf("%s has an H1 and a title:", p)
					break
				}
			}
			return nil
		})
	}
}
