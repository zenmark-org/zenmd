package cliset

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/zenmark-org/zenmd/engine/config"
	"github.com/zenmark-org/zenmd/engine/session"
	"github.com/zenmark-org/zenmd/engine/sitetemplates"
	"github.com/zenmark-org/zenmd/internal/cli"
)

// InitSite is `zenmd init` and `zenmd blueprint` of the command-line build:
// it lists blueprints, makes one, writes a newer built-in beside the user's
// copy, or creates a site, and tells what it did on stdout. The folders it
// shows (--dir) are printed, never opened.
func InitSite(i cli.Init, stdout, stderr io.Writer) error {
	return InitSiteWith(nil)(i, stdout, stderr)
}

// InitSiteWith is InitSite for a build that has a file manager: the app passes
// desktop.Reveal. `zenmd init --dir` and `zenmd templates --dir` still print the
// path and never call it; it is for the callers that show a folder on request.
func InitSiteWith(reveal func(dir string) error) cli.Initer {
	return InitWith(Templates(), reveal)
}

// InitWith is InitSiteWith with the built-in blueprints given (the app's tests replace them).
func InitWith(presets fs.FS, reveal func(dir string) error) cli.Initer {
	_ = reveal // init only prints folders (specs/packaging/linux-cli, Security)
	return func(i cli.Init, stdout, stderr io.Writer) error {
		globalPath, _ := config.GlobalPath()
		s := session.New(session.Options{GlobalPath: globalPath, Presets: presets})
		defer s.Close()
		return RunInit(s.Templates(), i, stdout, stderr)
	}
}

// RunInit does the work of InitSite on the templates api it is given.
func RunInit(tpls session.TemplatesAPI, i cli.Init, stdout, stderr io.Writer) error {
	defer func() { // what happened to the templates folder (the sites move) is told once
		for _, n := range tpls.Notes() {
			fmt.Fprintln(stderr, "zenmd: "+n)
		}
	}()
	if i.Pages {
		return initPages(tpls, i, stdout, stderr)
	}
	if i.ShowDir {
		dir, _ := tpls.Folder()
		if dir == "" {
			return errors.New("there is no blueprints folder")
		}
		fmt.Fprintln(stdout, sitetemplates.BlueprintsDir(dir))
		return nil
	}
	if i.New {
		out, err := tpls.New(i.Template, i.From)
		if err != nil {
			return err
		}
		fmt.Fprintf(stdout, "Made the blueprint %s in %s. Edit its blueprint.yml, then try it with: zenmd init --blueprint %s <folder>\n", i.Template, out, i.Template)
		return nil
	}
	if i.Update {
		res, err := tpls.Update(i.Template)
		if err != nil {
			return err
		}
		fmt.Fprintln(stdout, updateSummary(i.Template, res))
		return nil
	}

	l := tpls.Listing()
	for _, w := range l.Warnings {
		fmt.Fprintln(stderr, "zenmd: "+w)
	}
	if l.Note != "" {
		fmt.Fprintln(stderr, "zenmd: "+l.Note)
	}
	if i.List {
		width := 0
		for _, t := range l.Templates {
			width = max(width, len(t.ID))
		}
		for _, t := range l.Templates {
			switch {
			case t.Err != "":
				fmt.Fprintf(stdout, "%-*s  (invalid: %s)\n", width, t.ID, t.Err)
			default:
				line := fmt.Sprintf("%-*s  %s", width, t.ID, t.Name)
				if t.Description != "" {
					line += " — " + t.Description
				}
				if t.Update != "" {
					line += fmt.Sprintf(" [update available: %s, see zenmd init --update %s]", t.Update, t.ID)
				}
				fmt.Fprintln(stdout, line)
			}
		}
		if len(l.Templates) == 0 {
			fmt.Fprintln(stdout, "No blueprints.")
		}
		return nil
	}

	var t sitetemplates.Template
	found := false
	for _, x := range l.Templates {
		if x.ID == i.Template {
			t, found = x, true
		}
	}
	if !found {
		var ids []string
		for _, x := range l.Templates {
			if x.Err == "" {
				ids = append(ids, x.ID)
			}
		}
		return cli.UsageError(fmt.Sprintf("there is no blueprint %q (blueprints: %s)", i.Template, strings.Join(ids, ", ")))
	}
	if t.Err != "" {
		return fmt.Errorf("blueprint %s is not valid: %s", t.ID, t.Err)
	}
	if l.Holds(i.Dir) {
		return errors.New("choose a folder outside the blueprints folder")
	}
	res, err := sitetemplates.Create(t, i.Dir, i.Values)
	if err != nil {
		return err
	}
	n := fmt.Sprintf("%d files", len(res.Files))
	if len(res.Files) == 1 {
		n = "1 file"
	}
	shown := shownFolder(i.Dir)
	fmt.Fprintf(stdout, "Created %s in %s (%s). Open it with: zenmd %s\n", n, shown, t.Name, shellQuote(shown))
	return nil
}

