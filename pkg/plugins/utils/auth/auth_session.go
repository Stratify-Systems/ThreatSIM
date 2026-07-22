package auth

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// AuthConfig defines the parameters required to authenticate and extract session details.
type AuthConfig struct {
	URL           string
	Payload       string
	TokenJSONPath string
	IDJSONPath    string
	Headers       map[string]string
}

// AuthResult contains the extracted credentials and session information.
type AuthResult struct {
	Token    string
	ID       string
	Body     []byte
	Response *http.Response
}

// ExtractJSONPath extracts a value from a nested JSON map using dot notation (e.g. "data.token").
func ExtractJSONPath(data map[string]interface{}, path string) (string, bool) {
	if path == "" {
		return "", false
	}
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

// AuthenticateAndExtract performs an authentication request and extracts tokens/IDs via JSON paths.
func AuthenticateAndExtract(client *http.Client, cfg AuthConfig) (*AuthResult, error) {
	if client == nil {
		client = http.DefaultClient
	}

	req, err := http.NewRequest("POST", cfg.URL, bytes.NewBufferString(cfg.Payload))
	if err != nil {
		return nil, fmt.Errorf("failed to create auth request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	for k, v := range cfg.Headers {
		req.Header.Set(k, v)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("auth request failed: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, 5*1024*1024))
	if err != nil {
		return nil, fmt.Errorf("failed to read auth response body: %w", err)
	}

	result := &AuthResult{
		Body:     bodyBytes,
		Response: resp,
	}

	var data map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &data); err != nil {
		return nil, fmt.Errorf("failed to parse auth response JSON")
	}

	if cfg.TokenJSONPath != "" {
		token, ok := ExtractJSONPath(data, cfg.TokenJSONPath)
		if !ok {
			return nil, fmt.Errorf("token path %q not found in response", cfg.TokenJSONPath)
		}
		result.Token = token
	}

	if cfg.IDJSONPath != "" {
		id, ok := ExtractJSONPath(data, cfg.IDJSONPath)
		if !ok {
			return nil, fmt.Errorf("id path %q not found in response", cfg.IDJSONPath)
		}
		result.ID = id
	}

	return result, nil
}
