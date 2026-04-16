package api

import (
	"github.com/Stratify-Systems/ThreatSIM/internal/core"
	"time"
)

// Store defines the state interface matching our newly created package.
type Store interface {
	AddSimulation(sim SimulationState) error
	CompleteSimulation(id string, totalEvents int, elapsed time.Duration) error
	GetSimulations() ([]SimulationState, error)
	AddEvent(event core.Event) error
	GetEvents() ([]core.Event, error)
	AddAlert(score core.RiskScore) error
	GetAlerts() ([]core.RiskScore, error)
	SetBroadcaster(b func(interface{}))
}
