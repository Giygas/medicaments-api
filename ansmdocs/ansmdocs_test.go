package ansmdocs

import (
	"context"
	"encoding/json"
	"errors"
	"html"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/giygas/medicaments-api/config"
	"github.com/giygas/medicaments-api/logging"
)

// init initializes the logger for all tests in this package: console logs are
// ERROR-only and no log files are created.
func init() {
	logging.InitLoggerWithEnvironment("", config.EnvTest, "", 4, 100*1024*1024)
}

// loadFixture reads a testdata HTML fixture.
func loadFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("failed to read fixture %s: %v", name, err)
	}
	return data
}

// newTestFetcher builds a Fetcher pointed at an httptest server with the
// rate limiter disabled for speed.
func newTestFetcher(baseURL string, mutate func(*Config)) *Fetcher {
	cfg := Config{BaseURL: baseURL, RatePerSec: -1}
	if mutate != nil {
		mutate(&cfg)
	}
	return NewFetcher(cfg)
}

// sectionIDs returns the ordered section IDs of a document.
func sectionIDs(doc *Document) []string {
	ids := make([]string, 0, len(doc.Sections))
	for _, s := range doc.Sections {
		ids = append(ids, s.ID)
	}
	return ids
}

// assertIDs fails unless the document sections carry exactly the wanted IDs
// in order.
func assertIDs(t *testing.T, doc *Document, want []string) {
	t.Helper()
	got := sectionIDs(doc)
	if len(got) != len(want) {
		t.Fatalf("section IDs = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("section IDs = %v, want %v", got, want)
		}
	}
}

// findSection returns the section with the given ID.
func findSection(t *testing.T, doc *Document, id string) Section {
	t.Helper()
	for _, s := range doc.Sections {
		if s.ID == id {
			return s
		}
	}
	t.Fatalf("section %q not found among %v", id, sectionIDs(doc))
	return Section{}
}

// assertNotInAnyContenu fails when any forbidden substring appears in any
// section contenu of the document.
func assertNotInAnyContenu(t *testing.T, doc *Document, forbidden ...string) {
	t.Helper()
	for _, s := range doc.Sections {
		for _, f := range forbidden {
			if strings.Contains(s.Contenu, f) {
				t.Errorf("section %q contenu contains forbidden %q", s.ID, f)
			}
		}
	}
}

// --- Sanitizer -----------------------------------------------------------

func TestSanitize(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			"keeps whitelisted elements",
			`<p>Texte <strong>gras</strong> et <em>italique</em>, <u>souligné</u>, H<sub>2</sub>O, x<sup>2</sup>.</p>`,
			`<p>Texte <strong>gras</strong> et <em>italique</em>, <u>souligné</u>, H<sub>2</sub>O, x<sup>2</sup>.</p>`,
		},
		{
			"keeps lists and strips their attributes",
			`<ul class="x"><li>un</li><li>deux</li></ul><ol start="2"><li>trois</li></ol>`,
			`<ul><li>un</li><li>deux</li></ul><ol><li>trois</li></ol>`,
		},
		{
			"keeps table with only colspan and rowspan on cells",
			`<table class="t"><thead><tr><th colspan="2" style="color:red">A</th></tr></thead><tbody><tr><td rowspan="3" id="cell">B</td><td align="left">C</td></tr></tbody></table>`,
			`<table><thead><tr><th colspan="2">A</th></tr></thead><tbody><tr><td rowspan="3">B</td><td>C</td></tr></tbody></table>`,
		},
		{
			"drops script and its content",
			`<p>ok</p><script>alert("xss")</script>`,
			`<p>ok</p>`,
		},
		{
			"drops style and its content",
			`<style>.a{color:red}</style><p>ok</p>`,
			`<p>ok</p>`,
		},
		{
			"strips div and span wrappers keeping text",
			`<div class="w" style="x"><span style="color:black">texte</span></div>`,
			`texte`,
		},
		{
			"strips anchors and event handlers keeping text",
			`<p>voir <a href="http://evil.example" onclick="evil()">le site</a></p>`,
			`<p>voir le site</p>`,
		},
		{
			"strips attributes from paragraphs",
			`<p class="AmmCorpsTexte" style="text-align:center" id="p1" onclick="x()">Contenu</p>`,
			`<p>Contenu</p>`,
		},
		{
			"keeps br and headings",
			`<h3>Titre</h3><h4>Sous-titre</h4><br>`,
			`<h3>Titre</h3><h4>Sous-titre</h4><br>`,
		},
		{
			"drops images and iframes",
			`<img src="http://x/y.png"><iframe src="http://x"></iframe><p>fin</p>`,
			`<p>fin</p>`,
		},
		{
			"colspan on non-cell element is dropped",
			`<p colspan="2">texte</p>`,
			`<p>texte</p>`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			got := Sanitize(tt.in)

			// Assert
			if got != tt.want {
				t.Errorf("Sanitize() =\n%s\nwant\n%s", got, tt.want)
			}
		})
	}
}

