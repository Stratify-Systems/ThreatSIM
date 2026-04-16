package scenario

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/Stratify-Systems/ThreatSIM/internal/core"
	"github.com/Stratify-Systems/ThreatSIM/internal/plugins"
	"gopkg.in/yaml.v3"
)

// Engine is responsible for running multi-step attack scenarios.
type Engine struct {
	registry *plugins.Registry
}

// NewEngine creates a new Engine instance.
func NewEngine(registry *plugins.Registry) *Engine {
	return &Engine{
		registry: registry,
	}
}

// LoadFromFile parses a YAML scenario file returning the Scenario.
func LoadFromFile(path string) (*core.Scenario, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open scenario file: %w", err)
	}
	defer file.Close()

	var scenario core.Scenario
	decoder := yaml.NewDecoder(file)
	if err := decoder.Decode(&scenario); err != nil {
		return nil, fmt.Errorf("failed to parse scenario yaml: %w", err)
	}

	return &scenario, nil
}

// Run executes the scenario steps in sequence.
func (e *Engine) Run(ctx context.Context, scenario *core.Scenario, sink core.EventSink) error {
	fmt.Printf("\n🚀 Starting Scenario: %s\n", scenario.Name)
	if scenario.Description != "" {
		fmt.Printf("   Description:     %s\n", scenario.Description)
	}
	fmt.Printf("   Total Steps:     %d\n\n", len(scenario.Steps))

	for i, step := range scenario.Steps {
		// Check global cancellation
		if ctx.Err() != nil {
			return ctx.Err()
		}

		fmt.Printf("▶️  Step %d/%d: Executing [%s]\n", i+1, len(scenario.Steps), step.PluginID)

		// Find the plugin from the registry
		plugin, err := e.registry.Get(step.PluginID)
		if err != nil {
			return fmt.Errorf("step %d failed: plugin '%s' not found: %w", i+1, step.PluginID, err)
		}

		// Merge default config with step config
		config := plugin.DefaultConfig()
		if step.Config.Target != "" {
			config.Target = step.Config.Target
		}
		if step.Config.Service != "" {
			config.Service = step.Config.Service
		}
		if step.Config.Duration != "" {
			config.Duration = step.Config.Duration
		}
		if step.Config.Rate != 0 {
			config.Rate = step.Config.Rate
		}
		if len(step.Config.Params) > 0 {
			if config.Params == nil {
				config.Params = make(map[string]any)
			}
			for k, v := range step.Config.Params {
				config.Params[k] = v
			}
		}

		// Run the plugin step (blocking until step is done)
		err = plugin.Execute(ctx, config, sink)
		if err != nil && err != context.Canceled {
			return fmt.Errorf("step %d plugin execution failed: %w", i+1, err)
		}

		fmt.Printf("✅ Step %d [%s] Completed.\n", i+1, step.PluginID)

		// Handle step delay if there are more steps
		if i < len(scenario.Steps)-1 && step.Delay != "" {
			duration, err := time.ParseDuration(step.Delay)
			if err != nil {
				return fmt.Errorf("step %d failed: invalid delay format '%s': %w", i+1, step.Delay, err)
			}

			fmt.Printf("⏳ Waiting %s before next step...\n\n", duration)
			select {
			case <-time.After(duration):
			// Expected wait
			case <-ctx.Done():
				return ctx.Err()
			}
		} else {
			fmt.Println()
		}
	}

	fmt.Printf("🎉 Scenario '%s' finished successfully.\n", scenario.Name)
	return nil
}
