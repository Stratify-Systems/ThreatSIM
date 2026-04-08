package store

import (
"time"

"github.com/Stratify-Systems/ThreatSIM/internal/api"
"github.com/Stratify-Systems/ThreatSIM/internal/core"
)

// Store defines the persistent storage layer for the API to retrieve state.
type Store interface {
// Simulations
AddSimulation(sim api.SimulationState) error
CompleteSimulation(id string, totalEvents int, elapsed time.Duration) error
GetSimulations() ([]api.SimulationState, error)

// Events
AddEvent(event core.Event) error
GetEvents() ([]core.Event, error)

// Alerts
AddAlert(score core.RiskScore) error
GetAlerts() ([]core.RiskScore, error)

// Set Broadcaster for realtime websocket connections
SetBroadcaster(b func(interface{}))
}
