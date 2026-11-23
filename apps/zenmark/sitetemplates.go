package app

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/zenmark-org/zenmd/apps/zenmark/cliset"
	"github.com/zenmark-org/zenmd/engine/session"
	"github.com/zenmark-org/zenmd/engine/sitetemplates"
	"github.com/zenmark-org/zenmd/internal/cli"
	"github.com/zenmark-org/zenmd/kit/desktop"
)

// EventNewSite asks the main window to show the New Site dialog (File > New
// from Blueprint…, Shift+Cmd+N). No payload.
const EventNewSite = "menu:new-site"

// siteBuiltins are the built-in presets of this app (its assembly); tests replace them.
var siteBuiltins fs.FS = cliset.Templates()

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

// InitSite is `zenmd init` and `zenmd blueprint` (the engine's, in cliset), with this app's built-in
// blueprints and the Finder as the reveal.
func InitSite(i cli.Init, stdout, stderr io.Writer) error {
	return cliset.InitWith(siteBuiltins, desktop.Reveal)(i, stdout, stderr)
}

func initSite(tpls session.TemplatesAPI, i cli.Init, stdout, stderr io.Writer) error {
	return cliset.RunInit(tpls, i, stdout, stderr)
}

func sourceLabel(src string, edited bool) string { return cliset.SourceLabel(src, edited) }
