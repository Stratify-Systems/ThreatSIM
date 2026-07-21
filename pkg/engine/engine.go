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

	"github.com/suryatk2007/threatsim/pkg/payloads"
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
		if sim.Expected.StatusCode == 0 && sim.Expected.BodyContains == "" && len(sim.Expected.Headers) == 0 {
			return nil, fmt.Errorf("simulation %q is missing expected criteria (e.g. status_code, body_contains, or headers)", sim.Name)
		}
	}

	return &def, nil
}

// Execute runs all simulations defined in the file and returns a comprehensive validation report.
func (e *Engine) Execute(def *types.SimulationDefinition) *types.ValidationReport {
	var expandedSimulations []types.Simulation
	for _, sim := range def.Simulations {
		expandedSimulations = append(expandedSimulations, expandSimulation(sim)...)
	}

	report := &types.ValidationReport{
		TotalSimulations: len(expandedSimulations),
	}

	start := time.Now()

	for _, sim := range expandedSimulations {
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

// expandSimulation multiplies a simulation definition if payloads are provided
func expandSimulation(sim types.Simulation) []types.Simulation {
	var payloadList []string

	if len(sim.Payloads) > 0 {
		payloadList = sim.Payloads
	} else if sim.PayloadType != "" {
		payloadList = payloads.Get(sim.PayloadType)
	}

	if len(payloadList) == 0 {
		return []types.Simulation{sim}
	}

	var expanded []types.Simulation
	for _, p := range payloadList {
		newSim := sim // copy by value
		newSim.Name = fmt.Sprintf("%s [Payload: %s]", sim.Name, p)

		// Replace {{payload}} in all relevant fields
		newSim.Request.Path = strings.ReplaceAll(newSim.Request.Path, "{{payload}}", p)
		newSim.Request.Body = strings.ReplaceAll(newSim.Request.Body, "{{payload}}", p)

		newHeaders := make(map[string]string)
		for k, v := range newSim.Request.Headers {
			newHeaders[k] = strings.ReplaceAll(v, "{{payload}}", p)
		}
		newSim.Request.Headers = newHeaders

		newQueryParams := make(map[string]string)
		for k, v := range newSim.Request.QueryParams {
			newQueryParams[k] = strings.ReplaceAll(v, "{{payload}}", p)
		}
		newSim.Request.QueryParams = newQueryParams

		expanded = append(expanded, newSim)
	}
	return expanded
}

// executeSimulation runs an individual simulation independently.
func (e *Engine) executeSimulation(sim types.Simulation) types.SimulationResult {
	start := time.Now()
	
	var expectedResults []string
	if sim.Expected.StatusCode != 0 {
		expectedResults = append(expectedResults, fmt.Sprintf("Status Code: %d", sim.Expected.StatusCode))
	}
	for k, v := range sim.Expected.Headers {
		expectedResults = append(expectedResults, fmt.Sprintf("Header %q: %q", k, v))
	}
	if sim.Expected.BodyContains != "" {
		expectedResults = append(expectedResults, fmt.Sprintf("Body Contains: %q", sim.Expected.BodyContains))
	}

	result := types.SimulationResult{
		SimulationName: sim.Name,
		ExpectedResult: strings.Join(expectedResults, " | "),
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

	result.Passed = true
	var actualResults []string
	var reasons []string

	// 1. Status Code Validation
	actualResults = append(actualResults, fmt.Sprintf("Status Code: %d", resp.StatusCode))
	if sim.Expected.StatusCode != 0 && resp.StatusCode != sim.Expected.StatusCode {
		result.Passed = false
		reasons = append(reasons, fmt.Sprintf("Expected status code %d but got %d", sim.Expected.StatusCode, resp.StatusCode))
	}

	// 2. Headers Validation
	for k, expectedVal := range sim.Expected.Headers {
		actualVal := resp.Header.Get(k)
		if actualVal == "" {
			result.Passed = false
			reasons = append(reasons, fmt.Sprintf("Expected header %q to be present, but it was missing", k))
		} else if actualVal != expectedVal {
			result.Passed = false
			reasons = append(reasons, fmt.Sprintf("Expected header %q to be %q, but got %q", k, expectedVal, actualVal))
		} else {
			actualResults = append(actualResults, fmt.Sprintf("Header %q: %q", k, actualVal))
		}
	}

	// 3. Body Contains Validation
	if sim.Expected.BodyContains != "" {
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			result.Passed = false
			reasons = append(reasons, "Failed to read response body")
		} else {
			bodyString := string(bodyBytes)
			if !strings.Contains(bodyString, sim.Expected.BodyContains) {
				result.Passed = false
				reasons = append(reasons, fmt.Sprintf("Expected body to contain %q, but it did not", sim.Expected.BodyContains))
			} else {
				actualResults = append(actualResults, fmt.Sprintf("Body Contains: %q", sim.Expected.BodyContains))
			}
		}
	}

	// Summarize results
	result.ActualResult = strings.Join(actualResults, " | ")
	if result.Passed {
		result.Reason = "All validations passed"
	} else {
		result.Reason = strings.Join(reasons, "; ")
	}

	return result
}
