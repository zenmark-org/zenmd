package structure

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"
)

type checker struct {
	d    *Doc
	t    *Template
	over *Template // the template, when t is the layout: what it declares wins
	out  []Problem
	last bool // inside the last of a repeated section
}

// Check runs the layout's structure and the page template's against the page.
// Both are checked; where they disagree (a field or a rule both name) the
// template wins. Either may be nil. Problems come in page order.
func Check(d *Doc, t *Template, layout *Template) []Problem {
	var out []Problem
	if layout != nil {
		c := &checker{d: d, t: layout, over: t}
		c.run()
		out = append(out, c.out...)
	}
	if t != nil {
		c := &checker{d: d, t: t}
		c.run()
		out = append(out, c.out...)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Line < out[j].Line })
	return out
}

func (c *checker) source() string { return c.t.Kind + " " + c.t.ID }

func (c *checker) add(rule, sev string, line int, anchor *node, msg, fix string) {
	p := Problem{Rule: rule, Severity: sev, Message: msg, Fix: fix, Line: line, Source: c.source()}
	if anchor != nil {
		p.Anchor = anchor.id
		if p.Line <= 0 {
			p.Line = anchor.line
		}
	}
	if p.Line <= 0 {
		p.Line = 1
	}
	c.out = append(c.out, p)
}

func (c *checker) run() {
	c.fields()
	c.seq(c.t.Structure, c.d.nodes, nil)
	c.pageWide(c.t.Structure)
	c.slides()
	c.rules()
}

// ---- fields

