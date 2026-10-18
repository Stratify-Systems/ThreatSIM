package ai

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/suryatk2007/threatsim/pkg/types"
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

type OpenAPIRoute struct {
	Method string
	Path   string
}

// LoadOpenAPIRoutes reads the paths and HTTP methods from an OpenAPI document.
func LoadOpenAPIRoutes(specFilePath string) ([]OpenAPIRoute, error) {
	doc, err := loadOpenAPIDoc(specFilePath)
	if err != nil {
		return nil, err
	}

	var routes []OpenAPIRoute
	for pathStr, pathObj := range doc.Paths {
		methodsMap, ok := pathObj.(map[string]interface{})
		if !ok {
			continue
		}
		for method := range methodsMap {
			upperMethod := strings.ToUpper(method)
			switch upperMethod {
			case "GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS":
				routes = append(routes, OpenAPIRoute{Method: upperMethod, Path: pathStr})
			}
		}
	}
	if len(routes) == 0 {
		return nil, fmt.Errorf("no HTTP routes found in OpenAPI spec %q", specFilePath)
	}
	return routes, nil
}

// ValidateDefinitionAgainstOpenAPI rejects generated requests that are absent from the source spec.
func ValidateDefinitionAgainstOpenAPI(def *types.SimulationDefinition, specFilePath string) error {
	routes, err := LoadOpenAPIRoutes(specFilePath)
	if err != nil {
		return err
	}

	var invalid []string
	for _, sim := range def.Simulations {
		for _, route := range simulationRoutes(sim) {
			if !routeExists(routes, route.Method, route.Path) {
				invalid = append(invalid, fmt.Sprintf("%s %s", route.Method, route.Path))
			}
		}
	}
	if len(invalid) > 0 {
		return fmt.Errorf("generated policy contains routes absent from OpenAPI spec: %s", strings.Join(invalid, ", "))
	}
	return nil
}

type simulationRoute struct {
	Method string
	Path   string
}

func simulationRoutes(sim types.Simulation) []simulationRoute {
	if sim.Plugin == "" {
		return []simulationRoute{{Method: sim.Request.Method, Path: sim.Request.Path}}
	}

	config := sim.PluginConfig
	var routes []simulationRoute
	switch sim.Plugin {
	case "idor":
		routes = append(routes,
			simulationRoute{Method: "POST", Path: stringConfig(config, "auth_path")},
			simulationRoute{Method: stringOrDefault(config, "target_method", "GET"), Path: stringConfig(config, "target_path")},
		)
	case "jwt_forge":
		routes = append(routes,
			simulationRoute{Method: "POST", Path: stringConfig(config, "auth_path")},
			simulationRoute{Method: stringOrDefault(config, "target_method", "GET"), Path: stringConfig(config, "target_path")},
		)
	case "bruteforce":
		routes = append(routes, simulationRoute{Method: "POST", Path: stringConfig(config, "path")})
	case "cors_audit":
		routes = append(routes, simulationRoute{Method: stringOrDefault(config, "method", "GET"), Path: stringConfig(config, "path")})
	case "rate_limit":
		routes = append(routes, simulationRoute{Method: stringOrDefault(config, "method", "GET"), Path: stringConfig(config, "path")})
	}
	return routes
}

func stringConfig(config map[string]interface{}, key string) string {
	value, _ := config[key].(string)
	return value
}

func stringOrDefault(config map[string]interface{}, key, fallback string) string {
	value := strings.TrimSpace(stringConfig(config, key))
	if value == "" {
		return fallback
	}
	return strings.ToUpper(value)
}

func routeExists(routes []OpenAPIRoute, method, path string) bool {
	method = strings.ToUpper(strings.TrimSpace(method))
	for _, route := range routes {
		if route.Method == method && openAPIPathMatches(route.Path, path) {
			return true
		}
	}
	return false
}

func openAPIPathMatches(specPath, generatedPath string) bool {
	if specPath == generatedPath {
		return true
	}
	specParts := strings.Split(strings.Trim(specPath, "/"), "/")
	generatedParts := strings.Split(strings.Trim(generatedPath, "/"), "/")
	if len(specParts) != len(generatedParts) {
		return false
	}
	for i := range specParts {
		if strings.HasPrefix(specParts[i], "{") && strings.HasSuffix(specParts[i], "}") {
			continue
		}
		if ok, _ := regexp.MatchString(`^[^{}]+$`, specParts[i]); !ok || specParts[i] != generatedParts[i] {
			return false
		}
	}
	return true
}

func loadOpenAPIDoc(specFilePath string) (OpenAPIDoc, error) {
	data, err := os.ReadFile(specFilePath)
	if err != nil {
		return OpenAPIDoc{}, fmt.Errorf("failed to read OpenAPI spec file %q: %w", specFilePath, err)
	}

	var doc OpenAPIDoc
	if jsonErr := json.Unmarshal(data, &doc); jsonErr != nil {
		if yamlErr := yaml.Unmarshal(data, &doc); yamlErr != nil {
			return OpenAPIDoc{}, fmt.Errorf("failed to parse OpenAPI spec as JSON or YAML: %v", yamlErr)
		}
	}
	if len(doc.Paths) == 0 {
		return OpenAPIDoc{}, fmt.Errorf("no endpoint paths found in OpenAPI spec %q", specFilePath)
	}
	return doc, nil
}

// ConvertOpenAPIToPrompt parses an OpenAPI spec file (JSON or YAML) and formats a structured prompt for ThreatSim AI generation.
func ConvertOpenAPIToPrompt(specFilePath string) (string, error) {
	doc, err := loadOpenAPIDoc(specFilePath)
	if err != nil {
		return "", err
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
