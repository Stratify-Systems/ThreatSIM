package engine

import (
	"fmt"
	"io"

	"github.com/go-pdf/fpdf"
	"github.com/suryatk2007/threatsim/pkg/types"
)

// PDFReporter implements Reporter for PDF output
type PDFReporter struct{}

// Generate writes the validation report as PDF to the specified io.Writer.
func (p *PDFReporter) Generate(w io.Writer, report *types.ValidationReport, state map[string]string) error {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.AddPage()
	pdf.SetFont("Arial", "B", 16)
	pdf.Cell(40, 10, "ThreatSim Validation Report")
	pdf.Ln(12)

	pdf.SetFont("Arial", "", 12)
	pdf.Cell(40, 10, fmt.Sprintf("Total Simulations: %d", report.TotalSimulations))
	pdf.Ln(8)
	
	pdf.SetTextColor(0, 128, 0)
	pdf.Cell(40, 10, fmt.Sprintf("Passed: %d", report.Passed))
	pdf.Ln(8)

	pdf.SetTextColor(255, 0, 0)
	pdf.Cell(40, 10, fmt.Sprintf("Failed: %d", report.Failed))
	pdf.Ln(8)

	pdf.SetTextColor(0, 0, 0)
	pdf.Cell(40, 10, fmt.Sprintf("Success Rate: %.2f%%", report.SuccessRate))
	pdf.Ln(15)

	if report.Failed > 0 {
		pdf.SetFont("Arial", "B", 14)
		pdf.Cell(40, 10, "Failed Simulations")
		pdf.Ln(10)
		pdf.SetFont("Arial", "", 12)

		for _, res := range report.Results {
			if !res.Passed {
				pdf.SetTextColor(255, 0, 0)
				pdf.Cell(0, 8, fmt.Sprintf("[FAIL] %s", MaskSensitiveData(res.SimulationName, state)))
				pdf.Ln(6)
				pdf.SetTextColor(0, 0, 0)
				pdf.SetFont("Arial", "I", 10)
				pdf.Cell(0, 6, fmt.Sprintf("  Reason: %s", MaskSensitiveData(res.Reason, state)))
				pdf.SetFont("Arial", "", 12)
				pdf.Ln(8)
			}
		}
	} else {
		pdf.SetTextColor(0, 128, 0)
		pdf.Cell(0, 10, "All simulations passed successfully!")
	}

	return pdf.Output(w)
}
