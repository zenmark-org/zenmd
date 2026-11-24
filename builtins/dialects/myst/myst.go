// Package myst is the MyST dialect (specs/engine/dialects): the directive
// syntax of MyST, Jupyter Book and Sphinx.
//
//	```{note} Remember
//	:class: wide
//	Body text.
//	```
//
// is the container plugin that owns block "note", with title "Remember" and
// class "wide", and :::{note} Remember … ::: is the same in MyST's colon form.
// {code-block}, {code} and {code-cell} are plain code. Scan is one pass over
// the page and finds the outermost directives; engine/render scans a
// directive's body again when it renders it, so nesting costs what it costs
// for `:::` containers.
package myst

import (
	"bytes"
	"io"
	"io/fs"
	"regexp"

	"github.com/zenmark-org/zenmd/engine/dialects"
	"github.com/zenmark-org/zenmd/engine/plugins"
)

type dialect struct{}

// New returns the MyST dialect.
func New() dialects.Dialect { return dialect{} }

func (dialect) ID() string   { return "myst" }
func (dialect) Name() string { return "MyST" }
func (dialect) Summary() string {
	return "Directives such as ```{note} and :::{note}, as written for MyST, Jupyter Book and Sphinx."
}

var (
	keyRE  = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_-]*$`)
	langRE = regexp.MustCompile(`^[A-Za-z0-9_+#.-]{1,32}$`)
)

var codeNames = map[string]bool{"code-block": true, "code": true, "code-cell": true}

// Scan finds the directives of src, outside ordinary code fences.
func (dialect) Scan(src []byte) []dialects.Span {
	var out []dialects.Span
	noCloser := false // no line from some point on ends with $$: later $$ openers cannot close either
	for pos := 0; pos < len(src); {
		ls := pos
		text, next := cutLine(src, pos)
		pos = next
		ind := 0
		for ind < len(text) && ind < 3 && text[ind] == ' ' {
			ind++
		}
		rest := text[ind:]
		if bytes.HasPrefix(rest, []byte("$$")) {
			if !noCloser {
				sp, after, found, dead := dollarBlock(src, ls, ind, next)
				if dead {
					noCloser = true
				}
				if found {
					out = append(out, sp)
					pos = after
				}
			}
			continue
		}
		if len(rest) == 0 || (rest[0] != '`' && rest[0] != '~' && rest[0] != ':') {
			continue
		}
		c := rest[0]
		n := run(rest, c)
		if n < 3 {
			continue
		}
		info := bytes.TrimSpace(rest[n:])
		var name, arg string
		var ok bool
		switch {
		case c == ':':
			name, arg, ok = opener(info)
			if !ok {
				continue // zen's own :::name is not this dialect's
			}
		default:
			if c == '`' && bytes.IndexByte(info, '`') >= 0 {
				continue // not a fence
			}
			name, arg, ok = opener(info)
		}
		// the closer: the same character, at least as long, nothing after it
		end, bodyEnd := len(src), len(src)
		after := len(src)
		for j := next; j < len(src); {
			cs := j
			t, nx := cutLine(src, j)
			if isCloser(t, c, n) {
				bodyEnd, end, after = cs, cs+len(trimEOL(src[cs:nx])), nx
				break
			}
			j = nx
			if c == ':' { // a code fence in a colon fence keeps its colons
				if fc, fn := fenceOf(t); fn > 0 {
					for j < len(src) {
						t2, nx2 := cutLine(src, j)
						j = nx2
						if isCloser(t2, fc, fn) {
							break
						}
					}
				}
			}
		}
		if bodyEnd == len(src) { // unclosed: to the end of the page
			end = trimmedLen(src)
		}
		if !ok { // an ordinary fence: its content is code
			if c == ':' {
				continue
			}
			pos = after
			continue
		}
		out = append(out, build(src, ls, end, next, bodyEnd, name, arg))
		pos = after
	}
	return out
}

