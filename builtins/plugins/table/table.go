// Package table renders the MyST directives {table}, {list-table} and
// {csv-table}, and the same blocks written as :::table, :::list-table and
// :::csv-table. Each gives a framed table with an optional caption, numbered
// "Table 1", "Table 2" … per page.
//
//   - table: the argument is the caption, the body a markdown table.
//   - list-table: a two-level bullet list, one top-level item per row.
//   - csv-table: CSV in the body, or a file of the folder with :file:.
//
// Cells are markdown, rendered by the page's own pipeline (Context.Markdown).
// The output is plain HTML that reads without scripts. :file: reads only a
// file inside the open folder: no .., no absolute path, no symlink leading
// out, at most 1 MB.
package table

import (
	"embed"
	"encoding/csv"
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

//go:embed assets/table.css
var assets embed.FS

const (
	maxFile  = 1 << 20 // the most a :file: may hold
	maxCells = 20000   // the most cells one table may have
)

type plugin struct{}

// New returns the table plugin.
func New() plugins.Plugin { return plugin{} }

func (plugin) Manifest() plugins.Manifest {
	return plugins.Manifest{
		ID:          "table",
		Version:     "0.1.0",
		APIVersion:  plugins.APIVersion,
		Description: "Tables with captions: table, list-table and csv-table",
		Default:     true,
		Help: plugins.Help{
			Summary: "A table with an optional caption, numbered per page (Table 1, Table 2). table takes a markdown table in its body, list-table a bullet list of bullet lists (one item per row) and csv-table CSV text, or file= for a CSV file of the folder (at most 1 MB). Options follow MyST: header-rows, stub-columns, widths, align, name and, for csv-table, header and delim.",
			Snippets: []plugins.Snippet{
				{Title: "A table with a caption", Markdown: ":::table{title=\"Stock levels\" name=stock}\n| Name | Qty |\n|------|-----|\n| Ink  | 3   |\n:::\n"},
				{Title: "A list table, MyST form", Markdown: "```{list-table} Supplies\n:header-rows: 1\n:widths: 30 70\n\n* - Item\n  - Notes\n* - Ink\n  - Black, refillable\n```\n"},
				{Title: "A CSV table", Markdown: ":::csv-table{title=Prices header-rows=1}\nItem, Price\nApple, 1.20\nPear, 2.50\n:::\n"},
				{Title: "A CSV file of the folder, MyST form", Markdown: "```{csv-table} Prices\n:file: data/prices.csv\n:header-rows: 1\n```\n"},
			},
		},
		Blocks: []string{"table", "list-table", "csv-table"},
	}
}

func (plugin) Assets() fs.FS {
	sub, err := fs.Sub(assets, "assets")
	if err != nil {
		panic(err)
	}
	return sub
}

func (plugin) Entry() plugins.Assets { return plugins.Assets{CSS: []string{"table.css"}} }

// numMarker stands in the caption until HTML numbers the tables of the page.
const numMarker = `<span class="zt-num"></span>`

// HTML numbers the tables of the finished page in the order they appear; the
// numbers belong to one render, so every page starts at 1.
func (plugin) HTML(_ plugins.Context, s string) (string, error) {
	if !strings.Contains(s, numMarker) {
		return s, nil
	}
	var sb strings.Builder
	n := 0
	for {
		i := strings.Index(s, numMarker)
		if i < 0 {
			break
		}
		n++
		sb.WriteString(s[:i])
		sb.WriteString(`<span class="zt-num">` + strconv.Itoa(n) + `</span>`)
		s = s[i+len(numMarker):]
	}
	sb.WriteString(s)
	return sb.String(), nil
}

var base = []string{"arg", "title", "name", "class", "align", "width", "nonumber", "numbered"}

// supported options per block name; anything else is shown as ignored.
var known = map[string]map[string]bool{
	"table":      set(base...),
	"list-table": set(append([]string{"header-rows", "stub-columns", "widths"}, base...)...),
	"csv-table":  set(append([]string{"header-rows", "stub-columns", "widths", "header", "delim", "file"}, base...)...),
}

func set(k ...string) map[string]bool {
	m := make(map[string]bool, len(k))
	for _, s := range k {
		m[s] = true
	}
	return m
}

var (
	classRE = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_-]{0,40}$`)
	keyRE   = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_-]*$`)
	dimRE   = regexp.MustCompile(`^([0-9]{1,5})(px)?$`)
	pctRE   = regexp.MustCompile(`^([0-9]{1,3})%$`)
	numRE   = regexp.MustCompile(`^[0-9]{1,6}$`)
	itemRE  = regexp.MustCompile(`^([*+-])( +|$)`)
	listRE  = regexp.MustCompile(`^[0-9]+[.)]( |$)`)
)

