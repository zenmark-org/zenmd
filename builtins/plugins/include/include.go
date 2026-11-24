// Package include renders the include directives of MyST and Sphinx and the
// same blocks written as :::include and :::literalinclude:
//
//   - include: the markdown of a file of the folder, rendered where the
//     directive stands (through the page's own pipeline, so its blocks and
//     links work). A file that includes itself stops with a warning.
//   - literalinclude: the text of a file as a code block, with an optional
//     language, line selection and line numbers.
//
// Both read a file the way a page read goes: a regular file inside the open
// folder, found next to the page or from the folder's root. A path with .., an
// absolute path, a symlink that leads out of the folder (or whose target
// cannot be checked) and a file over 1 MB are refused. A message never names
// a path.
//
// Lines: :start-line: counts from 0 and :end-line: is the first line left out
// (as in Sphinx and MyST); :lines: counts from 1 and takes ranges such as
// 1-3,5 and 4-.
package include

import (
	"embed"
	"errors"
	"fmt"
	"html"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/zenmark-org/zenmd/engine/plugins"
)

//go:embed assets/include.css
var assets embed.FS

const (
	maxFile     = 1 << 20 // the most a file may hold
	maxNumbered = 5000    // the most lines that get a number
)

var (
	classRE = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_-]{0,63}$`)
	langRE  = regexp.MustCompile(`^[A-Za-z0-9_+#.-]{1,32}$`)
	frontRE = regexp.MustCompile(`(?s)\A---[ \t]*\r?\n.*?\r?\n---[ \t]*(?:\r?\n|\z)`)
)

// languages maps a file extension to a code language.
var languages = map[string]string{
	".go": "go", ".py": "python", ".js": "javascript", ".mjs": "javascript", ".ts": "typescript", ".tsx": "tsx", ".jsx": "jsx",
	".json": "json", ".yml": "yaml", ".yaml": "yaml", ".toml": "toml", ".sh": "bash", ".bash": "bash", ".zsh": "bash",
	".md": "markdown", ".html": "html", ".htm": "html", ".css": "css", ".xml": "xml", ".sql": "sql",
	".rs": "rust", ".c": "c", ".h": "c", ".cpp": "cpp", ".cc": "cpp", ".hpp": "cpp", ".java": "java", ".kt": "kotlin",
	".rb": "ruby", ".php": "php", ".swift": "swift", ".cs": "csharp", ".lua": "lua", ".r": "r", ".tex": "latex",
	".ini": "ini", ".diff": "diff", ".patch": "diff", ".txt": "text",
}

var common = []string{"class", "name", "title", "arg", "file", "start-line", "end-line", "lines", "start-after", "end-before"}

// known options per block name; anything else is shown as ignored.
var known = map[string]map[string]bool{
	"include":        set(common...),
	"literalinclude": set(append([]string{"language", "linenos", "lineno-start", "lineno-match", "caption"}, common...)...),
}

func set(k ...string) map[string]bool {
	m := make(map[string]bool, len(k))
	for _, s := range k {
		m[s] = true
	}
	return m
}

type plugin struct{}

// New returns the include plugin.
func New() plugins.Plugin { return plugin{} }

func (plugin) Manifest() plugins.Manifest {
	return plugins.Manifest{
		ID:          "include",
		Version:     "0.1.0",
		APIVersion:  plugins.APIVersion,
		Description: "Files of the folder in a page: include and literalinclude",
		Default:     true,
		Blocks:      []string{"include", "literalinclude"},
		Allow:       []plugins.AllowRule{{Element: "span", Attrs: []string{"aria-hidden"}}},
		Help: plugins.Help{
			Summary: "Brings a file of the folder into a page. include shows the file's markdown where the block stands; literalinclude shows its text as code, with language, start-line, end-line, lines (such as 1-3,5) and linenos. A path is relative to the page or to the folder's root: no .., no absolute path, no symlink out, at most 1 MB. A file that includes itself stops with a warning. They read the same in markdown written for MyST, Jupyter Book or Sphinx.",
			Snippets: []plugins.Snippet{
				{Title: "Include a page", Markdown: ":::include{file=snippets/intro.md}\n:::\n"},
				{Title: "Include code, MyST form", Markdown: "```{literalinclude} code/hello.go\n:lines: 1-3,5\n:linenos:\n```\n"},
				{Title: "Include markdown, MyST form", Markdown: "```{include} snippets/intro.md\n:start-line: 2\n```\n"},
				{Title: "Code in the colon form", Markdown: ":::literalinclude{file=code/hello.go language=go caption=hello.go}\n:::\n"},
			},
		},
	}
}

