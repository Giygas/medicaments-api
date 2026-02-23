package handlers

import (
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/giygas/medicaments-api/medicamentsparser/entities"
)

// ============================================================================
// BENCHMARKS
// ============================================================================

// ============================================================================
// V1 ENDPOINT BENCHMARKS
// ============================================================================

// BenchmarkMedicamentsExport benchmarks export all endpoint (v1)
func BenchmarkMedicamentsExport(b *testing.B) {
	factory := NewTestDataFactory()
	medicaments := make([]entities.Medicament, 1000)
	for i := range 1000 {
		medicaments[i] = factory.CreateMedicament(i, fmt.Sprintf("Test Med %d", i))
	}

	mockStore := NewMockDataStoreBuilder().WithMedicaments(medicaments).Build()
	mockValidator := NewMockDataValidatorBuilder().Build()
	handler := NewHTTPHandler(mockStore, mockValidator, NewMockHealthCheckerBuilder().Build())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/v1/medicaments/export", nil)
		handler.ExportMedicaments(rr, req)
	}
}

// BenchmarkMedicamentsPagination benchmarks pagination endpoint (v1)
func BenchmarkMedicamentsPagination(b *testing.B) {
	factory := NewTestDataFactory()
	medicaments := make([]entities.Medicament, 10)
	for i := range 10 {
		medicaments[i] = factory.CreateMedicament(i, fmt.Sprintf("Test Med %d", i))
	}

	mockStore := NewMockDataStoreBuilder().WithMedicaments(medicaments).Build()
	mockValidator := NewMockDataValidatorBuilder().Build()
	handler := NewHTTPHandler(mockStore, mockValidator, NewMockHealthCheckerBuilder().Build())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/v1/medicaments?page=1", nil)
		handler.ServeMedicamentsV1(rr, req)
	}
}

// BenchmarkMedicamentsSearch benchmarks search endpoint (v1)
func BenchmarkMedicamentsSearch(b *testing.B) {
	mockStore := NewMockDataStoreBuilder().WithMedicaments(NewTestDataFactory().CreateMedicaments(1000)).Build()
	mockValidator := NewMockDataValidatorBuilder().Build()
	handler := NewHTTPHandler(mockStore, mockValidator, NewMockHealthCheckerBuilder().Build())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/v1/medicaments?search=paracetamol", nil)
		handler.ServeMedicamentsV1(rr, req)
	}
}

// BenchmarkMedicamentByCIS benchmarks CIS lookup endpoint (v1)
func BenchmarkMedicamentByCIS(b *testing.B) {
	medicaments := NewTestDataFactory().CreateMedicaments(1000)
	mockStore := NewMockDataStoreBuilder().WithMedicaments(medicaments).Build()
	mockValidator := NewMockDataValidatorBuilder().Build()
	handler := NewHTTPHandler(mockStore, mockValidator, NewMockHealthCheckerBuilder().Build())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/v1/medicaments?cis=1", nil)
		handler.ServeMedicamentsV1(rr, req)
	}
}

// BenchmarkMedicamentByCIP benchmarks CIP lookup endpoint (v1)
func BenchmarkMedicamentByCIP(b *testing.B) {
	factory := NewTestDataFactory()
	medicaments := make([]entities.Medicament, 1000)
	for i := range 1000 {
		medicaments[i] = factory.CreateMedicament(i, "PARACETAMOL 500 mg")
	}

	presentation := entities.Presentation{
		Cis:     1,
		Cip7:    1234567,
		Cip13:   1234567890123,
		Libelle: "Boîte de 8 comprimés",
	}

	presentationsCIP7Map := map[int]entities.Presentation{1234567: presentation}

	mockStore := NewMockDataStoreBuilder().
		WithMedicaments(medicaments).
		WithPresentationsCIP7Map(presentationsCIP7Map).
		Build()
	mockValidator := NewMockDataValidatorBuilder().Build()
	handler := NewHTTPHandler(mockStore, mockValidator, NewMockHealthCheckerBuilder().Build())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/v1/medicaments?cip=1234567", nil)
		handler.ServeMedicamentsV1(rr, req)
	}
}

// BenchmarkGeneriquesSearch benchmarks libelle search endpoint (v1)
func BenchmarkGeneriquesSearch(b *testing.B) {
	genericList := []entities.GeneriqueList{
		{
			GroupID: 1,
			Libelle: "PARACETAMOL 500 mg",
			Medicaments: []entities.GeneriqueMedicament{
				{Cis: 1, Denomination: "PARACETAMOL BIOGARAN"},
				{Cis: 2, Denomination: "DAFALGAN"},
			},
		},
	}

	generiquesMap := map[int]entities.GeneriqueList{
		1: {GroupID: 1, Libelle: "PARACETAMOL 500 mg"},
	}

	mockStore := NewMockDataStoreBuilder().
		WithGeneriques(genericList).
		WithGeneriquesMap(generiquesMap).
		Build()
	mockValidator := NewMockDataValidatorBuilder().Build()
	handler := NewHTTPHandler(mockStore, mockValidator, NewMockHealthCheckerBuilder().Build())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/v1/generiques?libelle=paracetamol", nil)
		handler.ServeGeneriquesV1(rr, req)
	}
}

// BenchmarkGeneriqueByGroup benchmarks group lookup endpoint (v1)
func BenchmarkGeneriqueByGroup(b *testing.B) {
	genericList := []entities.GeneriqueList{
		{
			GroupID: 1,
			Libelle: "PARACETAMOL 500 mg",
			Medicaments: []entities.GeneriqueMedicament{
				{Cis: 1, Denomination: "PARACETAMOL BIOGARAN"},
				{Cis: 2, Denomination: "DAFALGAN"},
			},
		},
	}

	generiquesMap := map[int]entities.GeneriqueList{
		1: {GroupID: 1, Libelle: "PARACETAMOL 500 mg"},
	}

	mockStore := NewMockDataStoreBuilder().
		WithGeneriques(genericList).
		WithGeneriquesMap(generiquesMap).
		Build()
	mockValidator := NewMockDataValidatorBuilder().Build()
	handler := NewHTTPHandler(mockStore, mockValidator, NewMockHealthCheckerBuilder().Build())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/v1/generiques?group=1", nil)
		handler.ServeGeneriquesV1(rr, req)
	}
}

// BenchmarkPresentationByCIP benchmarks presentation lookup endpoint (v1)
func BenchmarkPresentationByCIP(b *testing.B) {
	presentation := entities.Presentation{
		Cis:                  1,
		Cip7:                 1234567,
		Cip13:                1234567890123,
		Libelle:              "Boîte de 8 comprimés",
		StatusAdministratif:  "Présentation commercialisée",
		EtatComercialisation: "Commercialisée",
		DateDeclaration:      "2020-02-01",
	}

	presentationsCIP7Map := map[int]entities.Presentation{1234567: presentation}

	mockStore := NewMockDataStoreBuilder().
		WithPresentationsCIP7Map(presentationsCIP7Map).
		Build()
	mockValidator := NewMockDataValidatorBuilder().Build()
	handler := NewHTTPHandler(mockStore, mockValidator, NewMockHealthCheckerBuilder().Build())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/v1/presentations?cip=1234567", nil)
		handler.ServePresentationsV1(rr, req)
	}
}
