package idor

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/suryatk2007/threatsim/pkg/plugins/utils/auth"
	"github.com/suryatk2007/threatsim/pkg/types"
)

type IDORConfig struct {
	BaseURL              string
	AuthPath             string
	UserAPayload         string
	UserBPayload         string
	TokenJSONPath        string
	IDJSONPath           string
	TargetMethod         string // GET, POST, PUT, DELETE, PATCH (default: GET)
	TargetPath           string
	TargetPayload        string // JSON payload with optional {id} placeholder
	ExpectedStatusCode   int
	ExpectedBodyContains string
	Client               *http.Client
}

// RunIDORValidation executes the IDOR attack workflow across HTTP methods with {id} substitution.
func RunIDORValidation(simName string, cfg IDORConfig) []types.SimulationResult {
	start := time.Now()
	var expectedMessages []string
	if cfg.ExpectedStatusCode > 0 {
		expectedMessages = append(expectedMessages, fmt.Sprintf("status %d", cfg.ExpectedStatusCode))
	}
	if cfg.ExpectedBodyContains != "" {
		expectedMessages = append(expectedMessages, fmt.Sprintf("body containing %q", cfg.ExpectedBodyContains))
	}

	method := strings.ToUpper(strings.TrimSpace(cfg.TargetMethod))
	if method == "" {
		method = "GET"
	}

	res := types.SimulationResult{
		SimulationName: simName,
		Method:         method,
		ExpectedResult: fmt.Sprintf("Application should return %s (e.g., IDOR Prevented)", strings.Join(expectedMessages, " or ")),
	}

	authURL := fmt.Sprintf("%s/%s", cfg.BaseURL, strings.TrimLeft(cfg.AuthPath, "/"))

	// 1 & 2. Login as User A and User B concurrently in parallel
	var wg sync.WaitGroup
	var userARes, userBRes *auth.AuthResult
	var errA, errB error

	wg.Add(2)
	go func() {
		defer wg.Done()
		userARes, errA = auth.AuthenticateAndExtract(cfg.Client, auth.AuthConfig{
			URL:           authURL,
			Payload:       cfg.UserAPayload,
			TokenJSONPath: cfg.TokenJSONPath,
			IDJSONPath:    cfg.IDJSONPath,
		})
	}()

	go func() {
		defer wg.Done()
		userBRes, errB = auth.AuthenticateAndExtract(cfg.Client, auth.AuthConfig{
			URL:           authURL,
			Payload:       cfg.UserBPayload,
			TokenJSONPath: cfg.TokenJSONPath,
		})
	}()

	wg.Wait()

	if errA != nil {
		res.Passed = false
		res.ActualResult = "User A Auth Failed"
		res.Reason = errA.Error()
		return []types.SimulationResult{res}
	}

	if errB != nil {
		res.Passed = false
		res.ActualResult = "User B Auth Failed"
		res.Reason = errB.Error()
		return []types.SimulationResult{res}
	}

	userA_ID := userARes.ID
	userB_Token := userBRes.Token

	// 3. Substitute {id} in both TargetPath and TargetPayload
	targetPath := strings.ReplaceAll(cfg.TargetPath, "{id}", userA_ID)
	targetURL := fmt.Sprintf("%s/%s", cfg.BaseURL, strings.TrimLeft(targetPath, "/"))
	res.URL = targetURL

	targetPayload := strings.ReplaceAll(cfg.TargetPayload, "{id}", userA_ID)

	var reqBody io.Reader
	if targetPayload != "" {
		reqBody = bytes.NewBufferString(targetPayload)
	}

	req, err := http.NewRequest(method, targetURL, reqBody)
	if err != nil {
		res.Passed = false
		res.ActualResult = "Request Creation Failed"
		res.Reason = err.Error()
		return []types.SimulationResult{res}
	}

	req.Header.Set("Authorization", "Bearer "+userB_Token)
	if targetPayload != "" {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := cfg.Client.Do(req)
	res.Duration = time.Since(start)

	if err != nil {
		res.Passed = false
		res.ActualResult = "Execution Failed"
		res.Reason = err.Error()
		return []types.SimulationResult{res}
	}
	bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 5*1024*1024))
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
