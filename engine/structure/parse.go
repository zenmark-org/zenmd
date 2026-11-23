package structure

import (
	"fmt"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

const maxTemplate = 256 * 1024

var fieldTypes = map[string]bool{"string": true, "text": true, "int": true, "number": true, "bool": true, "date": true, "image": true, "list": true}

// element kinds, with the option each one's plain value stands for and the options it takes.
var kinds = map[string]struct {
	primary string
	opts    []string
}{
	"heading":   {"text", []string{"level", "text", "count", "optional", "severity"}},
	"section":   {"heading", []string{"heading", "level", "contains", "optional", "repeat", "min", "max", "severity"}},
	"paragraph": {"label", []string{"min", "max", "max-words", "label", "severity"}},
	"list":      {"type", []string{"type", "min-items", "max-items", "min", "severity"}},
	"code":      {"lang", []string{"lang", "min", "max", "severity"}},
	"table":     {"", []string{"min-rows", "columns", "min", "severity"}},
	"image":     {"alt", []string{"alt", "min", "severity"}},
	"block":     {"name", []string{"name", "min", "max", "ids", "except-last", "unless", "severity"}},
	"slide":     {"", []string{"min", "max", "heading", "max-bullets", "severity"}},
	"any":       {"", []string{"min", "max", "severity"}},
	"end":       {"", []string{"severity"}},
}

func validSeverity(s string) bool { return s == SevError || s == SevWarning || s == SevSuggestion }

// Parse reads a page template: its id, name, description, layout, fields,
// structure (the body, top to bottom), rules and skeleton. Keys it does not
// know are ignored; a mistake in the structure, rules or fields is an error.
func Parse(id string, data []byte) (*Template, error) {
	return parse(id, "template", data, false)
}

// ParseLayout reads the structure: value of a layout.yml: a mapping with
// fields, body (the elements, as a template's structure) and rules.
func ParseLayout(id string, data []byte) (*Template, error) {
	return parse(id, "layout", data, true)
}

func parse(id, kind string, data []byte, layout bool) (*Template, error) {
	if len(data) > maxTemplate {
		return nil, fmt.Errorf("%s %s is larger than 256 KB", kind, id)
	}
	var root yaml.Node
	if err := yaml.Unmarshal(data, &root); err != nil {
		return nil, fmt.Errorf("%s %s is not valid YAML: %s", kind, id, strings.TrimPrefix(err.Error(), "yaml: "))
	}
	t := &Template{ID: id, Kind: kind, Name: id, Fields: map[string]Field{}}
	if len(root.Content) == 0 {
		return t, nil
	}
	m := root.Content[0]
	if m.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("%s %s: expected a mapping", kind, id)
	}
	wrap := func(err error) error { return fmt.Errorf("%s %s: %w", kind, id, err) }
	for i := 0; i+1 < len(m.Content); i += 2 {
		k, v := m.Content[i].Value, m.Content[i+1]
		switch {
		case !layout && k == "id" && v.Value != "":
			// the file's id is the one the caller gives (the file name)
		case !layout && k == "name":
			t.Name = v.Value
		case !layout && k == "description":
			t.Description = v.Value
		case !layout && k == "layout":
			t.Layout = v.Value
		case !layout && k == "skeleton":
			t.Skeleton = v.Value
		case k == "fields":
			if err := parseFields(v, t.Fields); err != nil {
				return nil, wrap(err)
			}
		case (k == "structure" && !layout) || (k == "body" && layout):
			els, err := parseElements(v)
			if err != nil {
				return nil, wrap(err)
			}
			t.Structure = els
		case k == "rules":
			if err := parseRules(v, &t.Rules); err != nil {
				return nil, wrap(err)
			}
		case layout:
			return nil, wrap(fmt.Errorf("unknown key %q (fields, body and rules)", k))
		}
	}
	return t, nil
}

