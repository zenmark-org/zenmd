package desktop

// PickPath is File › Open…: the panel takes a folder or a markdown file, and
// returns the one chosen, "" on cancel. As with PickFolder, choosing one first
// lets editor windows keep unsaved changes (cancelling it). The app decides how
// a file opens (alone, or as part of its project folder).
func (d *DialogSet) PickPath() (string, error) {
	p, err := d.app.Dialog.OpenFile().
		SetTitle("Open").
		SetMessage("Choose a folder or a markdown file").
		CanChooseDirectories(true).
		CanChooseFiles(true).
		CanCreateDirectories(false).
		AddFilter("Markdown", "*.md;*.markdown").
		AttachToWindow(d.win).
		PromptForSingleSelection()
	if err != nil || p == "" {
		return p, err
	}
	if d.beforeOpen != nil && !d.beforeOpen() {
		return "", nil
	}
	return p, nil
}