// --- Extractor helpers ---------------------------------------------------

func TestParseDateNotif(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"standard", "ANSM - Mis à jour le : 07/11/2025", "2025-11-07"},
		{"date only", "07/11/2025", "2025-11-07"},
		{"surrounding text", "Mis à jour le : 01/02/2026 (ANSM)", "2026-02-01"},
		{"no date", "ANSM - Mis à jour le :", ""},
		{"invalid month", "07/13/2025", ""},
		{"invalid day", "32/11/2025", ""},
		{"empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			got := parseDateNotif(tt.in)

			// Assert
			if got != tt.want {
				t.Errorf("parseDateNotif(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestSlugify(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"simple", "Dénomination du médicament", "denomination-du-medicament"},
		{"accents folded", "Posologie et mode d'administration", "posologie-et-mode-d-administration"},
		{"punctuation collapsed and trimmed", "Qu'est-ce que c'est ?", "qu-est-ce-que-c-est"},
		{"numbers kept", "1. QU'EST-CE QUE X ET DANS QUELS CAS ?", "1-qu-est-ce-que-x-et-dans-quels-cas"},
		{"leading and trailing separators", "  ...Titre...  ", "titre"},
		{"only separators", "???", ""},
		{"typographic apostrophe", "l’emballage", "l-emballage"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			got := slugify(tt.in)

			// Assert
			if got != tt.want {
				t.Errorf("slugify(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestMatchCanonicalRubric(t *testing.T) {
	tests := []struct {
		name  string
		in    string
		want  string
		found bool
	}{
		{"exact", "Posologie et mode d'administration", "4.2", true},
		{"case and accents", "DÉNOMINATION DU MÉDICAMENT", "1", true},
		{"containment absorbs variations", "Mises en garde spéciales et précautions d'emploi liées au traitement", "4.4", true},
		{"titulaire", "Titulaire de l'autorisation de mise sur le marché", "7", true},
		{"no match", "Fabricant", "", false},
		{"too short for containment", "XY", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			got, found := matchCanonicalRubric(tt.in)

			// Assert
			if found != tt.found || got != tt.want {
				t.Errorf("matchCanonicalRubric(%q) = (%q, %v), want (%q, %v)", tt.in, got, found, tt.want, tt.found)
			}
		})
	}
}

func TestRCPSectionID(t *testing.T) {
	tests := []struct {
		name  string
		title string
		want  string
	}{
		{"leading sub number", "4.2. Posologie et mode d'administration", "4.2"},
		{"leading top number", "1. DENOMINATION DU MEDICAMENT", "1"},
		{"parenthesized number", "(4.3) Contre-indications", "4.3"},
		{"two digit number", "10. DATE DE MISE A JOUR DU TEXTE", "10"},
		{"unnumbered canonical", "DONNEES CLINIQUES", "4"},
		{"unnumbered canonical with accents", "Indications thérapeutiques", "4.1"},
		{"unknown falls back to slug", "Autres informations", "autres-informations"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			got := rcpSectionID(tt.title)

			// Assert
			if got != tt.want {
				t.Errorf("rcpSectionID(%q) = %q, want %q", tt.title, got, tt.want)
			}
		})
	}
}

func TestIDAssignerDeduplicates(t *testing.T) {
	// Arrange
	a := newIDAssigner(TypeNotice)

	// Act
	first := a.assign("Encadré")
	second := a.assign("Encadré")
	third := a.assign("Encadré")

	// Assert
	if first != "encadre" {
		t.Errorf("first ID = %q, want %q", first, "encadre")
	}
	if second != "encadre-2" {
		t.Errorf("second ID = %q, want %q", second, "encadre-2")
	}
	if third != "encadre-3" {
		t.Errorf("third ID = %q, want %q", third, "encadre-3")
	}
}

// --- Extractor -----------------------------------------------------------

func TestExtractBothTabsRCP(t *testing.T) {
	// Arrange
	page := loadFixture(t, "both_tabs.html")

	// Act
	doc, err := Extract("60016308", TypeRCP, page)

	// Assert
	if err != nil {
		t.Fatalf("Extract() error = %v", err)
	}
	if doc.CIS != "60016308" {
		t.Errorf("CIS = %q, want %q", doc.CIS, "60016308")
	}
	if doc.Type != TypeRCP {
		t.Errorf("Type = %q, want %q", doc.Type, TypeRCP)
	}
	if want := "CARVEDILOL VIATRIS 25 mg, comprimé pelliculé sécable"; doc.Titre != want {
		t.Errorf("Titre = %q, want %q", doc.Titre, want)
	}
	if doc.MiseAJour != "2025-11-07" {
		t.Errorf("MiseAJour = %q, want %q", doc.MiseAJour, "2025-11-07")
	}
	if doc.Source != SourceAttribution {
		t.Errorf("Source = %q, want %q", doc.Source, SourceAttribution)
	}

	// Canonical numbering; empty parents 4, 5 and 6 are dropped.
	assertIDs(t, doc, []string{
		"1", "2", "3",
		"4.1", "4.2", "4.3",
		"5.1", "5.2",
		"6.1",
		"7", "8", "9", "10",
	})

	sec1 := findSection(t, doc, "1")
	if want := "1. DENOMINATION DU MEDICAMENT"; sec1.Titre != want {
		t.Errorf("section 1 titre = %q, want %q", sec1.Titre, want)
	}
	if want := "<p>CARVEDILOL VIATRIS 25 mg, comprimé pelliculé sécable</p>"; sec1.Contenu != want {
		t.Errorf("section 1 contenu = %q, want %q", sec1.Contenu, want)
	}

	sec42 := findSection(t, doc, "4.2")
	// The HTML serializer escapes apostrophes as &#39;; assert on the
	// unescaped content so expectations stay human-readable.
	sec42Contenu := html.UnescapeString(sec42.Contenu)
	for _, want := range []string{
		`<h4>Posologie</h4>`,
		`<h4>Mode d'administration</h4>`,
		`<table`,
		`colspan="2"`,
		`rowspan="1"`,
		"3,125 mg",
		"Population",
		"le guide",
	} {
		if !strings.Contains(sec42Contenu, want) {
			t.Errorf("section 4.2 contenu missing %q:\n%s", want, sec42.Contenu)
		}
	}
	for _, forbidden := range []string{"<script", "window.track", "onclick", "example.com", "<a ", "class=", "style="} {
		if strings.Contains(sec42.Contenu, forbidden) {
			t.Errorf("section 4.2 contenu contains forbidden %q", forbidden)
		}
	}

	// Page furniture never leaks into any section.
	assertNotInAnyContenu(t, doc,
		"Redirection vers le haut de page",
		"Mis à jour le",
		"fr-sidemenu",
		"Sommaire",
		"class=",
	)
}

func TestExtractBothTabsNotice(t *testing.T) {
	// Arrange
	page := loadFixture(t, "both_tabs.html")

	// Act
	doc, err := Extract("60016308", TypeNotice, page)

	// Assert
	if err != nil {
		t.Fatalf("Extract() error = %v", err)
	}
	if doc.MiseAJour != "2026-01-15" {
		t.Errorf("MiseAJour = %q, want %q (per-panel date)", doc.MiseAJour, "2026-01-15")
	}

	// Slugified heading IDs (notice rubric structure).
	assertIDs(t, doc, []string{
		"denomination-du-medicament",
		"encadre",
		"que-contient-cette-notice",
		"1-qu-est-ce-que-carvedilol-viatris-25-mg-comprime-pellicule-secable-et-dans-quels-cas-est-il-utilise",
		"2-quelles-sont-les-informations-a-connaitre-avant-de-prendre-carvedilol-viatris-25-mg-comprime-pellicule-secable",
		"3-comment-prendre-carvedilol-viatris-25-mg-comprime-pellicule-secable",
	})

	first := doc.Sections[0]
	if want := "Dénomination du médicament"; first.Titre != want {
		t.Errorf("first section titre = %q, want %q", first.Titre, want)
	}
	encadre := findSection(t, doc, "encadre")
	if !strings.Contains(encadre.Contenu, "Veuillez lire attentivement") {
		t.Errorf("encadré contenu missing warning text:\n%s", encadre.Contenu)
	}

	sec2 := findSection(t, doc,
		"2-quelles-sont-les-informations-a-connaitre-avant-de-prendre-carvedilol-viatris-25-mg-comprime-pellicule-secable")
	if !strings.Contains(sec2.Contenu, "<h4>Ne prenez jamais") {
		t.Errorf("section 2 contenu missing demoted <h4> subheading:\n%s", sec2.Contenu)
	}
	if !strings.Contains(sec2.Contenu, `<th rowspan="2">`) {
		t.Errorf("section 2 contenu missing rowspan table header:\n%s", sec2.Contenu)
	}

	assertNotInAnyContenu(t, doc, "Redirection vers le haut de page", "Mis à jour le", "fr-sidemenu", "Sommaire")
}

func TestExtractMissingNoticeTab(t *testing.T) {
	// Arrange
	page := loadFixture(t, "missing_notice.html")

	// Act
	_, noticeErr := Extract("60016308", TypeNotice, page)
	rcp, rcpErr := Extract("60016308", TypeRCP, page)

	// Assert
	if !errors.Is(noticeErr, ErrNotAvailable) {
		t.Fatalf("Extract(notice) error = %v, want ErrNotAvailable", noticeErr)
	}
	if rcpErr != nil {
		t.Fatalf("Extract(rcp) error = %v", rcpErr)
	}
	if rcp.MiseAJour != "2025-11-07" {
		t.Errorf("rcp MiseAJour = %q, want %q", rcp.MiseAJour, "2025-11-07")
	}
	assertIDs(t, rcp, []string{"1", "3", "4.1", "10"})
}

func TestExtractMissingRCPTab(t *testing.T) {
	// Arrange
	page := loadFixture(t, "missing_rcp.html")

	// Act
	_, rcpErr := Extract("60016308", TypeRCP, page)
	notice, noticeErr := Extract("60016308", TypeNotice, page)

	// Assert
	if !errors.Is(rcpErr, ErrNotAvailable) {
		t.Fatalf("Extract(rcp) error = %v, want ErrNotAvailable", rcpErr)
	}
	if noticeErr != nil {
		t.Fatalf("Extract(notice) error = %v", noticeErr)
	}
	if notice.MiseAJour != "2026-01-15" {
		t.Errorf("notice MiseAJour = %q, want %q", notice.MiseAJour, "2026-01-15")
	}
	assertIDs(t, notice, []string{
		"denomination-du-medicament",
		"encadre",
		"3-comment-prendre-carvedilol-viatris-25-mg-comprime-pellicule-secable",
	})
}

func TestExtractMalformedPage(t *testing.T) {
	// Arrange
	page := loadFixture(t, "malformed.html")

	for _, docType := range []string{TypeRCP, TypeNotice} {
		t.Run(docType, func(t *testing.T) {
			// Act
			_, err := Extract("60016308", docType, page)

			// Assert: a malformed page is an error, never a tombstone.
			if err == nil {
				t.Fatal("Extract() error = nil, want parse error")
			}
			if errors.Is(err, ErrNotAvailable) {
				t.Fatalf("Extract() error = %v, must not be ErrNotAvailable on malformed input", err)
			}
		})
	}
}

func TestExtractEmptyPage(t *testing.T) {
	for _, input := range [][]byte{nil, {}} {
		// Act
		_, err := Extract("60016308", TypeRCP, input)

		// Assert
		if err == nil {
			t.Fatal("Extract() error = nil, want parse error for empty page")
		}
		if errors.Is(err, ErrNotAvailable) {
			t.Fatalf("Extract() error = %v, must not be ErrNotAvailable on empty input", err)
		}
	}
}

func TestExtractUnsupportedDocType(t *testing.T) {
	// Arrange
	page := loadFixture(t, "both_tabs.html")

	// Act
	_, err := Extract("60016308", "fiche", page)

	// Assert
	if err == nil {
		t.Fatal("Extract() error = nil, want unsupported docType error")
	}
	if !strings.Contains(err.Error(), "unsupported docType") {
		t.Errorf("Extract() error = %v, want unsupported docType message", err)
	}
}

func TestExtractRCPUnnumberedHeadings(t *testing.T) {
	// Arrange: old-generation markup (AmmNomRubrique) without numbers; the
	// canonical table must supply the stable IDs.
	page := []byte(`<!doctype html><html><body><main>
	  <h3>MEDICAMENT TEST 1 mg, comprimé</h3>
	  <div id="tabpanel-notice-panel">
	    <p class="DateNotif">ANSM - Mis à jour le : 01/02/2026</p>
	    <p class="AmmCorpsTexte">notice</p>
	  </div>
	  <div id="tabpanel-rcp-panel">
	    <p class="DateNotif">ANSM - Mis à jour le : 07/11/2025</p>
	    <p class="AmmNomRubrique">Dénomination du médicament</p>
	    <p class="AmmCorpsTexte">MEDICAMENT TEST 1 mg, comprimé.</p>
	    <p class="AmmNomRubrique">Posologie et mode d'administration</p>
	    <p class="AmmCorpsTexte">Voie orale.</p>
	    <p class="AmmNomRubrique">Rubrique inconnue personnalisée</p>
	    <p class="AmmCorpsTexte">Contenu.</p>
	  </div>
	</main></body></html>`)

	// Act
	doc, err := Extract("12345678", TypeRCP, page)

	// Assert
	if err != nil {
		t.Fatalf("Extract() error = %v", err)
	}
	if want := "MEDICAMENT TEST 1 mg, comprimé"; doc.Titre != want {
		t.Errorf("Titre = %q, want %q", doc.Titre, want)
	}
	assertIDs(t, doc, []string{"1", "4.2", "rubrique-inconnue-personnalisee"})
}

// --- Fetcher -------------------------------------------------------------

func TestFetcherSendsUserAgent(t *testing.T) {
	// Arrange
	var mu sync.Mutex
	var gotUA string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		gotUA = r.Header.Get("User-Agent")
		mu.Unlock()
		w.Write(loadFixture(t, "both_tabs.html"))
	}))
	t.Cleanup(ts.Close)

	ua := "medicaments-api/1.2.3-test (+https://medicaments-api.giygas.dev)"
	f := newTestFetcher(ts.URL, func(c *Config) { c.UserAgent = ua })

	// Act
	if _, _, err := f.Fetch(context.Background(), "60016308", TypeRCP); err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}

	// Assert
	mu.Lock()
	defer mu.Unlock()
	if gotUA != ua {
		t.Errorf("User-Agent = %q, want %q", gotUA, ua)
	}
}