func parseFields(n *yaml.Node, out map[string]Field) error {
	if n.Kind != yaml.MappingNode {
		return fmt.Errorf("fields must be a mapping of names to options")
	}
	for i := 0; i+1 < len(n.Content); i += 2 {
		name, v := n.Content[i].Value, n.Content[i+1]
		var f struct {
			Type     string   `yaml:"type"`
			Required bool     `yaml:"required"`
			Enum     []string `yaml:"enum"`
			Pattern  string   `yaml:"pattern"`
			Severity string   `yaml:"severity"`
		}
		if v.Kind == yaml.MappingNode {
			if err := v.Decode(&f); err != nil {
				return fmt.Errorf("field %s: %s", name, strings.TrimPrefix(err.Error(), "yaml: "))
			}
		}
		if f.Type != "" && !fieldTypes[f.Type] {
			return fmt.Errorf("field %s: type %q is not string, text, int, number, bool, date, image or list", name, f.Type)
		}
		if f.Severity != "" && !validSeverity(f.Severity) {
			return fmt.Errorf("field %s: severity must be error, warning or suggestion", name)
		}
		fd := Field{Type: f.Type, Required: f.Required, Enum: f.Enum, Pattern: f.Pattern, Severity: f.Severity}
		if f.Pattern != "" {
			re, err := compile(f.Pattern)
			if err != nil {
				return fmt.Errorf("field %s: pattern: %w", name, err)
			}
			fd.re = re
		}
		out[name] = fd
	}
	return nil
}

func compile(p string) (*regexp.Regexp, error) {
	if len(p) > MaxPattern {
		return nil, fmt.Errorf("a pattern is at most %d characters", MaxPattern)
	}
	re, err := regexp.Compile(p)
	if err != nil {
		return nil, fmt.Errorf("%q is not a valid pattern", p)
	}
	return re, nil
}

func parseRules(n *yaml.Node, r *Rules) error {
	if n.Kind != yaml.MappingNode {
		return fmt.Errorf("rules must be a mapping")
	}
	for i := 0; i+1 < len(n.Content); i += 2 {
		name, v := n.Content[i].Value, n.Content[i+1]
		var rule Rule
		var err error
		switch name {
		case "max-words":
			rule, err = ruleValue(name, v, "words")
		case "heading-order", "code-lang":
			rule, err = ruleValue(name, v, "")
		case "image-alt":
			rule, err = ruleValue(name, v, "")
			if v.Kind == yaml.ScalarNode && v.Value == "required" {
				rule.Severity = SevError
			}
		case "needs-sections":
			rule, err = ruleValue(name, v, "words")
		default:
			return fmt.Errorf("unknown rule %q (max-words, heading-order, image-alt, code-lang, needs-sections)", name)
		}
		if err != nil {
			return err
		}
		switch name {
		case "max-words":
			r.MaxWords = rule
		case "heading-order":
			r.HeadingOrder = rule
		case "image-alt":
			r.ImageAlt = rule
		case "code-lang":
			r.CodeLang = rule
		case "needs-sections":
			r.NeedsSections = rule
			if rule.Level == 0 {
				r.NeedsSections.Level = 2
			}
		}
	}
	return nil
}

// ruleValue reads a rule: true or false, a severity, a number (the rules that
// take one), or a mapping with words, level and severity.
func ruleValue(name string, v *yaml.Node, number string) (Rule, error) {
	bad := fmt.Errorf("rule %s: expected true, false, a severity (error, warning, suggestion)%s", name, map[bool]string{true: " or a number", false: ""}[number != ""])
	switch v.Kind {
	case yaml.ScalarNode:
		switch v.Value {
		case "true", "required":
			return Rule{Severity: SevWarning}, nil
		case "false":
			return Rule{}, nil
		case SevError, SevWarning, SevSuggestion:
			return Rule{Severity: v.Value}, nil
		}
		var n int
		if number != "" && v.Decode(&n) == nil && n > 0 {
			return Rule{Severity: SevWarning, Words: n}, nil
		}
	case yaml.MappingNode:
		var m struct {
			Words    int    `yaml:"words"`
			Level    int    `yaml:"level"`
			Severity string `yaml:"severity"`
		}
		if err := v.Decode(&m); err != nil {
			return Rule{}, fmt.Errorf("rule %s: %s", name, strings.TrimPrefix(err.Error(), "yaml: "))
		}
		if m.Severity == "" {
			m.Severity = SevWarning
		}
		if !validSeverity(m.Severity) {
			return Rule{}, fmt.Errorf("rule %s: severity must be error, warning or suggestion", name)
		}
		return Rule{Severity: m.Severity, Words: m.Words, Level: m.Level}, nil
	}
	return Rule{}, bad
}

