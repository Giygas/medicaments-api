package main

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/giygas/medicaments-api/logging"
)

// Minimum response size to consider compression (1KB)
const compressionThreshold = 1024

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {

	data, err := json.Marshal(payload)
	if err != nil {
		logging.Error("Failed to marshal JSON response", "error", err, "payload_type", fmt.Sprintf("%T", payload))
		w.WriteHeader(500)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Last-Modified", time.Now().UTC().Format(http.TimeFormat))
	w.WriteHeader(code)

	// Check if compression should be used
	dataSize := len(data)
	shouldCompress := dataSize >= compressionThreshold &&
		strings.Contains(strings.ToLower(w.Header().Get("Accept-Encoding")), "gzip")

	if shouldCompress {
		w.Header().Set("Content-Encoding", "gzip")
		gz := gzip.NewWriter(w)
		defer gz.Close()
		gz.Write(data)
		logging.Debug("Compressed JSON response",
			"original_size", dataSize,
			"compressed", true)
	} else {
		w.Write(data)
		logging.Debug("Sent uncompressed JSON response",
			"original_size", dataSize,
			"compressed", false,
			"above_threshold", dataSize >= compressionThreshold,
			"accepts_gzip", strings.Contains(strings.ToLower(w.Header().Get("Accept-Encoding")), "gzip"))
	}
}

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
