package engine

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/suryatk2007/threatsim/pkg/types"
)

// Reporter defines the interface for generating validation reports
type Reporter interface {
	Generate(w io.Writer, report *types.ValidationReport, state map[string]string) error
}

// MaskSensitiveData masks any value found in the state map from the result strings
func MaskSensitiveData(text string, state map[string]string) string {
	for _, v := range state {
		if len(v) > 3 { // Only mask values longer than 3 chars to avoid over-masking
			text = strings.ReplaceAll(text, v, "********")
		}
	}
	return text
}

// ConsoleReporter implements Reporter for human-readable terminal output
type ConsoleReporter struct{}

const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorCyan   = "\033[36m"
	colorBold   = "\033[1m"
)

// Generate writes the validation report to the specified io.Writer.
func (c *ConsoleReporter) Generate(w io.Writer, report *types.ValidationReport, state map[string]string) error {
	fmt.Fprintf(w, "%s%s========================================\n", colorBold, colorCyan)
	fmt.Fprintf(w, "       ThreatSim Validation Report      \n")
	fmt.Fprintf(w, "========================================%s\n", colorReset)

	fmt.Fprintf(w, "Total Simulations: %d\n", report.TotalSimulations)
	fmt.Fprintf(w, "Passed:            %s%d%s\n", colorGreen, report.Passed, colorReset)

	if report.Failed > 0 {
		fmt.Fprintf(w, "Failed:            %s%d%s\n", colorRed, report.Failed, colorReset)
	} else {
		fmt.Fprintf(w, "Failed:            %d\n", report.Failed)
	}

	fmt.Fprintf(w, "Success Rate:      %.2f%%\n", report.SuccessRate)
	fmt.Fprintf(w, "Execution Time:    %v\n", report.ExecutionTime)

	if report.Failed > 0 {
		fmt.Fprintf(w, "\n%s%s--- FAILED SIMULATIONS ---%s\n", colorBold, colorRed, colorReset)
		for _, res := range report.Results {
			if !res.Passed {
				fmt.Fprintf(w, "\n%s✗ [FAIL] %s%s\n", colorRed, MaskSensitiveData(res.SimulationName, state), colorReset)
				if res.Method != "" && res.URL != "" {
					fmt.Fprintf(w, "  Request:  %s %s\n", res.Method, MaskSensitiveData(res.URL, state))
				}
				fmt.Fprintf(w, "  Expected: %s\n", res.ExpectedResult)
				fmt.Fprintf(w, "  Actual:   %s\n", MaskSensitiveData(res.ActualResult, state))
				fmt.Fprintf(w, "  Reason:   %s%s%s\n", colorYellow, MaskSensitiveData(res.Reason, state), colorReset)
			}
		}
	}

	fmt.Fprintf(w, "\n%s%s--- PASSED SIMULATIONS ---%s\n", colorBold, colorGreen, colorReset)
	for _, res := range report.Results {
		if res.Passed {
			timeStr := fmt.Sprintf("%.2fms", float64(res.Duration.Microseconds())/1000.0)
			fmt.Fprintf(w, "%s✓ [PASS]%s %s (%s)\n", colorGreen, colorReset, MaskSensitiveData(res.SimulationName, state), timeStr)
		}
	}

	fmt.Fprintf(w, "%s%s========================================%s\n", colorBold, colorCyan, colorReset)
	return nil
}

// JSONReporter implements Reporter for machine-readable JSON output
type JSONReporter struct{}

// Generate writes the validation report as JSON to the specified io.Writer.
func (j *JSONReporter) Generate(w io.Writer, report *types.ValidationReport, state map[string]string) error {
	maskedReport := *report
	maskedReport.Results = make([]types.SimulationResult, len(report.Results))
	for i, r := range report.Results {
		r.SimulationName = MaskSensitiveData(r.SimulationName, state)
		r.URL = MaskSensitiveData(r.URL, state)
		r.ActualResult = MaskSensitiveData(r.ActualResult, state)
		r.Reason = MaskSensitiveData(r.Reason, state)
		maskedReport.Results[i] = r
	}
	
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(maskedReport)
}
