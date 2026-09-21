package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/giygas/medicaments-api/ansmdocs"
	"github.com/giygas/medicaments-api/data"
	"github.com/giygas/medicaments-api/docstore"
	"github.com/giygas/medicaments-api/handlers"
	"github.com/giygas/medicaments-api/health"
	"github.com/giygas/medicaments-api/interfaces"
	"github.com/giygas/medicaments-api/logging"
	"github.com/giygas/medicaments-api/medicamentsparser/entities"
	"github.com/giygas/medicaments-api/validation"
)

// TestApplicationStartupSmoke tests basic application startup and functionality
// This is a fast sanity check that should run before expensive integration tests
// Expected duration: < 2 seconds
func TestApplicationStartupSmoke(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping smoke test in short mode")
	}

	t.Log("Starting application smoke test...")

	// 0. Initialize logging first
	logging.InitLogger("logs")

	// 1. Create minimal data container with sample data
	// This ensures data is available for endpoints
	sampleMedicament := entities.Medicament{
		Cis:                  12345678,
		Denomination:         "Smoke Test Medicament",
		FormePharmaceutique:  "Tablet",
		StatusAutorisation:   "Autorisée",
		EtatComercialisation: "Commercialisée",
		Conditions:           []string{"Smoke condition"},
		Presentation: []entities.Presentation{
			{
				Cip7:    7001001,
				Cip13:   3400912345678,
				Libelle: "Boîte de 10 comprimés",
			},
		},
		Composition: []entities.Composition{
			{
				CodeSubstance:         1,
				DenominationSubstance: "Test substance",
			},
		},
		Generiques: []entities.Generique{
			{
				Cis:   12345678,
				Group: 100,
			},
		},
	}

	sampleGenerique := entities.GeneriqueList{
		GroupID: 100,
		Libelle: "Smoke Test Group",
		Medicaments: []entities.GeneriqueMedicament{
			{Cis: 12345678, Denomination: "Smoke Test Medicament"},
		},
	}

	dataContainer := data.NewDataContainer()
	dataContainer.UpdateData(
		[]entities.Medicament{sampleMedicament},
		[]entities.GeneriqueList{sampleGenerique},
		map[int]entities.Medicament{sampleMedicament.Cis: sampleMedicament},
		map[int]entities.GeneriqueList{sampleGenerique.GroupID: sampleGenerique},
		map[int]entities.Presentation{},
		map[int]entities.Presentation{},
		&interfaces.DataQualityReport{},
	)

	// 2. Create validator and handler
	validator := validation.NewDataValidator()
	healthChecker := health.NewHealthChecker(dataContainer)
	httpHandler := handlers.NewHTTPHandler(dataContainer, validator, healthChecker)

	// 3. Test health endpoint
	t.Log("Testing health endpoint...")
	healthRR := httptest.NewRequest("GET", "/health", nil)
	healthRecorder := httptest.NewRecorder()
	httpHandler.HealthCheck(healthRecorder, healthRR)

	if healthRecorder.Code != http.StatusOK {
		t.Fatalf("Health endpoint returned status %d, expected %d", healthRecorder.Code, http.StatusOK)
	}

	var healthResp map[string]any
	if err := json.Unmarshal(healthRecorder.Body.Bytes(), &healthResp); err != nil {
		t.Fatalf("Failed to decode health response: %v", err)
	}

	// Verify required health endpoint fields
	if status, ok := healthResp["status"].(string); !ok {
		t.Error("Health response missing status field")
	} else if status != "healthy" {
		t.Errorf("Unexpected health status: %s, expected healthy", status)
	}

	if _, ok := healthResp["data"]; !ok {
		t.Error("Health response missing data field")
	}

	t.Log("Health endpoint responded correctly")

	// 4. Test database export endpoint
	t.Log("Testing database export endpoint...")
	exportRR := httptest.NewRequest("GET", "/v1/medicaments/export", nil)
	exportRecorder := httptest.NewRecorder()
	httpHandler.ExportMedicaments(exportRecorder, exportRR)

	if exportRecorder.Code != http.StatusOK {
		t.Fatalf("Database export endpoint returned status %d, expected %d", exportRecorder.Code, http.StatusOK)
	}

	var exportResp []entities.Medicament
	if err := json.Unmarshal(exportRecorder.Body.Bytes(), &exportResp); err != nil {
		t.Fatalf("Failed to decode export response: %v", err)
	}

	if len(exportResp) != 1 {
		t.Errorf("Expected 1 medicament in export, got %d", len(exportResp))
	}

	t.Log("Database export endpoint responded correctly")

	t.Log("Smoke test completed successfully")
}

// smokeDocsPage is a minimal ANSM-like "extrait" page: it carries an RCP tab
// panel but no notice panel, so one smoke run exercises both the lazy
// fetch-and-cache path (RCP) and the tombstone path (notice).
const smokeDocsPage = `<!doctype html>
<html>
<body>
<main>
<h3>Smoke Test Medicament</h3>
<div id="tabpanel-rcp-panel">
<p class="DateNotif">ANSM - Mis à jour le : 01/09/2026</p>
<p class="AmmAnnexeTitre1">1. DENOMINATION DU MEDICAMENT</p>
<p>Smoke Test Medicament, comprimé</p>
</div>
</main>
</body>
</html>`

