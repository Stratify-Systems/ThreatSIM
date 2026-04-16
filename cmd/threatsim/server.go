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
return &cobra.Command{
Use:   "server",
Short: "Start the ThreatSIM API server (Phase 4)",
Long:  `Run the ThreatSIM REST API to serve telemetry, events, and alerts.`,
RunE: func(cmd *cobra.Command, args []string) error {
		color.Cyan("Starting ThreatSIM API Server on :8080...")

		// Initialize the Postgres store
		dsn := os.Getenv("DATABASE_URL")
		if dsn == "" {
			dsn = "host=localhost port=5433 user=threatsim password=password123 dbname=threatsim sslmode=disable"
		}
		pgStore, err := store.NewPostgresStore(dsn)
		if err != nil {
			color.Red("Failed to connect to Postgres. Falling back to InMemoryStore.")
			color.Red("Error: %v", err)
			return err
		}

		color.Cyan("Running DB Schema migrations...")
		if err := pgStore.Migrate("db/migrations"); err != nil {
			color.Red("Migration failed: %v", err)
		} else {
			color.Green("Migrations successful.")
		}

		// Initialize and start the HTTP server using api.Store interface
		var apiStore api.Store = pgStore

		// --- Core Setup: Stream, Detection, Risk, Alerting ---
		stream := memory.NewStream()

		// Save all stream events to our store (Postgres)
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
fmt.Println("Listening on http://localhost:8080")
fmt.Println("- GET /api/v1/simulations")
fmt.Println("- GET /api/v1/events")
fmt.Println("- GET /api/v1/alerts")

// Start the blocking server
if err := server.Start(":8080"); err != nil {
return err
}

return nil
},
}
}
