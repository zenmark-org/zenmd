package sitetemplates

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"sort"

	"github.com/zenmark-org/zenmd/internal/atomicfile"
	"gopkg.in/yaml.v3"
)

// RecordFile records which built-in presets were installed, and in which
// version; it sits in <templates dir>/blueprints and in <templates dir>/pages.
const RecordFile = ".zen-templates.yml"

// record is what an install wrote. Fingerprints are the SHA-256 of what each
// install wrote ("sha256:<hex>"): a copy that still matches is untouched and an
// update may replace it in place. A record from before fingerprints has none,
// and its copies count as edited.
type record struct {
	Installed    map[string]string `yaml:"installed"`
	Fingerprints map[string]string `yaml:"fingerprints,omitempty"`
}

func readRecord(folder string) record {
	var r record
	if data, err := os.ReadFile(filepath.Join(folder, RecordFile)); err == nil {
		_ = yaml.Unmarshal(data, &r)
	}
	if r.Installed == nil {
		r.Installed = map[string]string{}
	}
	if r.Fingerprints == nil {
		r.Fingerprints = map[string]string{}
	}
	return r
}

func writeRecord(folder string, r record) error {
	data, err := yaml.Marshal(r)
	if err != nil {
		return err
	}
	head := "# Written by zen: which built-in templates were installed here, in which version,\n# and a fingerprint of what was written (a copy you have not edited can be updated in place).\n# A template that is listed here and was deleted is not installed again.\n"
	return atomicfile.Write(filepath.Join(folder, RecordFile), append([]byte(head), data...))
}

// fingerprintFS is the SHA-256 of a folder's files (their paths and contents,
// in name order), as "sha256:<hex>". Folders themselves are not part of it.
func fingerprintFS(fsys fs.FS) (string, error) {
	h := sha256.New()
	err := fs.WalkDir(fsys, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		fmt.Fprintf(h, "%s\x00", p)
		if !d.Type().IsRegular() { // a link or special file: its presence counts
			fmt.Fprintf(h, "%s\x00", d.Type())
			return nil
		}
		data, err := fs.ReadFile(fsys, p)
		if err != nil {
			return err
		}
		fmt.Fprintf(h, "%d\x00", len(data))
		h.Write(data)
		return nil
	})
	if err != nil {
		return "", err
	}
	return "sha256:" + hex.EncodeToString(h.Sum(nil)), nil
}

// fingerprintDir is fingerprintFS of a real folder; a link or a missing folder is an error.
func fingerprintDir(dir string) (string, error) {
	fi, err := os.Lstat(dir)
	if err != nil {
		return "", err
	}
	if !fi.IsDir() {
		return "", errors.New(dir + " is not a folder")
	}
	return fingerprintFS(os.DirFS(dir))
}

// fingerprintBytes is the fingerprint of one file.
func fingerprintBytes(data []byte) string {
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}

// UpdateResult says what an update did: Path is the copy that was replaced
// (InPlace) or the new copy written beside the user's edited one.
type UpdateResult struct {
	Path    string
	InPlace bool
}

