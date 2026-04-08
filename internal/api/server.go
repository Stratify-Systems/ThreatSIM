package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type Server struct {
	router *chi.Mux
	store  *InMemoryStore
	hub    *Hub
}

func NewServer(store *InMemoryStore) *Server {
	hub := NewHub()
	
	// Hook the store's Broadcast emitter to our WebSocket Hub
	store.SetBroadcaster(hub.Broadcast)

	s := &Server{
		router: chi.NewRouter(),
		store:  store,
		hub:    hub,
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
		r.Get("/alerts", s.handleGetAlerts)
		r.Get("/events", s.handleGetEvents)
	})
}

// Start spawns the HTTP server (blocking)
func (s *Server) Start(addr string) error {
	return http.ListenAndServe(addr, s.router)
}

func (s *Server) handleGetSimulations(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, s.store.GetSimulations())
}

func (s *Server) handleGetAlerts(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, s.store.GetAlerts())
}

func (s *Server) handleGetEvents(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, s.store.GetEvents())
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
