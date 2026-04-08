package main

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/Stratify-Systems/ThreatSIM/internal/api"
	"github.com/Stratify-Systems/ThreatSIM/internal/store"
)

func newServerCmd() *cobra.Command {
return &cobra.Command{
Use:   "server",
Short: "Start the ThreatSIM API server (Phase 4)",
Long:  `Run the ThreatSIM REST API to serve telemetry, events, and alerts.`,
RunE: func(cmd *cobra.Command, args []string) error {
		color.Cyan("Starting ThreatSIM API Server on :8080...")

		// Initialize the Postgres store
		dsn := "host=localhost port=5433 user=threatsim password=password123 dbname=threatsim sslmode=disable"
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
		
		populateMockData(apiStore) // MOCK DATA TO TEST API IS WORKING

		// Initialize and start the HTTP server
		server := api.NewServer(apiStore)
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
