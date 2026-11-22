package app

// Reload scans the open folder again (the sidebar's Reload button and ⌘R) and
// returns its workspace; null with no folder open.
func (z *Zen) Reload() (r0 *Workspace, err error) {
	defer z.logCall("Reload", &err)
	return z.s.Reload(), nil
}
