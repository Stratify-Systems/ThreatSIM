package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/Stratify-Systems/ThreatSIM/internal/core"
	"github.com/Stratify-Systems/ThreatSIM/internal/detection"
	"github.com/Stratify-Systems/ThreatSIM/internal/risk"
	"github.com/Stratify-Systems/ThreatSIM/internal/scenario"
	"github.com/Stratify-Systems/ThreatSIM/internal/streaming/memory"
)

func newScenarioCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "scenario <name>",
		Short: "Run a multi-step attack scenario by name",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			filePath := fmt.Sprintf("configs/scenarios/%s.yaml", name)

			// Fallback to checking if the argument is actually a direct file path
			if _, err := os.Stat(filePath); os.IsNotExist(err) {
				filePath = name
			}

			color.Cyan("\n🔍 Loading Scenario: %s (from %s)\n", name, filePath)

			// Setup context with cancellation (Ctrl+C)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			sigChan := make(chan os.Signal, 1)
			signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
			go func() {
				<-sigChan
				fmt.Println("\n\n⏹  Cancellation received, stopping scenario...")
				cancel()
			}()

			// Load the scenario
			s, err := scenario.LoadFromFile(filePath)
			if err != nil {
				color.Red("✗ Failed to load scenario: %v", err)
				return err
			}

			// Initialize the stream
			stream := memory.NewStream()

			// --- Setup Detection & Risk Engine ---
			riskEngine := risk.NewEngine()
			detectEngine := detection.NewEngine(stream)

			if err := detectEngine.LoadRulesFromDir("configs/rules"); err != nil {
				color.Yellow("⚠ Warning: Could not load detection rules: %v", err)
			}

			// Wire Detection -> Risk
			detectEngine.AlertSink = riskEngine.ProcessAlert

			// Wire Risk -> Output (Terminal Alert)
			riskEngine.RiskUpdateSink = func(sc core.RiskScore) {
				printRiskAlert(sc)
			}

			// Start Detection engine in background
			go detectEngine.Start(ctx)

			// Create the event sink to pass to the engine
			var eventCount int32
			sink := func(event core.Event) error {
				atomic.AddInt32(&eventCount, 1)
				// Publish to stream for downstream consumers (Detection Engine)
				return stream.Publish(ctx, core.TopicAttackEvents, event)
			}

			startTime := time.Now()

			// Run the scenario engine
			engine := scenario.NewEngine(registry)
			err = engine.Run(ctx, s, sink)

			elapsed := time.Since(startTime)

			if err != nil && err != context.Canceled {
				color.Red("\n✗ Scenario failed: %v", err)
				return err
			}

			// Stop the detection engine and flush the stream
			cancel()

			color.Green("\n🏁 Scenario Complete!")
			fmt.Printf("Total run time: %v\n", elapsed)
			fmt.Printf("Total events generated: %d\n\n", atomic.LoadInt32(&eventCount))

			return nil
		},
	}

	return cmd
}
