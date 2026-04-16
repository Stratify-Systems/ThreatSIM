package privilege_escalation

import (
	"context"
	"fmt"
	"time"

	"github.com/Stratify-Systems/ThreatSIM/internal/core"
	"github.com/google/uuid"
)

type Plugin struct{}

func (p *Plugin) ID() string   { return "privilege_escalation" }
func (p *Plugin) Name() string { return "Privilege Escalation Attack" }
func (p *Plugin) Description() string {
	return "Simulates attempts to gain higher system privileges."
}

func (p *Plugin) DefaultConfig() core.PluginConfig {
	return core.PluginConfig{
		Target:   "10.0.0.1",
		Service:  "system",
		SourceIP: "10.1.2.3",
		Duration: "5s",
		Rate:     1,
	}
}

func (p *Plugin) Execute(ctx context.Context, config core.PluginConfig, sink core.EventSink) error {
	duration, err := time.ParseDuration(config.Duration)
	if err != nil {
		duration = 5 * time.Second
	}

	rate := config.Rate
	if rate <= 0 {
		rate = 1
	}

	interval := time.Second / time.Duration(rate)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	timer := time.NewTimer(duration)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
			return nil
		case <-ticker.C:
			event := core.Event{
				ID:        fmt.Sprintf("priv-%s", uuid.New().String()[:8]),
				Type:      "priv_escalation",
				SourceIP:  config.SourceIP,
				Target:    config.Target,
				Service:   config.Service,
				User:      "admin",
				Timestamp: time.Now(),
				PluginID:  p.ID(),
				Metadata: map[string]any{
					"exploit": "sudo_misconfiguration",
				},
			}

			if err := sink(event); err != nil {
				return err
			}
		}
	}
}
