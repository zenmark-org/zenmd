package structure

import (
	"bytes"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	east "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"

	"github.com/zenmark-org/zenmd/engine/config"
	"github.com/zenmark-org/zenmd/engine/plugins"
)

// node is one top-level block of a page.
type node struct {
	kind    string // heading, paragraph, list, code, table, image, block, break, quote, html, other
	line    int
	level   int // heading
	text    string
	id      string // heading: the id the page gives it
	words   int
	ordered bool
	task    bool
	items   int
	lang    string // code
	name    string // block
	attrs   map[string]string
	rows    int
	cols    []string
	alt     string // image
	fenced  bool
}

type image struct {
	line int
	alt  string
}

// Doc is a page as the checker reads it.
type Doc struct {
	FM      map[string]any
	fmLines map[string]int
	nodes   []node
	images  []image
	words   int
	hasH1   bool
}

var fmKeyRE = regexp.MustCompile(`^([A-Za-z0-9_-]+)\s*:`)
var placeholderRE = regexp.MustCompile(`^<!--zb:(\d+)-->`)

func countWords(s string) int { return len(strings.Fields(s)) }

// Analyse reads the page source: its frontmatter and the blocks of its body.
// A `:::name` container is one block, as the renderer sees it.
func Analyse(src []byte) *Doc {
	d := &Doc{fmLines: map[string]int{}}
	meta, body, _ := config.SplitFrontmatter(src)
	d.FM = meta
	base := 0
	if len(body) < len(src) {
		base = bytes.Count(src[:len(src)-len(body)], []byte("\n"))
		for i, line := range strings.Split(string(src[:len(src)-len(body)]), "\n") {
			if m := fmKeyRE.FindStringSubmatch(line); m != nil && i > 0 {
				if _, seen := d.fmLines[m[1]]; !seen {
					d.fmLines[m[1]] = i + 1
				}
			}
		}
	}

	// Containers become a one-line HTML comment, so line numbers stay.
	var spans []plugins.Span
	var mod bytes.Buffer
	last := 0
	for _, sp := range plugins.ScanContainers(body) {
		if sp.Start < last {
			continue
		}
		mod.Write(body[last:sp.Start])
		fmt.Fprintf(&mod, "<!--zb:%d-->", len(spans))
		mod.Write(bytes.Repeat([]byte("\n"), bytes.Count(body[sp.Start:sp.End], []byte("\n"))))
		spans = append(spans, sp)
		last = sp.End
	}
	mod.Write(body[last:])
	b := mod.Bytes()

	starts := []int{0}
	for i, c := range b {
		if c == '\n' {
			starts = append(starts, i+1)
		}
	}
	lineAt := func(off int) int {
		if off < 0 {
			off = 0
		}
		return base + sort.Search(len(starts), func(i int) bool { return starts[i] > off })
	}

	md := goldmark.New(
		goldmark.WithExtensions(extension.GFM, extension.Footnote),
		goldmark.WithParserOptions(parser.WithAutoHeadingID()),
	)
	doc := md.Parser().Parse(text.NewReader(b))

	var first func(n ast.Node) (int, bool)
	first = func(n ast.Node) (int, bool) {
		if n.Type() == ast.TypeBlock && n.Lines().Len() > 0 {
			return n.Lines().At(0).Start, true
		}
		if t, ok := n.(*ast.Text); ok {
			return t.Segment.Start, true
		}
		for c := n.FirstChild(); c != nil; c = c.NextSibling() {
			if off, ok := first(c); ok {
				return off, true
			}
		}
		return 0, false
	}
	var plain func(n ast.Node, sb *strings.Builder)
	plain = func(n ast.Node, sb *strings.Builder) {
		for c := n.FirstChild(); c != nil; c = c.NextSibling() {
			switch v := c.(type) {
			case *ast.Text:
				sb.Write(v.Segment.Value(b))
				if v.SoftLineBreak() || v.HardLineBreak() {
					sb.WriteByte(' ')
				}
			case *ast.String:
				sb.Write(v.Value)
			case *ast.CodeBlock, *ast.FencedCodeBlock, *ast.HTMLBlock:
			default:
				plain(c, sb)
				if c.Type() == ast.TypeBlock {
					sb.WriteByte(' ')
				}
			}
		}
	}
	plainOf := func(n ast.Node) string {
		var sb strings.Builder
		plain(n, &sb)
		return strings.TrimSpace(sb.String())
	}

	prevLine := base
	for n := doc.FirstChild(); n != nil; n = n.NextSibling() {
		nd := node{kind: "other"}
		if off, ok := first(n); ok {
			nd.line = lineAt(off)
		}
		switch v := n.(type) {
		case *ast.Heading:
			nd.kind, nd.level, nd.text = "heading", v.Level, plainOf(v)
			if id, ok := v.AttributeString("id"); ok {
				if bs, ok := id.([]byte); ok {
					nd.id = string(bs)
				}
			}
			nd.words = countWords(nd.text)
			if v.Level == 1 {
				d.hasH1 = true
			}
		case *ast.Paragraph:
			nd.kind, nd.text = "paragraph", plainOf(v)
			nd.words = countWords(nd.text)
			if img, ok := v.FirstChild().(*ast.Image); ok && v.ChildCount() == 1 {
				nd.kind, nd.alt = "image", plainOf(img)
			}
		case *ast.List:
			nd.kind, nd.ordered, nd.items = "list", v.IsOrdered(), v.ChildCount()
			nd.words = countWords(plainOf(v))
			if li := v.FirstChild(); li != nil {
				if tb := li.FirstChild(); tb != nil {
					_, nd.task = tb.FirstChild().(*east.TaskCheckBox)
				}
			}
		case *ast.FencedCodeBlock:
			nd.kind, nd.fenced = "code", true
			nd.lang = string(v.Language(b))
			switch {
			case v.Info != nil:
				nd.line = lineAt(v.Info.Segment.Start)
			case v.Lines().Len() > 0:
				nd.line = lineAt(v.Lines().At(0).Start - 1)
			default:
				nd.line = prevLine + 1
			}
			if nd.lang != "" && v.Info != nil {
				if rest := strings.TrimSpace(strings.TrimPrefix(string(v.Info.Segment.Value(b)), nd.lang)); strings.HasPrefix(rest, "{") {
					nd.attrs, _ = plugins.ParseAttrs(rest)
				}
			}
		case *ast.CodeBlock:
			nd.kind = "code"
		case *ast.HTMLBlock:
			nd.kind = "html"
			if v.Lines().Len() > 0 {
				seg := v.Lines().At(0)
				if m := placeholderRE.FindSubmatch(seg.Value(b)); m != nil {
					var i int
					fmt.Sscan(string(m[1]), &i)
					if i < len(spans) {
						nd.kind, nd.name, nd.attrs = "block", spans[i].Name, spans[i].Attrs
					}
				}
			}
		case *east.Table:
			nd.kind = "table"
			for c := v.FirstChild(); c != nil; c = c.NextSibling() {
				if _, ok := c.(*east.TableHeader); ok {
					for cell := c.FirstChild(); cell != nil; cell = cell.NextSibling() {
						nd.cols = append(nd.cols, plainOf(cell))
					}
				} else {
					nd.rows++
				}
			}
			nd.words = countWords(plainOf(v))
		case *ast.ThematicBreak:
			nd.kind = "break"
		case *ast.Blockquote:
			nd.kind = "quote"
			nd.words = countWords(plainOf(v))
		}
		if nd.line == 0 {
			nd.line = prevLine + 1
		}
		prevLine = nd.line
		d.words += nd.words
		d.nodes = append(d.nodes, nd)
	}

	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		img, ok := n.(*ast.Image)
		if !ok || !entering {
			return ast.WalkContinue, nil
		}
		line := 0
		if t, ok := img.PreviousSibling().(*ast.Text); ok {
			line = lineAt(t.Segment.Stop - 1)
		} else {
			for p := n.Parent(); p != nil && line == 0; p = p.Parent() {
				if off, ok := first(p); ok {
					line = lineAt(off)
				}
			}
		}
		d.images = append(d.images, image{line: line, alt: plainOf(img)})
		return ast.WalkSkipChildren, nil
	})
	return d
}
