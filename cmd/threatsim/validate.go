package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/Stratify-Systems/ThreatSIM/internal/core"
	"github.com/Stratify-Systems/ThreatSIM/internal/detection"
	"github.com/Stratify-Systems/ThreatSIM/internal/risk"
	"github.com/Stratify-Systems/ThreatSIM/internal/streaming/memory"
)

// ValidationReport is the structured output of a validation run.
// CI/CD systems can parse this JSON to make deployment decisions.
type ValidationReport struct {
	Status      string           `json:"status"`       // "PASS" or "FAIL"
	Mode        string           `json:"mode"`         // "internal" or "external"
	PluginID    string           `json:"plugin_id"`    // Which attack was simulated
	Target      string           `json:"target"`       // What was attacked
	EventsTotal int              `json:"events_total"` // How many events were generated
	AlertsFired int              `json:"alerts_fired"` // Internal detection alerts
	RiskScores  []core.RiskScore `json:"risk_scores"`  // Final risk assessments
	RulesLoaded int              `json:"rules_loaded"` // How many detection rules were loaded
	Duration    string           `json:"duration"`     // Wall-clock time of the validation
	ExpectAlert bool             `json:"expect_alert"` // Whether we expected alerts to fire
	Message     string           `json:"message"`      // Human-readable summary

	// External validation fields (only populated when --verify is used)
	ExternalAlerts int    `json:"external_alerts,omitempty"` // Alerts detected by the external target
	ExternalEvents int    `json:"external_events,omitempty"` // Events logged by the external target
	VerifyURL      string `json:"verify_url,omitempty"`      // The URL that was queried
}

// externalAlertsResponse matches the JSON from the target app's /security/alerts
type externalAlertsResponse struct {
	Alerts []struct {
		Type       string `json:"type"`
		SourceIP   string `json:"source_ip"`
		EventCount int    `json:"event_count"`
		Message    string `json:"message"`
	} `json:"alerts"`
	TotalEvents int `json:"total_events"`
}

func newValidateCmd() *cobra.Command {
	var (
		pluginID    string
		duration    string
		rate        int
		target      string
		expectAlert bool
		jsonOutput  bool
		rulesDir    string
		verifyURL   string
	)

	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Run a security detection validation gate (for CI/CD pipelines)",
		Long: `Validates that security detection systems are working correctly.

There are two modes:

  INTERNAL MODE (default):
    Simulates an attack and checks ThreatSIM's own detection engine.
    Useful for validating that your YAML detection rules are correct.

  EXTERNAL MODE (with --verify):
    Sends REAL attack traffic to a target application, then queries
    the target's security API to verify it detected the attack.
    This validates the EXTERNAL system's detection capabilities.

Examples:
  # Internal: validate your detection rules
  threatsim validate --plugin brute_force --expect-alert

  # External: attack a real target and verify it caught it
  threatsim validate --plugin brute_force \
      --target http://localhost:9999/login \
      --verify http://localhost:9999/security/alerts

  # CI mode: JSON output for machine parsing
  threatsim validate --plugin brute_force --expect-alert --json

  # Custom rules and rate
  threatsim validate --plugin brute_force --rate 20 --duration 3s`,
		RunE: func(cmd *cobra.Command, args []string) error {
			startTime := time.Now()

			// Determine mode
			isExternal := verifyURL != ""

			// Validate plugin exists
			plugin, err := registry.Get(pluginID)
			if err != nil {
				if jsonOutput {
					report := ValidationReport{Status: "FAIL", PluginID: pluginID,
						Message: fmt.Sprintf("Plugin not found: %s", pluginID)}
					return outputReport(report)
				}
				color.Red("✗ Plugin not found: %s", pluginID)
				color.Yellow("\nAvailable plugins:")
				for _, id := range registry.IDs() {
					fmt.Printf("  • %s\n", id)
				}
				return fmt.Errorf("plugin not found: %s", pluginID)
			}

			// Build plugin config
			config := plugin.DefaultConfig()
			if target != "" {
				config.Target = target
			}
			if duration != "" {
				config.Duration = duration
			} else {
				config.Duration = "5s"
			}
			if rate > 0 {
				config.Rate = rate
			}

			// Enable active mode for external validation
			if isExternal {
				config.ActiveMode = true
			}

			// ════════════════════════════════════════
			// EXTERNAL VALIDATION MODE
			// ════════════════════════════════════════
			if isExternal {
				return runExternalValidation(plugin, config, verifyURL, expectAlert, jsonOutput, startTime)
			}

			// ════════════════════════════════════════
			// INTERNAL VALIDATION MODE
			// ════════════════════════════════════════
			return runInternalValidation(plugin, config, rulesDir, expectAlert, jsonOutput, startTime)
		},
	}

	cmd.Flags().StringVar(&pluginID, "plugin", "brute_force", "Attack plugin to simulate")
	cmd.Flags().StringVar(&duration, "duration", "5s", "How long to run the simulation")
	cmd.Flags().IntVar(&rate, "rate", 15, "Events per second")
	cmd.Flags().StringVar(&target, "target", "", "Target to simulate against (uses plugin default if empty)")
	cmd.Flags().BoolVar(&expectAlert, "expect-alert", true, "If true, PASS when alerts fire. If false, PASS when no alerts fire.")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output result as JSON (for CI/CD systems)")
	cmd.Flags().StringVar(&rulesDir, "rules", "", "Path to detection rules directory (default: configs/rules)")
	cmd.Flags().StringVar(&verifyURL, "verify", "", "URL to query for external alert verification (enables external mode)")

	return cmd
}

