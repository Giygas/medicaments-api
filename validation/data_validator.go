// Package validation provides data validation functionality for the medicaments API.
package validation

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/giygas/medicaments-api/interfaces"
	"github.com/giygas/medicaments-api/medicamentsparser/entities"
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
	groupIdMap := make(map[int]bool)
	for _, gen := range generiques {
		if groupIdMap[gen.GroupID] {
			return fmt.Errorf("duplicate generique group ID found: %d", gen.GroupID)
		}
		groupIdMap[gen.GroupID] = true

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

	// Check for potentially dangerous patterns
	dangerousPatterns := []string{
		"<script", "</script>", "javascript:", "vbscript:", "onload=", "onerror=",
		"onclick=", "onmouseover=", "onfocus=", "onblur=", "onchange=", "onsubmit=",
		"eval(", "expression(", "url(", "import ", "@import", "binding(", "behavior(",
	}

	lowerInput := strings.ToLower(input)
	for _, pattern := range dangerousPatterns {
		if strings.Contains(lowerInput, pattern) {
			return fmt.Errorf("input contains potentially dangerous content")
		}
	}

	// Allow only alphanumeric characters, spaces, and safe punctuation
	// More restrictive pattern: letters, numbers, spaces, hyphens, apostrophes, and periods
	validPattern := regexp.MustCompile(`^[a-zA-Z0-9\s\-\.'àâäéèêëïîôöùûüÿç]+$`)
	if !validPattern.MatchString(input) {
		return fmt.Errorf("input contains invalid characters. Only letters, numbers, spaces, hyphens, apostrophes, periods, and common French accented characters are allowed")
	}

	// Additional checks for repeated characters (potential DoS)
	if v.hasExcessiveRepetition(input) {
		return fmt.Errorf("input contains excessive character repetition")
	}

	return nil
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
