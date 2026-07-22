package unit

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/suryatk2007/threatsim/pkg/ai"
)

func TestSanitizeYAML(t *testing.T) {
	t.Parallel()

	// 1. Markdown code block with ```yaml
	raw1 := "Here is your file:\n```yaml\nversion: \"1.0\"\nsimulations:\n  - name: \"Test\"\n    request:\n      method: \"GET\"\n      path: \"/\"\n    expected:\n      status_code: 200\n```\nHope this helps!"
	clean1 := ai.SanitizeYAML(raw1)
	if os.Getenv("DEBUG") == "" && clean1[:14] != "version: \"1.0\"" {
		t.Errorf("Expected cleaned YAML to start with version: \"1.0\", got %q", clean1)
	}

	// 2. Raw YAML without markdown
	raw2 := "version: \"1.0\"\nsimulations:\n  - name: \"Raw Test\"\n    request:\n      method: \"GET\"\n      path: \"/\"\n    expected:\n      status_code: 200"
	clean2 := ai.SanitizeYAML(raw2)
	if clean2 != raw2 {
		t.Errorf("Expected raw YAML to remain unchanged, got %q", clean2)
	}
}

func TestLoadConfigFromEnv(t *testing.T) {
	os.Setenv("THREATSIM_AI_PROVIDER", "groq")
	os.Setenv("THREATSIM_AI_API_KEY", "gsk_test_key_12345")
	defer os.Unsetenv("THREATSIM_AI_PROVIDER")
	defer os.Unsetenv("THREATSIM_AI_API_KEY")

	cfg, err := ai.LoadConfigFromEnv()
	if err != nil {
		t.Fatalf("LoadConfigFromEnv failed: %v", err)
	}

	if cfg.Provider != "groq" {
		t.Errorf("Expected provider 'groq', got %q", cfg.Provider)
	}
	if cfg.BaseURL != "https://api.groq.com/openai/v1" {
		t.Errorf("Expected Groq base URL, got %q", cfg.BaseURL)
	}
	if cfg.Model != "llama-3.3-70b-versatile" {
		t.Errorf("Expected default Groq model, got %q", cfg.Model)
	}
}

func TestGenerator_GenerateWithMockServer(t *testing.T) {
	t.Parallel()

	validYAML := `version: "1.0"
simulations:
  - name: "Admin Delete Security Test"
    request:
      method: "DELETE"
      path: "/api/admin/user/10"
    expected:
      status_code: 403`

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}

		authHeader := r.Header.Get("Authorization")
		if authHeader != "Bearer test_secret_key" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		resp := ai.ChatCompletionResponse{
			ID:    "chatcmpl-mock-123",
			Model: "mock-model",
			Choices: []ai.ChatCompletionChoice{
				{
					Index: 0,
					Message: ai.ChatMessage{
						Role:    "assistant",
						Content: "```yaml\n" + validYAML + "\n```",
					},
					FinishReason: "stop",
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	client := ai.NewClient(ai.Config{
		Provider: "mock",
		BaseURL:  ts.URL,
		Model:    "mock-model",
		APIKey:   "test_secret_key",
		Timeout:  5 * time.Second,
	})

	generator := ai.NewGenerator(client)
	yamlOutput, def, err := generator.Generate(context.Background(), "Admins only should be able to delete users.")
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if def == nil || len(def.Simulations) != 1 {
		t.Fatalf("Expected 1 simulation defined, got %v", def)
	}

	if def.Simulations[0].Name != "Admin Delete Security Test" {
		t.Errorf("Expected simulation name 'Admin Delete Security Test', got %q", def.Simulations[0].Name)
	}

	if yamlOutput[:14] != "version: \"1.0\"" {
		t.Errorf("Expected sanitized output to start with version, got %q", yamlOutput)
	}
}

func TestGenerator_SelfCorrectionRetry(t *testing.T) {
	t.Parallel()

	attempt := 0
	invalidYAML := `version: "1.0"
simulations:
  - name: "Invalid Entry Missing Method"` // Invalid simulation missing method/expected

	validYAML := `version: "1.0"
simulations:
  - name: "Corrected Simulation Entry"
    request:
      method: "GET"
      path: "/api/test"
    expected:
      status_code: 200`

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempt++
		w.Header().Set("Content-Type", "application/json")

		content := invalidYAML
		if attempt > 1 {
			content = validYAML
		}

		resp := ai.ChatCompletionResponse{
			ID:    "chatcmpl-retry-123",
			Model: "mock-model",
			Choices: []ai.ChatCompletionChoice{
				{
					Index: 0,
					Message: ai.ChatMessage{
						Role:    "assistant",
						Content: content,
					},
					FinishReason: "stop",
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	client := ai.NewClient(ai.Config{
		Provider: "mock",
		BaseURL:  ts.URL,
		Model:    "mock-model",
		APIKey:   "test_secret_key",
		Timeout:  5 * time.Second,
	})

	generator := ai.NewGenerator(client)
	_, def, err := generator.Generate(context.Background(), "Security validation for test endpoint")
	if err != nil {
		t.Fatalf("Expected self-correction retry to succeed on 2nd attempt, but failed: %v", err)
	}

	if attempt != 2 {
		t.Errorf("Expected 2 attempts for self-correction retry, got %d", attempt)
	}

	if def.Simulations[0].Name != "Corrected Simulation Entry" {
		t.Errorf("Expected corrected simulation name, got %q", def.Simulations[0].Name)
	}
}
