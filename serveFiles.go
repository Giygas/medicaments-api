package main

import (
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/giygas/medicaments-api/logging"
	"github.com/giygas/medicaments-api/medicamentsparser/entities"
	"github.com/go-chi/chi/v5"
)

// Helper function to check cached versions
func checkCachedVersion(w http.ResponseWriter, r *http.Request, lastUpdated time.Time) bool {
	// Check If-Modified-Since
	if ims := r.Header.Get("If-Modified-Since"); ims != "" {
		if t, err := time.Parse(http.TimeFormat, ims); err == nil {
			if !lastUpdated.After(t) {
				w.WriteHeader(http.StatusNotModified)
				return true
			}
		}
	}
	// Check If-None-Match (ETag)
	if inm := r.Header.Get("If-None-Match"); inm != "" {
		etag := `"` + strconv.FormatInt(lastUpdated.Unix(), 10) + `"`
		if inm == etag {
			w.WriteHeader(http.StatusNotModified)
			return true
		}
	}

	return false
}

func serveAllMedicaments(w http.ResponseWriter, r *http.Request) {
	medicaments := GetMedicaments()
	lastUpdated := GetLastUpdated()

	// Set caching headers
	w.Header().Set("Cache-Control", "public, max-age=21600") // 6 hours
	w.Header().Set("Last-Modified", lastUpdated.UTC().Format(http.TimeFormat))
	w.Header().Set("ETag", `"`+strconv.FormatInt(lastUpdated.Unix(), 10)+`"`)

	// Check if client has cached version
	if checkCachedVersion(w, r, lastUpdated) {
		return
	}

	respondWithJSON(w, 200, medicaments)
}

func servePagedMedicaments(w http.ResponseWriter, r *http.Request) {

	pageNumber := chi.URLParam(r, "pageNumber")

	page, err := strconv.Atoi(pageNumber)

	if err != nil || page < 1 {
		logging.Warn("Unusual user input", "pageNumber", pageNumber)
		respondWithError(w, 400, "Invalid page number")
		return
	}

	medicaments := GetMedicaments()
	lastUpdated := GetLastUpdated()

	// Set caching headers
	w.Header().Set("Cache-Control", "public, max-age=21600")
	w.Header().Set("Last-Modified", lastUpdated.UTC().Format(http.TimeFormat))

	totalItems := len(medicaments)
	pageSize := 10
	maxPage := (totalItems + pageSize - 1) / pageSize // Ceiling division

	if page > maxPage {
		respondWithError(w, 404, "Maximum page possible is: "+strconv.Itoa(maxPage))
		return
	}

	start := (page - 1) * pageSize
	end := min(start+pageSize, totalItems)

	result := struct {
		Data       []entities.Medicament `json:"data"`
		Page       int                   `json:"page"`
		PageSize   int                   `json:"pageSize"`
		TotalItems int                   `json:"totalItems"`
		MaxPage    int                   `json:"maxPage"`
	}{
		Data:       medicaments[start:end],
		Page:       page,
		PageSize:   pageSize,
		TotalItems: totalItems,
		MaxPage:    maxPage,
	}

	respondWithJSON(w, 200, result)
}

