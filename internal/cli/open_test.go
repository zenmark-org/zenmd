package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// zenmd <file.md> opens the file alone unless a folder above it (below home)
// has a marker; --file and --folder force the one or the other.
func TestOpenSingleFile(t *testing.T) {
	h := home(t) // a fake HOME and XDG_CONFIG_HOME: the real settings are never read
	dl := filepath.Join(h, "Downloads")
	write(t, filepath.Join(dl, "notes.md"))
	write(t, filepath.Join(dl, "deep", "more", "b.markdown"))
	write(t, filepath.Join(dl, "other.md")) // a sibling nothing may scan

	// No marker above: the file alone.
	expect(t, run(t, h, nil, "Downloads/notes.md"), Launch{Dir: dl, Page: "notes.md", Mode: "read", File: true})
	expect(t, run(t, dl, nil, "notes.md"), Launch{Dir: dl, Page: "notes.md", Mode: "read", File: true})
	expect(t, run(t, h, nil, filepath.Join(dl, "deep", "more", "b.markdown")),
		Launch{Dir: filepath.Join(dl, "deep", "more"), Page: "b.markdown", Mode: "read", File: true})
	// zenmd edit does the same.
	expect(t, run(t, dl, nil, "edit", "notes.md"), Launch{Dir: dl, Page: "notes.md", Mode: "edit", File: true})
	// Flags go before or after the path.
	expect(t, run(t, dl, nil, "notes.md", "--file"), Launch{Dir: dl, Page: "notes.md", Mode: "read", File: true})
	expect(t, run(t, dl, nil, "--file", "notes.md"), Launch{Dir: dl, Page: "notes.md", Mode: "read", File: true})
	expect(t, run(t, dl, nil, "edit", "--file", "notes.md"), Launch{Dir: dl, Page: "notes.md", Mode: "edit", File: true})

	// --folder forces the folder, even with no marker: what zenmd <file> did before.
	expect(t, run(t, dl, nil, "notes.md", "--folder"), Launch{Dir: dl, Page: "notes.md", Mode: "read"})
	expect(t, run(t, dl, nil, "--folder", "notes.md"), Launch{Dir: dl, Page: "notes.md", Mode: "read"})

	// A marker folder above the file: the folder, at that page, as before.
	proj := filepath.Join(h, "proj")
	write(t, filepath.Join(proj, ".zenmd.yml"))
	write(t, filepath.Join(proj, "docs", "a.md"))
	expect(t, run(t, h, nil, "proj/docs/a.md"), Launch{Dir: proj, Page: "docs/a.md", Mode: "read"})
	legacy := filepath.Join(h, "old")
	write(t, filepath.Join(legacy, ".zen.yml"))
	write(t, filepath.Join(legacy, "a.md"))
	expect(t, run(t, h, nil, "old/a.md"), Launch{Dir: legacy, Page: "a.md", Mode: "read"})
	repo := filepath.Join(h, "repo")
	if err := os.MkdirAll(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(repo, "README.md"))
	write(t, filepath.Join(repo, "x", "y.md"))
	// README.md at the root of a git repo opens the repo folder ...
	expect(t, run(t, repo, nil, "README.md"), Launch{Dir: repo, Page: "README.md", Mode: "read"})
	expect(t, run(t, h, nil, "repo/x/y.md"), Launch{Dir: repo, Page: "x/y.md", Mode: "read"})
	// ... unless --file is given.
	expect(t, run(t, repo, nil, "README.md", "--file"), Launch{Dir: repo, Page: "README.md", Mode: "read", File: true})
	expect(t, run(t, h, nil, "--file", "proj/docs/a.md"), Launch{Dir: filepath.Join(proj, "docs"), Page: "a.md", Mode: "read", File: true})
	// edit inside a project is still the folder.
	expect(t, run(t, h, nil, "edit", "proj/docs/a.md"), Launch{Dir: proj, Page: "docs/a.md", Mode: "edit"})

	// A folder is always a folder; --file with one is a usage error (exit 2).
	expect(t, run(t, h, nil, "Downloads"), Launch{Dir: dl, Mode: "read"})
	expect(t, run(t, h, nil, "--folder", "Downloads"), Launch{Dir: dl, Mode: "read"})
	for _, args := range [][]string{
		{"--file", "Downloads"},
		{"Downloads", "--file"},
		{"edit", "--file", "Downloads"},
		{"--file"},                            // the current folder is a folder
		{"--file", "--folder", "Downloads/notes.md"}, // not both
		{"--file", "Downloads/missing.md"},    // there is no file to open alone
		{"--file", "Downloads/img.png"},
	} {
		write(t, filepath.Join(dl, "img.png"))
		r := run(t, h, nil, args...)
		if r.code != 2 || r.launched || !strings.Contains(r.stderr, "zenmd:") {
			t.Errorf("%v: want a usage error, got %+v", args, r)
		}
	}
	r := run(t, h, nil, "--file", "Downloads")
	if !strings.Contains(r.stderr, "--file") {
		t.Errorf("the error does not name the flag: %q", r.stderr)
	}

	// A new file (edit creates it on save) cannot open alone: it opens its folder, as before.
	expect(t, run(t, h, nil, "edit", "Downloads/new.md"), Launch{Dir: dl, Page: "new.md", Mode: "edit"})

	// The marker rule stops at the home folder: a marker in home itself is not a project.
	write(t, filepath.Join(h, ".git", "HEAD"))
	expect(t, run(t, h, nil, "Downloads/notes.md"), Launch{Dir: dl, Page: "notes.md", Mode: "read", File: true})
}

func TestOpenFlagsInHelp(t *testing.T) {
	h := home(t)
	r := run(t, h, nil, "--help")
	i := strings.Index(r.stdout, "Opening")
	if i < 0 {
		t.Fatalf("the help has no Opening section:\n%s", r.stdout)
	}
	section := r.stdout[i:]
	if j := strings.Index(section[len("Opening"):], "\n\n"); j >= 0 {
		section = section[:len("Opening")+j]
	}
	for _, want := range []string{"--file", "--folder"} {
		if !strings.Contains(section, want) {
			t.Errorf("Opening lacks %s:\n%s", want, section)
		}
	}
}
