package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/Stratify-Systems/ThreatSIM/internal/alerting"
	"github.com/Stratify-Systems/ThreatSIM/internal/core"
	"github.com/Stratify-Systems/ThreatSIM/internal/detection"
	"github.com/Stratify-Systems/ThreatSIM/internal/risk"
	"github.com/Stratify-Systems/ThreatSIM/internal/streaming/memory"
)

func newSimulateCmd() *cobra.Command {
	var (
		target   string
		service  string
		sourceIP string
		duration string
		rate     int
		useRedis bool
	)

	cmd := &cobra.Command{
		Use:   "simulate <plugin>",
		Short: "Run an attack simulation",
		Long: `Simulate a cyber attack using the specified plugin.

Examples:
  threatsim simulate brute_force
  threatsim simulate port_scan --target 10.0.0.1 --duration 60s
  threatsim simulate brute_force --rate 10 --service auth-api`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pluginID := args[0]

			// Look up the plugin
			plugin, err := registry.Get(pluginID)
			if err != nil {
				color.Red("✗ %v", err)
				color.Yellow("\nAvailable plugins:")
				for _, id := range registry.IDs() {
					fmt.Printf("  • %s\n", id)
				}
				return err
			}

			// Build config (merge defaults with CLI flags)
			config := plugin.DefaultConfig()
			if target != "" {
				config.Target = target
			}
			if service != "" {
				config.Service = service
			}
			if sourceIP != "" {
				config.SourceIP = sourceIP
			}
			if duration != "" {
				config.Duration = duration
			}
			if rate > 0 {
				config.Rate = rate
			}

			// Set up event stream
			var stream core.EventStream
			if useRedis {
				// TODO: Add Redis stream support with --redis-addr flag
				color.Yellow("Redis streaming not yet configured. Using in-memory stream.")
				stream = memory.NewStream()
			} else {
				stream = memory.NewStream()
			}
			defer stream.Close()

			// Set up graceful shutdown
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			// --- Setup Detection & Risk Engine ---
			riskEngine := risk.NewEngine()
			detectEngine := detection.NewEngine(stream)

			// Load rules
			if err := detectEngine.LoadRulesFromDir("configs/rules"); err != nil {
				color.Yellow("⚠ Could not load detection rules: %v", err)
			}

			// --- Setup Alerting Dispatcher ---
			dispatcher := alerting.NewDispatcher()
			// You would typically load these URLs/credentials from a config or env block
			if hook := os.Getenv("THREATSIM_WEBHOOK_URL"); hook != "" {
				dispatcher.Register(alerting.NewWebhookNotifier(hook))
			}
			if slack := os.Getenv("THREATSIM_SLACK_URL"); slack != "" {
				dispatcher.Register(alerting.NewSlackNotifier(slack))
			}

			// Wire Detection -> Risk
			detectEngine.AlertSink = riskEngine.ProcessAlert

			// Wire Risk -> Output (Terminal Alert) + External Dispatcher
			riskEngine.RiskUpdateSink = func(sc core.RiskScore) {
				printRiskAlert(sc)
				dispatcher.Dispatch(sc)
			}

			// Start Detection engine in background
			go detectEngine.Start(ctx)
			// -------------------------------------

			sigChan := make(chan os.Signal, 1)
			signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

			go func() {
				<-sigChan
				color.Yellow("\n⚠ Stopping simulation...")
				cancel()
			}()

			// Print simulation header
			printSimulationHeader(plugin, config)

			// Event counter
			eventCount := 0
			startTime := time.Now()

			// Create event sink that prints events and publishes to stream
			sink := func(event core.Event) error {
				eventCount++

				// Publish to stream for downstream consumers
				if err := stream.Publish(ctx, core.TopicAttackEvents, event); err != nil {
					return err
				}

				// Pretty-print the event to console
				printEvent(event, eventCount)
				return nil
			}

			// Run the attack!
			color.Green("▶ Simulation started\n")
			err = plugin.Execute(ctx, config, sink)

			elapsed := time.Since(startTime)

			if err != nil && err != context.Canceled {
				color.Red("\n✗ Simulation failed: %v", err)
				return err
			}

			// Print summary
			printSimulationSummary(plugin, eventCount, elapsed)
			return nil
		},
	}

	// CLI flags
	cmd.Flags().StringVar(&target, "target", "", "Target IP or hostname")
	cmd.Flags().StringVar(&service, "service", "", "Target service name")
	cmd.Flags().StringVar(&sourceIP, "source-ip", "", "Source IP to simulate from")
	cmd.Flags().StringVarP(&duration, "duration", "d", "", "How long to run (e.g., 30s, 5m)")
	cmd.Flags().IntVarP(&rate, "rate", "r", 0, "Events per second")
	cmd.Flags().BoolVar(&useRedis, "redis", false, "Use Redis Streams (requires running Redis)")

	return cmd
}

