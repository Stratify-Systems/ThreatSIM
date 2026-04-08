package alerting

import "github.com/Stratify-Systems/ThreatSIM/internal/core"

// Notifier defines the interface for an alert notification channel.
type Notifier interface {
// Send dispatches a risk score alert to an external system.
Send(score core.RiskScore) error
}
