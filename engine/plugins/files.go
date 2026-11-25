package plugins

import (
	"errors"
	"path"
	"strings"
)

// MaxReadFile is the most Context.ReadFile reads from one file.
const MaxReadFile = 1 << 20

// maxPathLen is the longest path a page may write for a file.
const maxPathLen = 1024

var (
	// ErrBadPath is returned by ResolvePath for a path that is empty, too long
	// or holds a backslash, NUL or a newline.
	ErrBadPath = errors.New("the path is not valid")
	// ErrOutside is returned by ResolvePath for a path that leaves the folder.
	ErrOutside = errors.New("the path is outside the folder")
)

// FileRef finds the file a block names: Attrs["file"], else a leading
// ":file: <path>" line of the body, else Attrs["arg"]. body is the block's
// text without that line. ok is false when the block names no file.
func FileRef(b Block) (p, body string, ok bool) {
	body = b.Body
	line := ""
	rest := body
	for rest != "" {
		l, after, _ := strings.Cut(rest, "\n")
		if strings.TrimSpace(l) == "" {
			rest = after
			continue
		}
		if t := strings.TrimSpace(l); strings.HasPrefix(t, ":file:") && (len(t) == 6 || t[6] == ' ' || t[6] == '\t') {
			line = strings.TrimSpace(t[6:])
			body = after
		}
		break
	}
	for _, c := range []string{b.Attrs["file"], line, b.Attrs["arg"]} {
		if c = strings.TrimSpace(c); c != "" {
			return c, body, true
		}
	}
	return "", body, false
}

// ResolvePath turns a path written on page into a folder-relative one: "/a/b"
// is "a/b"; "b" on "docs/p.md" is "docs/b"; "../b" on "docs/p.md" is "b". The
// root itself is ".". It is an error when the path leaves the folder
// (ErrOutside) or is not a valid path (ErrBadPath).
func ResolvePath(page, p string) (string, error) {
	p = strings.TrimSpace(p)
	if p == "" || len(p) > maxPathLen || strings.ContainsAny(p, "\\\x00\r\n") {
		return "", ErrBadPath
	}
	var segs []string
	if !strings.HasPrefix(p, "/") {
		if dir := path.Dir(strings.ReplaceAll(page, "\\", "/")); dir != "." && dir != "/" {
			segs = strings.Split(strings.Trim(dir, "/"), "/")
		}
	}
	for _, s := range strings.Split(p, "/") {
		switch s {
		case "", ".":
		case "..":
			if len(segs) == 0 {
				return "", ErrOutside
			}
			segs = segs[:len(segs)-1]
		default:
			segs = append(segs, s)
		}
	}
	if len(segs) == 0 {
		return ".", nil
	}
	return strings.Join(segs, "/"), nil
}