func (plugin) Assets() fs.FS {
	sub, err := fs.Sub(assets, "assets")
	if err != nil {
		panic(err)
	}
	return sub
}

func (plugin) Entry() plugins.Assets { return plugins.Assets{CSS: []string{"include.css"}} }

func (plugin) RenderBlock(ctx plugins.Context, b plugins.Block) (string, error) {
	warn := func(msg string) {
		if ctx.Warn != nil {
			ctx.Warn(msg)
		}
	}
	ok, found := known[b.Name]
	if !found {
		return "", fmt.Errorf("include does not own %q", b.Name)
	}
	var unknown []string
	for k := range b.Attrs {
		if !ok[k] {
			unknown = append(unknown, k)
		}
	}
	sort.Strings(unknown)
	for _, k := range unknown {
		warn("option :" + k + ": is not supported and was ignored")
	}
	rel := strings.TrimSpace(b.Attrs["file"])
	if rel == "" {
		rel = strings.TrimSpace(b.Attrs["arg"])
	}
	if rel == "" {
		return "", errors.New(b.Name + " needs a file: write it after the name, or as file=")
	}
	real, raw, err := readFile(ctx, rel)
	if err != nil {
		return "", err
	}
	if b.Name == "include" && sameFile(ctx, real) {
		warn("a page cannot include itself; the include was skipped")
		return status("This include was skipped: a page cannot include itself."), nil
	}
	text := selectLines(raw, b.Attrs, warn)

	var userClass []string
	for _, tok := range strings.Fields(b.Attrs["class"]) {
		if classRE.MatchString(tok) && !strings.HasPrefix(tok, "zen-") {
			userClass = append(userClass, tok)
		}
	}
	idAttr := ""
	if n := b.Attrs["name"]; n != "" {
		if id := slug(n); id != "" {
			idAttr = ` id="` + id + `"`
		} else {
			warn("option :name: has a value that is not supported and was ignored")
		}
	}

	if b.Name == "include" {
		text.text = frontRE.ReplaceAllString(text.text, "")
		body := ""
		if strings.TrimSpace(text.text) != "" {
			if ctx.Markdown == nil {
				body = "<p>" + html.EscapeString(strings.TrimSpace(text.text)) + "</p>"
			} else {
				h, err := ctx.Markdown(text.text)
				if err != nil {
					warn("includes are nested too deep (does a file include itself?); this one was skipped")
					return status("This include was skipped: includes are nested too deep."), nil
				}
				body = h
			}
		}
		return `<div` + idAttr + ` class="` + strings.Join(append([]string{"zen-include", "zen-include-markdown"}, userClass...), " ") + `">` + body + `</div>`, nil
	}
	return literal(b, text, real, idAttr, userClass, warn), nil
}

// status is a kit block with one muted line, for an include that was skipped.
func status(msg string) string {
	return `<div class="zen-block zen-include-note"><div class="zen-block-body"><span class="zen-block-status">` + html.EscapeString(msg) + `</span></div></div>`
}

