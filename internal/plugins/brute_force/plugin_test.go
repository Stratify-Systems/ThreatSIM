package bruteforce

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Stratify-Systems/ThreatSIM/internal/core"
)

func TestPlugin_Interface(t *testing.T) {
	var p core.Plugin = &Plugin{}
	if p.ID() != "brute_force" {
		t.Errorf("expected ID 'brute_force', got '%s'", p.ID())
	}
	if p.Name() == "" {
		t.Error("expected non-empty Name()")
	}
	if p.Description() == "" {
		t.Error("expected non-empty Description()")
	}
}

func TestPlugin_DefaultConfig(t *testing.T) {
	p := &Plugin{}
	config := p.DefaultConfig()

	if config.Target == "" {
		t.Error("expected non-empty default target")
	}
	if config.Rate <= 0 {
		t.Errorf("expected positive rate, got %d", config.Rate)
	}
	if config.Duration == "" {
		t.Error("expected non-empty default duration")
	}
}

func TestPlugin_Execute_GeneratesEvents(t *testing.T) {
	p := &Plugin{}
	config := core.PluginConfig{
		Target:   "10.0.0.1",
		Service:  "auth-service",
		SourceIP: "10.1.2.3",
		Duration: "1s",
		Rate:     10,
		Params: map[string]any{
			"usernames": []string{"admin", "root"},
		},
	}

	var count int32
	sink := func(event core.Event) error {
		atomic.AddInt32(&count, 1)

		// Validate event fields
		if event.Type != "login_failed" {
			t.Errorf("expected event type 'login_failed', got '%s'", event.Type)
		}
		if event.SourceIP != "10.1.2.3" {
			t.Errorf("expected source IP '10.1.2.3', got '%s'", event.SourceIP)
		}
		if event.PluginID != "brute_force" {
			t.Errorf("expected plugin ID 'brute_force', got '%s'", event.PluginID)
		}
		if event.Target != "10.0.0.1" {
			t.Errorf("expected target '10.0.0.1', got '%s'", event.Target)
		}
		return nil
	}

	err := p.Execute(context.Background(), config, sink)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	total := atomic.LoadInt32(&count)
	if total == 0 {
		t.Fatal("expected at least 1 event, got 0")
	}
	// With 10 events/sec for 1 second, we expect ~10 events (allow some variance)
	if total < 5 || total > 15 {
		t.Errorf("expected roughly 10 events (1s * 10/sec), got %d", total)
	}
}

func TestPlugin_Execute_RespectsContextCancel(t *testing.T) {
	p := &Plugin{}
	config := core.PluginConfig{
		Target:   "10.0.0.1",
		Duration: "30s", // Long duration
		Rate:     5,
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Cancel after 200ms
	go func() {
		time.Sleep(200 * time.Millisecond)
		cancel()
	}()

	err := p.Execute(ctx, config, func(event core.Event) error {
		return nil
	})

	if err != context.Canceled {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

func TestPlugin_Execute_RandomIPWhenEmpty(t *testing.T) {
	p := &Plugin{}
	config := core.PluginConfig{
		Target:   "10.0.0.1",
		SourceIP: "", // Empty — should generate random
		Duration: "500ms",
		Rate:     2,
	}

	var sourceIP string
	sink := func(event core.Event) error {
		if sourceIP == "" {
			sourceIP = event.SourceIP
		}
		return nil
	}

	err := p.Execute(context.Background(), config, sink)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if sourceIP == "" {
		t.Fatal("expected a generated source IP, got empty string")
	}
}
