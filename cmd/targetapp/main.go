package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

// ══════════════════════════════════════════════════════════════
// Target Application with Built-in Security Monitoring
// ══════════════════════════════════════════════════════════════
//
// This simulates a realistic staging application that:
//   1. Has a login endpoint (POST /login)
//   2. Monitors failed login attempts (security middleware)
//   3. Detects brute force attacks when threshold is exceeded
//   4. Exposes a security API for external tools to query
//
// ThreatSIM attacks this app, then queries /security/alerts
// to verify the app's OWN security system caught the attack.
//
// Run:
//   go run ./cmd/targetapp
//   go run ./cmd/targetapp --port 9999
// ══════════════════════════════════════════════════════════════

// --- Security Event Types ---

type SecurityEvent struct {
	Type      string    `json:"type"`
	SourceIP  string    `json:"source_ip"`
	Username  string    `json:"username,omitempty"`
	Path      string    `json:"path"`
	Timestamp time.Time `json:"timestamp"`
}

type SecurityAlert struct {
	Type       string    `json:"type"`
	SourceIP   string    `json:"source_ip"`
	EventCount int       `json:"event_count"`
	DetectedAt time.Time `json:"detected_at"`
	Message    string    `json:"message"`
}

type AlertsResponse struct {
	Alerts      []SecurityAlert `json:"alerts"`
	TotalEvents int             `json:"total_events"`
	MonitoredAt time.Time       `json:"monitored_at"`
}

// --- Security Monitor ---

// SecurityMonitor watches for suspicious patterns in application traffic.
// This is the kind of security middleware that a real app would have.
type SecurityMonitor struct {
	events       []SecurityEvent
	alerts       []SecurityAlert
	failedLogins map[string][]time.Time // IP → timestamps of failed logins
	mu           sync.RWMutex

	// Detection thresholds
	bruteForceThreshold int
	bruteForceWindow    time.Duration
}

func NewSecurityMonitor() *SecurityMonitor {
	return &SecurityMonitor{
		events:              make([]SecurityEvent, 0),
		alerts:              make([]SecurityAlert, 0),
		failedLogins:        make(map[string][]time.Time),
		bruteForceThreshold: 10,           // 10 failed logins
		bruteForceWindow:    30 * time.Second, // within 30 seconds
	}
}

// RecordFailedLogin records a failed login and checks for brute force patterns
func (sm *SecurityMonitor) RecordFailedLogin(sourceIP, username, path string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	now := time.Now()

	// Record the event
	event := SecurityEvent{
		Type:      "login_failed",
		SourceIP:  sourceIP,
		Username:  username,
		Path:      path,
		Timestamp: now,
	}
	sm.events = append(sm.events, event)

	// Track for brute force detection
	sm.failedLogins[sourceIP] = append(sm.failedLogins[sourceIP], now)

	// Prune old entries outside the window
	cutoff := now.Add(-sm.bruteForceWindow)
	var recent []time.Time
	for _, t := range sm.failedLogins[sourceIP] {
		if t.After(cutoff) {
			recent = append(recent, t)
		}
	}
	sm.failedLogins[sourceIP] = recent

	// Check threshold
	if len(recent) == sm.bruteForceThreshold {
		alert := SecurityAlert{
			Type:       "brute_force_detected",
			SourceIP:   sourceIP,
			EventCount: len(recent),
			DetectedAt: now,
			Message:    fmt.Sprintf("Brute force attack detected: %d failed logins from %s in %v", len(recent), sourceIP, sm.bruteForceWindow),
		}
		sm.alerts = append(sm.alerts, alert)

		// Print alert to terminal with visual emphasis
		fmt.Println()
		fmt.Println("  ┌──────────────────────────────────────────────────┐")
		fmt.Println("  │ 🚨 SECURITY ALERT: Brute Force Detected         │")
		fmt.Printf("  │    Source IP:   %-33s │\n", sourceIP)
		fmt.Printf("  │    Attempts:    %-33d │\n", len(recent))
		fmt.Printf("  │    Window:      %-33s │\n", sm.bruteForceWindow.String())
		fmt.Println("  │    Action:      IP flagged for investigation      │")
		fmt.Println("  └──────────────────────────────────────────────────┘")
		fmt.Println()
	}

	// Cap stored events
	if len(sm.events) > 50000 {
		sm.events = sm.events[10000:]
	}
}