// notes reports what was ignored as page warnings (ctx.Warn).
type notes struct{ warn func(string) }

func (n *notes) add(key, why string) { n.say("option :" + key + ": " + why + " and was ignored") }

func (n *notes) say(msg string) {
	if n != nil && n.warn != nil {
		n.warn(msg)
	}
}

func (plugin) RenderBlock(ctx plugins.Context, b plugins.Block) (string, error) {
	opts := known[b.Name]
	if opts == nil {
		return "", fmt.Errorf("unknown block %q", b.Name)
	}
	attrs, body := yamlOptions(b.Attrs, b.Body)
	ns := notes{warn: ctx.Warn}
	keys := make([]string, 0, len(attrs))
	for k := range attrs {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if !opts[k] {
			ns.add(k, "is not supported")
		}
	}

	classes := []string{"zen-block", "zt"}
	switch a := strings.ToLower(strings.TrimSpace(attrs["align"])); a {
	case "left", "center", "right":
		classes = append(classes, "zt-"+a)
	case "", "default":
	default:
		ns.add("align", "has a value that is not supported (left, center or right)")
	}
	for _, c := range strings.Fields(attrs["class"]) {
		if classRE.MatchString(c) && !strings.HasPrefix(c, "zen") && !strings.HasPrefix(c, "zt") {
			classes = append(classes, c)
		} else {
			ns.add("class", "has a class name that is not supported")
		}
	}
	idAttr := ""
	if n := attrs["name"]; n != "" {
		if id := slug(n); id != "" {
			idAttr = ` id="` + id + `"`
		} else {
			ns.add("name", "has a value that is not supported")
		}
	}
	widthAttr := ""
	if w := strings.TrimSpace(attrs["width"]); w != "" {
		if m := pctRE.FindStringSubmatch(w); m != nil {
			widthAttr = ` width="` + m[1] + `%"`
		} else if m := dimRE.FindStringSubmatch(w); m != nil {
			widthAttr = ` width="` + m[1] + `"`
		} else {
			ns.add("width", "has a value that is not supported (use px or %)")
		}
	}
	caption := ""
	if t := strings.TrimSpace(attrs["title"]); t != "" {
		caption = `<caption>`
		if numbered(attrs) {
			caption += `<span class="zt-label">Table ` + numMarker + `</span>: `
		}
		caption += inline(ctx, t) + `</caption>`
	}

	var content string
	if b.Name == "table" {
		out := markdown(ctx, body)
		i := strings.Index(out, "<table>")
		if i < 0 {
			ns.say("the body has no markdown table")
			content = out
		} else {
			content = out[:i] + "<table" + idAttr + widthAttr + ">" + caption + out[i+len("<table>"):]
		}
	} else {
		var rows [][]string
		var err error
		header := 0
		if b.Name == "list-table" {
			rows = parseList(body)
		} else {
			rows, header, err = csvRows(ctx, attrs, body, &ns)
			if err != nil {
				return "", err
			}
		}
		if rowsCells(rows) > maxCells {
			return "", fmt.Errorf("the table has more than %d cells", maxCells)
		}
		if v := strings.TrimSpace(attrs["header-rows"]); v != "" {
			if numRE.MatchString(v) {
				n, _ := strconv.Atoi(v)
				header += n
			} else {
				ns.add("header-rows", "has a value that is not a number")
			}
		}
		stub := 0
		if v := strings.TrimSpace(attrs["stub-columns"]); v != "" {
			if numRE.MatchString(v) {
				stub, _ = strconv.Atoi(v)
			} else {
				ns.add("stub-columns", "has a value that is not a number")
			}
		}
		if len(rows) == 0 {
			ns.say("the table has no rows")
		}
		content = gridHTML(ctx, rows, header, stub, attrs["widths"], idAttr, widthAttr, caption, &ns)
	}

	return `<div class="` + strings.Join(classes, " ") + `"><div class="zen-block-body"><div class="zt-scroll">` +
		content + `</div></div></div>`, nil
}

