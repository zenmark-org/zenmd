package app

import (
	"errors"
	"path/filepath"

	"github.com/zenmark-org/zenmd/internal/cli"
)

// pathPicker is a dialog set whose panel takes a folder or a markdown file
// (File › Open…, ⌘O); kit/desktop's DialogSet is one. Dialogs without it only
// pick folders.
type pathPicker interface{ PickPath() (string, error) }

// launchFor is what `zenmd <path>` (or `zenmd --folder <path>`) would open: the
// same rules as the command line, so File › Open…, Finder's Open With and the
// command agree. A markdown file opens alone unless a project folder is above
// it. Nothing is opened here.
func launchFor(path string, folder bool) (cli.Launch, error) {
	if !filepath.IsAbs(path) {
		return cli.Launch{}, errors.New("not a folder or markdown file")
	}
	l, err := cli.Resolve(path, false, folder)
	if err != nil {
		return cli.Launch{}, errors.New("not a folder or markdown file")
	}
	return l, nil
}

// openLaunch opens what l names, as a folder at its page or as one file alone,
// and returns what the window shows. A newer open replacing it is
// ErrOpenCancelled.
func (z *Zen) openLaunch(l cli.Launch) (Startup, error) {
	var err error
	if l.File {
		err = z.s.OpenFile(filepath.Join(l.Dir, l.Page))
	} else {
		err = z.s.Open(l.Dir)
	}
	if err != nil {
		return Startup{}, z.s.OpenError(err)
	}
	z.setLaunchErr(nil)
	z.folderOpened(true)
	z.page = l.Page
	return z.Startup(), nil
}

// openAny opens a folder or markdown file handed over by macOS (Open With,
// `open -a Zenmark <path>`) or chosen in File › Open…, and returns what the
// frontend should show. Not bound to the frontend.
func (z *Zen) openAny(path string) (Startup, error) {
	l, err := launchFor(path, false)
	if err != nil {
		return Startup{}, err
	}
	return z.openLaunch(l)
}

// openAndEmit opens a folder or a markdown file: a file opens alone, without its
// folder, unless it is in a project folder (a .zenmd.yml or .git
// above it), then the folder opens at that page. path is absolute. The window
// gets workspace:opened. Not bound: the frontend never hands Go a path to open. A newer open that replaces this one is not an error.
func (z *Zen) openAndEmit(path string) (err error) {
	defer z.logCall("Open", &err)
	st, err := z.openAny(path)
	if errors.Is(err, ErrOpenCancelled) {
		return nil
	}
	if err != nil {
		return err
	}
	z.emitOpened(st)
	return nil
}

// openPicked is File › Open… (⌘O): the panel takes a folder or a markdown
// file, and the choice opens as Open does. Cancelling changes nothing.
func (z *Zen) openPicked() error {
	var path string
	var err error
	if p, ok := z.dialogs.(pathPicker); ok {
		path, err = p.PickPath()
	} else {
		path, err = z.dialogs.PickFolder()
	}
	if err != nil || path == "" {
		return err
	}
	return z.openAndEmit(path)
}

// OpenEnclosingFolder opens the project folder of the file that is open on
// its own, at that page: the nearest folder above it with a .zenmd.yml,
// or .git, else the file's own folder (`zenmd <file> --folder`).
// Editor windows close first, each asking about unsaved changes; the window
// gets workspace:opened.
func (z *Zen) OpenEnclosingFolder() (err error) {
	defer z.logCall("OpenEnclosingFolder", &err)
	ws := z.s.Workspace()
	if ws == nil || !ws.Single {
		return errors.New("no file is open on its own")
	}
	l, err := launchFor(filepath.Join(ws.Root, ws.File), true)
	if err != nil {
		return err
	}
	if z.windows != nil && !z.windows.CloseEditorsWait() {
		return nil // an editor kept its unsaved changes
	}
	st, err := z.openLaunch(l)
	if errors.Is(err, ErrOpenCancelled) {
		return nil
	}
	if err != nil {
		return err
	}
	z.emitOpened(st)
	return nil
}

// emitOpened tells the frontend to show st (workspace:opened).
func (z *Zen) emitOpened(st Startup) {
	if z.events != nil {
		z.events.Emit(EventWorkspaceOpened, st)
	}
}
