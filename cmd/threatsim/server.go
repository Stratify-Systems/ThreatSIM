package main

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/Stratify-Systems/ThreatSIM/internal/api"
)

func newServerCmd() *cobra.Command {
return &cobra.Command{
Use:   "server",
Short: "Start the ThreatSIM API server (Phase 4)",
Long:  `Run the ThreatSIM REST API to serve telemetry, events, and alerts.`,
RunE: func(cmd *cobra.Command, args []string) error {
color.Cyan("Starting ThreatSIM API Server on :8080...")

// Initialize the in-memory store
store := api.NewInMemoryStore()
populateMockData(store) // MOCK DATA TO TEST API IS WORKING

// Initialize and start the HTTP server
server := api.NewServer(store)

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