// gridHTML draws rows as a table: header rows first, then the body, with the
// first stub columns of the body as row headers.
func gridHTML(ctx plugins.Context, rows [][]string, header, stub int, widths, idAttr, widthAttr, caption string, ns *notes) string {
	cols := 0
	for _, r := range rows {
		cols = max(cols, len(r))
	}
	header = min(header, len(rows))
	var sb strings.Builder
	sb.WriteString("<table" + idAttr + widthAttr + ">" + caption)
	if w := columnWidths(widths, cols, ns); w != nil {
		sb.WriteString("<colgroup>")
		for _, p := range w {
			sb.WriteString(`<col width="` + strconv.Itoa(p) + `%">`)
		}
		sb.WriteString("</colgroup>")
	}
	cell := func(tag, scope, text string) {
		sb.WriteString("<" + tag + scope + ">" + cellHTML(ctx, text) + "</" + tag + ">")
	}
	row := func(r []string, head bool) {
		sb.WriteString("<tr>")
		for c := 0; c < cols; c++ {
			text := ""
			if c < len(r) {
				text = r[c]
			}
			switch {
			case head:
				cell("th", ` scope="col"`, text)
			case c < stub:
				cell("th", ` scope="row"`, text)
			default:
				cell("td", "", text)
			}
		}
		sb.WriteString("</tr>")
	}
	if header > 0 {
		sb.WriteString("<thead>")
		for _, r := range rows[:header] {
			row(r, true)
		}
		sb.WriteString("</thead>")
	}
	if header < len(rows) {
		sb.WriteString("<tbody>")
		for _, r := range rows[header:] {
			row(r, false)
		}
		sb.WriteString("</tbody>")
	}
	sb.WriteString("</table>")
	return sb.String()
}

// columnWidths reads :widths: ("auto" and "grid" leave the browser to choose)
// as relative numbers and returns whole percents, nil when there are none.
func columnWidths(v string, cols int, ns *notes) []int {
	v = strings.TrimSpace(v)
	if v == "" || v == "auto" || v == "grid" || cols == 0 {
		return nil
	}
	f := strings.FieldsFunc(v, func(r rune) bool { return r == ',' || r == ' ' || r == '\t' })
	sum := 0
	nums := make([]int, len(f))
	for i, s := range f {
		if !numRE.MatchString(s) {
			ns.add("widths", "has a value that is not supported (numbers, auto or grid)")
			return nil
		}
		nums[i], _ = strconv.Atoi(s)
		sum += nums[i]
	}
	if len(f) != cols || sum == 0 {
		ns.add("widths", "does not match the number of columns")
		return nil
	}
	out := make([]int, cols)
	for i, n := range nums {
		out[i] = (n*100 + sum/2) / sum
	}
	return out
}

func rowsCells(rows [][]string) int {
	n := 0
	for _, r := range rows {
		n += len(r)
	}
	return n
}

// parseList reads a two-level bullet list: each top-level item is a row, each
// item inside it a cell. Text under an item belongs to it, and so do blank
// lines, so a cell may hold several paragraphs or a list.
func parseList(body string) [][]string {
	var rows [][]string
	for _, block := range items(body) {
		cells := items(block)
		if len(cells) == 0 {
			cells = []string{strings.TrimSpace(block)} // an item that is not a list is one cell
		}
		for i := range cells {
			cells[i] = strings.TrimSpace(cells[i])
		}
		rows = append(rows, cells)
	}
	return rows
}

