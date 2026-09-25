package ansmdocs

import (
	"regexp"
	"strings"
	"testing"
)

// TestNormalizeContenu covers the audited cleaning rules with real artifacts
// harvested from ANSM pages (September 2026 corpus audit: 300 CIS pages,
// 444 documents, 509 dot runs, zero invariant violations).
func TestNormalizeContenu(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			// ADRIBLASTINE (audit sample): classic composition leader.
			name: "composition leader dots",
			in:   "<p>Chlorhydrate de doxorubicine................................................................................................. 10 mg</p><p>Pour un flacon.</p>",
			want: "<p>Chlorhydrate de doxorubicine 10 mg</p><p>Pour un flacon.</p>",
		},
		{
			// MORNIFLUMATE (audit "suspicious" case): leader with no space
			// between the dots and the quantity in the source.
			name: "leader dots flush against quantity",
			in:   "<p>Morniflumate................................................................400 mg</p>",
			want: "<p>Morniflumate 400 mg</p>",
		},
		{
			// Homeopathic composition (audit shortest run): 6 dots.
			name: "short leader run",
			in:   "<p>volumes d&#39;oxygène actif...... 100 ml</p>",
			want: "<p>volumes d&#39;oxygène actif 100 ml</p>",
		},
		{
			// Genuine ellipses are three dots and must survive.
			name: "genuine ellipsis preserved",
			in:   "<p>Des signes (tachycardie, tremblements, troubles du rythme...)</p>",
			want: "<p>Des signes (tachycardie, tremblements, troubles du rythme...)</p>",
		},
		{
			// METOJECT discovery: U+0005 control bytes used as fill.
			name: "control characters stripped",
			in:   "<p>Méthotrexate ........\x05\x05\x05\x05\x05 15 mg</p>",
			want: "<p>Méthotrexate 15 mg</p>",
		},
		{
			// LEVOTHYROX: pseudo-list bullets are "·" plus a tab-wide gap.
			name: "pseudo-list bullet gap collapsed",
			in:   "<p>·         Hypothyroïdies,</p><p>·         Circonstances associées.</p>",
			want: "<p>· Hypothyroïdies,</p><p>· Circonstances associées.</p>",
		},
		{
			name: "trailing whitespace trimmed",
			in:   "<p>La posologie est de 25 µg par jour. </p>",
			want: "<p>La posologie est de 25 µg par jour.</p>",
		},
		{
			name: "leading space after opening tag trimmed",
			in:   "<p>\n\tTexte en retrait.</p>",
			want: "<p>Texte en retrait.</p>",
		},
		{
			// Paragraph-separating whitespace between tags is kept.
			name: "inter-tag space kept",
			in:   "<p>a</p> <p>b</p>",
			want: "<p>a</p> <p>b</p>",
		},
		{
			// Space around inline elements is preserved so words never
			// merge when tags are stripped.
			name: "inline element spacing preserved",
			in:   "<p>grosse <b>Alerte</b> et <i>grave</i></p>",
			want: "<p>grosse <b>Alerte</b> et <i>grave</i></p>",
		},
		{
			name: "tabs and newlines collapsed to single space",
			in:   "<p>Voie\n\norale\t\tuniquement.</p>",
			want: "<p>Voie orale uniquement.</p>",
		},
		{
			name: "source spacer paragraph dropped",
			in:   "<p>x</p><p> </p><p>y</p>",
			want: "<p>x</p><p>y</p>",
		},
		{
			// A paragraph holding only a leader line becomes empty and is
			// dropped like any other spacer.
			name: "dots-only paragraph dropped",
			in:   "<p>............................................</p><p>keep</p>",
			want: "<p>keep</p>",
		},
		{
			// The collapse keeps one space between an inline element and
			// the next word: adjacent words never merge.
			name: "superscript spacing preserved",
			in:   "<p>au cours de la 1<sup>ère</sup> moitié de la grossesse</p>",
			want: "<p>au cours de la 1<sup>ère</sup> moitié de la grossesse</p>",
		},
		{
			// Single dots are content: rubrique numbers and decimals survive.
			name: "single dots preserved",
			in:   "<p>Voir rubrique 4.2. Dose de 20,00 mg.</p>",
			want: "<p>Voir rubrique 4.2. Dose de 20,00 mg.</p>",
		},
		{
			// Already-clean fragments pass through byte-identical.
			name: "idempotent on clean fragment",
			in:   "<p>CARVEDILOL VIATRIS 25 mg, comprimé pelliculé sécable</p>",
			want: "<p>CARVEDILOL VIATRIS 25 mg, comprimé pelliculé sécable</p>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeContenu(tt.in); got != tt.want {
				t.Errorf("normalizeContenu() =\n%q\nwant\n%q", got, tt.want)
			}
		})
	}
}

// Audit regexps reused for the end-to-end assertions.
var (
	residualDotRe  = regexp.MustCompile(`\.{4,}`)
	residualSpace  = regexp.MustCompile(`\s{2,}`)
	residualCtrlRe = regexp.MustCompile("[\x00-\x08\x0B\x0C\x0E-\x1F\x7F]")
)

// TestExtractNormalizesContenu runs the full pipeline on the real fixture
// page (which carries a 126-dot composition leader, "·" pseudo-lists and
// multi-space runs) and asserts no artifact survives extraction.
func TestExtractNormalizesContenu(t *testing.T) {
	page := loadFixture(t, "both_tabs.html")
	doc, err := Extract("61266250", TypeRCP, page)
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if len(doc.Sections) == 0 {
		t.Fatal("no sections extracted")
	}
	for _, sec := range doc.Sections {
		if residualDotRe.MatchString(sec.Contenu) {
			t.Errorf("section %s contenu still carries leader dots", sec.ID)
		}
		if residualSpace.MatchString(sec.Contenu) {
			t.Errorf("section %s contenu still carries whitespace runs: %q", sec.ID, sec.Contenu)
		}
		if residualCtrlRe.MatchString(sec.Contenu) {
			t.Errorf("section %s contenu still carries control characters", sec.ID)
		}
		if strings.Contains(sec.Contenu, "<p></p>") {
			t.Errorf("section %s contenu still carries empty paragraphs", sec.ID)
		}
	}
	// Clean content is untouched through the whole pipeline.
	sec1 := findSection(t, doc, "1")
	if want := "<p>CARVEDILOL VIATRIS 25 mg, comprimé pelliculé sécable</p>"; sec1.Contenu != want {
		t.Errorf("section 1 contenu = %q, want %q", sec1.Contenu, want)
	}
}
