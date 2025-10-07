package validation

import (
	"fmt"
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
	for i := 0; i < 201; i++ {
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
	for i := 0; i < 101; i++ {
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
	for i := 0; i < 51; i++ {
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
	for i := 0; i < 51; i++ {
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
	for i := 0; i < 201; i++ {
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
		"paracétamol",
		"ibuprofène 200mg",
		"aspirine-500",
		"test'medicament",
		"dr. smith",
		"àâäéèêëïîôöùûüÿç",
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
	for i := 0; i < 51; i++ {
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
		"test+medicament",
		"test=medicament",
		"test|medicament",
		"test\\medicament",
		"test/medicament",
		"test<medicament>",
		"test[medicament]",
		"test{medicament}",
		"test(medicament)",
		"test^medicament",
		"test`medicament",
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

			expectedError := "input contains invalid characters. Only letters, numbers, spaces, hyphens, apostrophes, periods, and common French accented characters are allowed"
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
	for i := 0; i < b.N; i++ {
		validator.ValidateMedicament(medicament)
	}
}

func BenchmarkValidateInput(b *testing.B) {
	validator := NewDataValidator()

	input := "paracétamol 500mg"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		validator.ValidateInput(input)
	}
}

func BenchmarkValidateDataIntegrity(b *testing.B) {
	validator := NewDataValidator()

	medicaments := make([]entities.Medicament, 1000)
	for i := 0; i < 1000; i++ {
		medicaments[i] = entities.Medicament{
			Cis:          i,
			Denomination: "Test Medicament",
		}
	}

	generiques := make([]entities.GeneriqueList, 100)
	for i := 0; i < 100; i++ {
		generiques[i] = entities.GeneriqueList{
			GroupID: i,
			Libelle: "Test Generique",
			Medicaments: []entities.GeneriqueMedicament{
				{Cis: i * 10, Denomination: "Test"},
			},
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		validator.ValidateDataIntegrity(medicaments, generiques)
	}
}
