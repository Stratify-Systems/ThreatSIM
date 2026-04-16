package detection

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/Stratify-Systems/ThreatSIM/internal/core"
	"github.com/Stratify-Systems/ThreatSIM/internal/streaming/memory"
)

func TestEngine_HandleEvent_TriggersAlert(t *testing.T) {
	stream := memory.NewStream()
	engine := NewEngine(stream)

	engine.rules = []core.Rule{
		{
			Name:        "test_brute_force",
			Description: "Test brute force detection",
			Condition: core.RuleCondition{
				EventType: "login_failed",
				GroupBy:   "source_ip",
				Threshold: 3,
				Window:    "10s",
			},
			Severity:  core.SeverityHigh,
			RiskScore: 60,
		},
	}
	engine.windows["test_brute_force"] = make(map[string][]time.Time)

	var alertFired bool
	var firedAlert core.Alert
	var mu sync.Mutex

	engine.AlertSink = func(alert core.Alert) {
		mu.Lock()
		defer mu.Unlock()
		alertFired = true
		firedAlert = alert
	}

	// Send 3 events (threshold) from the same IP
	for i := 0; i < 3; i++ {
		event := core.Event{
			Type:      "login_failed",
			SourceIP:  "10.0.0.1",
			Target:    "10.0.0.2",
			Timestamp: time.Now(),
		}
		err := engine.handleEvent(context.Background(), event)
		if err != nil {
			t.Fatalf("handleEvent returned error: %v", err)
		}
	}

	mu.Lock()
	defer mu.Unlock()
	if !alertFired {
		t.Fatal("expected alert to fire after 3 events, but it did not")
	}
	if firedAlert.RuleName != "test_brute_force" {
		t.Errorf("expected rule name 'test_brute_force', got '%s'", firedAlert.RuleName)
	}
	if firedAlert.Severity != core.SeverityHigh {
		t.Errorf("expected severity HIGH, got '%s'", firedAlert.Severity)
	}
	if firedAlert.SourceIP != "10.0.0.1" {
		t.Errorf("expected source IP '10.0.0.1', got '%s'", firedAlert.SourceIP)
	}
	if firedAlert.EventCount != 3 {
		t.Errorf("expected event count 3, got %d", firedAlert.EventCount)
	}
}

func TestEngine_HandleEvent_DoesNotTriggerBelowThreshold(t *testing.T) {
	stream := memory.NewStream()
	engine := NewEngine(stream)

	engine.rules = []core.Rule{
		{
			Name: "test_rule",
			Condition: core.RuleCondition{
				EventType: "login_failed",
				GroupBy:   "source_ip",
				Threshold: 5,
				Window:    "10s",
			},
			Severity: core.SeverityMedium,
		},
	}
	engine.windows["test_rule"] = make(map[string][]time.Time)

	alertFired := false
	engine.AlertSink = func(alert core.Alert) {
		alertFired = true
	}

	// Send only 4 events (below threshold of 5)
	for i := 0; i < 4; i++ {
		engine.handleEvent(context.Background(), core.Event{
			Type:      "login_failed",
			SourceIP:  "10.0.0.1",
			Timestamp: time.Now(),
		})
	}

	if alertFired {
		t.Fatal("alert should NOT fire below threshold")
	}
}

func TestEngine_HandleEvent_IgnoresWrongEventType(t *testing.T) {
	stream := memory.NewStream()
	engine := NewEngine(stream)

	engine.rules = []core.Rule{
		{
			Name: "port_scan_rule",
			Condition: core.RuleCondition{
				EventType: "port_probe",
				GroupBy:   "source_ip",
				Threshold: 1,
				Window:    "10s",
			},
			Severity: core.SeverityMedium,
		},
	}
	engine.windows["port_scan_rule"] = make(map[string][]time.Time)

	alertFired := false
	engine.AlertSink = func(alert core.Alert) {
		alertFired = true
	}

	// Send login_failed events — rule watches port_probe
	engine.handleEvent(context.Background(), core.Event{
		Type:      "login_failed",
		SourceIP:  "10.0.0.1",
		Timestamp: time.Now(),
	})

	if alertFired {
		t.Fatal("alert should NOT fire for wrong event type")
	}
}

