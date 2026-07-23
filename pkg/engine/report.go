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

const ASCIIBanner = `
  _____ _   _ ____  _____    _  _____ ____ ___ __  __ 
 |_   _| | | |  _ \| ____|  / \|_   _/ ___|_ _|  \/  |
   | | | |_| | |_) |  _|   / _ \ | | \___ \| | | |\/| |
   | | |  _  |  _ <| |___ / ___ \| |  ___) | | | |  | |
   |_| |_| |_|_| \_\_____/_/   \_\_| |____/___|_|  |_|
          Security Behavior Validation Engine v1.0.0
`

const (
	colorReset     = "\033[0m"
	colorRed       = "\033[31m"
	colorGreen     = "\033[32m"
	colorYellow    = "\033[33m"
	colorBlue      = "\033[34m"
	colorMagenta   = "\033[35m"
	colorCyan      = "\033[36m"
	colorWhite     = "\033[37m"
	colorBold      = "\033[1m"
	colorDim       = "\033[2m"
	colorUnderline = "\033[4m"
)

// PrintBanner prints the ThreatSim ASCII art banner to the writer
func PrintBanner(w io.Writer) {
	fmt.Fprintf(w, "%s%s%s%s\n", colorBold, colorCyan, ASCIIBanner, colorReset)
}

// ConsoleReporter implements Reporter for human-readable terminal output
type ConsoleReporter struct{}