func TestFetcherHappyPath(t *testing.T) {
	// Arrange
	page := loadFixture(t, "both_tabs.html")
	var mu sync.Mutex
	var paths []string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		paths = append(paths, r.URL.Path)
		mu.Unlock()
		w.Header().Set("Content-Type", "text/html")
		w.Write(page)
	}))
	t.Cleanup(ts.Close)
	f := newTestFetcher(ts.URL, nil)

	// Act
	payload, date, err := f.Fetch(context.Background(), "60016308", TypeRCP)

	// Assert
	if err != nil {
		t.Fatalf("Fetch(rcp) error = %v", err)
	}
	if date != "2025-11-07" {
		t.Errorf("rcp source date = %q, want %q", date, "2025-11-07")
	}
	var doc Document
	if err := json.Unmarshal(payload, &doc); err != nil {
		t.Fatalf("payload is not valid document JSON: %v", err)
	}
	if doc.CIS != "60016308" || doc.Type != TypeRCP || len(doc.Sections) == 0 {
		t.Errorf("unexpected document envelope: %+v", doc)
	}
	// The Etalab 2.0 attribution fields are mandatory in the served JSON.
	var probe map[string]any
	if err := json.Unmarshal(payload, &probe); err != nil {
		t.Fatalf("payload is not a JSON object: %v", err)
	}
	if probe["source"] != SourceAttribution {
		t.Errorf("payload source = %v, want %q", probe["source"], SourceAttribution)
	}
	if probe["miseAJour"] != "2025-11-07" {
		t.Errorf("payload miseAJour = %v, want %q", probe["miseAJour"], "2025-11-07")
	}
	if _, ok := probe["sections"].([]any); !ok {
		t.Errorf("payload sections missing or empty: %v", probe["sections"])
	}
	mu.Lock()
	if len(paths) != 1 || paths[0] != "/60016308/extrait" {
		t.Errorf("requested paths = %v, want [/60016308/extrait]", paths)
	}
	mu.Unlock()

	// Act
	_, noticeDate, err := f.Fetch(context.Background(), "60016308", TypeNotice)

	// Assert
	if err != nil {
		t.Fatalf("Fetch(notice) error = %v", err)
	}
	if noticeDate != "2026-01-15" {
		t.Errorf("notice source date = %q, want %q", noticeDate, "2026-01-15")
	}
}

