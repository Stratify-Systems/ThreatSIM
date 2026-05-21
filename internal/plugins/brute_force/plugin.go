package bruteforce

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/Stratify-Systems/ThreatSIM/internal/core"
)

// Plugin simulates a brute force login attack.
//
// What it does:
//   - Generates rapid login_failed events against a target service
//   - Cycles through a list of usernames
//   - Simulates an attacker trying many passwords quickly
//
// What this tests in your security pipeline:
//   - Failed login monitoring
//   - Rate-limiting detection
//   - Account lockout alerting
type Plugin struct{}

// Ensure Plugin implements the core.Plugin interface at compile time
var _ core.Plugin = (*Plugin)(nil)

func (p *Plugin) ID() string   { return "brute_force" }
func (p *Plugin) Name() string { return "Brute Force Login Attack" }
func (p *Plugin) Description() string {
	return "Simulates rapid failed login attempts against a target service"
}

func (p *Plugin) DefaultConfig() core.PluginConfig {
	return core.PluginConfig{
		Target:   "10.0.0.1",
		Service:  "auth-service",
		SourceIP: "10.1.2.3",
		Duration: "30s",
		Rate:     5, // 5 login attempts per second
		Params: map[string]any{
			"usernames": []string{"admin", "root", "user", "test", "deploy"},
		},
	}
}

func (p *Plugin) Execute(ctx context.Context, config core.PluginConfig, sink core.EventSink) error {
	// Parse duration
	duration, err := time.ParseDuration(config.Duration)
	if err != nil {
		duration = 30 * time.Second
	}

	// Parse rate (events per second)
	rate := config.Rate
	if rate <= 0 {
		rate = 5
	}
	interval := time.Second / time.Duration(rate)

	// Get usernames from params
	usernames := []string{"admin", "root", "user"}
	if params := config.Params; params != nil {
		if u, ok := params["usernames"].([]string); ok && len(u) > 0 {
			usernames = u
		}
	}

	sourceIP := config.SourceIP
	if sourceIP == "" {
		sourceIP = randomIP()
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	deadline := time.After(duration)
	eventCount := 0

	client := &http.Client{Timeout: 5 * time.Second}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case <-deadline:
			// Attack duration completed
			return nil

		case <-ticker.C:
			// Pick a random username to try
			username := usernames[rand.Intn(len(usernames))]
			password := fmt.Sprintf("pass%d", rand.Intn(999999))

			// Fire Active Network Request if enabled
			if config.ActiveMode && strings.HasPrefix(config.Target, "http") {
				go func(target, u, p string) {
					payload, _ := json.Marshal(map[string]string{
						"username": u,
						"password": p,
					})
					req, err := http.NewRequest("POST", target, bytes.NewBuffer(payload))
					if err == nil {
						req.Header.Set("Content-Type", "application/json")
						// Fire and forget, we don't care if it fails because we are just simulating traffic
						client.Do(req)
					}
				}(config.Target, username, password)
			}

			event := core.Event{
				ID:        fmt.Sprintf("bf-%d-%d", time.Now().UnixNano(), eventCount),
				Type:      "login_failed",
				SourceIP:  sourceIP,
				Target:    config.Target,
				Service:   config.Service,
				User:      username,
				Timestamp: time.Now(),
				PluginID:  p.ID(),
				Metadata: map[string]any{
					"method":        "password",
					"attempt":       eventCount + 1,
					"response_code": 401,
				},
			}

			if err := sink(event); err != nil {
				return fmt.Errorf("failed to send event: %w", err)
			}

			eventCount++
		}
	}
}

// randomIP generates a random private IP address for simulation
func randomIP() string {
	return fmt.Sprintf("10.%d.%d.%d", rand.Intn(256), rand.Intn(256), rand.Intn(256))
}