func findMedicament(w http.ResponseWriter, r *http.Request) {

	userPattern := chi.URLParam(r, "element")
	if len(userPattern) < 3 {
		respondWithError(w, 400, "Search term must be at least 3 characters")
		return
	}

	if len(userPattern) > 50 || !regexp.MustCompile(`^[a-zA-Z0-9 ]+$`).MatchString(userPattern) {
		logging.Warn("Suspicious user pattern", "userPattern", userPattern)
		respondWithError(w, 400, "Invalid search term: must be alphanumeric with spaces, max 50 chars")
		return
	}

	pattern, err := regexp.Compile(`(?i)` + regexp.QuoteMeta(userPattern))
	if err != nil {
		respondWithError(w, 400, "Invalid search pattern")
		return
	}

	medicaments := GetMedicaments()
	var matchingMedicaments []entities.Medicament

	for _, med := range medicaments {
		if pattern.MatchString(med.Denomination) {
			matchingMedicaments = append(matchingMedicaments, med)
		}
	}

	if len(matchingMedicaments) == 0 {
		respondWithError(w, 404, "No medicaments found matching the search term")
		return
	}

	// Set shorter cache for search results
	w.Header().Set("Cache-Control", "public, max-age=3600") // 1 hour

	result := struct {
		Results []entities.Medicament `json:"results"`
		Count   int                   `json:"count"`
		// Limited bool                  `json:"limited"`   -- In case of limiting results
	}{
		Results: matchingMedicaments,
		Count:   len(matchingMedicaments),
		// Limited: len(matchingMedicaments) == maxResults,
	}

	respondWithJSON(w, 200, result)

}

func findMedicamentByID(w http.ResponseWriter, r *http.Request) {
	cis, err := strconv.Atoi(chi.URLParam(r, "cis"))

	if err != nil || cis <= 0 || cis >= 1000000000 {
		respondWithError(w, 400, "Invalid CIS: must be a positive integer less than 1.000.000.000")
		return
	}

	medicamentsMap := GetMedicamentsMap()
	medicament, exists := medicamentsMap[cis]

	if !exists {
		respondWithError(w, 404, "Medicament not found")
		return
	}

	// Set caching headers for individual medicaments
	lastUpdated := GetLastUpdated()
	w.Header().Set("Cache-Control", "public, max-age=43200") // 12 hours
	w.Header().Set("Last-Modified", lastUpdated.UTC().Format(http.TimeFormat))

	respondWithJSON(w, 200, medicament)
}

func findGeneriques(w http.ResponseWriter, r *http.Request) {

	userPattern := chi.URLParam(r, "libelle")

	if len(userPattern) < 3 {
		respondWithError(w, 400, "Search term must be at least 3 characters")
		return
	}

	if len(userPattern) > 50 || !regexp.MustCompile(`^[a-zA-Z0-9 ]+$`).MatchString(userPattern) {
		logging.Warn("Suspicious user pattern", "userPattern", userPattern)
		respondWithError(w, 400, "Invalid search term: must be alphanumeric with spaces, max 50 chars")
		return
	}

	pattern, err := regexp.Compile(`(?i)` + regexp.QuoteMeta(userPattern))
	if err != nil {
		respondWithError(w, 400, "Invalid search pattern")
		return
	}

	generiques := GetGeneriques()
	var matchingGeneriques []entities.GeneriqueList

	for _, gen := range generiques {
		if pattern.MatchString(gen.Libelle) {
			matchingGeneriques = append(matchingGeneriques, gen)
		}
	}

	if len(matchingGeneriques) == 0 {
		respondWithError(w, 404, "No generiques found matching the search term")
		return
	}

	w.Header().Set("Cache-Control", "public, max-age=3600")
	respondWithJSON(w, 200, matchingGeneriques)

}

func findGeneriquesByGroupID(w http.ResponseWriter, r *http.Request) {
	groupID, err := strconv.Atoi(chi.URLParam(r, "groupId"))

	if err != nil {
		respondWithError(w, 400, "Invalid group ID: must be a positive integer")
		return
	}
	if groupID <= 0 || groupID >= 100000 {
		respondWithError(w, 400, "Invalid group ID: must be a positive integer less than 100000")
		return
	}

	generiquesMap := GetGeneriquesMap()
	generique, exists := generiquesMap[groupID]

	if !exists {
		respondWithError(w, 404, "Generique group not found")
		return
	}

	lastUpdated := GetLastUpdated()
	w.Header().Set("Cache-Control", "public, max-age=43200")
	w.Header().Set("Last-Modified", lastUpdated.UTC().Format(http.TimeFormat))

	respondWithJSON(w, 200, generique)
}
