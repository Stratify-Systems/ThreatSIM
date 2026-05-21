# ThreatSIM — Complete Commands Reference

All commands to build, test, and run ThreatSIM.

---

## Prerequisites

- **Go 1.25+** (only requirement)

---

## Build

```bash
# Build everything (CLI + target app)
make build

# Or manually:
go build -o bin/threatsim ./cmd/threatsim/
go build -o bin/targetapp ./cmd/targetapp/
```

---

## List Available Plugins

```bash
./bin/threatsim list
```

Shows all 5 attack plugins:
- `brute_force` — Rapid failed login attempts
- `port_scan` — Network port scanning
- `ddos` — HTTP flood attack
- `credential_stuffing` — Automated logins with stolen creds
- `privilege_escalation` — Privilege escalation attempts

---

## Internal Validation (Tests Own Detection Rules)

Runs the full pipeline in-memory: Plugin → Stream → Detection → Risk → Pass/Fail.
No external dependencies required.

```bash
# Validate brute force detection (default)
./bin/threatsim validate --plugin brute_force --expect-alert

# Validate port scan detection
./bin/threatsim validate --plugin port_scan --expect-alert

# Validate privilege escalation detection
./bin/threatsim validate --plugin privilege_escalation --expect-alert

# Validate all at once
make validate-all

# JSON output (for CI/CD pipelines)
./bin/threatsim validate --plugin brute_force --expect-alert --json

# Custom options
./bin/threatsim validate \
    --plugin brute_force \
    --rate 20 \
    --duration 3s \
    --expect-alert
```

**Exit codes:**
- `0` — PASS (alerts fired as expected)
- `1` — FAIL (no alerts, detection is broken)

---

## External Validation (Attacks a Real Target)

This is the core feature — sends REAL HTTP traffic to a target app,
then queries the target's security API to verify it detected the attack.

### Step 1: Start the target app

```bash
# In Terminal 1:
./bin/targetapp --port 9999
```

The target app starts with:
- `POST /login` — Login endpoint (returns 401)
- Built-in brute force detection (10+ failures in 30s)
- `GET /security/alerts` — API to query detected alerts

### Step 2: Run external validation

```bash
# In Terminal 2:
./bin/threatsim validate \
    --plugin brute_force \
    --target http://localhost:9999/login \
    --verify http://localhost:9999/security/alerts
```

### Step 2 (alternative): JSON output for CI/CD

```bash
./bin/threatsim validate \
    --plugin brute_force \
    --target http://localhost:9999/login \
    --verify http://localhost:9999/security/alerts \
    --json
```

### Shortcut with Make

```bash
# Terminal 1:
make target

# Terminal 2:
make validate-external
```

---

## Run a Simulation (Live Event Output)

Watch an attack happen in real-time with per-event output.

```bash
# Brute force with 5 events/sec for 5 seconds
./bin/threatsim simulate brute_force -d 5s -r 5

# Port scan with 10 events/sec for 10 seconds
./bin/threatsim simulate port_scan -d 10s -r 10

# DDoS simulation
./bin/threatsim simulate ddos -d 3s -r 20

# With active mode (sends real HTTP traffic)
./bin/threatsim simulate brute_force \
    --target http://localhost:9999/login \
    --active -d 5s -r 10

# Custom source IP and target
./bin/threatsim simulate brute_force \
    --target 192.168.1.100 \
    --source-ip 10.0.0.50 \
    --service ssh-server \
    -d 10s -r 3
```

---

## Run a Multi-Step Scenario

Execute chained attack sequences from YAML scenario files.

```bash
# Run the account takeover scenario
./bin/threatsim run scenario account_takeover

# Run the network recon scenario
./bin/threatsim run scenario network_recon
```

---

## Start the API Server

```bash
# Start (works without Postgres — falls back to in-memory store)
./bin/threatsim server

# With custom port
./bin/threatsim server --addr :9090

# With Postgres (if available)
DATABASE_URL="host=localhost port=5433 user=threatsim password=password123 dbname=threatsim sslmode=disable" \
    ./bin/threatsim server
```

### API Endpoints (when server is running)

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/health` | Health check |
| `GET` | `/api/v1/simulations` | List all simulations |
| `POST` | `/api/v1/simulations` | Start a new simulation |
| `GET` | `/api/v1/events` | List generated events |
| `GET` | `/api/v1/alerts` | List fired alerts |
| `WS` | `/ws/live` | Live WebSocket event feed |
| `GET` | `/metrics` | Prometheus metrics |

### Trigger simulation via API

```bash
curl -X POST http://localhost:8080/api/v1/simulations \
    -H "Content-Type: application/json" \
    -d '{"plugin_id": "brute_force", "target": "10.0.0.1", "rate": 10, "duration": "5s"}'
```

### Query alerts via API

```bash
curl http://localhost:8080/api/v1/alerts
```

---

## Run Tests

```bash
# Run all tests
go test ./internal/...

# Run with verbose output
go test -v ./internal/...

# Run specific test
go test -v -run TestFullPipelineEndToEnd ./internal/detection/

# Run with coverage
make test-cover
```

---

## Docker

```bash
# Start full stack (Postgres + Backend + Target App)
docker compose up --build

# Stop
docker compose down
```

---

## Quick Reference (Make targets)

```bash
make help             # Show all available commands

# Core Validation
make validate         # Internal validation (brute force)
make validate-all     # Validate all attack types
make validate-json    # JSON output for CI/CD
make target           # Start target app (port 9999)
make validate-external # External validation (needs 'make target')

# Development
make build            # Build all binaries
make test             # Run tests
make test-cover       # Tests with coverage report
make fmt              # Format code
make clean            # Remove build artifacts

# Simulation
make list             # List plugins
make simulate         # Quick brute force simulation
make server           # Start API server

# Docker
make docker-up        # Start Docker stack
make docker-down      # Stop Docker stack
```

---

## Project Structure (After Cleanup)

```
ThreatSIM/
├── cmd/
│   ├── threatsim/              # CLI application
│   │   ├── main.go             # Entry point, plugin registry
│   │   ├── validate.go         # Security validation gate (core feature)
│   │   ├── simulate.go         # Live attack simulation
│   │   ├── server.go           # REST API server
│   │   ├── run.go              # Scenario runner
│   │   ├── list.go             # Plugin lister
│   │   └── version.go          # Version info
│   └── targetapp/
│       └── main.go             # Staging target app (for external validation)
├── internal/
│   ├── core/                   # Types: Event, Alert, Plugin, Rule, RiskScore
│   ├── plugins/                # Attack plugins (brute_force, port_scan, etc.)
│   ├── detection/              # YAML-based detection engine
│   ├── risk/                   # Risk scoring engine
│   ├── streaming/memory/       # In-memory event pub/sub
│   ├── alerting/               # Alert dispatcher (webhook, slack)
│   ├── api/                    # REST API server + WebSocket
│   ├── store/                  # Postgres persistence
│   └── scenario/               # Multi-step scenario engine
├── configs/
│   ├── rules/detection.yaml    # Detection rules (YAML)
│   └── scenarios/              # Attack scenario definitions
├── db/migrations/              # Database migration SQL
├── docs/
│   ├── DEMO_GUIDE.md           # Non-technical demo walkthrough
│   └── COMMANDS.md             # This file
├── scripts/
│   └── ci-validation.sh        # CI/CD helper script
├── .github/workflows/ci.yml    # GitHub Actions pipeline
├── Dockerfile                  # Backend Docker image
├── docker-compose.yml          # Full stack composition
├── Makefile                    # Build automation
└── README.md                   # Project overview
```
