package ddos

import (
	"context"
	"fmt"
	"time"

	"github.com/Stratify-Systems/ThreatSIM/internal/core"
	"github.com/google/uuid"
)

type Plugin struct{}

func (p *Plugin) ID() string   { return "ddos" }
func (p *Plugin) Name() string { return "DDoS HTTP Burst Attack" }
func (p *Plugin) Description() string {
	return "Simulates a high-volume HTTP request burst to overwhelm a service"
}

func (p *Plugin) DefaultConfig() core.PluginConfig {
	return core.PluginConfig{
		Target:   "10.0.0.1",
		Service:  "web-server",
		SourceIP: "10.1.2.3",
		Duration: "10s",
		Rate:     200,
	}
}

func (p *Plugin) Execute(ctx context.Context, config core.PluginConfig, sink core.EventSink) error {
	duration, err := time.ParseDuration(config.Duration)
	if err != nil {
		duration = 10 * time.Second
	}

	rate := config.Rate
	if rate <= 0 {
		rate = 200
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
				ID:        fmt.Sprintf("ddos-%s", uuid.New().String()[:8]),
				Type:      "http_flood",
				SourceIP:  config.SourceIP,
				Target:    config.Target,
				Service:   config.Service,
				Timestamp: time.Now(),
				PluginID:  p.ID(),
				Metadata: map[string]any{
					"method": "GET",
					"path":   "/",
					"size":   512,
				},
			}

			if err := sink(event); err != nil {
				return err
			}
		}
	}
}
