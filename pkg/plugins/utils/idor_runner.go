package utils

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/suryatk2007/threatsim/pkg/types"
)

type IDORConfig struct {
	BaseURL            string
	AuthPath           string
	UserAPayload       string
	UserBPayload       string
	TokenJSONPath      string
	IDJSONPath         string
	TargetPath         string
	ExpectedStatusCode int
	ExpectedBodyContains string
	Client             *http.Client
}

// RunIDORValidation executes the IDOR attack workflow
func RunIDORValidation(simName string, cfg IDORConfig) []types.SimulationResult {
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
		ExpectedResult: fmt.Sprintf("Application should return %s (e.g., IDOR Prevented)", strings.Join(expectedMessages, " or ")),
	}

	authURL := fmt.Sprintf("%s/%s", cfg.BaseURL, strings.TrimLeft(cfg.AuthPath, "/"))

	// 1. Login as User A and get ID + Token
	userARes, err := AuthenticateAndExtract(cfg.Client, AuthConfig{
		URL:           authURL,
		Payload:       cfg.UserAPayload,
		TokenJSONPath: cfg.TokenJSONPath,
		IDJSONPath:    cfg.IDJSONPath,
	})
	if err != nil {
		res.Passed = false
		res.ActualResult = "User A Auth Failed"
		res.Reason = err.Error()
		return []types.SimulationResult{res}
	}

	// 2. Login as User B and get Token
	userBRes, err := AuthenticateAndExtract(cfg.Client, AuthConfig{
		URL:           authURL,
		Payload:       cfg.UserBPayload,
		TokenJSONPath: cfg.TokenJSONPath,
	})
	if err != nil {
		res.Passed = false
		res.ActualResult = "User B Auth Failed"
		res.Reason = err.Error()
		return []types.SimulationResult{res}
	}

	userA_ID := userARes.ID
	userB_Token := userBRes.Token

	// 3. Try to access User A's resource as User B
	targetPath := strings.ReplaceAll(cfg.TargetPath, "{id}", userA_ID)
	targetURL := fmt.Sprintf("%s/%s", cfg.BaseURL, strings.TrimLeft(targetPath, "/"))
	res.URL = targetURL

	req, _ := http.NewRequest("GET", targetURL, nil)
	req.Header.Set("Authorization", "Bearer "+userB_Token)

	resp, err := cfg.Client.Do(req)
	res.Duration = time.Since(start)

	if err != nil {
		res.Passed = false
		res.ActualResult = "Execution Failed"
		res.Reason = err.Error()
		return []types.SimulationResult{res}
	}
	bodyBytes, _ := io.ReadAll(resp.Body)
	bodyString := string(bodyBytes)
	resp.Body.Close()

	isStatusHit := cfg.ExpectedStatusCode > 0 && resp.StatusCode == cfg.ExpectedStatusCode
	isBodyHit := cfg.ExpectedBodyContains != "" && strings.Contains(bodyString, cfg.ExpectedBodyContains)

	if isStatusHit || isBodyHit {
		res.Passed = true
		res.ActualResult = "Security control triggered"
		res.Reason = "Security control successfully detected and blocked IDOR attempt."
	} else {
		res.Passed = false
		res.ActualResult = fmt.Sprintf("Status %d", resp.StatusCode)
		res.Reason = fmt.Sprintf("Expected %s, got Status %d. IDOR vulnerability potentially exists!", strings.Join(expectedMessages, " or "), resp.StatusCode)
	}

	return []types.SimulationResult{res}
}
