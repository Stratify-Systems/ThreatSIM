package plugins

import (
	"fmt"
	"net/http"

	"github.com/suryatk2007/threatsim/pkg/types"
)

// Context provides plugins with necessary execution information
type Context struct {
	TargetURL string
	Client    *http.Client
	State     map[string]string
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
