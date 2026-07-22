package ai

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// BuildSystemPrompt constructs the system prompt for the LLM by embedding schema references and strict generation rules.
func BuildSystemPrompt() string {
	var sb strings.Builder

	sb.WriteString("You are ThreatSim AI, an expert Security-as-Code policy generator.\n")
	sb.WriteString("Your sole task is to convert natural language application security requirements into a single valid ThreatSim YAML simulation file.\n\n")

	sb.WriteString("CRITICAL GENERATION RULES:\n")
	sb.WriteString("1. Output MUST be ONLY valid ThreatSim YAML. No markdown backticks (e.g. ```yaml), no preamble, no explanations, no conversational text.\n")
	sb.WriteString("2. Output MUST start immediately with: version: \"1.0\"\n")
	sb.WriteString("3. Use ONLY documented ThreatSim plugins and request/expected fields. NEVER invent new plugins or schema properties.\n")
	sb.WriteString("4. Choose the appropriate ThreatSim plugin when advanced security logic is described:\n")
	sb.WriteString("   - Cross-tenant access, IDOR, user isolation -> plugin: \"idor\"\n")
	sb.WriteString("   - JWT signature verification, alg none, weak secrets -> plugin: \"jwt_forge\"\n")
	sb.WriteString("   - Login brute-force, password lockout, failed attempts -> plugin: \"bruteforce\"\n")
	sb.WriteString("   - CORS origins, credentials, reflection -> plugin: \"cors_audit\"\n")
	sb.WriteString("   - Public endpoint rate limiting / throttling -> plugin: \"rate_limit\"\n")
	sb.WriteString("5. For standard endpoints, generate standard HTTP simulation entries with request (method, path, headers, query_params, body) and expected (status_code, body_contains, body_regex, headers).\n")
	sb.WriteString("6. Ensure every simulation entry has a clear, descriptive 'name'.\n\n")

	// Attempt to dynamically load authoritative schema specifications from schemas/
	schemaData := loadSchemaFiles()
	if schemaData != "" {
		sb.WriteString("AUTHORITATIVE THREATSIM SCHEMA DEFINITIONS:\n")
		sb.WriteString(schemaData)
		sb.WriteString("\n")
	}

	return sb.String()
}

// loadSchemaFiles dynamically reads simulation schema files from schemas/ if available.
func loadSchemaFiles() string {
	var sb strings.Builder

	schemaPaths := []string{
		"schemas/simulation.yaml",
		"schemas/plugins/idor.yaml",
		"schemas/plugins/jwt_forge.yaml",
		"schemas/plugins/bruteforce.yaml",
		"schemas/plugins/cors_audit.yaml",
		"schemas/plugins/rate_limit.yaml",
	}

	for _, p := range schemaPaths {
		if data, err := os.ReadFile(p); err == nil && len(data) > 0 {
			sb.WriteString(fmt.Sprintf("--- Schema: %s ---\n", filepath.Base(p)))
			sb.WriteString(string(data))
			sb.WriteString("\n\n")
		}
	}

	return sb.String()
}

// BuildUserPrompt wraps the user's natural language requirements.
func BuildUserPrompt(userDescription string) string {
	return fmt.Sprintf("Convert the following application security requirements into a complete ThreatSim simulation YAML:\n\n%s", strings.TrimSpace(userDescription))
}