func parseElements(n *yaml.Node) ([]Element, error) {
	if n.Kind != yaml.SequenceNode {
		return nil, fmt.Errorf("structure must be a list of elements")
	}
	var out []Element
	for _, item := range n.Content {
		e, err := parseElement(item)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, nil
}

func parseElement(n *yaml.Node) (Element, error) {
	if n.Kind != yaml.MappingNode || len(n.Content) < 2 {
		return Element{}, fmt.Errorf("each structure element is a mapping such as { section: { heading: Steps } }")
	}
	kind := n.Content[0].Value
	info, ok := kinds[kind]
	if !ok {
		return Element{}, fmt.Errorf("unknown element %q (heading, section, paragraph, list, code, table, image, block, slide, any, end)", kind)
	}
	opts := map[string]*yaml.Node{}
	add := func(m []*yaml.Node) {
		for i := 0; i+1 < len(m); i += 2 {
			opts[m[i].Value] = m[i+1]
		}
	}
	first := n.Content[1]
	switch {
	case first.Kind == yaml.MappingNode:
		add(first.Content)
	case first.Kind == yaml.ScalarNode && first.Tag != "!!null" && first.Value != "":
		if info.primary == "" {
			return Element{}, fmt.Errorf("%s takes options in a mapping, not %q", kind, first.Value)
		}
		opts[info.primary] = first
	}
	add(n.Content[2:])
	allowed := map[string]bool{}
	for _, o := range info.opts {
		allowed[o] = true
	}
	for k := range opts {
		if !allowed[k] {
			return Element{}, fmt.Errorf("%s: unknown option %q", kind, k)
		}
	}
	e := Element{Kind: kind, Count: -1, Min: -1, Max: -1, MinItems: -1, MaxItems: -1, MinRows: -1, MaxBullets: -1}
	str := func(k string) string {
		if v, ok := opts[k]; ok {
			return v.Value
		}
		return ""
	}
	num := func(k string, dst *int) error {
		if v, ok := opts[k]; ok {
			if err := v.Decode(dst); err != nil || *dst < 0 {
				return fmt.Errorf("%s: %s must be a whole number", kind, k)
			}
		}
		return nil
	}
	flag := func(k string) bool {
		v, ok := opts[k]
		return ok && v.Value == "true"
	}
	e.Severity = str("severity")
	if e.Severity == "" {
		e.Severity = SevWarning
	}
	if !validSeverity(e.Severity) {
		return e, fmt.Errorf("%s: severity must be error, warning or suggestion", kind)
	}
	for _, c := range []struct {
		k string
		d *int
	}{{"level", &e.Level}, {"count", &e.Count}, {"min", &e.Min}, {"max", &e.Max}, {"max-words", &e.MaxWords},
		{"min-items", &e.MinItems}, {"max-items", &e.MaxItems}, {"min-rows", &e.MinRows}, {"max-bullets", &e.MaxBullets}} {
		if err := num(c.k, c.d); err != nil {
			return e, err
		}
	}
	e.Optional, e.Repeat, e.ExceptLast = flag("optional"), flag("repeat"), flag("except-last")
	e.Label, e.Lang, e.Alt, e.Name, e.IDs, e.Unless = str("label"), str("lang"), str("alt"), str("name"), str("ids"), str("unless")
	switch kind {
	case "heading":
		e.Text = str("text")
	case "section":
		e.Text = str("heading")
	case "slide":
		e.HeadingRequired = str("heading") == "required"
	}
	if e.Text != "" && len(e.Text) > 1 && strings.HasPrefix(e.Text, "/") && strings.HasSuffix(e.Text, "/") {
		re, err := compile(e.Text[1 : len(e.Text)-1])
		if err != nil {
			return e, fmt.Errorf("%s: %w", kind, err)
		}
		e.re = re
	}
	if kind == "section" && e.Level == 0 {
		e.Level = 2
	}
	if kind == "list" {
		e.ListKind = str("type")
		if e.ListKind != "" && e.ListKind != "ordered" && e.ListKind != "unordered" && e.ListKind != "task" {
			return e, fmt.Errorf("list: %q is not ordered, unordered or task", e.ListKind)
		}
	}
	if kind == "block" && e.Name == "" {
		return e, fmt.Errorf("block needs a name, such as { block: check }")
	}
	if v, ok := opts["columns"]; ok {
		if err := v.Decode(&e.Columns); err != nil {
			return e, fmt.Errorf("table: columns must be a list of names")
		}
	}
	if v, ok := opts["contains"]; ok {
		els, err := parseElements(v)
		if err != nil {
			return e, fmt.Errorf("contains: %w", err)
		}
		e.Contains = els
	}
	return e, nil
}
