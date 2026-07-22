package unit

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/suryatk2007/threatsim/pkg/plugins"
	"github.com/suryatk2007/threatsim/pkg/plugins/utils/bruteforce"
)

func TestGeneratePasswords_CustomWordlist(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	wordlistPath := filepath.Join(tmpDir, "custom_passwords.txt")
	content := "alpha123\nbeta456\ngamma789\n"
	if err := os.WriteFile(wordlistPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write temp wordlist: %v", err)
	}

	passwords, err := bruteforce.GeneratePasswords(3, wordlistPath)
	if err != nil {
		t.Fatalf("GeneratePasswords with wordlist failed: %v", err)
	}

	if len(passwords) != 3 {
		t.Fatalf("Expected 3 passwords, got %d", len(passwords))
	}

	if passwords[0] != "alpha123" || passwords[1] != "beta456" || passwords[2] != "gamma789" {
		t.Errorf("Unexpected passwords parsed: %v", passwords)
	}
}

func TestBruteforcePlugin_CustomPayloadFields(t *testing.T) {
	t.Parallel()

	var receivedUserKey, receivedPassKey string
	var mu sync.Mutex

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bodyBytes, _ := io.ReadAll(r.Body)
		var payload map[string]string
		json.Unmarshal(bodyBytes, &payload)

		mu.Lock()
		for k := range payload {
			if k == "custom_user" {
				receivedUserKey = k
			}
			if k == "custom_pass" {
				receivedPassKey = k
			}
		}
		mu.Unlock()

		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"status":"rate_limited"}`))
	}))
	defer ts.Close()

	p, err := plugins.Get("bruteforce")
	if err != nil {
		t.Fatalf("Failed to get bruteforce plugin: %v", err)
	}

	ctx := plugins.Context{
		TargetURL: ts.URL,
		Client:    ts.Client(),
	}

	config := map[string]interface{}{
		"path":                 "/login",
		"username":             "admin@example.com",
		"username_field":       "custom_user",
		"password_field":       "custom_pass",
		"num_requests":         5,
		"expected_status_code": 429,
	}

	results := p.Execute("Custom Bruteforce Field Test", ctx, config)
	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}

	if !results[0].Passed {
		t.Errorf("Expected test to pass with status 429, failed with reason: %s", results[0].Reason)
	}

	if receivedUserKey != "custom_user" {
		t.Errorf("Expected payload field 'custom_user', got %q", receivedUserKey)
	}

	if receivedPassKey != "custom_pass" {
		t.Errorf("Expected payload field 'custom_pass', got %q", receivedPassKey)
	}
}
