package main

import (
	"fmt"
	"testing"

	"github.com/giygas/medicaments-api/logging"
	"github.com/giygas/medicaments-api/medicamentsparser"
	"github.com/giygas/medicaments-api/medicamentsparser/entities"
)

// TestIntegrationCrossFileConsistency verifies data integrity across all 5 TSV files
// This is critical because:
// - Medicaments reference Presentations via CIS
// - Medicaments reference Generiques via group IDs
// - Medicaments reference Compositions via CIS
// - Medicaments reference Conditions via CIS
// - Any inconsistencies could cause runtime panics or incorrect API responses
func TestIntegrationCrossFileConsistency(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	fmt.Println("Starting cross-file consistency integration test...")

	// Initialize logging
	logging.InitLogger("logs")

	// Parse all TSV files
	medicaments, presentationsCIP7Map, presentationsCIP13Map, err := medicamentsparser.ParseAllMedicaments()
	if err != nil {
		t.Fatalf("Failed to parse medicaments: %v", err)
	}

	fmt.Printf("Parsed %d medicaments\n", len(medicaments))
	fmt.Printf("CIP7 map has %d entries\n", len(presentationsCIP7Map))
	fmt.Printf("CIP13 map has %d entries\n", len(presentationsCIP13Map))

	// Create medicaments map for lookup
	medicamentsMap := make(map[int]entities.Medicament)
	for i := range medicaments {
		medicamentsMap[medicaments[i].Cis] = medicaments[i]
	}

	// Parse generiques
	_, generiquesMap, err := medicamentsparser.GeneriquesParser(&medicaments, &medicamentsMap)
	if err != nil {
		t.Fatalf("Failed to parse generiques: %v", err)
	}

	fmt.Printf("Parsed %d generique groups\n", len(generiquesMap))

	// ============================
	// Test 1: Verify Presentation Consistency
	// ============================
	fmt.Println("\n--- Presentation Consistency Check ---")

	orphanedPresentations := 0
	presentationsWithoutMedicament := 0
	cisWithoutPresentations := 0

	// Check all presentations have valid medicaments
	for _, med := range medicaments {
		if len(med.Presentation) > 0 {
			for _, pres := range med.Presentation {
				// Verify CIP7 is in map
				if _, exists := presentationsCIP7Map[pres.Cip7]; !exists {
					t.Errorf("Presentation CIP7 %d not found in CIP7 map (CIS %d)", pres.Cip7, med.Cis)
					orphanedPresentations++
				}

				// Verify CIP13 is in map
				if _, exists := presentationsCIP13Map[pres.Cip13]; !exists {
					t.Errorf("Presentation CIP13 %d not found in CIP13 map (CIS %d)", pres.Cip13, med.Cis)
					orphanedPresentations++
				}
			}
		} else {
			cisWithoutPresentations++
		}
	}

	// Check all CIP7 maps reference valid medicaments
	for cip7, pres := range presentationsCIP7Map {
		if _, exists := medicamentsMap[pres.Cis]; !exists {
			t.Logf("CIP7 map entry references non-existent CIS %d (CIP7 %d)", pres.Cis, cip7)
			presentationsWithoutMedicament++
		}
	}

	// Check all CIP13 maps reference valid medicaments
	for cip13, pres := range presentationsCIP13Map {
		if _, exists := medicamentsMap[pres.Cis]; !exists {
			t.Logf("CIP13 map entry references non-existent CIS %d (CIP13 %d)", pres.Cis, cip13)
			presentationsWithoutMedicament++
		}
	}

	// Allow some orphaned records (BDPM data quality issues) but alert if excessive
	maxAllowedOrphans := 100 // Threshold for data quality
	if presentationsWithoutMedicament > maxAllowedOrphans {
		t.Errorf("Too many orphaned presentations: %d (threshold: %d)",
			presentationsWithoutMedicament, maxAllowedOrphans)
	}

	fmt.Printf("Presentations consistency check:\n")
	fmt.Printf("  - Medicaments with presentations: %d\n", len(medicaments)-cisWithoutPresentations)
	fmt.Printf("  - Medicaments without presentations: %d\n", cisWithoutPresentations)
	fmt.Printf("  - Presentations with orphaned CIP7/CIP13: %d\n", orphanedPresentations)
	fmt.Printf("  - Presentations referencing non-existent CIS: %d\n", presentationsWithoutMedicament)

	// ============================
	// Test 2: Verify Generique Consistency
	// ============================
	fmt.Println("\n--- Generique Consistency Check ---")

	orphanedGeneriqueCIS := 0
	generiquesWithMedicaments := 0
	totalOrphanCIS := 0

	// With GeneriqueList, we iterate over Medicaments slice within each group
	// and validate that all referenced CIS values exist in medicaments map.
	// This correctly tests data integrity across files.
	for _, genList := range generiquesMap {
		for _, genMed := range genList.Medicaments {
			if _, exists := medicamentsMap[genMed.Cis]; !exists {
				t.Errorf("Generique for CIS %d not found in medicaments map", genMed.Cis)
				orphanedGeneriqueCIS++
				generiquesWithMedicaments--
			} else {
				generiquesWithMedicaments++
			}
		}
		// Count orphan CIS but don't treat them as errors - they're expected to not exist
		totalOrphanCIS += len(genList.OrphanCIS)
	}

	fmt.Printf("Generiques consistency check:\n")
	fmt.Printf("  - Generique groups: %d\n", len(generiquesMap))
	fmt.Printf("  - Generique groups with medicaments: %d\n", generiquesWithMedicaments)
	fmt.Printf("  - Orphaned generique CIS (expected): %d\n", totalOrphanCIS)
	fmt.Printf("  - Invalid generique CIS references: %d\n", orphanedGeneriqueCIS)

	// ============================
	// Test 3: Verify Composition Consistency
	// ============================
	fmt.Println("\n--- Composition Consistency Check ---")

	orphanedCompositionCIS := 0
	medicamentsWithoutCompositions := 0

	for _, med := range medicaments {
		if len(med.Composition) > 0 {
			for _, comp := range med.Composition {
				// Verify composition references valid CIS
				if _, exists := medicamentsMap[comp.Cis]; !exists {
					t.Errorf("Composition references non-existent CIS %d", comp.Cis)
					orphanedCompositionCIS++
				}
			}
		} else {
			medicamentsWithoutCompositions++
		}
	}

	fmt.Printf("Composition consistency check:\n")
	fmt.Printf("  - Medicaments with compositions: %d\n", len(medicaments)-medicamentsWithoutCompositions)
	fmt.Printf("  - Medicaments without compositions: %d\n", medicamentsWithoutCompositions)
	fmt.Printf("  - Orphaned composition references: %d\n", orphanedCompositionCIS)

	// ============================
	// Test 4: Verify Condition Consistency
	// ============================
	fmt.Println("\n--- Condition Consistency Check ---")

	orphanedConditionCIS := 0
	medicamentsWithoutConditions := 0

	for _, med := range medicaments {
		if len(med.Conditions) > 0 {
			// Conditions are strings, just check they're associated with valid medicaments
			// No need to verify CIS since conditions don't reference medicaments
		} else {
			medicamentsWithoutConditions++
		}
	}

	fmt.Printf("Condition consistency check:\n")
	fmt.Printf("  - Medicaments with conditions: %d\n", len(medicaments)-medicamentsWithoutConditions)
	fmt.Printf("  - Medicaments without conditions: %d\n", medicamentsWithoutConditions)
	fmt.Printf("  - Orphaned condition references: %d\n", orphanedConditionCIS)

	// ============================
	// Overall Consistency Summary
	// ============================
	// Calculate total orphan CIS from generiques
	totalOrphanCISFromGeneriques := 0
	for _, genList := range generiquesMap {
		totalOrphanCISFromGeneriques += len(genList.OrphanCIS)
	}

	// Only count true data inconsistencies (not expected orphans)
	totalOrphanedRecords := orphanedPresentations + orphanedGeneriqueCIS +
		orphanedCompositionCIS + orphanedConditionCIS

	fmt.Println("\n=== Overall Consistency Summary ===")
	fmt.Printf("  Total medicaments: %d\n", len(medicaments))
	fmt.Printf("  Medicaments with presentations: %d (%.1f%%)\n", len(medicaments)-cisWithoutPresentations, float64(len(medicaments)-cisWithoutPresentations)*100/float64(len(medicaments)))
	fmt.Printf("  Medicaments with generiques: %d (%.1f%%)\n", generiquesWithMedicaments, float64(generiquesWithMedicaments)*100/float64(len(medicaments)))
	fmt.Printf("  Medicaments with compositions: %d (%.1f%%)\n", len(medicaments)-medicamentsWithoutCompositions, float64(len(medicaments)-medicamentsWithoutCompositions)*100/float64(len(medicaments)))
	fmt.Printf("  Medicaments with conditions: %d (%.1f%%)\n", len(medicaments)-medicamentsWithoutConditions, float64(len(medicaments)-medicamentsWithoutConditions)*100/float64(len(medicaments)))
	fmt.Printf("  Orphan generique CIS (expected): %d\n", totalOrphanCISFromGeneriques)
	fmt.Printf("  Presentations referencing non-existent CIS: %d\n", presentationsWithoutMedicament)
	fmt.Printf("  Data inconsistency errors: %d\n", totalOrphanedRecords)

	// Note: We don't assert failure on totalOrphanedRecords because orphaned records
	// are expected in real-world BDPM data. All individual issues are already
	// logged above via t.Errorf() for visibility.

	// Verify maps are properly populated
	if len(presentationsCIP7Map) == 0 {
		t.Error("CIP7 presentation map should not be empty")
	}

	if len(presentationsCIP13Map) == 0 {
		t.Error("CIP13 presentation map should not be empty")
	}

	if len(generiquesMap) == 0 {
		t.Error("Generiques map should not be empty")
	}

	fmt.Println("\nCross-file consistency check completed")
}
