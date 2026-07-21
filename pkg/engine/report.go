package engine

import (
	"fmt"
	"io"

	"github.com/suryatk2007/threatsim/pkg/types"
)

// PrintReport formats and prints the validation report in a human-readable format.
// This output is suitable for developers and CI/CD logs.
func PrintReport(w io.Writer, report *types.ValidationReport) {
	fmt.Fprintln(w, "========================================")
	fmt.Fprintln(w, "       ThreatSim Validation Report      ")
	fmt.Fprintln(w, "========================================")
	
	fmt.Fprintf(w, "Total Simulations: %d\n", report.TotalSimulations)
	fmt.Fprintf(w, "Passed:            %d\n", report.Passed)
	fmt.Fprintf(w, "Failed:            %d\n", report.Failed)
	fmt.Fprintf(w, "Success Rate:      %.2f%%\n", report.SuccessRate)
	fmt.Fprintf(w, "Execution Time:    %v\n", report.ExecutionTime)
	
	if report.Failed > 0 {
		fmt.Fprintln(w, "\n--- Failed Simulations ---")
		for _, res := range report.Results {
			if !res.Passed {
				fmt.Fprintf(w, "\n[FAIL] %s\n", res.SimulationName)
				fmt.Fprintf(w, "  Expected: %s\n", res.ExpectedResult)
				fmt.Fprintf(w, "  Actual:   %s\n", res.ActualResult)
				fmt.Fprintf(w, "  Reason:   %s\n", res.Reason)
			}
		}
	}

	fmt.Fprintln(w, "========================================")
}
