package unit

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/suryatk2007/threatsim/pkg/plugins/utils/idor"
)

func TestRunIDORValidation_MultiMethodAndBodySubstitution(t *testing.T) {
	t.Parallel()

	var receivedMethod string
	var receivedPath string
	var receivedBody string

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Auth endpoint handler
		if r.URL.Path == "/auth/login" {
			bodyBytes, _ := io.ReadAll(r.Body)
			var payload map[string]string
			json.Unmarshal(bodyBytes, &payload)

			if payload["username"] == "user_a" {
				json.NewEncoder(w).Encode(map[string]interface{}{
					"data": map[string]interface{}{
						"token": "token_user_a",
						"user":  map[string]interface{}{"id": "target_id_888"},
					},
				})
				return
			}
			if payload["username"] == "user_b" {
				json.NewEncoder(w).Encode(map[string]interface{}{
					"data": map[string]interface{}{
						"token": "token_user_b",
						"user":  map[string]interface{}{"id": "id_999"},
					},
				})
				return
			}
		}

		// Target endpoint handler
		receivedMethod = r.Method
		receivedPath = r.URL.Path
		bodyBytes, _ := io.ReadAll(r.Body)
		receivedBody = string(bodyBytes)

		// Reject cross-tenant attempt
		if r.Header.Get("Authorization") == "Bearer token_user_b" {
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte(`{"error": "Forbidden cross-tenant action"}`))
			return
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	cfg := idor.IDORConfig{
		BaseURL:              ts.URL,
		AuthPath:             "/auth/login",
		UserAPayload:         `{"username":"user_a","password":"123"}`,
		UserBPayload:         `{"username":"user_b","password":"123"}`,
		TokenJSONPath:        "data.token",
		IDJSONPath:           "data.user.id",
		TargetMethod:         "PUT",
		TargetPath:           "/api/items/{id}/update",
		TargetPayload:        `{"target_user":"{id}","status":"modified"}`,
		ExpectedStatusCode:   403,
		ExpectedBodyContains: "Forbidden",
		Client:               ts.Client(),
	}

	results := idor.RunIDORValidation("Multi-Method Body IDOR Test", cfg)
	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}

	res := results[0]
	if !res.Passed {
		t.Errorf("Expected IDOR test to pass security control check, failed with reason: %s", res.Reason)
	}

	if receivedMethod != "PUT" {
		t.Errorf("Expected HTTP method 'PUT', got %q", receivedMethod)
	}

	if receivedPath != "/api/items/target_id_888/update" {
		t.Errorf("Expected path '/api/items/target_id_888/update', got %q", receivedPath)
	}

	expectedBody := `{"target_user":"target_id_888","status":"modified"}`
	if receivedBody != expectedBody {
		t.Errorf("Expected body %q, got %q", expectedBody, receivedBody)
	}
}
