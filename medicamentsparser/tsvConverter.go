package medicamentsparser

import (
	"bufio"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/giygas/medicaments-api/logging"
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
	defer func() {
		if err := tsvFile.Close(); err != nil {
			logging.Warn("Failed to close presentations TSV file", "error", err)
		}
	}()

	scanner := bufio.NewScanner(tsvFile)

	var jsonRecords []entities.Presentation
	lineCount := 0
	skippedEmptyLines := 0
	skippedMissingColumns := 0
	skippedFormatErrors := 0

	for scanner.Scan() {
		lineCount++
		line := scanner.Text()

		// Skip empty lines silently
		if len(line) == 0 {
			skippedEmptyLines++
			continue
		}

		fields := strings.Split(line, "\t")

		// Check for missing columns
		if len(fields) < 10 {
			skippedMissingColumns++
			continue
		}

		cis, err := strconv.Atoi(fields[0])
		if err != nil {
			skippedFormatErrors++
			continue
		}

		cip7, err := strconv.Atoi(fields[1])
		if err != nil {
			skippedFormatErrors++
			continue
		}

		cip13, err := strconv.Atoi(fields[6])
		if err != nil {
			skippedFormatErrors++
			continue
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
			p, err := strconv.ParseFloat(strings.ReplaceAll(fields[9], ",", "."), 32)

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

	// Log skip statistics if any lines were skipped
	if skippedEmptyLines > 0 || skippedMissingColumns > 0 || skippedFormatErrors > 0 {
		logging.Info("Presentations.txt skip statistics",
			"empty_lines", skippedEmptyLines,
			"missing_columns", skippedMissingColumns,
			"format_errors", skippedFormatErrors,
			"total_lines", lineCount,
			"records_parsed", len(jsonRecords))
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
	defer func() {
		if err := tsvFile.Close(); err != nil {
			logging.Warn("Failed to close generiques TSV file", "error", err)
		}
	}()

	scanner := bufio.NewScanner(tsvFile)

	// Create the variables to use in the loop
	var jsonRecords []entities.Generique

	// Use a map for creating the generiques list
	generiquesList := make(map[int][]int)

	lineCount := 0
	skippedEmptyLines := 0
	skippedMissingColumns := 0
	skippedFormatErrors := 0

	for scanner.Scan() {
		lineCount++
		line := scanner.Text()

		// Skip empty lines silently
		if len(line) == 0 {
			skippedEmptyLines++
			continue
		}

		fields := strings.Split(line, "\t")

		// Check for missing columns (need at least 4 columns for required fields)
		// Note: TSV file has 5 columns total, but we only use 4 (fields[0-3])
		if len(fields) < 4 {
			skippedMissingColumns++
			continue
		}

		cis, cisError := strconv.Atoi(fields[2])
		if cisError != nil {
			skippedFormatErrors++
			continue
		}

		group, groupErr := strconv.Atoi(fields[0])
		if groupErr != nil {
			skippedFormatErrors++
			continue
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

	// Log skip statistics if any lines were skipped
	if skippedEmptyLines > 0 || skippedMissingColumns > 0 || skippedFormatErrors > 0 {
		logging.Info("Generiques.txt skip statistics",
			"empty_lines", skippedEmptyLines,
			"missing_columns", skippedMissingColumns,
			"format_errors", skippedFormatErrors,
			"total_lines", lineCount,
			"records_parsed", len(jsonRecords))
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
	defer func() {
		if err := tsvFile.Close(); err != nil {
			logging.Warn("Failed to close compositions TSV file", "error", err)
		}
	}()

	scanner := bufio.NewScanner(tsvFile)

	var jsonRecords []entities.Composition
	lineCount := 0
	skippedEmptyLines := 0
	skippedMissingColumns := 0
	skippedFormatErrors := 0

	for scanner.Scan() {
		lineCount++
		line := scanner.Text()

		// Skip empty lines silently
		if len(line) == 0 {
			skippedEmptyLines++
			continue
		}

		fields := strings.Split(line, "\t")

		// Check for missing columns (expected 7 columns)
		if len(fields) < 7 {
			skippedMissingColumns++
			continue
		}

		cis, err := strconv.Atoi(fields[0])
		if err != nil {
			skippedFormatErrors++
			continue
		}

		codeS, err := strconv.Atoi(fields[2])
		if err != nil {
			skippedFormatErrors++
			continue
		}

		record := entities.Composition{
			Cis:                   cis,
			ElementPharmaceutique: fields[1],
			CodeSubstance:         codeS,
			DenominationSubstance: fields[3],
			Dosage:                fields[4],
			ReferenceDosage:       fields[5],
			NatureComposant:       fields[6],
		}

		jsonRecords = append(jsonRecords, record)
	}

	// Log skip statistics if any lines were skipped
	if skippedEmptyLines > 0 || skippedMissingColumns > 0 || skippedFormatErrors > 0 {
		logging.Info("Compositions.txt skip statistics",
			"empty_lines", skippedEmptyLines,
			"missing_columns", skippedMissingColumns,
			"format_errors", skippedFormatErrors,
			"total_lines", lineCount,
			"records_parsed", len(jsonRecords))
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
	defer func() {
		if err := tsvFile.Close(); err != nil {
			logging.Warn("Failed to close specialites TSV file", "error", err)
		}
	}()

	scanner := bufio.NewScanner(tsvFile)

	var jsonRecords []entities.Specialite
	lineCount := 0
	skippedEmptyLines := 0
	skippedMissingColumns := 0
	skippedFormatErrors := 0

	for scanner.Scan() {
		lineCount++
		line := scanner.Text()

		// Skip empty lines silently
		if len(line) == 0 {
			skippedEmptyLines++
			continue
		}

		fields := strings.Split(line, "\t")

		// Check for missing columns (expected 12 columns)
		if len(fields) < 12 {
			skippedMissingColumns++
			continue
		}

		cis, err := strconv.Atoi(fields[0])
		if err != nil {
			skippedFormatErrors++
			continue
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

	// Log skip statistics if any lines were skipped
	if skippedEmptyLines > 0 || skippedMissingColumns > 0 || skippedFormatErrors > 0 {
		logging.Info("Specialites.txt skip statistics",
			"empty_lines", skippedEmptyLines,
			"missing_columns", skippedMissingColumns,
			"format_errors", skippedFormatErrors,
			"total_lines", lineCount,
			"records_parsed", len(jsonRecords))
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
	defer func() {
		if err := tsvFile.Close(); err != nil {
			logging.Warn("Failed to close conditions TSV file", "error", err)
		}
	}()

	scanner := bufio.NewScanner(tsvFile)

	var jsonRecords []entities.Condition
	lineCount := 0
	skippedEmptyLines := 0
	skippedMissingColumns := 0
	skippedFormatErrors := 0

	for scanner.Scan() {
		lineCount++
		line := scanner.Text()
		fields := strings.Split(line, "\t")

		// For some weird reason, the csv file from the site has some empty lines between the data
		// Empty lines are expected, so we don't log warnings for them
		if len(line) == 0 {
			skippedEmptyLines++
			continue
		}

		// Check for missing columns (expected 2 columns)
		if len(fields) < 2 {
			skippedMissingColumns++
			continue
		}

		cis, err := strconv.Atoi(fields[0])
		if err != nil {
			skippedFormatErrors++
			continue
		}

		record := entities.Condition{
			Cis:       cis,
			Condition: fields[1],
		}

		jsonRecords = append(jsonRecords, record)
	}

	// Log skip statistics if any lines were skipped (excluding expected empty lines)
	if skippedMissingColumns > 0 || skippedFormatErrors > 0 {
		logging.Info("Conditions.txt skip statistics",
			"empty_lines", skippedEmptyLines,
			"missing_columns", skippedMissingColumns,
			"format_errors", skippedFormatErrors,
			"total_lines", lineCount,
			"records_parsed", len(jsonRecords))
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
	defer func() {
		if err := tsvFile.Close(); err != nil {
			logging.Warn("Failed to close generique TSV file", "error", err)
		}
	}()

	scanner := bufio.NewScanner(tsvFile)

	lineCount := 0
	skippedEmptyLines := 0
	skippedMissingColumns := 0
	skippedFormatErrors := 0

	for scanner.Scan() {
		lineCount++
		line := scanner.Text()

		// Skip empty lines silently
		if len(line) == 0 {
			skippedEmptyLines++
			continue
		}

		fields := strings.Split(line, "\t")

		// Check for missing columns (expected 4 columns for type mapping)
		if len(fields) < 4 {
			skippedMissingColumns++
			continue
		}

		cis, err := strconv.Atoi(fields[2])
		if err != nil {
			skippedFormatErrors++
			continue
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
			generiqueType = "Générique substituable"
		}

		medsType[cis] = generiqueType
	}

	// Log skip statistics if any lines were skipped
	if skippedEmptyLines > 0 || skippedMissingColumns > 0 || skippedFormatErrors > 0 {
		logging.Info("Generiques.txt type mapping skip statistics",
			"empty_lines", skippedEmptyLines,
			"missing_columns", skippedMissingColumns,
			"format_errors", skippedFormatErrors,
			"total_lines", lineCount,
			"records_parsed", len(medsType))
	}

	return medsType, nil
}
