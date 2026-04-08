package main

import (
"math/rand"
"time"

"github.com/Stratify-Systems/ThreatSIM/internal/api"
"github.com/Stratify-Systems/ThreatSIM/internal/core"
)

func populateMockData(store api.Store) {
// Add mock simulation
store.AddSimulation(api.SimulationState{
ID:        "sim-1234",
PluginID:  "brute_force",
Target:    "10.0.0.1",
EventsNum: 0,
Status:    "RUNNING",
StartTime: time.Now(),
})

// Start a background spamer to simulate live traffic for WebSockets
go func() {
for {
time.Sleep(3 * time.Second)

store.AddEvent(core.Event{
ID:        "evt-live",
Type:      "login_failed",
SourceIP: "192.168.1.100",
				Target: "10.0.0.1",
				Service: "ssh",
Timestamp: time.Now(),
PluginID:  "brute_force",
})

// Randomly escalate an alert
if rand.Intn(10) > 7 {
store.AddAlert(core.RiskScore{
SourceIP: "192.168.1.100",
Score:    85,
ThreatLevel:    core.ThreatCritical,
Factors:  []string{"brute_force_attack"},
UpdatedAt:   time.Now(),
})
}
}
}()
}
