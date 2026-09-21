package interfaces

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/giygas/medicaments-api/docstore"
	"github.com/giygas/medicaments-api/medicamentsparser/entities"
)

// MockDataStore implements DataStore interface for testing
type MockDataStore struct {
	medicaments           []entities.Medicament
	generiques            []entities.GeneriqueList
	medicamentsMap        map[int]entities.Medicament
	generiquesMap         map[int]entities.GeneriqueList
	presentationsCIP7Map  map[int]entities.Presentation
	presentationsCIP13Map map[int]entities.Presentation
	lastUpdated           time.Time
	updating              bool
}

func (m *MockDataStore) GetMedicaments() []entities.Medicament {
	return m.medicaments
}

func (m *MockDataStore) GetGeneriques() []entities.GeneriqueList {
	return m.generiques
}

func (m *MockDataStore) GetMedicamentsMap() map[int]entities.Medicament {
	return m.medicamentsMap
}

func (m *MockDataStore) GetGeneriquesMap() map[int]entities.GeneriqueList {
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

func (m *MockDataStore) UpdateData(medicaments []entities.Medicament, generiques []entities.GeneriqueList, medicamentsMap map[int]entities.Medicament, generiquesMap map[int]entities.GeneriqueList, presentationsCIP7Map map[int]entities.Presentation, presentationsCIP13Map map[int]entities.Presentation, report *DataQualityReport) {
	m.medicaments = medicaments
	m.generiques = generiques
	m.medicamentsMap = medicamentsMap
	m.generiquesMap = generiquesMap
	m.presentationsCIP7Map = presentationsCIP7Map
	m.presentationsCIP13Map = presentationsCIP13Map
	m.lastUpdated = time.Now()
}

func (m *MockDataStore) GetDataQualityReport() *DataQualityReport {
	return &DataQualityReport{
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

func (m *MockDataStore) BeginUpdate() bool {
	if m.updating {
		return false
	}
	m.updating = true
	return true
}

func (m *MockDataStore) EndUpdate() {
	m.updating = false
}

func (m *MockDataStore) GetServerStartTime() time.Time {
	return time.Time{} // Return zero time for mock
}

// MockParser implements Parser interface for testing
type MockParser struct {
	shouldFail bool
}

func (m *MockParser) ParseAllMedicaments() ([]entities.Medicament, map[int]entities.Presentation, map[int]entities.Presentation, error) {
	if m.shouldFail {
		return nil, nil, nil, &mockError{"parse failed"}
	}

	return []entities.Medicament{
		{Cis: 1, Denomination: "Test Medicament"},
		{Cis: 2, Denomination: "Another Test"},
	}, map[int]entities.Presentation{
		1234567: {Cis: 1, Cip7: 1234567, Cip13: 3400912345678},
	}, map[int]entities.Presentation{
		3400912345678: {Cis: 1, Cip7: 1234567, Cip13: 3400912345678},
	}, nil
}

func (m *MockParser) GeneriquesParser(medicaments *[]entities.Medicament, medicamentsMap *map[int]entities.Medicament) ([]entities.GeneriqueList, map[int]entities.GeneriqueList, error) {
	if m.shouldFail {
		return nil, nil, &mockError{"generiques parse failed"}
	}

	generiques := []entities.GeneriqueList{
		{GroupID: 1, Libelle: "Test Generique"},
	}
	generiquesMap := map[int]entities.GeneriqueList{
		1: {GroupID: 1, Libelle: "Test Generique"},
	}

	return generiques, generiquesMap, nil
}

// MockScheduler implements Scheduler interface for testing
type MockScheduler struct {
	started bool
	stopped bool
}

func (m *MockScheduler) Start() error {
	if m.started {
		return &mockError{"already started"}
	}
	m.started = true
	return nil
}

func (m *MockScheduler) Stop() {
	m.stopped = true
}

// MockHTTPHandler implements HTTPHandler interface for testing
type MockHTTPHandler struct {
	responseCode int
	responseBody string
}

func (m *MockHTTPHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(m.responseCode)
	_, _ = w.Write([]byte(m.responseBody))
}

func (m *MockHTTPHandler) ExportMedicaments(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(m.responseCode)
	_, _ = w.Write([]byte(m.responseBody))
}

func (m *MockHTTPHandler) ServePagedMedicaments(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(m.responseCode)
	_, _ = w.Write([]byte(m.responseBody))
}

func (m *MockHTTPHandler) FindMedicament(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(m.responseCode)
	_, _ = w.Write([]byte(m.responseBody))
}

func (m *MockHTTPHandler) FindMedicamentByCIS(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(m.responseCode)
	_, _ = w.Write([]byte(m.responseBody))
}

func (m *MockHTTPHandler) FindMedicamentByCIP(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(m.responseCode)
	_, _ = w.Write([]byte(m.responseBody))
}

func (m *MockHTTPHandler) FindGeneriques(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(m.responseCode)
	_, _ = w.Write([]byte(m.responseBody))
}

func (m *MockHTTPHandler) FindGeneriquesByGroupID(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(m.responseCode)
	_, _ = w.Write([]byte(m.responseBody))
}

func (m *MockHTTPHandler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(m.responseCode)
	_, _ = w.Write([]byte(m.responseBody))
}

func (m *MockHTTPHandler) ServeMedicamentsV1(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(m.responseCode)
	_, _ = w.Write([]byte(m.responseBody))
}

func (m *MockHTTPHandler) ServePresentationsV1(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(m.responseCode)
	_, _ = w.Write([]byte(m.responseBody))
}

func (m *MockHTTPHandler) ServePresentationsMissingCIP(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(m.responseCode)
	_, _ = w.Write([]byte(m.responseBody))
}

func (m *MockHTTPHandler) ServeGeneriquesV1(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(m.responseCode)
	_, _ = w.Write([]byte(m.responseBody))
}

func (m *MockHTTPHandler) ServeDiagnosticsV1(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(m.responseCode)
	_, _ = w.Write([]byte(m.responseBody))
}

// MockDataValidator implements DataValidator interface for testing
type MockDataValidator struct {
	shouldFail bool
}

func (m *MockDataValidator) ValidateMedicament(med *entities.Medicament) error {
	if m.shouldFail {
		return fmt.Errorf("validation failed")
	}
	return nil
}

func (m *MockDataValidator) CheckDuplicateCIP(presentations []entities.Presentation) error {
	if m.shouldFail {
		return fmt.Errorf("validation failed")
	}
	return nil
}

func (m *MockDataValidator) ReportDataQuality(
	medicaments []entities.Medicament,
	generiques []entities.GeneriqueList,
	presentationsCIP7Map map[int]entities.Presentation,
	presentationsCIP13Map map[int]entities.Presentation,
) *DataQualityReport {
	return &DataQualityReport{
		DuplicateCIS:                        []int{},
		DuplicateGroupIDs:                   []int{},
		MedicamentsWithoutConditions:        0,
		MedicamentsWithoutGeneriques:        0,
		MedicamentsWithoutPresentations:     0,
		MedicamentsWithoutCompositions:      0,
		GeneriqueOnlyCIS:                    0,
		PresentationsWithOrphanedCIS:        0,
		MedicamentsWithoutConditionsCIS:     []int{},
		MedicamentsWithoutGeneriquesCIS:     []int{},
		MedicamentsWithoutPresentationsCIS:  []int{},
		MedicamentsWithoutCompositionsCIS:   []int{},
		GeneriqueOnlyCISList:                []int{},
		PresentationsWithOrphanedCISCIPList: []int{},
	}
}

func (m *MockDataValidator) ValidateDataIntegrity(medicaments []entities.Medicament, generiques []entities.GeneriqueList) error {
	if m.shouldFail {
		return fmt.Errorf("validation failed")
	}
	return nil
}

func (m *MockDataValidator) ValidateInput(input string) error {
	if m.shouldFail {
		return fmt.Errorf("input validation failed")
	}
	return nil
}

func (m *MockDataValidator) ValidateCIP(input string) (int, error) {
	if m.shouldFail {
		return -1, fmt.Errorf("CIP validation failed")
	}
	// Try to parse as int for simple validation
	val, err := strconv.Atoi(input)
	if err != nil {
		return -1, fmt.Errorf("input is not a number")
	}
	return val, nil
}

func (m *MockDataValidator) ValidateCIS(input string) (int, error) {
	if m.shouldFail {
		return -1, fmt.Errorf("CIS validation failed")
	}
	// Try to parse as int for simple validation
	val, err := strconv.Atoi(input)
	if err != nil {
		return -1, fmt.Errorf("input is not a number")
	}
	return val, nil
}

// mockError is a simple error type for testing
type mockError struct {
	msg string
}

func (e *mockError) Error() string {
	return e.msg
}

// Sentinel errors mimicking docstore.ErrNotFound / docstore.ErrTombstone and
// ansmdocs.ErrNotAvailable for the mocks below. They stay local so the
// interface tests exercise the contracts, not the concrete packages.
var (
	errMockNotFound     = errors.New("mock: document not found")
	errMockTombstone    = errors.New("mock: document marked as missing (tombstone)")
	errMockNotAvailable = errors.New("mock: document not available upstream")
)

// mockStoredDoc is one cached document held by MockDocumentStore.
type mockStoredDoc struct {
	payload    []byte
	sourceDate string
}

// MockDocumentStore implements DocumentStore interface for testing
type MockDocumentStore struct {
	documents  map[string]mockStoredDoc
	tombstones map[string]bool
	putCount   int
	tombstoned int
}

func (m *MockDocumentStore) Get(cis, docType string) ([]byte, *docstore.DocumentMeta, error) {
	key := cis + "|" + docType
	if m.tombstones[key] {
		return nil, &docstore.DocumentMeta{Tombstone: true}, errMockTombstone
	}
	doc, ok := m.documents[key]
	if !ok {
		return nil, nil, errMockNotFound
	}
	return doc.payload, &docstore.DocumentMeta{SourceDate: doc.sourceDate}, nil
}

func (m *MockDocumentStore) Put(cis, docType string, jsonBytes []byte, sourceDate string) error {
	key := cis + "|" + docType
	if m.documents == nil {
		m.documents = make(map[string]mockStoredDoc)
	}
	// A real document always supersedes a tombstone for the same key.
	delete(m.tombstones, key)
	m.documents[key] = mockStoredDoc{payload: jsonBytes, sourceDate: sourceDate}
	m.putCount++
	return nil
}

func (m *MockDocumentStore) PutTombstone(cis, docType string) error {
	if m.tombstones == nil {
		m.tombstones = make(map[string]bool)
	}
	m.tombstones[cis+"|"+docType] = true
	m.tombstoned++
	return nil
}

func (m *MockDocumentStore) Stats() docstore.Stats {
	return docstore.Stats{
		Documents:  len(m.documents),
		Tombstones: len(m.tombstones),
	}
}

// MockANSMFetcher implements ANSMFetcher interface for testing
type MockANSMFetcher struct {
	payload    []byte
	sourceDate string
	err        error
	fetchCount int
}

func (m *MockANSMFetcher) Fetch(ctx context.Context, cis, docType string) ([]byte, string, error) {
	m.fetchCount++
	return m.payload, m.sourceDate, m.err
}

// Test functions demonstrating the benefits of interfaces

func TestDataStoreInterface(t *testing.T) {
	// We can easily test with a mock implementation
	store := &MockDataStore{
		medicaments: []entities.Medicament{{Cis: 1, Denomination: "Test"}},
	}

	medicaments := store.GetMedicaments()
	if len(medicaments) != 1 {
		t.Errorf("Expected 1 medicament, got %d", len(medicaments))
	}
}

func TestParserInterface(t *testing.T) {
	// Test successful parsing
	parser := &MockParser{shouldFail: false}
	medicaments, cip7Map, cip13Map, err := parser.ParseAllMedicaments()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if len(medicaments) != 2 {
		t.Errorf("Expected 2 medicaments, got %d", len(medicaments))
	}
	if len(cip7Map) != 1 {
		t.Errorf("Expected 1 CIP7 map entry, got %d", len(cip7Map))
	}
	if len(cip13Map) != 1 {
		t.Errorf("Expected 1 CIP13 map entry, got %d", len(cip13Map))
	}

	// Test failed parsing
	parser = &MockParser{shouldFail: true}
	_, _, _, err = parser.ParseAllMedicaments()
	if err == nil {
		t.Error("Expected error but got none")
	}
}

func TestSchedulerInterface(t *testing.T) {
	scheduler := &MockScheduler{}

	err := scheduler.Start()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if !scheduler.started {
		t.Error("Scheduler should be started")
	}

	scheduler.Stop()
	if !scheduler.stopped {
		t.Error("Scheduler should be stopped")
	}
}

func TestHTTPHandlerInterface(t *testing.T) {
	handler := &MockHTTPHandler{
		responseCode: http.StatusOK,
		responseBody: "test response",
	}

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	if w.Body.String() != "test response" {
		t.Errorf("Expected body 'test response', got '%s'", w.Body.String())
	}
}

func TestDataValidatorInterface(t *testing.T) {
	validator := &MockDataValidator{shouldFail: false}

	med := &entities.Medicament{Cis: 1, Denomination: "Test"}
	err := validator.ValidateMedicament(med)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Test validation failure
	validator = &MockDataValidator{shouldFail: true}
	err = validator.ValidateMedicament(med)
	if err == nil {
		t.Error("Expected validation error but got none")
	}
}

func TestDocumentStoreInterface(t *testing.T) {
	// Arrange: empty store
	store := &MockDocumentStore{
		documents:  make(map[string]mockStoredDoc),
		tombstones: make(map[string]bool),
	}

	// Act & Assert: miss on the empty store
	if _, _, err := store.Get("60016308", "rcp"); err == nil {
		t.Error("Expected miss error on empty store, got nil")
	}

	// Act: cache a document
	if err := store.Put("60016308", "rcp", []byte(`{"cis":"60016308"}`), "2025-11-07"); err != nil {
		t.Fatalf("Unexpected error on Put: %v", err)
	}

	// Assert: hit returns the payload and its metadata
	payload, meta, err := store.Get("60016308", "rcp")
	if err != nil {
		t.Fatalf("Unexpected error on Get after Put: %v", err)
	}
	if string(payload) != `{"cis":"60016308"}` {
		t.Errorf("Expected cached payload, got %s", string(payload))
	}
	if meta == nil || meta.SourceDate != "2025-11-07" {
		t.Errorf("Expected metadata with source date 2025-11-07, got %+v", meta)
	}

	// Act: record a tombstone for another document type
	if err := store.PutTombstone("60016308", "notice"); err != nil {
		t.Fatalf("Unexpected error on PutTombstone: %v", err)
	}

	// Assert: tombstone lookup surfaces the sentinel error
	if _, _, err := store.Get("60016308", "notice"); !errors.Is(err, errMockTombstone) {
		t.Errorf("Expected tombstone error, got %v", err)
	}

	// Assert: stats reflect both entries
	stats := store.Stats()
	if stats.Documents != 1 {
		t.Errorf("Expected 1 document in stats, got %d", stats.Documents)
	}
	if stats.Tombstones != 1 {
		t.Errorf("Expected 1 tombstone in stats, got %d", stats.Tombstones)
	}

	// Act: a real document supersedes the tombstone for the same key
	if err := store.Put("60016308", "notice", []byte(`{"cis":"60016308","type":"notice"}`), "2025-11-07"); err != nil {
		t.Fatalf("Unexpected error on Put over tombstone: %v", err)
	}
	if _, _, err := store.Get("60016308", "notice"); err != nil {
		t.Errorf("Expected hit after tombstone superseded, got %v", err)
	}

	// Assert: call counters tracked the operations above
	if store.putCount != 2 {
		t.Errorf("Expected 2 Put calls, got %d", store.putCount)
	}
	if store.tombstoned != 1 {
		t.Errorf("Expected 1 PutTombstone call, got %d", store.tombstoned)
	}
}

func TestANSMFetcherInterface(t *testing.T) {
	// Arrange: successful fetch
	fetcher := &MockANSMFetcher{
		payload:    []byte(`{"cis":"60016308","type":"rcp"}`),
		sourceDate: "2025-11-07",
	}

	// Act
	payload, sourceDate, err := fetcher.Fetch(context.Background(), "60016308", "rcp")

	// Assert
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if string(payload) != `{"cis":"60016308","type":"rcp"}` {
		t.Errorf("Expected fetched payload, got %s", string(payload))
	}
	if sourceDate != "2025-11-07" {
		t.Errorf("Expected source date 2025-11-07, got %s", sourceDate)
	}
	if fetcher.fetchCount != 1 {
		t.Errorf("Expected 1 fetch call, got %d", fetcher.fetchCount)
	}

	// Arrange: document definitively absent upstream
	fetcher = &MockANSMFetcher{err: errMockNotAvailable}

	// Act
	payload, sourceDate, err = fetcher.Fetch(context.Background(), "60016308", "notice")

	// Assert: the sentinel surfaces to the caller (tombstone signal)
	if !errors.Is(err, errMockNotAvailable) {
		t.Errorf("Expected errMockNotAvailable, got %v", err)
	}
	if payload != nil {
		t.Errorf("Expected nil payload on error, got %s", string(payload))
	}
	if sourceDate != "" {
		t.Errorf("Expected empty source date on error, got %s", sourceDate)
	}
}

// Example of how interfaces enable dependency injection
type Service struct {
	dataStore DataStore
	parser    Parser
	scheduler Scheduler
}

func NewService(dataStore DataStore, parser Parser, scheduler Scheduler) *Service {
	return &Service{
		dataStore: dataStore,
		parser:    parser,
		scheduler: scheduler,
	}
}

func (s *Service) GetMedicamentCount() int {
	return len(s.dataStore.GetMedicaments())
}

func TestServiceWithDependencyInjection(t *testing.T) {
	// We can easily test the service with mock dependencies
	mockStore := &MockDataStore{
		medicaments: []entities.Medicament{{Cis: 1}, {Cis: 2}},
	}
	mockParser := &MockParser{}
	mockScheduler := &MockScheduler{}

	service := NewService(mockStore, mockParser, mockScheduler)

	count := service.GetMedicamentCount()
	if count != 2 {
		t.Errorf("Expected 2 medicaments, got %d", count)
	}
}

// Compile-time checks to ensure our implementations implement the interfaces
func TestCompileTimeChecks(t *testing.T) {
	// These will fail to compile if the implementations don't match the interfaces
	var _ DataStore = (*MockDataStore)(nil)
	var _ Parser = (*MockParser)(nil)
	var _ Scheduler = (*MockScheduler)(nil)
	var _ HTTPHandler = (*MockHTTPHandler)(nil)
	var _ DataValidator = (*MockDataValidator)(nil)
	var _ DocumentStore = (*MockDocumentStore)(nil)
	var _ ANSMFetcher = (*MockANSMFetcher)(nil)
}
