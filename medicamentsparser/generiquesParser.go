package medicamentsparser

import (
	"encoding/json"
	"log/slog"
	"os"
	"strconv"

	"github.com/giygas/medicamentsfr/medicamentsparser/entities"
)

var medsType map[int]string

func GeneriquesParser(medicaments *[]entities.Medicament, mMap *map[int]entities.Medicament) ([]entities.GeneriqueList, map[int]entities.Generique) {
	var err error

	// allGeneriques: []Generique
	allGeneriques := makeGeneriques(nil)

	// Create a map of all the generiques to reduce algorithm complexity
	generiquesMap := make(map[int]entities.Generique)
	for i := range allGeneriques {
		generiquesMap[allGeneriques[i].Group] = allGeneriques[i]
	}

	// generiques file: [groupid]:[]cis of medicaments in the same group
	generiquesFile, err := generiqueFileToJSON()
	if err != nil {
		slog.Error("Failed to read generiques file", "error", err)
		os.Exit(1)
	}

	// The medsType is a map where the key are the medicament cis and the value is the
	// type of generique
	medsType = createMedicamentGeneriqueType()

	var generiques []entities.GeneriqueList

	for i, v := range generiquesFile {

		// Convert the string index to integer
		groupInt, convErr := strconv.Atoi(i)
		if err != nil {
			slog.Error("An error occurred converting the generiques group to integer", "error", convErr, "group_id", i)
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
	marshalledGeneriques, err := json.MarshalIndent(generiques, "", " ")
	if err != nil {
		slog.Error("Error marshalling generiques", "error", err)
	} else {
		if writeErr := os.WriteFile("src/GeneriquesFull.json", marshalledGeneriques, 0644); writeErr != nil {
			slog.Error("Error writing GeneriquesFull.json", "error", writeErr)
		} else {
			slog.Info("GeneriquesFull.json created")
		}
	}

	slog.Info("Generiques parsing completed", "count", len(generiques))
	return generiques, generiquesMap
}

func createGeneriqueComposition(medicamentComposition *[]entities.Composition) []entities.GeneriqueComposition {
	var compositions []entities.GeneriqueComposition
	for _, v := range *medicamentComposition {
		compo := entities.GeneriqueComposition{
			ElementParmaceutique:  v.ElementParmaceutique,
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
