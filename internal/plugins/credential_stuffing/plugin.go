package credential_stuffing

import (
	"context"
	"fmt"
	"time"

	"github.com/Stratify-Systems/ThreatSIM/internal/core"
	"github.com/google/uuid"
)

type Plugin struct{}

func (p *Plugin) ID() string   { return "credential_stuffing" }
func (p *Plugin) Name() string { return "Credential Stuffing Attack" }
func (p *Plugin) Description() string {
	return "Simulates automated logins with stolen credential lists"
}

func (p *Plugin) DefaultConfig() core.PluginConfig {
	return core.PluginConfig{
		Target:   "10.0.0.1",
		Service:  "auth-service",
		SourceIP: "10.1.2.3",
		Duration: "15s",
		Rate:     5,
	}
}

func (p *Plugin) Execute(ctx context.Context, config core.PluginConfig, sink core.EventSink) error {
	duration, err := time.ParseDuration(config.Duration)
	if err != nil {
		duration = 15 * time.Second
	}

	rate := config.Rate
	if rate <= 0 {
		rate = 5
	}

	interval := time.Second / time.Duration(rate)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	timer := time.NewTimer(duration)
	defer timer.Stop()

	users := []string{"admin", "root", "dev", "user1", "testuser"}
	userIdx := 0

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
			return nil
		case <-ticker.C:
			user := users[userIdx%len(users)]
			userIdx++

			event := core.Event{
				ID:        fmt.Sprintf("cred-%s", uuid.New().String()[:8]),
				Type:      "login_attempt",
				SourceIP:  config.SourceIP,
				Target:    config.Target,
				Service:   config.Service,
				User:      user,
				Timestamp: time.Now(),
				PluginID:  p.ID(),
				Metadata: map[string]any{
					"method": "credential_stuffing",
				},
			}

			if err := sink(event); err != nil {
				return err
			}
		}
	}
}