// literal renders the text of a file as a code block.
func literal(b plugins.Block, s selected, real, idAttr string, userClass []string, warn func(string)) string {
	lang := strings.TrimSpace(b.Attrs["language"])
	if lang != "" && !langRE.MatchString(lang) {
		warn("option :language: has a value that is not a language name and was ignored")
		lang = ""
	}
	if lang == "" {
		lang = languages[strings.ToLower(filepath.Ext(real))]
	}
	numbered := false
	if v, ok := b.Attrs["linenos"]; ok && v != "false" {
		numbered = true
	}
	first := 1
	if v := strings.TrimSpace(b.Attrs["lineno-start"]); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 && n < 1_000_000 {
			first, numbered = n, true
		} else {
			warn("option :lineno-start: is not a number and was ignored")
		}
	} else if v, ok := b.Attrs["lineno-match"]; ok && v != "false" {
		first, numbered = s.first+1, true
	}
	lines := strings.Split(strings.TrimSuffix(s.text, "\n"), "\n")
	if numbered && len(lines) > maxNumbered {
		warn("line numbers are shown for at most 5000 lines and were left out")
		numbered = false
	}
	var code strings.Builder
	if numbered {
		for i, l := range lines {
			code.WriteString(`<span class="zen-include-line"><span class="zen-include-n" aria-hidden="true">` + strconv.Itoa(first+i) + `</span>` + html.EscapeString(l) + "</span>\n")
		}
	} else {
		code.WriteString(html.EscapeString(strings.TrimSuffix(s.text, "\n")) + "\n")
	}
	classes := []string{"zen-block", "zen-include", "zen-include-code"}
	if numbered {
		classes = append(classes, "zen-include-numbered")
	}
	classes = append(classes, userClass...)
	var sb strings.Builder
	sb.WriteString(`<div` + idAttr + ` class="` + strings.Join(classes, " ") + `">`)
	if c := strings.TrimSpace(b.Attrs["caption"]); c != "" {
		sb.WriteString(`<div class="zen-block-head">` + html.EscapeString(c) + `</div>`)
	}
	open := "<code>"
	if lang != "" {
		open = `<code class="language-` + html.EscapeString(lang) + `">`
	}
	sb.WriteString(`<div class="zen-block-body"><pre>` + open + code.String() + "</code></pre></div></div>")
	return sb.String()
}

// selected is the text that is shown and the 0-based index of its first line in the file.
type selected struct {
	text  string
	first int
}

// selectLines applies :lines: (or :start-line: and :end-line:), then
// :start-after: and :end-before:. What cannot be read is ignored with a warning.
func selectLines(text string, attrs map[string]string, warn func(string)) selected {
	lines := strings.SplitAfter(text, "\n")
	if n := len(lines); n > 0 && lines[n-1] == "" {
		lines = lines[:n-1]
	}
	idx := make([]int, len(lines))
	for i := range idx {
		idx[i] = i
	}
	if v := strings.TrimSpace(attrs["lines"]); v != "" {
		if sel, ok := parseLines(v, len(lines)); ok {
			idx = sel
		} else {
			warn("option :lines: is not a list of lines such as 1-3,5 and was ignored")
		}
	} else {
		start, end := 0, len(lines)
		if v := strings.TrimSpace(attrs["start-line"]); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n >= 0 {
				start = min(n, len(lines))
			} else {
				warn("option :start-line: is not a number and was ignored")
			}
		}
		if v := strings.TrimSpace(attrs["end-line"]); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n >= 0 {
				end = min(n, len(lines))
			} else {
				warn("option :end-line: is not a number and was ignored")
			}
		}
		if end < start {
			end = start
		}
		idx = idx[start:end]
	}
	if m := attrs["start-after"]; m != "" {
		found := -1
		for i, li := range idx {
			if strings.Contains(lines[li], m) {
				found = i
				break
			}
		}
		if found < 0 {
			warn("option :start-after: text was not found and the option was ignored")
		} else {
			idx = idx[found+1:]
		}
	}
	if m := attrs["end-before"]; m != "" {
		found := -1
		for i, li := range idx {
			if strings.Contains(lines[li], m) {
				found = i
				break
			}
		}
		if found < 0 {
			warn("option :end-before: text was not found and the option was ignored")
		} else {
			idx = idx[:found]
		}
	}
	var sb strings.Builder
	first := 0
	for i, li := range idx {
		if i == 0 {
			first = li
		}
		sb.WriteString(lines[li])
	}
	return selected{text: sb.String(), first: first}
}

