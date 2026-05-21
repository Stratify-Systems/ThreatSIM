package detection_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/Stratify-Systems/ThreatSIM/internal/core"
	"github.com/Stratify-Systems/ThreatSIM/internal/detection"
	"github.com/Stratify-Systems/ThreatSIM/internal/plugins"
	bruteforce "github.com/Stratify-Systems/ThreatSIM/internal/plugins/brute_force"
	"github.com/Stratify-Systems/ThreatSIM/internal/risk"
	"github.com/Stratify-Systems/ThreatSIM/internal/streaming/memory"
)

// TestFullPipelineEndToEnd validates the entire ThreatSIM pipeline:
//
//	Plugin → Stream → Detection Engine → Risk Engine → Alert Output
//
// This is the most important test in the project. It proves the core concept works:
// an attack simulation triggers detection rules and produces alerts.
func TestFullPipelineEndToEnd(t *testing.T) {
	// --- Setup the pipeline ---
	stream := memory.NewStream()
	defer stream.Close()

	detectEngine := detection.NewEngine(stream)
	riskEngine := risk.NewEngine()

	// Load real detection rules
	if err := detectEngine.LoadRulesFromDir("../../configs/rules"); err != nil {
		t.Fatalf("Failed to load detection rules: %v", err)
	}

	if detectEngine.RulesCount() == 0 {
		t.Fatal("No detection rules loaded — check configs/rules/")
	}

	// Collect alerts
	var mu sync.Mutex
	var alerts []core.RiskScore

	// Wire: Detection → Risk
	detectEngine.AlertSink = riskEngine.ProcessAlert

	// Wire: Risk → Collector
	riskEngine.RiskUpdateSink = func(sc core.RiskScore) {
		mu.Lock()
		alerts = append(alerts, sc)
		mu.Unlock()
	}

	// Start detection engine
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go detectEngine.Start(ctx)

	// --- Run the brute force plugin ---
	reg := plugins.NewRegistry()
	reg.Register(&bruteforce.Plugin{})

	plugin, _ := reg.Get("brute_force")
	config := plugin.DefaultConfig()
	config.Duration = "3s"
	config.Rate = 15 // 15 events/sec for 3 seconds = 45 events (threshold is 10)

	eventCount := 0
	sink := func(event core.Event) error {
		eventCount++
		return stream.Publish(ctx, core.TopicAttackEvents, event)
	}

	err := plugin.Execute(ctx, config, sink)
	if err != nil {
		t.Fatalf("Plugin execution failed: %v", err)
	}

	// Give detection engine time to process
	time.Sleep(300 * time.Millisecond)
	cancel()

	// --- Assert ---
	if eventCount == 0 {
		t.Fatal("Plugin generated 0 events — something is broken")
	}

	mu.Lock()
	alertCount := len(alerts)
	mu.Unlock()

	if alertCount == 0 {
		t.Fatalf("PIPELINE FAILURE: %d events generated but 0 alerts fired. The detection engine didn't catch the brute force attack.", eventCount)
	}

	t.Logf("✅ Pipeline validated: %d events → %d alert(s) fired", eventCount, alertCount)

	// Verify alert contents
	mu.Lock()
	firstAlert := alerts[0]
	mu.Unlock()

	if firstAlert.Score <= 0 {
		t.Errorf("Expected positive risk score, got %d", firstAlert.Score)
	}

	if firstAlert.ThreatLevel == core.ThreatLow {
		t.Errorf("Expected threat level > LOW for brute force, got %s", firstAlert.ThreatLevel)
	}

	t.Logf("   Source IP: %s, Score: %d, Level: %s", firstAlert.SourceIP, firstAlert.Score, firstAlert.ThreatLevel)
}
