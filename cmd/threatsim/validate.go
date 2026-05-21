package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
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
	Status       string          `json:"status"`        // "PASS" or "FAIL"
	PluginID     string          `json:"plugin_id"`     // Which attack was simulated
	EventsTotal  int             `json:"events_total"`  // How many events were generated
	AlertsFired  int             `json:"alerts_fired"`  // How many detection alerts fired
	RiskScores   []core.RiskScore `json:"risk_scores"`  // Final risk assessments
	RulesLoaded  int             `json:"rules_loaded"`  // How many detection rules were loaded
	Duration     string          `json:"duration"`      // Wall-clock time of the validation
	ExpectAlert  bool            `json:"expect_alert"`  // Whether we expected alerts to fire
	Message      string          `json:"message"`       // Human-readable summary
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
	)

	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Run a security detection validation gate (for CI/CD pipelines)",
		Long: `Validates that your security detection rules are working correctly.

This is the core CI/CD command. It:
  1. Simulates an attack using the specified plugin
  2. Feeds events through the detection engine with your YAML rules
  3. Checks whether the expected alerts fired
  4. Exits with code 0 (PASS) or 1 (FAIL)

Use this in your CI/CD pipeline to block deployments when detection is broken.

Examples:
  # Basic: verify brute force detection works
  threatsim validate --plugin brute_force --expect-alert

  # CI mode: JSON output for machine parsing
  threatsim validate --plugin brute_force --expect-alert --json

  # Custom rules directory
  threatsim validate --plugin port_scan --expect-alert --rules ./my-rules/

  # Fast validation with higher event rate
  threatsim validate --plugin brute_force --expect-alert --rate 20 --duration 3s`,
		RunE: func(cmd *cobra.Command, args []string) error {
			startTime := time.Now()

			// Validate plugin exists
			plugin, err := registry.Get(pluginID)
			if err != nil {
				if jsonOutput {
					report := ValidationReport{
						Status:   "FAIL",
						PluginID: pluginID,
						Message:  fmt.Sprintf("Plugin not found: %s", pluginID),
					}
					return outputReport(report)
				}
				color.Red("✗ Plugin not found: %s", pluginID)
				color.Yellow("\nAvailable plugins:")
				for _, id := range registry.IDs() {
					fmt.Printf("  • %s\n", id)
				}
				return fmt.Errorf("plugin not found: %s", pluginID)
			}

			if !jsonOutput {
				printValidationHeader(plugin, expectAlert)
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
					report := ValidationReport{
						Status:      "FAIL",
						PluginID:    pluginID,
						RulesLoaded: 0,
						Message:     "No detection rules loaded. Cannot validate.",
					}
					return outputReport(report)
				}
				color.Red("✗ No detection rules loaded. Cannot validate.")
				color.Yellow("  Make sure YAML rule files exist in: %s", rulesDir)
				os.Exit(1)
			}

			// Collect alerts
			var alertCount int32
			var riskScores []core.RiskScore

			// Wire: Detection → Risk
			detectEngine.AlertSink = riskEngine.ProcessAlert

			// Wire: Risk → Collector
			riskEngine.RiskUpdateSink = func(sc core.RiskScore) {
				atomic.AddInt32(&alertCount, 1)
				riskScores = append(riskScores, sc)
				if !jsonOutput {
					printValidationAlert(sc)
				}
			}

			// Start detection engine in background
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			go detectEngine.Start(ctx)

			// Build plugin config
			config := plugin.DefaultConfig()
			if target != "" {
				config.Target = target
			}
			if duration != "" {
				config.Duration = duration
			} else {
				config.Duration = "5s" // Quick default for validation
			}
			if rate > 0 {
				config.Rate = rate
			}

			// Create event sink
			var eventCount int32
			sink := func(event core.Event) error {
				atomic.AddInt32(&eventCount, 1)
				return stream.Publish(ctx, core.TopicAttackEvents, event)
			}

			if !jsonOutput {
				color.Cyan("  ▶ Running %s simulation...\n", plugin.Name())
			}

			// Execute the attack
			_ = plugin.Execute(ctx, config, sink)

			// Give the detection engine a moment to process final events
			time.Sleep(200 * time.Millisecond)
			cancel() // Stop the detection engine

			elapsed := time.Since(startTime)
			totalEvents := int(atomic.LoadInt32(&eventCount))
			totalAlerts := int(atomic.LoadInt32(&alertCount))

			// --- Evaluate Result ---
			var status string
			var message string

			if expectAlert {
				if totalAlerts > 0 {
					status = "PASS"
					message = fmt.Sprintf("Detection validated: %d alert(s) fired from %d events. Your security rules are working.", totalAlerts, totalEvents)
				} else {
					status = "FAIL"
					message = fmt.Sprintf("Detection FAILED: 0 alerts from %d events. Your security rules may be broken or misconfigured.", totalEvents)
				}
			} else {
				// Inverse validation: ensure NO alerts fire (e.g., testing that benign traffic doesn't trigger false positives)
				if totalAlerts == 0 {
					status = "PASS"
					message = fmt.Sprintf("Negative validation passed: 0 false positive alerts from %d events.", totalEvents)
				} else {
					status = "FAIL"
					message = fmt.Sprintf("Negative validation FAILED: %d unexpected alert(s) from %d events.", totalAlerts, totalEvents)
				}
			}

			report := ValidationReport{
				Status:      status,
				PluginID:    pluginID,
				EventsTotal: totalEvents,
				AlertsFired: totalAlerts,
				RiskScores:  riskScores,
				RulesLoaded: rulesCount,
				Duration:    elapsed.Round(time.Millisecond).String(),
				ExpectAlert: expectAlert,
				Message:     message,
			}

			if jsonOutput {
				return outputReport(report)
			}

			printValidationSummary(report)

			if status == "FAIL" {
				os.Exit(1)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&pluginID, "plugin", "brute_force", "Attack plugin to simulate")
	cmd.Flags().StringVar(&duration, "duration", "5s", "How long to run the simulation")
	cmd.Flags().IntVar(&rate, "rate", 15, "Events per second")
	cmd.Flags().StringVar(&target, "target", "", "Target to simulate against (uses plugin default if empty)")
	cmd.Flags().BoolVar(&expectAlert, "expect-alert", true, "If true, PASS when alerts fire. If false, PASS when no alerts fire.")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output result as JSON (for CI/CD systems)")
	cmd.Flags().StringVar(&rulesDir, "rules", "", "Path to detection rules directory (default: configs/rules)")

	return cmd
}

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

func printValidationHeader(plugin core.Plugin, expectAlert bool) {
	header := color.New(color.FgCyan, color.Bold)
	label := color.New(color.FgWhite, color.Faint)

	fmt.Println()
	header.Println("  ╔════════════════════════════════════════════╗")
	header.Println("  ║   🛡️  ThreatSIM Security Validation Gate   ║")
	header.Println("  ╚════════════════════════════════════════════╝")
	fmt.Println()
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

func printValidationSummary(report ValidationReport) {
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
