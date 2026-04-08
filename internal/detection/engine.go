package detection

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/Stratify-Systems/ThreatSIM/internal/core"
	"github.com/google/uuid"
	"gopkg.in/yaml.v3"
)

// ruleFile is a helper struct to parse YAML
type ruleFile struct {
	Rules []core.Rule `yaml:"rules"`
}

// Engine evaluates events streaming from the simulator against YAML-based detection rules.
// It maintains a sliding window of events to calculate rate thresholds.
type Engine struct {
	stream core.EventStream
	rules  []core.Rule

	// AlertSink is called when a rule's threshold is met.
	AlertSink func(core.Alert)

	// windows tracks event timestamps to calculate rates.
	// Structure: [RuleName][GroupByValue][]time.Time
	windows map[string]map[string][]time.Time
	mu      sync.RWMutex
}

// NewEngine creates a new detection engine instance.
func NewEngine(stream core.EventStream) *Engine {
	return &Engine{
		stream:  stream,
		windows: make(map[string]map[string][]time.Time),
	}
}

// LoadRulesFromDir recursively parses all .yaml files in the specified directory.
func (e *Engine) LoadRulesFromDir(dir string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // Permissive if no rules folder exists yet
		}
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		
		ext := filepath.Ext(entry.Name())
		if ext == ".yaml" || ext == ".yml" {
			path := filepath.Join(dir, entry.Name())
			data, err := os.ReadFile(path)
			if err != nil {
				return fmt.Errorf("failed to read rule file %s: %w", path, err)
			}

			var rf ruleFile
			if err := yaml.Unmarshal(data, &rf); err != nil {
				return fmt.Errorf("failed to parse rule YAML %s: %w", path, err)
			}

			// Add the rules and initialize window maps
			for _, r := range rf.Rules {
				e.rules = append(e.rules, r)
				
				if _, ok := e.windows[r.Name]; !ok {
					e.windows[r.Name] = make(map[string][]time.Time)
				}
			}
		}
	}

	return nil
}

// Start begins subscribing to the event stream, blocking as it listens.
func (e *Engine) Start(ctx context.Context) error {
	return e.stream.Subscribe(ctx, core.TopicAttackEvents, e.handleEvent)
}

// Rules returns the number of loaded rules
func (e *Engine) RulesCount() int {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return len(e.rules)
}

// handleEvent represents the sliding window evaluation logic for a single event.
func (e *Engine) handleEvent(ctx context.Context, ev core.Event) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	now := time.Now()

	for _, rule := range e.rules {
		if ev.Type != rule.Condition.EventType {
			continue
		}

		windowDur, err := time.ParseDuration(rule.Condition.Window)
		if err != nil {
			// Invalid window string in YAML, skip
			continue 
		}

		// Determine the group key (e.g. by SourceIP, Target, or global)
		groupKey := calculateGroupKey(rule.Condition.GroupBy, ev)

		// 1. Add new event to history
		history := e.windows[rule.Name][groupKey]
		history = append(history, ev.Timestamp)

		// 2. Prune old events outside the sliding window
		cutoff := now.Add(-windowDur)
		var newHistory []time.Time
		for _, ts := range history {
			// Include events strictly newer than cutoff
			if ts.After(cutoff) || ts.Equal(cutoff) {
				newHistory = append(newHistory, ts)
			}
		}
		
		e.windows[rule.Name][groupKey] = newHistory

		// 3. Evaluate Threshold Condition
		if len(newHistory) >= rule.Condition.Threshold {
			// Trigger the Alert
			alert := core.Alert{
				ID:          uuid.New().String(),
				RuleName:    rule.Name,
				Type:        rule.Condition.EventType,
				Description: rule.Description,
				Severity:    rule.Severity,
				SourceIP:    ev.SourceIP,
				Service:     ev.Service,
				EventCount:  len(newHistory),
				Timestamp:   now,
				ScenarioID:  ev.ScenarioID,
			}

			// Clear the window after firing an alert so we don't spam 
			// the sink on every subsequent event, requiring the threshold to build up again
			e.windows[rule.Name][groupKey] = nil

			if e.AlertSink != nil {
				e.AlertSink(alert)
			}
		}
	}

	return nil
}

func calculateGroupKey(groupBy string, ev core.Event) string {
	switch groupBy {
	case "source_ip":
		return ev.SourceIP
	case "target":
		return ev.Target
	case "service":
		return ev.Service
	case "user":
		return ev.User
	case "":
		return "global"
	default:
		// Attempt to extract from embedded metadata if available
		if val, exists := ev.Metadata[groupBy]; exists {
			return fmt.Sprintf("%v", val)
		}
		return "global"
	}
}
