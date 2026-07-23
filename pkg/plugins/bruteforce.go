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

	"github.com/suryatk2007/threatsim/pkg/plugins/utils/bruteforce"
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

	path := ParseString(config, "path")
	username := ParseString(config, "username")
	expectedStatusCode := ParseInt(config, "expected_status_code", 0)
	expectedBodyContains := ParseString(config, "expected_body_contains")

	// Now read num_requests
	if _, okReq := config["num_requests"]; !okReq {
		results = append(results, types.SimulationResult{
			SimulationName: simName,
			Passed:         false,
			ExpectedResult: "num_requests should be defined",
			ActualResult:   "Missing num_requests in config",
			Reason:         "Plugin misconfigured",
		})
		return results
	}

	
	numRequests := ParseInt(config, "num_requests", 0)

	if path == "" || username == "" {
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

	wordlistPath := ParseString(config, "wordlist_path")
	usernameField := ParseString(config, "username_field")
	if usernameField == "" {
		usernameField = "username"
	}
	passwordField := ParseString(config, "password_field")
	if passwordField == "" {
		passwordField = "password"
	}


	passwords, err := bruteforce.GeneratePasswords(numRequests, wordlistPath)
	if err != nil {
		results = append(results, types.SimulationResult{
			SimulationName: simName,
			Passed:         false,
			ExpectedResult: "Valid wordlist_path",
			ActualResult:   "Failed reading wordlist",
			Reason:         err.Error(),
		})
		return results
	}

	targetURL := fmt.Sprintf("%s/%s", ctx.TargetURL, strings.TrimLeft(path, "/"))

	var wg sync.WaitGroup
	var mu sync.Mutex
	var pwdSucceeded string
	var expectedHit bool
	var successCount int
	var lastErr error

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
					usernameField: username,
					passwordField: pwd,
				}
				bodyBytes, _ := json.Marshal(payload)

				req, err := http.NewRequest("POST", targetURL, bytes.NewBuffer(bodyBytes))
				if err != nil {
					continue
				}
				req.Header.Set("Content-Type", "application/json")

				resp, err := ctx.Client.Do(req)
				if err != nil {
					mu.Lock()
					lastErr = err
					mu.Unlock()
					continue
				}
				mu.Lock()
				successCount++
				mu.Unlock()
				
				respBodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 5*1024*1024))
				bodyString := string(respBodyBytes)
				resp.Body.Close()

				if resp.StatusCode == 200 || resp.StatusCode == 201 {
					mu.Lock()
					pwdSucceeded = pwd
					mu.Unlock()
				}
				
				isStatusHit := expectedStatusCode > 0 && resp.StatusCode == expectedStatusCode
				isBodyHit := expectedBodyContains != "" && strings.Contains(bodyString, expectedBodyContains)

				if isStatusHit || isBodyHit {
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

	var expectedMessages []string
	if expectedStatusCode > 0 {
		expectedMessages = append(expectedMessages, fmt.Sprintf("status %d", expectedStatusCode))
	}
	if expectedBodyContains != "" {
		expectedMessages = append(expectedMessages, fmt.Sprintf("body containing %q", expectedBodyContains))
	}

	if len(expectedMessages) > 0 {
		res.ExpectedResult = fmt.Sprintf("Application should return %s (e.g., Bruteforce Detected)", strings.Join(expectedMessages, " or "))
	} else {
		res.ExpectedResult = "All login attempts safely rejected (No 200 OK)"
	}

	hasExpectations := len(expectedMessages) > 0

	if successCount == 0 && lastErr != nil {
		res.Passed = false
		res.IsError = true
		res.ActualResult = "Target Server Unreachable"
		res.Reason = fmt.Sprintf("HTTP request failed: %v", lastErr)
	} else if pwdSucceeded != "" {
		res.Passed = false
		res.ActualResult = "200 OK Received"
		res.Reason = fmt.Sprintf("SECURITY BEHAVIOR VIOLATED: Password '%s' succeeded!", pwdSucceeded)
	} else if hasExpectations && !expectedHit {
		res.Passed = false
		res.ActualResult = "Expected behavior not encountered"
		res.Reason = fmt.Sprintf("Executed %d requests but never received %s. Security control failed.", numRequests, strings.Join(expectedMessages, " or "))
	} else if hasExpectations && expectedHit {
		res.Passed = true
		res.ActualResult = "Security control triggered"
		res.Reason = "Security control successfully detected and blocked bruteforce."
	} else {
		res.Passed = true
		res.ActualResult = "All logins rejected"
		res.Reason = "Login endpoints safely rejected all invalid attempts."
	}


	return []types.SimulationResult{res}
}