func TestEngine_HandleEvent_GroupBySource(t *testing.T) {
	stream := memory.NewStream()
	engine := NewEngine(stream)

	engine.rules = []core.Rule{
		{
			Name: "grouped_rule",
			Condition: core.RuleCondition{
				EventType: "login_failed",
				GroupBy:   "source_ip",
				Threshold: 3,
				Window:    "10s",
			},
			Severity: core.SeverityHigh,
		},
	}
	engine.windows["grouped_rule"] = make(map[string][]time.Time)

	alertFired := false
	engine.AlertSink = func(alert core.Alert) {
		alertFired = true
	}

	// Send 2 events from IP-A and 1 from IP-B — neither crosses threshold of 3
	for i := 0; i < 2; i++ {
		engine.handleEvent(context.Background(), core.Event{
			Type:      "login_failed",
			SourceIP:  "10.0.0.1",
			Timestamp: time.Now(),
		})
	}
	engine.handleEvent(context.Background(), core.Event{
		Type:      "login_failed",
		SourceIP:  "10.0.0.2",
		Timestamp: time.Now(),
	})

	if alertFired {
		t.Fatal("alert should NOT fire — events are split across IPs")
	}
}

func TestEngine_HandleEvent_WindowExpiration(t *testing.T) {
	stream := memory.NewStream()
	engine := NewEngine(stream)

	engine.rules = []core.Rule{
		{
			Name: "window_rule",
			Condition: core.RuleCondition{
				EventType: "login_failed",
				GroupBy:   "source_ip",
				Threshold: 3,
				Window:    "1s",
			},
			Severity: core.SeverityHigh,
		},
	}
	engine.windows["window_rule"] = make(map[string][]time.Time)

	alertFired := false
	engine.AlertSink = func(alert core.Alert) {
		alertFired = true
	}

	// Send 2 events with old timestamps (outside the 1s window)
	oldTime := time.Now().Add(-5 * time.Second)
	for i := 0; i < 2; i++ {
		engine.handleEvent(context.Background(), core.Event{
			Type:      "login_failed",
			SourceIP:  "10.0.0.1",
			Timestamp: oldTime,
		})
	}

	// Send 1 more event with current time — total in window should be 1, not 3
	engine.handleEvent(context.Background(), core.Event{
		Type:      "login_failed",
		SourceIP:  "10.0.0.1",
		Timestamp: time.Now(),
	})

	if alertFired {
		t.Fatal("alert should NOT fire — old events should have expired from window")
	}
}

func TestEngine_HandleEvent_ClearsWindowAfterAlert(t *testing.T) {
	stream := memory.NewStream()
	engine := NewEngine(stream)

	engine.rules = []core.Rule{
		{
			Name: "clear_rule",
			Condition: core.RuleCondition{
				EventType: "login_failed",
				GroupBy:   "source_ip",
				Threshold: 2,
				Window:    "10s",
			},
			Severity: core.SeverityHigh,
		},
	}
	engine.windows["clear_rule"] = make(map[string][]time.Time)

	alertCount := 0
	engine.AlertSink = func(alert core.Alert) {
		alertCount++
	}

	// Trigger an alert (2 events = threshold)
	for i := 0; i < 2; i++ {
		engine.handleEvent(context.Background(), core.Event{
			Type:      "login_failed",
			SourceIP:  "10.0.0.1",
			Timestamp: time.Now(),
		})
	}

	if alertCount != 1 {
		t.Fatalf("expected exactly 1 alert, got %d", alertCount)
	}

	// Send 1 more event — should NOT trigger another alert since window was cleared
	engine.handleEvent(context.Background(), core.Event{
		Type:      "login_failed",
		SourceIP:  "10.0.0.1",
		Timestamp: time.Now(),
	})

	if alertCount != 1 {
		t.Fatalf("expected still 1 alert after window clear, got %d", alertCount)
	}
}

func TestEngine_RulesCount(t *testing.T) {
	stream := memory.NewStream()
	engine := NewEngine(stream)

	if engine.RulesCount() != 0 {
		t.Fatalf("expected 0 rules initially, got %d", engine.RulesCount())
	}

	engine.rules = []core.Rule{{Name: "r1"}, {Name: "r2"}}

	if engine.RulesCount() != 2 {
		t.Fatalf("expected 2 rules, got %d", engine.RulesCount())
	}
}

func TestCalculateGroupKey(t *testing.T) {
	event := core.Event{
		SourceIP: "10.0.0.1",
		Target:   "10.0.0.2",
		Service:  "auth",
		User:     "admin",
		Metadata: map[string]any{"port": 22},
	}

	tests := []struct {
		groupBy  string
		expected string
	}{
		{"source_ip", "10.0.0.1"},
		{"target", "10.0.0.2"},
		{"service", "auth"},
		{"user", "admin"},
		{"", "global"},
		{"port", "22"},
		{"nonexistent", "global"},
	}

	for _, tt := range tests {
		result := calculateGroupKey(tt.groupBy, event)
		if result != tt.expected {
			t.Errorf("calculateGroupKey(%q) = %q, want %q", tt.groupBy, result, tt.expected)
		}
	}
}
