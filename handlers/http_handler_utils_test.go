package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/giygas/medicaments-api/data"
	"github.com/giygas/medicaments-api/interfaces"
	"github.com/giygas/medicaments-api/medicamentsparser/entities"
	"github.com/go-chi/chi/v5"
)

// ============================================================================
// TEST DATA FACTORY
// ============================================================================

// TestDataFactory creates consistent test data across all tests
type TestDataFactory struct{}

func NewTestDataFactory() *TestDataFactory {
	return &TestDataFactory{}
}

// CreateMedicament creates a single test medicament with full realistic data
func (f *TestDataFactory) CreateMedicament(cis int, denomination string) entities.Medicament {
	return entities.Medicament{
		Cis:                    cis,
		Denomination:           denomination,
		DenominationNormalized: strings.ReplaceAll(strings.ToLower(denomination), "+", " "),
		FormePharmaceutique:    "Comprimé pelliculé",
		VoiesAdministration:    []string{"Orale"},
		StatusAutorisation:     "Autorisé",
		TypeProcedure:          "Procédure nationale",
		EtatComercialisation:   "Commercialisé",
		DateAMM:                "2020-01-15",
		Titulaire:              "Laboratoire Test",
		SurveillanceRenforcee:  "Non",
		Composition: []entities.Composition{
			{
				Cis:                   cis,
				CodeSubstance:         1001,
				DenominationSubstance: "Test Substance",
				Dosage:                "500 mg",
				ReferenceDosage:       "mg",
				NatureComposant:       "Actif",
			},
		},
		Generiques: []entities.Generique{},
		Presentation: []entities.Presentation{
			{
				Cis:                  cis,
				Cip7:                 1234567,
				Libelle:              "Boîte de 20 comprimés",
				StatusAdministratif:  "Présentation commercialisée",
				EtatComercialisation: "Commercialisée",
				DateDeclaration:      "2020-02-01",
			},
		},
		Conditions: []string{},
	}
}

// CreateMedicaments creates multiple test medicaments
func (f *TestDataFactory) CreateMedicaments(count int) []entities.Medicament {
	medicaments := make([]entities.Medicament, count)
	for i := 0; i < count; i++ {
		medicaments[i] = f.CreateMedicament(i+1, fmt.Sprintf("Médicament Test %d", i+1))
	}
	return medicaments
}

// CreateMedicamentsMap creates a medicaments map for O(1) lookups
func (f *TestDataFactory) CreateMedicamentsMap(medicaments []entities.Medicament) map[int]entities.Medicament {
	medicamentsMap := make(map[int]entities.Medicament)
	for _, med := range medicaments {
		medicamentsMap[med.Cis] = med
	}
	return medicamentsMap
}

// CreateGeneriqueList creates a test generique list
func (f *TestDataFactory) CreateGeneriqueList(groupID int, libelle string, medicamentCIS []int) entities.GeneriqueList {
	generiqueMedicaments := make([]entities.GeneriqueMedicament, len(medicamentCIS))
	for i, cis := range medicamentCIS {
		generiqueMedicaments[i] = entities.GeneriqueMedicament{
			Cis:          cis,
			Denomination: fmt.Sprintf("Médicament Générique %d", cis),
		}
	}

	return entities.GeneriqueList{
		GroupID:           groupID,
		Libelle:           libelle,
		LibelleNormalized: strings.ReplaceAll(strings.ToLower(libelle), "+", " "),
		Medicaments:       generiqueMedicaments,
		OrphanCIS:         []int{},
	}
}

// CreateDataContainer creates a fully populated data container
func (f *TestDataFactory) CreateDataContainer(medicamentsCount int, generiquesCount int) *data.DataContainer {
	medicaments := f.CreateMedicaments(medicamentsCount)
	medicamentsMap := f.CreateMedicamentsMap(medicaments)

	var generiques []entities.GeneriqueList
	for i := 0; i < generiquesCount; i++ {
		generiques = append(generiques, f.CreateGeneriqueList(
			i+1,
			fmt.Sprintf("Générique Test %d", i+1),
			[]int{(i % medicamentsCount) + 1}, // Link to existing medicaments
		))
	}

	generiquesMap := make(map[int]entities.GeneriqueList)
	// Note: In real implementation, you'd populate this with actual GeneriqueList entities

	dataContainer := &data.DataContainer{}
	dataContainer.UpdateData(medicaments, generiques, medicamentsMap, generiquesMap,
		map[int]entities.Presentation{}, map[int]entities.Presentation{}, &interfaces.DataQualityReport{
			DuplicateCIS:                       []int{},
			DuplicateGroupIDs:                  []int{},
			MedicamentsWithoutConditions:       0,
			MedicamentsWithoutGeneriques:       0,
			MedicamentsWithoutPresentations:    0,
			MedicamentsWithoutCompositions:     0,
			GeneriqueOnlyCIS:                   0,
			MedicamentsWithoutConditionsCIS:    []int{},
			MedicamentsWithoutGeneriquesCIS:    []int{},
			MedicamentsWithoutPresentationsCIS: []int{},
			MedicamentsWithoutCompositionsCIS:  []int{},
			GeneriqueOnlyCISList:               []int{},
		})
	return dataContainer
}