var dateRE = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}`)

func (c *checker) fields() {
	names := make([]string, 0, len(c.t.Fields))
	for n := range c.t.Fields {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, name := range names {
		if c.over != nil {
			if _, ok := c.over.Fields[name]; ok {
				continue
			}
		}
		f := c.t.Fields[name]
		sev := f.Severity
		if sev == "" {
			sev = SevWarning
		}
		v, ok := lookup(c.d.FM, name)
		if s, isStr := v.(string); ok && isStr && strings.TrimSpace(s) == "" {
			ok = false
		}
		if v == nil {
			ok = false
		}
		line := c.d.fmLines[strings.SplitN(name, ".", 2)[0]]
		if !ok {
			if name == "title" && c.d.hasH1 {
				continue
			}
			if f.Required {
				c.add("fields", sev, 1, nil, fmt.Sprintf("%s is required by the %s %s", name, c.t.ID, c.t.Kind),
					fmt.Sprintf("Add %s: … to the frontmatter at the top of the page", name))
			}
			continue
		}
		bad := func(what string) {
			c.add("fields", sev, line, nil, fmt.Sprintf("%s must be %s", name, what), fmt.Sprintf("Change %s: in the frontmatter", name))
		}
		switch f.Type {
		case "date":
			switch x := v.(type) {
			case time.Time:
			case string:
				if _, err := time.Parse("2006-01-02", strings.TrimSpace(x)); err != nil && !dateRE.MatchString(x) {
					bad("a date such as 2026-11-17")
				}
			default:
				bad("a date such as 2026-11-17")
			}
		case "list":
			if _, ok := v.([]any); !ok {
				bad("a list, such as [a, b]")
			}
		case "bool":
			if _, ok := v.(bool); !ok {
				bad("true or false")
			}
		case "number", "int":
			switch x := v.(type) {
			case int, int64, uint64:
			case float64:
				if f.Type == "int" && x != float64(int64(x)) {
					bad("a whole number")
				}
			default:
				bad("a number")
			}
		case "string", "text", "image":
			switch v.(type) {
			case []any, map[string]any:
				bad("text")
			}
		}
		if len(f.Enum) > 0 {
			s := fmt.Sprint(v)
			found := false
			for _, e := range f.Enum {
				found = found || e == s
			}
			if !found {
				bad("one of " + strings.Join(f.Enum, ", "))
			}
		}
		if f.re != nil {
			if s, isStr := v.(string); isStr && !f.re.MatchString(s) {
				c.add("fields", sev, line, nil, fmt.Sprintf("%s does not match the pattern %s", name, f.Pattern), fmt.Sprintf("Change %s: to match %s", name, f.Pattern))
			}
		}
	}
}

// lookup reads a field; a name with dots (layoutOptions.id) goes into mappings.
func lookup(m map[string]any, name string) (any, bool) {
	var cur any = m
	for _, part := range strings.Split(name, ".") {
		mm, ok := cur.(map[string]any)
		if !ok {
			return nil, false
		}
		if cur, ok = mm[part]; !ok {
			return nil, false
		}
	}
	return cur, true
}

// ---- the body

func headingMatch(n *node, e *Element) bool {
	if n.kind != "heading" || (e.Level != 0 && n.level != e.Level) {
		return false
	}
	switch {
	case e.re != nil:
		return e.re.MatchString(n.text)
	case e.Text != "":
		return strings.EqualFold(strings.TrimSpace(n.text), strings.TrimSpace(e.Text))
	}
	return true
}

func extentEnd(nodes []node, idx, level int) int {
	j := idx + 1
	for j < len(nodes) && !(nodes[j].kind == "heading" && nodes[j].level <= level) {
		j++
	}
	return j
}

func findSection(nodes []node, from int, e *Element) int {
	for i := from; i < len(nodes); i++ {
		if headingMatch(&nodes[i], e) {
			return i
		}
	}
	return -1
}

func nextHeading(nodes []node, from int) int {
	for i := from; i < len(nodes); i++ {
		if nodes[i].kind == "heading" {
			return i
		}
	}
	return len(nodes)
}

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}

func (c *checker) titleField() bool {
	s, ok := c.d.FM["title"].(string)
	return ok && strings.TrimSpace(s) != ""
}

// seq checks elements, in order, against nodes. parent is the heading the
// nodes sit under (nil at the top of the page); a gap is marked at the last
// heading seen before it.
func (c *checker) seq(elems []Element, nodes []node, parent *node) {
	cur, anchor := 0, parent
	for k := range elems {
		e := &elems[k]
		switch e.Kind {
		case "heading":
			var idxs []int
			for i := cur; i < len(nodes); i++ {
				if headingMatch(&nodes[i], e) {
					idxs = append(idxs, i)
				}
			}
			n := len(idxs)
			if e.Level == 1 && n == 0 && c.titleField() && e.Count <= 1 {
				continue
			}
			label := "heading"
			if e.Text != "" {
				label = fmt.Sprintf("heading %q", e.Text)
			} else if e.Level > 0 {
				label = fmt.Sprintf("H%d heading", e.Level)
			}
			hashes := strings.Repeat("#", max(e.Level, 1))
			fix := fmt.Sprintf("Add a '%s %s' line", hashes, orElse(plainText(e.Text), "Heading"))
			if e.Level == 1 {
				fix = "Add a '# Title' line at the top, or title: in the frontmatter"
			}
			switch {
			case e.Count >= 0 && n > e.Count:
				c.add("heading", e.Severity, nodes[idxs[e.Count]].line, &nodes[idxs[e.Count]],
					fmt.Sprintf("Page has %d H%d headings; expected %d", n, max(e.Level, 1), e.Count), "Keep one, and make the others lower-level headings")
			case e.Count >= 0 && n < e.Count, e.Count < 0 && n == 0 && !e.Optional:
				c.add("heading", e.Severity, 0, anchor, "Missing "+label, fix)
			}
			if n > 0 {
				cur, anchor = idxs[0]+1, &nodes[idxs[0]]
			}
		case "section":
			c.section(e, nodes, &cur, &anchor)
		case "paragraph":
			end := nextHeading(nodes, cur)
			var paras []int
			for i := cur; i < end; i++ {
				if nodes[i].kind == "paragraph" {
					paras = append(paras, i)
				}
			}
			min := e.Min
			if min < 0 {
				min = 1
			}
			label := orElse(e.Label, "paragraph")
			if len(paras) < min {
				c.add("paragraph", e.Severity, 0, anchor, "Missing the "+label+" paragraph", "Add a short paragraph here, before the next heading")
				continue
			}
			if e.Max >= 0 && len(paras) > e.Max {
				c.add("paragraph", e.Severity, nodes[paras[e.Max]].line, &nodes[paras[e.Max]],
					fmt.Sprintf("Too many paragraphs here (%d); at most %d", len(paras), e.Max), "Move the rest under a heading")
			}
			take := min
			if e.Max >= 0 {
				take = min2(len(paras), e.Max)
			}
			for _, i := range paras[:min2(take, len(paras))] {
				if e.MaxWords > 0 && nodes[i].words > e.MaxWords {
					c.add("paragraph", e.Severity, nodes[i].line, anchor,
						fmt.Sprintf("%s has %d words; at most %d", capitalise(label), nodes[i].words, e.MaxWords), "Shorten it, or move detail further down")
				}
			}
			if take > 0 && take <= len(paras) {
				cur = paras[take-1] + 1
			}
		case "list":
			var found []int
			for i := cur; i < len(nodes); i++ {
				if nodes[i].kind == "list" && listMatch(&nodes[i], e.ListKind) {
					found = append(found, i)
				}
			}
			min := orInt(e.Min, 1)
			noun := map[string]string{"ordered": "ordered list", "unordered": "bulleted list", "task": "task list", "": "list"}[e.ListKind]
			if len(found) < min {
				fix := map[string]string{"ordered": "Add a numbered list (1. …)", "unordered": "Add a bulleted list (- …)", "task": "Add a task list (- [ ] …)", "": "Add a list"}[e.ListKind]
				c.add("list", e.Severity, 0, anchor, "Missing "+noun, fix)
				continue
			}
			if len(found) == 0 {
				continue
			}
			n := &nodes[found[0]]
			if e.MinItems >= 0 && n.items < e.MinItems {
				c.add("list", e.Severity, n.line, anchor, fmt.Sprintf("The %s has %d %s; at least %d", noun, n.items, plural(n.items, "item", "items"), e.MinItems), "Add the missing items")
			}
			if e.MaxItems >= 0 && n.items > e.MaxItems {
				c.add("list", e.Severity, n.line, anchor, fmt.Sprintf("The %s has %d items; at most %d", noun, n.items, e.MaxItems), "Shorten the list, or split it")
			}
			cur = found[0] + 1
		case "code":
			var found []int
			for i := cur; i < len(nodes); i++ {
				if nodes[i].kind == "code" {
					found = append(found, i)
				}
			}
			if len(found) < orInt(e.Min, 1) {
				c.add("code", e.Severity, 0, anchor, "Missing code block", "Add a fenced code block (```lang)")
				continue
			}
			if e.Max >= 0 && len(found) > e.Max {
				c.add("code", e.Severity, nodes[found[e.Max]].line, anchor, fmt.Sprintf("Too many code blocks (%d); at most %d", len(found), e.Max), "Remove or merge code blocks")
			}
			for _, i := range found {
				switch {
				case e.Lang == "required" && nodes[i].lang == "":
					c.add("code", e.Severity, nodes[i].line, anchor, "Code block has no language", "Write the language after the opening fence, such as ```go")
				case e.Lang != "" && e.Lang != "required" && nodes[i].lang != e.Lang:
					c.add("code", e.Severity, nodes[i].line, anchor, fmt.Sprintf("Code block should be %s", e.Lang), "Use ```"+e.Lang)
				}
			}
			if len(found) > 0 {
				cur = found[len(found)-1] + 1
			}
		case "table":
			idx := -1
			for i := cur; i < len(nodes) && idx < 0; i++ {
				if nodes[i].kind == "table" {
					idx = i
				}
			}
			if idx < 0 {
				if orInt(e.Min, 1) > 0 {
					c.add("table", e.Severity, 0, anchor, "Missing table", "Add a table")
				}
				continue
			}
			n := &nodes[idx]
			if e.MinRows >= 0 && n.rows < e.MinRows {
				c.add("table", e.Severity, n.line, anchor, fmt.Sprintf("The table has %d rows; at least %d", n.rows, e.MinRows), "Add the missing rows")
			}
			for _, want := range e.Columns {
				found := false
				for _, col := range n.cols {
					found = found || strings.EqualFold(strings.TrimSpace(col), strings.TrimSpace(want))
				}
				if !found {
					c.add("table", e.Severity, n.line, anchor, fmt.Sprintf("The table has no %q column", want), fmt.Sprintf("Add a '%s' column to the table header", want))
				}
			}
			cur = idx + 1
		case "image":
			idx := -1
			for i := cur; i < len(nodes) && idx < 0; i++ {
				if nodes[i].kind == "image" {
					idx = i
				}
			}
			if idx < 0 {
				if orInt(e.Min, 1) > 0 {
					c.add("image", e.Severity, 0, anchor, "Missing image", "Add an image: ![what it shows](file.png)")
				}
				continue
			}
			if e.Alt == "required" && strings.TrimSpace(nodes[idx].alt) == "" {
				c.add("image", e.Severity, nodes[idx].line, anchor, "Image has no alt text", "Describe the image between the brackets: ![what it shows](file)")
			}
			cur = idx + 1
		case "block":
			if e.ExceptLast && c.last {
				continue
			}
			if e.Unless != "" {
				skip := false
				for i := range nodes {
					skip = skip || blockMatch(&nodes[i], e.Unless)
				}
				if skip {
					continue
				}
			}
			var found []int
			for i := cur; i < len(nodes); i++ {
				if blockMatch(&nodes[i], e.Name) {
					found = append(found, i)
				}
			}
			if len(found) < orInt(e.Min, 1) {
				c.add("block", e.Severity, 0, anchor, fmt.Sprintf("Missing %s block", e.Name), fmt.Sprintf("Add a :::%s block", e.Name))
				continue
			}
			if e.Max >= 0 && len(found) > e.Max {
				c.add("block", e.Severity, nodes[found[e.Max]].line, anchor, fmt.Sprintf("Too many %s blocks (%d); at most %d", e.Name, len(found), e.Max), "Remove the extra blocks")
			}
			if len(found) > 0 {
				cur = found[0] + 1
			}
		case "any":
			end := nextHeading(nodes, cur)
			n := end - cur
			if e.Min >= 0 && n < e.Min {
				c.add("any", e.Severity, 0, anchor, fmt.Sprintf("Expected at least %d blocks of text here, found %d", e.Min, n), "Add the text")
			}
			if e.Max >= 0 && n > e.Max {
				c.add("any", e.Severity, nodes[cur+e.Max].line, anchor, fmt.Sprintf("Too much content here (%d blocks); at most %d", n, e.Max), "Shorten it, or move the rest under a heading")
				n = e.Max
			}
			cur += n
		case "end":
			for i := cur; i < len(nodes); i++ {
				if nodes[i].kind == "html" || nodes[i].kind == "other" {
					continue
				}
				c.add("end", e.Severity, nodes[i].line, anchor, fmt.Sprintf("Unexpected %s after the last expected element", nodes[i].kind), "Remove it, or move it above")
				break
			}
		}
	}
}