func TestFetcherHTTP404YieldsNotAvailable(t *testing.T) {
	// Arrange
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	t.Cleanup(ts.Close)
	f := newTestFetcher(ts.URL, nil)

	// Act
	_, _, err := f.Fetch(context.Background(), "60016308", TypeRCP)

	// Assert
	if !errors.Is(err, ErrNotAvailable) {
		t.Fatalf("Fetch() error = %v, want ErrNotAvailable", err)
	}
}

func TestFetcherRetriesOn429AndHonorsRetryAfter(t *testing.T) {
	// Arrange
	page := loadFixture(t, "both_tabs.html")
	var mu sync.Mutex
	attempts := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		attempts++
		current := attempts
		mu.Unlock()
		if current == 1 {
			w.Header().Set("Retry-After", "0")
			http.Error(w, "slow down", http.StatusTooManyRequests)
			return
		}
		w.Write(page)
	}))
	t.Cleanup(ts.Close)
	f := newTestFetcher(ts.URL, func(c *Config) {
		c.Backoff = time.Millisecond
		c.MaxRetries = 1
	})

	// Act
	payload, date, err := f.Fetch(context.Background(), "60016308", TypeRCP)

	// Assert
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
	if date != "2025-11-07" {
		t.Errorf("source date = %q, want %q", date, "2025-11-07")
	}
	if len(payload) == 0 {
		t.Error("payload is empty")
	}
	mu.Lock()
	defer mu.Unlock()
	if attempts != 2 {
		t.Errorf("upstream attempts = %d, want 2 (initial + 1 retry)", attempts)
	}
}

