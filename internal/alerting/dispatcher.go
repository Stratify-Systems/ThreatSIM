package alerting

import (
	"log"
	"sync"

	"github.com/Stratify-Systems/ThreatSIM/internal/core"
)

// Dispatcher manages multiple notifiers and routes alerts to them based on thresholds.
type Dispatcher struct {
	notifiers []Notifier
	stateMap  map[string]core.ThreatLevel // key: source_ip
	mu        sync.Mutex
}

func NewDispatcher() *Dispatcher {
	return &Dispatcher{
		notifiers: make([]Notifier, 0),
		stateMap:  make(map[string]core.ThreatLevel),
	}
}

func (d *Dispatcher) Register(n Notifier) {
	d.notifiers = append(d.notifiers, n)
}

// levelWeight converts a ThreatLevel to an integer for easy comparison.
func levelWeight(level core.ThreatLevel) int {
	switch level {
	case core.ThreatCritical:
		return 4
	case core.ThreatHigh:
		return 3
	case core.ThreatMedium:
		return 2
	case core.ThreatLow:
		return 1
	default:
		return 0
	}
}

// Dispatch evaluates the risk score and sends alerts based on state transitions.
// It suppresses duplicate alerts for the same source_ip unless the threat level escalates.
func (d *Dispatcher) Dispatch(score core.RiskScore) {
	d.mu.Lock()

	key := score.SourceIP
	lastLevel, exists := d.stateMap[key]
	shouldSend := false

	if !exists {
		// First time detection for the source_ip
		shouldSend = true
	} else if levelWeight(score.ThreatLevel) > levelWeight(lastLevel) {
		// Threat level increases
		shouldSend = true
	}

	if shouldSend {
		d.stateMap[key] = score.ThreatLevel
	}
	d.mu.Unlock()

	// If suppressed, exit early
	if !shouldSend {
		return
	}

	for _, n := range d.notifiers {
		go func(notifier Notifier, sc core.RiskScore) {
			if err := notifier.Send(sc); err != nil {
				log.Printf("⚠️ [Alerting] Failed to dispatch alert: %v\n", err)
			}
		}(n, score)
	}
}
