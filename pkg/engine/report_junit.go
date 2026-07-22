package engine

import (
	"encoding/xml"
	"fmt"
	"io"

	"github.com/suryatk2007/threatsim/pkg/types"
)

// JUnitReporter generates standard JUnit XML reports for GitLab CI, Jenkins, Azure DevOps, and CircleCI.
type JUnitReporter struct{}

type JUnitTestSuites struct {
	XMLName    xml.Name         `xml:"testsuites"`
	Name       string           `xml:"name,attr"`
	Tests      int              `xml:"tests,attr"`
	Failures   int              `xml:"failures,attr"`
	Errors     int              `xml:"errors,attr"`
	Time       string           `xml:"time,attr"`
	TestSuite  []JUnitTestSuite `xml:"testsuite"`
}

type JUnitTestSuite struct {
	Name     string          `xml:"name,attr"`
	Tests    int             `xml:"tests,attr"`
	Failures int             `xml:"failures,attr"`
	Errors   int             `xml:"errors,attr"`
	Time     string          `xml:"time,attr"`
	TestCase []JUnitTestCase `xml:"testcase"`
}

type JUnitTestCase struct {
	Name      string        `xml:"name,attr"`
	ClassName string        `xml:"classname,attr"`
	Time      string        `xml:"time,attr"`
	Failure   *JUnitFailure `xml:"failure,omitempty"`
}

type JUnitFailure struct {
	Message string `xml:"message,attr"`
	Type    string `xml:"type,attr"`
	Content string `xml:",chardata"`
}

// Generate formats the validation report into standard JUnit XML.
func (j *JUnitReporter) Generate(w io.Writer, report *types.ValidationReport, state map[string]string) error {
	totalSecs := report.ExecutionTime.Seconds()
	
	var testCases []JUnitTestCase

	for _, res := range report.Results {
		simName := MaskSensitiveData(res.SimulationName, state)
		targetURL := MaskSensitiveData(res.URL, state)
		actual := MaskSensitiveData(res.ActualResult, state)
		reason := MaskSensitiveData(res.Reason, state)

		tcDurationSecs := res.Duration.Seconds()

		tc := JUnitTestCase{
			Name:      simName,
			ClassName: "ThreatSim.SecuritySimulation",
			Time:      fmt.Sprintf("%.4f", tcDurationSecs),
		}

		if !res.Passed {
			failContent := fmt.Sprintf("Request:  %s %s\nExpected: %s\nActual:   %s\nReason:   %s",
				res.Method, targetURL, res.ExpectedResult, actual, reason)

			tc.Failure = &JUnitFailure{
				Message: fmt.Sprintf("Simulation %q failed: %s", simName, reason),
				Type:    "SecurityControlFailure",
				Content: failContent,
			}
		}

		testCases = append(testCases, tc)
	}

	suite := JUnitTestSuite{
		Name:     "ThreatSim Security Simulations",
		Tests:    report.TotalSimulations,
		Failures: report.Failed,
		Errors:   0,
		Time:     fmt.Sprintf("%.4f", totalSecs),
		TestCase: testCases,
	}

	suites := JUnitTestSuites{
		Name:      "ThreatSim Validation Suite",
		Tests:     report.TotalSimulations,
		Failures:  report.Failed,
		Errors:    0,
		Time:      fmt.Sprintf("%.4f", totalSecs),
		TestSuite: []JUnitTestSuite{suite},
	}

	io.WriteString(w, xml.Header)
	encoder := xml.NewEncoder(w)
	encoder.Indent("", "  ")
	return encoder.Encode(suites)
}
