package index

import (
	"strings"
	"unicode"
)

// Fold is a title with case and accents ignored, for sorting and
// comparing: "Éclair" and "eclair" are the same.
func Fold(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(strings.TrimSpace(s)) {
		switch {
		case unicode.Is(unicode.Mn, r):
			continue // a combining accent
		case r < 0x80:
			b.WriteRune(r)
		default:
			if f, ok := accents[r]; ok {
				b.WriteString(f)
			} else {
				b.WriteRune(r)
			}
		}
	}
	return b.String()
}

var accents = func() map[rune]string {
	m := map[rune]string{}
	for base, set := range map[string]string{
		"a": "àáâãäåāăąǎ", "c": "çćĉċč", "d": "ďđð", "e": "èéêëēĕėęě", "g": "ĝğġģ", "h": "ĥħ",
		"i": "ìíîïĩīĭįıǐ", "j": "ĵ", "k": "ķ", "l": "ĺļľŀł", "n": "ñńņňŉ", "o": "òóôõöøōŏőǒ",
		"r": "ŕŗř", "s": "śŝşšș", "t": "ţťŧț", "u": "ùúûüũūŭůűųǔ", "w": "ŵ", "y": "ýÿŷ", "z": "źżž",
	} {
		for _, r := range set {
			m[r] = base
		}
	}
	m['ß'], m['æ'], m['œ'], m['þ'] = "ss", "ae", "oe", "th"
	return m
}()
