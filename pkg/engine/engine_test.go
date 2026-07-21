package engine

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/suryatk2007/threatsim/pkg/types"
)

func TestExtractJSONPath(t *testing.T) {
	data := map[string]interface{}{
		"user": map[string]interface{}{
			"id": 42,
			"profile": map[string]interface{}{
				"token": "secret123",
			},
		},
		"status": "success",
	}

	tests := []struct {
		path     string
		expected interface{}
		found    bool
	}{
		{"status", "success", true},
		{"user.id", 42, true},
		{"user.profile.token", "secret123", true},
		{"user.profile.invalid", nil, false},
		{"invalid.path", nil, false},
	}

	for _, tt := range tests {
		val, ok := extractJSONPath(data, tt.path)
		if ok != tt.found {
			t.Errorf("Path %q: expected found=%v, got %v", tt.path, tt.found, ok)
		}
		if ok && val != tt.expected {
			t.Errorf("Path %q: expected val=%v, got %v", tt.path, tt.expected, val)
		}
	}
}

func TestInterpolate(t *testing.T) {
	e := New("http://localhost")
	e.State["token"] = "abc-123"
	e.State["id"] = "99"

	input := "/api/users/{{state.id}}?auth={{state.token}}"
	expected := "/api/users/99?auth=abc-123"

	result := e.interpolate(input)
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

func TestExecuteSimulationAndExtraction(t *testing.T) {
	// Create a mock HTTP server to validate our requests and return specific data
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/auth" {
			w.Header().Set("X-Session-ID", "sess-888")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"token": "jwt-token-xyz", "nested": {"id": 100}}`))
			return
		}
		if r.URL.Path == "/data/100" {
			authHeader := r.Header.Get("Authorization")
			if authHeader != "Bearer jwt-token-xyz" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Secret Data Result: sess-888"))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	e := New(server.URL)

	// Step 1: Login and Extract
	sim1 := types.Simulation{
		Name: "Login",
		Request: types.Request{
			Method: "POST",
			Path:   "/auth",
		},
		Expected: types.Expected{
			StatusCode: 200,
		},
		Extract: types.Extract{
			JSON: map[string]string{
				"jwt": "token",
				"uid": "nested.id",
			},
			Header: map[string]string{
				"session": "X-Session-Id",
			},
			Regex: map[string]string{
				"raw_token": `"token":\s*"([^"]+)"`,
			},
		},
	}

	res1 := e.executeSimulation(sim1)
	if !res1.Passed {
		t.Fatalf("Simulation 1 failed: %v", res1.Reason)
	}

	// Verify Extractions
	if e.State["jwt"] != "jwt-token-xyz" {
		t.Errorf("Expected state.jwt = jwt-token-xyz, got %v", e.State["jwt"])
	}
	if e.State["uid"] != "100" {
		t.Errorf("Expected state.uid = 100, got %v", e.State["uid"])
	}
	if e.State["session"] != "sess-888" {
		t.Errorf("Expected state.session = sess-888, got %v", e.State["session"])
	}
	if e.State["raw_token"] != "jwt-token-xyz" {
		t.Errorf("Expected state.raw_token = jwt-token-xyz, got %v", e.State["raw_token"])
	}

	// Step 2: Use Extracted state in the next request
	sim2 := types.Simulation{
		Name: "Fetch Data",
		Request: types.Request{
			Method: "GET",
			Path:   "/data/{{state.uid}}",
			Headers: map[string]string{
				"Authorization": "Bearer {{state.jwt}}",
			},
		},
		Expected: types.Expected{
			StatusCode:   200,
			BodyContains: "Secret Data Result: sess-888",
		},
	}

	res2 := e.executeSimulation(sim2)
	if !res2.Passed {
		t.Fatalf("Simulation 2 failed: %v", res2.Reason)
	}
}

func TestLoadSimulation(t *testing.T) {
	yamlContent := `
version: "1.0"
simulations:
  - name: "Test YAML"
    request:
      method: "GET"
      path: "/"
    expected:
      status_code: 200
`
	tmpFile, err := os.CreateTemp("", "threatsim_*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	tmpFile.Write([]byte(yamlContent))
	tmpFile.Close()

	e := New("http://localhost")
	def, err := e.LoadSimulation(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to load simulation: %v", err)
	}

	if len(def.Simulations) != 1 {
		t.Fatalf("Expected 1 simulation, got %d", len(def.Simulations))
	}
	if def.Simulations[0].Name != "Test YAML" {
		t.Errorf("Expected name 'Test YAML', got %q", def.Simulations[0].Name)
	}
}
