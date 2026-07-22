package ai

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// OpenAPIDoc represents a simplified structure for extracting endpoints and security schemes from OpenAPI v2/v3 specs.
type OpenAPIDoc struct {
	OpenAPI    string                 `json:"openapi" yaml:"openapi"`
	Swagger    string                 `json:"swagger" yaml:"swagger"`
	Info       map[string]interface{} `json:"info" yaml:"info"`
	Paths      map[string]interface{} `json:"paths" yaml:"paths"`
	Components map[string]interface{} `json:"components" yaml:"components"`
}

// ConvertOpenAPIToPrompt parses an OpenAPI spec file (JSON or YAML) and formats a structured prompt for ThreatSim AI generation.
func ConvertOpenAPIToPrompt(specFilePath string) (string, error) {
	data, err := os.ReadFile(specFilePath)
	if err != nil {
		return "", fmt.Errorf("failed to read OpenAPI spec file %q: %w", specFilePath, err)
	}

	var doc OpenAPIDoc
	// Attempt JSON unmarshaling first, fallback to YAML
	if jsonErr := json.Unmarshal(data, &doc); jsonErr != nil {
		if yamlErr := yaml.Unmarshal(data, &doc); yamlErr != nil {
			return "", fmt.Errorf("failed to parse OpenAPI spec as JSON or YAML: %v", yamlErr)
		}
	}

	if len(doc.Paths) == 0 {
		return "", fmt.Errorf("no endpoint paths found in OpenAPI spec %q", specFilePath)
	}

	var sb strings.Builder
	sb.WriteString("OpenAPI Specification Security Validation Suite Requirements:\n\n")

	if title, ok := doc.Info["title"].(string); ok {
		sb.WriteString(fmt.Sprintf("API Title: %s\n", title))
	}

	sb.WriteString("\nEndpoints to audit and validate:\n")

	for pathStr, pathObj := range doc.Paths {
		methodsMap, ok := pathObj.(map[string]interface{})
		if !ok {
			continue
		}

		for method, methodObj := range methodsMap {
			upperMethod := strings.ToUpper(method)
			if upperMethod != "GET" && upperMethod != "POST" && upperMethod != "PUT" && upperMethod != "DELETE" && upperMethod != "PATCH" {
				continue
			}

			summary := ""
			if mDetail, ok := methodObj.(map[string]interface{}); ok {
				if sVal, ok := mDetail["summary"].(string); ok {
					summary = sVal
				}
			}

			if summary != "" {
				sb.WriteString(fmt.Sprintf("- Method %s %s: %s\n", upperMethod, pathStr, summary))
			} else {
				sb.WriteString(fmt.Sprintf("- Method %s %s\n", upperMethod, pathStr))
			}
		}
	}

	sb.WriteString("\nSecurity Requirements:\n")
	sb.WriteString("1. Generate JWT forgery validation tests (`jwt_forge`) for protected endpoints.\n")
	sb.WriteString("2. Generate IDOR cross-tenant isolation tests (`idor`) for path parameter endpoints containing user or resource IDs.\n")
	sb.WriteString("3. Generate bruteforce rate-limiting tests (`bruteforce`) for authentication/login endpoints.\n")
	sb.WriteString("4. Generate CORS origin reflection audit tests (`cors_audit`) for public API routes.\n")
	sb.WriteString("5. Generate endpoint throttling tests (`rate_limit`) for public endpoints.\n")
	sb.WriteString("6. Generate standard HTTP status code checks (401 Unauthorized, 403 Forbidden) for unauthenticated access.\n")

	return sb.String(), nil
}
