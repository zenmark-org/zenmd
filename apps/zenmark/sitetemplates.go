package app

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/zenmark-org/zenmd/engine/config"
	"github.com/zenmark-org/zenmd/engine/session"
	"github.com/zenmark-org/zenmd/engine/sitetemplates"
	"github.com/zenmark-org/zenmd/internal/cli"
	"github.com/zenmark-org/zenmd/kit/desktop"
)

// EventNewSite asks the main window to show the New Site dialog (File > New
// from Blueprint…, Shift+Cmd+N). No payload.
const EventNewSite = "menu:new-site"

// siteBuiltins are the built-in presets of this app (its assembly); tests replace them.
var siteBuiltins fs.FS = assemblyTemplates()

// revealInFinder shows a folder in Finder; tests replace it.
var revealInFinder = func(dir string) error { return desktop.Reveal(dir) }

// SiteTemplates lists the blueprints (formerly site templates) for the New Site dialog and the
// welcome screen, installing the built-in presets on first use.
func (z *Zen) SiteTemplates() ([]SiteTemplate, error) {
	if err := requireFeature("newsite"); err != nil {
		return nil, err
	}
	return z.s.Templates().List(), nil
}

// siteFolderPicker is the native folder panel with New Folder, for the New
// Site dialog; it opens nothing and closes no editors.
type siteFolderPicker interface{ PickSiteFolder() (string, error) }

// PickSiteFolder shows the folder panel (New Folder enabled) and returns the
// chosen folder, "" on cancel.
func (z *Zen) PickSiteFolder() (r0 string, err error) {
	defer z.logCall("PickSiteFolder", &err)
	if err := requireFeature("newsite"); err != nil {
		return "", err
	}
	if p, ok := z.dialogs.(siteFolderPicker); ok {
		return p.PickSiteFolder()
	}
	if d, ok := z.dialogs.(*desktop.DialogSet); ok {
		return d.PickFolderTitled("Choose a Folder for the Site")
	}
	return z.dialogs.PickFolder()
}

// CreateSite makes a site from template id in dir (new or existing; nothing
// is overwritten), then opens the folder as Open Folder does: editor windows
// are closed first (each asks about unsaved changes), and the result is what
// the frontend shows.
func (z *Zen) CreateSite(id, dir string, values map[string]string) (r0 Startup, err error) {
	defer z.logCall("CreateSite", &err)
	if err := requireFeature("newsite"); err != nil {
		return Startup{}, err
	}
	plan, err := z.s.Templates().Plan(id, dir)
	if err != nil {
		return Startup{}, err
	}
	// Unsaved editors are asked first, as with Open Folder: nothing is
	// written when one keeps its changes.
	if z.windows != nil && !z.windows.CloseEditorsWait() {
		return Startup{}, errors.New("nothing was created: an editor window kept its unsaved changes")
	}
	if err := plan.Create(values); err != nil {
		return Startup{}, err
	}
	if err := z.s.Open(dir); err != nil {
		return Startup{}, z.s.OpenError(err)
	}
	z.page = ""
	z.setLaunchErr(nil)
	z.folderOpened(true)
	return z.Startup(), nil
}

// TemplatesDir is the templates folder (templates.dir in config.yml, or the
// default next to it), "" when there is none. It holds blueprints/ and pages/.
func (z *Zen) TemplatesDir() (string, error) {
	if err := requireFeature("newsite"); err != nil {
		return "", err
	}
	return z.s.Templates().Dir(), nil
}

