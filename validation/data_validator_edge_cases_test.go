package validation

import (
	"testing"
)

// ============================================================================
// EDGE CASE TESTS
// ============================================================================

func TestValidateInput_OnlySpecialCharacters(t *testing.T) {
	validator := NewDataValidator()

	testCases := []struct {
		name  string
		input string
	}{
		{"Only special chars", "!@#$%^&*()"},
		{"Only punctuation", "...,,,---"},
		{"Mixed special", "!!!???"},

		{"At signs only", "@@@@@"},
		{"Hash only", "####"},
		{"Underscore only", "____"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validator.ValidateInput(tc.input)
			if err == nil {
				t.Errorf("Expected error for input with only special characters: '%s'", tc.input)
			}
		})
	}
}

func TestValidateInput_NullBytes(t *testing.T) {
	validator := NewDataValidator()

	inputWithNull := "abc\x00def"
	err := validator.ValidateInput(inputWithNull)
	if err == nil {
		t.Errorf("Expected error for input with null bytes")
	}
}

func TestValidateInput_UnicodeBeyondFrench(t *testing.T) {
	validator := NewDataValidator()

	testCases := []struct {
		name  string
		input string
	}{
		{"Japanese characters", "漢字テスト"},
		{"Arabic characters", "مرحبا"},
		{"Hebrew characters", "שלום"},
		{"Cyrillic characters", "Привет"},
		{"Thai characters", "สวัสดี"},
		{"Korean characters", "안녕하세요"},
		{"Chinese characters", "你好"},
		{"Greek characters", "Γειά"},
		{"Hindi characters", "नमस्ते"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// These should be rejected as they don't match the ASCII-only pattern
			err := validator.ValidateInput(tc.input)
			if err == nil {
				t.Errorf("Expected error for non-French Unicode input: '%s'", tc.input)
			}
		})
	}
}

func TestValidateInput_Emojis(t *testing.T) {
	validator := NewDataValidator()

	testCases := []struct {
		name  string
		input string
	}{
		{"Simple emoji", "😀"},
		{"Medicine emoji", "💊"},
		{"Pill emoji", "💊"},
		{"Multiple emojis", "😀😀😀"},
		{"Emoji with text", "test😀test"},
		{"Flag emoji", "🏳"},
		{"Heart emoji", "❤️"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validator.ValidateInput(tc.input)
			if err == nil {
				t.Errorf("Expected error for input with emojis: '%s'", tc.input)
			}
		})
	}
}

func TestValidateCIP_WithLeadingZeros(t *testing.T) {
	validator := NewDataValidator()

	testCases := []struct {
		name     string
		input    string
		expected int
	}{
		{"CIP7 with leading zeros", "0012345", 12345},
		{"CIP7 all zeros", "0000000", 0},
		{"CIP13 with leading zeros", "0123456789012", 123456789012},
		{"CIP13 all zeros", "0000000000000", 0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := validator.ValidateCIP(tc.input)
			if err != nil {
				t.Errorf("Unexpected error for valid CIP with leading zeros '%s': %v", tc.input, err)
			}
			// Leading zeros should be preserved in the integer conversion
			if result != tc.expected {
				t.Errorf("Expected %d for '%s', got %d", tc.expected, tc.input, result)
			}
		})
	}
}

func TestValidateCIP_WithSpaces(t *testing.T) {
	validator := NewDataValidator()

	testCases := []struct {
		name  string
		input string
	}{
		{"Leading space", " 1234567"},
		{"Trailing space", "1234567 "},
		{"Multiple spaces", "  1234567  "},
		{"Middle space", "123 4567"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := validator.ValidateCIP(tc.input)
			if err == nil {
				t.Errorf("Expected error for CIP with spaces: '%s'", tc.input)
			}
		})
	}
}

func TestValidateCIS_WithLeadingZeros(t *testing.T) {
	validator := NewDataValidator()

	testCases := []struct {
		name     string
		input    string
		expected int
	}{
		{"CIS with leading zeros", "00012345", 12345},
		{"CIS all zeros", "00000000", 0},
		{"CIS multiple leading zeros", "00000123", 123},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := validator.ValidateCIS(tc.input)
			if err != nil {
				t.Errorf("Unexpected error for valid CIS with leading zeros '%s': %v", tc.input, err)
			}
			// Leading zeros should be preserved in the integer conversion
			if result != tc.expected {
				t.Errorf("Expected %d for '%s', got %d", tc.expected, tc.input, result)
			}
		})
	}
}

func TestValidateInput_VeryLongInput(t *testing.T) {
	validator := NewDataValidator()

	// Test with input exactly at boundary
	validInput := "abcdeabcdeabcdeabcdeabcdeabcdeabcdeabcdeabcdeabcde" // 50 chars
	err := validator.ValidateInput(validInput)
	if err != nil {
		t.Errorf("Expected no error for input at max length (50 chars), got: %v", err)
	}

	// Test with input just over boundary
	invalidInput := validInput + "a" // 51 chars
	err = validator.ValidateInput(invalidInput)
	if err == nil {
		t.Error("Expected error for input exceeding max length (51 chars)")
	}
}

func TestValidateInput_MinimumLength(t *testing.T) {
	validator := NewDataValidator()

	testCases := []struct {
		name  string
		input string
		valid bool
	}{
		{"Exactly 2 chars", "ab", false},
		{"Exactly 3 chars", "abc", true},
		{"Exactly 1 char", "a", false},
		{"Empty string", "", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validator.ValidateInput(tc.input)
			if tc.valid && err != nil {
				t.Errorf("Expected no error for valid input '%s', got: %v", tc.input, err)
			}
			if !tc.valid && err == nil {
				t.Errorf("Expected error for invalid input '%s', got: %v", tc.input, err)
			}
		})
	}
}