// Generate writes the detailed validation report to the specified io.Writer.
func (c *ConsoleReporter) Generate(w io.Writer, report *types.ValidationReport, state map[string]string) error {
	PrintBanner(w)

	divider := strings.Repeat("═", 78)
	thinDivider := strings.Repeat("─", 78)

	fmt.Fprintf(w, "%s%s%s\n", colorBold, colorCyan, divider)
	fmt.Fprintf(w, "                      %s%sSECURITY VALIDATION REPORT%s%s\n", colorBold, colorWhite, colorReset, colorCyan)
	fmt.Fprintf(w, "%s%s\n", divider, colorReset)

	// Summary Statistics Section
	var statusBadge string
	if report.Failed > 0 {
		statusBadge = fmt.Sprintf("%s%s[ FAILED - SECURITY RISKS DETECTED ]%s", colorBold, colorRed, colorReset)
	} else if report.Errors > 0 {
		statusBadge = fmt.Sprintf("%s%s[ ATTENTION - TEST SETUP / AUTH ERRORS ]%s", colorBold, colorYellow, colorReset)
	} else {
		statusBadge = fmt.Sprintf("%s%s[ PASSED - ALL CONTROLS VERIFIED ]%s", colorBold, colorGreen, colorReset)
	}

	fmt.Fprintf(w, "  %sTotal Simulations: %s %d\n", colorBold, colorReset, report.TotalSimulations)
	fmt.Fprintf(w, "  %sPassed Controls:   %s %s%d%s\n", colorBold, colorReset, colorGreen, report.Passed, colorReset)
	
	if report.Failed > 0 {
		fmt.Fprintf(w, "  %sFailed Controls:   %s %s%d%s\n", colorBold, colorReset, colorRed, report.Failed, colorReset)
	} else {
		fmt.Fprintf(w, "  %sFailed Controls:   %s 0\n", colorBold, colorReset)
	}

	if report.Errors > 0 {
		fmt.Fprintf(w, "  %sSetup/Auth Errors: %s %s%d%s\n", colorBold, colorReset, colorYellow, report.Errors, colorReset)
	}

	fmt.Fprintf(w, "  %sSuccess Rate:      %s %.2f%%\n", colorBold, colorReset, report.SuccessRate)
	fmt.Fprintf(w, "  %sExecution Time:    %s %v\n", colorBold, colorReset, report.ExecutionTime)
	fmt.Fprintf(w, "  %sOverall Status:    %s %s\n", colorBold, colorReset, statusBadge)
	fmt.Fprintf(w, "%s%s%s\n\n", colorCyan, divider, colorReset)

	// Section 1: Security Control Failures (Vulnerabilities)
	if report.Failed > 0 {
		fmt.Fprintf(w, "%s%s--- SECURITY RISKS & VULNERABILITIES DETECTED (%d) ---%s\n", colorBold, colorRed, report.Failed, colorReset)
		for i, res := range report.Results {
			if !res.Passed && !res.IsError {
				fmt.Fprintf(w, "\n %s%d. ✗ [FAIL]%s %s%s%s\n", colorRed, i+1, colorReset, colorBold, MaskSensitiveData(res.SimulationName, state), colorReset)
				if res.Method != "" && res.URL != "" {
					fmt.Fprintf(w, "    %s• Target Endpoint:%s  %s %s\n", colorDim, colorReset, colorCyan+res.Method+colorReset, MaskSensitiveData(res.URL, state))
				}
				if res.ExpectedResult != "" {
					fmt.Fprintf(w, "    %s• Expected Policy:%s  %s\n", colorDim, colorReset, res.ExpectedResult)
				}
				if res.ActualResult != "" {
					fmt.Fprintf(w, "    %s• Actual Response:%s  %s\n", colorDim, colorReset, MaskSensitiveData(res.ActualResult, state))
				}
				if res.Reason != "" {
					fmt.Fprintf(w, "    %s• Root Cause:     %s  %s%s%s\n", colorDim, colorReset, colorRed, MaskSensitiveData(res.Reason, state), colorReset)
				}
				timeStr := fmt.Sprintf("%.2fms", float64(res.Duration.Microseconds())/1000.0)
				fmt.Fprintf(w, "    %s• Latency:        %s  %s\n", colorDim, colorReset, timeStr)
			}
		}
		fmt.Fprintf(w, "\n%s%s%s\n\n", colorRed, thinDivider, colorReset)
	}

	// Section 2: Test Setup & Authentication Errors
	if report.Errors > 0 {
		fmt.Fprintf(w, "%s%s--- TEST SETUP & AUTHENTICATION ERRORS (%d) ---%s\n", colorBold, colorYellow, report.Errors, colorReset)
		for i, res := range report.Results {
			if !res.Passed && res.IsError {
				fmt.Fprintf(w, "\n %s%d. ⚠️ [ERROR]%s %s%s%s %s(Test Setup Failure)%s\n", colorYellow, i+1, colorReset, colorBold, MaskSensitiveData(res.SimulationName, state), colorReset, colorDim, colorReset)
				if res.ActualResult != "" {
					fmt.Fprintf(w, "    %s• Error Issue:    %s  %s%s%s\n", colorDim, colorReset, colorYellow, MaskSensitiveData(res.ActualResult, state), colorReset)
				}
				if res.Reason != "" {
					fmt.Fprintf(w, "    %s• Root Cause:     %s  %s\n", colorDim, colorReset, MaskSensitiveData(res.Reason, state))
				}
				if res.Method != "" && res.URL != "" {
					fmt.Fprintf(w, "    %s• Target Endpoint:%s  %s %s\n", colorDim, colorReset, colorCyan+res.Method+colorReset, MaskSensitiveData(res.URL, state))
				}
				fmt.Fprintf(w, "    %s• Diagnostic Tip: %s  %sVerify target login path (auth_path), credentials, and token_json_path in YAML%s\n", colorDim, colorReset, colorCyan, colorReset)
				timeStr := fmt.Sprintf("%.2fms", float64(res.Duration.Microseconds())/1000.0)
				fmt.Fprintf(w, "    %s• Latency:        %s  %s\n", colorDim, colorReset, timeStr)
			}
		}
		fmt.Fprintf(w, "\n%s%s%s\n\n", colorYellow, thinDivider, colorReset)
	}

	// Section 3: Passed Simulations
	if report.Passed > 0 {
		fmt.Fprintf(w, "%s%s--- PASSED CONTROLS (%d) ---%s\n", colorBold, colorGreen, report.Passed, colorReset)
		for i, res := range report.Results {
			if res.Passed {
				timeStr := fmt.Sprintf("%.2fms", float64(res.Duration.Microseconds())/1000.0)
				fmt.Fprintf(w, "\n %s%d. ✓ [PASS]%s %s%s%s\n", colorGreen, i+1, colorReset, colorBold, MaskSensitiveData(res.SimulationName, state), colorReset)
				if res.Method != "" && res.URL != "" {
					fmt.Fprintf(w, "    %s• Target Endpoint:%s  %s %s\n", colorDim, colorReset, colorCyan+res.Method+colorReset, MaskSensitiveData(res.URL, state))
				}
				if res.ActualResult != "" {
					fmt.Fprintf(w, "    %s• Assertion Match:%s  %s\n", colorDim, colorReset, MaskSensitiveData(res.ActualResult, state))
				}
				fmt.Fprintf(w, "    %s• Latency:        %s  %s\n", colorDim, colorReset, timeStr)
			}
		}
		fmt.Fprintf(w, "\n%s%s%s\n\n", colorGreen, thinDivider, colorReset)
	}

	fmt.Fprintf(w, "%s%s==============================================================================%s\n", colorBold, colorCyan, colorReset)
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
