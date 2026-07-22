package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Client is an HTTP client for OpenAI-compatible chat completion APIs (Groq, OpenAI, Ollama, etc.).
type Client struct {
	Config Config
	HTTP   *http.Client
}

// LoadConfigFromEnv attempts to load .env file and reads AI configuration settings from environment variables.
func LoadConfigFromEnv() (*Config, error) {
	_ = godotenv.Load() // Ignore error if .env doesn't exist

	provider := strings.ToLower(strings.TrimSpace(os.Getenv("THREATSIM_AI_PROVIDER")))
	baseURL := strings.TrimSpace(os.Getenv("THREATSIM_AI_BASE_URL"))
	model := strings.TrimSpace(os.Getenv("THREATSIM_AI_MODEL"))
	apiKey := strings.TrimSpace(os.Getenv("THREATSIM_AI_API_KEY"))

	// Default provider to groq if unspecified
	if provider == "" {
		provider = "groq"
	}

	// Apply provider-specific defaults
	if provider == "groq" {
		if baseURL == "" {
			baseURL = "https://api.groq.com/openai/v1"
		}
		if model == "" {
			model = "llama-3.3-70b-versatile"
		}
	} else if provider == "openai" {
		if baseURL == "" {
			baseURL = "https://api.openai.com/v1"
		}
		if model == "" {
			model = "gpt-4o"
		}
	}

	if apiKey == "" {
		return nil, fmt.Errorf("AI API key missing. Set THREATSIM_AI_API_KEY in .env or environment")
	}

	if baseURL == "" {
		return nil, fmt.Errorf("AI base URL missing. Set THREATSIM_AI_BASE_URL in .env or environment")
	}

	return &Config{
		Provider: provider,
		BaseURL:  strings.TrimRight(baseURL, "/"),
		Model:    model,
		APIKey:   apiKey,
		Timeout:  60 * time.Second,
	}, nil
}

// NewClient constructs a new AI Client instance.
func NewClient(cfg Config) *Client {
	if cfg.Timeout == 0 {
		cfg.Timeout = 60 * time.Second
	}
	return &Client{
		Config: cfg,
		HTTP: &http.Client{
			Timeout: cfg.Timeout,
		},
	}
}

// CreateChatCompletion sends a completion request to the configured OpenAI-compatible API endpoint.
func (c *Client) CreateChatCompletion(ctx context.Context, req ChatCompletionRequest) (*ChatCompletionResponse, error) {
	if req.Model == "" {
		req.Model = c.Config.Model
	}

	payloadBytes, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal chat completion request: %w", err)
	}

	endpoint := fmt.Sprintf("%s/chat/completions", strings.TrimRight(c.Config.BaseURL, "/"))
	httpReq, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.Config.APIKey))

	resp, err := c.HTTP.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("AI API request failed: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, 10*1024*1024)) // 10MB safety limit
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var apiErr ChatCompletionResponse
		if json.Unmarshal(bodyBytes, &apiErr) == nil && apiErr.Error != nil {
			return nil, fmt.Errorf("AI API error (status %d): %s", resp.StatusCode, apiErr.Error.Message)
		}
		return nil, fmt.Errorf("AI API returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var chatResp ChatCompletionResponse
	if err := json.Unmarshal(bodyBytes, &chatResp); err != nil {
		return nil, fmt.Errorf("failed to parse AI response JSON: %w", err)
	}

	if chatResp.Error != nil {
		return nil, fmt.Errorf("AI API error: %s", chatResp.Error.Message)
	}

	if len(chatResp.Choices) == 0 {
		return nil, fmt.Errorf("AI API returned no completion choices")
	}

	return &chatResp, nil
}
