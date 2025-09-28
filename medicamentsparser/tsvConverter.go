package medicamentsparser

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/giygas/medicamentsfr/medicamentsparser/entities"
)

func makePresentations(wg *sync.WaitGroup) []entities.Presentation {
	if wg != nil {
		defer wg.Done()
	}

	tsvFile, err := os.Open("files/Presentations.txt")
	if err != nil {
		fmt.Println(fmt.Errorf("error opening Presentations.txt: %w", err))
	}
	defer tsvFile.Close()

	scanner := bufio.NewScanner(tsvFile)

	var jsonRecords []entities.Presentation

	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Split(line, "\t")

		cis, err := strconv.Atoi(fields[0])
		if err != nil {
			fmt.Println(fmt.Errorf("error converting cis in Presentations file: %w", err))
		}

		cip7, err := strconv.Atoi(fields[1])
		if err != nil {
			fmt.Println(fmt.Errorf("error converting cip7 in Presentations file: %w", err))
		}

		cip13, err := strconv.Atoi(fields[6])
		if err != nil {
			fmt.Println(fmt.Errorf("error converting cip13 in Presentations file: %w", err))
		}

		// Because the downloaded database has commas as thousands and decimal separators,
		// all the commas have to be removed except for the last one
		// If the prix is empty, 0.0 will we added in the prix section
		var prix float32

		if fields[9] != "" {

			// Count the number of commas
			numCommas := strings.Count(fields[9], ",")

			// If there's more than one comma, replace all but the last one
			if numCommas > 1 {
				fields[9] = strings.Replace(fields[9], ",", "", numCommas-1)
			}

			// Replace the last comma with a period
			p, err := strconv.ParseFloat(strings.Replace(fields[9], ",", ".", -1), 32)

			if err != nil {
				fmt.Println(fmt.Errorf("error parsing price in Presentations file: %w", err))
			}
			p = math.Trunc(p*100) / 100

			prix = float32(p)
		} else {
			prix = 0.0
		}

		record := entities.Presentation{
			Cis:                  cis,
			Cip7:                 cip7,
			Libelle:              fields[2],
			StatusAdministratif:  fields[3],
			EtatComercialisation: fields[4],
			DateDeclaration:      fields[5],
			Cip13:                cip13,
			Agreement:            fields[7],
			TauxRemboursement:    fields[8],
			Prix:                 prix,
		}

		jsonRecords = append(jsonRecords, record)
	}

	fmt.Println("Presentations done")
	return jsonRecords
}

func makeGeneriques(wg *sync.WaitGroup) []entities.Generique {
	if wg != nil {
		defer wg.Done()
	} else {
		fmt.Println("Second creation of generiques for the mapping of the medicaments")
	}

	tsvFile, err := os.Open("files/Generiques.txt")
	if err != nil {
		fmt.Println(fmt.Errorf("error opening Generiques.txt: %w", err))
	}
	defer tsvFile.Close()

	scanner := bufio.NewScanner(tsvFile)

	// Create the variables to use in the loop
	var jsonRecords []entities.Generique

	// Use a map for creating the generiques list
	generiquesList := make(map[int][]int)

	lineCount := 0
	for scanner.Scan() {
		lineCount++
		line := scanner.Text()
		fields := strings.Split(line, "\t")

		if len(fields) < 4 {
			continue
		}

		cis, cisError := strconv.Atoi(fields[2])
		if cisError != nil {
			slog.Error("Error converting to int cis in generiques file", "cis", fields[2], "error", cisError)
			os.Exit(1)
		}

		group, groupErr := strconv.Atoi(fields[0])
		if groupErr != nil {
			slog.Error("Error converting to int group in generiques file", "group", fields[0], "error", groupErr)
			os.Exit(1)
		}

		var generiqueType string

		switch fields[3] {
		case "0":
			generiqueType = "Princeps"
		case "1":
			generiqueType = "Générique"
		case "2":
			generiqueType = "Génériques par complémentarité posologique"
		case "3":
			generiqueType = "Générique substitutable"
		}

		record := entities.Generique{
			Cis:     cis,
			Group:   group,
			Libelle: fields[1],
			Type:    generiqueType,
		}

		jsonRecords = append(jsonRecords, record)

		// Append to the array of generiques
		if cis != 0 {
			generiquesList[group] = append(generiquesList[group], cis)
		}
	}

	jsonGeneriques, err := json.MarshalIndent(generiquesList, "", "  ")
	if err != nil {
		fmt.Println(fmt.Errorf("error marshalling generiques: %w", err))
	}
	if wg != nil {
		fmt.Println("Generiques done")
		_ = os.WriteFile("src/Generiques.json", jsonGeneriques, 0644)
	}

	return jsonRecords
}