// ============================================================================
// MOCK BUILDERS
// ============================================================================

// MockDataStoreBuilder provides fluent interface for building mock data stores
type MockDataStoreBuilder struct {
	mock *MockDataStore
}

func NewMockDataStoreBuilder() *MockDataStoreBuilder {
	return &MockDataStoreBuilder{
		mock: &MockDataStore{
			medicaments:    []entities.Medicament{},
			generiques:     []entities.GeneriqueList{},
			medicamentsMap: make(map[int]entities.Medicament),
			generiquesMap:  make(map[int]entities.GeneriqueList),
			lastUpdated:    time.Now(),
			updating:       false,
		},
	}
}

func (b *MockDataStoreBuilder) WithMedicaments(medicaments []entities.Medicament) *MockDataStoreBuilder {
	b.mock.medicaments = medicaments
	b.mock.medicamentsMap = make(map[int]entities.Medicament)
	for _, med := range medicaments {
		b.mock.medicamentsMap[med.Cis] = med
	}
	return b
}

func (b *MockDataStoreBuilder) WithGeneriques(generiques []entities.GeneriqueList) *MockDataStoreBuilder {
	b.mock.generiques = generiques
	return b
}

func (b *MockDataStoreBuilder) WithUpdating(updating bool) *MockDataStoreBuilder {
	b.mock.updating = updating
	return b
}

func (b *MockDataStoreBuilder) WithLastUpdated(lastUpdated time.Time) *MockDataStoreBuilder {
	b.mock.lastUpdated = lastUpdated
	return b
}

func (b *MockDataStoreBuilder) WithPresentationsCIP7Map(presentationsCIP7Map map[int]entities.Presentation) *MockDataStoreBuilder {
	b.mock.presentationsCIP7Map = presentationsCIP7Map
	return b
}

func (b *MockDataStoreBuilder) WithPresentationsCIP13Map(presentationsCIP13Map map[int]entities.Presentation) *MockDataStoreBuilder {
	b.mock.presentationsCIP13Map = presentationsCIP13Map
	return b
}

func (b *MockDataStoreBuilder) WithGeneriquesMap(generiquesMap map[int]entities.GeneriqueList) *MockDataStoreBuilder {
	b.mock.generiquesMap = generiquesMap
	return b
}

func (b *MockDataStoreBuilder) Build() *MockDataStore {
	return b.mock
}

// MockDataValidatorBuilder provides fluent interface for building mock validators
type MockDataValidatorBuilder struct {
	mock *MockDataValidator
}

func NewMockDataValidatorBuilder() *MockDataValidatorBuilder {
	return &MockDataValidatorBuilder{
		mock: &MockDataValidator{
			validateInputError:      nil,
			validateMedicamentError: nil,
		},
	}
}

func (b *MockDataValidatorBuilder) WithInputError(err error) *MockDataValidatorBuilder {
	b.mock.validateInputError = err
	return b
}

func (b *MockDataValidatorBuilder) WithMedicamentError(err error) *MockDataValidatorBuilder {
	b.mock.validateMedicamentError = err
	return b
}

func (b *MockDataValidatorBuilder) Build() *MockDataValidator {
	return b.mock
}

// ============================================================================
// HTTP TEST UTILITIES
// ============================================================================

// HTTPTestHelper provides utilities for HTTP handler testing
type HTTPTestHelper struct {
	t *testing.T
}

func NewHTTPTestHelper(t *testing.T) *HTTPTestHelper {
	return &HTTPTestHelper{t: t}
}

// ExecuteRequest executes an HTTP handler with given parameters
func (h *HTTPTestHelper) ExecuteRequest(handler http.HandlerFunc, method, path string, urlParams map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)

	if len(urlParams) > 0 {
		rctx := chi.NewRouteContext()
		for key, value := range urlParams {
			rctx.URLParams.Add(key, value)
		}
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	}

	rr := httptest.NewRecorder()
	handler(rr, req)
	return rr
}

// AssertJSONResponse asserts that response contains valid JSON with expected status
func (h *HTTPTestHelper) AssertJSONResponse(resp *httptest.ResponseRecorder, expectedStatus int, target any) {
	if resp.Code != expectedStatus {
		h.t.Errorf("Expected status %d, got %d", expectedStatus, resp.Code)
	}

	bodyStr := resp.Body.String()
	if bodyStr == "" {
		h.t.Error("Response body should not be empty")
	}

	err := json.Unmarshal([]byte(bodyStr), target)
	if err != nil {
		h.t.Errorf("Response should be valid JSON, got error: %v", err)
	}
}

