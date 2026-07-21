package engine

import (
	"os"
	"testing"
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