func TestFetcherExhaustsRetries(t *testing.T) {
	// Arrange
	var mu sync.Mutex
	attempts := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		attempts++
		mu.Unlock()
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}))
	t.Cleanup(ts.Close)
	f := newTestFetcher(ts.URL, func(c *Config) { c.Backoff = time.Millisecond; c.MaxRetries = 1 })

	// Act
	_, _, err := f.Fetch(context.Background(), "60016308", TypeRCP)

	// Assert
	if err == nil {
		t.Fatal("Fetch() error = nil, want failure")
	}
	if errors.Is(err, ErrNotAvailable) {
		t.Fatalf("Fetch() error = %v, a 503 must not look like a tombstone", err)
	}
	if !strings.Contains(err.Error(), "2 attempts") {
		t.Errorf("error = %v, want mention of 2 attempts", err)
	}
	mu.Lock()
	defer mu.Unlock()
	if attempts != 2 {
		t.Errorf("upstream attempts = %d, want 2", attempts)
	}
}

func TestFetcherUnexpectedStatusNotRetried(t *testing.T) {
	// Arrange
	var mu sync.Mutex
	attempts := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		attempts++
		mu.Unlock()
		http.Error(w, "forbidden", http.StatusForbidden)
	}))
	t.Cleanup(ts.Close)
	f := newTestFetcher(ts.URL, nil)

	// Act
	_, _, err := f.Fetch(context.Background(), "60016308", TypeRCP)

	// Assert
	if err == nil {
		t.Fatal("Fetch() error = nil, want failure")
	}
	if !strings.Contains(err.Error(), "unexpected HTTP status 403") {
		t.Errorf("error = %v, want unexpected HTTP status 403", err)
	}
	mu.Lock()
	defer mu.Unlock()
	if attempts != 1 {
		t.Errorf("upstream attempts = %d, want 1 (403 is not retryable)", attempts)
	}
}