func (c *checker) section(e *Element, nodes []node, cur *int, anchor **node) {
	repeat := e.Repeat || e.Min >= 0 || e.Max >= 0
	min := e.Min
	if min < 0 {
		min = 1
		if e.Optional {
			min = 0
		}
	}
	start, pos := *cur, *cur
	var found []int
	for {
		idx := findSection(nodes, pos, e)
		if idx < 0 {
			break
		}
		found = append(found, idx)
		pos = extentEnd(nodes, idx, e.Level)
		if !repeat {
			break
		}
	}
	prevLast := c.last
	if len(e.Contains) > 0 {
		for i, idx := range found {
			c.last = i == len(found)-1
			c.seq(e.Contains, nodes[idx+1:extentEnd(nodes, idx, e.Level)], &nodes[idx])
		}
	}
	c.last = prevLast
	if len(found) > 0 {
		*cur, *anchor = pos, &nodes[found[len(found)-1]]
	}
	label := "section"
	if e.Text != "" {
		label = fmt.Sprintf("section '%s'", plainText(e.Text))
	}
	switch {
	case len(found) < min && len(found) == 0 && findSection(nodes[:start], 0, e) >= 0:
		i := findSection(nodes[:start], 0, e)
		c.add("section", e.Severity, nodes[i].line, &nodes[i], fmt.Sprintf("The %s is out of order", label), "Move it to where the template puts it")
	case len(found) < min:
		hashes := strings.Repeat("#", e.Level)
		msg := "Missing " + label
		fix := fmt.Sprintf("Add a '%s %s' section", hashes, orElse(plainText(e.Text), "Heading"))
		if e.Text == "" || min > 1 {
			msg = fmt.Sprintf("Expected at least %d sections, found %d", min, len(found))
			fix = fmt.Sprintf("Add %d more '%s' %s", min-len(found), hashes, plural(min-len(found), "section", "sections"))
		}
		if *anchor != nil {
			fix += " after '" + (*anchor).text + "'"
		}
		c.add("section", e.Severity, 0, *anchor, msg, fix)
	}
	if e.Max >= 0 && len(found) > e.Max {
		n := &nodes[found[e.Max]]
		c.add("section", e.Severity, n.line, n, fmt.Sprintf("Page has %d sections here; at most %d", len(found), e.Max), "Merge or remove the extra sections")
	}
}

