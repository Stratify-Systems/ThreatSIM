package rate_limit

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/suryatk2007/threatsim/pkg/types"
)

type RateLimitConfig struct {
	BaseURL              string
	Path                 string
	Method               string
	NumRequests          int
	Concurrency          int
	Headers              map[string]string
	Body                 string
	ExpectedStatusCode   int
	ExpectedBodyContains string
	Client               *http.Client
}

// RunRateLimitValidation tests API endpoint throttling with concurrent bursts.
func RunRateLimitValidation(simName string, cfg RateLimitConfig) []types.SimulationResult {
	start := time.Now()
	if cfg.Client == nil {
		cfg.Client = http.DefaultClient
	}

	method := strings.ToUpper(strings.TrimSpace(cfg.Method))
	if method == "" {
		method = "GET"
	}

	numRequests := cfg.NumRequests
	if numRequests <= 0 {
		numRequests = 20
	}
	if numRequests > 1000 {
		return []types.SimulationResult{{
			SimulationName: simName,
			Passed:         false,
			ExpectedResult: "num_requests <= 1000",
			ActualResult:   fmt.Sprintf("num_requests %d exceeds maximum safe limit of 1000", numRequests),
			Reason:         "Safety limit exceeded. Aborting to prevent accidental DoS.",
		}}
	}

	concurrency := cfg.Concurrency
	if concurrency <= 0 {
		concurrency = 10
	}
	if concurrency > numRequests {
		concurrency = numRequests
	}

	expectedStatusCode := cfg.ExpectedStatusCode
	if expectedStatusCode == 0 && cfg.ExpectedBodyContains == "" {
		expectedStatusCode = 429
	}

	targetURL := fmt.Sprintf("%s/%s", cfg.BaseURL, strings.TrimLeft(cfg.Path, "/"))

	var expectedMessages []string
	if expectedStatusCode > 0 {
		expectedMessages = append(expectedMessages, fmt.Sprintf("status %d", expectedStatusCode))
	}
	if cfg.ExpectedBodyContains != "" {
		expectedMessages = append(expectedMessages, fmt.Sprintf("body containing %q", cfg.ExpectedBodyContains))
	}

	res := types.SimulationResult{
		SimulationName: simName,
		Method:         method,
		URL:            targetURL,
		ExpectedResult: fmt.Sprintf("Application should trigger rate limiting returning %s", strings.Join(expectedMessages, " or ")),
	}

	jobs := make(chan int, numRequests)
	for i := 0; i < numRequests; i++ {
		jobs <- i
	}
	close(jobs)

	var wg sync.WaitGroup
	var mu sync.Mutex
	var expectedHit bool
	var lastStatus int
	var successCount int
	var lastErr error

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range jobs {
				var reqBody io.Reader
				if cfg.Body != "" {
					reqBody = bytes.NewBufferString(cfg.Body)
				}

				req, err := http.NewRequest(method, targetURL, reqBody)
				if err != nil {
					continue
				}

				for k, v := range cfg.Headers {
					req.Header.Set(k, v)
				}

				resp, err := cfg.Client.Do(req)
				if err != nil {
					mu.Lock()
					lastErr = err
					mu.Unlock()
					continue
				}

				mu.Lock()
				successCount++
				mu.Unlock()

				bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 5*1024*1024))
				bodyString := string(bodyBytes)
				resp.Body.Close()

				isStatusHit := expectedStatusCode > 0 && resp.StatusCode == expectedStatusCode
				isBodyHit := cfg.ExpectedBodyContains != "" && strings.Contains(bodyString, cfg.ExpectedBodyContains)

				mu.Lock()
				lastStatus = resp.StatusCode
				if isStatusHit || isBodyHit {
					expectedHit = true
				}
				mu.Unlock()
			}
		}()
	}

	wg.Wait()
	res.Duration = time.Since(start)

	if successCount == 0 && lastErr != nil {
		res.Passed = false
		res.IsError = true
		res.ActualResult = "Target Server Unreachable"
		res.Reason = fmt.Sprintf("HTTP request failed: %v", lastErr)
	} else if expectedHit {
		res.Passed = true
		res.ActualResult = "Rate limiting triggered"
		res.Reason = fmt.Sprintf("API endpoint rate limiting successfully triggered after burst of %d requests.", numRequests)
	} else {
		res.Passed = false
		res.ActualResult = fmt.Sprintf("Status %d", lastStatus)
		res.Reason = fmt.Sprintf("Executed %d burst requests but rate limiting was never triggered (Expected %s).", numRequests, strings.Join(expectedMessages, " or "))
	}


	return []types.SimulationResult{res}
}
