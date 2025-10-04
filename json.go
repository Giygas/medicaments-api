package main

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/giygas/medicaments-api/logging"
)

// Minimum response size to consider compression (1KB)
const compressionThreshold = 1024

func respondWithError(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Last-Modified", time.Now().UTC().Format(http.TimeFormat))
	w.WriteHeader(code)

	// Create a map to hold the error message
	errorResponse := map[string]string{"error": msg}

	// Marshal the map into JSON
	jsonResponse, err := json.Marshal(errorResponse)
	if err != nil {
		// If there's an error, log it and return a generic error message
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Error responses are typically small, so don't compress them
	w.Write(jsonResponse)
	logging.Debug("Sent error response", "size", len(jsonResponse), "compressed", false)
}
