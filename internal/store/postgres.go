package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Stratify-Systems/ThreatSIM/internal/api"
	"github.com/Stratify-Systems/ThreatSIM/internal/core"
	_ "github.com/lib/pq"
	"github.com/pressly/goose/v3"
)

type PostgresStore struct {
db        *sql.DB
broadcast func(interface{})
}

func NewPostgresStore(dsn string) (*PostgresStore, error) {
db, err := sql.Open("postgres", dsn)
if err != nil {
return nil, fmt.Errorf("failed to open database: %w", err)
}

if err := db.Ping(); err != nil {
return nil, fmt.Errorf("failed to ping database: %w", err)
}

return &PostgresStore{db: db}, nil
}

func (s *PostgresStore) Close() error {
return s.db.Close()
}

func (s *PostgresStore) Migrate(dir string) error {
// Set dialect for postgres
if err := goose.SetDialect("postgres"); err != nil {
return err
}
return goose.Up(s.db, dir)
}

func (s *PostgresStore) SetBroadcaster(b func(interface{})) {
s.broadcast = b
}

func (s *PostgresStore) AddEvent(event core.Event) error {
query := `INSERT INTO events (id, event_type, source_ip, target, service, timestamp, plugin_id) VALUES ($1, $2, $3, $4, $5, $6, $7)`
_, err := s.db.Exec(query, event.ID, event.Type, event.SourceIP, event.Target, event.Service, event.Timestamp, event.PluginID)
if err != nil {
return err
}

if s.broadcast != nil {
s.broadcast(map[string]interface{}{"type": "event", "data": event})
}
return nil
}

func (s *PostgresStore) GetEvents() ([]core.Event, error) {
query := `SELECT id, event_type, source_ip, target, service, timestamp, plugin_id FROM events ORDER BY timestamp DESC LIMIT 1000`
rows, err := s.db.Query(query)
if err != nil {
return nil, err
}
defer rows.Close()

var events []core.Event
for rows.Next() {
var e core.Event
var source, target, service string
if err := rows.Scan(&e.ID, &e.Type, &source, &target, &service, &e.Timestamp, &e.PluginID); err != nil {
return nil, err
}
e.SourceIP = source
e.Target = target
e.Service = service
events = append(events, e)
}
return events, nil
}

func (s *PostgresStore) AddAlert(score core.RiskScore) error {
factorsJSON, err := json.Marshal(score.Factors)
if err != nil {
return err
}

query := `INSERT INTO alerts (source_ip, score, threat_level, factors, updated_at) VALUES ($1, $2, $3, $4, $5)`
	_, err = s.db.Exec(query, score.SourceIP, score.Score, score.ThreatLevel, factorsJSON, score.UpdatedAt)
if err != nil {
return err
}

if s.broadcast != nil {
s.broadcast(map[string]interface{}{"type": "alert", "data": score})
}
return nil
}

func (s *PostgresStore) GetAlerts() ([]core.RiskScore, error) {
query := `SELECT source_ip, score, threat_level, factors, updated_at FROM alerts ORDER BY updated_at DESC LIMIT 1000`
rows, err := s.db.Query(query)
if err != nil {
return nil, err
}
defer rows.Close()

var alerts []core.RiskScore
for rows.Next() {
var a core.RiskScore
var factorsJSON []byte
if err := rows.Scan(&a.SourceIP, &a.Score, &a.ThreatLevel, &factorsJSON, &a.UpdatedAt); err != nil {
return nil, err
}
if err := json.Unmarshal(factorsJSON, &a.Factors); err != nil {
return nil, err
}
alerts = append(alerts, a)
}
return alerts, nil
}

func (s *PostgresStore) AddSimulation(sim api.SimulationState) error {
query := `INSERT INTO simulations (id, plugin_id, target, events_num, status, start_time, duration) VALUES ($1, $2, $3, $4, $5, $6, $7)`
var dur sql.NullString
if sim.Duration != "" {
dur.String = sim.Duration
dur.Valid = true
}
_, err := s.db.Exec(query, sim.ID, sim.PluginID, sim.Target, sim.EventsNum, sim.Status, sim.StartTime, dur)
if err != nil {
return err
}

if s.broadcast != nil {
s.broadcast(map[string]interface{}{"type": "simulation_started", "data": sim})
}
return nil
}

func (s *PostgresStore) CompleteSimulation(id string, totalEvents int, elapsed time.Duration) error {
query := `UPDATE simulations SET status = $1, events_num = $2, duration = $3 WHERE id = $4`
_, err := s.db.Exec(query, "COMPLETED", totalEvents, elapsed.String(), id)
if err != nil {
return err
}

// Just send what we know
if s.broadcast != nil {
updates := map[string]interface{}{
"id": id,
"status": "COMPLETED",
"events_num": totalEvents,
"duration": elapsed.String(),
}
s.broadcast(map[string]interface{}{"type": "simulation_completed", "data": updates})
}
return nil
}

func (s *PostgresStore) GetSimulations() ([]api.SimulationState, error) {
query := `SELECT id, plugin_id, target, events_num, status, start_time, duration FROM simulations ORDER BY start_time DESC`
rows, err := s.db.Query(query)
if err != nil { // Handle empty
return nil, err
}
defer rows.Close()

var sims []api.SimulationState
for rows.Next() {
var sim api.SimulationState
var dur sql.NullString

if err := rows.Scan(&sim.ID, &sim.PluginID, &sim.Target, &sim.EventsNum, &sim.Status, &sim.StartTime, &dur); err != nil {
return nil, err
}
if dur.Valid {
sim.Duration = dur.String
}
sims = append(sims, sim)
}
return sims, nil
}
