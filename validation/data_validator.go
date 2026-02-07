// Package validation provides data validation functionality for the medicaments API.
package validation

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/giygas/medicaments-api/interfaces"
	"github.com/giygas/medicaments-api/logging"
	"github.com/giygas/medicaments-api/medicamentsparser/entities"
)

// Pre-compiled regex patterns for performance optimization
// Compiled once at package initialization and reused for all validations
var (
	// Input validation: alphanumeric + French accents + safe punctuation
	inputRegex = regexp.MustCompile(`^[a-zA-Z0-9\s\-\.\+'àâäéèêëïîôöùûüÿç]+$`)

	// Dangerous patterns as strings (faster than regex for simple substring matching)
	// strings.Contains is 5-10x faster than regex for these patterns
	dangerousPatterns = []string{
		"<script", "</script>", "javascript:", "vbscript:", "onload=", "onerror=",
		"onclick=", "onmouseover=", "onfocus=", "onblur=", "onchange=", "onsubmit=",
		"eval(", "expression(", "url(", "import ", "@import", "binding(", "behavior(",
		// SQL injection patterns
		"' or ", "\" or ", "union select", "drop table", "delete from", "insert into",
		"update set", "--", "/*", "*/", "xp_", "sp_", "exec(", "execute(",
		// Command injection patterns
		"; ", "| ", "& ", "`", "$(", "${", // Command injection
		// Path traversal patterns
		"../", "..\\", "%2e%2e", "file://", // Path traversal
		// LDAP injection patterns
		"*)(", "*|(", "*)%", // LDAP injection
		// NoSQL injection patterns
		"{$ne:", "{$gt:", "{$where:", "{$or:", "{$regex:", "{$expr:", // NoSQL injection
	}
)

// DataValidatorImpl implements the interfaces.DataValidator interface
type DataValidatorImpl struct{}

// NewDataValidator creates a new data validator
func NewDataValidator() interfaces.DataValidator {
	return &DataValidatorImpl{}
}

// ValidateMedicament checks if a medicament entity is valid
func (v *DataValidatorImpl) ValidateMedicament(m *entities.Medicament) error {
	if m == nil {
		return fmt.Errorf("medicament is nil")
	}

	// Validate CIS code
	if m.Cis <= 0 {
		return fmt.Errorf("invalid CIS code: %d", m.Cis)
	}

	// Validate denomination
	if strings.TrimSpace(m.Denomination) == "" {
		return fmt.Errorf("empty denomination for CIS %d", m.Cis)
	}

	if len(m.Denomination) > 200 {
		return fmt.Errorf("denomination too long for CIS %d: %d characters", m.Cis, len(m.Denomination))
	}

	// Validate forme pharmaceutique
	if len(m.FormePharmaceutique) > 100 {
		return fmt.Errorf("forme pharmaceutique too long for CIS %d: %d characters", m.Cis, len(m.FormePharmaceutique))
	}

	// Validate voies admin
	for _, voie := range m.VoiesAdministration {
		if len(voie) > 50 {
			return fmt.Errorf("voie d'administration too long for CIS %d: %d characters", m.Cis, len(voie))
		}
	}

	// Validate statut
	if len(m.StatusAutorisation) > 50 {
		return fmt.Errorf("status autorisation too long for CIS %d: %d characters", m.Cis, len(m.StatusAutorisation))
	}

	return nil
}

func (v *DataValidatorImpl) CheckDuplicateCIP(presentations []entities.Presentation) error {
	cip7Count := make(map[int]int)
	cip13Count := make(map[int]int)

	for _, pres := range presentations {
		cip7Count[pres.Cip7]++
		cip13Count[pres.Cip13]++
	}

	// Find duplicate CIP7 values
	var cip7Duplicates []int
	for cip, count := range cip7Count {
		if count > 1 {
			cip7Duplicates = append(cip7Duplicates, cip)
		}
	}

	// Find duplicate CIP13 values
	var cip13Duplicates []int
	for cip, count := range cip13Count {
		if count > 1 {
			cip13Duplicates = append(cip13Duplicates, cip)
		}
	}

	// Log duplicates as errors
	if len(cip7Duplicates) > 0 {
		logging.Error("Duplicate CIP7 values detected",
			"count", len(cip7Duplicates),
			"duplicates", cip7Duplicates,
		)
	}

	if len(cip13Duplicates) > 0 {
		logging.Error("Duplicate CIP13 values detected",
			"count", len(cip13Duplicates),
			"duplicates", cip13Duplicates,
		)
	}

	// Return error if any duplicates found
	if len(cip7Duplicates) > 0 || len(cip13Duplicates) > 0 {
		return fmt.Errorf("found %d duplicate CIP7 and %d duplicate CIP13 values",
			len(cip7Duplicates), len(cip13Duplicates))
	}

	return nil
}

