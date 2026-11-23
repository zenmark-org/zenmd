// Package app is Zenmark itself: its menu, the Zen service bound to the
// frontend, its dialogs and its settings. The window shell (windows, the close
// handshake, dialogs, print, Finder, the App Store channel) is kit/desktop.
package app

import (
	"errors"
	"io/fs"
	"path/filepath"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/zenmark-org/zenmd/engine/config"
	"github.com/zenmark-org/zenmd/kit/desktop"
)

// Name is the product's name as people see it: window titles, the app menu.
// The command and executable stay `zen`.
const Name = "Zenmark"

// Options configures a GUI run.
type Options struct {
	// Assets is the built frontend.
	Assets fs.FS
	// Dir is the absolute workspace directory to open, "" for none.
	Dir string
	// Page is the workspace-relative page to open, "" for the default.
	Page string
	// File opens the markdown file Dir/Page on its own, without scanning Dir (zenmd notes.md outside a project).
	File bool
	// Mode is "read" or "edit".
	Mode string
}

// startDir is what the session opens at launch: the folder, or the file itself
// when it opens alone (the session's Open takes a markdown file).
func (o Options) startDir() string {
	if o.File && o.Dir != "" && o.Page != "" {
		return filepath.Join(o.Dir, o.Page)
	}
	return o.Dir
}

// Run starts the GUI and blocks until the application exits.
func Run(opts Options) error {
	reg := builtinPlugins()
	globalPath, _ := config.GlobalPath()
	configDir := ""
	if globalPath != "" {
		configDir = filepath.Dir(globalPath)
	}
	sink := desktop.NewSink()
	logs, stopLogs := installLogging(sink)
	defer stopLogs()
	logStartup()                    // the first entry of the Console
	dialogs := desktop.Dialogs(nil) // bound to the application by desktop.Run
	windows := desktop.NewWindows(desktop.WindowsOptions{Title: Name})
	// The main window always reads; `zenmd edit` opens an editor window once
	// the application runs.
	z, err := newZen(zenOptions{
		Registry: reg, GlobalPath: globalPath,
		Dir: opts.startDir(), Page: opts.Page, Mode: "read",
		ConfigDir: configDir,
		Events:    sink, Dialogs: dialogs, Logs: logs,
		DeferOpen: true, // opened once the window can show progress
	})
	if err != nil {
		return err
	}
	defer z.close()
	// Folder-only menu items start enabled only when a folder was given.
	z.menu = newMenuState(opts.Dir != "")
	z.windows = windows

	return desktop.Run(desktop.Options{
		Name:        Name,
		Description: "Markdown reader and editor",
		Assets:      opts.Assets,
		Middleware:  z.middleware,
		Services:    []application.Service{application.NewService(z)},
		Sink:        sink,
		Dialogs:     dialogs,
		Windows:     windows,
		OnCreated: func(a *application.App, _ *application.WebviewWindow) {
			z.open = a.Browser.OpenURL
		},
		OnStarted: func(*application.App, *application.WebviewWindow) error {
			z.openInitial()     // the folder given on the command line (Startup may have done it)
			z.s.PromptPending() // plugins of the folder given on the command line
			z.s.PromptPendingLayouts()
			return z.openStartupEditor(opts.Mode)
		},
		OpenWith: func(path string) (any, error) {
			st, err := z.openAny(path) // a file opens alone unless it is in a project folder
			if errors.Is(err, ErrOpenCancelled) {
				return nil, nil // stopped from the loader
			}
			if err != nil {
				return nil, err
			}
			return st, nil
		},
		OpenedEvent: EventWorkspaceOpened,
		// Menu: see menu.go.
		Menu: func(a *application.App, win *application.WebviewWindow) *application.Menu {
			return buildMenu(a, menuActions{
				NewSite:      showAndEmit(win, sink, EventNewSite),
				BuildWebsite: showAndEmit(win, sink, EventBuildWebsite),
				// Print Page goes to the window that has the focus (main or editor).
				PrintPage: func() { _ = z.PrintPage() },
				PrintSite: func() { _ = z.PrintSite() },
				NewPage:   newPageAction(win, sink),
				// Find goes to the window in front: every window gets the event, the focused one acts.
				Find:         emitAction(sink, EventFind),
				FindNext:     emitAction(sink, EventFindNext),
				FindPrevious: emitAction(sink, EventFindPrevious),
				// File > Open… (⌘O): a folder or a markdown file; a file opens alone.
				Open: func() {
					go func() {
						if err := z.openPicked(); err != nil {
							a.Dialog.Warning().SetTitle("Could not open").SetMessage(err.Error()).Show()
						}
					}()
				},
				// File > Open Enclosing Folder (⇧⌘O): the project folder of the file that is open alone.
				OpenEnclosing: func() {
					go func() {
						if err := z.OpenEnclosingFolder(); err != nil {
							a.Dialog.Warning().SetTitle("Could not open the folder").SetMessage(err.Error()).Show()
						}
					}()
				},
				Settings:    settingsAction(win, sink),
				CloseFolder: showAndEmit(win, sink, EventCloseFolder),
			}, z.menu, desktop.DevBuild)
		},
	})
}
