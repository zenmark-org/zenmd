package session

// Reload scans the folder again: config, renderer and index are rebuilt from
// what is on disk now, so files the watcher missed show up (the sidebar's
// Reload button). It returns the workspace, nil with no folder open, and sends
// workspace:changed when the tree changed.
func (s *Session) Reload() *Workspace {
	s.mu.Lock()
	if s.ws == nil {
		s.mu.Unlock()
		return nil
	}
	changed := s.rebuildLocked()
	ws := s.workspaceLocked()
	s.mu.Unlock()
	if changed {
		s.emit(EventWorkspaceChanged, ws)
	}
	return ws
}
