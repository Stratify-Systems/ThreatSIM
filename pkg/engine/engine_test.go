package engine

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/suryatk2007/threatsim/pkg/types"
)

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

func TestEngineInsecureTLS(t *testing.T) {
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	def := &types.SimulationDefinition{
		Simulations: []types.Simulation{
			{
				Name: "Strict TLS Test",
				Request: types.Request{
					Method: "GET",
					Path:   "/",
				},
				Expected: types.Expected{
					StatusCode: 200,
				},
			},
		},
	}

	// 1. Connection without WithInsecure(true) should fail TLS verification
	strictEng := New(ts.URL, WithInsecure(false))
	strictReport := strictEng.Execute(def)
	if strictReport.Passed != 0 {
		t.Errorf("Expected strict TLS connection to fail on self-signed cert, but got passed")
	}

	// 2. Connection with WithInsecure(true) should succeed
	insecureEng := New(ts.URL, WithInsecure(true))
	insecureReport := insecureEng.Execute(def)
	if insecureReport.Passed != 1 {
		t.Errorf("Expected insecure TLS connection to pass, but got failed: %v", insecureReport.Results[0].Reason)
	}
}

func TestEngineTimeout(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(150 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	// Engine configured with 50ms default timeout
	shortEng := New(ts.URL, WithTimeout(50*time.Millisecond))
	def := &types.SimulationDefinition{
		Simulations: []types.Simulation{
			{
				Name: "Timeout Test",
				Request: types.Request{
					Method: "GET",
					Path:   "/",
				},
				Expected: types.Expected{
					StatusCode: 200,
				},
			},
		},
	}
	report := shortEng.Execute(def)
	if report.Passed != 0 {
		t.Errorf("Expected request to timeout and fail, but it passed")
	}

	// Per-simulation timeout override (400ms) should allow request to succeed
	overrideDef := &types.SimulationDefinition{
		Simulations: []types.Simulation{
			{
				Name: "Timeout Override Test",
				Request: types.Request{
					Method:  "GET",
					Path:    "/",
					Timeout: "400ms",
				},
				Expected: types.Expected{
					StatusCode: 200,
				},
			},
		},
	}
	overrideReport := shortEng.Execute(overrideDef)
	if overrideReport.Passed != 1 {
		t.Errorf("Expected per-simulation timeout override to succeed, but failed: %v", overrideReport.Results[0].Reason)
	}
}
