package utils

import (
	"bytes"
	"encoding/json"
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

// extract is a simple helper to get values from a JSON map using dot notation
func extract(data map[string]interface{}, path string) (string, bool) {
	keys := strings.Split(path, ".")
	var current interface{} = data
	for i, key := range keys {
		m, ok := current.(map[string]interface{})
		if !ok {
			return "", false
		}
		if i == len(keys)-1 {
			val, exists := m[key]
			if !exists {
				return "", false
			}
			return fmt.Sprintf("%v", val), true
		}
		current = m[key]
	}
	return "", false
}

// doAuthAndExtract performs the login request and extracts the required JSON paths
func doAuthAndExtract(client *http.Client, url, payload, tokenPath, idPath string) (token, id string, err error) {
	req, _ := http.NewRequest("POST", url, bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		return "", "", fmt.Errorf("failed to parse auth response JSON")
	}

	token, ok := extract(data, tokenPath)
	if !ok {
		return "", "", fmt.Errorf("token path %q not found in response", tokenPath)
	}

	if idPath != "" {
		id, ok = extract(data, idPath)
		if !ok {
			return "", "", fmt.Errorf("id path %q not found in response", idPath)
		}
	}

	return token, id, nil
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
	_, userA_ID, err := doAuthAndExtract(cfg.Client, authURL, cfg.UserAPayload, cfg.TokenJSONPath, cfg.IDJSONPath)
	if err != nil {
		res.Passed = false
		res.ActualResult = "User A Auth Failed"
		res.Reason = err.Error()
		return []types.SimulationResult{res}
	}

	// 2. Login as User B and get Token
	userB_Token, _, err := doAuthAndExtract(cfg.Client, authURL, cfg.UserBPayload, cfg.TokenJSONPath, "")
	if err != nil {
		res.Passed = false
		res.ActualResult = "User B Auth Failed"
		res.Reason = err.Error()
		return []types.SimulationResult{res}
	}

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
