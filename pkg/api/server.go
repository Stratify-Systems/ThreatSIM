package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/suryatk2007/threatsim/pkg/engine"
	"github.com/suryatk2007/threatsim/pkg/plugins"
	"github.com/suryatk2007/threatsim/pkg/types"
)

type Server struct {
	RootDir string
	runs    map[string]*Run
	mu      sync.RWMutex
}

type RunRequest struct {
	TargetURL string `json:"target_url"`
	Policy    string `json:"policy"`
	Timeout   string `json:"timeout"`
	Insecure  bool   `json:"insecure"`
}

type Run struct {
	ID          string                  `json:"id"`
	Status      string                  `json:"status"`
	TargetURL   string                  `json:"target_url"`
	Policy      string                  `json:"policy"`
	StartedAt   time.Time               `json:"started_at"`
	CompletedAt *time.Time              `json:"completed_at,omitempty"`
	Error       string                  `json:"error,omitempty"`
	Report      *types.ValidationReport `json:"report,omitempty"`
}

func NewServer(rootDir string) *Server {
	return &Server{RootDir: rootDir, runs: make(map[string]*Run)}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/health", s.handleHealth)
	mux.HandleFunc("/api/plugins", s.handlePlugins)
	mux.HandleFunc("/api/policies", s.handlePolicies)
	mux.HandleFunc("/api/runs", s.handleRuns)
	mux.HandleFunc("/api/runs/", s.handleRun)
	return withCORS(mux)
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handlePlugins(w http.ResponseWriter, _ *http.Request) {
	names := make([]string, 0)
	for name := range plugins.List() {
		names = append(names, name)
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"plugins": names})
}

func (s *Server) handlePolicies(w http.ResponseWriter, _ *http.Request) {
	root := filepath.Join(s.RootDir, "tests", "simulations")
	entries, err := os.ReadDir(root)
	if err != nil {
		http.Error(w, fmt.Sprintf("read policies: %v", err), http.StatusInternalServerError)
		return
	}
	policies := make([]string, 0)
	for _, entry := range entries {
		if !entry.IsDir() && (strings.HasSuffix(entry.Name(), ".yaml") || strings.HasSuffix(entry.Name(), ".yml") || strings.HasSuffix(entry.Name(), ".json")) {
			policies = append(policies, filepath.ToSlash(filepath.Join("tests", "simulations", entry.Name())))
		}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"policies": policies})
}

func (s *Server) handleRuns(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		s.mu.RLock()
		runs := make([]*Run, 0, len(s.runs))
		for _, run := range s.runs {
			runs = append(runs, cloneRun(run))
		}
		s.mu.RUnlock()
		writeJSON(w, http.StatusOK, map[string]interface{}{"runs": runs})
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request RunRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "invalid JSON request", http.StatusBadRequest)
		return
	}
	request.TargetURL = strings.TrimSpace(request.TargetURL)
	request.Policy = strings.TrimSpace(request.Policy)
	if request.TargetURL == "" || request.Policy == "" {
		http.Error(w, "target_url and policy are required", http.StatusBadRequest)
		return
	}

	policyPath := request.Policy
	if !filepath.IsAbs(policyPath) {
		policyPath = filepath.Join(s.RootDir, filepath.Clean(policyPath))
	}
	if !isWithinRoot(s.RootDir, policyPath) {
		http.Error(w, "policy must be inside the project directory", http.StatusBadRequest)
		return
	}
	if _, err := os.Stat(policyPath); err != nil {
		http.Error(w, fmt.Sprintf("policy unavailable: %v", err), http.StatusBadRequest)
		return
	}

	timeout := 15 * time.Second
	if request.Timeout != "" {
		parsed, err := time.ParseDuration(request.Timeout)
		if err != nil || parsed <= 0 {
			http.Error(w, "timeout must be a positive duration", http.StatusBadRequest)
			return
		}
		timeout = parsed
	}

	id := strconv.FormatInt(time.Now().UnixNano(), 10)
	run := &Run{ID: id, Status: "running", TargetURL: request.TargetURL, Policy: request.Policy, StartedAt: time.Now()}
	s.mu.Lock()
	s.runs[id] = run
	s.mu.Unlock()

	go s.execute(run, policyPath, timeout, request.Insecure)
	writeJSON(w, http.StatusAccepted, run)
}

func (s *Server) handleRun(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/runs/")
	s.mu.RLock()
	run, ok := s.runs[id]
	if ok {
		run = cloneRun(run)
	}
	s.mu.RUnlock()
	if !ok {
		http.NotFound(w, r)
		return
	}
	writeJSON(w, http.StatusOK, run)
}

func (s *Server) execute(run *Run, policyPath string, timeout time.Duration, insecure bool) {
	eng := engine.New(run.TargetURL, engine.WithTimeout(timeout), engine.WithInsecure(insecure))
	def, err := eng.LoadSimulation(policyPath)
	var report *types.ValidationReport
	if err == nil {
		report = eng.Execute(def)
	}
	now := time.Now()
	s.mu.Lock()
	run.Report = report
	run.CompletedAt = &now
	if err != nil {
		run.Status = "failed"
		run.Error = err.Error()
	} else {
		run.Status = "completed"
	}
	s.mu.Unlock()
}

func cloneRun(run *Run) *Run {
	copy := *run
	if run.Report != nil {
		reportCopy := *run.Report
		reportCopy.Results = append([]types.SimulationResult(nil), run.Report.Results...)
		copy.Report = &reportCopy
	}
	return &copy
}

func isWithinRoot(root, target string) bool {
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return false
	}
	targetAbs, err := filepath.Abs(target)
	if err != nil {
		return false
	}
	relative, err := filepath.Rel(rootAbs, targetAbs)
	return err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(os.PathSeparator))
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, value interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