// items splits text into the bodies of its bullet items at indent 0. Lines
// under an item lose the indent of its text. Fenced code keeps its lines.
func items(text string) []string {
	var out []string
	var cur []string
	off := 0
	in := false
	fence := ""
	flush := func() {
		if in {
			out = append(out, strings.Join(cur, "\n"))
		}
		cur, in = nil, false
	}
	for _, l := range strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n") {
		if m := itemRE.FindStringSubmatch(l); m != nil && fence == "" {
			flush()
			in, off = true, len(m[0])
			cur = []string{l[len(m[0]):]}
			continue
		}
		if !in {
			continue
		}
		t := strings.TrimLeft(l, " ")
		if fence == "" {
			if strings.HasPrefix(t, "```") || strings.HasPrefix(t, "~~~") {
				fence = t[:3]
			}
		} else if strings.HasPrefix(t, fence) {
			fence = ""
		}
		k := 0
		for k < len(l) && k < off && l[k] == ' ' {
			k++
		}
		cur = append(cur, l[k:])
	}
	flush()
	return out
}

// csvRows reads the table of a csv-table: the :file: when there is one, else
// the body. It returns the rows and how many of them (the :header: row) are
// header rows already.
func csvRows(ctx plugins.Context, attrs map[string]string, body string, ns *notes) ([][]string, int, error) {
	delim := ','
	if d := attrs["delim"]; d != "" {
		switch r := []rune(d); {
		case d == "tab":
			delim = '\t'
		case d == "space":
			delim = ' '
		case len(r) == 1 && r[0] != '"' && r[0] != '\n' && r[0] != '\r' && r[0] < 0xFFFD:
			delim = r[0]
		default:
			ns.add("delim", "has a value that is not supported (one character, tab or space)")
		}
	}
	text := body
	if f := strings.TrimSpace(attrs["file"]); f != "" {
		b, err := readFile(ctx, f)
		if err != nil {
			return nil, 0, err
		}
		text = b
	}
	read := func(s string) ([][]string, error) {
		r := csv.NewReader(strings.NewReader(s))
		r.Comma = delim
		r.LazyQuotes = true
		r.TrimLeadingSpace = true
		r.FieldsPerRecord = -1
		return r.ReadAll()
	}
	rows, err := read(strings.TrimSpace(text))
	if err != nil {
		return nil, 0, errors.New("the CSV text cannot be read")
	}
	header := 0
	if h := strings.TrimSpace(attrs["header"]); h != "" {
		hr, err := read(h)
		if err != nil || len(hr) == 0 {
			ns.add("header", "has text that is not CSV")
		} else {
			rows = append(hr[:1], rows...)
			header = 1
		}
	}
	return rows, header, nil
}

// readFile reads :file: of a csv-table the way a page read goes: a regular
// file inside the open folder, found next to the page or from the folder's
// root. A path with .., an absolute path, a symlink that leads out of the
// folder and a file over 1 MB are refused. The messages never name a path.
func readFile(ctx plugins.Context, rel string) (string, error) {
	if ctx.Workspace == "" {
		return "", errors.New("csv-table :file: needs an open folder")
	}
	bad := errors.New("csv-table :file: must be a file inside the folder (no .., no absolute path)")
	if strings.ContainsAny(rel, "\x00\\") || strings.HasPrefix(rel, "/") || filepath.IsAbs(rel) || len(rel) > 1024 {
		return "", bad
	}
	for _, seg := range strings.Split(rel, "/") {
		if seg == ".." {
			return "", bad
		}
	}
	root, err := filepath.EvalSymlinks(ctx.Workspace)
	if err != nil {
		return "", errors.New("csv-table :file: cannot reach the folder")
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
					return "", bad // a dangling symlink: its target cannot be checked
				}
				continue
			}
			return "", errors.New("csv-table :file: cannot be read")
		}
		if real != root && !strings.HasPrefix(real, root+string(filepath.Separator)) {
			return "", bad
		}
		f, err := os.Open(real)
		if err != nil {
			return "", errors.New("csv-table :file: cannot be read")
		}
		data, err := readCapped(f)
		f.Close()
		if err != nil {
			return "", err
		}
		return string(data), nil
	}
	return "", errors.New("csv-table :file: was not found in the folder")
}