// labelRE is the (label) MyST's dollarmath allows after the closing $$.
var labelRE = regexp.MustCompile(`[ \t]*\(([A-Za-z0-9_:.-]{1,100})\)[ \t]*$`)

// closing reports whether line ends a $$ block: it ends with $$, optionally
// followed by a (label). It returns the text before the $$ and the label.
func closing(line []byte) (before []byte, label string, ok bool) {
	line = bytes.TrimRight(line, " \t")
	if m := labelRE.FindSubmatchIndex(line); m != nil && bytes.HasSuffix(bytes.TrimRight(line[:m[0]], " \t"), []byte("$$")) {
		label = string(line[m[2]:m[3]])
		line = bytes.TrimRight(line[:m[0]], " \t")
	}
	if !bytes.HasSuffix(line, []byte("$$")) {
		return nil, "", false
	}
	return line[:len(line)-2], label, true
}

// dollarBlock reads a $$ display math block whose first line starts at ls
// (the $$ after ind spaces). It is a block when a line ending with $$ closes
// it; the text between is the TeX, never empty and with no $$ inside. dead
// reports that no later line ends with $$, so the caller need not look again.
func dollarBlock(src []byte, ls, ind, next int) (sp dialects.Span, after int, found, dead bool) {
	first, _ := cutLine(src, ls)
	first = first[ind+2:]
	var parts [][]byte
	var label string
	end := 0
	if before, l, ok := closing(first); ok && len(bytes.TrimSpace(before)) > 0 {
		parts, label, end, after, found = [][]byte{before}, l, ls+len(trimEOL(src[ls:next])), next, true
	} else if bytes.Contains(first, []byte("$$")) {
		return sp, 0, false, false
	} else {
		parts = [][]byte{first}
		for j := next; j < len(src); {
			cs := j
			t, nx := cutLine(src, j)
			if before, l, ok := closing(t); ok {
				parts, label = append(parts, before), l
				end, after, found = cs+len(trimEOL(src[cs:nx])), nx, true
				break
			}
			parts = append(parts, t)
			j = nx
		}
		if !found {
			return sp, 0, false, true
		}
	}
	body := bytes.TrimSpace(bytes.Join(parts, []byte("\n")))
	if len(body) == 0 || bytes.Contains(body, []byte("$$")) {
		return sp, 0, false, false
	}
	attrs := map[string]string{}
	if label != "" {
		attrs["label"] = label
	}
	return dialects.Span{Span: plugins.Span{Start: ls, End: end, Name: "math", Body: string(body), Attrs: attrs}}, after, true, false
}

func trimmedLen(src []byte) int { return len(trimEOL(src)) }

func trimEOL(b []byte) []byte {
	b = bytes.TrimSuffix(b, []byte("\n"))
	return bytes.TrimSuffix(b, []byte("\r"))
}

// cutLine returns the line at pos without its terminator and the position of
// the next line.
func cutLine(src []byte, pos int) (text []byte, next int) {
	next = len(src)
	if nl := bytes.IndexByte(src[pos:], '\n'); nl >= 0 {
		next = pos + nl + 1
	}
	return trimEOL(src[pos:next]), next
}

func run(b []byte, c byte) int {
	n := 0
	for n < len(b) && b[n] == c {
		n++
	}
	return n
}

// opener parses `{name} argument` (the text after the fence).
func opener(info []byte) (name, arg string, ok bool) {
	if len(info) < 3 || info[0] != '{' || info[1] < 'a' || info[1] > 'z' {
		return "", "", false
	}
	i := 2
	for i < len(info) && (info[i] >= 'a' && info[i] <= 'z' || info[i] >= '0' && info[i] <= '9' || info[i] == '-') {
		i++
	}
	if i >= len(info) || info[i] != '}' {
		return "", "", false
	}
	return string(info[1:i]), string(bytes.TrimSpace(info[i+1:])), true
}