func printSimulationHeader(plugin core.Plugin, config core.PluginConfig) {
	header := color.New(color.FgCyan, color.Bold)
	label := color.New(color.FgWhite, color.Faint)

	fmt.Println()
	header.Printf("  ⚔  %s\n", plugin.Name())
	fmt.Println("  " + "─────────────────────────────────────")
	label.Printf("  Plugin:    ")
	fmt.Println(plugin.ID())
	label.Printf("  Target:    ")
	fmt.Println(config.Target)
	label.Printf("  Service:   ")
	fmt.Println(config.Service)
	label.Printf("  Source IP: ")
	fmt.Println(config.SourceIP)
	label.Printf("  Duration:  ")
	fmt.Println(config.Duration)
	label.Printf("  Rate:      ")
	fmt.Printf("%d events/sec\n", config.Rate)
	fmt.Println("  " + "─────────────────────────────────────")
	fmt.Println()
}

func printEvent(event core.Event, count int) {
	ts := event.Timestamp.Format("15:04:05.000")

	// Color based on event type
	var typeColor *color.Color
	switch event.Type {
	case "login_failed":
		typeColor = color.New(color.FgRed)
	case "port_probe":
		typeColor = color.New(color.FgYellow)
	default:
		typeColor = color.New(color.FgWhite)
	}

	dim := color.New(color.Faint)

	dim.Printf("  [%s] ", ts)
	typeColor.Printf("%-15s", event.Type)
	dim.Printf(" │ ")
	fmt.Printf("%s → %s", event.SourceIP, event.Target)

	// Print extra context based on event type
	if event.User != "" {
		dim.Printf(" │ user=")
		fmt.Printf("%s", event.User)
	}

	if port, ok := event.Metadata["port"]; ok {
		dim.Printf(" │ port=")
		fmt.Printf("%v", port)
		if status, ok := event.Metadata["port_status"]; ok {
			if status == "open" {
				color.New(color.FgGreen).Printf(" [OPEN]")
			}
		}
	}

	fmt.Println()
}

func printSimulationSummary(plugin core.Plugin, eventCount int, elapsed time.Duration) {
	summary := color.New(color.FgGreen, color.Bold)
	label := color.New(color.FgWhite, color.Faint)

	fmt.Println()
	fmt.Println("  " + "─────────────────────────────────────")
	summary.Println("  ✓ Simulation Complete")
	label.Printf("  Plugin:     ")
	fmt.Println(plugin.Name())
	label.Printf("  Events:     ")
	fmt.Printf("%d events generated\n", eventCount)
	label.Printf("  Duration:   ")
	fmt.Printf("%s\n", elapsed.Round(time.Millisecond))
	label.Printf("  Throughput: ")
	if elapsed.Seconds() > 0 {
		fmt.Printf("%.1f events/sec\n", float64(eventCount)/elapsed.Seconds())
	} else {
		fmt.Println("N/A")
	}
	fmt.Println("  " + "─────────────────────────────────────")
	fmt.Println()
}

func printRiskAlert(sc core.RiskScore) {
fmt.Println()
box := color.New(color.FgWhite, color.Bold)

switch sc.ThreatLevel {
case core.ThreatCritical:
box = color.New(color.BgHiRed, color.FgWhite, color.Bold)
case core.ThreatHigh:
box = color.New(color.BgRed, color.FgWhite, color.Bold)
case core.ThreatMedium:
box = color.New(color.BgYellow, color.FgBlack, color.Bold)
case core.ThreatLow:
box = color.New(color.BgGreen, color.FgWhite, color.Bold)
}

fmt.Println("  ┌──────────────────────────────────────────┐")
fmt.Printf("  │ 🚨 ")
box.Printf("%-38s", fmt.Sprintf("ALERT: %s", sc.ThreatLevel))
fmt.Println(" │")

fmt.Println("  │                                          │")
fmt.Println("  │  Suspicious Activity Detected            │")
fmt.Printf("  │  Source IP:       %-22s │\n", sc.SourceIP)
fmt.Printf("  │  Risk Score:      %-22v │\n", fmt.Sprintf("%v", sc.Score))

rulesJoined := ""
for _, f := range sc.Factors {
if len(rulesJoined) > 0 {
rulesJoined += ", "
}
rulesJoined += f
}
if len(rulesJoined) > 20 {
rulesJoined = rulesJoined[:17] + "..."
}
fmt.Printf("  │  Rules Tripped:   %-22s │\n", rulesJoined)
fmt.Println("  └──────────────────────────────────────────┘")
}
