package app

import "sync"

// folderOnly reports whether a custom menu item needs an open folder (or an
// open file): it is greyed out on the home screen (app/menu).
func folderOnly(label string) bool {
	switch label {
	case labelNewPage, labelCloseFolder, labelPrintPage, labelExport, labelExportWebsite, labelExportPDF, labelFind, labelFindNext, labelFindPrevious:
		return true
	}
	return false
}

// realFolderOnly reports whether a folderOnly item also needs a real folder: it
// is greyed out while one file is open on its own (app/single-file). Print Page
// and Export › PDF work for one file.
func realFolderOnly(label string) bool {
	switch label {
	case labelNewPage, labelExportWebsite:
		return true
	}
	return false
}

// singleOnly reports whether a custom menu item is for one file open on its
// own: it is greyed out otherwise.
func singleOnly(label string) bool { return label == labelOpenEnclosing }

// menuState keeps the enabled state of the folder-only menu items in step
// with whether a folder, or one file on its own, is open. It holds setters,
// not Wails items, so tests drive it without an application.
type menuState struct {
	mu     sync.Mutex
	open   bool
	single bool
	items  []menuItem2
}

// menuItem2 is a registered item: enabled while open (any), while open and not
// single (real folder), or while single.
type menuItem2 struct {
	kind itemKind
	set  func(enabled bool)
}

type itemKind int

const (
	anyOpen    itemKind = iota // a folder or a file
	realFolder                 // a folder, not one file on its own
	singleFile                 // one file on its own
)

func (k itemKind) enabled(open, single bool) bool {
	switch k {
	case realFolder:
		return open && !single
	case singleFile:
		return open && single
	}
	return open
}

func newMenuState(folderOpen bool) *menuState { return &menuState{open: folderOpen} }

func (m *menuState) register(k itemKind, set func(enabled bool)) {
	m.mu.Lock()
	m.items = append(m.items, menuItem2{k, set})
	open, single := m.open, m.single
	m.mu.Unlock()
	set(k.enabled(open, single))
}

// add registers an item that needs a folder or a file open and gives it the current state.
func (m *menuState) add(set func(enabled bool)) { m.register(anyOpen, set) }

// addReal registers an item that needs a real folder (New Page, Export › Website).
func (m *menuState) addReal(set func(enabled bool)) { m.register(realFolder, set) }

// addSingle registers an item for one file on its own (Open Enclosing Folder).
func (m *menuState) addSingle(set func(enabled bool)) { m.register(singleFile, set) }

// addFor registers the item with the label, if the state governs it.
func (m *menuState) addFor(label string, set func(enabled bool)) {
	switch {
	case singleOnly(label):
		m.addSingle(set)
	case realFolderOnly(label):
		m.addReal(set)
	case folderOnly(label):
		m.add(set)
	}
}

// setOpen records what is open (nothing, a folder, or one file on its own) and updates the items.
func (m *menuState) setOpen(open, single bool) {
	if m == nil {
		return
	}
	m.mu.Lock()
	m.open, m.single = open, open && single
	items := append([]menuItem2{}, m.items...)
	single = m.single
	m.mu.Unlock()
	for _, it := range items {
		it.set(it.kind.enabled(open, single))
	}
}

// setFolderOpen enables the folder-only items when a folder is open and
// disables them otherwise. A nil menuState (tests, headless) does nothing.
func (m *menuState) setFolderOpen(open bool) { m.setOpen(open, false) }

// folderOpened tells the menu whether a folder is open now, and whether it is
// one file on its own.
func (z *Zen) folderOpened(open bool) { z.menu.setOpen(open, open && z.s != nil && z.s.IsSingle()) }
