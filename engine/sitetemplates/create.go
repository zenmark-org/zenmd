package sitetemplates

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// Limits of one template: larger ones are refused before anything is written.
var (
	MaxFiles       = 2000
	MaxBytes int64 = 50 << 20
)

// maxClashes is how many clashing files an error names.
const maxClashes = 10

// ErrClash is wrapped by the error of a Create that found files in the way.
var ErrClash = errors.New("files already exist")

// Result lists what Create wrote: target-relative paths, in creation order.
type Result struct{ Files []string }

// now is the clock behind {{date}}; tests replace it.
var now = time.Now

// placeholderRE is one {{name}}: no spaces, so "{{ title }}" stays as it is.
var placeholderRE = regexp.MustCompile(`\{\{([A-Za-z][A-Za-z0-9_-]*)\}\}`)

// textExts are the files whose contents get placeholders.
var textExts = map[string]bool{".md": true, ".yml": true, ".yaml": true, ".txt": true, ".css": true}

func replace(s string, vals map[string]string) string {
	if !strings.Contains(s, "{{") {
		return s
	}
	return placeholderRE.ReplaceAllStringFunc(s, func(m string) string {
		if v, ok := vals[m[2:len(m)-2]]; ok {
			return v
		}
		return m
	})
}

// values works out every placeholder: the built-ins, then each ask key from
// values (an empty one takes its default, which may use {{folder}} and {{key}} of an earlier ask).
func (t Template) values(given map[string]string, folder string) (map[string]string, error) {
	today := now().Format("2006-01-02")
	out := map[string]string{"date": today, "today": today, "folder": folder, "title": "", "author": ""}
	asked := map[string]bool{}
	for _, a := range t.Ask {
		asked[a.Key] = true
	}
	if !asked["title"] {
		out["title"] = folder
	}
	given1 := func(k string) (string, error) {
		v := given[k]
		if strings.ContainsRune(v, 0) {
			return "", fmt.Errorf("the value of %s contains a NUL byte", k)
		}
		if len(v) > 4096 {
			return "", fmt.Errorf("the value of %s is too long", k)
		}
		return v, nil
	}
	if !asked["title"] {
		v, err := given1("title")
		if err != nil {
			return nil, err
		}
		if v != "" {
			out["title"] = v
		}
	}
	if v, err := given1("author"); err != nil {
		return nil, err
	} else if v != "" && !asked["author"] {
		out["author"] = v
	}
	// In ask order, so a default may use {{key}} of any ask before it.
	for _, a := range t.Ask {
		v, err := given1(a.Key)
		if err != nil {
			return nil, err
		}
		if v == "" {
			v = replace(a.Default, out)
		}
		out[a.Key] = v
	}
	return out, nil
}

type item struct {
	src  string // slash path in the template; "" for a generated file
	dest string // slash path in the target
	size int64
	data []byte // a generated file's contents (placeholders already filled)
	add  string // appended to a copied file (the sections' part of .zenmd.yml)
}