func plainText(s string) string {
	if len(s) > 1 && strings.HasPrefix(s, "/") && strings.HasSuffix(s, "/") {
		return ""
	}
	return s
}

func listMatch(n *node, kind string) bool {
	switch kind {
	case "ordered":
		return n.ordered
	case "unordered":
		return !n.ordered
	case "task":
		return n.task
	}
	return true
}

func blockMatch(n *node, name string) bool {
	return (n.kind == "block" && n.name == name) || (n.kind == "code" && n.fenced && n.lang == name)
}

// pageWide checks `ids: unique` over the whole page, whatever section the blocks sit in.
func (c *checker) pageWide(els []Element) {
	for i := range els {
		e := &els[i]
		if e.Kind == "block" && e.IDs == "unique" {
			seen := map[string]bool{}
			for j := range c.d.nodes {
				n := &c.d.nodes[j]
				if !blockMatch(n, e.Name) || n.attrs["id"] == "" {
					continue
				}
				if seen[n.attrs["id"]] {
					c.add("block", e.Severity, n.line, nil, fmt.Sprintf("Duplicate %s id '%s'", e.Name, n.attrs["id"]), fmt.Sprintf("Give each %s block its own id", e.Name))
				}
				seen[n.attrs["id"]] = true
			}
		}
		c.pageWide(e.Contains)
	}
}

