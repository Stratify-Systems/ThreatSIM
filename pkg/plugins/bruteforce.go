package plugins

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/suryatk2007/threatsim/pkg/plugins/utils"
	"github.com/suryatk2007/threatsim/pkg/types"
)

func init() {
	Register(&BruteforcePlugin{})
}

type BruteforcePlugin struct{}

func (b *BruteforcePlugin) Name() string {
	return "bruteforce"
}

func (b *BruteforcePlugin) Description() string {
	return "Validates that authentication endpoints correctly reject invalid credentials by rapidly iterating through a dictionary."
}

func (b *BruteforcePlugin) Execute(simName string, ctx Context, config map[string]interface{}) []types.SimulationResult {
	var results []types.SimulationResult

	path, okPath := config["path"].(string)
	username, okUser := config["username"].(string)

	var expectedStatusCode int
	if escRaw, ok := config["expected_status_code"]; ok {
		switch v := escRaw.(type) {
		case int:
			expectedStatusCode = v
		case float64:
			expectedStatusCode = int(v)
		}
	}

	// Now read num_requests
	numRequestsRaw, okReq := config["num_requests"]
	if !okReq {
		results = append(results, types.SimulationResult{
			SimulationName: simName,
			Passed:         false,
			ExpectedResult: "num_requests should be defined",
			ActualResult:   "Missing num_requests in config",
			Reason:         "Plugin misconfigured",
		})
		return results
	}
	
	var numRequests int
	switch v := numRequestsRaw.(type) {
	case int:
		numRequests = v
	case float64:
		numRequests = int(v)
	default:
		results = append(results, types.SimulationResult{
			SimulationName: simName,
			Passed:         false,
			ExpectedResult: "num_requests should be an integer",
			ActualResult:   "Invalid type for num_requests in config",
			Reason:         "Plugin misconfigured",
		})
		return results
	}

	if !okPath || !okUser || path == "" || username == "" {
		results = append(results, types.SimulationResult{
			SimulationName: simName,
			Passed:         false,
			ExpectedResult: "Valid configuration (path string, username string)",
			ActualResult:   "Missing or invalid path/username in config",
			Reason:         "Plugin misconfigured",
		})
		return results
	}

	// Safety check limit
	if numRequests > 1000 {
		results = append(results, types.SimulationResult{
			SimulationName: simName,
			Passed:         false,
			ExpectedResult: "num_requests <= 1000",
			ActualResult:   fmt.Sprintf("num_requests %d exceeds maximum safe limit of 1000", numRequests),
			Reason:         "Safety limit exceeded. Aborting to prevent accidental DoS.",
		})
		return results
	}

	passwords := utils.GeneratePasswords(numRequests)

	targetURL := fmt.Sprintf("%s/%s", ctx.TargetURL, strings.TrimLeft(path, "/"))

	var wg sync.WaitGroup
	var mu sync.Mutex
	var pwdSucceeded string
	var expectedHit bool

	jobs := make(chan string, len(passwords))
	for _, p := range passwords {
		jobs <- p
	}
	close(jobs)

	numWorkers := 5
	if len(passwords) < numWorkers {
		numWorkers = len(passwords)
	}

	pluginStartTime := time.Now()

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for pwd := range jobs {
				payload := map[string]string{
					"username": username,
					"password": pwd,
				}
				bodyBytes, _ := json.Marshal(payload)

				req, err := http.NewRequest("POST", targetURL, bytes.NewBuffer(bodyBytes))
				if err != nil {
					continue
				}
				req.Header.Set("Content-Type", "application/json")

				resp, err := ctx.Client.Do(req)
				if err != nil {
					continue
				}
				io.Copy(io.Discard, resp.Body)
				resp.Body.Close()

				if resp.StatusCode == 200 || resp.StatusCode == 201 {
					mu.Lock()
					pwdSucceeded = pwd
					mu.Unlock()
				}
				if expectedStatusCode > 0 && resp.StatusCode == expectedStatusCode {
					mu.Lock()
					expectedHit = true
					mu.Unlock()
				}
			}
		}()
	}

	wg.Wait()

	res := types.SimulationResult{
		SimulationName: simName,
		Method:         "POST",
		URL:            targetURL,
		Duration:       time.Since(pluginStartTime),
	}

	if expectedStatusCode > 0 {
		res.ExpectedResult = fmt.Sprintf("Application should return status %d (e.g., Bruteforce Detected)", expectedStatusCode)
	} else {
		res.ExpectedResult = "All login attempts safely rejected (No 200 OK)"
	}

	if pwdSucceeded != "" {
		res.Passed = false
		res.ActualResult = "200 OK Received"
		res.Reason = fmt.Sprintf("SECURITY BEHAVIOR VIOLATED: Password '%s' succeeded!", pwdSucceeded)
	} else if expectedStatusCode > 0 && !expectedHit {
		res.Passed = false
		res.ActualResult = "Expected status not encountered"
		res.Reason = fmt.Sprintf("Executed %d requests but never received status %d. Security control failed.", numRequests, expectedStatusCode)
	} else if expectedStatusCode > 0 && expectedHit {
		res.Passed = true
		res.ActualResult = fmt.Sprintf("Status %d encountered", expectedStatusCode)
		res.Reason = "Security control successfully detected and blocked bruteforce."
	} else {
		res.Passed = true
		res.ActualResult = "All logins rejected"
		res.Reason = "Login endpoints safely rejected all invalid attempts."
	}

	return []types.SimulationResult{res}
}