// fenceOf reports a code fence opening at the start of line: its character and
// length (0 when it is none).
func fenceOf(line []byte) (byte, int) {
	ind := 0
	for ind < len(line) && ind < 3 && line[ind] == ' ' {
		ind++
	}
	line = line[ind:]
	if len(line) == 0 || (line[0] != '`' && line[0] != '~') {
		return 0, 0
	}
	n := run(line, line[0])
	if n < 3 || (line[0] == '`' && bytes.IndexByte(line[n:], '`') >= 0) {
		return 0, 0
	}
	return line[0], n
}

func isCloser(line []byte, c byte, n int) bool {
	ind := 0
	for ind < len(line) && ind < 3 && line[ind] == ' ' {
		ind++
	}
	line = line[ind:]
	m := run(line, c)
	return m >= n && len(bytes.TrimRight(line[m:], " \t")) == 0
}

func build(src []byte, start, end, bodyStart, bodyEnd int, name, arg string) dialects.Span {
	body := src[bodyStart:bodyEnd]
	attrs := map[string]string{}
	var warns []string
	for len(body) > 0 {
		text, nx := cutLine(body, 0)
		key, val, isOpt := option(text)
		if !isOpt {
			break
		}
		if keyRE.MatchString(key) {
			if val == "" {
				val = "true"
			}
			attrs[key] = val
		} else {
			warns = append(warns, "myst: option :"+key+": is not a valid attribute name and was dropped")
		}
		body = body[nx:]
	}
	sp := dialects.Span{Span: plugins.Span{Start: start, End: end, Name: name, Body: string(body), Attrs: attrs}, Warnings: warns}
	if codeNames[name] {
		sp.Code = true
		if name == "code-cell" {
			sp.Warnings = append(sp.Warnings, "myst: the directive {code-cell} is not supported (nothing runs) and stays as code")
		}
		if f := bytes.Fields([]byte(arg)); len(f) > 0 && langRE.Match(f[0]) {
			sp.Lang = string(f[0])
		}
		sp.Attrs = map[string]string{}
		return sp
	}
	if arg != "" {
		if _, set := attrs["title"]; !set {
			attrs["title"] = arg
		}
		attrs["arg"] = arg
	}
	return sp
}

// option parses a `:key: value` line.
func option(line []byte) (key, val string, ok bool) {
	if len(line) < 3 || line[0] != ':' {
		return "", "", false
	}
	k := bytes.IndexByte(line[1:], ':')
	if k < 1 {
		return "", "", false
	}
	k++
	if bytes.ContainsAny(line[1:k], " \t") {
		return "", "", false
	}
	if k+1 < len(line) && line[k+1] != ' ' && line[k+1] != '\t' {
		return "", "", false
	}
	return string(line[1:k]), string(bytes.TrimSpace(line[k+1:])), true
}

// maxDetect is how much of a root file Detect reads.
const maxDetect = 64 * 1024

// Detect reports whether the root holds a myst.yml, a Jupyter Book _config.yml
// with a parse: or myst_ key, or a conf.py that mentions myst_parser. It opens
// those three names only, reads at most 64 KB of each as text and runs nothing.
func (dialect) Detect(root fs.FS) bool {
	if root == nil {
		return false
	}
	if f, err := root.Open("myst.yml"); err == nil {
		f.Close()
		return true
	}
	if b := readHead(root, "_config.yml"); b != nil {
		for _, l := range bytes.Split(b, []byte("\n")) {
			l = bytes.TrimRight(l, " \t\r")
			if bytes.Equal(l, []byte("parse:")) || bytes.HasPrefix(bytes.TrimLeft(l, " \t"), []byte("myst_")) {
				return true
			}
		}
	}
	if b := readHead(root, "conf.py"); b != nil && bytes.Contains(b, []byte("myst_parser")) {
		return true
	}
	return false
}

func readHead(root fs.FS, name string) []byte {
	f, err := root.Open(name)
	if err != nil {
		return nil
	}
	defer f.Close()
	b, err := io.ReadAll(io.LimitReader(f, maxDetect))
	if err != nil {
		return nil
	}
	return b
}
