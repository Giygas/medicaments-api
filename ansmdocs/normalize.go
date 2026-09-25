package ansmdocs

import (
	"regexp"
	"strings"
)

// Contenu normalization rules, validated by the September 2026 corpus audit
// (300 random CIS pages, 444 documents, 9,811 sections, zero invariant
// violations — the audit verified per section that only these artifact
// classes are removed, never wording):
//
//   - leader dots: French pharmacopoeia composition lines align the
//     substance name and the quantity with full-stop runs (print layout;
//     observed runs span 6–133 dots, always followed by the quantity).
//     Genuine ellipses are always three dots and every 4+ run observed in
//     the audit was a layout artifact, so the threshold eats no real
//     punctuation. Runs are replaced by a single space: no separator is
//     inserted because quantities always start with a digit (splitting is
//     trivial) and hyphens already occur in substance names (DL-Lysine).
//   - control characters: some ANSM pages carry C0 control bytes (U+0005
//     observed as composition fill, 21 bytes in a row) that would survive
//     into the served JSON as invisible garbage. Tab, LF and CR are kept
//     (whitespace, handled by the collapse rule).
//   - whitespace runs: the source renders pseudo-lists as "·" plus a
//     tab-wide gap and leaves trailing spaces; HTML collapses runs to a
//     single space when rendering anyway, so the served text does the same.
//     The collapse keeps one space between words and tags, so adjacent
//     inline elements (1<sup>ère</sup> moitié) never merge.
//   - spacer paragraphs: paragraphs left empty by the rules above (or empty
//     in the source) carry no content and are dropped.
//
// normalizeContenu is applied after Sanitize at extraction time, so the
// cleaned text is what gets cached and served. It is idempotent: an already
// clean fragment passes through unchanged.
var (
	leaderDotRe  = regexp.MustCompile(`\.{4,}`)
	ctrlCharRe   = regexp.MustCompile("[\x00-\x08\x0B\x0C\x0E-\x1F\x7F]")
	wsCollapseRe = regexp.MustCompile(`\s+`)
	// Whitespace hugging a block-element boundary (trailing space before
	// </p>, leading space after <p>) renders as nothing and is dropped;
	// spaces between two tags are kept (they read as paragraph separation).
	// Inline elements (b, i, sup …) never match, so text-adjacent spaces
	// around them survive and words never merge.
	blockCloseRe = regexp.MustCompile(`\s+(</(?:p|h3|h4|br|ul|ol|li|table|thead|tbody|tr|td|th)>)`)
	blockOpenRe  = regexp.MustCompile(`(<(?:p|h3|h4|br|ul|ol|li|table|thead|tbody|tr|td|th)\b[^>]*>)\s+`)
	emptyParaRe  = regexp.MustCompile(`<p>\s*</p>`)
)

// normalizeContenu removes print-layout artifacts from a sanitized section
// contenu (leader dots, control characters, redundant whitespace, empty
// paragraphs). Wording stays verbatim.
func normalizeContenu(s string) string {
	s = leaderDotRe.ReplaceAllString(s, " ")
	s = ctrlCharRe.ReplaceAllString(s, "")
	s = wsCollapseRe.ReplaceAllString(s, " ")
	s = blockCloseRe.ReplaceAllString(s, "$1")
	s = blockOpenRe.ReplaceAllString(s, "$1")
	s = strings.TrimSpace(s)
	s = emptyParaRe.ReplaceAllString(s, "")
	return strings.TrimSpace(s)
}
