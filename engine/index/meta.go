package index

import (
	"fmt"
	"math"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Meta is the navigation metadata of a page, from its frontmatter nav key.
// Parent is a workspace path; the zero value changes nothing.
type Meta struct {
	Title  string
	Order  *int
	Parent string
	Hidden bool
	// Sort is nav.sort (title or newest-first): how the children of this page
	// are ordered. When set it decides over the children's nav.order.
	Sort string
	// Children is nav.children: "folder" makes every page of the folder that
	// no link places a child of this page (an index.md or README.md).
	Children string
	// Date and Version are the frontmatter date and version, for newest-first.
	Date, Version string
}

// Parent markers for pages without a parent page.
const (
	atRoot  = -1 // top level of the tree
	inOther = -2 // "Other pages" group
)

// BuildWithMeta is Build with per-page navigation metadata; meta may be nil.
// Pages are first placed by links exactly as Build does, then nav.parent
// overrides (in path order) move pages, nav.order sorts siblings and hidden
// pages are left out of Tree but keep their breadcrumbs.
func BuildWithMeta(files []string, home string, links func(path string) []string, titles func(path string) string, meta func(path string) Meta) *Index {
	canon := make(map[string]string, len(files))
	for _, f := range files {
		k := strings.ToLower(f)
		if _, ok := canon[k]; !ok {
			canon[k] = f
		}
	}

	// 1 link placement: breadth-first from the home page (or root pages).
	pos := make(map[string]int, len(files)) // lower-cased path -> id
	var seq []string
	var par []int
	add := func(p string, parent int) {
		pos[strings.ToLower(p)] = len(seq)
		seq = append(seq, p)
		par = append(par, parent)
	}
	placed := func(p string) bool { _, ok := pos[strings.ToLower(p)]; return ok }

	var seeds []string
	if h, ok := canon[strings.ToLower(home)]; ok && home != "" {
		seeds = []string{h}
	} else {
		seen := map[string]bool{}
		for _, f := range files {
			if k := strings.ToLower(f); !strings.Contains(f, "/") && !seen[k] {
				seen[k] = true
				seeds = append(seeds, canon[k])
			}
		}
		sort.Strings(seeds)
	}
	for _, s := range seeds {
		add(s, atRoot)
	}
	for q := 0; q < len(seq); q++ {
		for _, l := range links(seq[q]) {
			if c, ok := canon[strings.ToLower(l)]; ok && !placed(c) {
				add(c, q)
			}
		}
	}
	var rest []string
	for _, f := range files {
		if canon[strings.ToLower(f)] == f && !placed(f) {
			rest = append(rest, f)
		}
	}
	sort.Strings(rest)
	// nav.children: folder on an index.md or README.md adopts the pages of
	// its folder that no link placed.
	folderIndex := map[string]string{} // lower-cased folder -> its index page
	if meta != nil {
		for _, f := range files {
			base := strings.ToLower(path.Base(f))
			if canon[strings.ToLower(f)] != f || (base != "index.md" && base != "readme.md") || meta(f).Children != "folder" {
				continue
			}
			dir := strings.ToLower(path.Dir(f))
			if cur, ok := folderIndex[dir]; !ok || strings.ToLower(path.Base(cur)) != "index.md" && base == "index.md" {
				folderIndex[dir] = f
			}
		}
	}
	for _, f := range rest {
		if placed(f) {
			continue
		}
		if ixp, ok := folderIndex[strings.ToLower(path.Dir(f))]; ok && ixp != f {
			// the index is placed before its children (it may itself be in Other pages)
			if !placed(ixp) {
				add(ixp, inOther)
			}
			add(f, pos[strings.ToLower(ixp)])
			continue
		}
		add(f, inOther)
	}
	n := len(seq)

	ix := &Index{crumbs: make(map[string][]Crumb, n), warnings: map[string][]string{}}
	metaOf := make([]Meta, n)
	if meta != nil {
		for i, p := range seq {
			metaOf[i] = meta(p)
		}
	}
	warn := func(i int, format string, args ...any) {
		k := strings.ToLower(seq[i])
		ix.warnings[k] = append(ix.warnings[k], "frontmatter: "+fmt.Sprintf(format, args...))
	}

	// 2 nav.parent overrides, in path order so the result is deterministic.
	var moved []int
	for i := range seq {
		if metaOf[i].Parent != "" {
			moved = append(moved, i)
		}
	}
	sort.Slice(moved, func(a, b int) bool { return seq[moved[a]] < seq[moved[b]] })
	for _, i := range moved {
		want := metaOf[i].Parent
		j, ok := pos[strings.ToLower(strings.TrimLeft(want, "/"))]
		switch {
		case !ok:
			warn(i, "nav.parent %q: no such page", want)
		case j == i:
			warn(i, "nav.parent %q: a page cannot be its own parent", want)
		case ancestor(par, i, j):
			warn(i, "nav.parent %q: would create a cycle", want)
		default:
			par[i] = j
		}
	}

	// 3 children in placement order, then by nav.order.
	children := make([][]int, n)
	var roots, others []int
	for i := range seq {
		switch par[i] {
		case atRoot:
			roots = append(roots, i)
		case inOther:
			others = append(others, i)
		default:
			children[par[i]] = append(children[par[i]], i)
		}
	}
	// A page moved here by nav.parent takes the place of this parent's link to it, not the place where another
	// page's link first reached it. Only parents that gained a moved child are reordered; pages the parent does not
	// link to keep their order after the linked ones.
	for _, i := range moved {
		p := par[i]
		if p < 0 || len(children[p]) < 2 {
			continue
		}
		rank := map[string]int{}
		for k, l := range links(seq[p]) {
			if c, ok := canon[strings.ToLower(l)]; ok {
				if _, seen := rank[strings.ToLower(c)]; !seen {
					rank[strings.ToLower(c)] = k
				}
			}
		}
		at := func(id int) int {
			if r, ok := rank[strings.ToLower(seq[id])]; ok {
				return r
			}
			return len(rank)
		}
		sort.SliceStable(children[p], func(a, b int) bool { return at(children[p][a]) < at(children[p][b]) })
	}
	byOrder := func(ids []int) {
		sort.SliceStable(ids, func(a, b int) bool { return less(metaOf[ids[a]].Order, metaOf[ids[b]].Order) })
	}
	byOrder(roots)
	for _, c := range children {
		byOrder(c)
	}
	titleOf := make([]string, n)
	for i, p := range seq {
		switch {
		case metaOf[i].Title != "":
			titleOf[i] = metaOf[i].Title
		case par[i] == inOther:
			name := path.Base(p)
			titleOf[i] = strings.TrimSuffix(name, path.Ext(name))
		default:
			titleOf[i] = titles(p)
		}
	}
	for i, c := range children {
		switch metaOf[i].Sort {
		case "title":
			sort.SliceStable(c, func(a, b int) bool {
				ka, kb := Fold(titleOf[c[a]]), Fold(titleOf[c[b]])
				if ka != kb {
					return ka < kb
				}
				return seq[c[a]] < seq[c[b]]
			})
		case "newest-first":
			sort.SliceStable(c, func(a, b int) bool { return newer(metaOf[c[a]], seq[c[a]], metaOf[c[b]], seq[c[b]]) })
		}
	}

	// 4 parents before children: titles, hidden and breadcrumbs.
	topo := make([]int, 0, n)
	topo = append(append(topo, roots...), others...)
	for q := 0; q < len(topo); q++ {
		topo = append(topo, children[topo[q]]...)
	}
	hidden := make([]bool, n)
	for _, i := range topo {
		p := seq[i]
		hidden[i] = metaOf[i].Hidden || par[i] >= 0 && hidden[par[i]]
		var bc []Crumb
		if par[i] >= 0 {
			pc := ix.crumbs[strings.ToLower(seq[par[i]])]
			bc = make([]Crumb, len(pc), len(pc)+1)
			copy(bc, pc)
		}
		ix.crumbs[strings.ToLower(p)] = append(bc, Crumb{titleOf[i], p})
	}

	// 5 nodes, children first; hidden pages are left out.
	built := make([]Node, n)
	for q := len(topo) - 1; q >= 0; q-- {
		i := topo[q]
		nd := Node{Title: titleOf[i], Path: seq[i], Kind: "page"}
		for _, c := range children[i] {
			if !hidden[c] {
				nd.Children = append(nd.Children, built[c])
			}
		}
		built[i] = nd
	}
	for _, r := range roots {
		if !hidden[r] {
			ix.tree = append(ix.tree, built[r])
		}
	}
	other := Node{Title: "Other pages", Kind: "group"}
	ordered := false
	for _, i := range others {
		if !hidden[i] {
			insert(&other, strings.Split(seq[i], "/"), 0, built[i])
			ordered = ordered || metaOf[i].Order != nil
		}
	}
	if ordered {
		orderGroups(&other, func(p string) *int { return metaOf[pos[strings.ToLower(p)]].Order })
	}
	if len(other.Children) > 0 {
		ix.tree = append(ix.tree, other)
	}
	return ix
}

// newer orders pages newest first: by date, then version (semver), then file name, all descending.
func newer(a Meta, ap string, b Meta, bp string) bool {
	if a.Date != b.Date {
		return a.Date > b.Date
	}
	if c := cmpVersion(a.Version, b.Version); c != 0 {
		return c > 0
	}
	return ap > bp
}

// cmpVersion compares dotted versions numerically; a pre-release (1.0.0-rc1)
// is below its release and an empty version below any.
func cmpVersion(a, b string) int {
	if a == b {
		return 0
	}
	if a == "" || b == "" {
		if a == "" {
			return -1
		}
		return 1
	}
	split := func(v string) ([]int, bool) {
		v = strings.TrimPrefix(strings.TrimSpace(v), "v")
		pre := false
		if i := strings.IndexAny(v, "-+"); i >= 0 {
			pre = v[i] == '-'
			v = v[:i]
		}
		var out []int
		for _, part := range strings.Split(v, ".") {
			n, _ := strconv.Atoi(part)
			out = append(out, n)
		}
		return out, pre
	}
	x, xp := split(a)
	y, yp := split(b)
	for i := 0; i < len(x) || i < len(y); i++ {
		var p, q int
		if i < len(x) {
			p = x[i]
		}
		if i < len(y) {
			q = y[i]
		}
		if p != q {
			if p < q {
				return -1
			}
			return 1
		}
	}
	switch {
	case xp && !yp:
		return -1
	case !xp && yp:
		return 1
	}
	return 0
}

// ancestor reports whether a is j or one of its ancestors.
func ancestor(par []int, a, j int) bool {
	for k := j; k >= 0; k = par[k] {
		if k == a {
			return true
		}
	}
	return false
}

// less sorts pages with an order first, ascending.
func less(a, b *int) bool {
	if a == nil {
		return false
	}
	return b == nil || *a < *b
}

// orderGroups sorts the pages inside the folder groups of "Other pages".
func orderGroups(n *Node, order func(path string) *int) {
	of := func(c Node) *int {
		if c.Kind != "page" {
			return nil
		}
		return order(c.Path)
	}
	sort.SliceStable(n.Children, func(a, b int) bool { return less(of(n.Children[a]), of(n.Children[b])) })
	for i := range n.Children {
		if n.Children[i].Kind == "group" {
			orderGroups(&n.Children[i], order)
		}
	}
}

// Warnings returns the navigation problems found for the page at path.
func (i *Index) Warnings(path string) []string {
	return append([]string(nil), i.warnings[strings.ToLower(path)]...)
}

var slugPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*(/[a-z0-9][a-z0-9-]*)*$`)

// maxSlug is the longest slug accepted.
const maxSlug = 100

// ParseMeta reads the slug and nav keys of a page's frontmatter. Invalid
// values are ignored and reported as warnings.
func ParseMeta(frontmatter map[string]any) (m Meta, slug string, warnings []string) {
	warn := func(format string, args ...any) {
		warnings = append(warnings, "frontmatter: "+fmt.Sprintf(format, args...))
	}
	if v, ok := frontmatter["slug"]; ok && v != nil {
		s, isStr := v.(string)
		if n, ok := intValue(v); ok {
			s, isStr = strconv.Itoa(n), true
		}
		if isStr && len(s) <= maxSlug && slugPattern.MatchString(s) {
			slug = s
		} else {
			warn("slug must be lower-case letters, digits, - and / between parts (at most %d characters)", maxSlug)
		}
	}
	m.Date, m.Version = textOf(frontmatter["date"]), textOf(frontmatter["version"])
	// What a layout implies for the sidebar; the page's nav: can override each.
	switch l, _ := frontmatter["layout"].(string); l {
	case "glossary":
		m.Children, m.Sort = "folder", "title"
	case "term":
		m.Hidden = true // the Glossary page lists the terms; the sidebar shows only it
	}
	v, ok := frontmatter["nav"]
	if !ok || v == nil {
		return m, slug, warnings
	}
	nav, ok := v.(map[string]any)
	if !ok {
		warn("nav must be a mapping")
		return m, slug, warnings
	}
	keys := make([]string, 0, len(nav))
	for k := range nav {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		v := nav[k]
		switch k {
		case "title":
			if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
				m.Title = strings.TrimSpace(s)
			} else {
				warn("nav.title must be a non-empty string")
			}
		case "order":
			if n, ok := intValue(v); ok {
				m.Order = &n
			} else {
				warn("nav.order must be an integer")
			}
		case "parent":
			if s, ok := v.(string); ok && strings.TrimLeft(strings.TrimSpace(s), "/") != "" {
				m.Parent = strings.TrimLeft(strings.TrimSpace(s), "/")
			} else {
				warn("nav.parent must be a page path or slug")
			}
		case "hidden":
			if b, ok := v.(bool); ok {
				m.Hidden = b
			} else {
				warn("nav.hidden must be true or false")
			}
		case "sort":
			if s, ok := v.(string); ok && (s == "title" || s == "newest-first") {
				m.Sort = s
			} else {
				warn("nav.sort must be title or newest-first")
			}
		case "children":
			if s, ok := v.(string); ok && s == "folder" {
				m.Children = s
			} else {
				warn("nav.children must be folder")
			}
		default:
			warn("nav.%s is not a known key (title, order, parent, hidden, sort, children)", k)
		}
	}
	return m, slug, warnings
}

// textOf renders a frontmatter scalar (a date may come as a time) as text; dates as YYYY-MM-DD.
func textOf(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(t)
	case time.Time:
		return t.Format("2006-01-02")
	case float64, int, int64:
		return fmt.Sprint(t)
	}
	return ""
}

func intValue(v any) (int, bool) {
	switch n := v.(type) {
	case int:
		return n, true
	case int64:
		if n >= math.MinInt32 && n <= math.MaxInt32 {
			return int(n), true
		}
	case uint64:
		if n <= math.MaxInt32 {
			return int(n), true
		}
	case float64:
		if n == math.Trunc(n) && n >= math.MinInt32 && n <= math.MaxInt32 {
			return int(n), true
		}
	}
	return 0, false
}
