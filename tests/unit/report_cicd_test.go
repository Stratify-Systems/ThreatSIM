package unit

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"strings"
	"testing"
	"time"

	"github.com/suryatk2007/threatsim/pkg/engine"
	"github.com/suryatk2007/threatsim/pkg/types"
)

func TestSARIFReporter_Generate(t *testing.T) {
	t.Parallel()

	report := &types.ValidationReport{
		TotalSimulations: 2,
		Passed:           1,
		Failed:           1,
		SuccessRate:      50.0,
		ExecutionTime:    10 * time.Millisecond,
		Results: []types.SimulationResult{
			{
				SimulationName: "IDOR Test PASS",
				Passed:         true,
				Method:         "GET",
				URL:            "http://localhost:8080/api/users/1",
				Duration:       5 * time.Millisecond,
			},
			{
				SimulationName: "JWT Forge Test FAIL",
				Passed:         false,
				Method:         "GET",
				URL:            "http://localhost:8080/api/secrets",
				ExpectedResult: "Status 401 Unauthorized",
				ActualResult:   "Status 200 OK",
				Reason:         "Backend accepted untrusted forged signature",
				Duration:       5 * time.Millisecond,
			},
		},
	}

	reporter := &engine.SARIFReporter{}
	var buf bytes.Buffer

	err := reporter.Generate(&buf, report, nil)
	if err != nil {
		t.Fatalf("Failed to generate SARIF report: %v", err)
	}

	var sarifLog engine.SARIFLog
	if err := json.Unmarshal(buf.Bytes(), &sarifLog); err != nil {
		t.Fatalf("Generated SARIF report is invalid JSON: %v", err)
	}

	if sarifLog.Version != "2.1.0" {
		t.Errorf("Expected SARIF version 2.1.0, got %s", sarifLog.Version)
	}

	if len(sarifLog.Runs) != 1 {
		t.Fatalf("Expected 1 SARIF run, got %d", len(sarifLog.Runs))
	}

	results := sarifLog.Runs[0].Results
	if len(results) != 1 {
		t.Fatalf("Expected 1 SARIF security failure result, got %d", len(results))
	}

	if results[0].RuleID != "THREATSIM-001" {
		t.Errorf("Expected RuleID THREATSIM-001, got %s", results[0].RuleID)
	}

	if !strings.Contains(results[0].Message.Text, "JWT Forge Test FAIL") {
		t.Errorf("Expected result message to contain simulation name, got: %s", results[0].Message.Text)
	}
}

func TestJUnitReporter_Generate(t *testing.T) {
	t.Parallel()

	report := &types.ValidationReport{
		TotalSimulations: 2,
		Passed:           1,
		Failed:           1,
		SuccessRate:      50.0,
		ExecutionTime:    10 * time.Millisecond,
		Results: []types.SimulationResult{
			{
				SimulationName: "IDOR Test PASS",
				Passed:         true,
				Method:         "GET",
				URL:            "http://localhost:8080/api/users/1",
				Duration:       5 * time.Millisecond,
			},
			{
				SimulationName: "JWT Forge Test FAIL",
				Passed:         false,
				Method:         "GET",
				URL:            "http://localhost:8080/api/secrets",
				ExpectedResult: "Status 401 Unauthorized",
				ActualResult:   "Status 200 OK",
				Reason:         "Backend accepted untrusted forged signature",
				Duration:       5 * time.Millisecond,
			},
		},
	}

	reporter := &engine.JUnitReporter{}
	var buf bytes.Buffer

	err := reporter.Generate(&buf, report, nil)
	if err != nil {
		t.Fatalf("Failed to generate JUnit report: %v", err)
	}

	var suites engine.JUnitTestSuites
	if err := xml.Unmarshal(buf.Bytes(), &suites); err != nil {
		t.Fatalf("Generated JUnit report is invalid XML: %v", err)
	}

	if suites.Tests != 2 || suites.Failures != 1 {
		t.Errorf("Expected 2 tests and 1 failure, got tests=%d failures=%d", suites.Tests, suites.Failures)
	}

	if len(suites.TestSuite) != 1 {
		t.Fatalf("Expected 1 testsuite, got %d", len(suites.TestSuite))
	}

	testCases := suites.TestSuite[0].TestCase
	if len(testCases) != 2 {
		t.Fatalf("Expected 2 testcases, got %d", len(testCases))
	}

	var failTC *engine.JUnitTestCase
	for i := range testCases {
		if testCases[i].Failure != nil {
			failTC = &testCases[i]
		}
	}

	if failTC == nil {
		t.Fatalf("Expected at least 1 testcase with a failure element")
	}

	if failTC.Failure.Type != "SecurityControlFailure" {
		t.Errorf("Expected failure type SecurityControlFailure, got %s", failTC.Failure.Type)
	}
}