// builtinIDs lists the valid presets of a bundle, sorted.
func builtinIDs(builtins fs.FS) ([]Template, error) {
	list, _ := Builtins(builtins)
	var out []Template
	for _, t := range list {
		if t.Err == "" {
			out = append(out, t)
		}
	}
	if len(out) == 0 {
		return nil, errors.New("no built-in templates")
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

// Install copies the built-in presets that were never installed into
// <dir>/blueprints. A copy that is older than its built-in and still matches the
// fingerprint of its install is updated in place, silently. Any other installed
// template is never overwritten and one the user
// deleted is not installed again (the record remembers). A folder the user
// made with the same name is theirs and is left alone. An error means the
// folder cannot be written; the presets can then be used from the bundle.
func Install(dir string, builtins fs.FS) error {
	sites := BlueprintsDir(dir)
	if err := os.MkdirAll(sites, 0o755); err != nil {
		return err
	}
	list, err := builtinIDs(builtins)
	if err != nil {
		return err
	}
	rec := readRecord(sites)
	changed := false
	for _, t := range list {
		if have, done := rec.Installed[t.ID]; done {
			// A copy nobody edited follows the built-in; an edited one, one
			// without a fingerprint and a deleted one are left alone.
			if want, ok := rec.Fingerprints[t.ID]; ok && newer(t.Version, have) {
				copyPath := filepath.Join(sites, t.ID)
				if cur, err := fingerprintDir(copyPath); err == nil && cur == want {
					if fp, err := fingerprintFS(t.fsys); err == nil && replaceTree(t.fsys, copyPath) == nil {
						rec.Installed[t.ID], rec.Fingerprints[t.ID] = t.Version, fp
						slog.Info("blueprint "+t.ID+" updated to "+t.Version+" (your copy was not edited)", "source", "app")
						changed = true
					}
				}
			}
			continue
		}
		dest := filepath.Join(sites, t.ID)
		if _, err := os.Lstat(dest); err == nil {
			continue // the user's own
		}
		if err := copyTree(t.fsys, dest); err != nil {
			return fmt.Errorf("installing %s: %w", t.ID, err)
		}
		rec.Installed[t.ID] = t.Version
		if fp, err := fingerprintFS(t.fsys); err == nil {
			rec.Fingerprints[t.ID] = fp
		}
		changed = true
	}
	if changed || !fileExists(filepath.Join(sites, RecordFile)) {
		return writeRecord(sites, rec)
	}
	return nil
}

func fileExists(p string) bool { _, err := os.Lstat(p); return err == nil }

// Update brings the built-in template id to its newer version. A copy that
// still matches the fingerprint of its install is untouched, and is replaced
// in place. An edited copy (or one whose install recorded no fingerprint) is
// never touched: the new version is written to <dir>/blueprints/<id>.new
// beside it instead.
func Update(dir string, builtins fs.FS, id string) (UpdateResult, error) {
	sites := BlueprintsDir(dir)
	rec := readRecord(sites)
	have, ok := rec.Installed[id]
	if !ok {
		return UpdateResult{}, fmt.Errorf("%q is not a built-in blueprint that was installed here", id)
	}
	var t *Template
	list, _ := Builtins(builtins)
	for i := range list {
		if list[i].ID == id && list[i].Err == "" {
			t = &list[i]
		}
	}
	if t == nil {
		return UpdateResult{}, fmt.Errorf("this zen has no built-in blueprint %q", id)
	}
	if !newer(t.Version, have) {
		return UpdateResult{}, fmt.Errorf("%s is up to date (version %s)", id, have)
	}
	fp, err := fingerprintFS(t.fsys)
	if err != nil {
		return UpdateResult{}, err
	}
	copyPath := filepath.Join(sites, id)
	if want, ok := rec.Fingerprints[id]; ok {
		if cur, err := fingerprintDir(copyPath); err == nil && cur == want {
			if err := replaceTree(t.fsys, copyPath); err != nil {
				return UpdateResult{}, fmt.Errorf("updating %s: %w", id, err)
			}
			rec.Installed[id], rec.Fingerprints[id] = t.Version, fp
			return UpdateResult{Path: copyPath, InPlace: true}, writeRecord(sites, rec)
		}
	}
	dest := filepath.Join(sites, id+".new")
	if fileExists(dest) {
		return UpdateResult{}, fmt.Errorf("%s already exists; compare it with your copy and delete it first", path.Base(dest))
	}
	if err := copyTree(t.fsys, dest); err != nil {
		return UpdateResult{}, err
	}
	rec.Installed[id] = t.Version
	if err := writeRecord(sites, rec); err != nil {
		return UpdateResult{Path: dest}, err
	}
	return UpdateResult{Path: dest}, nil
}

// replaceTree replaces the folder dest with a copy of src. The new copy is
// made beside it first, and the old one is put back when the swap fails.
func replaceTree(src fs.FS, dest string) error {
	parent, base := filepath.Dir(dest), filepath.Base(dest)
	staged := filepath.Join(parent, ".tmp-update-"+base)
	old := filepath.Join(parent, ".tmp-old-"+base)
	_ = os.RemoveAll(staged)
	_ = os.RemoveAll(old)
	if err := copyTree(src, staged); err != nil {
		return err
	}
	if err := os.Rename(dest, old); err != nil {
		os.RemoveAll(staged)
		return err
	}
	if err := os.Rename(staged, dest); err != nil {
		_ = os.Rename(old, dest)
		os.RemoveAll(staged)
		return err
	}
	return os.RemoveAll(old)
}

// copyTree copies a template folder to dest through a temporary folder, so a
// failure leaves nothing half done. Files are 0644, folders 0755.
func copyTree(src fs.FS, dest string) error {
	parent := filepath.Dir(dest)
	tmp, err := os.MkdirTemp(parent, ".tmp-install-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)
	err = fs.WalkDir(src, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		target := filepath.Join(tmp, filepath.FromSlash(p))
		switch {
		case d.IsDir():
			return os.MkdirAll(target, 0o755)
		case d.Type().IsRegular():
			data, err := fs.ReadFile(src, p)
			if err != nil {
				return err
			}
			return os.WriteFile(target, data, 0o644)
		}
		return nil // links and special files are not copied
	})
	if err != nil {
		return err
	}
	if err := os.Chmod(tmp, 0o755); err != nil {
		return err
	}
	return os.Rename(tmp, dest)
}
