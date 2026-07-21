package engine

import (
	"io"
	"text/template"

	"github.com/suryatk2007/threatsim/pkg/types"
)

// HTMLReporter implements Reporter for HTML output
type HTMLReporter struct{}

// Generate writes the validation report as HTML to the specified io.Writer.
func (h *HTMLReporter) Generate(w io.Writer, report *types.ValidationReport, state map[string]string) error {
	tmpl := `<!DOCTYPE html>
<html>
<head>
<title>ThreatSim Report</title>
<style>
body { font-family: sans-serif; margin: 20px; }
.passed { color: green; }
.failed { color: red; }
table { border-collapse: collapse; width: 100%; }
th, td { border: 1px solid #ddd; padding: 8px; text-align: left; }
th { background-color: #f2f2f2; }
</style>
</head>
<body>
<h2>ThreatSim Validation Report</h2>
<p>Total Simulations: {{.TotalSimulations}}</p>
<p class="passed">Passed: {{.Passed}}</p>
<p class="failed">Failed: {{.Failed}}</p>
<p>Success Rate: {{printf "%.2f" .SuccessRate}}%</p>
<h3>Failed Simulations</h3>
{{if gt .Failed 0}}
<table>
<tr><th>Simulation Name</th><th>Method</th><th>URL</th><th>Reason</th></tr>
{{range .Results}}
{{if not .Passed}}
<tr>
	<td>{{.SimulationName}}</td>
	<td>{{.Method}}</td>
	<td>{{.URL}}</td>
	<td>{{.Reason}}</td>
</tr>
{{end}}
{{end}}
</table>
{{else}}
<p class="passed">All simulations passed successfully!</p>
{{end}}
</body>
</html>`

	maskedReport := *report
	maskedReport.Results = make([]types.SimulationResult, len(report.Results))
	for i, r := range report.Results {
		r.SimulationName = MaskSensitiveData(r.SimulationName, state)
		r.URL = MaskSensitiveData(r.URL, state)
		r.ActualResult = MaskSensitiveData(r.ActualResult, state)
		r.Reason = MaskSensitiveData(r.Reason, state)
		maskedReport.Results[i] = r
	}

	t, err := template.New("report").Parse(tmpl)
	if err != nil {
		return err
	}
	return t.Execute(w, maskedReport)
}
