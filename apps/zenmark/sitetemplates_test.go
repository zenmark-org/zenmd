package app

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"sort"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/zenmark-org/zenmd/engine/lint"
	"github.com/zenmark-org/zenmd/engine/plugins"
	"github.com/zenmark-org/zenmd/engine/session"
	"github.com/zenmark-org/zenmd/engine/sitetemplates"
	"github.com/zenmark-org/zenmd/internal/cli"
	"github.com/zenmark-org/zenmd/kit/desktop"
)

// siteRig is a settings folder with a global config, and an env that uses
// zen's real presets.
func siteRig(t *testing.T, config string) (siteEnv, string) {
	t.Helper()
	cfg := t.TempDir()
	global := filepath.Join(cfg, "config.yml")
	if config != "" {
		if err := os.WriteFile(global, []byte(config), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return siteEnv{globalPath: global, builtins: siteBuiltins}, cfg
}

func idsOf(l []sitetemplates.Template) string {
	var ids []string
	for _, t := range l {
		ids = append(ids, t.ID)
	}
	return strings.Join(ids, ",")
}

func TestSiteTemplatesService(t *testing.T) {
	cfg := t.TempDir()
	z, err := newZen(zenOptions{Registry: plugins.NewRegistry(), GlobalPath: filepath.Join(cfg, "config.yml"), Events: &sink{}, Dialogs: &fakeDialogs{}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(z.close)
	got, _ := z.SiteTemplates()
	if len(got) != 3 || got[0].ID != "techdocs" || got[1].ID != "learning" || got[2].ID != "planner" {
		t.Fatalf("templates: %+v", got)
	}
	if got[0].Name != "Technical documentation" || len(got[0].Ask) != 3 || got[0].Ask[0].Key != "title" || got[0].Ask[0].Default != "{{folder}}" || !got[0].Builtin || got[0].Error != "" {
		t.Errorf("techdocs: %+v", got[0])
	}
	// They were installed for editing, next to config.yml.
	if _, err := os.Stat(filepath.Join(cfg, "templates", "blueprints", "techdocs", "blueprint.yml")); err != nil {
		t.Errorf("not installed: %v", err)
	}
	// The JSON the frontend reads.
	b, _ := json.Marshal(got[0])
	for _, want := range []string{`"id":"techdocs"`, `"ask":[{"key":"title","label":"Site title","default":"{{folder}}"}`, `"builtin":true`, `"update":""`, `"error":""`} {
		if !strings.Contains(string(b), want) {
			t.Errorf("json lacks %s: %s", want, b)
		}
	}
	// Edits are picked up; an invalid template is listed with its error.
	if err := os.WriteFile(filepath.Join(cfg, "templates", "blueprints", "planner", "blueprint.yml"), []byte("id: planner\nname: My planner\norder: 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(cfg, "templates", "blueprints", "broken"), 0o755); err != nil {
		t.Fatal(err)
	}
	got, _ = z.SiteTemplates()
	if got[0].Name != "My planner" || got[len(got)-1].ID != "broken" || got[len(got)-1].Error == "" {
		t.Errorf("edited: %+v", got)
	}
	// No settings folder at all: the presets come straight from zen.
	z2, _, _, _ := newTestZen(t, "", "", "")
	if l, _ := z2.SiteTemplates(); len(l) != 3 {
		t.Errorf("without settings folder: %+v", l)
	}
}

func TestTemplatesDirFromConfig(t *testing.T) {
	custom := filepath.Join(t.TempDir(), "my templates")
	env, cfg := siteRig(t, "templates:\n  dir: "+custom+"\n")
	l := env.list()
	if l.Dir != custom || len(l.Templates) != 3 || l.Note != "" {
		t.Fatalf("custom dir: %+v", l)
	}
	if _, err := os.Stat(filepath.Join(custom, "blueprints", "learning", "blueprint.yml")); err != nil {
		t.Error(err)
	}
	if _, err := os.Stat(filepath.Join(cfg, "templates")); err == nil {
		t.Error("the default folder was used as well")
	}
}

func TestUnwritableTemplatesDir(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root writes everywhere")
	}
	base := t.TempDir()
	if err := os.Chmod(base, 0o500); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(base, 0o700)
	env := siteEnv{globalPath: filepath.Join(base, "config.yml"), builtins: siteBuiltins}
	l := env.list()
	if idsOf(l.Templates) != "techdocs,learning,planner" || !strings.Contains(l.Note, "cannot be written to") {
		t.Errorf("fallback: %s %q", idsOf(l.Templates), l.Note)
	}
	// And they work.
	var out, errOut bytes.Buffer
	target := filepath.Join(t.TempDir(), "docs")
	if err := initSite(env.api(), cli.Init{Dir: target, Template: "planner", Values: map[string]string{}}, &out, &errOut); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(errOut.String(), "cannot be written to") || !strings.Contains(out.String(), "Created 8 files") {
		t.Errorf("out %q err %q", out.String(), errOut.String())
	}
}

func TestCreateSite(t *testing.T) {
	z, ev, dl, _ := newTestZen(t, "", "", "")
	_ = ev
	// Relative folders, unknown templates and the templates folder are refused.
	if _, err := z.CreateSite("techdocs", "docs", nil); err == nil {
		t.Error("relative folder accepted")
	}
	if _, err := z.CreateSite("nosuch", t.TempDir(), nil); err == nil || !strings.Contains(err.Error(), "nosuch") {
		t.Errorf("unknown: %v", err)
	}
	cfg := t.TempDir()
	zc, err := newZen(zenOptions{Registry: plugins.NewRegistry(), GlobalPath: filepath.Join(cfg, "config.yml"), Events: &sink{}, Dialogs: &fakeDialogs{}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(zc.close)
	_, _ = zc.SiteTemplates()
	if _, err := zc.CreateSite("techdocs", filepath.Join(cfg, "templates", "blueprints"), nil); err == nil || !strings.Contains(err.Error(), "blueprints folder") {
		t.Errorf("templates folder: %v", err)
	}

	// A folder with spaces and unicode; the result is the new workspace, starting at its home page.
	target := filepath.Join(t.TempDir(), "Dokümente ✨")
	st, err := z.CreateSite("techdocs", target, map[string]string{"title": "Käse", "author": "Ana", "extra": "x"})
	if err != nil {
		t.Fatal(err)
	}
	if st.Workspace == nil || sameDir(st.Workspace.Root, target) == false || st.Page != "index.md" || st.Mode != "read" {
		t.Fatalf("startup: %+v", st)
	}
	if src, err := z.Source("index.md"); err != nil || !strings.Contains(src, "# Käse") {
		t.Errorf("home: %v %q", err, src)
	}
	// The same folder again: every file clashes, nothing changes, the workspace stays.
	if _, err := z.CreateSite("techdocs", target, nil); err == nil || !strings.Contains(err.Error(), "already exist") {
		t.Errorf("second create: %v", err)
	}
	// Into the folder that is open right now, with other files: allowed when nothing clashes.
	busy := t.TempDir()
	if err := os.WriteFile(filepath.Join(busy, "mine.md"), []byte("# Mine\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := z.s.Open(busy); err != nil {
		t.Fatal(err)
	}
	if st, err := z.CreateSite("planner", busy, nil); err != nil || !sameDir(st.Workspace.Root, busy) {
		t.Errorf("into the open folder: %+v %v", st, err)
	} else if b, _ := os.ReadFile(filepath.Join(busy, "mine.md")); string(b) != "# Mine\n" {
		t.Error("existing file changed")
	}

	// The folder panel: the dialogs' PickFolder when there is no special panel.
	dl.folder = "/somewhere"
	if p, err := z.PickSiteFolder(); err != nil || p != "/somewhere" {
		t.Errorf("pick: %q %v", p, err)
	}
}

func TestCreateSiteClosesEditors(t *testing.T) {
	r := newWindowRig(t)
	z := windowsZen(t, r, map[string]string{"README.md": "# Old\n", "a.md": "# A\n", "b.md": "# B\n"})
	r.open(t, "a.md", "b.md")
	target := filepath.Join(t.TempDir(), "new site")

	// An editor keeps its unsaved changes: nothing is created or opened.
	type result struct {
		st  Startup
		err error
	}
	done := make(chan result, 1)
	go func() { st, err := z.CreateSite("learning", target, nil); done <- result{st, err} }()
	waitAsked(t, r, 1)
	r.m.Reply("editor:a.md", desktop.ReplyPending)
	r.m.Reply("editor:a.md", desktop.ReplyCancel)
	res := <-done
	if res.err == nil || !strings.Contains(res.err.Error(), "nothing was created") || !strings.Contains(res.err.Error(), "unsaved") {
		t.Fatalf("cancelled: %+v", res)
	}
	if _, err := os.Stat(filepath.Join(target, "index.md")); err == nil {
		t.Error("the site was written although an editor kept its changes")
	}
	if z.Startup().Workspace.Name == filepath.Base(target) {
		t.Error("the folder was opened although an editor said no")
	}

	// Every editor closes: the new folder replaces the old one.
	target2 := filepath.Join(t.TempDir(), "second")
	go func() { st, err := z.CreateSite("learning", target2, nil); done <- result{st, err} }()
	waitAsked(t, r, 2)
	r.m.Reply("editor:a.md", desktop.ReplyClose)
	waitAsked(t, r, 3)
	r.m.Reply("editor:b.md", desktop.ReplyClose)
	res = <-done
	if res.err != nil || res.st.Workspace == nil || !sameDir(res.st.Workspace.Root, target2) || res.st.Page != "index.md" || r.m.EditorCount() != 0 {
		t.Fatalf("closed: %+v editors %d", res, r.m.EditorCount())
	}
}

func TestRevealTemplates(t *testing.T) {
	var shown []string
	old := revealInFinder
	revealInFinder = func(d string) error { shown = append(shown, d); return nil }
	t.Cleanup(func() { revealInFinder = old })
	cfg := t.TempDir()
	z, err := newZen(zenOptions{Registry: plugins.NewRegistry(), GlobalPath: filepath.Join(cfg, "config.yml"), Events: &sink{}, Dialogs: &fakeDialogs{}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(z.close)
	if err := z.RevealTemplates(); err != nil || len(shown) != 1 || shown[0] != filepath.Join(cfg, "templates", "blueprints") {
		t.Errorf("reveal: %v %v", shown, err)
	}
	if _, err := os.Stat(filepath.Join(cfg, "templates", "blueprints", "techdocs")); err != nil {
		t.Error("the folder has no templates in it")
	}
	z2, _, _, _ := newTestZen(t, "", "", "")
	if err := z2.RevealTemplates(); err == nil {
		t.Error("no settings folder: want an error")
	}
}

func TestInitSite(t *testing.T) {
	env, cfg := siteRig(t, "")
	run := func(i cli.Init) (string, string, error) {
		var out, errOut bytes.Buffer
		if i.Values == nil {
			i.Values = map[string]string{}
		}
		err := initSite(env.api(), i, &out, &errOut)
		return out.String(), errOut.String(), err
	}
	// A private working folder: the summary names the target relative to it,
	// and nothing is created in the source tree (where go list may be looking).
	cwd := t.TempDir()
	t.Chdir(cwd)

	out, _, err := run(cli.Init{List: true})
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if err != nil || len(lines) != 3 || !strings.HasPrefix(lines[0], "techdocs  Technical documentation — ") || !strings.HasPrefix(lines[1], "learning  Learning path — ") {
		t.Fatalf("list: %q %v", out, err)
	}
	if out, _, err := run(cli.Init{ShowDir: true}); err != nil || out != filepath.Join(cfg, "templates", "blueprints")+"\n" {
		t.Errorf("dir: %q %v", out, err)
	}

	// zenmd blueprint new: a commented starter, or a copy of another blueprint; it shows up in the list and builds.
	out, _, err = run(cli.Init{New: true, Template: "mine"})
	if err != nil || !strings.Contains(out, "Made the blueprint mine in "+filepath.Join(cfg, "templates", "blueprints", "mine")) || !strings.Contains(out, "zenmd init --blueprint mine") {
		t.Fatalf("blueprint new: %q %v", out, err)
	}
	if b, err := os.ReadFile(filepath.Join(cfg, "templates", "blueprints", "mine", "blueprint.yml")); err != nil || !strings.Contains(string(b), "id: mine") {
		t.Errorf("blueprint.yml: %v %s", err, b)
	}
	if _, _, err := run(cli.Init{New: true, Template: "mine"}); err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Errorf("second new: %v", err)
	}
	if _, _, err := run(cli.Init{New: true, Template: "tech2", From: "techdocs"}); err != nil {
		t.Errorf("new --from: %v", err)
	}
	if _, _, err := run(cli.Init{New: true, Template: "x", From: "nosuch"}); err == nil || !strings.Contains(err.Error(), "nosuch") {
		t.Errorf("new --from unknown: %v", err)
	}
	if out, _, err = run(cli.Init{Dir: filepath.Join(cwd, "from-mine"), Template: "mine"}); err != nil || !strings.Contains(out, "Created ") {
		t.Errorf("init from the new blueprint: %q %v", out, err)
	}
	if out, _, _ = run(cli.Init{List: true}); !strings.Contains(out, "mine ") || !strings.Contains(out, "tech2") {
		t.Errorf("list after new: %q", out)
	}
	os.RemoveAll(filepath.Join(cfg, "templates", "blueprints", "mine"))
	os.RemoveAll(filepath.Join(cfg, "templates", "blueprints", "tech2"))

	// Create: the summary names the folder as the user would type it.
	target, err := os.MkdirTemp(cwd, "zen-init-test-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(target) })
	rel := filepath.Base(target)
	out, _, err = run(cli.Init{Dir: target, Template: "techdocs", Values: map[string]string{"title": "T"}})
	if err != nil || out != "Created 24 files in "+rel+" (Technical documentation). Open it with: zenmd "+rel+"\n" {
		t.Fatalf("create: %q %v", out, err)
	}
	// A clash is an error that lists the files; nothing is overwritten.
	if _, _, err := run(cli.Init{Dir: target, Template: "techdocs"}); err == nil || !errors.Is(err, sitetemplates.ErrClash) || !strings.Contains(err.Error(), "index.md") {
		t.Errorf("clash: %v", err)
	}
	// A folder with a space is quoted in the hint, and one outside the current folder is shown in full.
	spaced := filepath.Join(t.TempDir(), "my docs")
	out, _, err = run(cli.Init{Dir: spaced, Template: "planner"})
	if err != nil || !strings.Contains(out, "Created 8 files in "+spaced+" (Planner). Open it with: zenmd '"+spaced+"'") {
		t.Errorf("spaced: %q %v", out, err)
	}
	// An unknown template is a usage error naming the choices; the templates folder is not a target.
	_, _, err = run(cli.Init{Dir: t.TempDir(), Template: "nosuch"})
	if err == nil || !strings.Contains(err.Error(), `"nosuch"`) || !strings.Contains(err.Error(), "techdocs, learning, planner") {
		t.Errorf("unknown: %v", err)
	}
	var o, e bytes.Buffer
	code := cli.Run([]string{"init", "site", "--nosuch", t.TempDir()}, cwd, &o, &e, func(cli.Launch) error { return nil }, nil,
		func(i cli.Init, so, se io.Writer) error { return initSite(env.api(), i, so, se) }, nil)
	if code != 2 || !strings.Contains(e.String(), "Usage:") {
		t.Errorf("unknown template through the CLI: %d %q", code, e.String())
	}
	if _, _, err := run(cli.Init{Dir: filepath.Join(cfg, "templates", "blueprints", "x"), Template: "techdocs"}); err == nil || !strings.Contains(err.Error(), "blueprints folder") {
		t.Errorf("into templates: %v", err)
	}
	// An invalid template cannot be used, and is marked in the list.
	if err := os.MkdirAll(filepath.Join(cfg, "templates", "blueprints", "bad"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, _, err := run(cli.Init{Dir: t.TempDir(), Template: "bad"}); err == nil || !strings.Contains(err.Error(), "not valid") {
		t.Errorf("invalid: %v", err)
	}
	if out, _, _ := run(cli.Init{List: true}); !strings.Contains(out, "bad       (invalid: blueprint.yml is missing") {
		t.Errorf("list with invalid: %q", out)
	}

	// Updates: a newer built-in is listed, and written next to the copy on request.
	newer := fstest.MapFS{}
	for _, id := range []string{"techdocs", "learning", "planner"} {
		data, _ := fs.ReadFile(siteBuiltins, id+"/blueprint.yml")
		if id == "techdocs" {
			data = []byte(strings.Replace(string(data), "version: 2.2.0", "version: 2.3.0", 1))
		}
		newer[id+"/blueprint.yml"] = &fstest.MapFile{Data: data}
	}
	env.builtins = newer
	if out, _, _ := run(cli.Init{List: true}); !strings.Contains(out, "[update available: 2.3.0, see zenmd init --update techdocs]") {
		t.Errorf("update marker: %q", out)
	}
	out, _, err = run(cli.Init{Update: true, Template: "techdocs"})
	if err != nil || !strings.Contains(out, "Updated techdocs in place: "+filepath.Join(cfg, "templates", "blueprints", "techdocs")) {
		t.Errorf("update: %q %v", out, err)
	}
	if _, _, err := run(cli.Init{Update: true, Template: "techdocs"}); err == nil {
		t.Error("second update accepted")
	}
	// A copy with edits is never touched: the newer one is written beside it.
	if err := os.WriteFile(filepath.Join(cfg, "templates", "blueprints", "techdocs", "mine.md"), []byte("mine"), 0o644); err != nil {
		t.Fatal(err)
	}
	newer["techdocs/blueprint.yml"] = &fstest.MapFile{Data: []byte(strings.Replace(string(newer["techdocs/blueprint.yml"].Data), "version: 2.3.0", "version: 2.4.0", 1))}
	out, _, err = run(cli.Init{Update: true, Template: "techdocs"})
	if err != nil || !strings.Contains(out, filepath.Join(cfg, "templates", "blueprints", "techdocs.new")) || !strings.Contains(out, "has edits") {
		t.Errorf("update of an edited copy: %q %v", out, err)
	}
	os.RemoveAll(filepath.Join(cfg, "templates", "blueprints", "techdocs.new"))
}

// presetFiles is what each preset creates for the title "Acme Docs".
func presetFiles(id, date string) []string {
	files := map[string][]string{
		"techdocs": {".zen/templates/explanation.yml", ".zen/templates/how-to.yml", ".zen/templates/overview.yml", ".zen/templates/reference.yml", ".zen/templates/release-notes.yml",
			".zen/templates/term.yml", ".zen/templates/tutorial.yml", ".zenmd.yml", "CONTRIBUTING.md", "index.md", "presentation.md",
			"explanation/architecture.md", "explanation/index.md", "getting-started/first-steps.md", "getting-started/index.md", "getting-started/installation.md",
			"glossary/example-term.md", "glossary/index.md", "how-tos/example-task.md", "how-tos/index.md",
			"reference/configuration.md", "reference/index.md", "releases/0.1.0.md", "releases/index.md"},
		"learning": {".zenmd.yml", "index.md", "authoring.md", "glossary.md", "paths/first-path/index.md"},
		"planner": {".zenmd.yml", "index.md", "decisions/0001-record-decisions.md", "decisions/template.md", "ideas.md", "meetings/template.md", "roadmap.md",
			"weeks/" + date + ".md"},
	}[id]
	sort.Strings(files)
	return files
}

// ignoreMissingLayouts drops the warning about the term and glossary layouts,
// which a separate change adds; techdocs names them already.
func ignoreMissingLayouts(ws []string) []string {
	var out []string
	for _, w := range ws {
		if strings.Contains(w, "layout") && (strings.Contains(w, "term") || strings.Contains(w, "glossary")) {
			continue
		}
		out = append(out, w)
	}
	return out
}

func TestPresets(t *testing.T) {
	date := time.Now().Format("2006-01-02")
	layoutsWant := map[string]map[string]string{
		"techdocs": {"index.md": "docs", "getting-started/index.md": "landing", "getting-started/installation.md": "docs", "presentation.md": "slides", "CONTRIBUTING.md": "docs", "explanation/architecture.md": "docs"},
		"learning": {"index.md": "landing", "paths/first-path/index.md": "course", "glossary.md": "article", "authoring.md": "article"},
		"planner":  {"index.md": "landing", "roadmap.md": "wide", "weeks/" + date + ".md": "article", "decisions/0001-record-decisions.md": "article"},
	}
	cards := map[string]int{"techdocs": -1, "learning": 3, "planner": 5}
	for _, id := range []string{"techdocs", "learning", "planner"} {
		t.Run(id, func(t *testing.T) {
			env, _ := siteRig(t, "")
			tmpl, _, ok := env.find(id)
			if !ok || tmpl.Err != "" {
				t.Fatalf("not listed: %+v", tmpl)
			}
			target := filepath.Join(t.TempDir(), "Acme Docs")
			res, err := sitetemplates.Create(tmpl, target, map[string]string{"title": "Acme Docs", "author": "Ana"})
			if err != nil {
				t.Fatal(err)
			}
			got := append([]string(nil), res.Files...)
			sort.Strings(got)
			if strings.Join(got, "\n") != strings.Join(presetFiles(id, date), "\n") {
				t.Fatalf("files:\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(presetFiles(id, date), "\n"))
			}
			// A template brings no code: no plugins or layouts of its own, and no placeholder is left over.
			_ = filepath.Walk(target, func(p string, fi os.FileInfo, err error) error {
				if err != nil || fi.IsDir() {
					return nil
				}
				rel, _ := filepath.Rel(target, p)
				if strings.HasPrefix(rel, ".zen"+string(filepath.Separator)) {
					if !strings.HasPrefix(filepath.ToSlash(rel), ".zen/templates/") {
						t.Errorf("preset writes into .zen/: %s", rel)
					}
					return nil // page templates keep their own placeholders
				}
				b, _ := os.ReadFile(p)
				if m := regexp.MustCompile(`\{\{[a-z]+\}\}`).Find(b); m != nil {
					t.Errorf("%s: placeholder %s left", rel, m)
				}
				return nil
			})
			if b, _ := os.ReadFile(filepath.Join(target, "index.md")); !strings.Contains(string(b), "# Acme Docs") {
				t.Errorf("home does not use the title: %s", b)
			}

			// It opens, and every page renders with no warning and no broken link.
			z, err := newZen(zenOptions{Registry: session.DefaultPlugins(), ConfigDir: t.TempDir(), Dir: target, Events: &sink{}, Dialogs: &fakeDialogs{}})
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(z.close)
			var files []string
			_ = filepath.WalkDir(target, func(p string, d fs.DirEntry, err error) error {
				if err != nil {
					return nil
				}
				if d.IsDir() && strings.HasPrefix(d.Name(), ".") && p != target { // .zen/templates holds yml, not pages
					return filepath.SkipDir
				}
				if rel, err := filepath.Rel(target, p); err == nil && !d.IsDir() && strings.HasSuffix(p, ".md") {
					files = append(files, filepath.ToSlash(rel))
				}
				return nil
			})
			if want := len(presetFiles(id, date)) - map[string]int{"techdocs": 8, "learning": 1, "planner": 1}[id]; len(files) != want { // the .zenmd.yml and the page templates are not pages
				t.Errorf("pages: %v", files)
			}
			for _, p := range files {
				pg, err := z.Page(p)
				if err != nil {
					t.Errorf("%s: %v", p, err)
					continue
				}
				if w := ignoreMissingLayouts(pg.Warnings); len(w) > 0 {
					t.Errorf("%s: warnings %v", p, w)
				}
				if n := strings.Count(pg.HTML, `data-zen-link="broken"`); n != 0 {
					t.Errorf("%s: %d broken links", p, n)
				}
				if want, ok := layoutsWant[id][p]; ok && pg.Layout.Name != want {
					t.Errorf("%s: layout %q, want %q", p, pg.Layout.Name, want)
				}
			}
			home, _ := z.Page("index.md")
			if n := strings.Count(home.HTML, `class="zl-card"`); cards[id] >= 0 && n != cards[id] {
				t.Errorf("home has %d cards, want %d", n, cards[id])
			}
			if tr := z.s.Workspace().Tree; tr[len(tr)-1].Title == "Other pages" {
				other := tr[len(tr)-1]
				// The glossary's terms are listed by the glossary layout, and the sidebar will
				// follow nav.sort; until it does, they sit under "Other pages".
				for _, c := range other.Children {
					if id != "techdocs" || c.Path != "glossary" {
						t.Errorf("pages nothing links to: %+v", c)
					}
				}
			}
			switch id {
			case "learning":
				pg, _ := z.Page("paths/first-path/index.md")
				if strings.Count(pg.HTML, `data-zen-name="quiz"`) != 1 || strings.Count(pg.HTML, `data-zen-name="check"`) != 1 {
					t.Errorf("course page lacks its quiz or checklist: %s", pg.HTML)
				}
				if strings.Count(pg.HTML, "<h2") != 4 {
					t.Errorf("a course of four steps has %d headings", strings.Count(pg.HTML, "<h2"))
				}
			case "planner":
				pg, _ := z.Page("roadmap.md")
				if strings.Count(pg.HTML, `class="zen-diagram"`) != 1 || !strings.Contains(pg.HTML, "<table>") {
					t.Errorf("roadmap lacks its chart or table")
				}
			}

			// It builds with no warnings.
			cfg2 := t.TempDir()
			var out, errOut bytes.Buffer
			err = session.WriteSite(session.SiteEnv{GlobalPath: filepath.Join(cfg2, "config.yml"), Runtime: stubAssets()}, session.SiteOptions{Dir: target}, &out, &errOut)
			stderr, stdout := errOut.String(), out.String()
			if err != nil || stderr != "" || !strings.HasPrefix(stdout, "Built ") || strings.Contains(stdout, "warning") {
				t.Errorf("build: %v\nstdout %q\nstderr %q", err, out.String(), errOut.String())
			}
		})
	}
}

// TestPresetsLint runs Vale with the demo's vocabulary over each preset. It
// needs the vale helper that `task` fetches into build/darwin/helpers.
func TestPresetsLint(t *testing.T) {
	_, file, _, _ := runtime.Caller(0)
	root := filepath.Join(filepath.Dir(file), "..", "..")
	vale := filepath.Join(root, "build", "darwin", "helpers", runtime.GOARCH, "vale")
	if fi, err := os.Stat(vale); err != nil || fi.IsDir() {
		t.Skip("no vale helper in build/darwin/helpers")
	}
	demo := filepath.Join(root, "examples", "showcase")
	linter := lint.New(vale)
	for _, id := range []string{"techdocs", "learning", "planner"} {
		env, _ := siteRig(t, "")
		tmpl, _, _ := env.find(id)
		target := filepath.Join(t.TempDir(), "docs")
		res, err := sitetemplates.Create(tmpl, target, map[string]string{"title": "Acme Docs", "author": "Ana"})
		if err != nil {
			t.Fatal(err)
		}
		// The demo's Vale settings and vocabulary.
		copyDemo := func(rel string) {
			b, err := os.ReadFile(filepath.Join(demo, rel))
			if err != nil {
				t.Fatal(err)
			}
			dst := filepath.Join(target, rel)
			_ = os.MkdirAll(filepath.Dir(dst), 0o755)
			_ = os.WriteFile(dst, b, 0o644)
		}
		copyDemo(".vale.ini")
		copyDemo(".vale/styles/config/vocabularies/Zen/accept.txt")
		for _, f := range res.Files {
			if !strings.HasSuffix(f, ".md") {
				continue
			}
			content, _ := os.ReadFile(filepath.Join(target, f))
			alerts, err := linter.Lint(context.Background(), target, f, content)
			if err != nil {
				t.Fatalf("%s/%s: %v", id, f, err)
			}
			for _, a := range alerts {
				if a.Severity == "error" || a.Severity == "warning" {
					t.Errorf("%s/%s:%d: %s (%s)", id, f, a.Line, a.Message, a.Check)
				}
			}
		}
	}
}

// sameDir compares two folders after resolving links (macOS temp folders are links).
func sameDir(a, b string) bool {
	ra, _ := filepath.EvalSymlinks(a)
	rb, _ := filepath.EvalSymlinks(b)
	return ra != "" && ra == rb
}

// zenWith is an app over a folder with a settings folder of its own (no real one is ever touched).
func zenWith(t *testing.T, cfg string, files map[string]string) (*Zen, string) {
	t.Helper()
	dir := tree(t, files)
	z, err := newZen(zenOptions{Registry: plugins.NewRegistry(), Dir: dir, Mode: "read", GlobalPath: filepath.Join(cfg, "config.yml"), Events: &sink{}, Dialogs: &fakeDialogs{}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(z.close)
	return z, dir
}

// An existing templates/sites is moved to templates/blueprints on first use, with a notice under Problems.
func TestSitesFolderMovesOnFirstUse(t *testing.T) {
	cfg := t.TempDir()
	old := filepath.Join(cfg, "templates", "sites")
	if err := os.MkdirAll(filepath.Join(old, "mine"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(old, "mine", "blueprint.yml"), []byte("id: mine\nname: Mine\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	z, _ := zenWith(t, cfg, docs)
	got, err := z.SiteTemplates()
	if err != nil || len(got) != 4 {
		t.Fatalf("list: %+v %v", got, err)
	}
	if _, err := os.Stat(filepath.Join(cfg, "templates", "blueprints", "mine", "blueprint.yml")); err != nil {
		t.Errorf("not moved: %v", err)
	}
	if _, err := os.Stat(old); err == nil {
		t.Error("sites is still there")
	}
	p, err := z.Page("README.md")
	if err != nil || !slices.Contains(p.Warnings, "templates/sites is now templates/blueprints") {
		t.Errorf("Problems: %v %v", p.Warnings, err)
	}
	// And the command line says it on stderr, once.
	cfg2 := t.TempDir()
	if err := os.MkdirAll(filepath.Join(cfg2, "templates", "sites", "mine"), 0o755); err != nil {
		t.Fatal(err)
	}
	env := siteEnv{globalPath: filepath.Join(cfg2, "config.yml"), builtins: siteBuiltins}
	var out, errOut bytes.Buffer
	if err := initSite(env.api(), cli.Init{List: true, Values: map[string]string{}}, &out, &errOut); err != nil || !strings.Contains(errOut.String(), "zenmd: templates/sites is now templates/blueprints") {
		t.Errorf("stderr %q %v", errOut.String(), err)
	}
	errOut.Reset()
	if err := initSite(env.api(), cli.Init{List: true, Values: map[string]string{}}, &out, &errOut); err != nil || strings.Contains(errOut.String(), "templates/sites") {
		t.Errorf("second run: %q %v", errOut.String(), err)
	}
}

func TestPageTemplatesService(t *testing.T) {
	cfg := t.TempDir()
	z, dir := zenWith(t, cfg, map[string]string{"README.md": "# Home\n", ".zenmd.yml": "templates:\n  \"how-tos/**\": how-to\n  \"guides/**\": how-to\n"})
	list, err := z.PageTemplates()
	if err != nil || len(list) != 7 {
		t.Fatalf("list: %d %v", len(list), err)
	}
	by := map[string]PageTemplate{}
	for _, p := range list {
		by[p.ID] = p
	}
	h := by["how-to"]
	if h.Source != "yours" || h.Path != filepath.Join(cfg, "templates", "pages", "how-to.yml") || h.Version != "1.0.0" || h.Layout != "docs" || len(h.Fields) != 1 || h.Fields[0].Name != "title" || !h.Fields[0].Required || !strings.Contains(h.Skeleton, "## Steps") {
		t.Errorf("how-to: %+v", h)
	}
	if strings.Join(h.Rules, ",") != "how-tos/**,guides/**" || len(by["term"].Rules) != 0 {
		t.Errorf("rules: %v %v", h.Rules, by["term"].Rules)
	}
	b, _ := json.Marshal(h)
	for _, want := range []string{`"id":"how-to"`, `"source":"yours"`, `"rules":["how-tos/**","guides/**"]`, `"outline":[]`, `"update":""`, `"error":""`} {
		if !strings.Contains(string(b), want) {
			t.Errorf("json lacks %s: %s", want, b)
		}
	}
	// Copy to This Folder: written once, never replaced.
	rel, err := z.CopyPageTemplate("term")
	if err != nil || rel != ".zen/templates/term.yml" {
		t.Fatalf("copy: %q %v", rel, err)
	}
	if b, err := os.ReadFile(filepath.Join(dir, ".zen", "templates", "term.yml")); err != nil || !strings.Contains(string(b), "id: term") {
		t.Errorf("copied file: %v", err)
	}
	if _, err := z.CopyPageTemplate("term"); err == nil || !strings.Contains(err.Error(), "already has") {
		t.Errorf("second copy: %v", err)
	}
	for _, bad := range []string{"nosuch", "../x", ""} {
		if _, err := z.CopyPageTemplate(bad); err == nil {
			t.Errorf("copy %q accepted", bad)
		}
	}
	list, _ = z.PageTemplates()
	for _, p := range list {
		if p.ID == "term" && p.Source != "folder" {
			t.Errorf("term source %q", p.Source)
		}
	}
	// A folder's copy beats the one on this Mac, and Open in Editor opens its file.
	realDir, _ := filepath.EvalSymlinks(dir)
	var opened []string
	old := openFile
	openFile = func(p string) error { opened = append(opened, p); return nil }
	t.Cleanup(func() { openFile = old })
	if err := z.OpenPageTemplate("term"); err != nil || len(opened) != 1 || opened[0] != filepath.Join(realDir, ".zen", "templates", "term.yml") {
		t.Errorf("open: %v %v", opened, err)
	}
	if err := z.OpenPageTemplate("nosuch"); err == nil {
		t.Error("open unknown")
	}
	// Show in Finder reveals templates/pages.
	var shown []string
	oldR := revealInFinder
	revealInFinder = func(d string) error { shown = append(shown, d); return nil }
	t.Cleanup(func() { revealInFinder = oldR })
	if err := z.RevealPageTemplates(); err != nil || len(shown) != 1 || shown[0] != filepath.Join(cfg, "templates", "pages") {
		t.Errorf("reveal: %v %v", shown, err)
	}
}

func TestUpdateTemplateService(t *testing.T) {
	cfg := t.TempDir()
	z, _ := zenWith(t, cfg, docs)
	// An edited copy gets .new, an untouched one is replaced in place: here nothing is newer yet.
	if _, err := z.UpdateTemplate("page", "how-to"); err == nil || !strings.Contains(err.Error(), "up to date") {
		t.Errorf("page: %v", err)
	}
	if _, err := z.UpdateTemplate("blueprint", "techdocs"); err == nil || !strings.Contains(err.Error(), "up to date") {
		t.Errorf("blueprint: %v", err)
	}
	if _, err := z.UpdateTemplate("layout", "x"); err == nil {
		t.Error("kind accepted")
	}
	// Pretend the install was an older version: the record says 0.9.0 with this copy's fingerprint.
	pages := filepath.Join(cfg, "templates", "pages")
	data, _ := os.ReadFile(filepath.Join(pages, "term.yml"))
	old := strings.Replace(string(data), "version: 1.0.0", "version: 0.9.0", 1)
	if err := os.WriteFile(filepath.Join(pages, "term.yml"), []byte(old), 0o644); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256([]byte(old))
	rec := "installed:\n    term: 0.9.0\n    how-to: 1.0.0\nfingerprints:\n    term: sha256:" + hex.EncodeToString(sum[:]) + "\n"
	if err := os.WriteFile(filepath.Join(pages, ".zen-templates.yml"), []byte(rec), 0o644); err != nil {
		t.Fatal(err)
	}
	list, _ := z.PageTemplates()
	for _, p := range list {
		if p.ID == "term" && p.Update != "1.0.0" {
			t.Errorf("update flag: %+v", p)
		}
	}
	res, err := z.UpdateTemplate("page", "term")
	if err != nil || !res.InPlace || res.Path != filepath.Join(pages, "term.yml") {
		t.Fatalf("in place: %+v %v", res, err)
	}
	if b, _ := os.ReadFile(filepath.Join(pages, "term.yml")); !strings.Contains(string(b), "version: 1.0.0") {
		t.Error("not replaced")
	}
}

func TestTemplatesCommand(t *testing.T) {
	env, cfg := siteRig(t, "")
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".zen", "templates"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".zen", "templates", "lesson.yml"), []byte("id: lesson\nname: Lesson\nlayout: docs\nfields:\n  title: { required: true }\nskeleton: |\n  # {{title}}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".zenmd.yml"), []byte("templates:\n  \"lessons/**\": lesson\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run := func(i cli.Init) (string, string, error) {
		var out, errOut bytes.Buffer
		i.Values, i.Pages, i.Dir = map[string]string{}, true, root
		err := initSite(env.api(), i, &out, &errOut)
		return out.String(), errOut.String(), err
	}
	out, _, err := run(cli.Init{List: true})
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 8 || !strings.HasPrefix(lines[0], "lesson") || !strings.Contains(lines[0], "this folder") {
		t.Errorf("list:\n%s", out)
	}
	if !strings.Contains(out, "how-to") || !strings.Contains(out, "yours") || !strings.Contains(out, "1.0.0") {
		t.Errorf("list lacks the installed ones:\n%s", out)
	}
	out, _, err = run(cli.Init{Show: true, Template: "how-to"})
	for _, want := range []string{"how-to: How-to guide", "Layout: docs", "title", "required", "Skeleton", "  ## Steps"} {
		if err != nil || !strings.Contains(out, want) {
			t.Errorf("show lacks %q:\n%s %v", want, out, err)
		}
	}
	if out, _, _ = run(cli.Init{Show: true, Template: "lesson"}); !strings.Contains(out, "Rules here: lessons/**") {
		t.Errorf("rules:\n%s", out)
	}
	if _, _, err := run(cli.Init{Show: true, Template: "nosuch"}); err == nil || !strings.Contains(err.Error(), "nosuch") {
		t.Errorf("unknown: %v", err)
	}
	if out, _, err := run(cli.Init{ShowDir: true}); err != nil || out != filepath.Join(cfg, "templates")+"\n" {
		t.Errorf("dir: %q %v", out, err)
	}
	if _, _, err := run(cli.Init{Update: true, Template: "how-to"}); err == nil || !strings.Contains(err.Error(), "up to date") {
		t.Errorf("update: %v", err)
	}
	// zenmd init --dir is the blueprints folder.
	var o, e bytes.Buffer
	if err := initSite(env.api(), cli.Init{ShowDir: true, Values: map[string]string{}}, &o, &e); err != nil || o.String() != filepath.Join(cfg, "templates", "blueprints")+"\n" {
		t.Errorf("init --dir: %q %v", o.String(), err)
	}
}
