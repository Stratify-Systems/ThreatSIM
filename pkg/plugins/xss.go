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
	Register(&XSSPlugin{})
}

type XSSPlugin struct{}

func (x *XSSPlugin) Name() string {
	return "xss"
}

func (x *XSSPlugin) Execute(simName string, ctx Context, config map[string]interface{}) []types.SimulationResult {
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

	var baseQueryParams map[string]interface{}
	if q, ok := config["query_params"].(map[string]interface{}); ok {
		baseQueryParams = q
	}

	var baseBody map[string]interface{}
	if b, ok := config["body"].(map[string]interface{}); ok {
		baseBody = b
	}

	xssPayloads := payloads.Get("xss")

	// 1. Automatically Fuzz all Query Parameters
	for k := range baseQueryParams {
		for _, payload := range xssPayloads {
			start := time.Now()
			res := types.SimulationResult{
				SimulationName: fmt.Sprintf("%s [XSS -> Query:%s] Payload:%s", simName, k, payload),
				ExpectedResult: "Payload sanitized (Not Reflected)",
				Method:         method,
			}

			reqURL, _ := url.Parse(baseReqURL)
			q := reqURL.Query()
			for bK, bV := range baseQueryParams {
				q.Add(bK, fmt.Sprintf("%v", bV))
			}
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

			// XSS Detection Logic:
			// If the raw payload (e.g. <script>) is reflected unescaped, it's a vulnerability.
			if strings.Contains(bodyStr, payload) {
				res.Passed = false
				res.Reason = "VULNERABILITY: Raw unescaped XSS payload was reflected in the response body."
			} else {
				res.Passed = true
				res.Reason = "Payload safely sanitized or not reflected."
			}

			results = append(results, res)
		}
	}

	// 2. Automatically Fuzz all Body Parameters (JSON)
	for k := range baseBody {
		for _, payload := range xssPayloads {
			start := time.Now()
			res := types.SimulationResult{
				SimulationName: fmt.Sprintf("%s [XSS -> Body:%s] Payload:%s", simName, k, payload),
				ExpectedResult: "Payload sanitized (Not Reflected)",
				Method:         method,
				URL:            baseReqURL,
			}

			newBody := make(map[string]interface{})
			for bK, bV := range baseBody {
				newBody[bK] = bV
			}
			newBody[k] = payload 

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

			if strings.Contains(bodyStr, payload) {
				res.Passed = false
				res.Reason = "VULNERABILITY: Raw unescaped XSS payload was reflected in the response body."
			} else {
				res.Passed = true
				res.Reason = "Payload safely sanitized or not reflected."
			}

			results = append(results, res)
		}
	}

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