// plan walks the template and works out what would be written. It refuses
// symlinks, special files, names that leave the target, duplicates and
// templates over the limits, all before anything is written.
func (t Template) plan(vals map[string]string) ([]item, error) {
	if t.Err != "" {
		return nil, fmt.Errorf("template %q is not valid: %s", t.ID, t.Err)
	}
	if t.fsys == nil {
		return nil, fmt.Errorf("template %q has no files", t.ID)
	}
	for _, f := range t.Files {
		if _, err := fs.Stat(t.fsys, f); err != nil {
			return nil, fmt.Errorf("files: %s is not in the blueprint", f)
		}
	}
	var items []item
	var total int64
	seen := map[string]string{}
	err := fs.WalkDir(t.fsys, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if p == "." {
			return nil
		}
		if d.Type()&fs.ModeSymlink != 0 {
			return fmt.Errorf("%s is a symbolic link; templates cannot contain links", p)
		}
		if t.ignored(p) {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			return nil
		}
		if !d.Type().IsRegular() {
			return fmt.Errorf("%s is not a regular file", p)
		}
		if p == ManifestFile || p == LegacyManifestFile {
			return nil
		}
		clean, err := cleanDest(p, replace(p, vals), seen)
		if err != nil {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		items = append(items, item{src: p, dest: clean, size: info.Size()})
		total += info.Size()
		if len(items) > MaxFiles {
			return fmt.Errorf("the template has more than %d files", MaxFiles)
		}
		if total > MaxBytes {
			return fmt.Errorf("the template is larger than %d MB", MaxBytes>>20)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	// What the sections make: index.md, indexes, pages, page templates and
	// the sections' part of .zenmd.yml.
	gens, frag, err := t.generate(vals)
	if err != nil {
		return nil, err
	}
	for _, g := range gens {
		clean, err := cleanDest(g.dest, g.dest, seen)
		if err != nil {
			return nil, err
		}
		items = append(items, item{dest: clean, size: int64(len(g.data)), data: []byte(g.data)})
		total += int64(len(g.data))
	}
	if frag != "" {
		found := false
		for i := range items {
			if strings.EqualFold(items[i].dest, ".zenmd.yml") && items[i].src != "" {
				items[i].add, found = frag, true
			}
		}
		if !found {
			head := "# Settings for this folder. Edit freely: zen reads this file again when it changes.\nversion: 1\n\n"
			if _, taken := seen[".zenmd.yml"]; taken {
				return nil, errors.New(".zenmd.yml would be written twice")
			}
			seen[".zenmd.yml"] = ".zenmd.yml"
			items = append(items, item{dest: ".zenmd.yml", data: []byte(head + frag)})
		}
	}
	if len(items) > MaxFiles {
		return nil, fmt.Errorf("the blueprint has more than %d files", MaxFiles)
	}
	if total > MaxBytes {
		return nil, fmt.Errorf("the blueprint is larger than %d MB", MaxBytes>>20)
	}
	return items, nil
}

// cleanDest checks a destination (slash path) stays inside the new folder and
// is not written twice; seen maps a lower case path to what wrote it.
func cleanDest(from, dest string, seen map[string]string) (string, error) {
	clean := path.Clean(strings.ReplaceAll(dest, "\\", "/"))
	if strings.ContainsRune(dest, 0) || path.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, "../") || clean == "." {
		return "", fmt.Errorf("%s would be written outside the new folder", from)
	}
	if other, dup := seen[strings.ToLower(clean)]; dup {
		return "", fmt.Errorf("%s and %s would both be written to %s", other, from, clean)
	}
	seen[strings.ToLower(clean)] = from
	return clean, nil
}

// addConfig appends the sections' part to a .zenmd.yml of the blueprint: its
// layout: only when the file has none, and its templates: rules, which the
// file must not have.
func addConfig(data, frag string) (string, error) {
	if regexp.MustCompile(`(?m)^templates:`).MatchString(data) && strings.Contains(frag, "templates:") {
		return "", errors.New(".zenmd.yml already has templates:; the sections write the path rules")
	}
	if regexp.MustCompile(`(?m)^layout:`).MatchString(data) {
		var keep []string
		for _, l := range strings.Split(frag, "\n") {
			if !strings.HasPrefix(l, "layout:") {
				keep = append(keep, l)
			}
		}
		frag = strings.TrimLeft(strings.Join(keep, "\n"), "\n")
	}
	if data != "" && !strings.HasSuffix(data, "\n") {
		data += "\n"
	}
	return data + "\n" + frag, nil
}

// ignored reports whether the template leaves p out: by its ignore: list
// (matched against the name and the whole path) and always .DS_Store.
func (t Template) ignored(p string) bool {
	base := path.Base(p)
	if base == ".DS_Store" {
		return true
	}
	for _, pat := range t.ignore {
		if ok, _ := path.Match(pat, base); ok {
			return true
		}
		if ok, _ := path.Match(pat, p); ok {
			return true
		}
	}
	return false
}

// Create copies the template into target, a new or an existing folder.
// Nothing is ever overwritten: when any file the template would write exists,
// nothing is written and the error wraps ErrClash with up to 10 names.
// Placeholders are replaced in text files and in file names. A failure
// half-way removes what was written.
func Create(t Template, target string, values map[string]string) (Result, error) {
	if target == "" {
		return Result{}, errors.New("choose a folder")
	}
	abs, err := filepath.Abs(target)
	if err != nil {
		return Result{}, err
	}
	if t.insideTemplates(abs) {
		return Result{}, errors.New("choose a folder outside the templates folder")
	}
	folder := filepath.Base(abs)
	vals, err := t.values(values, folder)
	if err != nil {
		return Result{}, err
	}
	items, err := t.plan(vals)
	if err != nil {
		return Result{}, err
	}

	// The target: new, or an existing folder.
	if fi, err := os.Stat(abs); err == nil && !fi.IsDir() {
		return Result{}, fmt.Errorf("%s is a file, not a folder", folder)
	}
	var clashes []string
	for _, it := range items {
		if c := clashAt(abs, it.dest); c {
			clashes = append(clashes, it.dest)
		}
	}
	if len(clashes) > 0 {
		list := strings.Join(clashes[:min(len(clashes), maxClashes)], ", ")
		if len(clashes) > maxClashes {
			list += fmt.Sprintf(" and %d more", len(clashes)-maxClashes)
		}
		return Result{}, fmt.Errorf("%w: %s", ErrClash, list)
	}

	// Write, remembering what is ours so a failure can undo it.
	var madeDirs, madeFiles []string
	undo := func() {
		for i := len(madeFiles) - 1; i >= 0; i-- {
			os.Remove(madeFiles[i])
		}
		for i := len(madeDirs) - 1; i >= 0; i-- {
			os.Remove(madeDirs[i])
		}
	}
	mkdirs := func(dir string) error {
		var todo []string
		for d := dir; ; d = filepath.Dir(d) {
			if _, err := os.Lstat(d); err == nil {
				break
			}
			todo = append(todo, d)
			if filepath.Dir(d) == d {
				break
			}
		}
		for i := len(todo) - 1; i >= 0; i-- {
			if err := os.Mkdir(todo[i], 0o755); err != nil {
				return err
			}
			_ = os.Chmod(todo[i], 0o755)
			madeDirs = append(madeDirs, todo[i])
		}
		return nil
	}
	if err := mkdirs(abs); err != nil {
		undo()
		return Result{}, err
	}
	var res Result
	for _, it := range items {
		full := filepath.Join(abs, filepath.FromSlash(it.dest))
		if err := mkdirs(filepath.Dir(full)); err != nil {
			undo()
			return Result{}, err
		}
		if err := t.writeFile(it, full, vals); err != nil {
			undo()
			if errors.Is(err, fs.ErrExist) {
				return Result{}, fmt.Errorf("%w: %s", ErrClash, it.dest)
			}
			return Result{}, fmt.Errorf("writing %s: %w", it.dest, err)
		}
		madeFiles = append(madeFiles, full)
		res.Files = append(res.Files, it.dest)
	}
	return res, nil
}

// clashAt reports whether dest (slash path) cannot be created under root
// without touching something: it exists, or a parent is a file or a link.
func clashAt(root, dest string) bool {
	cur := root
	parts := strings.Split(dest, "/")
	for i, part := range parts {
		cur = filepath.Join(cur, part)
		fi, err := os.Lstat(cur)
		switch {
		case err != nil:
			return false // does not exist (or its parent does not): free
		case i == len(parts)-1:
			return true
		case !fi.IsDir(): // a file or a symlink where a folder goes
			return true
		}
	}
	return false
}

// insideTemplates reports whether abs is the templates folder or inside it.
func (t Template) insideTemplates(abs string) bool {
	if t.Dir == "" {
		return false
	}
	tdir := filepath.Dir(t.Dir) // .../blueprints
	if filepath.Base(tdir) == "blueprints" {
		tdir = filepath.Dir(tdir)
	}
	return within(real(abs), real(tdir))
}

// real resolves symlinks of the longest existing part of p.
func real(p string) string {
	rest := ""
	for cur := p; ; cur = filepath.Dir(cur) {
		if r, err := filepath.EvalSymlinks(cur); err == nil {
			return filepath.Join(r, rest)
		}
		rest = filepath.Join(filepath.Base(cur), rest)
		if filepath.Dir(cur) == cur {
			return p
		}
	}
}

func within(p, dir string) bool {
	rel, err := filepath.Rel(dir, p)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// writeFile writes one file through a temporary file in its folder; the final
// name is taken with a hard link, which fails when something appeared there
// meanwhile (nothing is ever replaced). Files are 0644.
func (t Template) writeFile(it item, full string, vals map[string]string) error {
	var in io.Reader
	if it.src == "" { // generated by the sections
		in = bytes.NewReader(it.data)
	} else {
		f, err := t.fsys.Open(it.src)
		if err != nil {
			return err
		}
		defer f.Close()
		in = f
	}
	tmp, err := os.CreateTemp(filepath.Dir(full), ".zen-new-*")
	if err != nil {
		return err
	}
	name := tmp.Name()
	defer os.Remove(name)
	var werr error
	if it.src != "" && textExts[strings.ToLower(path.Ext(it.src))] && !strings.HasPrefix(it.src, ".zen/templates/") {
		var data []byte
		if data, werr = io.ReadAll(io.LimitReader(in, MaxBytes+1)); werr == nil {
			text := replace(string(data), vals)
			if it.add != "" {
				text, werr = addConfig(text, it.add)
			}
			if werr == nil {
				_, werr = tmp.WriteString(text)
			}
		}
	} else {
		_, werr = io.Copy(tmp, in)
	}
	if werr == nil {
		werr = tmp.Chmod(0o644)
	}
	if cerr := tmp.Close(); werr == nil {
		werr = cerr
	}
	if werr != nil {
		return werr
	}
	if err := os.Link(name, full); err != nil {
		if errors.Is(err, fs.ErrExist) {
			return err
		}
		// No hard links on this volume: rename, after one more look.
		if _, serr := os.Lstat(full); serr == nil {
			return fs.ErrExist
		}
		return os.Rename(name, full)
	}
	return nil
}