func readCapped(f *os.File) ([]byte, error) {
	fi, err := f.Stat()
	if err != nil || !fi.Mode().IsRegular() {
		return nil, errors.New("csv-table :file: must be a file")
	}
	if fi.Size() > maxFile {
		return nil, errors.New("csv-table :file: is larger than 1 MB")
	}
	data, err := io.ReadAll(io.LimitReader(f, maxFile+1))
	if err != nil {
		return nil, errors.New("csv-table :file: cannot be read")
	}
	if len(data) > maxFile {
		return nil, errors.New("csv-table :file: is larger than 1 MB")
	}
	return data, nil
}

// cellHTML renders one cell: markdown through the page, except that plain
// text (most cells) is only escaped.
func cellHTML(ctx plugins.Context, s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if !needsMarkdown(s) {
		return html.EscapeString(s)
	}
	return inline(ctx, s)
}

func needsMarkdown(s string) bool {
	if strings.ContainsAny(s, "*_`[]<>&\\!~|$#{}\n") {
		return true
	}
	if i := strings.IndexByte(s, ':'); i >= 0 && i+1 < len(s) && (s[i+1] >= 'a' && s[i+1] <= 'z' || s[i+1] >= 'A' && s[i+1] <= 'Z') {
		return true
	}
	return s[0] == '-' || s[0] == '+' || s[0] == '=' || listRE.MatchString(s)
}

func numbered(attrs map[string]string) bool {
	if v, ok := attrs["nonumber"]; ok && v != "false" {
		return false
	}
	switch strings.ToLower(attrs["numbered"]) {
	case "false", "no", "0", "off":
		return false
	}
	return true
}

func markdown(ctx plugins.Context, s string) string {
	if ctx.Markdown == nil {
		return "<p>" + html.EscapeString(s) + "</p>"
	}
	out, err := ctx.Markdown(s)
	if err != nil {
		return "<p>" + html.EscapeString(s) + "</p>"
	}
	return out
}

// inline renders s and drops the paragraph around it when it is only one.
func inline(ctx plugins.Context, s string) string {
	out := strings.TrimSpace(markdown(ctx, s))
	if strings.HasPrefix(out, "<p>") && strings.HasSuffix(out, "</p>") && strings.Count(out, "<p>") == 1 {
		return strings.ReplaceAll(out[3:len(out)-4], "\n", " ")
	}
	return out
}

// slug turns a :name: into an id: lower case, letters, digits and . _ : -.
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

// yamlOptions reads a body that starts with a --- block of `key: value` lines,
// MyST's other way of writing options. Options given as :key: lines win.
func yamlOptions(in map[string]string, body string) (map[string]string, string) {
	out := make(map[string]string, len(in)+4)
	for k, v := range in {
		out[k] = v
	}
	t := strings.TrimLeft(body, "\r\n")
	if !strings.HasPrefix(t, "---\n") && !strings.HasPrefix(t, "---\r\n") {
		return out, body
	}
	lines := strings.Split(t, "\n")
	end := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimRight(lines[i], " \t\r") == "---" {
			end = i
			break
		}
	}
	if end < 0 {
		return out, body
	}
	for _, l := range lines[1:end] {
		k, v, ok := strings.Cut(l, ":")
		k = strings.TrimSpace(k)
		if !ok || !keyRE.MatchString(k) {
			continue
		}
		v = strings.TrimSpace(v)
		if len(v) >= 2 && (v[0] == '"' && v[len(v)-1] == '"' || v[0] == '\'' && v[len(v)-1] == '\'') {
			v = v[1 : len(v)-1]
		}
		if v == "" {
			v = "true"
		}
		if _, set := out[k]; !set {
			out[k] = v
		}
	}
	return out, strings.Join(lines[end+1:], "\n")
}
