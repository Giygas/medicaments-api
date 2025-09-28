// Package medicamentsparser provides functionality for downloading and parsing medicament data from external sources.
package medicamentsparser

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/giygas/medicamentsfr/logging"
	"github.com/giygas/medicamentsfr/medicamentsparser/entities"
)

func validateMedicament(m *entities.Medicament) error {
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

func ParseAllMedicaments() []entities.Medicament {

	// Download the neccesary files from https://base-donnees-publique.medicaments.gouv.fr/telechargement
	downloadAndParseAll()

	//Make all the json files concurrently
	var wg sync.WaitGroup
	wg.Add(5)

	conditionsChan := make(chan []entities.Condition)
	presentationsChan := make(chan []entities.Presentation)
	specialitesChan := make(chan []entities.Specialite)
	generiquesChan := make(chan []entities.Generique)
	compositionsChan := make(chan []entities.Composition)

	go func() {
		conditionsChan <- makeConditions(&wg)
	}()

	go func() {
		presentationsChan <- makePresentations(&wg)
	}()

	go func() {
		specialitesChan <- makeSpecialites(&wg)
	}()

	go func() {
		generiquesChan <- makeGeneriques(&wg)
	}()

	go func() {
		compositionsChan <- makeCompositions(&wg)
	}()

	wg.Wait()

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

	presentationsMap := make(map[int][]entities.Presentation)
	for _, pres := range presentations {
		presentationsMap[pres.Cis] = append(presentationsMap[pres.Cis], pres)
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
		if err := validateMedicament(medicament); err != nil {
			logging.Warn("Skipping invalid medicament: ", "error", err, "cis", med.Cis)
			continue
		}

		medicamentsSlice = append(medicamentsSlice, *medicament)

	}

	logging.Info("All medicaments parsed successfully")
	jsonMedicament, err := json.MarshalIndent(medicamentsSlice, "", "  ")
	if err != nil {
		logging.Error("Error marshalling medicaments", "error", err)
		return nil
	}

	err = os.WriteFile("src/Medicaments.json", jsonMedicament, 0644)
	if err != nil {
		logging.Error("Error writing Medicaments.json", "error", err)
		return nil
	}
	os.Stdout.Sync()

	conditions = nil
	presentations = nil
	specialites = nil
	generiques = nil
	compositions = nil

	return medicamentsSlice
}