func TestFetcherBodyTooLarge(t *testing.T) {
	// Arrange
	big := []byte(strings.Repeat("a", 4096))
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(big)
	}))
	t.Cleanup(ts.Close)
	f := newTestFetcher(ts.URL, func(c *Config) { c.MaxBodyBytes = 512; c.Backoff = time.Millisecond })

	// Act
	_, _, err := f.Fetch(context.Background(), "60016308", TypeRCP)

	// Assert
	if err == nil {
		t.Fatal("Fetch() error = nil, want body size failure")
	}
	if !strings.Contains(err.Error(), "exceeds") {
		t.Errorf("error = %v, want body size mention", err)
	}
}

func TestFetcherValidatesInput(t *testing.T) {
	// Arrange
	tests := []struct {
		name   string
		cis    string
		docTyp string
	}{
		{"non-numeric cis", "abcdef", TypeRCP},
		{"empty cis", "", TypeRCP},
		{"too long cis", "12345678901", TypeRCP},
		{"unsupported docType", "60016308", "fiche"},
	}
	f := newTestFetcher("", nil)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			_, _, err := f.Fetch(context.Background(), tt.cis, tt.docTyp)

			// Assert: rejected before any network activity.
			if err == nil {
				t.Fatal("Fetch() error = nil, want validation error")
			}
			if !strings.Contains(err.Error(), "invalid") {
				t.Errorf("error = %v, want invalid input message", err)
			}
		})
	}
}