// AssertErrorResponse asserts that response contains an error with expected status
func (h *HTTPTestHelper) AssertErrorResponse(resp *httptest.ResponseRecorder, expectedStatus int) {
	if resp.Code != expectedStatus {
		h.t.Errorf("Expected status %d, got %d", expectedStatus, resp.Code)
	}

	var errorResp map[string]any
	err := json.Unmarshal(resp.Body.Bytes(), &errorResp)
	if err != nil {
		h.t.Errorf("Error response should be valid JSON, got error: %v", err)
	}

	// Check that it has error fields
	if _, ok := errorResp["message"]; !ok {
		h.t.Error("Error response should have message field")
	}
	if _, ok := errorResp["code"]; !ok {
		h.t.Error("Error response should have code field")
	}
}

// AssertPaginationResponse asserts pagination-specific response structure
func (h *HTTPTestHelper) AssertPaginationResponse(resp *httptest.ResponseRecorder, expectedPage, expectedMaxPage, expectedDataCount int) {
	var response map[string]any
	h.AssertJSONResponse(resp, http.StatusOK, &response)

	if response["page"] != float64(expectedPage) {
		h.t.Errorf("Page number mismatch: expected %d, got %v", expectedPage, response["page"])
	}
	if response["maxPage"] != float64(expectedMaxPage) {
		h.t.Errorf("Max page mismatch: expected %d, got %v", expectedMaxPage, response["maxPage"])
	}

	data, ok := response["data"].([]any)
	if !ok {
		h.t.Error("Data field should be an array")
	}
	if len(data) != expectedDataCount {
		h.t.Errorf("Data count mismatch: expected %d, got %d", expectedDataCount, len(data))
	}
}

// AssertHealthResponse asserts health check response structure
func (h *HTTPTestHelper) AssertHealthResponse(resp *httptest.ResponseRecorder, expectedStatus string) {
	var response map[string]any
	h.AssertJSONResponse(resp, http.StatusOK, &response)

	if response["status"] != expectedStatus {
		h.t.Errorf("Status mismatch: expected %s, got %v", expectedStatus, response["status"])
	}
	if _, ok := response["data"]; !ok {
		h.t.Error("Response should have data field")
	}
	if _, ok := response["system"]; !ok {
		h.t.Error("Response should have system field")
	}
}

// ============================================================================
// MOCK IMPLEMENTATIONS
// ============================================================================

// MockDataStore implements interfaces.DataStore for testing
type MockDataStore struct {
	medicaments           []entities.Medicament
	generiques            []entities.GeneriqueList
	medicamentsMap        map[int]entities.Medicament
	generiquesMap         map[int]entities.GeneriqueList
	presentationsCIP7Map  map[int]entities.Presentation
	presentationsCIP13Map map[int]entities.Presentation
	lastUpdated           time.Time
	updating              bool

	// Method call tracking
	getMedicamentsCalled    bool
	getGeneriquesCalled     bool
	getMedicamentsMapCalled bool
	getGeneriquesMapCalled  bool
	beginUpdateCalled       bool
	endUpdateCalled         bool
	updateDataCalled        bool
}

func (m *MockDataStore) GetMedicaments() []entities.Medicament {
	m.getMedicamentsCalled = true
	return m.medicaments
}

func (m *MockDataStore) GetGeneriques() []entities.GeneriqueList {
	m.getGeneriquesCalled = true
	return m.generiques
}

func (m *MockDataStore) GetMedicamentsMap() map[int]entities.Medicament {
	m.getMedicamentsMapCalled = true
	return m.medicamentsMap
}

func (m *MockDataStore) GetGeneriquesMap() map[int]entities.GeneriqueList {
	m.getGeneriquesMapCalled = true
	return m.generiquesMap
}

func (m *MockDataStore) GetPresentationsCIP7Map() map[int]entities.Presentation {
	return m.presentationsCIP7Map
}

func (m *MockDataStore) GetPresentationsCIP13Map() map[int]entities.Presentation {
	return m.presentationsCIP13Map
}

func (m *MockDataStore) GetLastUpdated() time.Time {
	return m.lastUpdated
}

func (m *MockDataStore) IsUpdating() bool {
	return m.updating
}

func (m *MockDataStore) UpdateData(medicaments []entities.Medicament, generiques []entities.GeneriqueList,
	medicamentsMap map[int]entities.Medicament, generiquesMap map[int]entities.GeneriqueList,
	presentationsCIP7Map map[int]entities.Presentation, presentationsCIP13Map map[int]entities.Presentation,
	report *interfaces.DataQualityReport) {
	m.updateDataCalled = true
	m.medicaments = medicaments
	m.generiques = generiques
	m.medicamentsMap = medicamentsMap
	m.generiquesMap = generiquesMap
	m.presentationsCIP7Map = presentationsCIP7Map
	m.presentationsCIP13Map = presentationsCIP13Map
	m.lastUpdated = time.Now()
}

