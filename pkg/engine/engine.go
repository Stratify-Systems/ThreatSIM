package engine

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/suryatk2007/threatsim/pkg/types"
	"gopkg.in/yaml.v3"
)

// Engine is the core execution engine responsible for loading and running simulations.
type Engine struct {
	TargetURL string
	Client    *http.Client
}

// New creates a new Engine instance.
func New(targetURL string) *Engine {
	return &Engine{
		TargetURL: strings.TrimRight(targetURL, "/"),
		Client: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// LoadSimulation reads and validates a simulation file (supports YAML and JSON).
func (e *Engine) LoadSimulation(filePath string) (*types.SimulationDefinition, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var def types.SimulationDefinition
	
	// Attempt YAML parsing first (YAML is a superset of JSON, so this works for both).
	if err := yaml.Unmarshal(data, &def); err != nil {
		// Fallback to strict JSON if YAML fails (though unlikely if it's valid JSON)
		if jsonErr := json.Unmarshal(data, &def); jsonErr != nil {
			return nil, fmt.Errorf("invalid simulation file format (must be valid YAML or JSON): %w", err)
		}
	}

	if len(def.Simulations) == 0 {
		return nil, fmt.Errorf("no simulations found in the file")
	}

	// Basic validation of each simulation
	for i, sim := range def.Simulations {
		if sim.Name == "" {
			return nil, fmt.Errorf("simulation at index %d is missing a name", i)
		}
		if sim.Request.Method == "" {
			return nil, fmt.Errorf("simulation %q is missing a request method", sim.Name)
		}
		if sim.Expected.StatusCode == 0 {
			return nil, fmt.Errorf("simulation %q is missing expected status_code", sim.Name)
		}
	}

	return &def, nil
}

// Execute runs all simulations defined in the file and returns a comprehensive validation report.
func (e *Engine) Execute(def *types.SimulationDefinition) *types.ValidationReport {
	report := &types.ValidationReport{
		TotalSimulations: len(def.Simulations),
	}

	start := time.Now()

	for _, sim := range def.Simulations {
		result := e.executeSimulation(sim)
		report.Results = append(report.Results, result)

		if result.Passed {
			report.Passed++
		} else {
			report.Failed++
		}
	}

	report.ExecutionTime = time.Since(start)
	if report.TotalSimulations > 0 {
		report.SuccessRate = (float64(report.Passed) / float64(report.TotalSimulations)) * 100
	}

	return report
}

// executeSimulation runs an individual simulation independently.
func (e *Engine) executeSimulation(sim types.Simulation) types.SimulationResult {
	start := time.Now()
	result := types.SimulationResult{
		SimulationName: sim.Name,
		ExpectedResult: fmt.Sprintf("Status Code: %d", sim.Expected.StatusCode),
	}

	// Construct full request URL
	reqURL := fmt.Sprintf("%s/%s", e.TargetURL, strings.TrimLeft(sim.Request.Path, "/"))

	// Append query parameters
	if len(sim.Request.QueryParams) > 0 {
		u, err := url.Parse(reqURL)
		if err == nil {
			q := u.Query()
			for k, v := range sim.Request.QueryParams {
				q.Add(k, v)
			}
			u.RawQuery = q.Encode()
			reqURL = u.String()
		}
	}

	var bodyReader io.Reader
	if sim.Request.Body != "" {
		bodyReader = bytes.NewBufferString(sim.Request.Body)
	}

	req, err := http.NewRequest(sim.Request.Method, reqURL, bodyReader)
	if err != nil {
		result.Passed = false
		result.ActualResult = "Request creation failed"
		result.Reason = err.Error()
		result.Duration = time.Since(start)
		return result
	}

	// Apply headers
	for k, v := range sim.Request.Headers {
		req.Header.Set(k, v)
	}

	// Default to application/json if body is present and Content-Type is missing
	if sim.Request.Body != "" && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}

	// Send request
	resp, err := e.Client.Do(req)
	result.Duration = time.Since(start)

	if err != nil {
		result.Passed = false
		result.ActualResult = "Request execution failed"
		result.Reason = err.Error()
		return result
	}
	defer resp.Body.Close()

	// Currently, validation is limited to the HTTP status code
	result.ActualResult = fmt.Sprintf("Status Code: %d", resp.StatusCode)

	if resp.StatusCode == sim.Expected.StatusCode {
		result.Passed = true
		result.Reason = "Status code matched expected value"
	} else {
		result.Passed = false
		result.Reason = fmt.Sprintf("Expected status code %d but got %d", sim.Expected.StatusCode, resp.StatusCode)
	}

	return result
}
