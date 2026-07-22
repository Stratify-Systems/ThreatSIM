package unit

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/suryatk2007/threatsim/pkg/plugins/utils/cors"
)

func TestRunCORSAuditValidation_SecurePolicy(t *testing.T) {
	t.Parallel()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// SECURE! Rejects untrusted origins by omitting Access-Control-Allow-Origin
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data":"secure"}`))
	}))
	defer ts.Close()

	cfg := cors.CORSAuditConfig{
		BaseURL:                  ts.URL,
		Path:                     "/api/profile",
		CustomOrigin:             "https://attacker.com",
		TestNullOrigin:           true,
		ExpectedAllowCredentials: false,
		Client:                   ts.Client(),
	}

	results := cors.RunCORSAuditValidation("CORS Audit Test", cfg)
	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}

	if !results[0].Passed {
		t.Errorf("Expected CORS test to pass on secure server, failed with reason: %s", results[0].Reason)
	}
}

func TestRunCORSAuditValidation_InsecurePolicy(t *testing.T) {
	t.Parallel()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		// INSECURE! Reflects any incoming origin and allows credentials
		if origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data":"vulnerable"}`))
	}))
	defer ts.Close()

	cfg := cors.CORSAuditConfig{
		BaseURL:      ts.URL,
		Path:         "/api/profile",
		CustomOrigin: "https://attacker.com",
		Client:       ts.Client(),
	}

	results := cors.RunCORSAuditValidation("CORS Audit Test", cfg)
	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}

	if results[0].Passed {
		t.Errorf("Expected CORS audit to fail on insecure origin reflection server, but passed")
	}
}