func TestFetcherRateLimiterRespectsContext(t *testing.T) {
	// Arrange
	page := loadFixture(t, "both_tabs.html")
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(page)
	}))
	t.Cleanup(ts.Close)
	// Capacity 1 at 0.5 req/s: the first request consumes the initial
	// token, the second must wait ~2s for a new one.
	f := NewFetcher(Config{BaseURL: ts.URL, RatePerSec: 0.5})

	// Act
	if _, _, err := f.Fetch(context.Background(), "60016308", TypeRCP); err != nil {
		t.Fatalf("first Fetch() error = %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, _, err := f.Fetch(ctx, "12345678", TypeRCP)

	// Assert
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("second Fetch() error = %v, want context.Canceled", err)
	}
}

func TestFetcherSingleflightDeduplicates(t *testing.T) {
	// Arrange
	page := loadFixture(t, "both_tabs.html")
	var mu sync.Mutex
	hits := 0
	release := make(chan struct{})
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		hits++
		mu.Unlock()
		<-release
		w.Write(page)
	}))
	t.Cleanup(ts.Close)
	f := newTestFetcher(ts.URL, nil)

	const callers = 5
	type outcome struct {
		payload []byte
		date    string
		err     error
	}
	outcomes := make([]outcome, callers)
	started := make(chan struct{}, callers)

	var wg sync.WaitGroup
	for i := 0; i < callers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			started <- struct{}{}
			payload, date, err := f.Fetch(context.Background(), "60016308", TypeRCP)
			outcomes[i] = outcome{payload: payload, date: date, err: err}
		}(i)
	}

	// Act: let every caller enter Fetch (joining the singleflight group
	// while the single upstream request is parked), then release it.
	for i := 0; i < callers; i++ {
		<-started
	}
	time.Sleep(100 * time.Millisecond)
	close(release)
	wg.Wait()

	// Assert
	mu.Lock()
	defer mu.Unlock()
	if hits != 1 {
		t.Fatalf("upstream hits = %d, want exactly 1 (singleflight)", hits)
	}
	for i, o := range outcomes {
		if o.err != nil {
			t.Errorf("caller %d error: %v", i, o.err)
			continue
		}
		if o.date != "2025-11-07" {
			t.Errorf("caller %d date = %q, want %q", i, o.date, "2025-11-07")
		}
		if len(o.payload) == 0 {
			t.Errorf("caller %d got empty payload", i)
		}
		if string(o.payload) != string(outcomes[0].payload) {
			t.Errorf("caller %d payload differs from caller 0", i)
		}
	}
}