// TestDocsEndpointsSmoke tests the ANSM document (RCP/notice) endpoints
// against a fake upstream: lazy fetch + disk cache for the RCP, tombstone 404
// for the absent notice.
// This is a fast sanity check that should run before expensive integration
// tests. Expected duration: < 1 second
func TestDocsEndpointsSmoke(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping smoke test in short mode")
	}

	t.Log("Starting ANSM docs endpoints smoke test...")

	// 1. Minimal data container holding the smoke medicament
	medicament := entities.Medicament{
		Cis:          12345678,
		Denomination: "Smoke Test Medicament",
	}
	dataContainer := data.NewDataContainer()
	dataContainer.UpdateData(
		[]entities.Medicament{medicament},
		[]entities.GeneriqueList{},
		map[int]entities.Medicament{medicament.Cis: medicament},
		map[int]entities.GeneriqueList{},
		map[int]entities.Presentation{},
		map[int]entities.Presentation{},
		&interfaces.DataQualityReport{},
	)

	// 2. Fake ANSM upstream (offline: loopback httptest server)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(smokeDocsPage))
	}))
	defer upstream.Close()

	// 3. Handler with the document dependencies wired (DOCS_ENABLED=true
	// equivalent), routed through chi so path values resolve like production.
	store, err := docstore.NewDocumentStore(t.TempDir())
	if err != nil {
		t.Fatalf("Failed to create document store: %v", err)
	}
	fetcher := ansmdocs.NewFetcher(ansmdocs.Config{
		BaseURL:    upstream.URL + "/medicament",
		RatePerSec: -1, // tests only: no upstream rate limiting
		Timeout:    2 * time.Second,
		MaxRetries: 0,
	})
	validator := validation.NewDataValidator()
	healthChecker := health.NewHealthChecker(dataContainer)
	httpHandler := handlers.NewHandler(dataContainer, validator, healthChecker,
		handlers.WithDocuments(store, fetcher, 2*time.Second))

	router := chi.NewRouter()
	router.Get("/v1/medicaments/{cis}/rcp", httpHandler.ServeRCPV1)
	router.Get("/v1/medicaments/{cis}/notice", httpHandler.ServeNoticeV1)

	// 4. Test RCP endpoint: lazily fetched, cached, served as sectioned JSON
	t.Log("Testing RCP document endpoint...")
	rcpReq := httptest.NewRequest("GET", "/v1/medicaments/12345678/rcp", nil)
	rcpRecorder := httptest.NewRecorder()
	router.ServeHTTP(rcpRecorder, rcpReq)

	if rcpRecorder.Code != http.StatusOK {
		t.Fatalf("RCP endpoint returned status %d, expected %d\nbody: %s",
			rcpRecorder.Code, http.StatusOK, rcpRecorder.Body.String())
	}

	var rcpDoc docsTestDocument
	if err := json.Unmarshal(rcpRecorder.Body.Bytes(), &rcpDoc); err != nil {
		t.Fatalf("Failed to decode RCP response: %v", err)
	}

	if rcpDoc.Type != "rcp" {
		t.Errorf("Unexpected RCP type: %s, expected rcp", rcpDoc.Type)
	}
	if rcpDoc.MiseAJour != "2026-09-01" {
		t.Errorf("Unexpected RCP miseAJour: %s, expected 2026-09-01", rcpDoc.MiseAJour)
	}
	if rcpDoc.Source != "ANSM - Base de données publique des médicaments" {
		t.Errorf("Unexpected RCP source: %s, expected the ANSM attribution", rcpDoc.Source)
	}
	if len(rcpDoc.Sections) == 0 || rcpDoc.Sections[0].ID != "1" {
		t.Errorf("Expected RCP sections to start with canonical rubrique 1, got %d sections", len(rcpDoc.Sections))
	}

	if rcpRecorder.Header().Get("ETag") == "" {
		t.Error("RCP response missing ETag header")
	}
	if cc := rcpRecorder.Header().Get("Cache-Control"); cc != "public, max-age=3600, s-maxage=86400" {
		t.Errorf("Unexpected RCP Cache-Control: %s", cc)
	}

	// 5. Second RCP request served from the disk cache
	rcpReq2 := httptest.NewRequest("GET", "/v1/medicaments/12345678/rcp", nil)
	rcpRecorder2 := httptest.NewRecorder()
	router.ServeHTTP(rcpRecorder2, rcpReq2)

	if rcpRecorder2.Code != http.StatusOK {
		t.Errorf("Second RCP request returned status %d, expected %d", rcpRecorder2.Code, http.StatusOK)
	}
	if rcpRecorder2.Body.String() != rcpRecorder.Body.String() {
		t.Error("Second RCP response body differs from the first (cache should be stable)")
	}

	// 6. Test notice endpoint: absent upstream -> 404 (tombstone)
	t.Log("Testing notice document endpoint (absent upstream)...")
	noticeReq := httptest.NewRequest("GET", "/v1/medicaments/12345678/notice", nil)
	noticeRecorder := httptest.NewRecorder()
	router.ServeHTTP(noticeRecorder, noticeReq)

	if noticeRecorder.Code != http.StatusNotFound {
		t.Fatalf("Notice endpoint returned status %d, expected %d (document absent upstream)",
			noticeRecorder.Code, http.StatusNotFound)
	}
	if !strings.Contains(noticeRecorder.Body.String(), "not available") {
		t.Errorf("Notice 404 body should state the document is not available, got: %s",
			noticeRecorder.Body.String())
	}

	t.Log("ANSM docs endpoints smoke test completed successfully")
}
