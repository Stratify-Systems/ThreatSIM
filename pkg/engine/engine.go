package engine

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/suryatk2007/threatsim/pkg/plugins"
	"github.com/suryatk2007/threatsim/pkg/types"
	"gopkg.in/yaml.v3"
)

// Engine is the core execution engine responsible for loading and running simulations.
type Engine struct {
	TargetURL string
	Client    *http.Client
	State     map[string]string
	mu        sync.RWMutex
}

// New creates a new Engine instance.
func New(targetURL string) *Engine {
	return &Engine{
		TargetURL: strings.TrimRight(targetURL, "/"),
		Client: &http.Client{
			Timeout: 15 * time.Second,
		},
		State: make(map[string]string),
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
		
		if sim.Plugin != "" {
			continue // Skip standard HTTP validations if using a plugin
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
	report := &types.ValidationReport{}
	start := time.Now()
	var allResults []types.SimulationResult

	for _, sim := range def.Simulations {
		if sim.Plugin != "" {
			// Execute via Plugin Architecture
			p, err := plugins.Get(sim.Plugin)
			if err != nil {
				allResults = append(allResults, types.SimulationResult{
					SimulationName: sim.Name,
					Passed:         false,
					Reason:         err.Error(),
				})
				continue
			}

			ctx := plugins.Context{
				TargetURL: e.TargetURL,
				Client:    e.Client,
				State:     e.State,
			}
			pluginResults := p.Execute(sim.Name, ctx, sim.PluginConfig)
			allResults = append(allResults, pluginResults...)
		} else {
			// Execute standard HTTP validation
			allResults = append(allResults, e.executeSimulation(sim))
		}
	}

	report.Results = allResults
	report.TotalSimulations = len(allResults)

	for _, result := range allResults {
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



// interpolate replaces any {{state.VAR}} with its value from Engine.State
func (e *Engine) interpolate(input string) string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	for k, v := range e.State {
		input = strings.ReplaceAll(input, fmt.Sprintf("{{state.%s}}", k), v)
	}
	return input
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
		Method:         sim.Request.Method,
	}

	// INTERPOLATE STATE VARIABLES
	sim.Request.Path = e.interpolate(sim.Request.Path)
	sim.Request.Body = e.interpolate(sim.Request.Body)
	for k, v := range sim.Request.Headers {
		sim.Request.Headers[k] = e.interpolate(v)
	}
	for k, v := range sim.Request.QueryParams {
		sim.Request.QueryParams[k] = e.interpolate(v)
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
	
	result.URL = reqURL

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
	
	// Unconditionally read body for validation and extraction
	bodyBytes, readErr := io.ReadAll(resp.Body)
	resp.Body.Close()
	var bodyString string
	if readErr == nil {
		bodyString = string(bodyBytes)
	}

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
		if readErr != nil {
			result.Passed = false
			reasons = append(reasons, "Failed to read response body")
		} else {
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
		
		// EXTRACTION LOGIC
		for varName, headerKey := range sim.Extract.Header {
			if val := resp.Header.Get(headerKey); val != "" {
				e.mu.Lock()
				e.State[varName] = val
				e.mu.Unlock()
			}
		}

		if len(sim.Extract.Regex) > 0 && readErr == nil {
			for varName, regexPattern := range sim.Extract.Regex {
				re, reErr := regexp.Compile(regexPattern)
				if reErr == nil {
					matches := re.FindStringSubmatch(bodyString)
					if len(matches) > 1 {
						e.mu.Lock()
						e.State[varName] = matches[1]
						e.mu.Unlock()
					}
				}
			}
		}

		if len(sim.Extract.JSON) > 0 && readErr == nil {
			var jsonData map[string]interface{}
			if jsonErr := json.Unmarshal(bodyBytes, &jsonData); jsonErr == nil {
				for varName, jsonPath := range sim.Extract.JSON {
					if val, ok := extractJSONPath(jsonData, jsonPath); ok {
						e.mu.Lock()
						e.State[varName] = fmt.Sprintf("%v", val)
						e.mu.Unlock()
					}
				}
			}
		}
	} else {
		result.Reason = strings.Join(reasons, "; ")
	}

	return result
}

// extractJSONPath allows deep key extraction using dot notation (e.g. "user.token")
func extractJSONPath(data map[string]interface{}, path string) (interface{}, bool) {
	keys := strings.Split(path, ".")
	var current interface{} = data

	for _, key := range keys {
		if m, ok := current.(map[string]interface{}); ok {
			if val, exists := m[key]; exists {
				current = val
			} else {
				return nil, false
			}
		} else {
			return nil, false
		}
	}
	return current, true
}
