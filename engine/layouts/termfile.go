package layouts

import (
	"strings"
	"unicode"
)

// TermSlug is the file name (without .md) of a term: lower case, accents
// folded, every run of other characters one "-", only letters, digits and "-"
// kept. "" when the term has none.
func TermSlug(term string) string {
	var b strings.Builder
	dash := false
	for _, r := range foldKey(term) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			dash = false
		} else if !dash && b.Len() > 0 {
			b.WriteByte('-')
			dash = true
		}
	}
	return strings.TrimSuffix(b.String(), "-")
}

// TermText is the text of a new term page: the term layout in the frontmatter
// (no title: the H1 is the term), then the skeleton of the term page template
// with {{title}} filled in. A non-empty definition replaces the skeleton's
// first paragraph after the H1.
func TermText(skeleton, term, definition string) string {
	body := strings.ReplaceAll(strings.ReplaceAll(skeleton, "\r\n", "\n"), "{{title}}", term)
	if !strings.HasPrefix(strings.TrimLeft(body, "\n"), "# ") {
		body = "# " + term + "\n\n" + strings.TrimLeft(body, "\n")
	}
	if definition = strings.Join(strings.Fields(definition), " "); definition != "" {
		lines := strings.Split(body, "\n")
		i := 0
		for i < len(lines) && strings.TrimSpace(lines[i]) == "" {
			i++
		}
		i++ // the H1
		for i < len(lines) && strings.TrimSpace(lines[i]) == "" {
			i++
		}
		j := i
		for j < len(lines) && strings.TrimSpace(lines[j]) != "" && !strings.HasPrefix(lines[j], "#") {
			j++
		}
		rest := append([]string{definition}, lines[j:]...)
		body = strings.Join(append(lines[:i:i], rest...), "\n")
	}
	if !strings.HasSuffix(body, "\n") {
		body += "\n"
	}
	return "---\nlayout: term\n---\n" + body
}
