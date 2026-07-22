package engine

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/suryatk2007/threatsim/pkg/types"
)

// SARIFReporter generates OASIS SARIF v2.1.0 JSON reports compatible with GitHub Security Code Scanning
type SARIFReporter struct{}

// SARIFLog represents the top-level SARIF v2.1.0 document schema
type SARIFLog struct {
	Schema  string     `json:"$schema"`
	Version string     `json:"version"`
	Runs    []SARIFRun `json:"runs"`
}

type SARIFRun struct {
	Tool    SARIFTool     `json:"tool"`
	Results []SARIFResult `json:"results"`
}

type SARIFTool struct {
	Driver SARIFDriver `json:"driver"`
}

type SARIFDriver struct {
	Name            string      `json:"name"`
	SemanticVersion string      `json:"semanticVersion"`
	InformationURI  string      `json:"informationUri"`
	Rules           []SARIFRule `json:"rules"`
}

type SARIFRule struct {
	ID               string                `json:"id"`
	Name             string                `json:"name"`
	ShortDescription SARIFMultiformatText `json:"shortDescription"`
	FullDescription  SARIFMultiformatText `json:"fullDescription"`
	HelpURI          string                `json:"helpUri"`
}

type SARIFResult struct {
	RuleID    string          `json:"ruleId"`
	Level     string          `json:"level"`
	Message   SARIFMessage    `json:"message"`
	Locations []SARIFLocation `json:"locations,omitempty"`
}

type SARIFMessage struct {
	Text string `json:"text"`
}

type SARIFMultiformatText struct {
	Text string `json:"text"`
}

type SARIFLocation struct {
	PhysicalLocation SARIFPhysicalLocation `json:"physicalLocation"`
}

type SARIFPhysicalLocation struct {
	ArtifactLocation SARIFArtifactLocation `json:"artifactLocation"`
}

type SARIFArtifactLocation struct {
	URI string `json:"uri"`
}

// Generate formats the validation report into SARIF v2.1.0 JSON
func (s *SARIFReporter) Generate(w io.Writer, report *types.ValidationReport, state map[string]string) error {
	rules := []SARIFRule{
		{
			ID:   "THREATSIM-001",
			Name: "SecurityBehaviorValidationFailure",
			ShortDescription: SARIFMultiformatText{
				Text: "Application Security Control Boundary Failure",
			},
			FullDescription: SARIFMultiformatText{
				Text: "ThreatSim detected an application security behavior policy assertion failure or vulnerability.",
			},
			HelpURI: "https://github.com/suryatk2007/threatsim/tree/main/docs",
		},
	}

	var results []SARIFResult

	for _, res := range report.Results {
		if !res.Passed {
			simName := MaskSensitiveData(res.SimulationName, state)
			targetURL := MaskSensitiveData(res.URL, state)
			if targetURL == "" {
				targetURL = "http://target-application"
			}
			actual := MaskSensitiveData(res.ActualResult, state)
			reason := MaskSensitiveData(res.Reason, state)

			msgText := fmt.Sprintf("Security Boundary Failure in simulation %q: Expected %q, got %q. Reason: %s",
				simName, res.ExpectedResult, actual, reason)

			sarifRes := SARIFResult{
				RuleID: "THREATSIM-001",
				Level:  "error",
				Message: SARIFMessage{
					Text: msgText,
				},
				Locations: []SARIFLocation{
					{
						PhysicalLocation: SARIFPhysicalLocation{
							ArtifactLocation: SARIFArtifactLocation{
								URI: strings.TrimPrefix(targetURL, "/"),
							},
						},
					},
				},
			}
			results = append(results, sarifRes)
		}
	}

	sarifLog := SARIFLog{
		Schema:  "https://raw.githubusercontent.com/oasis-tcs/sarif-spec/master/Schemata/sarif-schema-2.1.0.json",
		Version: "2.1.0",
		Runs: []SARIFRun{
			{
				Tool: SARIFTool{
					Driver: SARIFDriver{
						Name:            "ThreatSim",
						SemanticVersion: "1.0.0",
						InformationURI:  "https://github.com/suryatk2007/threatsim",
						Rules:           rules,
					},
				},
				Results: results,
			},
		},
	}

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(sarifLog)
}
