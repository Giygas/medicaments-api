package medicamentsparser

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/giygas/medicaments-api/medicamentsparser/entities"
)

func specialitesFileToJSON() ([]entities.Specialite, error) {
	fileData, err := os.ReadFile("src/Specialites.json")
	if err != nil {
		return nil, fmt.Errorf("error reading Specialites.json: %w", err)
	}

	var specialites []entities.Specialite
	err = json.Unmarshal(fileData, &specialites)
	if err != nil {
		return nil, fmt.Errorf("error decoding Specialites JSON: %w", err)
	}

	//Returns the full Specialites json as a slice of Specialites
	return specialites, nil
}

func compositionFileToJSON() ([]entities.Composition, error) {
	fileData, err := os.ReadFile("src/Compositions.json")
	if err != nil {
		return nil, fmt.Errorf("error reading Compositions.json: %w", err)
	}

	var data []entities.Composition
	err = json.Unmarshal(fileData, &data)
	if err != nil {
		return nil, fmt.Errorf("error decoding Compositions JSON: %w", err)
	}

	//Returns the full Composition json as a slice of Composition
	return data, nil
}

func conditionFileToJSON() ([]entities.Condition, error) {
	fileData, err := os.ReadFile("src/Conditions.json")
	if err != nil {
		return nil, fmt.Errorf("error reading Conditions.json: %w", err)
	}

	var data []entities.Condition
	err = json.Unmarshal(fileData, &data)
	if err != nil {
		return nil, fmt.Errorf("error decoding Conditions JSON: %w", err)
	}

	//Returns the full Condition json as a slice of Condition
	return data, nil
}

func generiqueFileToJSON() (map[string][]int, error) {
	fileData, err := os.ReadFile("src/Generiques.json")
	if err != nil {
		return nil, fmt.Errorf("error reading Generiques.json: %w", err)
	}

	var data map[string][]int

	err = json.Unmarshal(fileData, &data)
	if err != nil {
		return nil, fmt.Errorf("error decoding Generiques JSON: %w", err)
	}

	//Returns the full Generique json as a slice of Generique
	return data, nil
}

func presentationFileToJSON() ([]entities.Presentation, error) {
	fileData, err := os.ReadFile("src/Presentations.json")
	if err != nil {
		return nil, fmt.Errorf("error reading Presentations.json: %w", err)
	}

	var data []entities.Presentation
	err = json.Unmarshal(fileData, &data)
	if err != nil {
		return nil, fmt.Errorf("error decoding Presentations JSON: %w", err)
	}

	//Returns the full Presentation json as a slice of Presentation
	return data, nil
}