// ══════════════════════════════════════════════════════════════
// EXTERNAL VALIDATION — Attacks a real target and verifies
// that the target's OWN security system detected the attack.
// ══════════════════════════════════════════════════════════════

func runExternalValidation(plugin core.Plugin, config core.PluginConfig, verifyURL string, expectAlert bool, jsonOutput bool, startTime time.Time) error {

	if !jsonOutput {
		printExternalHeader(plugin, config, verifyURL, expectAlert)
	}

	// Step 1: Reset the target's security state for a clean test
	resetURL := strings.TrimSuffix(verifyURL, "/alerts") + "/reset"
	if !jsonOutput {
		color.Cyan("  [1/4] Resetting target security state...")
	}
	resetReq, _ := http.NewRequest("POST", resetURL, nil)
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(resetReq)
	if err != nil {
		if !jsonOutput {
			color.Yellow("  ⚠ Could not reset target state: %v (continuing anyway)", err)
		}
	} else {
		resp.Body.Close()
		if !jsonOutput {
			color.Green("  ✓ Target state reset.")
		}
	}

	// Step 2: Send real attack traffic
	if !jsonOutput {
		fmt.Println()
		color.Cyan("  [2/4] Sending real attack traffic to %s...", config.Target)
	}

	var eventCount int32
	ctx := context.Background()
	sink := func(event core.Event) error {
		atomic.AddInt32(&eventCount, 1)
		return nil // We don't need the internal stream for external validation
	}

	_ = plugin.Execute(ctx, config, sink)
	totalEvents := int(atomic.LoadInt32(&eventCount))

	if !jsonOutput {
		color.Green("  ✓ Sent %d attack requests to target.", totalEvents)
	}

	// Step 3: Wait for target to process
	if !jsonOutput {
		fmt.Println()
		color.Cyan("  [3/4] Waiting for target to process events...")
	}
	time.Sleep(1 * time.Second)

	// Step 4: Query the target's security API
	if !jsonOutput {
		fmt.Println()
		color.Cyan("  [4/4] Querying target security system...")
		fmt.Printf("        GET %s\n", verifyURL)
	}

	resp, err = client.Get(verifyURL)
	if err != nil {
		msg := fmt.Sprintf("Failed to query target security API at %s: %v", verifyURL, err)
		if jsonOutput {
			return outputReport(ValidationReport{Status: "FAIL", Mode: "external", PluginID: plugin.ID(), Message: msg})
		}
		color.Red("\n  ✗ %s", msg)
		os.Exit(1)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var externalResult externalAlertsResponse
	if err := json.Unmarshal(body, &externalResult); err != nil {
		msg := fmt.Sprintf("Failed to parse target response: %v", err)
		if jsonOutput {
			return outputReport(ValidationReport{Status: "FAIL", Mode: "external", PluginID: plugin.ID(), Message: msg})
		}
		color.Red("\n  ✗ %s", msg)
		os.Exit(1)
	}

	externalAlertCount := len(externalResult.Alerts)
	externalEventCount := externalResult.TotalEvents

	if !jsonOutput {
		if externalAlertCount > 0 {
			color.Green("  ✓ Target detected: %d alert(s) from %d events", externalAlertCount, externalEventCount)
			for _, a := range externalResult.Alerts {
				color.Yellow("    🔔 %s: %s", a.Type, a.Message)
			}
		} else {
			color.Red("  ✗ Target detected: 0 alerts from %d events", externalEventCount)
		}
	}

	// --- Evaluate ---
	elapsed := time.Since(startTime)
	var status, message string

	if expectAlert {
		if externalAlertCount > 0 {
			status = "PASS"
			message = fmt.Sprintf("External validation passed: Target's security system detected %d alert(s) from %d requests. The target application's defenses are working.",
				externalAlertCount, totalEvents)
		} else {
			status = "FAIL"
			message = fmt.Sprintf("External validation FAILED: Sent %d attack requests but target detected 0 alerts. The target application's security monitoring may be broken.",
				totalEvents)
		}
	} else {
		if externalAlertCount == 0 {
			status = "PASS"
			message = fmt.Sprintf("Negative external validation passed: %d requests sent, 0 false positives from target.", totalEvents)
		} else {
			status = "FAIL"
			message = fmt.Sprintf("Negative external validation FAILED: %d unexpected alert(s) from target.", externalAlertCount)
		}
	}

	report := ValidationReport{
		Status:         status,
		Mode:           "external",
		PluginID:       plugin.ID(),
		Target:         config.Target,
		EventsTotal:    totalEvents,
		ExternalAlerts: externalAlertCount,
		ExternalEvents: externalEventCount,
		VerifyURL:      verifyURL,
		Duration:       elapsed.Round(time.Millisecond).String(),
		ExpectAlert:    expectAlert,
		Message:        message,
	}

	if jsonOutput {
		return outputReport(report)
	}

	printExternalSummary(report)

	if status == "FAIL" {
		os.Exit(1)
	}
	return nil
}

// ══════════════════════════════════════════════════════════════
// INTERNAL VALIDATION — Tests ThreatSIM's own detection engine
// against its own YAML rules. Useful for rule validation.
// ══════════════════════════════════════════════════════════════

func runInternalValidation(plugin core.Plugin, config core.PluginConfig, rulesDir string, expectAlert bool, jsonOutput bool, startTime time.Time) error {

	if !jsonOutput {
		printInternalHeader(plugin, expectAlert)
	}

	// --- Build the full pipeline in-memory ---
	stream := memory.NewStream()
	defer stream.Close()

	detectEngine := detection.NewEngine(stream)
	riskEngine := risk.NewEngine()

	// Load detection rules
	if rulesDir == "" {
		rulesDir = "configs/rules"
	}
	if err := detectEngine.LoadRulesFromDir(rulesDir); err != nil {
		if !jsonOutput {
			color.Yellow("⚠ Could not load detection rules from %s: %v", rulesDir, err)
		}
	}

	rulesCount := detectEngine.RulesCount()
	if rulesCount == 0 {
		if jsonOutput {
			return outputReport(ValidationReport{Status: "FAIL", Mode: "internal", PluginID: plugin.ID(),
				RulesLoaded: 0, Message: "No detection rules loaded. Cannot validate."})
		}
		color.Red("✗ No detection rules loaded. Cannot validate.")
		color.Yellow("  Make sure YAML rule files exist in: %s", rulesDir)
		os.Exit(1)
	}

	// Collect alerts
	var alertCount int32
	var mu sync.Mutex
	var riskScores []core.RiskScore

	// Wire: Detection → Risk
	detectEngine.AlertSink = riskEngine.ProcessAlert

	// Wire: Risk → Collector
	riskEngine.RiskUpdateSink = func(sc core.RiskScore) {
		atomic.AddInt32(&alertCount, 1)
		mu.Lock()
		riskScores = append(riskScores, sc)
		mu.Unlock()
		if !jsonOutput {
			printValidationAlert(sc)
		}
	}

	// Start detection engine in background
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go detectEngine.Start(ctx)

	if !jsonOutput {
		color.Cyan("  ▶ Running %s simulation...\n", plugin.Name())
	}

	// Create event sink
	var eventCount int32
	sink := func(event core.Event) error {
		atomic.AddInt32(&eventCount, 1)
		return stream.Publish(ctx, core.TopicAttackEvents, event)
	}

	// Execute the attack
	_ = plugin.Execute(ctx, config, sink)

	// Give the detection engine a moment to process final events
	time.Sleep(200 * time.Millisecond)
	cancel()

	elapsed := time.Since(startTime)
	totalEvents := int(atomic.LoadInt32(&eventCount))
	totalAlerts := int(atomic.LoadInt32(&alertCount))

	// --- Evaluate ---
	var status, message string

	if expectAlert {
		if totalAlerts > 0 {
			status = "PASS"
			message = fmt.Sprintf("Detection validated: %d alert(s) fired from %d events. Your security rules are working.", totalAlerts, totalEvents)
		} else {
			status = "FAIL"
			message = fmt.Sprintf("Detection FAILED: 0 alerts from %d events. Your security rules may be broken or misconfigured.", totalEvents)
		}
	} else {
		if totalAlerts == 0 {
			status = "PASS"
			message = fmt.Sprintf("Negative validation passed: 0 false positive alerts from %d events.", totalEvents)
		} else {
			status = "FAIL"
			message = fmt.Sprintf("Negative validation FAILED: %d unexpected alert(s) from %d events.", totalAlerts, totalEvents)
		}
	}

	mu.Lock()
	scoresCopy := make([]core.RiskScore, len(riskScores))
	copy(scoresCopy, riskScores)
	mu.Unlock()

	report := ValidationReport{
		Status:      status,
		Mode:        "internal",
		PluginID:    plugin.ID(),
		Target:      config.Target,
		EventsTotal: totalEvents,
		AlertsFired: totalAlerts,
		RiskScores:  scoresCopy,
		RulesLoaded: rulesCount,
		Duration:    elapsed.Round(time.Millisecond).String(),
		ExpectAlert: expectAlert,
		Message:     message,
	}

	if jsonOutput {
		return outputReport(report)
	}

	printInternalSummary(report)

	if status == "FAIL" {
		os.Exit(1)
	}
	return nil
}

// ══════════════════════════════════════════════════════════════
// Output Helpers
// ══════════════════════════════════════════════════════════════

func outputReport(report ValidationReport) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(report); err != nil {
		return err
	}
	if report.Status == "FAIL" {
		os.Exit(1)
	}
	return nil
}

