package api

import (
	"sync"
	"time"

	"github.com/Stratify-Systems/ThreatSIM/internal/core"
)

// InMemoryStore holds temporary state to power the REST API during Phase 4 Step 2.
// It will eventually be replaced by the internal/store interface interacting with standard PG databases.
type InMemoryStore struct {
	simulations []SimulationState
	alerts      []core.RiskScore
	events      []core.Event
	mu          sync.RWMutex
}

// SimulationState represents the high-level metrics of an executed sequence.
type SimulationState struct {
	ID        string    `json:"id"`
	PluginID  string    `json:"plugin_id"`
	Target    string    `json:"target"`
	EventsNum int       `json:"events_num"`
	Status    string    `json:"status"`
	StartTime time.Time `json:"start_time"`
	Duration  string    `json:"duration,omitempty"` // For post-mortem parsing
}

func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{
		simulations: make([]SimulationState, 0),
		alerts:      make([]core.RiskScore, 0),
		events:      make([]core.Event, 0),
	}
}

// --- Tracking Setters ---

// AddEvent adds a raw plugin-generated security event to the tracker
func (s *InMemoryStore) AddEvent(event core.Event) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, event)

	// Keep size bounded in memory
	if len(s.events) > 10000 {
		s.events = s.events[1000:]
	}
}

// AddAlert adds an alert struct pushed from risk thresholds
func (s *InMemoryStore) AddAlert(score core.RiskScore) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.alerts = append(s.alerts, score)

	if len(s.alerts) > 1000 {
		s.alerts = s.alerts[100:]
	}
}

// AddSimulation registers an active simulation runtime container
func (s *InMemoryStore) AddSimulation(sim SimulationState) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.simulations = append(s.simulations, sim)
}

// CompleteSimulation updates an existing simulation status when context is cancelled or expires
func (s *InMemoryStore) CompleteSimulation(id string, totalEvents int, elapsed time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, sim := range s.simulations {
		if sim.ID == id {
			s.simulations[i].Status = "COMPLETED"
			s.simulations[i].EventsNum = totalEvents
			s.simulations[i].Duration = elapsed.String()
			return
		}
	}
}

// --- Tracking Getters ---

func (s *InMemoryStore) GetEvents() []core.Event {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return copyEvents(s.events)
}

func (s *InMemoryStore) GetAlerts() []core.RiskScore {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return copyAlerts(s.alerts)
}

func (s *InMemoryStore) GetSimulations() []SimulationState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return copySimulations(s.simulations)
}

// Shallow array copiers avoiding slice-leak iteration panics outside mutex range
func copyEvents(in []core.Event) []core.Event {
	out := make([]core.Event, len(in))
	copy(out, in)
	return out
}

func copyAlerts(in []core.RiskScore) []core.RiskScore {
	out := make([]core.RiskScore, len(in))
	copy(out, in)
	return out
}

func copySimulations(in []SimulationState) []SimulationState {
	out := make([]SimulationState, len(in))
	copy(out, in)
	return out
}
