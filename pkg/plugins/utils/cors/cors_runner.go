package cors

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/suryatk2007/threatsim/pkg/types"
)

type CORSAuditConfig struct {
	BaseURL                  string
	Path                     string
	Method                   string
	CustomOrigin             string
	TestNullOrigin           bool
	ExpectedAllowCredentials bool
	ExpectedStatusCode       int
	Client                   *http.Client
}

// RunCORSAuditValidation audits CORS headers against untrusted origin requests.
func RunCORSAuditValidation(simName string, cfg CORSAuditConfig) []types.SimulationResult {
	start := time.Now()
	if cfg.Client == nil {
		cfg.Client = http.DefaultClient
	}

	method := strings.ToUpper(strings.TrimSpace(cfg.Method))
	if method == "" {
		method = "GET"
	}

	customOrigin := strings.TrimSpace(cfg.CustomOrigin)
	if customOrigin == "" {
		customOrigin = "https://attacker.com"
	}

	originsToTest := []string{customOrigin}
	if cfg.TestNullOrigin {
		originsToTest = append(originsToTest, "null")
	}

	targetURL := fmt.Sprintf("%s/%s", cfg.BaseURL, strings.TrimLeft(cfg.Path, "/"))

	res := types.SimulationResult{
		SimulationName: simName,
		Method:         method,
		URL:            targetURL,
		ExpectedResult: "Application should strictly enforce CORS rules and reject untrusted origins.",
	}

	var violations []string
	var mu sync.Mutex
	var wg sync.WaitGroup
	var successCount int
	var lastErr error

	for _, origin := range originsToTest {
		wg.Add(1)
		go func(testOrigin string) {
			defer wg.Done()

			// 1. Send Preflight OPTIONS Request
			preflightReq, err := http.NewRequest("OPTIONS", targetURL, nil)
			if err == nil {
				preflightReq.Header.Set("Origin", testOrigin)
				preflightReq.Header.Set("Access-Control-Request-Method", method)

				preflightResp, err := cfg.Client.Do(preflightReq)
				if err == nil {
					mu.Lock()
					successCount++
					mu.Unlock()
					acao := preflightResp.Header.Get("Access-Control-Allow-Origin")
					acac := strings.ToLower(preflightResp.Header.Get("Access-Control-Allow-Credentials"))
					preflightResp.Body.Close()

					if acao == testOrigin && acac == "true" {
						mu.Lock()
						violations = append(violations, fmt.Sprintf("Preflight OPTIONS allowed untrusted origin %q with credentials (Access-Control-Allow-Credentials: true)", testOrigin))
						mu.Unlock()
					} else if acao == "*" && acac == "true" {
						mu.Lock()
						violations = append(violations, "Preflight OPTIONS returned wildcard origin '*' with credentials enabled")
						mu.Unlock()
					} else if acao == testOrigin {
						mu.Lock()
						violations = append(violations, fmt.Sprintf("Preflight OPTIONS dynamically reflected untrusted origin %q", testOrigin))
						mu.Unlock()
					}
				}
			}

			// 2. Send Standard HTTP Request with Origin Header
			req, err := http.NewRequest(method, targetURL, nil)
			if err != nil {
				return
			}
			req.Header.Set("Origin", testOrigin)

			resp, err := cfg.Client.Do(req)
			if err != nil {
				mu.Lock()
				lastErr = err
				mu.Unlock()
				return
			}
			mu.Lock()
			successCount++
			mu.Unlock()

			bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 5*1024*1024))
			_ = string(bodyBytes)
			resp.Body.Close()

			acao := resp.Header.Get("Access-Control-Allow-Origin")
			acac := strings.ToLower(resp.Header.Get("Access-Control-Allow-Credentials"))

			if acao == testOrigin && acac == "true" {
				mu.Lock()
				violations = append(violations, fmt.Sprintf("Actual request allowed untrusted origin %q with credentials (Access-Control-Allow-Credentials: true)", testOrigin))
				mu.Unlock()
			} else if acao == "*" && acac == "true" {
				mu.Lock()
				violations = append(violations, "Actual request returned wildcard origin '*' with credentials enabled")
				mu.Unlock()
			} else if acao == testOrigin {
				mu.Lock()
				violations = append(violations, fmt.Sprintf("Actual request dynamically reflected untrusted origin %q", testOrigin))
				mu.Unlock()
			}
		}(origin)
	}

	wg.Wait()
	res.Duration = time.Since(start)

	if successCount == 0 && lastErr != nil {
		res.Passed = false
		res.IsError = true
		res.ActualResult = "Target Server Unreachable"
		res.Reason = fmt.Sprintf("HTTP request failed: %v", lastErr)
	} else if len(violations) > 0 {
		res.Passed = false
		res.ActualResult = "Insecure CORS Policy Detected"
		res.Reason = fmt.Sprintf("CORS Security Violation: %s", strings.Join(violations, " | "))
	} else {
		res.Passed = true
		res.ActualResult = "CORS Policy Enforced"
		res.Reason = "Backend strictly enforced CORS policy and rejected untrusted origins."
	}


	return []types.SimulationResult{res}
}
