package medicamentsparser

import (
	"bufio"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/giygas/medicaments-api/medicamentsparser/entities"
)

func makePresentations(wg *sync.WaitGroup) ([]entities.Presentation, error) {
	if wg != nil {
		defer wg.Done()
	}

	tsvFile, err := os.Open("files/Presentations.txt")
	if err != nil {
		return nil, fmt.Errorf("failed to open Presentations.txt: %w", err)
	}
	defer tsvFile.Close()

	scanner := bufio.NewScanner(tsvFile)

	var jsonRecords []entities.Presentation

	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Split(line, "\t")

		cis, err := strconv.Atoi(fields[0])
		if err != nil {
			return nil, fmt.Errorf("invalid cis value '%s': %w", fields[0], err)
		}

		cip7, err := strconv.Atoi(fields[1])
		if err != nil {
			return nil, fmt.Errorf("invalid cip7 value '%s': %w", fields[1], err)
		}

		cip13, err := strconv.Atoi(fields[6])
		if err != nil {
			return nil, fmt.Errorf("invalid cip13 value '%s': %w", fields[6], err)
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
				return nil, fmt.Errorf("invalid price value '%s': %w", fields[9], err)
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

	fmt.Println("Presentations file conversion completed", "records_count", len(jsonRecords))
	return jsonRecords, nil
}

func makeGeneriques(wg *sync.WaitGroup) ([]entities.Generique, error) {

	if wg != nil {
		defer wg.Done()
	}

	tsvFile, err := os.Open("files/Generiques.txt")
	if err != nil {
		return nil, fmt.Errorf("failed to open generiques file: %w", err)
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
			return nil, fmt.Errorf("invalid CIS value '%s' at line %d: %w", fields[2], lineCount, cisError)
		}

		group, groupErr := strconv.Atoi(fields[0])
		if groupErr != nil {
			return nil, fmt.Errorf("invalid group value '%s' at line %d: %w", fields[0], lineCount, groupErr)
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
		return nil, fmt.Errorf("error marshalling generiques: %w", err)
	}

	if wg != nil {
		_ = os.WriteFile("src/Generiques.json", jsonGeneriques, 0644)
	}

	fmt.Println("Generiques file conversion completed", "records_count", len(jsonRecords))
	return jsonRecords, nil
}

func makeCompositions(wg *sync.WaitGroup) ([]entities.Composition, error) {
	if wg != nil {
		defer wg.Done()
	}

	tsvFile, err := os.Open("files/Compositions.txt")
	if err != nil {
		return nil, fmt.Errorf("failed to open compositions file: %w", err)
	}
	defer tsvFile.Close()

	scanner := bufio.NewScanner(tsvFile)

	var jsonRecords []entities.Composition

	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Split(line, "\t")

		cis, err := strconv.Atoi(fields[0])
		if err != nil {
			return nil, fmt.Errorf("invalid cis value '%s': %w", fields[0], err)
		}

		codeS, err := strconv.Atoi(fields[2])
		if err != nil {
			return nil, fmt.Errorf("invalid codeSubstance value '%s': %w", fields[2], err)
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

	fmt.Println("Compositions file conversion completed", "records_count", len(jsonRecords))
	return jsonRecords, nil
}

func makeSpecialites(wg *sync.WaitGroup) ([]entities.Specialite, error) {
	if wg != nil {
		defer wg.Done()
	}

	tsvFile, err := os.Open("files/Specialites.txt")
	if err != nil {
		return nil, fmt.Errorf("failed to open Specialites.txt: %w", err)
	}
	defer tsvFile.Close()

	scanner := bufio.NewScanner(tsvFile)

	var jsonRecords []entities.Specialite

	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Split(line, "\t")

		cis, err := strconv.Atoi(fields[0])
		if err != nil {
			return nil, fmt.Errorf("invalid cis value '%s': %w", fields[0], err)
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

	fmt.Println("Specialites file conversion completed", "records_count", len(jsonRecords))
	return jsonRecords, nil
}

func makeConditions(wg *sync.WaitGroup) ([]entities.Condition, error) {
	if wg != nil {
		defer wg.Done()
	}

	tsvFile, err := os.Open("files/Conditions.txt")
	if err != nil {
		return nil, fmt.Errorf("failed to open Conditions.txt: %w", err)
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
			return nil, fmt.Errorf("invalid cis value '%s': %w", fields[0], err)
		}

		record := entities.Condition{
			Cis:       cis,
			Condition: fields[1],
		}

		jsonRecords = append(jsonRecords, record)
	}

	fmt.Println("Conditions file conversion completed", "records_count", len(jsonRecords))
	return jsonRecords, nil
}

// Creates a mapping where the key is the medicament cis and the value is the type of generique of the medicament
// Returns a map where key:cis and value:typeOfGenerique
func createMedicamentGeneriqueType() (map[int]string, error) {
	medsType := make(map[int]string)

	tsvFile, err := os.Open("files/Generiques.txt")
	if err != nil {
		return nil, fmt.Errorf("failed to open generique file: %w", err)
	}
	defer tsvFile.Close()

	scanner := bufio.NewScanner(tsvFile)

	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Split(line, "\t")

		cis, err := strconv.Atoi(fields[2])
		if err != nil {
			return nil, fmt.Errorf("invalid cis value in Generiques file '%s': %w", fields[2], err)
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

	return medsType, nil
}
