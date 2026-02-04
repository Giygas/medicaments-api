package medicamentsparser

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/giygas/medicaments-api/logging"
	"github.com/giygas/medicaments-api/medicamentsparser/entities"
)

// readGeneriquesFromTSV reads generiques data directly from TSV file
//
// Returns: map where the key is the groupID (string) and the value is an array
// of the medicaments CIS in the group
func readGeneriquesFromTSV() (map[string][]int, error) {
	generiquesMap := make(map[string][]int)

	tsvFile, err := os.Open("files/Generiques.txt")
	if err != nil {
		return nil, fmt.Errorf("failed to open generiques file: %w", err)
	}
	defer func() {
		if err := tsvFile.Close(); err != nil {
			logging.Warn("Failed to close generiques TSV file", "error", err)
		}
	}()

	scanner := bufio.NewScanner(tsvFile)

	// Skip header line
	scanner.Scan()

	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Split(line, "\t")
		if len(fields) < 5 {
			continue
		}

		groupID := fields[0] // Group ID is in the 1st column (index 0)
		cisStr := fields[2]  // CIS is in the 3rd column (index 2)

		cis, err := strconv.Atoi(cisStr)
		if err != nil {
			continue // Skip invalid CIS values
		}

		if groupID != "" {
			generiquesMap[groupID] = append(generiquesMap[groupID], cis)
		}
	}

	return generiquesMap, nil
}

func GeneriquesParser(medicaments *[]entities.Medicament, mMap *map[int]entities.Medicament) ([]entities.GeneriqueList, map[int]entities.GeneriqueList, error) {

	// allGeneriques: []Generique
	allGeneriques, err := makeGeneriques(nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse generiques: %w", err)
	}

	// Create a map of libelles
	libelle := make(map[int]string)
	for el := range allGeneriques {
		groupID := allGeneriques[el].Group
		libelle[groupID] = allGeneriques[el].Libelle
	}

	// generiques file: [groupid]:[]cis of medicaments in the same group
	generiquesFile, err := readGeneriquesFromTSV()
	if err != nil {
		logging.Error("Failed to read generiques file", "error", err)
		return nil, nil, fmt.Errorf("failed to read generiques file: %w", err)
	}

	// The medsType is a map where the key are the medicament cis and the value is the
	// type of generique
	medsType, err := createMedicamentGeneriqueType()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create medicament generique type mapping: %w", err)
	}

	var generiques []entities.GeneriqueList
	generiquesMap := make(map[int]entities.GeneriqueList)

	for i, v := range generiquesFile {

		// Convert the string index to integer
		groupInt, convErr := strconv.Atoi(i)
		if convErr != nil {
			logging.Error("An error occurred converting the generiques group to integer", "error", convErr, "group_id", i)
			continue
		}

		medicaments, orphaned := getMedicamentsInArray(v, mMap, medsType)

		currentGenerique := entities.GeneriqueList{
			GroupID:           groupInt,
			Libelle:           libelle[groupInt],
			LibelleNormalized: strings.ReplaceAll(strings.ToLower(libelle[groupInt]), "+", " "),
			Medicaments:       medicaments,
			OrphanCIS:         orphaned,
		}

		generiques = append(generiques, currentGenerique)
		generiquesMap[groupInt] = currentGenerique
	}

	// Write debug
	fmt.Println("Generiques parsing completed", "count", len(generiques))
	return generiques, generiquesMap, nil
}

func createGeneriqueComposition(medicamentComposition *[]entities.Composition) []entities.GeneriqueComposition {
	var compositions []entities.GeneriqueComposition
	for _, v := range *medicamentComposition {
		compo := entities.GeneriqueComposition{
			ElementPharmaceutique: v.ElementPharmaceutique,
			DenominationSubstance: v.DenominationSubstance,
			Dosage:                v.Dosage,
		}
		compositions = append(compositions, compo)
	}
	return compositions
}

// getMedicamentsInArray splits CIS into two categories based on medicament availability:
// - Generiques with full medicament data (returns as GeneriqueMedicament with composition, type, etc.)
// - Orphan CIS without medicament data (returns as raw CIS integers)
//
// Parameters:
//   - medicamentsIds: Array of CIS values to process
//   - medicamentMap: Map of CIS to full Medicament entities for O(1) lookup
//   - medsType: Map of CIS to medicament type
//
// Returns:
//   - generiquesMedicaments: CIS that exist in medicamentMap with full data populated
//   - orphanCIS: CIS values that don't have corresponding medicament entries
func getMedicamentsInArray(medicamentsIds []int, medicamentMap *map[int]entities.Medicament, medsType map[int]string) (generiquesMedicaments []entities.GeneriqueMedicament, orphanCIS []int) {
	for _, v := range medicamentsIds {
		if medicament, ok := (*medicamentMap)[v]; ok {
			generiqueComposition := createGeneriqueComposition(&medicament.Composition)
			generiqueMed := entities.GeneriqueMedicament{
				Cis:                 medicament.Cis,
				Denomination:        medicament.Denomination,
				FormePharmaceutique: medicament.FormePharmaceutique,
				Type:                medsType[medicament.Cis],
				Composition:         generiqueComposition,
			}
			generiquesMedicaments = append(generiquesMedicaments, generiqueMed)
		} else {
			orphanCIS = append(orphanCIS, v)
		}
	}
	return
}
