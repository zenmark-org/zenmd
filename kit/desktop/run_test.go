package desktop

import (
	"errors"
	"testing"
)

type openWithRig struct {
	opened   []string
	emitted  []emitted
	warned   []string
	editorOK bool
}

type emitted struct {
	name string
	data any
}

func (r *openWithRig) handle(path string, args []string, o Options) {
	handleOpenWith(path, args, o, func() bool { return r.editorOK },
		func(name string, data any) { r.emitted = append(r.emitted, emitted{name, data}) },
		func(title, msg string) { r.warned = append(r.warned, title+": "+msg) })
}

// Finder's Open With hands a file or a folder to the app, which decides how it opens (a file alone, a project
// folder); the window gets what the app answers.
func TestOpenWithFile(t *testing.T) {
	r := &openWithRig{editorOK: true}
	o := Options{
		OpenedEvent: "workspace:opened",
		OpenWith: func(path string) (any, error) {
			r.opened = append(r.opened, path)
			switch path {
			case "/x/cancelled.md":
				return nil, nil
			case "/x/bad.png":
				return nil, errors.New("not a folder or markdown file")
			}
			return "startup:" + path, nil
		},
	}

	r.handle("/x/notes.md", []string{"edit"}, o)
	if len(r.opened) != 1 || r.opened[0] != "/x/notes.md" || len(r.emitted) != 1 ||
		r.emitted[0].name != "workspace:opened" || r.emitted[0].data != "startup:/x/notes.md" {
		t.Fatalf("a file: opened %v emitted %+v", r.opened, r.emitted)
	}

	// The process's own arguments are not Open With events (the CLI handled them), nor is a relative path.
	r = &openWithRig{editorOK: true}
	r.handle("/x/notes.md", []string{"/x/notes.md"}, o)
	r.handle("notes.md", nil, o)
	if len(r.opened) != 0 || len(r.emitted) != 0 {
		t.Fatalf("own argument: %v %+v", r.opened, r.emitted)
	}

	// An editor that keeps its unsaved changes stops it.
	r = &openWithRig{editorOK: false}
	r.handle("/x/notes.md", nil, o)
	if len(r.opened) != 0 || len(r.emitted) != 0 {
		t.Fatalf("editor kept changes: %v %+v", r.opened, r.emitted)
	}

	// Cancelled from the loader: nothing is sent and nothing is said.
	r = &openWithRig{editorOK: true}
	r.handle("/x/cancelled.md", nil, o)
	if len(r.opened) != 1 || len(r.emitted) != 0 || len(r.warned) != 0 {
		t.Fatalf("cancelled: %v %+v %v", r.opened, r.emitted, r.warned)
	}

	// A refusal is shown in a dialog, with the app's words.
	r.handle("/x/bad.png", nil, o)
	if len(r.emitted) != 0 || len(r.warned) != 1 || r.warned[0] != "Could not open: not a folder or markdown file" {
		t.Fatalf("refused: %+v %v", r.emitted, r.warned)
	}
}
