package blueprints_test

import (
	"path/filepath"
	"testing"

	"github.com/zenmark-org/zenmd/builtins/blueprints"
	"github.com/zenmark-org/zenmd/engine/session"
	enginetemplates "github.com/zenmark-org/zenmd/engine/sitetemplates"
)

// Every built-in blueprint starts a folder that passes zenmd check: no errors, and no structure
// problem at all except suggestions (the demo of what a fresh folder looks like is clean).
func TestBlueprintsStartClean(t *testing.T) {
	home := t.TempDir() // the check reads the settings folder: never the real one
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	list, _ := enginetemplates.Builtins(blueprints.FS)
	if len(list) < 3 {
		t.Fatalf("%d blueprints", len(list))
	}
	for _, tp := range list {
		t.Run(tp.ID, func(t *testing.T) {
			root := fresh(t, tp.ID)
			rep, err := session.CheckFolder(session.SiteEnv{GlobalPath: filepath.Join(home, "config.yml")}, session.CheckOptions{Dir: root})
			if err != nil {
				t.Fatal(err)
			}
			if rep.Pages == 0 {
				t.Fatal("no pages were checked")
			}
			for _, r := range rep.Results {
				for _, p := range r.Problems {
					if p.Severity == "error" {
						t.Errorf("%s:%d: error %s: %s", r.Path, p.Line, p.Rule, p.Message)
					} else {
						t.Logf("%s:%d: %s %s: %s", r.Path, p.Line, p.Severity, p.Rule, p.Message)
					}
				}
			}
		})
	}
}
