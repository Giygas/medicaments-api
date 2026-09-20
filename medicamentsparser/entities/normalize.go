package entities

import (
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// NormalizeText prepares text for accent-insensitive, case-insensitive
// matching: NFD decomposition strips combining marks (é -> e), letters are
// lowercased, and "+" is treated as a word separator.
//
// It is applied symmetrically at index time (DenominationNormalized,
// LibelleNormalized) and at query time, so "comprime" matches "COMPRIMÉ"
// and vice versa. Folding unaccented text is the identity function, which
// keeps previously-working ASCII queries returning a superset of their
// old results.
func NormalizeText(s string) string {
	decomposed := norm.NFD.String(s)

	var b strings.Builder
	b.Grow(len(decomposed))
	for _, r := range decomposed {
		if unicode.Is(unicode.Mn, r) {
			continue
		}
		if r == '+' {
			r = ' '
		}
		b.WriteRune(unicode.ToLower(r))
	}
	return b.String()
}
