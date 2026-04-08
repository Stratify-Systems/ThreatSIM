import os

with open("internal/api/server.go", "r") as f:
    content = f.read()

if '"github.com/Stratify-Systems/ThreatSIM/internal/scenario"' in content:
    print("Already applied!")
    exit(0)

# 1. Add 'fmt' and 'scenario' imports
content = content.replace('"github.com/Stratify-Systems/ThreatSIM/internal/plugins"', '"github.com/Stratify-Systems/ThreatSIM/internal/plugins"\n\t"github.com/Stratify-Systems/ThreatSIM/internal/scenario"\n\t"fmt"', 1)

# 2. Add /scenarios route
content = content.replace('r.Post("/simulations", s.handlePostSimulations)', 'r.Post("/simulations", s.handlePostSimulations)\n\t\tr.Post("/scenarios", s.handlePostScenarios)', 1)

# 3. Add handlePostScenarios function at the bottom before handleWebSocket
scenario_func = """
type StartScenarioRequest struct {
\tScenarioID string `json:"scenario_id"`
\tTarget     string `json:"target"`
}

func (s *Server) handlePostScenarios(w http.ResponseWriter, r *http.Request) {
\tvar req StartScenarioRequest
\tif err := json.NewDecoder(r.Body).Decode(&req); err != nil {
\t\thttp.Error(w, err.Error(), http.StatusBadRequest)
\t\treturn
\t}

\tfilePath := fmt.Sprintf("configs/scenarios/%s.yaml", req.ScenarioID)
\tsc, err := scenario.LoadFromFile(filePath)
\tif err != nil {
\t\thttp.Error(w, fmt.Sprintf("Failed to load scenario: %v", err), http.StatusNotFound)
\t\treturn
\t}

\t// Override target in steps if provided by UI
\tif req.Target != "" {
\t\tfor i := range sc.Steps {
\t\t\tsc.Steps[i].Config.Target = req.Target
\t\t}
\t}

\tsimID := "scenario-" + req.ScenarioID + "-" + time.Now().Format("150405")
\ts.store.AddSimulation(SimulationState{
\t\tID: simID, PluginID: req.ScenarioID, Target: req.Target,
\t\tStatus: "RUNNING", StartTime: time.Now(),
\t})

\tgo func() {
\t\tstart := time.Now()
\t\teventsGenerated := 0
\t\tsink := func(event core.Event) error {
\t\t\teventsGenerated++
\t\t\treturn s.stream.Publish(context.Background(), core.TopicAttackEvents, event)
\t\t}

\t\tengine := scenario.NewEngine(s.registry)
\t\t_ = engine.Run(context.Background(), sc, sink)

\t\ttime.Sleep(100 * time.Millisecond) // buffer for events to finish sending
\t\ts.store.CompleteSimulation(simID, eventsGenerated, time.Since(start))
\t}()
\tw.WriteHeader(http.StatusAccepted)
\twriteJSON(w, map[string]string{"id": simID, "status": "started"})
}

// handleWebSocket"""

content = content.replace('// handleWebSocket', scenario_func, 1)

with open("internal/api/server.go", "w") as f:
    f.write(content)

print("Applied backend patch for scenarios!")
