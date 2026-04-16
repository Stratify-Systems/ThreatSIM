package risk

import (
	"sync"
	"testing"

	"github.com/Stratify-Systems/ThreatSIM/internal/core"
)

func TestEngine_ProcessAlert_ScoreAccumulation(t *testing.T) {
	engine := NewEngine()

	// Process a HIGH severity alert (inc = 60)
	engine.ProcessAlert(core.Alert{
		RuleName: "brute_force_attack",
		Severity: core.SeverityHigh,
		SourceIP: "10.0.0.1",
	})

	score := engine.GetScore("10.0.0.1")
	if score.Score != 60 {
		t.Errorf("expected score 60, got %d", score.Score)
	}
	if score.ThreatLevel != core.ThreatMedium {
		t.Errorf("expected MEDIUM threat level for score 60, got %s", score.ThreatLevel)
	}

	// Process another MEDIUM severity alert (inc = 30)
	engine.ProcessAlert(core.Alert{
		RuleName: "port_scan_detected",
		Severity: core.SeverityMedium,
		SourceIP: "10.0.0.1",
	})

	score = engine.GetScore("10.0.0.1")
	if score.Score != 90 {
		t.Errorf("expected accumulated score 90, got %d", score.Score)
	}
	if score.ThreatLevel != core.ThreatCritical {
		t.Errorf("expected CRITICAL threat level for score 90, got %s", score.ThreatLevel)
	}
}

func TestEngine_ProcessAlert_ScoreCap(t *testing.T) {
	engine := NewEngine()

	// Process two CRITICAL alerts (2 * 90 = 180, should cap at 100)
	engine.ProcessAlert(core.Alert{
		RuleName: "ddos",
		Severity: core.SeverityCritical,
		SourceIP: "10.0.0.1",
	})
	engine.ProcessAlert(core.Alert{
		RuleName: "ddos_again",
		Severity: core.SeverityCritical,
		SourceIP: "10.0.0.1",
	})

	score := engine.GetScore("10.0.0.1")
	if score.Score != 100 {
		t.Errorf("expected capped score 100, got %d", score.Score)
	}
}

func TestEngine_ProcessAlert_ThreatLevels(t *testing.T) {
	tests := []struct {
		name          string
		severity      core.Severity
		expectedScore int
		expectedLevel core.ThreatLevel
	}{
		{"low", core.SeverityLow, 10, core.ThreatLow},
		{"medium", core.SeverityMedium, 30, core.ThreatLow},
		{"high", core.SeverityHigh, 60, core.ThreatMedium},
		{"critical", core.SeverityCritical, 90, core.ThreatCritical},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := NewEngine()
			engine.ProcessAlert(core.Alert{
				RuleName: "test_rule",
				Severity: tt.severity,
				SourceIP: "10.0.0.1",
			})

			score := engine.GetScore("10.0.0.1")
			if score.Score != tt.expectedScore {
				t.Errorf("expected score %d, got %d", tt.expectedScore, score.Score)
			}
			if score.ThreatLevel != tt.expectedLevel {
				t.Errorf("expected threat level %s, got %s", tt.expectedLevel, score.ThreatLevel)
			}
		})
	}
}

func TestEngine_ProcessAlert_IsolatedPerIP(t *testing.T) {
	engine := NewEngine()

	engine.ProcessAlert(core.Alert{
		RuleName: "rule_a",
		Severity: core.SeverityHigh,
		SourceIP: "10.0.0.1",
	})
	engine.ProcessAlert(core.Alert{
		RuleName: "rule_b",
		Severity: core.SeverityLow,
		SourceIP: "10.0.0.2",
	})

	score1 := engine.GetScore("10.0.0.1")
	score2 := engine.GetScore("10.0.0.2")

	if score1.Score != 60 {
		t.Errorf("IP 10.0.0.1: expected 60, got %d", score1.Score)
	}
	if score2.Score != 10 {
		t.Errorf("IP 10.0.0.2: expected 10, got %d", score2.Score)
	}
}

func TestEngine_ProcessAlert_Factors(t *testing.T) {
	engine := NewEngine()

	engine.ProcessAlert(core.Alert{
		RuleName: "brute_force",
		Severity: core.SeverityHigh,
		SourceIP: "10.0.0.1",
	})
	engine.ProcessAlert(core.Alert{
		RuleName: "port_scan",
		Severity: core.SeverityMedium,
		SourceIP: "10.0.0.1",
	})

	score := engine.GetScore("10.0.0.1")
	if len(score.Factors) != 2 {
		t.Errorf("expected 2 factors, got %d", len(score.Factors))
	}

	factorSet := make(map[string]bool)
	for _, f := range score.Factors {
		factorSet[f] = true
	}
	if !factorSet["brute_force"] || !factorSet["port_scan"] {
		t.Errorf("expected factors brute_force and port_scan, got %v", score.Factors)
	}
}

func TestEngine_ProcessAlert_RiskUpdateSinkCalled(t *testing.T) {
	engine := NewEngine()

	var mu sync.Mutex
	var receivedScore core.RiskScore
	callCount := 0

	engine.RiskUpdateSink = func(sc core.RiskScore) {
		mu.Lock()
		defer mu.Unlock()
		receivedScore = sc
		callCount++
	}

	engine.ProcessAlert(core.Alert{
		RuleName: "test",
		Severity: core.SeverityHigh,
		SourceIP: "10.0.0.5",
	})

	mu.Lock()
	defer mu.Unlock()
	if callCount != 1 {
		t.Fatalf("expected RiskUpdateSink to be called 1 time, got %d", callCount)
	}
	if receivedScore.SourceIP != "10.0.0.5" {
		t.Errorf("expected source IP 10.0.0.5, got %s", receivedScore.SourceIP)
	}
	if receivedScore.Score != 60 {
		t.Errorf("expected score 60, got %d", receivedScore.Score)
	}
}

func TestEngine_GetScore_UnknownIP(t *testing.T) {
	engine := NewEngine()

	score := engine.GetScore("192.168.1.1")
	if score.Score != 0 {
		t.Errorf("expected score 0 for unknown IP, got %d", score.Score)
	}
	if score.ThreatLevel != core.ThreatLow {
		t.Errorf("expected LOW threat for unknown IP, got %s", score.ThreatLevel)
	}
	if score.SourceIP != "192.168.1.1" {
		t.Errorf("expected source IP returned, got %s", score.SourceIP)
	}
}

func TestEngine_ProcessAlert_EmptySourceIPFallback(t *testing.T) {
	engine := NewEngine()

	engine.ProcessAlert(core.Alert{
		RuleName: "test",
		Severity: core.SeverityMedium,
		SourceIP: "",
	})

	score := engine.GetScore("global")
	if score.Score != 30 {
		t.Errorf("expected score 30 for global fallback, got %d", score.Score)
	}
}
