package myst

import (
	"bytes"
	"io/fs"
	"regexp"
	"strings"

	"github.com/zenmark-org/zenmd/engine/dialects"
	"github.com/zenmark-org/zenmd/engine/plugins"
)

// The syntax of MyST that is not a directive: roles written {name}`content`,
// dollarmath's $…$, target lines, comments, block breaks, substitutions and
// field lists. The dialect finds them and writes plain markdown or spans;
// the plugin that owns a role's name draws it.

var _ dialects.InlineScanner = dialect{}
var _ dialects.Rewriter = dialect{}
var _ dialects.FieldLister = dialect{}
var _ dialects.Extensioner = dialect{}
var _ dialects.DefinitionLists = dialect{}

// DefinitionLists: MyST writes a term, then `: definition`.
func (dialect) DefinitionLists() bool { return true }

func isRoleNameByte(c byte, first bool) bool {
	if c >= 'a' && c <= 'z' {
		return true
	}
	return !first && (c >= '0' && c <= '9' || c == '-' || c == ':' || c == '_' || c == '.')
}

// codeSpanEnd returns the index after the code span opened by the run of n
// backticks at line[i], or after the run itself when no equal run closes it.
func codeSpanEnd(line []byte, i, n int) int {
	for j := i + n; j < len(line); {
		if line[j] != '`' {
			j++
			continue
		}
		m := run(line[j:], '`')
		if m == n {
			return j + m
		}
		j += m
	}
	return i + n
}

// ScanInline finds the roles of one line of a page, outside code spans, and
// with the dollarmath extension on, $…$ inline math (a span named "math").
func (dialect) ScanInline(src []byte, ext dialects.Extensions) []plugins.Span {
	var out []plugins.Span
	for i := 0; i < len(src); {
		switch src[i] {
		case '\\':
			i += 2
			continue
		case '`':
			i = codeSpanEnd(src, i, run(src[i:], '`'))
			continue
		case '{':
			if sp, ok := roleAt(src, i); ok {
				out = append(out, sp)
				i = sp.End
				continue
			}
		case '$':
			if !ext["dollarmath"] {
				break
			}
			if n := run(src[i:], '$'); n > 1 {
				i += n // $$ belongs to the display blocks
				continue
			}
			if sp, ok := dollarAt(src, i); ok {
				out = append(out, sp)
				i = sp.End
				continue
			}
		}
		i++
	}
	return out
}

// roleAt parses {name}`content` at src[i] == '{'.
func roleAt(src []byte, i int) (plugins.Span, bool) {
	j := i + 1
	for j < len(src) && isRoleNameByte(src[j], j == i+1) {
		j++
	}
	if j == i+1 || j >= len(src) || src[j] != '}' || j+1 >= len(src) || src[j+1] != '`' {
		return plugins.Span{}, false
	}
	name := string(src[i+1 : j])
	open := j + 1
	n := run(src[open:], '`')
	start := open + n
	for k := start; k < len(src); {
		switch {
		case src[k] == '\n':
			return plugins.Span{}, false
		case src[k] != '`':
			k++
		default:
			m := run(src[k:], '`')
			if m == n {
				body := src[start:k]
				if len(bytes.TrimSpace(body)) == 0 {
					return plugins.Span{}, false
				}
				return plugins.Span{Start: i, End: k + m, Name: name, Body: string(body), Attrs: map[string]string{}}, true
			}
			k += m
		}
	}
	return plugins.Span{}, false
}

// dollarAt parses $…$ at src[i] with MyST's rules: the TeX does not start or
// end with a space and the closing $ is not followed by a digit, so "$5 and
// $10" is money.
func dollarAt(src []byte, i int) (plugins.Span, bool) {
	if i+1 >= len(src) || src[i+1] == ' ' || src[i+1] == '\t' || src[i+1] == '\n' {
		return plugins.Span{}, false
	}
	for j := i + 1; j < len(src); j++ {
		switch src[j] {
		case '\\':
			j++
		case '\n':
			return plugins.Span{}, false
		case '$':
			if src[j-1] == ' ' || src[j-1] == '\t' || (j+1 < len(src) && (src[j+1] >= '0' && src[j+1] <= '9' || src[j+1] == '$')) {
				continue
			}
			return plugins.Span{Start: i, End: j + 1, Name: "math", Body: string(src[i+1 : j]), Attrs: map[string]string{}}, true
		}
	}
	return plugins.Span{}, false
}

// ---- extensions -----------------------------------------------------------

var identRE = regexp.MustCompile(`[a-z][a-z0-9_]*`)

// Extensions reads the MyST extensions a folder turns on from its root files:
// myst_enable_extensions in a Jupyter Book _config.yml or a Sphinx conf.py,
// enable_extensions in a myst.yml. Text only, 64 KB of each; nothing runs.
func (dialect) Extensions(root fs.FS) dialects.Extensions {
	out := dialects.Extensions{}
	if root == nil {
		return out
	}
	for _, name := range []string{"_config.yml", "myst.yml", "conf.py"} {
		b := readHead(root, name)
		if b == nil {
			continue
		}
		lines := strings.Split(string(b), "\n")
		for i, l := range lines {
			k := strings.Index(l, "enable_extensions")
			if k < 0 || strings.HasPrefix(strings.TrimSpace(l), "#") {
				continue
			}
			block := l[k+len("enable_extensions"):]
			for _, next := range lines[i+1:] {
				t := strings.TrimSpace(next)
				if t != "" && !strings.ContainsAny(t[:1], "-\"'[]#),") {
					break
				}
				block += " " + t
			}
			for _, w := range identRE.FindAllString(block, -1) {
				out[w] = true
			}
		}
	}
	return out
}

// ---- rewrites ---------------------------------------------------------------