func makeCompositions(wg *sync.WaitGroup) []entities.Composition {
	if wg != nil {
		defer wg.Done()
	}

	tsvFile, err := os.Open("files/Compositions.txt")
	if err != nil {
		fmt.Println(fmt.Errorf("error opening Compositions.txt: %w", err))
	}
	defer tsvFile.Close()

	scanner := bufio.NewScanner(tsvFile)

	var jsonRecords []entities.Composition

	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Split(line, "\t")

		cis, err := strconv.Atoi(fields[0])
		if err != nil {
			fmt.Println(fmt.Errorf("error converting cis in Compositions file: %w", err))
		}

		codeS, err := strconv.Atoi(fields[2])
		if err != nil {
			fmt.Println(fmt.Errorf("error converting codeSubstance in Compositions file: %w", err))
		}

		record := entities.Composition{
			Cis:                   cis,
			ElementParmaceutique:  fields[1],
			CodeSubstance:         codeS,
			DenominationSubstance: fields[3],
			Dosage:                fields[4],
			ReferenceDosage:       fields[5],
			NatureComposant:       fields[6],
		}

		jsonRecords = append(jsonRecords, record)
	}

	fmt.Println("Compositions done")
	return jsonRecords
}

func makeSpecialites(wg *sync.WaitGroup) []entities.Specialite {
	if wg != nil {
		defer wg.Done()
	}

	tsvFile, err := os.Open("files/Specialites.txt")
	if err != nil {
		fmt.Println(fmt.Errorf("error opening Specialites.txt: %w", err))
	}
	defer tsvFile.Close()

	scanner := bufio.NewScanner(tsvFile)

	var jsonRecords []entities.Specialite

	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Split(line, "\t")

		cis, err := strconv.Atoi(fields[0])
		if err != nil {
			fmt.Println(fmt.Errorf("error converting cis in Specialites file: %w", err))
		}

		record := entities.Specialite{
			Cis:                   cis,
			Denomination:          fields[1],
			FormePharmaceutique:   fields[2],
			VoiesAdministration:   strings.Split(fields[3], ";"),
			StatusAutorisation:    fields[4],
			TypeProcedure:         fields[5],
			EtatComercialisation:  fields[6],
			DateAMM:               fields[7],
			Titulaire:             strings.TrimLeft(fields[10], " "),
			SurveillanceRenforcee: fields[11],
		}

		jsonRecords = append(jsonRecords, record)
	}

	fmt.Println("Specialites done")
	return jsonRecords
}

func makeConditions(wg *sync.WaitGroup) []entities.Condition {
	if wg != nil {
		defer wg.Done()
	}

	tsvFile, err := os.Open("files/Conditions.txt")
	if err != nil {
		fmt.Println(fmt.Errorf("error opening Conditions.txt: %w", err))
	}
	defer tsvFile.Close()

	scanner := bufio.NewScanner(tsvFile)

	var jsonRecords []entities.Condition

	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Split(line, "\t")

		// For some weird reason, the csv file from the site has some empty lines between the data
		if len(line) == 0 {
			continue
		}

		cis, err := strconv.Atoi(fields[0])
		if err != nil {
			fmt.Println(fmt.Errorf("error converting cis in Conditions file: %w", err))
		}

		record := entities.Condition{
			Cis:       cis,
			Condition: fields[1],
		}

		jsonRecords = append(jsonRecords, record)
	}

	fmt.Println("Conditions done")
	return jsonRecords
}

// Creates a mapping where the key is the medicament cis and the value is the type of generique of the medicament
// Returns a map where key:cis and value:typeOfGenerique
func createMedicamentGeneriqueType() map[int]string {
	medsType := make(map[int]string)

	tsvFile, err := os.Open("files/Generiques.txt")
	if err != nil {
		fmt.Println(fmt.Errorf("error opening Generiques.txt: %w", err))
	}
	defer tsvFile.Close()

	scanner := bufio.NewScanner(tsvFile)

	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Split(line, "\t")

		cis, err := strconv.Atoi(fields[2])
		if err != nil {
			fmt.Println(fmt.Errorf("error converting cis in Generiques file: %w", err))
		}

		var generiqueType string

		switch fields[3] {
		case "0":
			generiqueType = "Princeps"
		case "1":
			generiqueType = "Générique"
		case "2":
			generiqueType = "Génériques par complémentarité posologique"
		case "3":
			generiqueType = "Générique substitutable"
		}

		medsType[cis] = generiqueType
	}

	return medsType
}
