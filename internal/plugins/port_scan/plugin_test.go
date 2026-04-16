package portscan

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Stratify-Systems/ThreatSIM/internal/core"
)

func TestPlugin_Interface(t *testing.T) {
	var p core.Plugin = &Plugin{}
	if p.ID() != "port_scan" {
		t.Errorf("expected ID 'port_scan', got '%s'", p.ID())
	}
	if p.Name() == "" {
		t.Error("expected non-empty Name()")
	}
	if p.Description() == "" {
		t.Error("expected non-empty Description()")
	}
}

func TestPlugin_Execute_GeneratesPortProbes(t *testing.T) {
	p := &Plugin{}
	config := core.PluginConfig{
		Target:   "10.0.0.1",
		Service:  "network",
		SourceIP: "10.1.2.3",
		Duration: "2s",
		Rate:     20,
		Params: map[string]any{
			"port_start": 1,
			"port_end":   50,
		},
	}

	var count int32
	var hasOpenPort bool
	sink := func(event core.Event) error {
		atomic.AddInt32(&count, 1)

		if event.Type != "port_probe" {
			t.Errorf("expected event type 'port_probe', got '%s'", event.Type)
		}
		if event.PluginID != "port_scan" {
			t.Errorf("expected plugin ID 'port_scan', got '%s'", event.PluginID)
		}

		// Check that port metadata exists
		if _, ok := event.Metadata["port"]; !ok {
			t.Error("expected 'port' in metadata")
		}
		if status, ok := event.Metadata["port_status"]; ok && status == "open" {
			hasOpenPort = true
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

	// Port 22 (SSH) is in range 1-50 and is in the commonOpenPorts map
	if !hasOpenPort {
		t.Log("warning: no open ports detected in scan range (port 22 should be open)")
	}
}

func TestPlugin_Execute_StopsAtPortEnd(t *testing.T) {
	p := &Plugin{}
	config := core.PluginConfig{
		Target:   "10.0.0.1",
		SourceIP: "10.1.2.3",
		Duration: "30s", // Long duration — should stop when ports exhausted
		Rate:     100,   // Fast rate
		Params: map[string]any{
			"port_start": 1,
			"port_end":   5,
		},
	}

	var count int32
	err := p.Execute(context.Background(), config, func(event core.Event) error {
		atomic.AddInt32(&count, 1)
		return nil
	})

	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	total := atomic.LoadInt32(&count)
	if total != 5 {
		t.Errorf("expected 5 events (ports 1-5), got %d", total)
	}
}

func TestPlugin_Execute_RespectsContextCancel(t *testing.T) {
	p := &Plugin{}
	config := core.PluginConfig{
		Target:   "10.0.0.1",
		Duration: "30s",
		Rate:     5,
	}

	ctx, cancel := context.WithCancel(context.Background())
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