// ValidateDataIntegrity performs comprehensive data validation
func (v *DataValidatorImpl) ValidateDataIntegrity(medicaments []entities.Medicament, generiques []entities.GeneriqueList) error {
	// Validate medicaments
	if len(medicaments) == 0 {
		return fmt.Errorf("no medicaments found")
	}

	// Check for duplicate CIS codes
	cisMap := make(map[int]bool)
	for _, med := range medicaments {
		if cisMap[med.Cis] {
			return fmt.Errorf("duplicate CIS code found: %d", med.Cis)
		}
		cisMap[med.Cis] = true

		// Validate individual medicament
		if err := v.ValidateMedicament(&med); err != nil {
			return fmt.Errorf("invalid medicament CIS %d: %w", med.Cis, err)
		}
	}

	// Validate generiques
	if len(generiques) == 0 {
		return fmt.Errorf("no generiques found")
	}

	// Check for duplicate group IDs
	groupIDMap := make(map[int]bool)
	for _, gen := range generiques {
		if groupIDMap[gen.GroupID] {
			return fmt.Errorf("duplicate generique group ID found: %d", gen.GroupID)
		}
		groupIDMap[gen.GroupID] = true

		// Validate generique libelle
		if strings.TrimSpace(gen.Libelle) == "" {
			return fmt.Errorf("empty libelle for generique group %d", gen.GroupID)
		}

		if len(gen.Libelle) > 200 {
			return fmt.Errorf("libelle too long for generique group %d: %d characters", gen.GroupID, len(gen.Libelle))
		}

		// Validate that medicaments in generique exist
		for _, med := range gen.Medicaments {
			if !cisMap[med.Cis] {
				return fmt.Errorf("medicament CIS %d in generique group %d not found in medicaments list", med.Cis, gen.GroupID)
			}
		}
	}

	return nil
}

func (v *DataValidatorImpl) ReportDataQuality(
	medicaments []entities.Medicament,
	generiques []entities.GeneriqueList,
) *interfaces.DataQualityReport {
	report := &interfaces.DataQualityReport{
		DuplicateCIS:                       []int{},
		DuplicateGroupIDs:                  []int{},
		MedicamentsWithoutConditions:       0,
		MedicamentsWithoutGeneriques:       0,
		MedicamentsWithoutPresentations:    0,
		MedicamentsWithoutCompositions:     0,
		GeneriqueOnlyCIS:                   0,
		MedicamentsWithoutConditionsCIS:    []int{},
		MedicamentsWithoutGeneriquesCIS:    []int{},
		MedicamentsWithoutPresentationsCIS: []int{},
		MedicamentsWithoutCompositionsCIS:  []int{},
		GeneriqueOnlyCISList:               []int{},
	}

	// Check 1: Find all duplicate CIS codes
	cisMap := make(map[int]bool)
	for _, med := range medicaments {
		if cisMap[med.Cis] {
			report.DuplicateCIS = append(report.DuplicateCIS, med.Cis)
		}
		cisMap[med.Cis] = true
	}

	// Check 2: Find all duplicate Group IDs
	groupIDMap := make(map[int]bool)
	for _, gen := range generiques {
		if groupIDMap[gen.GroupID] {
			report.DuplicateGroupIDs = append(report.DuplicateGroupIDs, gen.GroupID)
		}
		groupIDMap[gen.GroupID] = true
	}

	// Check 3: Count medicaments without conditions (store first 10 CIS)
	for _, med := range medicaments {
		if len(med.Conditions) == 0 {
			report.MedicamentsWithoutConditions++
			if len(report.MedicamentsWithoutConditionsCIS) < 10 {
				report.MedicamentsWithoutConditionsCIS = append(report.MedicamentsWithoutConditionsCIS, med.Cis)
			}
		}
	}

	// Check 4: Count medicaments without generiques (store first 10 CIS)
	generiquesCISMap := make(map[int]bool)
	for _, gen := range generiques {
		for _, med := range gen.Medicaments {
			generiquesCISMap[med.Cis] = true
		}
	}
	for _, med := range medicaments {
		if !generiquesCISMap[med.Cis] {
			report.MedicamentsWithoutGeneriques++
			if len(report.MedicamentsWithoutGeneriquesCIS) < 10 {
				report.MedicamentsWithoutGeneriquesCIS = append(report.MedicamentsWithoutGeneriquesCIS, med.Cis)
			}
		}
	}

	// Check 5: Count medicaments without presentations (store first 10 CIS)
	for _, med := range medicaments {
		if len(med.Presentation) == 0 {
			report.MedicamentsWithoutPresentations++
			if len(report.MedicamentsWithoutPresentationsCIS) < 10 {
				report.MedicamentsWithoutPresentationsCIS = append(report.MedicamentsWithoutPresentationsCIS, med.Cis)
			}
		}
	}

	// Check 6: Count medicaments without compositions (store ALL CIS)
	for _, med := range medicaments {
		if len(med.Composition) == 0 {
			report.MedicamentsWithoutCompositions++
			report.MedicamentsWithoutCompositionsCIS = append(report.MedicamentsWithoutCompositionsCIS, med.Cis)
		}
	}

	// Check 7: Count generique-only CIS (store first 10 CIS)
	for _, gen := range generiques {
		report.GeneriqueOnlyCIS += len(gen.OrphanCIS)
		for _, cis := range gen.OrphanCIS {
			if len(report.GeneriqueOnlyCISList) < 10 {
				report.GeneriqueOnlyCISList = append(report.GeneriqueOnlyCISList, cis)
			}
		}
	}

	return report
}

