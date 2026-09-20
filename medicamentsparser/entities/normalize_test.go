package entities

import (
	"strings"
	"testing"
)

func TestNormalizeText(t *testing.T) {
	cases := []struct {
		in, want string
		desc     string
	}{
		{"PARACETAMOL", "paracetamol", "uppercase ASCII lowercased"},
		{"comprime", "comprime", "unaccented text unchanged (identity)"},
		{"Comprimé", "comprime", "lowercase accented é folded"},
		{"COMPRIMÉ", "comprime", "uppercase accented É folded and lowercased"},
		{"Pelliculé+sécable", "pellicule secable", "+ becomes separator, accents folded"},
		{"ÀÉÎÔÛàéîôû", "aeiouaeiou", "full French accent set folded"},
		{"ç Ç œ Œ", "c c œ œ", "cedilla folded; ligatures kept (no NFD mapping)"},
		{" solution   buvable ", " solution   buvable ", "whitespace preserved for strings.Fields"},
		{"500 mg/5 ml", "500 mg/5 ml", "digits and separators untouched"},
		{"", "", "empty string"},
	}

	for _, tc := range cases {
		if got := NormalizeText(tc.in); got != tc.want {
			t.Errorf("NormalizeText(%q) = %q, want %q (%s)", tc.in, got, tc.want, tc.desc)
		}
	}
}

// TestNormalizeTextChangedAgainstOldFormula pins the compatibility
// guarantee: for pure ASCII input, NormalizeText is exactly the previous
// normalization pipeline (ToLower + "+" -> " ").
func TestNormalizeTextChangedAgainstOldFormula(t *testing.T) {
	for _, s := range []string{
		"DOLIPRANE 1000 mg, comprime",
		"Paracetamol 500 mg + Codeine (Phosphate) hemihydrate 30 mg",
		"IBUPROFENE 400 mg",
	} {
		old := func(s string) string {
			return strings.ReplaceAll(strings.ToLower(s), "+", " ")
		}
		if got := NormalizeText(s); got != old(s) {
			t.Errorf("NormalizeText(%q) = %q, old formula = %q", s, got, old(s))
		}
	}
}