// --- Fetcher helpers -----------------------------------------------------

func TestParseRetryAfter(t *testing.T) {
	// Arrange
	now := time.Date(2026, 9, 21, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name  string
		value string
		min   time.Duration // inclusive lower bound
		max   time.Duration // exclusive upper bound
	}{
		{"empty", "", 0, 1},
		{"zero seconds", "0", 0, 1},
		{"negative seconds", "-5", 0, 1},
		{"garbage", "soon", 0, 1},
		{"delay seconds", "120", 120 * time.Second, 121 * time.Second},
		{"http date in the future", now.Add(30 * time.Second).Format(http.TimeFormat), 25 * time.Second, 35 * time.Second},
		{"http date in the past", now.Add(-time.Hour).Format(http.TimeFormat), 0, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			got := parseRetryAfter(tt.value, now)

			// Assert
			if got < tt.min || got >= tt.max {
				t.Errorf("parseRetryAfter(%q) = %v, want in [%v, %v)", tt.value, got, tt.min, tt.max)
			}
		})
	}
}

func TestNewFetcherAppliesDefaults(t *testing.T) {
	// Arrange / Act
	f := NewFetcher(Config{})

	// Assert
	if f.cfg.BaseURL != DefaultBaseURL {
		t.Errorf("BaseURL = %q, want %q", f.cfg.BaseURL, DefaultBaseURL)
	}
	if f.cfg.UserAgent != DefaultUserAgent {
		t.Errorf("UserAgent = %q, want %q", f.cfg.UserAgent, DefaultUserAgent)
	}
	if f.cfg.Timeout != DefaultTimeout {
		t.Errorf("Timeout = %v, want %v", f.cfg.Timeout, DefaultTimeout)
	}
	if f.cfg.MaxBodyBytes != DefaultMaxBodyBytes {
		t.Errorf("MaxBodyBytes = %d, want %d", f.cfg.MaxBodyBytes, DefaultMaxBodyBytes)
	}
	if f.limiter == nil {
		t.Error("limiter = nil, want default global rate limiter")
	}
	if f.client == nil {
		t.Fatal("client = nil, want default HTTP client")
	}
	if f.client.Timeout != DefaultTimeout {
		t.Errorf("client timeout = %v, want %v", f.client.Timeout, DefaultTimeout)
	}

	// Negative rate disables the limiter (test escape hatch).
	if g := NewFetcher(Config{RatePerSec: -1}); g.limiter != nil {
		t.Error("limiter = non-nil for negative RatePerSec, want disabled")
	}
}
