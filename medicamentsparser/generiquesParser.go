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

var medsType map[int]string

// readGeneriquesFromTSV reads generiques data directly from TSV file
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

func GeneriquesParser(medicaments *[]entities.Medicament, mMap *map[int]entities.Medicament) ([]entities.GeneriqueList, map[int]entities.Generique, error) {

	var err error

	// allGeneriques: []Generique
	allGeneriques, err := makeGeneriques(nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse generiques: %w", err)
	}

	// Create a map of all the generiques to reduce algorithm complexity
	// We need to preserve the libelle for each group, not overwrite it
	generiquesMap := make(map[int]entities.Generique)
	for i := range allGeneriques {
		group := allGeneriques[i].Group
		// Only set if not already present to preserve the first (correct) libelle
		if _, exists := generiquesMap[group]; !exists {
			generiquesMap[group] = allGeneriques[i]
		}
	}

	// generiques file: [groupid]:[]cis of medicaments in the same group
	generiquesFile, err := readGeneriquesFromTSV()
	if err != nil {
		logging.Error("Failed to read generiques file", "error", err)
		return nil, nil, fmt.Errorf("failed to read generiques file: %w", err)
	}

	// The medsType is a map where the key are the medicament cis and the value is the
	// type of generique
	medsType, err = createMedicamentGeneriqueType()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create medicament generique type mapping: %w", err)
	}

	var generiques []entities.GeneriqueList

	for i, v := range generiquesFile {

		// Convert the string index to integer
		groupInt, convErr := strconv.Atoi(i)
		if convErr != nil {
			logging.Error("An error occurred converting the generiques group to integer", "error", convErr, "group_id", i)
			continue
		}

		current := entities.GeneriqueList{
			GroupID:     groupInt,
			Libelle:     generiquesMap[groupInt].Libelle,
			Medicaments: getMedicamentsInArray(v, mMap),
		}

		generiques = append(generiques, current)
	}

	// Write debug file

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

func getMedicamentsInArray(medicamentsIds []int, medicamentMap *map[int]entities.Medicament) []entities.GeneriqueMedicament {
	var medicamentsArray []entities.GeneriqueMedicament

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
			medicamentsArray = append(medicamentsArray, generiqueMed)
		}
	}

	return medicamentsArray
}
