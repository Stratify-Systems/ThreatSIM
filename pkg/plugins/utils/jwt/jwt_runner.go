package jwt

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/suryatk2007/threatsim/pkg/plugins/utils/auth"
	"github.com/suryatk2007/threatsim/pkg/types"
)

type StateSetter interface {
	SetState(key, val string)
}

type JWTForgeConfig struct {
	BaseURL              string
	AuthPath             string
	AuthPayload          string
	TokenJSONPath        string
	TargetPath           string
	AttackMode           string // "signature_tamper", "alg_none", "weak_secret"
	WeakSecret           string // optional custom secret for weak_secret attack mode
	ForgeClaims          map[string]interface{}
	ExpectedStatusCode   int
	ExpectedBodyContains string
	Client               *http.Client
	StateSetter          StateSetter
}

// RunJWTForgeValidation executes the JWT validation attack based on the specified AttackMode.
func RunJWTForgeValidation(simName string, cfg JWTForgeConfig) []types.SimulationResult {
	start := time.Now()
	var expectedMessages []string
	if cfg.ExpectedStatusCode > 0 {
		expectedMessages = append(expectedMessages, fmt.Sprintf("status %d", cfg.ExpectedStatusCode))
	}
	if cfg.ExpectedBodyContains != "" {
		expectedMessages = append(expectedMessages, fmt.Sprintf("body containing %q", cfg.ExpectedBodyContains))
	}

	res := types.SimulationResult{
		SimulationName: simName,
		Method:         "GET",
		ExpectedResult: fmt.Sprintf("Application should return %s (e.g., Token Rejected)", strings.Join(expectedMessages, " or ")),
	}

	authURL := fmt.Sprintf("%s/%s", cfg.BaseURL, strings.TrimLeft(cfg.AuthPath, "/"))

	// 1. Login and get valid token
	authRes, err := auth.AuthenticateAndExtract(cfg.Client, auth.AuthConfig{
		URL:           authURL,
		Payload:       cfg.AuthPayload,
		TokenJSONPath: cfg.TokenJSONPath,
	})
	if err != nil {
		res.Passed = false
		res.IsError = true
		res.ActualResult = "Auth Failed"
		res.Reason = err.Error()
		return []types.SimulationResult{res}
	}
	token := authRes.Token

	if cfg.StateSetter != nil && token != "" {
		cfg.StateSetter.SetState("jwt_token", token)
	}

	// 2. Select Attack Mode (default: signature_tamper)
	attackMode := strings.ToLower(cfg.AttackMode)
	if attackMode == "" {
		attackMode = "signature_tamper"
	}

	var forgedTokens []string
	switch attackMode {
	case "alg_none":
		forged, err := ForgeAlgNone(token, cfg.ForgeClaims)
		if err != nil {
			res.Passed = false
			res.IsError = true
			res.ActualResult = "Forge Alg None Failed"
			res.Reason = err.Error()
			return []types.SimulationResult{res}
		}
		forgedTokens = append(forgedTokens, forged)

	case "weak_secret":
		forgedList, err := ForgeWeakSecret(token, cfg.ForgeClaims, cfg.WeakSecret)
		if err != nil {
			res.Passed = false
			res.IsError = true
			res.ActualResult = "Forge Weak Secret Failed"
			res.Reason = err.Error()
			return []types.SimulationResult{res}
		}
		forgedTokens = forgedList

	case "signature_tamper":
		fallthrough
	default:
		forged, err := ForgeSignatureTamper(token, cfg.ForgeClaims)
		if err != nil {
			res.Passed = false
			res.IsError = true
			res.ActualResult = "Forge Signature Tamper Failed"
			res.Reason = err.Error()
			return []types.SimulationResult{res}
		}
		forgedTokens = append(forgedTokens, forged)
	}


	// 3. Test forged token(s) against target endpoint concurrently in parallel
	targetURL := fmt.Sprintf("%s/%s", cfg.BaseURL, strings.TrimLeft(cfg.TargetPath, "/"))
	res.URL = targetURL

	var succeededToken string
	var expectedHit bool
	var mu sync.Mutex
	var wg sync.WaitGroup
	var successCount int
	var lastErr error

	for _, t := range forgedTokens {
		wg.Add(1)
		go func(tokenStr string) {
			defer wg.Done()
			req, err := http.NewRequest("GET", targetURL, nil)
			if err != nil {
				return
			}
			req.Header.Set("Authorization", "Bearer "+tokenStr)

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
			bodyString := string(bodyBytes)
			resp.Body.Close()

			isStatusHit := cfg.ExpectedStatusCode > 0 && resp.StatusCode == cfg.ExpectedStatusCode
			isBodyHit := cfg.ExpectedBodyContains != "" && strings.Contains(bodyString, cfg.ExpectedBodyContains)

			mu.Lock()
			defer mu.Unlock()
			if isStatusHit || isBodyHit {
				expectedHit = true
			} else if resp.StatusCode == 200 || resp.StatusCode == 201 {
				succeededToken = tokenStr
			}
		}(t)
	}

	wg.Wait()

	res.Duration = time.Since(start)

	if successCount == 0 && lastErr != nil {
		res.Passed = false
		res.IsError = true
		res.ActualResult = "Target Server Unreachable"
		res.Reason = fmt.Sprintf("HTTP request failed: %v", lastErr)
	} else if succeededToken != "" {
		res.Passed = false
		res.ActualResult = "Status 200 OK Received"
		res.Reason = fmt.Sprintf("SECURITY VIOLATION: Attack mode %q forged JWT token succeeded!", attackMode)
	} else if expectedHit {
		res.Passed = true
		res.ActualResult = "Security control triggered"
		res.Reason = fmt.Sprintf("Security control successfully rejected %q forged JWT attack.", attackMode)
	} else {
		res.Passed = false
		res.ActualResult = "Expected rejection not encountered"
		res.Reason = fmt.Sprintf("Expected %s, but security control failed to trigger for attack mode %q.", strings.Join(expectedMessages, " or "), attackMode)
	}


	return []types.SimulationResult{res}
}
