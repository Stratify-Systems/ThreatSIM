package alerting

import (
	"sync"
	"testing"
	"time"

	"github.com/Stratify-Systems/ThreatSIM/internal/core"
)

// mockNotifier records calls to Send for testing.
type mockNotifier struct {
	mu    sync.Mutex
	calls []core.RiskScore
}

func (m *mockNotifier) Send(score core.RiskScore) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = append(m.calls, score)
	return nil
}

func (m *mockNotifier) callCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.calls)
}

func TestDispatcher_SendsOnFirstDetection(t *testing.T) {
	d := NewDispatcher()
	notifier := &mockNotifier{}
	d.Register(notifier)

	d.Dispatch(core.RiskScore{
		SourceIP:    "10.0.0.1",
		Score:       60,
		ThreatLevel: core.ThreatHigh,
	})

	// Dispatch sends via goroutines, give them time to complete
	time.Sleep(50 * time.Millisecond)

	if notifier.callCount() != 1 {
		t.Errorf("expected 1 notification, got %d", notifier.callCount())
	}
}

func TestDispatcher_SuppressesDuplicateLevel(t *testing.T) {
	d := NewDispatcher()
	notifier := &mockNotifier{}
	d.Register(notifier)

	// First dispatch — should send
	d.Dispatch(core.RiskScore{
		SourceIP:    "10.0.0.1",
		Score:       60,
		ThreatLevel: core.ThreatHigh,
	})
	time.Sleep(50 * time.Millisecond)

	// Same threat level — should suppress
	d.Dispatch(core.RiskScore{
		SourceIP:    "10.0.0.1",
		Score:       65,
		ThreatLevel: core.ThreatHigh,
	})
	time.Sleep(50 * time.Millisecond)

	if notifier.callCount() != 1 {
		t.Errorf("expected 1 notification (duplicate suppressed), got %d", notifier.callCount())
	}
}

func TestDispatcher_SendsOnEscalation(t *testing.T) {
	d := NewDispatcher()
	notifier := &mockNotifier{}
	d.Register(notifier)

	// HIGH
	d.Dispatch(core.RiskScore{
		SourceIP:    "10.0.0.1",
		Score:       60,
		ThreatLevel: core.ThreatHigh,
	})
	time.Sleep(50 * time.Millisecond)

	// Escalate to CRITICAL
	d.Dispatch(core.RiskScore{
		SourceIP:    "10.0.0.1",
		Score:       90,
		ThreatLevel: core.ThreatCritical,
	})
	time.Sleep(50 * time.Millisecond)

	if notifier.callCount() != 2 {
		t.Errorf("expected 2 notifications (escalation), got %d", notifier.callCount())
	}
}

func TestDispatcher_DifferentIPsAreIndependent(t *testing.T) {
	d := NewDispatcher()
	notifier := &mockNotifier{}
	d.Register(notifier)

	d.Dispatch(core.RiskScore{
		SourceIP:    "10.0.0.1",
		Score:       60,
		ThreatLevel: core.ThreatHigh,
	})
	d.Dispatch(core.RiskScore{
		SourceIP:    "10.0.0.2",
		Score:       60,
		ThreatLevel: core.ThreatHigh,
	})

	time.Sleep(100 * time.Millisecond)

	if notifier.callCount() != 2 {
		t.Errorf("expected 2 notifications (different IPs), got %d", notifier.callCount())
	}
}

func TestLevelWeight(t *testing.T) {
	tests := []struct {
		level    core.ThreatLevel
		expected int
	}{
		{core.ThreatLow, 1},
		{core.ThreatMedium, 2},
		{core.ThreatHigh, 3},
		{core.ThreatCritical, 4},
		{"UNKNOWN", 0},
	}

	for _, tt := range tests {
		got := levelWeight(tt.level)
		if got != tt.expected {
			t.Errorf("levelWeight(%s) = %d, want %d", tt.level, got, tt.expected)
		}
	}
}