// slides checks the slide elements of the top level: a page is split at its
// `---` lines, or at each heading of level 1 or 2 when it has none.
func (c *checker) slides() {
	for i := range c.t.Structure {
		e := &c.t.Structure[i]
		if e.Kind != "slide" {
			continue
		}
		var segs [][]node
		hasBreak := false
		for _, n := range c.d.nodes {
			hasBreak = hasBreak || n.kind == "break"
		}
		var cur []node
		flush := func() {
			if len(cur) > 0 {
				segs = append(segs, cur)
			}
			cur = nil
		}
		for _, n := range c.d.nodes {
			switch {
			case hasBreak && n.kind == "break":
				flush()
			case !hasBreak && n.kind == "heading" && n.level <= 2:
				flush()
				cur = append(cur, n)
			default:
				cur = append(cur, n)
			}
		}
		flush()
		if e.Min >= 0 && len(segs) < e.Min {
			c.add("slide", e.Severity, 1, nil, fmt.Sprintf("Expected at least %d slides, found %d", e.Min, len(segs)), "Add a slide: a line with --- starts the next one")
		}
		if e.Max >= 0 && len(segs) > e.Max {
			c.add("slide", e.Severity, segs[e.Max][0].line, nil, fmt.Sprintf("Expected at most %d slides, found %d", e.Max, len(segs)), "Cut or merge slides")
		}
		for k, seg := range segs {
			hasHeading, bullets, bulletLine := false, 0, 0
			for _, n := range seg {
				hasHeading = hasHeading || n.kind == "heading"
				if n.kind == "list" {
					if bullets == 0 {
						bulletLine = n.line
					}
					bullets += n.items
				}
			}
			if e.HeadingRequired && !hasHeading {
				c.add("slide", e.Severity, seg[0].line, nil, fmt.Sprintf("Slide %d has no heading", k+1), "Start the slide with a heading")
			}
			if e.MaxBullets >= 0 && bullets > e.MaxBullets {
				c.add("slide", e.Severity, bulletLine, nil, fmt.Sprintf("Slide %d has %d bullets; at most %d", k+1, bullets, e.MaxBullets), "Cut bullets, or split the slide")
			}
		}
	}
}

