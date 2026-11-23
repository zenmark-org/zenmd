// Package structure checks a page against a structure: the frontmatter fields,
// the sections and blocks a page template (or a layout) asks for, and a few
// writing rules. Every gap is a Problem with a severity, a message, a Fix line
// and where to mark it. zenmd check and the Inspector's Problems › Structure
// share this code.
//
// render.Page holds only the HTML, so Check takes the page source, read once
// by Analyse (frontmatter by engine/config, `:::` containers by engine/plugins,
// the rest by goldmark with the extensions the renderer uses).
package structure

import "regexp"

// The severities. The default is warning.
const (
	SevError      = "error"
	SevWarning    = "warning"
	SevSuggestion = "suggestion"
)

// MaxPattern is the longest regular expression a template may use (Go RE2, so linear time).
const MaxPattern = 200

// Field is a frontmatter field a template or layout asks for.
type Field struct {
	Type     string // string, text, int, number, bool, date, image or list; "" for any
	Required bool
	Enum     []string
	Pattern  string
	Severity string
	re       *regexp.Regexp
}

// Element is one thing a page's body should have, in order.
type Element struct {
	Kind     string // heading, section, paragraph, list, code, table, image, block, slide, any, end
	Severity string

	Level    int    // heading, section (0: heading any level, section 2)
	Text     string // heading text, or the section's heading; "" for any
	re       *regexp.Regexp
	Count    int  // heading: exactly this many (-1: unset)
	Optional bool // heading, section
	Repeat   bool // section
	Min, Max int  // -1: unset
	Contains []Element

	MaxWords int    // paragraph (0: unset)
	Label    string // paragraph
	ListKind string // list: ordered, unordered, task or ""
	MinItems int    // list, -1 unset
	MaxItems int    // list, -1 unset
	Lang     string // code: "required", a language, or ""
	MinRows  int    // table, -1 unset
	Columns  []string
	Alt      string // image: "required" or ""
	Name     string // block
	IDs      string // block: "unique" or ""
	// ExceptLast: a block asked of every repeated section but the last.
	ExceptLast bool
	// Slide: HeadingRequired and MaxBullets.
	HeadingRequired bool
	MaxBullets      int // -1 unset
}

// Rule is one writing rule with its severity ("" for off).
type Rule struct {
	Severity string
	Words    int // max-words; needs-sections: over this many words
	Level    int // needs-sections: the heading level that is missing
}

// Rules are the checks that need no structure.
type Rules struct {
	MaxWords, HeadingOrder, ImageAlt, CodeLang, NeedsSections Rule
}

// Template is a page template, or a layout's structure.
type Template struct {
	ID, Name, Description, Layout string
	Kind                          string // "template" or "layout": where its problems say they come from
	Fields                        map[string]Field
	Structure                     []Element
	Rules                         Rules
	Skeleton                      string
}

// Problem is one thing a page lacks or does wrong.
type Problem struct {
	Rule     string // the element or rule: fields, heading, section, paragraph, list, code, table, image, block, slide, any, end, heading-order, image-alt, max-words, code-lang, needs-sections
	Message  string
	Fix      string
	Severity string
	Line     int    // 1-based, in the page file; never 0
	Anchor   string // the id of the heading to mark (the one before a gap); "" for the top of the page
	Source   string // "template how-to" or "layout post"
}
