package types

import "time"

// SimulationDefinition represents the structure of the security simulation file.
type SimulationDefinition struct {
	Version     string       `yaml:"version" json:"version"`
	Simulations []Simulation `yaml:"simulations" json:"simulations"`
}

// Simulation represents a single security test case.
type Simulation struct {
	Name     string   `yaml:"name" json:"name"`
	Request  Request  `yaml:"request" json:"request"`
	Expected Expected `yaml:"expected" json:"expected"`
}

// Request defines the HTTP request to be sent to the target application.
type Request struct {
	Method      string            `yaml:"method" json:"method"`
	Path        string            `yaml:"path" json:"path"` // Path is appended to the target URL
	Headers     map[string]string `yaml:"headers" json:"headers"`
	QueryParams map[string]string `yaml:"query_params" json:"query_params"`
	Body        string            `yaml:"body" json:"body"` // JSON request body (or any other string)
}

// Expected defines the expected response for validation.
type Expected struct {
	StatusCode   int               `yaml:"status_code,omitempty" json:"status_code,omitempty"`
	BodyContains string            `yaml:"body_contains,omitempty" json:"body_contains,omitempty"`
	Headers      map[string]string `yaml:"headers,omitempty" json:"headers,omitempty"`
}

// SimulationResult holds the outcome of a single simulation.
type SimulationResult struct {
	SimulationName string
	Passed         bool
	ExpectedResult string
	ActualResult   string
	Reason         string
	Duration       time.Duration
}

// ValidationReport holds the summary of all executed simulations.
type ValidationReport struct {
	TotalSimulations int
	Passed           int
	Failed           int
	ExecutionTime    time.Duration
	SuccessRate      float64
	Results          []SimulationResult
}