// ValidateInput validates user input strings with enhanced security
func (v *DataValidatorImpl) ValidateInput(input string) error {
	if strings.TrimSpace(input) == "" {
		return fmt.Errorf("input cannot be empty")
	}

	if len(input) < 3 {
		return fmt.Errorf("input too short: minimum 3 characters")
	}

	if len(input) > 50 {
		return fmt.Errorf("input too long: maximum 50 characters")
	}

	// Word count validation to prevent DoS attacks with many short words
	words := strings.Fields(input)
	if len(words) > 6 {
		return fmt.Errorf("search query too complex: maximum 6 words allowed")
	}

	// Check for potentially dangerous patterns using string matching (5-10x faster than regex)
	lowerInput := strings.ToLower(input)
	for _, pattern := range dangerousPatterns {
		if strings.Contains(lowerInput, pattern) {
			return fmt.Errorf("input contains potentially dangerous content")
		}
	}

	// Allow only alphanumeric characters, spaces, and safe punctuation
	// More restrictive pattern: letters, numbers, spaces, hyphens, apostrophes, periods, and plus sign
	if !inputRegex.MatchString(input) {
		return fmt.Errorf("input contains invalid characters. Only letters, numbers, spaces, hyphens, apostrophes, periods, plus sign, and common French accented characters are allowed")
	}

	// Additional checks for repeated characters (potential DoS)
	if v.hasExcessiveRepetition(input) {
		return fmt.Errorf("input contains excessive character repetition")
	}

	return nil
}

// ValidateCIP validates CIP codes
// CIP codes are numeric identifiers 7 or 13 digits long
// No regex used - strconv.Atoi() validates numeric format for free
func (v *DataValidatorImpl) ValidateCIP(input string) (int, error) {
	trimmedInput := strings.TrimSpace(input)
	if trimmedInput == "" {
		return -1, fmt.Errorf("input cannot be empty")
	}

	// Reject if original input contained whitespace (spaces, tabs, etc.)
	if len(input) != len(trimmedInput) {
		return -1, fmt.Errorf("input contains invalid characters. Only numeric characters are allowed")
	}

	if len(trimmedInput) != 7 && len(trimmedInput) != 13 {
		return -1, fmt.Errorf("CIP should have 7 or 13 characters")
	}

	// strconv.Atoi() validates that input contains only digits
	// Returns error for any non-numeric characters (no regex overhead)
	cip, err := strconv.Atoi(trimmedInput)
	if err != nil {
		return -1, fmt.Errorf("input contains invalid characters. Only numeric characters are allowed")
	}

	return cip, nil
}

// ValidateCIS validates CIS codes
// CIP codes are numeric identifiers 8 digits long
// No regex used - strconv.Atoi() validates numeric format for free
func (v *DataValidatorImpl) ValidateCIS(input string) (int, error) {
	trimmedInput := strings.TrimSpace(input)
	if trimmedInput == "" {
		return -1, fmt.Errorf("input cannot be empty")
	}

	// Reject if original input contained whitespace (spaces, tabs, etc.)
	if len(input) != len(trimmedInput) {
		return -1, fmt.Errorf("input contains invalid characters. Only numeric characters are allowed")
	}

	if len(trimmedInput) != 8 {
		return -1, fmt.Errorf("CIS should have 8 digits")
	}

	// strconv.Atoi() validates that input contains only digits
	// Returns error for any non-numeric characters (no regex overhead)
	cis, err := strconv.Atoi(trimmedInput)
	if err != nil {
		return -1, fmt.Errorf("input contains invalid characters. Only numeric characters are allowed")
	}

	return cis, nil
}

// hasExcessiveRepetition checks for potential DoS patterns with excessive character repetition
func (v *DataValidatorImpl) hasExcessiveRepetition(input string) bool {
	// Check for the same character repeated more than 10 times consecutively
	for i := 0; i < len(input)-10; i++ {
		allSame := true
		for j := 1; j <= 10; j++ {
			if input[i] != input[i+j] {
				allSame = false
				break
			}
		}
		if allSame {
			return true
		}
	}
	return false
}