var (
	targetRE = regexp.MustCompile(`^ {0,3}\(([^()\n]{1,100})\)=[ \t]*$`)
	breakRE  = regexp.MustCompile(`^ {0,3}\+\+\+([ \t].*)?$`)
	substRE  = regexp.MustCompile(`\{\{\s*([A-Za-z_][A-Za-z0-9_.-]*)\s*\}\}`)
)

const maxSubstitution = 2000

// anchorID turns a target label into an id: lower case, letters, digits and
// . _ : -. MyST compares labels without case or extra spaces, so the id does too.
func anchorID(s string) string {
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

// Rewrite turns target lines, comments, block breaks and {{ substitutions }}
// into markdown. Lines in code are left as they are.
func (dialect) Rewrite(src []byte, page dialects.Page) ([]byte, []string) {
	text := string(src)
	if !strings.Contains(text, "%") && !strings.Contains(text, "+++") && !strings.Contains(text, ")=") && !strings.Contains(text, "{{") {
		return src, nil
	}
	var out strings.Builder
	var warns []string
	warned := map[string]bool{}
	var fenceChar byte
	fenceLen := 0
	prevBlank, prevCode := true, false
	for _, line := range strings.SplitAfter(text, "\n") {
		if line == "" {
			continue
		}
		bare := strings.TrimRight(line, "\r\n")
		eol := line[len(bare):]
		if c, n := fenceOf([]byte(bare)); n > 0 {
			rest := strings.TrimSpace(strings.TrimLeft(strings.TrimLeft(bare, " "), string(c)))
			if fenceChar == 0 {
				fenceChar, fenceLen = c, n
			} else if c == fenceChar && n >= fenceLen && rest == "" {
				fenceChar = 0
			}
			out.WriteString(line)
			prevBlank, prevCode = false, false
			continue
		}
		blank := strings.TrimSpace(bare) == ""
		indented := strings.HasPrefix(bare, "    ") || strings.HasPrefix(bare, "\t")
		code := fenceChar != 0 || (indented && !blank && (prevBlank || prevCode))
		prevBlank, prevCode = blank, code
		if code {
			out.WriteString(line)
			continue
		}
		switch {
		case !indented && strings.HasPrefix(strings.TrimLeft(bare, " "), "%"):
			continue // a comment
		case breakRE.MatchString(bare):
			out.WriteString("\n***\n\n")
			continue
		}
		if m := targetRE.FindStringSubmatch(bare); m != nil {
			if id := anchorID(m[1]); id != "" {
				out.WriteString(`<span id="` + id + `"></span>` + "\n\n")
				continue
			}
		}
		if len(page.Substitutions) > 0 || strings.Contains(bare, "{{") {
			bare = substitute(bare, page.Substitutions, func(k string) {
				if len(page.Substitutions) > 0 && !warned[k] {
					warned[k] = true
					warns = append(warns, "myst: the substitution {{ "+k+" }} is not defined in the page's substitutions: and was left as text")
				}
			})
		}
		out.WriteString(bare + eol)
	}
	return []byte(out.String()), warns
}

// substitute replaces {{ key }} outside code spans. A replacement is text as
// it stands: it is not read for substitutions again.
func substitute(line string, subs map[string]string, missing func(string)) string {
	if !strings.Contains(line, "{{") {
		return line
	}
	var sb strings.Builder
	b := []byte(line)
	last := 0
	for i := 0; i < len(b); {
		switch b[i] {
		case '\\':
			i += 2
			continue
		case '`':
			i = codeSpanEnd(b, i, run(b[i:], '`'))
			continue
		case '{':
			if loc := substRE.FindSubmatchIndex(b[i:]); loc != nil && loc[0] == 0 {
				key := string(b[i+loc[2] : i+loc[3]])
				if v, ok := subs[key]; ok {
					sb.Write(b[last:i])
					sb.WriteString(v)
					last = i + loc[1]
				} else {
					missing(key)
				}
				i += loc[1]
				continue
			}
		}
		i++
	}
	if last == 0 {
		return line
	}
	sb.Write(b[last:])
	return sb.String()
}

// ---- field lists -------------------------------------------------------------

var fieldRE = regexp.MustCompile(`^:([A-Za-z][A-Za-z0-9_-]*):(?:[ \t]+(.*))?$`)

const maxFieldValue = 500

// Fields takes the field list off the top of a page: `:name: value` lines
// (a more indented line continues the value). With none, body comes back as it
// was.
func (dialect) Fields(body []byte) (map[string]string, []byte) {
	lines := strings.SplitAfter(string(body), "\n")
	i := 0
	for i < len(lines) && strings.TrimSpace(lines[i]) == "" {
		i++
	}
	fields := map[string]string{}
	var last string
	for ; i < len(lines); i++ {
		bare := strings.TrimRight(lines[i], "\r\n")
		if m := fieldRE.FindStringSubmatch(bare); m != nil {
			last = m[1]
			fields[last] = strings.TrimSpace(m[2])
			continue
		}
		if last != "" && (strings.HasPrefix(bare, "  ") || strings.HasPrefix(bare, "\t")) && strings.TrimSpace(bare) != "" {
			fields[last] = strings.TrimSpace(fields[last] + " " + strings.TrimSpace(bare))
			continue
		}
		break
	}
	if len(fields) == 0 || (i < len(lines) && strings.TrimSpace(lines[i]) != "") {
		return nil, body
	}
	for k, v := range fields {
		if len(v) > maxFieldValue {
			fields[k] = v[:maxFieldValue]
		}
	}
	for i < len(lines) && strings.TrimSpace(lines[i]) == "" {
		i++
	}
	return fields, []byte(strings.Join(lines[i:], ""))
}