func (sm *SecurityMonitor) GetAlerts() AlertsResponse {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	alerts := make([]SecurityAlert, len(sm.alerts))
	copy(alerts, sm.alerts)

	return AlertsResponse{
		Alerts:      alerts,
		TotalEvents: len(sm.events),
		MonitoredAt: time.Now(),
	}
}

func (sm *SecurityMonitor) GetEvents() []SecurityEvent {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	events := make([]SecurityEvent, len(sm.events))
	copy(events, sm.events)
	return events
}

func (sm *SecurityMonitor) Reset() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.events = make([]SecurityEvent, 0)
	sm.alerts = make([]SecurityAlert, 0)
	sm.failedLogins = make(map[string][]time.Time)
}

// --- HTTP Handlers ---

func main() {
	port := "9999"
	if len(os.Args) > 1 {
		for i, arg := range os.Args {
			if (arg == "--port" || arg == "-p") && i+1 < len(os.Args) {
				port = os.Args[i+1]
			}
		}
	}

	monitor := NewSecurityMonitor()
	mux := http.NewServeMux()

	// ── Application Endpoints ──────────────────────

	// POST /login - Simulated login endpoint
	mux.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		// Parse the login request
		var loginReq struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		json.NewDecoder(r.Body).Decode(&loginReq)

		// Extract source IP
		sourceIP := r.RemoteAddr
		if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
			sourceIP = strings.Split(fwd, ",")[0]
		}
		// Strip port from IP
		if idx := strings.LastIndex(sourceIP, ":"); idx != -1 {
			sourceIP = sourceIP[:idx]
		}

		username := loginReq.Username
		if username == "" {
			username = "unknown"
		}

		// Log the attempt
		ts := time.Now().Format("15:04:05.000")
		fmt.Printf("  [%s] POST /login │ IP: %-15s │ user: %-10s │ 401 Unauthorized\n", ts, sourceIP, username)

		// Record in security monitor (this is where detection happens)
		monitor.RecordFailedLogin(sourceIP, username, "/login")

		// Respond with 401
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{
			"error":   "invalid_credentials",
			"message": "Invalid username or password",
		})
	})

	// ── Security API Endpoints ─────────────────────

	// GET /security/alerts - Query detected security alerts
	mux.HandleFunc("/security/alerts", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(monitor.GetAlerts())
	})

	// GET /security/events - Query raw security event log
	mux.HandleFunc("/security/events", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		events := monitor.GetEvents()
		json.NewEncoder(w).Encode(map[string]interface{}{
			"events": events,
			"total":  len(events),
		})
	})

	// POST /security/reset - Clear all security state (for clean test runs)
	mux.HandleFunc("/security/reset", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		monitor.Reset()
		fmt.Println("  [RESET] Security state cleared.")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "reset"})
	})

	// GET /health - Health check
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	// ── Start Server ───────────────────────────────

	fmt.Println()
	fmt.Println("  ╔══════════════════════════════════════════════════╗")
	fmt.Println("  ║   🎯  Target Application (Staging Server)       ║")
	fmt.Println("  ╚══════════════════════════════════════════════════╝")
	fmt.Println()
	fmt.Printf("  Listening on http://localhost:%s\n", port)
	fmt.Println("  ──────────────────────────────────────────────────")
	fmt.Println("  App Endpoints:")
	fmt.Printf("    POST http://localhost:%s/login\n", port)
	fmt.Println("  Security API:")
	fmt.Printf("    GET  http://localhost:%s/security/alerts\n", port)
	fmt.Printf("    GET  http://localhost:%s/security/events\n", port)
	fmt.Printf("    POST http://localhost:%s/security/reset\n", port)
	fmt.Println("  ──────────────────────────────────────────────────")
	fmt.Println("  Security Monitor: Brute force detection enabled")
	fmt.Println("    Threshold: 10 failed logins / 30s window")
	fmt.Println("  ──────────────────────────────────────────────────")
	fmt.Println()
	fmt.Println("  Waiting for traffic...")
	fmt.Println()

	if err := http.ListenAndServe(":"+port, mux); err != nil {
		fmt.Printf("Server failed: %v\n", err)
		os.Exit(1)
	}
}
