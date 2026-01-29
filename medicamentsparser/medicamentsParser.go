// Package medicamentsparser provides functionality for downloading and parsing medicament data from external sources.
package medicamentsparser

import (
	"fmt"
	"sync"

	"github.com/giygas/medicaments-api/logging"
	"github.com/giygas/medicaments-api/medicamentsparser/entities"
)

func validateMedicamenti(m *entities.Medicament) error {
	if m.Cis <= 0 {
		return fmt.Errorf("invalid CIS: %d", m.Cis)
	}
	if m.Denomination == "" {
		return fmt.Errorf("missing denomination")
	}
	if m.FormePharmaceutique == "" {
		return fmt.Errorf("missing forme pharmaceutique")
	}
	// Add more checks as needed
	return nil
}

func checkDuplicateCIP(presentations []entities.Presentation) error {
	// Check for duplicate CIP7 and CIP13 values before creating the maps
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

func ParseAllMedicaments() ([]entities.Medicament, map[int]entities.Presentation, map[int]entities.Presentation, error) {

	// Download the neccesary files from https://base-donnees-publique.medicaments.gouv.fr/telechargement
	if err := downloadAndParseAll(); err != nil {
		logging.Error("Failed to download and parse files", "error", err)
		return nil, nil, nil, fmt.Errorf("failed to download files: %w", err)
	}

	//Make all the json files concurrently
	var wg sync.WaitGroup
	wg.Add(5)

	conditionsChan := make(chan []entities.Condition)
	presentationsChan := make(chan []entities.Presentation)
	specialitesChan := make(chan []entities.Specialite)
	generiquesChan := make(chan []entities.Generique)
	compositionsChan := make(chan []entities.Composition)
	errorChan := make(chan error, 5)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				logging.Error("Panic recovered in conditions goroutine", "panic", r)
				errorChan <- fmt.Errorf("panic in conditions: %v", r)
			}
		}()
		result, err := makeConditions(&wg)
		if err != nil {
			logging.Error("Failed to parse conditions", "error", err)
			errorChan <- err
			return
		}
		conditionsChan <- result
	}()

	go func() {
		defer func() {
			if r := recover(); r != nil {
				logging.Error("Panic recovered in presentations goroutine", "panic", r)
				errorChan <- fmt.Errorf("panic in presentations: %v", r)
			}
		}()
		result, err := makePresentations(&wg)
		if err != nil {
			logging.Error("Failed to parse presentations", "error", err)
			errorChan <- err
			return
		}
		presentationsChan <- result
	}()

	go func() {
		defer func() {
			if r := recover(); r != nil {
				logging.Error("Panic recovered in specialites goroutine", "panic", r)
				errorChan <- fmt.Errorf("panic in specialites: %v", r)
			}
		}()
		result, err := makeSpecialites(&wg)
		if err != nil {
			logging.Error("Failed to parse specialites", "error", err)
			errorChan <- err
			return
		}
		specialitesChan <- result
	}()

	go func() {
		defer func() {
			if r := recover(); r != nil {
				logging.Error("Panic recovered in generiques goroutine", "panic", r)
				errorChan <- fmt.Errorf("panic in generiques: %v", r)
			}
		}()
		result, err := makeGeneriques(&wg)
		if err != nil {
			logging.Error("Failed to parse generiques", "error", err)
			errorChan <- err
			return
		}
		generiquesChan <- result
	}()

	go func() {
		defer func() {
			if r := recover(); r != nil {
				logging.Error("Panic recovered in compositions goroutine", "panic", r)
				errorChan <- fmt.Errorf("panic in compositions: %v", r)
			}
		}()
		result, err := makeCompositions(&wg)
		if err != nil {
			logging.Error("Failed to parse compositions", "error", err)
			errorChan <- err
			return
		}
		compositionsChan <- result
	}()

	wg.Wait()

	// Check for any errors that occurred during concurrent processing
	select {
	case err := <-errorChan:
		return nil, nil, nil, fmt.Errorf("error during data parsing: %w", err)
	default:
		// No errors, continue processing
	}

	conditions := <-conditionsChan
	presentations := <-presentationsChan
	specialites := <-specialitesChan
	generiques := <-generiquesChan
	compositions := <-compositionsChan

	fmt.Printf("Number of conditions to process: %d\n", len(conditions))
	fmt.Printf("Number of presentations to process: %d\n", len(presentations))
	fmt.Printf("Number of generiques to process: %d\n", len(generiques))
	fmt.Printf("Number of specialites to process: %d\n", len(specialites))

	conditionsChan = nil
	presentationsChan = nil
	specialitesChan = nil
	generiquesChan = nil
	compositionsChan = nil

	// Make lookup maps (this is s O(n) task, but it makes possible searching as O(1))
	compositionsMap := make(map[int][]entities.Composition)
	for _, comp := range compositions {
		compositionsMap[comp.Cis] = append(compositionsMap[comp.Cis], comp)
	}

	generiquesMap := make(map[int][]entities.Generique)
	for _, gen := range generiques {
		generiquesMap[gen.Cis] = append(generiquesMap[gen.Cis], gen)

	}

	// Check for duplicate CIP values before building maps
	if err := checkDuplicateCIP(presentations); err != nil {
		logging.Warn("Duplicate CIP values detected, last occurrence will be used", "error", err)
	}

	presentationsMap := make(map[int][]entities.Presentation)
	presentationsCIP7Map := make(map[int]entities.Presentation)
	presentationsCIP13Map := make(map[int]entities.Presentation)
	for _, pres := range presentations {
		presentationsMap[pres.Cis] = append(presentationsMap[pres.Cis], pres)
		presentationsCIP7Map[pres.Cip7] = pres
		presentationsCIP13Map[pres.Cip13] = pres
	}

	conditionsMap := make(map[int][]string)
	for _, cond := range conditions {
		conditionsMap[cond.Cis] = append(conditionsMap[cond.Cis], cond.Condition)
	}

	medicamentsSlice := make([]entities.Medicament, 0, len(specialites))

	for _, med := range specialites {

		medicament := new(entities.Medicament)

		medicament.Cis = med.Cis
		medicament.Denomination = med.Denomination
		medicament.FormePharmaceutique = med.FormePharmaceutique
		medicament.VoiesAdministration = med.VoiesAdministration
		medicament.StatusAutorisation = med.StatusAutorisation
		medicament.TypeProcedure = med.TypeProcedure
		medicament.EtatComercialisation = med.EtatComercialisation
		medicament.DateAMM = med.DateAMM
		medicament.Titulaire = med.Titulaire
		medicament.SurveillanceRenforcee = med.SurveillanceRenforcee

		// Using map for O(1) lookup
		// Get all the compositions of this medicament
		if comps, exists := compositionsMap[med.Cis]; exists {
			medicament.Composition = comps
		}

		// Get all the generiques of this medicament
		if gen, exists := generiquesMap[med.Cis]; exists {
			medicament.Generiques = gen
		}

		// Get all the presentations of this medicament
		if pres, exists := presentationsMap[med.Cis]; exists {
			medicament.Presentation = pres
		}

		// Get the conditions of this medicament
		if cond, exists := conditionsMap[med.Cis]; exists {
			medicament.Conditions = cond
		}

		// Validate the medicament structure
		if err := validateMedicamenti(medicament); err != nil {
			logging.Warn("Skipping invalid medicament: ", "error", err, "cis", med.Cis)
			continue
		}

		medicamentsSlice = append(medicamentsSlice, *medicament)

	}

	logging.Info("All medicaments parsed successfully",
		"medicaments_parsed", len(medicamentsSlice))

	return medicamentsSlice, presentationsCIP7Map, presentationsCIP13Map, nil
}
