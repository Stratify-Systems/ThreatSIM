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
	var resMu sync.Mutex

	jobs := make(chan string, len(passwords))
	for _, p := range passwords {
		jobs <- p
	}
	close(jobs)

	numWorkers := 5
	if len(passwords) < numWorkers {
		numWorkers = len(passwords)
	}

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for pwd := range jobs {
				start := time.Now()
				
				res := types.SimulationResult{
					SimulationName: fmt.Sprintf("%s [Bruteforce: %s]", simName, pwd),
					ExpectedResult: "Status Code: 401 or 403 (Rejected)", 
					Method:         "POST",
					URL:            targetURL,
				}

				payload := map[string]string{
					"username": username,
					"password": pwd,
				}
				bodyBytes, _ := json.Marshal(payload)

				req, err := http.NewRequest("POST", targetURL, bytes.NewBuffer(bodyBytes))
				if err != nil {
					res.Passed = false
					res.ActualResult = "Request Failed"
					res.Reason = err.Error()
					res.Duration = time.Since(start)
					resMu.Lock()
					results = append(results, res)
					resMu.Unlock()
					continue
				}
				req.Header.Set("Content-Type", "application/json")

				resp, err := ctx.Client.Do(req)
				res.Duration = time.Since(start)

				if err != nil {
					res.Passed = false
					res.ActualResult = "Execution Failed"
					res.Reason = err.Error()
					resMu.Lock()
					results = append(results, res)
					resMu.Unlock()
					continue
				}
				io.Copy(io.Discard, resp.Body)
				resp.Body.Close()

				res.ActualResult = fmt.Sprintf("Status Code: %d", resp.StatusCode)

				// In a security context, if bruteforce returns 200 OK, the test FAILS!
				if resp.StatusCode == 200 || resp.StatusCode == 201 {
					res.Passed = false
					res.Reason = fmt.Sprintf("SECURITY BEHAVIOR VIOLATED: Password '%s' succeeded!", pwd)
				} else {
					res.Passed = true
					res.Reason = "Login safely rejected"
				}

				resMu.Lock()
				results = append(results, res)
				resMu.Unlock()
			}
		}()
	}

	wg.Wait()

	return results
}