func (m *MockDataStore) BeginUpdate() bool {
	m.beginUpdateCalled = true
	m.updating = true
	return true
}

func (m *MockDataStore) EndUpdate() {
	m.endUpdateCalled = true
	m.updating = false
}

func (m *MockDataStore) GetServerStartTime() time.Time {
	return time.Time{} // Return zero time for mock
}

func (m *MockDataStore) GetDataQualityReport() *interfaces.DataQualityReport {
	return &interfaces.DataQualityReport{
		DuplicateCIS:                       []int{},
		DuplicateGroupIDs:                  []int{},
		MedicamentsWithoutConditions:       0,
		MedicamentsWithoutGeneriques:       0,
		MedicamentsWithoutPresentations:    0,
		MedicamentsWithoutCompositions:     0,
		GeneriqueOnlyCIS:                   0,
		MedicamentsWithoutConditionsCIS:    []int{},
		MedicamentsWithoutGeneriquesCIS:    []int{},
		MedicamentsWithoutPresentationsCIS: []int{},
		MedicamentsWithoutCompositionsCIS:  []int{},
		GeneriqueOnlyCISList:               []int{},
	}
}

// MockDataValidator implements interfaces.DataValidator for testing
type MockDataValidator struct {
	validateInputError      error
	validateMedicamentError error

	validateInputCalled bool
	lastValidatedInput  string
}

func (m *MockDataValidator) ValidateMedicament(med *entities.Medicament) error {
	return m.validateMedicamentError
}

func (m *MockDataValidator) CheckDuplicateCIP(presentations []entities.Presentation) error {
	return nil
}

func (m *MockDataValidator) ReportDataQuality(medicaments []entities.Medicament, generiques []entities.GeneriqueList) *interfaces.DataQualityReport {
	return &interfaces.DataQualityReport{
		DuplicateCIS:                       []int{},
		DuplicateGroupIDs:                  []int{},
		MedicamentsWithoutConditions:       0,
		MedicamentsWithoutGeneriques:       0,
		MedicamentsWithoutPresentations:    0,
		MedicamentsWithoutCompositions:     0,
		GeneriqueOnlyCIS:                   0,
		MedicamentsWithoutConditionsCIS:    []int{},
		MedicamentsWithoutGeneriquesCIS:    []int{},
		MedicamentsWithoutPresentationsCIS: []int{},
		MedicamentsWithoutCompositionsCIS:  []int{},
		GeneriqueOnlyCISList:               []int{},
	}
}

func (m *MockDataValidator) ValidateDataIntegrity(medicaments []entities.Medicament, generiques []entities.GeneriqueList) error {
	return nil
}

func (m *MockDataValidator) ValidateInput(input string) error {
	m.validateInputCalled = true
	m.lastValidatedInput = input
	return m.validateInputError
}

func (m *MockDataValidator) ValidateCIP(input string) (int, error) {
	if m.validateInputError != nil {
		return -1, m.validateInputError
	}
	// Match real validator: check empty first
	if strings.TrimSpace(input) == "" {
		return -1, fmt.Errorf("input cannot be empty")
	}
	// Check length: CIP should be 7 or 13 digits
	if len(input) != 7 && len(input) != 13 {
		return -1, fmt.Errorf("CIP should have 7 or 13 characters")
	}
	// Check for non-numeric characters
	validPattern := regexp.MustCompile(`^[0-9]+$`)
	if !validPattern.MatchString(input) {
		return -1, fmt.Errorf("input contains invalid characters. Only numeric characters are allowed")
	}
	// Try to parse as int
	val, err := strconv.Atoi(input)
	if err != nil {
		return -1, fmt.Errorf("input is not a number")
	}
	return val, nil
}

func (m *MockDataValidator) ValidateCIS(input string) (int, error) {
	if m.validateInputError != nil {
		return -1, m.validateInputError
	}
	// Match real validator: check empty first
	if strings.TrimSpace(input) == "" {
		return -1, fmt.Errorf("input cannot be empty")
	}
	// Check length: CIS should be 8 digits
	if len(input) != 8 {
		return -1, fmt.Errorf("CIS should have 8 digits")
	}
	// Check for non-numeric characters
	validPattern := regexp.MustCompile(`^[0-9]+$`)
	if !validPattern.MatchString(input) {
		return -1, fmt.Errorf("input contains invalid characters. Only numeric characters are allowed")
	}
	// Try to parse as int
	val, err := strconv.Atoi(input)
	if err != nil {
		return -1, fmt.Errorf("input is not a number")
	}
	return val, nil
}
