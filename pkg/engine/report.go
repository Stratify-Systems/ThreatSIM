package engine

import (
	"fmt"
	"io"

	"github.com/suryatk2007/threatsim/pkg/types"
)

// PrintReport formats and prints the validation report in a human-readable format.
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorCyan   = "\033[36m"
	colorBold   = "\033[1m"
)

// PrintReport writes the validation report to the specified io.Writer.
func PrintReport(w io.Writer, report *types.ValidationReport) {
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
				fmt.Fprintf(w, "\n%s✗ [FAIL] %s%s\n", colorRed, res.SimulationName, colorReset)
				if res.Method != "" && res.URL != "" {
					fmt.Fprintf(w, "  Request:  %s %s\n", res.Method, res.URL)
				}
				fmt.Fprintf(w, "  Expected: %s\n", res.ExpectedResult)
				fmt.Fprintf(w, "  Actual:   %s\n", res.ActualResult)
				fmt.Fprintf(w, "  Reason:   %s%s%s\n", colorYellow, res.Reason, colorReset)
			}
		}
	}

	fmt.Fprintf(w, "\n%s%s--- PASSED SIMULATIONS ---%s\n", colorBold, colorGreen, colorReset)
	for _, res := range report.Results {
		if res.Passed {
			timeStr := fmt.Sprintf("%.2fms", float64(res.Duration.Microseconds())/1000.0)
			fmt.Fprintf(w, "%s✓ [PASS]%s %s (%s)\n", colorGreen, colorReset, res.SimulationName, timeStr)
		}
	}

	fmt.Fprintf(w, "%s%s========================================%s\n", colorBold, colorCyan, colorReset)
}