// RevealTemplates shows the blueprints folder (templates/blueprints) in Finder, creating it (with the
// built-in presets) when it is not there yet.
func (z *Zen) RevealTemplates() (err error) {
	defer z.logCall("RevealTemplates", &err)
	if err := requireFeature("newsite"); err != nil {
		return err
	}
	dir := z.s.Templates().Dir()
	if dir == "" {
		return errors.New("there is no blueprints folder")
	}
	dir = sitetemplates.BlueprintsDir(dir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return revealInFinder(dir)
}

// PageTemplates lists the page templates for Settings › Templates: the open folder's own, those in
// templates/pages and the built-in ones (installing the built-in ones into templates/pages on first use).
func (z *Zen) PageTemplates() ([]PageTemplate, error) {
	if err := requireFeature("reader"); err != nil {
		return nil, err
	}
	return z.s.Templates().Pages().Templates, nil
}

// UpdateTemplate brings a newer built-in template to the user's copy: kind is "blueprint" or "page".
// An untouched copy is replaced in place; an edited one gets <id>.new (a page template <id>.yml.new) beside it.
func (z *Zen) UpdateTemplate(kind, id string) (r0 TemplateUpdate, err error) {
	defer z.logCall("UpdateTemplate", &err)
	if z.ro {
		return TemplateUpdate{}, session.ErrReadOnly
	}
	var res sitetemplates.UpdateResult
	switch kind {
	case "blueprint":
		if err := requireFeature("newsite"); err != nil {
			return TemplateUpdate{}, err
		}
		res, err = z.s.Templates().Update(id)
	case "page":
		if err := requireFeature("reader"); err != nil {
			return TemplateUpdate{}, err
		}
		res, err = z.s.Templates().UpdatePage(id)
	default:
		return TemplateUpdate{}, errors.New("kind must be blueprint or page")
	}
	return TemplateUpdate{Path: res.Path, InPlace: res.InPlace}, err
}

// RevealPageTemplates shows the page templates folder (templates/pages) in Finder.
func (z *Zen) RevealPageTemplates() (err error) {
	defer z.logCall("RevealPageTemplates", &err)
	if err := requireFeature("reader"); err != nil {
		return err
	}
	dir := z.s.Templates().PagesDir()
	if dir == "" {
		return errors.New("there is no templates folder")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return revealInFinder(dir)
}

// CopyPageTemplate writes page template id to the open folder's .zen/templates/<id>.yml to edit
// there, and returns that path. An existing file is never replaced.
func (z *Zen) CopyPageTemplate(id string) (r0 string, err error) {
	defer z.logCall("CopyPageTemplate", &err)
	if err := requireFeature("editor"); err != nil {
		return "", err
	}
	return z.s.Templates().CopyPage(id)
}

// OpenPageTemplate opens the file of page template id (the copy in templates/pages, or the open
// folder's own) in the editor.
func (z *Zen) OpenPageTemplate(id string) (err error) {
	defer z.logCall("OpenPageTemplate", &err)
	if err := requireFeature("editor"); err != nil {
		return err
	}
	if z.ro {
		return session.ErrReadOnly
	}
	file, err := z.s.Templates().PagePath(id)
	if err != nil {
		return err
	}
	if err := openFile(file); err != nil {
		return errors.New("could not open " + filepath.Base(file))
	}
	return nil
}

// InitSite is `zenmd init` and `zenmd blueprint`: it lists blueprints, makes one, writes a newer built-in beside
// the user's copy, or creates a site, and tells what it did on stdout.
func InitSite(i cli.Init, stdout, stderr io.Writer) error {
	globalPath, _ := config.GlobalPath()
	s := session.New(session.Options{GlobalPath: globalPath, Presets: siteBuiltins})
	defer s.Close()
	return initSite(s.Templates(), i, stdout, stderr)
}

func initSite(tpls session.TemplatesAPI, i cli.Init, stdout, stderr io.Writer) error {
	defer func() { // what happened to the templates folder (the sites move) is told once
		for _, n := range tpls.Notes() {
			fmt.Fprintln(stderr, "zenmd: "+n)
		}
	}()
	if i.Pages {
		return initPages(tpls, i, stdout, stderr)
	}
	if i.ShowDir {
		dir, _ := tpls.Folder()
		if dir == "" {
			return errors.New("there is no blueprints folder")
		}
		fmt.Fprintln(stdout, sitetemplates.BlueprintsDir(dir))
		return nil
	}
	if i.New {
		out, err := tpls.New(i.Template, i.From)
		if err != nil {
			return err
		}
		fmt.Fprintf(stdout, "Made the blueprint %s in %s. Edit its blueprint.yml, then try it with: zenmd init --blueprint %s <folder>\n", i.Template, out, i.Template)
		return nil
	}
	if i.Update {
		res, err := tpls.Update(i.Template)
		if err != nil {
			return err
		}
		fmt.Fprintln(stdout, updateSummary(i.Template, res))
		return nil
	}

	l := tpls.Listing()
	for _, w := range l.Warnings {
		fmt.Fprintln(stderr, "zenmd: "+w)
	}
	if l.Note != "" {
		fmt.Fprintln(stderr, "zenmd: "+l.Note)
	}
	if i.List {
		width := 0
		for _, t := range l.Templates {
			width = max(width, len(t.ID))
		}
		for _, t := range l.Templates {
			switch {
			case t.Err != "":
				fmt.Fprintf(stdout, "%-*s  (invalid: %s)\n", width, t.ID, t.Err)
			default:
				line := fmt.Sprintf("%-*s  %s", width, t.ID, t.Name)
				if t.Description != "" {
					line += " — " + t.Description
				}
				if t.Update != "" {
					line += fmt.Sprintf(" [update available: %s, see zenmd init --update %s]", t.Update, t.ID)
				}
				fmt.Fprintln(stdout, line)
			}
		}
		if len(l.Templates) == 0 {
			fmt.Fprintln(stdout, "No blueprints.")
		}
		return nil
	}

	var t sitetemplates.Template
	found := false
	for _, x := range l.Templates {
		if x.ID == i.Template {
			t, found = x, true
		}
	}
	if !found {
		var ids []string
		for _, x := range l.Templates {
			if x.Err == "" {
				ids = append(ids, x.ID)
			}
		}
		return cli.UsageError(fmt.Sprintf("there is no blueprint %q (blueprints: %s)", i.Template, strings.Join(ids, ", ")))
	}
	if t.Err != "" {
		return fmt.Errorf("blueprint %s is not valid: %s", t.ID, t.Err)
	}
	if l.Holds(i.Dir) {
		return errors.New("choose a folder outside the blueprints folder")
	}
	res, err := sitetemplates.Create(t, i.Dir, i.Values)
	if err != nil {
		return err
	}
	n := fmt.Sprintf("%d files", len(res.Files))
	if len(res.Files) == 1 {
		n = "1 file"
	}
	shown := shownFolder(i.Dir)
	fmt.Fprintf(stdout, "Created %s in %s (%s). Open it with: zenmd %s\n", n, shown, t.Name, shellQuote(shown))
	return nil
}

// shownFolder names dir the short way: relative to the current folder when it
// is inside it ("." for the folder itself).
func shownFolder(dir string) string {
	if cwd, err := os.Getwd(); err == nil {
		if rel, err := filepath.Rel(cwd, dir); err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return rel
		}
	}
	return dir
}

// shellQuote quotes s for a shell only when it needs it.
func shellQuote(s string) string {
	if s != "" && strings.Trim(s, "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789._-/+,:@") == "" {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// updateSummary says what an update did.
func updateSummary(id string, res sitetemplates.UpdateResult) string {
	if res.InPlace {
		return fmt.Sprintf("Updated %s in place: %s. It had no edits of yours.", id, res.Path)
	}
	return fmt.Sprintf("Wrote the newer %s to %s. Your copy has edits, so it was left as it is: compare the two, keep what you like, then delete the new one.", id, res.Path)
}

// initPages is zenmd templates: the page templates (list, show, update) and the templates folder (--dir).
func initPages(tpls session.TemplatesAPI, i cli.Init, stdout, stderr io.Writer) error {
	if i.ShowDir {
		dir, _ := tpls.Folder()
		if dir == "" {
			return errors.New("there is no templates folder")
		}
		fmt.Fprintln(stdout, dir)
		return nil
	}
	if i.Update {
		res, err := tpls.UpdatePage(i.Template)
		if err != nil {
			return err
		}
		fmt.Fprintln(stdout, updateSummary(i.Template, res))
		return nil
	}
	l := tpls.PagesIn(i.Dir) // the folder zenmd is run in: its own .zen/templates show too
	for _, w := range l.Warnings {
		fmt.Fprintln(stderr, "zenmd: "+w)
	}
	if l.Note != "" {
		fmt.Fprintln(stderr, "zenmd: "+l.Note)
	}
	if i.List {
		width := 0
		for _, p := range l.Templates {
			width = max(width, len(p.ID))
		}
		for _, p := range l.Templates {
			if p.Error != "" {
				fmt.Fprintf(stdout, "%-*s  (invalid: %s)  %s\n", width, p.ID, p.Error, p.Path)
				continue
			}
			line := fmt.Sprintf("%-*s  %-9s", width, p.ID, sourceLabel(p.Source))
			if p.Version != "" {
				line += "  " + p.Version
			}
			line += "  " + p.Name
			if p.Update != "" {
				line += fmt.Sprintf("  [update available: %s, see zenmd templates update %s]", p.Update, p.ID)
			}
			fmt.Fprintln(stdout, strings.TrimRight(line, " "))
		}
		if len(l.Templates) == 0 {
			fmt.Fprintln(stdout, "No page templates.")
		}
		return nil
	}
	for _, p := range l.Templates {
		if p.ID != i.Template {
			continue
		}
		if p.Error != "" {
			return fmt.Errorf("page template %s is not valid: %s", p.ID, p.Error)
		}
		showPage(stdout, p)
		return nil
	}
	var ids []string
	for _, p := range l.Templates {
		ids = append(ids, p.ID)
	}
	return cli.UsageError(fmt.Sprintf("there is no page template %q (page templates: %s)", i.Template, strings.Join(ids, ", ")))
}

func sourceLabel(src string) string {
	switch src {
	case sitetemplates.SourceFolder:
		return "this folder"
	case sitetemplates.SourceYours:
		return "yours"
	}
	return "built-in"
}

func showPage(w io.Writer, p PageTemplate) {
	fmt.Fprintf(w, "%s: %s\n", p.ID, p.Name)
	if p.Description != "" {
		fmt.Fprintln(w, p.Description)
	}
	from := sourceLabel(p.Source)
	if p.Path != "" {
		from += ", " + p.Path
	}
	meta := "Layout: " + p.Layout + "\nFrom: " + from
	if p.Version != "" {
		meta += "\nVersion: " + p.Version
	}
	fmt.Fprintln(w, meta)
	if len(p.Rules) > 0 {
		fmt.Fprintln(w, "Rules here: "+strings.Join(p.Rules, ", "))
	}
	fmt.Fprintln(w, "\nFields")
	if len(p.Fields) == 0 {
		fmt.Fprintln(w, "  (none)")
	}
	width := 0
	for _, f := range p.Fields {
		width = max(width, len(f.Name))
	}
	for _, f := range p.Fields {
		line := fmt.Sprintf("  %-*s  %-7s", width, f.Name, f.Type)
		if f.Required {
			line += "  required"
		}
		if f.Default != "" {
			line += "  default " + f.Default
		}
		fmt.Fprintln(w, strings.TrimRight(line, " "))
	}
	if len(p.Outline) > 0 {
		fmt.Fprintln(w, "\nStructure")
		for _, o := range p.Outline {
			fmt.Fprintln(w, "  "+strings.Repeat("  ", o.Depth)+"- "+o.Text)
		}
	}
	fmt.Fprintln(w, "\nSkeleton")
	for _, line := range strings.Split(strings.TrimRight(p.Skeleton, "\n"), "\n") {
		fmt.Fprintln(w, strings.TrimRight("  "+line, " "))
	}
}
