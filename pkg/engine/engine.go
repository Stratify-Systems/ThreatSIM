package engine

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
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

var (
	ErrValidationFailed = errors.New("validation failed")
	ErrPluginNotFound   = errors.New("plugin not found")
	ErrSimulationFormat = errors.New("invalid simulation format")
)

// Engine is the core execution engine responsible for loading and running simulations.
type Engine struct {
	TargetURL string
	Client    *http.Client
	State     *plugins.StateMap
	Timeout   time.Duration
	Insecure  bool
	mu        sync.RWMutex
}

// EngineOption defines a functional option for configuring the Engine.
type EngineOption func(*Engine)

// WithTimeout sets a custom default timeout for HTTP requests.
func WithTimeout(d time.Duration) EngineOption {
	return func(e *Engine) {
		if d > 0 {
			e.Timeout = d
		}
	}
}

// WithInsecure sets whether to skip SSL/TLS certificate verification.
func WithInsecure(insecure bool) EngineOption {
	return func(e *Engine) {
		e.Insecure = insecure
	}
}

// New creates a new Engine instance.
func New(targetURL string, opts ...EngineOption) *Engine {
	eng := &Engine{
		TargetURL: strings.TrimRight(targetURL, "/"),
		Timeout:   15 * time.Second,
		State:     plugins.NewStateMap(),
	}

	for _, opt := range opts {
		opt(eng)
	}

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: eng.Insecure,
		},
	}

	eng.Client = &http.Client{
		Transport: tr,
	}

	return eng
}

// LoadSimulation reads and validates a simulation file (supports YAML and JSON).
func (e *Engine) LoadSimulation(filePath string) (*types.SimulationDefinition, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	expandedData := os.ExpandEnv(string(data))

	var def types.SimulationDefinition
	
	// Attempt YAML parsing first (YAML is a superset of JSON, so this works for both).
	if err := yaml.Unmarshal([]byte(expandedData), &def); err != nil {
		// Fallback to strict JSON if YAML fails (though unlikely if it's valid JSON)
		if jsonErr := json.Unmarshal([]byte(expandedData), &def); jsonErr != nil {
			return nil, fmt.Errorf("%w (must be valid YAML or JSON): %v", ErrSimulationFormat, err)
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
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, sim := range def.Simulations {
		wg.Add(1)
		go func(sim types.Simulation) {
			defer wg.Done()
			var res []types.SimulationResult

			if sim.Plugin != "" {
				// Validate plugin config against schema
				if err := plugins.ValidateConfig(sim.Plugin, sim.PluginConfig); err != nil {
					res = []types.SimulationResult{{
						SimulationName: sim.Name,
						Passed:         false,
						ExpectedResult: "Valid plugin configuration schema",
						ActualResult:   "Schema Validation Failed",
						Reason:         err.Error(),
					}}
				} else {
					// Execute via Plugin Architecture
					p, err := plugins.Get(sim.Plugin)
					if err != nil {
						res = []types.SimulationResult{{
							SimulationName: sim.Name,
							Passed:         false,
							Reason:         fmt.Sprintf("%v: %v", ErrPluginNotFound, err),
						}}
					} else {
						ctx := plugins.Context{
							TargetURL: e.TargetURL,
							Client:    e.Client,
							State:     e.State,
						}
						res = p.Execute(sim.Name, ctx, sim.PluginConfig)
					}
				}
			} else {
				// Execute standard HTTP validation
				res = []types.SimulationResult{e.executeSimulation(sim)}
			}

			mu.Lock()
			allResults = append(allResults, res...)
			mu.Unlock()
		}(sim)
	}

	wg.Wait()

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
	if sim.Expected.BodyRegex != "" {
		expectedResults = append(expectedResults, fmt.Sprintf("Body Matches Regex: %q", sim.Expected.BodyRegex))
	}

	result := types.SimulationResult{
		SimulationName: sim.Name,
		ExpectedResult: strings.Join(expectedResults, " | "),
		Method:         sim.Request.Method,
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

	reqTimeout := e.Timeout
	if sim.Request.Timeout != "" {
		if parsedTimeout, err := time.ParseDuration(sim.Request.Timeout); err == nil && parsedTimeout > 0 {
			reqTimeout = parsedTimeout
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), reqTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, sim.Request.Method, reqURL, bodyReader)
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
	
	// Unconditionally read body for validation and extraction (limit to 5MB)
	bodyBytes, readErr := io.ReadAll(io.LimitReader(resp.Body, 5*1024*1024))
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

	// 4. Body Regex Validation
	if sim.Expected.BodyRegex != "" {
		if readErr != nil {
			result.Passed = false
			reasons = append(reasons, "Failed to read response body for regex validation")
		} else {
			re, err := regexp.Compile(sim.Expected.BodyRegex)
			if err != nil {
				result.Passed = false
				reasons = append(reasons, fmt.Sprintf("Invalid BodyRegex pattern: %v", err))
			} else if !re.MatchString(bodyString) {
				result.Passed = false
				reasons = append(reasons, fmt.Sprintf("Expected body to match regex %q, but it did not", sim.Expected.BodyRegex))
			} else {
				actualResults = append(actualResults, fmt.Sprintf("Body Matches Regex: %q", sim.Expected.BodyRegex))
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