// --- External mode headers/summaries ---

func printExternalHeader(plugin core.Plugin, config core.PluginConfig, verifyURL string, expectAlert bool) {
	header := color.New(color.FgCyan, color.Bold)
	label := color.New(color.FgWhite, color.Faint)

	fmt.Println()
	header.Println("  ╔══════════════════════════════════════════════════╗")
	header.Println("  ║   🛡️  ThreatSIM External Validation Gate         ║")
	header.Println("  ╚══════════════════════════════════════════════════╝")
	fmt.Println()
	label.Printf("  Mode:         ")
	color.New(color.FgMagenta, color.Bold).Println("EXTERNAL (attacking a real target)")
	label.Printf("  Attack:       ")
	fmt.Println(plugin.Name())
	label.Printf("  Target:       ")
	color.New(color.FgYellow).Println(config.Target)
	label.Printf("  Verify:       ")
	color.New(color.FgYellow).Println(verifyURL)
	label.Printf("  Expect Alert: ")
	if expectAlert {
		color.Green("YES (PASS if target detects the attack)")
	} else {
		color.Yellow("NO (PASS if no false positives)")
	}
	fmt.Println("  ──────────────────────────────────────────────────")
	fmt.Println()
}

func printExternalSummary(report ValidationReport) {
	fmt.Println()
	fmt.Println("  ══════════════════════════════════════════════════")

	if report.Status == "PASS" {
		color.New(color.FgGreen, color.Bold).Println("  ✅ EXTERNAL VALIDATION PASSED")
	} else {
		color.New(color.FgRed, color.Bold).Println("  ❌ EXTERNAL VALIDATION FAILED")
	}

	label := color.New(color.FgWhite, color.Faint)
	label.Printf("  Requests Sent:  ")
	fmt.Printf("%d to target\n", report.EventsTotal)
	label.Printf("  Target Alerts:  ")
	if report.ExternalAlerts > 0 {
		color.Green("%d detected by target's security system", report.ExternalAlerts)
	} else {
		color.Red("0 — target did NOT detect the attack")
	}
	label.Printf("  Target Events:  ")
	fmt.Printf("%d logged by target\n", report.ExternalEvents)
	label.Printf("  Duration:       ")
	fmt.Println(report.Duration)
	label.Printf("  Message:        ")
	fmt.Println(report.Message)

	fmt.Println("  ══════════════════════════════════════════════════")
	fmt.Println()
}

