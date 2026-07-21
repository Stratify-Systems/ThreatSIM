package plugins

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/suryatk2007/threatsim/pkg/payloads"
	"github.com/suryatk2007/threatsim/pkg/types"
)

func init() {
	Register(&SQLiPlugin{})
}

type SQLiPlugin struct{}

func (s *SQLiPlugin) Name() string {
	return "sqli"
}

func (s *SQLiPlugin) Execute(simName string, ctx Context, config map[string]interface{}) []types.SimulationResult {
	var results []types.SimulationResult

	path, _ := config["path"].(string)
	method, _ := config["method"].(string)
	if method == "" {
		method = "GET"
	}

	if path == "" {
		results = append(results, types.SimulationResult{
			SimulationName: simName,
			Passed:         false,
			ExpectedResult: "Valid configuration (path)",
			ActualResult:   "Missing path in config",
			Reason:         "Plugin misconfigured",
		})
		return results
	}

	baseReqURL := fmt.Sprintf("%s/%s", ctx.TargetURL, strings.TrimLeft(path, "/"))

	// Extract optional base parameters
	var baseQueryParams map[string]interface{}
	if q, ok := config["query_params"].(map[string]interface{}); ok {
		baseQueryParams = q
	}

	var baseBody map[string]interface{}
	if b, ok := config["body"].(map[string]interface{}); ok {
		baseBody = b
	}

	sqliPayloads := payloads.Get("sqli")

	// 1. Automatically Fuzz all Query Parameters
	for k := range baseQueryParams {
		for _, payload := range sqliPayloads {
			start := time.Now()
			res := types.SimulationResult{
				SimulationName: fmt.Sprintf("%s [SQLi -> Query:%s] Payload:%s", simName, k, payload),
				ExpectedResult: "Status Code: 4xx (Graceful rejection)",
				Method:         method,
			}

			reqURL, _ := url.Parse(baseReqURL)
			q := reqURL.Query()
			// Copy all base params
			for bK, bV := range baseQueryParams {
				q.Add(bK, fmt.Sprintf("%v", bV))
			}
			// Override the target parameter with the malicious payload
			q.Set(k, payload)
			reqURL.RawQuery = q.Encode()
			
			res.URL = reqURL.String()

			req, _ := http.NewRequest(method, reqURL.String(), nil)
			resp, err := ctx.Client.Do(req)
			res.Duration = time.Since(start)

			if err != nil {
				res.Passed = false
				res.ActualResult = "Request Failed"
				res.Reason = err.Error()
				results = append(results, res)
				continue
			}

			bodyBytes, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			bodyStr := string(bodyBytes)

			res.ActualResult = fmt.Sprintf("Status Code: %d", resp.StatusCode)

			// SQLi Detection Logic: 
			// A 500 error or SQL-related strings in body usually indicates a successful injection causing a syntax error.
			if resp.StatusCode >= 500 || strings.Contains(strings.ToLower(bodyStr), "sql syntax") {
				res.Passed = false
				res.Reason = "EXPECTED SECURITY BEHAVIOR VIOLATED: Endpoint threw a 500 or SQL syntax error, indicating unhandled input validation."
			} else {
				res.Passed = true
				res.Reason = "Injection safely handled."
			}

			results = append(results, res)
		}
	}

	// 2. Automatically Fuzz all Body Parameters (JSON)
	for k := range baseBody {
		for _, payload := range sqliPayloads {
			start := time.Now()
			res := types.SimulationResult{
				SimulationName: fmt.Sprintf("%s [SQLi -> Body:%s] Payload:%s", simName, k, payload),
				ExpectedResult: "Status Code: 4xx (Graceful rejection)",
				Method:         method,
				URL:            baseReqURL,
			}

			// Copy base body and inject payload
			newBody := make(map[string]interface{})
			for bK, bV := range baseBody {
				newBody[bK] = bV
			}
			newBody[k] = payload // Inject payload into exactly one field at a time!

			bodyBytes, _ := json.Marshal(newBody)
			req, _ := http.NewRequest(method, baseReqURL, bytes.NewBuffer(bodyBytes))
			req.Header.Set("Content-Type", "application/json")

			resp, err := ctx.Client.Do(req)
			res.Duration = time.Since(start)

			if err != nil {
				res.Passed = false
				res.ActualResult = "Request Failed"
				res.Reason = err.Error()
				results = append(results, res)
				continue
			}

			respBodyBytes, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			bodyStr := string(respBodyBytes)

			res.ActualResult = fmt.Sprintf("Status Code: %d", resp.StatusCode)

			if resp.StatusCode >= 500 || strings.Contains(strings.ToLower(bodyStr), "sql syntax") {
				res.Passed = false
				res.Reason = "EXPECTED SECURITY BEHAVIOR VIOLATED: Endpoint threw a 500 or SQL syntax error, indicating unhandled input validation."
			} else {
				res.Passed = true
				res.Reason = "Injection safely handled."
			}

			results = append(results, res)
		}
	}

	// 3. Fallback if no params/body were provided
	if len(baseQueryParams) == 0 && len(baseBody) == 0 {
		results = append(results, types.SimulationResult{
			SimulationName: simName,
			Passed:         false,
			ExpectedResult: "Valid configuration",
			ActualResult:   "No injection points provided",
			Reason:         "Must provide 'query_params' or 'body' in config to fuzz",
		})
	}

	return results
}
