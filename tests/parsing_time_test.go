package main

import (
	"fmt"
	"testing"
	"time"

	"github.com/giygas/medicaments-api/medicamentsparser"
	"github.com/giygas/medicaments-api/medicamentsparser/entities"
)

func TestParsingTime(t *testing.T) {
	start := time.Now()

	medicaments, err := medicamentsparser.ParseAllMedicaments()
	if err != nil {
		t.Fatalf("Failed to parse medicaments: %v", err)
	}

	medicamentsMap := make(map[int]entities.Medicament)
	for i := range medicaments {
		medicamentsMap[medicaments[i].Cis] = medicaments[i]
	}

	_, _, err = medicamentsparser.GeneriquesParser(&medicaments, &medicamentsMap)
	if err != nil {
		t.Fatalf("Failed to parse generiques: %v", err)
	}

	duration := time.Since(start)
	fmt.Printf("Parsing time: %.2f seconds\n", duration.Seconds())

	// Check if it meets the claimed 0.5 seconds
	if duration.Seconds() > 0.75 { // 50% tolerance
		t.Errorf("Parsing took too long: %.2f seconds (claimed: 0.5 seconds)", duration.Seconds())
	}
}