// --- Internal mode headers/summaries ---

func printInternalHeader(plugin core.Plugin, expectAlert bool) {
	header := color.New(color.FgCyan, color.Bold)
	label := color.New(color.FgWhite, color.Faint)

	fmt.Println()
	header.Println("  ╔════════════════════════════════════════════╗")
	header.Println("  ║   🛡️  ThreatSIM Security Validation Gate   ║")
	header.Println("  ╚════════════════════════════════════════════╝")
	fmt.Println()
	label.Printf("  Mode:         ")
	fmt.Println("INTERNAL (testing own detection rules)")
	label.Printf("  Attack:       ")
	fmt.Println(plugin.Name())
	label.Printf("  Plugin:       ")
	fmt.Println(plugin.ID())
	label.Printf("  Expect Alert: ")
	if expectAlert {
		color.Green("YES (PASS if detection fires)")
	} else {
		color.Yellow("NO (PASS if no false positives)")
	}
	fmt.Println("  ─────────────────────────────────────────────")
	fmt.Println()
}

func printValidationAlert(sc core.RiskScore) {
	color.Yellow("  🔔 Alert fired: %s → Score: %d, Level: %s", sc.SourceIP, sc.Score, sc.ThreatLevel)
}

func printInternalSummary(report ValidationReport) {
	fmt.Println()
	fmt.Println("  ─────────────────────────────────────────────")

	if report.Status == "PASS" {
		color.New(color.FgGreen, color.Bold).Println("  ✅ VALIDATION PASSED")
	} else {
		color.New(color.FgRed, color.Bold).Println("  ❌ VALIDATION FAILED")
	}

	label := color.New(color.FgWhite, color.Faint)
	label.Printf("  Events:     ")
	fmt.Printf("%d generated\n", report.EventsTotal)
	label.Printf("  Alerts:     ")
	fmt.Printf("%d fired\n", report.AlertsFired)
	label.Printf("  Rules:      ")
	fmt.Printf("%d loaded\n", report.RulesLoaded)
	label.Printf("  Duration:   ")
	fmt.Println(report.Duration)
	label.Printf("  Message:    ")
	fmt.Println(report.Message)

	fmt.Println("  ─────────────────────────────────────────────")
	fmt.Println()
}
