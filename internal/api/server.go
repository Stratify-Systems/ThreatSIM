package api

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/Stratify-Systems/ThreatSIM/internal/core"
	"github.com/Stratify-Systems/ThreatSIM/internal/plugins"
	"github.com/Stratify-Systems/ThreatSIM/internal/scenario"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type Server struct {
	router   *chi.Mux
	store    Store
	hub      *Hub
	registry *plugins.Registry
	stream   core.EventStream
}

func NewServer(store Store, registry *plugins.Registry, stream core.EventStream) *Server {
	hub := NewHub()

	// Hook the store's Broadcast emitter to our WebSocket Hub
	store.SetBroadcaster(hub.Broadcast)

	s := &Server{
		router:   chi.NewRouter(),
		store:    store,
		hub:      hub,
		registry: registry,
		stream:   stream,
	}

	go s.hub.Run()
	s.setupRoutes()
	return s
}

func (s *Server) setupRoutes() {
	s.router.Use(middleware.RequestID)
	s.router.Use(middleware.RealIP)
	s.router.Use(middleware.Logger)
	s.router.Use(middleware.Recoverer)

	s.router.Get("/health", s.handleGetHealth)
	s.router.Get("/ws/live", s.handleWebSocket)

	s.router.Route("/api/v1", func(r chi.Router) {
		r.Get("/simulations", s.handleGetSimulations)
		r.Post("/simulations", s.handlePostSimulations)
		r.Post("/scenarios", s.handlePostScenarios)
		r.Get("/alerts", s.handleGetAlerts)
		r.Get("/events", s.handleGetEvents)
	})
}

// Start spawns the HTTP server (blocking)
func (s *Server) Start(addr string) error {
	return http.ListenAndServe(addr, s.router)
}

func (s *Server) handleGetSimulations(w http.ResponseWriter, r *http.Request) {
	data, err := s.store.GetSimulations()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, data)
}

type StartSimulationRequest struct {
	PluginID string `json:"plugin_id"`
	Target   string `json:"target"`
	Duration string `json:"duration,omitempty"`
	Rate     int    `json:"rate,omitempty"`
}

func (s *Server) handlePostSimulations(w http.ResponseWriter, r *http.Request) {
	var req StartSimulationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	plugin, err := s.registry.Get(req.PluginID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	config := plugin.DefaultConfig()
	if req.Target != "" { config.Target = req.Target }
	if req.Duration != "" { config.Duration = req.Duration }
	if req.Rate > 0 { config.Rate = req.Rate }

	simID := "sim-" + req.PluginID + "-" + time.Now().Format("150405")
	s.store.AddSimulation(SimulationState{
		ID: simID, PluginID: req.PluginID, Target: req.Target,
		Status: "RUNNING", StartTime: time.Now(),
	})

	go func() {
		start := time.Now()
			eventsGenerated := 0
			sink := func(event core.Event) error {
				eventsGenerated++
				return s.stream.Publish(context.Background(), core.TopicAttackEvents, event)
			}
			_ = plugin.Execute(context.Background(), config, sink)
			time.Sleep(100 * time.Millisecond) // buffer for events to finish sending
			s.store.CompleteSimulation(simID, eventsGenerated, time.Since(start))
		}()
		w.WriteHeader(http.StatusAccepted)
	writeJSON(w, map[string]string{"id": simID, "status": "started"})
}

func (s *Server) handleGetAlerts(w http.ResponseWriter, r *http.Request) {
	data, err := s.store.GetAlerts()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, data)
}

func (s *Server) handleGetEvents(w http.ResponseWriter, r *http.Request) {
	data, err := s.store.GetEvents()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, data)
}

func (s *Server) handleGetHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]string{"status": "ok"})
}

func writeJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}


type StartScenarioRequest struct {
	ScenarioID string `json:"scenario_id"`
	Target     string `json:"target"`
}

func (s *Server) handlePostScenarios(w http.ResponseWriter, r *http.Request) {
	var req StartScenarioRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	filePath := fmt.Sprintf("configs/scenarios/%s.yaml", req.ScenarioID)
	sc, err := scenario.LoadFromFile(filePath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load scenario: %v", err), http.StatusNotFound)
		return
	}

	// Override target in steps if provided by UI
	if req.Target != "" {
		for i := range sc.Steps {
			sc.Steps[i].Config.Target = req.Target
		}
	}

	simID := "scenario-" + req.ScenarioID + "-" + time.Now().Format("150405")
	s.store.AddSimulation(SimulationState{
		ID: simID, PluginID: req.ScenarioID, Target: req.Target,
		Status: "RUNNING", StartTime: time.Now(),
	})

	go func() {
		start := time.Now()
		eventsGenerated := 0
		sink := func(event core.Event) error {
			eventsGenerated++
			return s.stream.Publish(context.Background(), core.TopicAttackEvents, event)
		}

		engine := scenario.NewEngine(s.registry)
		_ = engine.Run(context.Background(), sc, sink)

		time.Sleep(100 * time.Millisecond) // buffer for events to finish sending
		s.store.CompleteSimulation(simID, eventsGenerated, time.Since(start))
	}()
	w.WriteHeader(http.StatusAccepted)
	writeJSON(w, map[string]string{"id": simID, "status": "started"})
}

// handleWebSocket handles the /ws/live endpoint upgrades and registrations
func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
conn, err := upgrader.Upgrade(w, r, nil)
if err != nil {
http.Error(w, "Failed to upgrade connection", http.StatusInternalServerError)
return
}

client := &Client{
hub:  s.hub,
conn: conn,
send: make(chan interface{}, 256),
}
client.hub.register <- client

// Start the background sender loop
go client.writePump()
}