// parseLines reads 1,3,5-7,9- and returns the 0-based indexes, in the order written.
func parseLines(spec string, n int) ([]int, bool) {
	var out []int
	for _, part := range strings.Split(spec, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			return nil, false
		}
		lo, hi := part, part
		if i := strings.Index(part, "-"); i >= 0 {
			lo, hi = strings.TrimSpace(part[:i]), strings.TrimSpace(part[i+1:])
		}
		a, b := 1, n
		var err error
		if lo != "" {
			if a, err = strconv.Atoi(lo); err != nil || a < 1 {
				return nil, false
			}
		}
		if hi != "" {
			if b, err = strconv.Atoi(hi); err != nil || b < 1 {
				return nil, false
			}
		} else if lo == "" {
			return nil, false
		}
		for l := a; l <= b && l <= n; l++ {
			out = append(out, l-1)
		}
	}
	return out, true
}

// readFile reads rel the way a page read goes: a regular file inside the open
// folder, found next to the page or from the folder's root. It returns the
// file's real path (for its extension and the self-inclusion check) and its
// text. A path with .., an absolute path, a symlink that leads out of the
// folder (a dangling one included) and a file over 1 MB are refused. The
// messages never name a path.
func readFile(ctx plugins.Context, rel string) (string, string, error) {
	const none = ""
	if ctx.Workspace == "" {
		return "", none, errors.New("include needs an open folder")
	}
	bad := errors.New("include: the file must be inside the folder (no .., no absolute path)")
	if strings.ContainsAny(rel, "\x00\\") || strings.HasPrefix(rel, "/") || filepath.IsAbs(rel) || len(rel) > 1024 {
		return "", none, bad
	}
	for _, seg := range strings.Split(rel, "/") {
		if seg == ".." {
			return "", none, bad
		}
	}
	root, err := filepath.EvalSymlinks(ctx.Workspace)
	if err != nil {
		return "", none, errors.New("include: cannot reach the folder")
	}
	cands := []string{path.Clean(rel)}
	if dir := path.Dir(ctx.Path); dir != "." && dir != "/" && dir != "" {
		cands = []string{path.Join(dir, rel), cands[0]}
	}
	for _, c := range cands {
		real, err := filepath.EvalSymlinks(filepath.Join(root, filepath.FromSlash(c)))
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				if _, lerr := os.Lstat(filepath.Join(root, filepath.FromSlash(c))); lerr == nil {
					return "", none, bad // a dangling symlink: its target cannot be checked
				}
				continue
			}
			return "", none, errors.New("include: the file cannot be read")
		}
		if real != root && !strings.HasPrefix(real, root+string(filepath.Separator)) {
			return "", none, bad
		}
		f, err := os.Open(real)
		if err != nil {
			return "", none, errors.New("include: the file cannot be read")
		}
		data, err := readCapped(f)
		f.Close()
		if err != nil {
			return "", none, err
		}
		return real, string(data), nil
	}
	return "", none, errors.New("include: the file was not found in the folder")
}

func readCapped(f *os.File) ([]byte, error) {
	fi, err := f.Stat()
	if err != nil || !fi.Mode().IsRegular() {
		return nil, errors.New("include: the path must be a file")
	}
	if fi.Size() > maxFile {
		return nil, errors.New("include: the file is larger than 1 MB")
	}
	data, err := io.ReadAll(io.LimitReader(f, maxFile+1))
	if err != nil {
		return nil, errors.New("include: the file cannot be read")
	}
	if len(data) > maxFile {
		return nil, errors.New("include: the file is larger than 1 MB")
	}
	return data, nil
}

// sameFile reports whether real is the page that is being rendered.
func sameFile(ctx plugins.Context, real string) bool {
	root, err := filepath.EvalSymlinks(ctx.Workspace)
	if err != nil || ctx.Path == "" {
		return false
	}
	page, err := filepath.EvalSymlinks(filepath.Join(root, filepath.FromSlash(ctx.Path)))
	return err == nil && page == real
}

// slug turns a :name: into an id: lower case, letters, digits and . _ : -
// (the same as the callout and figure built-ins).
func slug(s string) string {
	var sb strings.Builder
	dash := false
	for _, r := range strings.ToLower(strings.TrimSpace(s)) {
		if r < 128 && (r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '_' || r == '.' || r == ':') {
			sb.WriteRune(r)
			dash = false
		} else if !dash && sb.Len() > 0 {
			sb.WriteByte('-')
			dash = true
		}
		if sb.Len() >= 100 {
			break
		}
	}
	return strings.Trim(sb.String(), "-")
}