// shownFolder names dir the short way: relative to the current folder when it
// is inside it ("." for the folder itself).
func shownFolder(dir string) string {
	if cwd, err := os.Getwd(); err == nil {
		if rel, err := filepath.Rel(cwd, dir); err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return rel
		}
	}
	return dir
}

// shellQuote quotes s for a shell only when it needs it.
func shellQuote(s string) string {
	if s != "" && strings.Trim(s, "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789._-/+,:@") == "" {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// updateSummary says what an update did.
func updateSummary(id string, res sitetemplates.UpdateResult) string {
	if res.InPlace {
		return fmt.Sprintf("Updated %s in place: %s. It had no edits of yours.", id, res.Path)
	}
	return fmt.Sprintf("Wrote the newer %s to %s. Your copy has edits, so it was left as it is: compare the two, keep what you like, then delete the new one.", id, res.Path)
}

// initPages is zenmd templates: the page templates (list, show, update) and the templates folder (--dir).
func initPages(tpls session.TemplatesAPI, i cli.Init, stdout, stderr io.Writer) error {
	if i.ShowDir {
		dir, _ := tpls.Folder()
		if dir == "" {
			return errors.New("there is no templates folder")
		}
		fmt.Fprintln(stdout, dir)
		return nil
	}
	if i.Update {
		res, err := tpls.UpdatePage(i.Template)
		if err != nil {
			return err
		}
		fmt.Fprintln(stdout, updateSummary(i.Template, res))
		return nil
	}
	l := tpls.PagesIn(i.Dir) // the folder zenmd is run in: its own .zen/templates show too
	for _, w := range l.Warnings {
		fmt.Fprintln(stderr, "zenmd: "+w)
	}
	if l.Note != "" {
		fmt.Fprintln(stderr, "zenmd: "+l.Note)
	}
	if i.List {
		width := 0
		for _, p := range l.Templates {
			width = max(width, len(p.ID))
		}
		for _, p := range l.Templates {
			if p.Error != "" {
				fmt.Fprintf(stdout, "%-*s  (invalid: %s)  %s\n", width, p.ID, p.Error, p.Path)
				continue
			}
			line := fmt.Sprintf("%-*s  %-9s", width, p.ID, SourceLabel(p.Source, p.Edited))
			if p.Version != "" {
				line += "  " + p.Version
			}
			line += "  " + p.Name
			if p.Update != "" {
				line += fmt.Sprintf("  [update available: %s, see zenmd templates update %s]", p.Update, p.ID)
			}
			fmt.Fprintln(stdout, strings.TrimRight(line, " "))
		}
		if len(l.Templates) == 0 {
			fmt.Fprintln(stdout, "No page templates.")
		}
		return nil
	}
	for _, p := range l.Templates {
		if p.ID != i.Template {
			continue
		}
		if p.Error != "" {
			return fmt.Errorf("page template %s is not valid: %s", p.ID, p.Error)
		}
		showPage(stdout, p)
		return nil
	}
	var ids []string
	for _, p := range l.Templates {
		ids = append(ids, p.ID)
	}
	return cli.UsageError(fmt.Sprintf("there is no page template %q (page templates: %s)", i.Template, strings.Join(ids, ", ")))
}

func SourceLabel(src string, edited bool) string {
	switch src {
	case sitetemplates.SourceFolder:
		return "this folder"
	case sitetemplates.SourceCustom:
		return "custom"
	}
	if edited {
		return "built-in, edited"
	}
	return "built-in"
}

func showPage(w io.Writer, p session.PageTemplateInfo) {
	fmt.Fprintf(w, "%s: %s\n", p.ID, p.Name)
	if p.Description != "" {
		fmt.Fprintln(w, p.Description)
	}
	from := SourceLabel(p.Source, p.Edited)
	if p.Path != "" {
		from += ", " + p.Path
	}
	meta := "Layout: " + p.Layout + "\nFrom: " + from
	if p.Version != "" {
		meta += "\nVersion: " + p.Version
	}
	fmt.Fprintln(w, meta)
	if len(p.Rules) > 0 {
		fmt.Fprintln(w, "Rules here: "+strings.Join(p.Rules, ", "))
	}
	fmt.Fprintln(w, "\nFields")
	if len(p.Fields) == 0 {
		fmt.Fprintln(w, "  (none)")
	}
	width := 0
	for _, f := range p.Fields {
		width = max(width, len(f.Name))
	}
	for _, f := range p.Fields {
		line := fmt.Sprintf("  %-*s  %-7s", width, f.Name, f.Type)
		if f.Required {
			line += "  required"
		}
		if f.Default != "" {
			line += "  default " + f.Default
		}
		fmt.Fprintln(w, strings.TrimRight(line, " "))
	}
	if len(p.Outline) > 0 {
		fmt.Fprintln(w, "\nStructure")
		for _, o := range p.Outline {
			fmt.Fprintln(w, "  "+strings.Repeat("  ", o.Depth)+"- "+o.Text)
		}
	}
	fmt.Fprintln(w, "\nSkeleton")
	for _, line := range strings.Split(strings.TrimRight(p.Skeleton, "\n"), "\n") {
		fmt.Fprintln(w, strings.TrimRight("  "+line, " "))
	}
}
