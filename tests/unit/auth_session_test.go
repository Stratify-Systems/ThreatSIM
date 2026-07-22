package unit

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/suryatk2007/threatsim/pkg/plugins/utils"
)

func TestExtractJSONPath(t *testing.T) {
	jsonObj := map[string]interface{}{
		"code": 200,
		"data": map[string]interface{}{
			"token": "secret_jwt_token_123",
			"user": map[string]interface{}{
				"id": "user_999",
			},
		},
	}

	// 1. Root level
	if val, ok := utils.ExtractJSONPath(jsonObj, "code"); !ok || val != "200" {
		t.Errorf("Expected '200', got %q (ok=%v)", val, ok)
	}

	// 2. 2-level path
	if val, ok := utils.ExtractJSONPath(jsonObj, "data.token"); !ok || val != "secret_jwt_token_123" {
		t.Errorf("Expected 'secret_jwt_token_123', got %q (ok=%v)", val, ok)
	}

	// 3. 3-level nested path
	if val, ok := utils.ExtractJSONPath(jsonObj, "data.user.id"); !ok || val != "user_999" {
		t.Errorf("Expected 'user_999', got %q (ok=%v)", val, ok)
	}

	// 4. Missing path
	if _, ok := utils.ExtractJSONPath(jsonObj, "data.missing"); ok {
		t.Errorf("Expected ok=false for missing path")
	}
}

func TestAuthenticateAndExtract(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "success",
			"data": map[string]interface{}{
				"token": "token_abc123",
				"user": map[string]interface{}{
					"id": 42,
				},
			},
		})
	}))
	defer ts.Close()

	cfg := utils.AuthConfig{
		URL:           ts.URL,
		Payload:       `{"username":"test","password":"pwd"}`,
		TokenJSONPath: "data.token",
		IDJSONPath:    "data.user.id",
	}

	res, err := utils.AuthenticateAndExtract(ts.Client(), cfg)
	if err != nil {
		t.Fatalf("AuthenticateAndExtract failed: %v", err)
	}

	if res.Token != "token_abc123" {
		t.Errorf("Expected token 'token_abc123', got %q", res.Token)
	}

	if res.ID != "42" {
		t.Errorf("Expected ID '42', got %q", res.ID)
	}
}
