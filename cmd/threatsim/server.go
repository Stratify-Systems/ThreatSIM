package main

import (
	"context"
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/Stratify-Systems/ThreatSIM/internal/alerting"
	"github.com/Stratify-Systems/ThreatSIM/internal/api"
	"github.com/Stratify-Systems/ThreatSIM/internal/core"
	"github.com/Stratify-Systems/ThreatSIM/internal/detection"
	"github.com/Stratify-Systems/ThreatSIM/internal/risk"
	"github.com/Stratify-Systems/ThreatSIM/internal/store"
	"github.com/Stratify-Systems/ThreatSIM/internal/streaming/memory"
)

func newServerCmd() *cobra.Command {
	var addr string

	cmd := &cobra.Command{
		Use:   "server",
		Short: "Start the ThreatSIM API server",
		Long:  `Run the ThreatSIM REST API to serve telemetry, events, and alerts.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			color.Cyan("Starting ThreatSIM API Server on %s...", addr)

			// Initialize the store (Postgres if available, InMemory fallback)
			var apiStore api.Store

			dsn := os.Getenv("DATABASE_URL")
			if dsn == "" {
				dsn = "host=localhost port=5433 user=threatsim password=password123 dbname=threatsim sslmode=disable"
			}

			pgStore, err := store.NewPostgresStore(dsn)
			if err != nil {
				color.Yellow("⚠ Could not connect to Postgres: %v", err)
				color.Yellow("  Falling back to in-memory store (data will not persist across restarts).")
				apiStore = api.NewInMemoryStore()
			} else {
				color.Green("✓ Connected to Postgres.")
				color.Cyan("  Running DB Schema migrations...")
				if err := pgStore.Migrate("db/migrations"); err != nil {
					color.Red("  Migration failed: %v", err)
				} else {
					color.Green("  Migrations successful.")
				}
				apiStore = pgStore
			}

			// --- Core Setup: Stream, Detection, Risk, Alerting ---
			stream := memory.NewStream()

			// Save all stream events to our store
			go stream.Subscribe(context.Background(), core.TopicAttackEvents, func(ctx context.Context, event core.Event) error {
				return apiStore.AddEvent(event)
			})

			// Setup Detection & Risk
			riskEngine := risk.NewEngine()
			detectEngine := detection.NewEngine(stream)
			if err := detectEngine.LoadRulesFromDir("configs/rules"); err != nil {
				color.Yellow("⚠ Could not load detection rules: %v", err)
			}

			// Setup Alerting Dispatcher
			dispatcher := alerting.NewDispatcher()
			if hook := os.Getenv("THREATSIM_WEBHOOK_URL"); hook != "" {
				dispatcher.Register(alerting.NewWebhookNotifier(hook))
			}

			// Wire connections
			detectEngine.AlertSink = riskEngine.ProcessAlert
			riskEngine.RiskUpdateSink = func(sc core.RiskScore) {
				apiStore.AddAlert(sc)
				dispatcher.Dispatch(sc)
			}

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			go detectEngine.Start(ctx)

			// Initialize and start the HTTP server
			server := api.NewServer(apiStore, registry, stream)
			fmt.Printf("\nListening on http://localhost%s\n", addr)
			fmt.Println("─────────────────────────────────────")
			fmt.Println("  GET  /health")
			fmt.Println("  GET  /api/v1/simulations")
			fmt.Println("  POST /api/v1/simulations")
			fmt.Println("  GET  /api/v1/events")
			fmt.Println("  GET  /api/v1/alerts")
			fmt.Println("  WS   /ws/live")
			fmt.Println("  GET  /metrics")
			fmt.Println("─────────────────────────────────────")

			// Start the blocking server
			if err := server.Start(addr); err != nil {
				return err
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&addr, "addr", ":8080", "Address to listen on (e.g., :8080, :9090)")

	return cmd
}
