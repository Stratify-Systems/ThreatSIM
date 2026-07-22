package unit

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"

	"github.com/suryatk2007/threatsim/pkg/plugins/utils/jwt"
)

func TestForgeSignatureTamper(t *testing.T) {
	t.Parallel()
	// Standard test JWT: {"alg":"HS256","typ":"JWT"}.{"role":"user"}.signature
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	payload := base64.RawURLEncoding.EncodeToString([]byte(`{"role":"user"}`))
	originalToken := header + "." + payload + ".original_sig_123"

	forged, err := jwt.ForgeSignatureTamper(originalToken, map[string]interface{}{"role": "admin"})
	if err != nil {
		t.Fatalf("ForgeSignatureTamper failed: %v", err)
	}

	parts := strings.Split(forged, ".")
	if len(parts) != 3 {
		t.Fatalf("Expected 3-part JWT, got %d parts", len(parts))
	}

	if parts[2] != "original_sig_123" {
		t.Errorf("Expected original signature to be retained, got %q", parts[2])
	}

	payloadBytes, _ := base64.RawURLEncoding.DecodeString(parts[1])
	var p map[string]interface{}
	json.Unmarshal(payloadBytes, &p)

	if p["role"] != "admin" {
		t.Errorf("Expected role 'admin', got %v", p["role"])
	}
}

func TestForgeAlgNone(t *testing.T) {
	t.Parallel()
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	payload := base64.RawURLEncoding.EncodeToString([]byte(`{"role":"user"}`))
	originalToken := header + "." + payload + ".signature"

	forged, err := jwt.ForgeAlgNone(originalToken, map[string]interface{}{"role": "admin"})
	if err != nil {
		t.Fatalf("ForgeAlgNone failed: %v", err)
	}

	parts := strings.Split(forged, ".")
	if len(parts) != 3 || parts[2] != "" {
		t.Fatalf("Expected trailing dot with empty signature (e.g. header.payload.), got %q", forged)
	}

	headerBytes, _ := base64.RawURLEncoding.DecodeString(parts[0])
	var h map[string]interface{}
	json.Unmarshal(headerBytes, &h)

	if h["alg"] != "none" {
		t.Errorf("Expected alg 'none', got %v", h["alg"])
	}
}

func TestForgeWeakSecret(t *testing.T) {
	t.Parallel()
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	payload := base64.RawURLEncoding.EncodeToString([]byte(`{"role":"user"}`))
	originalToken := header + "." + payload + ".signature"

	forgedList, err := jwt.ForgeWeakSecret(originalToken, map[string]interface{}{"role": "admin"}, "secret")
	if err != nil {
		t.Fatalf("ForgeWeakSecret failed: %v", err)
	}

	if len(forgedList) != 1 {
		t.Fatalf("Expected 1 forged token for custom secret, got %d", len(forgedList))
	}

	forged := forgedList[0]
	parts := strings.Split(forged, ".")
	if len(parts) != 3 {
		t.Fatalf("Expected 3-part JWT, got %q", forged)
	}

	// Verify signature matches HS256 with secret "secret"
	signingInput := parts[0] + "." + parts[1]
	expectedSig := jwt.SignHS256(signingInput, "secret")
	if parts[2] != expectedSig {
		t.Errorf("Expected signature %q, got %q", expectedSig, parts[2])
	}
}
