package server

import (
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestGetTokenCost(t *testing.T) {
	tests := []struct {
		name         string
		path         string
		query        string
		expectedCost int64
	}{
		// V1 Health endpoint
		{"V1 health endpoint", "/v1/health", "", 5},
		{"Health endpoint", "/health", "", 5},

		// V1 Medicaments endpoint
		{"V1 export route", "/v1/medicaments/export", "", 200},
		{"V1 export all (deprecated query param)", "/v1/medicaments", "export=all", 5},
		{"V1 page query", "/v1/medicaments", "page=1", 20},
		{"V1 search query", "/v1/medicaments", "search=paracetamol", 50},
		{"V1 CIS query", "/v1/medicaments", "cis=12345678", 10},
		{"V1 CIP query", "/v1/medicaments", "cip=1234567", 10},
		{"V1 medicaments default", "/v1/medicaments", "", 5},

		// V1 Generiques endpoint
		{"V1 generiques libelle", "/v1/generiques", "libelle=paracetamol", 30},
		{"V1 generiques group", "/v1/generiques", "group=1", 5},
		{"V1 generiques default", "/v1/generiques", "", 5},

		// V1 Presentations endpoint (now uses path parameter)
		{"V1 presentations", "/v1/presentations/1234567", "", 5},

		// Legacy endpoints (for backward compatibility)
		{"Legacy database", "/database", "", 200},
		{"Legacy database page", "/database/1", "", 20},
		{"Legacy medicament by ID", "/medicament/id/12345678", "", 10},
		{"Legacy medicament by CIP", "/medicament/cip/1234567", "", 10},
		{"Legacy medicament search", "/medicament/test", "", 80},
		{"Legacy generiques", "/generiques/test", "", 20},

		// Default case
		{"Default endpoint", "/unknown", "", 5},
		{"Root path", "/", "", 5},

		// ===== EDGE CASES =====
		// Multi-parameter scenarios (should return default 5)
		{"V1 medicaments export query+CIP (invalid)", "/v1/medicaments", "export=all&cip=1234567", 5},
		{"V1 medicaments page+search", "/v1/medicaments", "page=1&search=test", 5},
		{"V1 medicaments CIS+CIP", "/v1/medicaments", "cis=123&cip=456", 5},
		{"V1 generiques libelle+group", "/v1/generiques", "libelle=test&group=1", 5},

		// Invalid parameter values (cost based on param type, handler validates value)
		{"V1 export invalid value (query param)", "/v1/medicaments", "export=invalid", 5}, // Falls to default
		{"V1 export case insensitive", "/v1/medicaments", "export=ALL", 5},                // Case sensitive, falls to default
		{"V1 page non-numeric", "/v1/medicaments", "page=abc", 20},                        // page param present, handler will reject
		{"V1 page zero", "/v1/medicaments", "page=0", 20},                                 // page param present, handler will reject
		{"V1 search empty string", "/v1/medicaments", "search=", 5},                       // Falls to default (empty value)
		{"V1 CIS empty string", "/v1/medicaments", "cis=", 5},                             // Falls to default (empty value)
		{"V1 CIP empty string", "/v1/medicaments", "cip=", 5},                             // Falls to default (empty value)

		// Health endpoint with params (should return default 5)
		{"V1 health with params", "/v1/health", "test=value", 5},
		{"Health with params", "/health", "test=value", 5},

		// Unknown parameters on valid endpoints (should return default 5)
		{"V1 medicaments unknown param", "/v1/medicaments", "unknown=value", 5},
		{"V1 generiques unknown param", "/v1/generiques", "unknown=value", 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path+"?"+tt.query, nil)
			cost := getTokenCost(req)

			if cost != tt.expectedCost {
				t.Errorf("Expected cost %d for path %s with query %s, got %d",
					tt.expectedCost, tt.path, tt.query, cost)
			}
		})
	}
}

func TestHasSingleParam(t *testing.T) {
	tests := []struct {
		name          string
		query         string
		allowedParams []string
		expected      bool
	}{
		{"Single param present", "export=all", []string{"export", "page"}, true},
		{"Single param from list", "page=1", []string{"export", "page", "search"}, true},
		{"No params present", "", []string{"export", "page"}, false},
		{"Two params present", "export=all&page=1", []string{"export", "page"}, false},
		{"Three params present", "a=1&b=2&c=3", []string{"a", "b", "c"}, false},
		{"Param not in allowed list", "other=value", []string{"export", "page"}, false},
		{"Empty string param", "param=", []string{"param"}, false},
		{"Empty allowed list", "param=1", []string{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			values, _ := url.ParseQuery(tt.query)
			result := HasSingleParam(values, tt.allowedParams)

			if result != tt.expected {
				t.Errorf("Expected %v for query %s with allowed %v, got %v",
					tt.expected, tt.query, tt.allowedParams, result)
			}
		})
	}
}
