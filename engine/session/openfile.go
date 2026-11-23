package session

import (
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"

	"github.com/zenmark-org/zenmd/engine/watch"
	"github.com/zenmark-org/zenmd/engine/workspace"
)

// ErrFolderOnly is what a folder-only call answers while one file is open on
// its own (Export › Website, folder settings): the same words as the greyed
// items' tooltip.
var ErrFolderOnly = errors.New("Open a folder to use this")

// OpenFile opens one markdown file on its own, without its folder: the
// workspace has that page only, nothing else in the folder is scanned, and
// only the file is watched. It follows the rules of Open: progress events,
// the newest open wins, and a file that cannot open leaves what was open. The
// folder's .zenmd.yml is ignored (only the global config.yml applies); the
// page's frontmatter still does, and plugin trust is asked as for the folder.
func (s *Session) OpenFile(path string) error { return s.open(path, true) }

// IsSingle reports whether one file is open on its own.
func (s *Session) IsSingle() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.ws != nil && s.ws.Single()
}

// isMarkdownFile reports whether path is an existing regular markdown file.
func isMarkdownFile(path string) bool {
	if path == "" || !isMarkdownPath(path) {
		return false
	}
	fi, err := os.Stat(path)
	return err == nil && fi.Mode().IsRegular()
}

// configRootLocked is the folder whose .zenmd.yml applies: none while one
// file is open on its own.
func (s *Session) configRootLocked() string {
	if s.ws.Single() {
		return ""
	}
	return s.ws.Root()
}

// folderOnly is the guard of what needs a folder: Export › Website, New Page
// and the folder's settings file. It is ErrFolderOnly for one file on its own.
func (s *Session) folderOnly() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ws != nil && s.ws.Single() {
		return ErrFolderOnly
	}
	return nil
}

// watchWorkspace starts the watcher the workspace needs: the folder's, or for
// one file a watcher of that file alone.
func (s *Session) watchWorkspace(ws *workspace.Workspace, async bool) {
	if ws.Single() {
		s.startFileWatch(ws)
		return
	}
	s.startWatch(ws.Root(), async)
}

// startFileWatch watches the one file. It starts at once (one directory, no
// tree to walk); a stop or a newer start drops it.
func (s *Session) startFileWatch(ws *workspace.Workspace) {
	s.watchMu.Lock()
	s.watchGen++
	s.watchMu.Unlock()
	w, err := newFileWatcher(ws.Root(), ws.File(), s.handle)
	if err != nil {
		// The window stays usable without live reload.
		logApp(slog.LevelWarn, "watcher failed to start; changes on disk will not show", "error", scrubRoot(ws.Root(), err.Error()))
		return
	}
	s.watchMu.Lock()
	s.w = w
	s.watchMu.Unlock()
}

// fileWatcher reports changes to one file. It watches the file's folder, not
// recursively (an atomic save replaces the file, which a watch on the file
// itself would lose), and passes on only the events of that file's name.
type fileWatcher struct {
	fw       *fsnotify.Watcher
	file     string
	onEvent  func([]watch.Event)
	done     chan struct{}
	finished chan struct{}
	stopOnce sync.Once
}

const fileDebounce = 100 * time.Millisecond

func newFileWatcher(dir, file string, onEvent func([]watch.Event)) (*fileWatcher, error) {
	fw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	if err := fw.Add(dir); err != nil {
		fw.Close()
		return nil, err
	}
	w := &fileWatcher{fw: fw, file: file, onEvent: onEvent, done: make(chan struct{}), finished: make(chan struct{})}
	go w.loop(dir)
	return w, nil
}

// Stop ends the watch; no callback runs after it returns.
func (w *fileWatcher) Stop() {
	w.stopOnce.Do(func() { close(w.done) })
	<-w.finished
}

func (w *fileWatcher) loop(dir string) {
	defer close(w.finished)
	defer w.fw.Close()
	var timer *time.Timer
	var timerC <-chan time.Time
	kind := watch.Changed
	for {
		select {
		case <-w.done:
			if timer != nil {
				timer.Stop()
			}
			return
		case ev, ok := <-w.fw.Events:
			if !ok {
				return
			}
			if filepath.Base(ev.Name) != w.file || filepath.Dir(ev.Name) != dir || ev.Op == fsnotify.Chmod {
				continue
			}
			if ev.Has(fsnotify.Remove) || ev.Has(fsnotify.Rename) {
				if _, err := os.Lstat(ev.Name); err != nil {
					kind = watch.Removed
				}
			} else {
				kind = watch.Changed
			}
			if timer != nil {
				timer.Stop()
			}
			timer = time.NewTimer(fileDebounce)
			timerC = timer.C
		case _, ok := <-w.fw.Errors:
			if !ok {
				return
			}
			slog.Warn("watcher error on the open file", "source", "app")
		case <-timerC:
			timer, timerC = nil, nil
			if _, err := os.Lstat(filepath.Join(dir, w.file)); err == nil {
				kind = watch.Changed // replaced right after the rename
			}
			select {
			case <-w.done:
				return
			default:
			}
			w.onEvent([]watch.Event{{Path: w.file, Kind: kind}})
		}
	}
}
