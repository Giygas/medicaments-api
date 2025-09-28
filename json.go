package main

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	data, err := json.Marshal(payload)
	if err != nil {
		slog.Error("Failed to marshal JSON response", "error", err, "payload_type", fmt.Sprintf("%T", payload))
		w.WriteHeader(500)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Last-Modified", time.Now().UTC().Format(http.TimeFormat))
	w.Header().Set("Content-Encoding", "gzip")
	w.WriteHeader(code)
	gz := gzip.NewWriter(w)
	defer gz.Close()
	gz.Write(data)
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

	// Write the JSON response to the client
	w.Write(jsonResponse)
}
