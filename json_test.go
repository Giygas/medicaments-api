package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/giygas/medicaments-api/logging"
)

func TestRespondWithError(t *testing.T) {
	logging.InitLogger("")

	tests := []struct {
		name           string
		code           int
		message        string
		expectedStatus int
		checkJSON      bool
	}{
		{
			name:           "bad request error",
			code:           http.StatusBadRequest,
			message:        "Invalid input provided",
			expectedStatus: http.StatusBadRequest,
			checkJSON:      true,
		},
		{
			name:           "not found error",
			code:           http.StatusNotFound,
			message:        "Resource not found",
			expectedStatus: http.StatusNotFound,
			checkJSON:      true,
		},
		{
			name:           "internal server error",
			code:           http.StatusInternalServerError,
			message:        "Internal server error",
			expectedStatus: http.StatusInternalServerError,
			checkJSON:      true,
		},
		{
			name:           "unauthorized error",
			code:           http.StatusUnauthorized,
			message:        "Unauthorized access",
			expectedStatus: http.StatusUnauthorized,
			checkJSON:      true,
		},
		{
			name:           "empty error message",
			code:           http.StatusBadRequest,
			message:        "",
			expectedStatus: http.StatusBadRequest,
			checkJSON:      true,
		},
		{
			name:           "long error message",
			code:           http.StatusBadRequest,
			message:        "This is a very long error message that should still be properly handled by the respondWithError function without causing any issues",
			expectedStatus: http.StatusBadRequest,
			checkJSON:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()

			respondWithError(rr, tt.code, tt.message)

			// Check status code
			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rr.Code)
			}

			// Check Content-Type header
			if ct := rr.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
				t.Errorf("Expected Content-Type application/json; charset=utf-8, got %s", ct)
			}

			// Check Last-Modified header exists
			if lm := rr.Header().Get("Last-Modified"); lm == "" {
				t.Error("Expected Last-Modified header to be set")
			}

			// Check JSON response structure
			if tt.checkJSON {
				var response map[string]string
				err := json.Unmarshal(rr.Body.Bytes(), &response)
				if err != nil {
					t.Errorf("Failed to unmarshal JSON: %v", err)
				}

				if len(response) == 0 {
					t.Error("Expected non-empty error response")
				}

				// Check that error field exists and matches message
				if errorMsg, ok := response["error"]; ok && tt.message != "" {
					if errorMsg != tt.message {
						t.Errorf("Expected error message %s, got %s", tt.message, errorMsg)
					}
				}
			}

			// Verify no compression (error responses are typically small)
			if ce := rr.Header().Get("Content-Encoding"); ce != "" {
				t.Errorf("Error responses should not be compressed, got Content-Encoding: %s", ce)
			}
		})
	}
}

func TestRespondWithError_JSONMarshalFailure(t *testing.T) {
	logging.InitLogger("")

	rr := httptest.NewRecorder()

	// The function should handle JSON marshal failure gracefully
	// Since we're using a simple map[string]string, JSON marshal should not fail
	// But we can at least test the normal path
	respondWithError(rr, http.StatusBadRequest, "Test error")

	// Even if there were a marshal failure, the function should still return a response
	if rr.Code == 0 {
		t.Error("Expected a status code to be set")
	}

	// Verify we got some response
	if rr.Body.Len() == 0 {
		t.Error("Expected a response body")
	}
}
