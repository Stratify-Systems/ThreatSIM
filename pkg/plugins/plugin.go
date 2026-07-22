package plugins

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/suryatk2007/threatsim/pkg/types"
	"gopkg.in/yaml.v3"
)

// StateMap is a thread-safe map container for cross-plugin state sharing.
type StateMap struct {
	mu sync.RWMutex
	m  map[string]string
}

// NewStateMap constructs a new thread-safe StateMap instance.
func NewStateMap() *StateMap {
	return &StateMap{
		m: make(map[string]string),
	}
}

// Set thread-safely sets a key-value pair in the state.
func (s *StateMap) Set(key, val string) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.m == nil {
		s.m = make(map[string]string)
	}
	s.m[key] = val
}

// Get thread-safely retrieves a value from the state.
func (s *StateMap) Get(key string) (string, bool) {
	if s == nil {
		return "", false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.m == nil {
		return "", false
	}
	val, ok := s.m[key]
	return val, ok
}

// GetAll returns a thread-safe snapshot copy of the state map.
func (s *StateMap) GetAll() map[string]string {
	if s == nil {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	copyMap := make(map[string]string, len(s.m))
	for k, v := range s.m {
		copyMap[k] = v
	}
	return copyMap
}

// Context provides plugins with execution state, HTTP client, and thread-safe state sharing.
type Context struct {
	TargetURL string
	Client    *http.Client
	State     *StateMap
}

// SetState safely sets a key-value pair in the shared execution context state.
func (c *Context) SetState(key, val string) {
	if c.State == nil {
		c.State = NewStateMap()
	}
	c.State.Set(key, val)
}

// GetState safely retrieves a value from the shared execution context state.
func (c *Context) GetState(key string) (string, bool) {
	if c.State == nil {
		return "", false
	}
	return c.State.Get(key)
}

// Plugin defines the interface that all security attack plugins must implement.
type Plugin interface {
	Name() string
	Description() string
	Execute(simName string, ctx Context, config map[string]interface{}) []types.SimulationResult
}

var registry = make(map[string]Plugin)

// Register adds a plugin to the global registry
func Register(p Plugin) {
	registry[p.Name()] = p
}

// Get retrieves a plugin by name
func Get(name string) (Plugin, error) {
	if p, exists := registry[name]; exists {
		return p, nil
	}
	return nil, fmt.Errorf("plugin %q not found", name)
}

// List returns all registered plugins
func List() map[string]Plugin {
	return registry
}

// SchemaTemplate represents the structure of plugin schema YAML files
type SchemaTemplate struct {
	Name   string                 `yaml:"name"`
	Plugin string                 `yaml:"plugin"`
	Config map[string]interface{} `yaml:"config"`
}

// ValidateConfig validates plugin execution config parameters against YAML schemas in schemas/plugins/
func ValidateConfig(pluginName string, config map[string]interface{}) error {
	if pluginName == "" {
		return fmt.Errorf("plugin name cannot be empty")
	}

	// 1. Try to load schema file from schemas/plugins/<pluginName>.yaml
	schemaPaths := []string{
		filepath.Join("schemas", "plugins", fmt.Sprintf("%s.yaml", pluginName)),
		filepath.Join("..", "schemas", "plugins", fmt.Sprintf("%s.yaml", pluginName)),
		filepath.Join("..", "..", "schemas", "plugins", fmt.Sprintf("%s.yaml", pluginName)),
	}

	var schemaData []byte
	var readErr error

	for _, p := range schemaPaths {
		data, err := os.ReadFile(p)
		if err == nil {
			schemaData = data
			readErr = nil
			break
		}
		readErr = err
	}

	// Standard required field checks per plugin if schema file not found on disk
	switch pluginName {
	case "idor":
		required := []string{"auth_path", "user_a_payload", "user_b_payload", "target_path"}
		for _, req := range required {
			val, ok := config[req].(string)
			if !ok || strings.TrimSpace(val) == "" {
				return fmt.Errorf("idor plugin schema validation failed: missing or empty required field %q", req)
			}
		}
	case "jwt_forge":
		required := []string{"auth_path", "auth_payload", "token_json_path", "target_path"}
		for _, req := range required {
			val, ok := config[req].(string)
			if !ok || strings.TrimSpace(val) == "" {
				return fmt.Errorf("jwt_forge plugin schema validation failed: missing or empty required field %q", req)
			}
		}
	case "bruteforce":
		if _, ok := config["path"].(string); !ok || strings.TrimSpace(config["path"].(string)) == "" {
			return fmt.Errorf("bruteforce plugin schema validation failed: missing or empty required field %q", "path")
		}
		if _, ok := config["username"].(string); !ok || strings.TrimSpace(config["username"].(string)) == "" {
			return fmt.Errorf("bruteforce plugin schema validation failed: missing or empty required field %q", "username")
		}
		if _, ok := config["num_requests"]; !ok {
			return fmt.Errorf("bruteforce plugin schema validation failed: missing required field %q", "num_requests")
		}
	case "cors_audit":
		if _, ok := config["path"].(string); !ok || strings.TrimSpace(config["path"].(string)) == "" {
			return fmt.Errorf("cors_audit plugin schema validation failed: missing or empty required field %q", "path")
		}
	case "rate_limit":
		if _, ok := config["path"].(string); !ok || strings.TrimSpace(config["path"].(string)) == "" {
			return fmt.Errorf("rate_limit plugin schema validation failed: missing or empty required field %q", "path")
		}
		if _, ok := config["num_requests"]; !ok {
			return fmt.Errorf("rate_limit plugin schema validation failed: missing required field %q", "num_requests")
		}
	}

	if readErr != nil || len(schemaData) == 0 {
		return nil // Field validation succeeded via built-in rules
	}

	var schema SchemaTemplate
	if err := yaml.Unmarshal(schemaData, &schema); err != nil {
		return nil // Ignore YAML parsing errors in schema files
	}

	// Validate config keys against schema config keys
	for schemaKey, commentVal := range schema.Config {
		// If schema marks field as (Required), check presence in config
		commentStr := fmt.Sprintf("%v", commentVal)
		if strings.Contains(strings.ToLower(commentStr), "required") {
			if _, exists := config[schemaKey]; !exists {
				return fmt.Errorf("plugin %q schema validation failed: missing required configuration field %q", pluginName, schemaKey)
			}
		}
	}

	return nil
}

// ParseString safely extracts a string value from plugin configuration.
func ParseString(config map[string]interface{}, key string) string {
	if val, ok := config[key].(string); ok {
		return strings.TrimSpace(val)
	}
	return ""
}

// ParseInt safely extracts an integer value (supporting int and float64) from plugin configuration.
func ParseInt(config map[string]interface{}, key string, defaultValue int) int {
	if raw, ok := config[key]; ok {
		switch v := raw.(type) {
		case int:
			return v
		case float64:
			return int(v)
		}
	}
	return defaultValue
}

// ParseBool safely extracts a boolean value from plugin configuration.
func ParseBool(config map[string]interface{}, key string, defaultValue bool) bool {
	if val, ok := config[key].(bool); ok {
		return val
	}
	return defaultValue
}