// ---- rules

func (c *checker) rules() {
	r := c.t.Rules
	skip := func(mine, theirs Rule) bool { return mine.Severity == "" || (c.over != nil && theirs.Severity != "") }
	var o Rules
	if c.over != nil {
		o = c.over.Rules
	}
	if !skip(r.HeadingOrder, o.HeadingOrder) {
		prev := 0
		for i := range c.d.nodes {
			n := &c.d.nodes[i]
			if n.kind != "heading" {
				continue
			}
			if prev > 0 && n.level > prev+1 {
				c.add("heading-order", r.HeadingOrder.Severity, n.line, n, fmt.Sprintf("Heading levels skip from H%d to H%d", prev, n.level), fmt.Sprintf("Use H%d here, or add the missing level", prev+1))
			}
			prev = n.level
		}
	}
	if !skip(r.ImageAlt, o.ImageAlt) {
		for _, im := range c.d.images {
			if strings.TrimSpace(im.alt) == "" {
				c.add("image-alt", r.ImageAlt.Severity, im.line, nil, "Image has no alt text", "Describe the image between the brackets: ![what it shows](file)")
			}
		}
	}
	if !skip(r.MaxWords, o.MaxWords) && r.MaxWords.Words > 0 && c.d.words > r.MaxWords.Words {
		c.add("max-words", r.MaxWords.Severity, 1, nil, fmt.Sprintf("Page has %d words; at most %d", c.d.words, r.MaxWords.Words), "Shorten the page, or split it into two")
	}
	if !skip(r.CodeLang, o.CodeLang) {
		for _, n := range c.d.nodes {
			if n.kind == "code" && n.fenced && n.lang == "" {
				c.add("code-lang", r.CodeLang.Severity, n.line, nil, "Code block has no language", "Write the language after the opening fence, such as ```go")
			}
		}
	}
	if !skip(r.NeedsSections, o.NeedsSections) && r.NeedsSections.Words > 0 && c.d.words > r.NeedsSections.Words {
		has := false
		for _, n := range c.d.nodes {
			has = has || (n.kind == "heading" && n.level == r.NeedsSections.Level)
		}
		if !has {
			c.add("needs-sections", r.NeedsSections.Severity, 1, nil, fmt.Sprintf("Page has %d words and no H%d sections", c.d.words, r.NeedsSections.Level), "Break the text up with headings")
		}
	}
}

func orElse(a, b string) string {
	if a == "" {
		return b
	}
	return a
}

func orInt(v, def int) int {
	if v < 0 {
		return def
	}
	return v
}

func min2(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func capitalise(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
