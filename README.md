<p align="center">
  <img src="docs/assets/banner.png" alt="ThreatSIM Banner" width="800" />
</p>

<h1 align="center">ThreatSIM</h1>

<p align="center">
  <strong>Open-source cyber attack simulation platform to test your security detection systems.</strong>
</p>

<p align="center">
  <a href="#-quick-start">Quick Start</a> •
  <a href="#-the-problem">The Problem</a> •
  <a href="#-cicd-integration">CI/CD</a> •
  <a href="#-architecture">Architecture</a> •
  <a href="#-attack-plugins">Plugins</a> •
  <a href="#-detection-engine">Detection</a> •
  <a href="#%EF%B8%8F-cli-reference">CLI</a> •
  <a href="#-roadmap">Roadmap</a> •
  <a href="#-contributing">Contributing</a>
</p>

<p align="center">
  <img src="https://img.shields.io/badge/version-0.1.0-blue?style=flat-square" alt="Version" />
  <img src="https://img.shields.io/badge/go-1.24+-00ADD8?style=flat-square&logo=go" alt="Go" />
  <img src="https://img.shields.io/badge/license-MIT-green?style=flat-square" alt="License" />
  <img src="https://img.shields.io/badge/status-active_development-orange?style=flat-square" alt="Status" />
</p>

---

## 🎯 The Problem

Security teams deploy monitoring tools, detection rules, SIEM systems, and alert pipelines — but **they rarely test whether those systems actually detect attacks**.

```
Security team creates a brute force detection rule
                    ↓
        But they never test it
                    ↓
         Real attack happens
                    ↓
          Detection fails ❌
```

**ThreatSIM fixes this.** It simulates attacks safely, sends them through your security pipeline, and lets you verify:

- ✅ Does my detection rule work?
- ✅ Does my alert trigger?
- ✅ How fast does the system respond?
- ✅ Are there blind spots in my coverage?

---

## ⚙️ CI/CD Integration

ThreatSIM is designed for "Security as Code." You can run it directly in your CI/CD pipeline (e.g., GitHub Actions, GitLab CI) immediately after deploying to a staging environment to automatically test if your newly deployed detection rules are working.

If the staging environment correctly detects the simulated ThreatSIM attack, the pipeline passes. If it fails to detect it, the pipeline fails, preventing you from deploying inadequate security monitoring to production.

📖 **Read more about the CI/CD workflow:** [CI/CD Pipeline Integration Design](docs/CI_CD_PIPELINE.md)

---

## 🚀 Quick Start

### Prerequisites

- **Go 1.24+** — [Install Go](https://go.dev/dl/)
- **Docker** (optional) — for Redis & full-stack deployment

### Install & Run

```bash
# Clone the repository
git clone https://github.com/Stratify-Systems/ThreatSIM.git
cd ThreatSIM

# Build the CLI
go build -o threatsim ./cmd/threatsim/

# List available attack plugins
./threatsim list

# Run a brute force simulation
./threatsim simulate brute_force

# Run a port scan with custom settings
./threatsim simulate port_scan --target 10.0.0.1 --duration 10s --rate 20
```

### Example Output

```
  ⚔  Brute Force Login Attack
  ─────────────────────────────────────
  Plugin:    brute_force
  Target:    10.0.0.1
  Service:   auth-service
  Source IP: 10.1.2.3
  Duration:  5s
  Rate:      3 events/sec
  ─────────────────────────────────────

▶ Simulation started
  [20:36:13.144] login_failed    │ 10.1.2.3 → 10.0.0.1 │ user=admin
  [20:36:13.478] login_failed    │ 10.1.2.3 → 10.0.0.1 │ user=root
  [20:36:13.811] login_failed    │ 10.1.2.3 → 10.0.0.1 │ user=deploy
  [20:36:14.144] login_failed    │ 10.1.2.3 → 10.0.0.1 │ user=test
  ...

  ─────────────────────────────────────
  ✓ Simulation Complete
  Plugin:     Brute Force Login Attack
  Events:     15 events generated
  Duration:   5.001s
  Throughput: 3.0 events/sec
  ─────────────────────────────────────
```

---

## 🏗 Architecture

ThreatSIM is built as a pipeline of independent components:

```
Attack Plugins
      ↓
Attack Scenario Engine
      ↓
Event Streaming System (Redis Streams / Kafka)
      ↓
Detection Engine (YAML rule-based)
      ↓
Risk Scoring Engine
      ↓
Alert System (Slack / Email / Webhook)
      ↓
Dashboard & Observability (React + Prometheus + Grafana)
```

### System Flow Diagram

```
┌─────────────────┐     ┌──────────────────┐     ┌─────────────────┐
│  Attack Plugins  │────▶│  Event Streaming │────▶│Detection Engine │
│                  │     │                  │     │                 │
│ • Brute Force    │     │ • Redis Streams  │     │ • YAML Rules    │
│ • Port Scan      │     │ • Kafka          │     │ • Window-based  │
│ • DDoS           │     │ • In-Memory      │     │ • Threshold     │
│ • Cred Stuffing  │     └──────────────────┘     └────────┬────────┘
│ • Priv Escalation│                                       │
└─────────────────┘                                        ▼
                                                 ┌─────────────────┐
┌─────────────────┐     ┌──────────────────┐     │  Risk Scoring   │
│    Dashboard     │◀────│  Alert System    │◀────│    Engine       │
│                  │     │                  │     │                 │
│ • Live Timeline  │     │ • Slack          │     │ • Per-IP Score  │
│ • Threat Score   │     │ • Email          │     │ • Threat Levels │
│ • Alert Feed     │     │ • Webhook        │     │ • Accumulation  │
│ • Top Attackers  │     │ • Dashboard Push │     └─────────────────┘
└─────────────────┘     └──────────────────┘
```

### Tech Stack

| Layer | Technology | Rationale |
|-------|-----------|-----------|
| Language | **Go** | Concurrent, fast, single binary, standard in security tooling |
| CLI | **Cobra** | Industry-standard Go CLI framework |
| Streaming | **Redis Streams** / Kafka | Redis for local dev, Kafka for production scale |
| Detection | **Custom Go Engine** | Lightweight, YAML-configurable rules |
| Database | **PostgreSQL** | Reliable, JSON column support |
| Dashboard | **Vite + React** | Fast, modern, WebSocket-powered |
| Metrics | **Prometheus + Grafana** | Standard observability stack |
| Deployment | **Docker Compose** | `docker compose up` runs everything |

---

## 🔌 Attack Plugins

ThreatSIM uses a plugin architecture — each attack type is an independent module.

### Available Plugins

| Plugin | ID | Event Type | Description |
|--------|----|-----------|-------------|
| **Brute Force** | `brute_force` | `login_failed` | Rapid failed login attempts against a service |
| **Port Scan** | `port_scan` | `port_probe` | Network port scanning to discover open services |
| **DDoS** | `ddos` | `http_flood` | High-volume HTTP request burst | 
| **Credential Stuffing** | `credential_stuffing` | `login_attempt` | Automated login with stolen credential lists |
| **Privilege Escalation** | `privilege_escalation` | `priv_escalation` | Attempt to gain higher system privileges |

> **Status:** ✅ Brute Force & Port Scan are implemented. Others are coming in Phase 3.

### Creating a Plugin

Every plugin implements the `Plugin` interface:

```go
type Plugin interface {
    ID() string
    Name() string
    Description() string
    DefaultConfig() PluginConfig
    Execute(ctx context.Context, config PluginConfig, sink EventSink) error
}
```

Example — a minimal attack plugin:

```go
package my_attack

import (
    "context"
    "github.com/Stratify-Systems/ThreatSIM/internal/core"
)

type Plugin struct{}

func (p *Plugin) ID() string          { return "my_attack" }
func (p *Plugin) Name() string        { return "My Custom Attack" }
func (p *Plugin) Description() string { return "Simulates a custom attack" }

func (p *Plugin) DefaultConfig() core.PluginConfig {
    return core.PluginConfig{
        Target:   "10.0.0.1",
        Duration: "30s",
        Rate:     10,
    }
}

func (p *Plugin) Execute(ctx context.Context, config core.PluginConfig, sink core.EventSink) error {
    // Generate your attack events here
    event := core.Event{
        Type:     "custom_event",
        SourceIP: config.SourceIP,
        Target:   config.Target,
    }
    return sink(event)
}
```

Then register it in `cmd/threatsim/main.go`:

```go
registry.Register(&myattack.Plugin{})
```

### Event Format

Every plugin produces events in this standardized format:

```json
{
  "id": "bf-1741538173-001",
  "event_type": "login_failed",
  "source_ip": "10.1.2.3",
  "target": "10.0.0.1",
  "service": "auth-service",
  "user": "admin",
  "timestamp": "2026-03-09T19:00:00Z",
  "plugin_id": "brute_force",
  "metadata": {
    "method": "password",
    "attempt": 1,
    "response_code": 401
  }
}
```

---

## 🔗 Attack Scenario Engine

Real attackers execute **attack chains**, not single attacks. ThreatSIM models this:

```yaml
# configs/scenarios/account_takeover.yaml
scenario:
  name: account_takeover
  description: "Simulates a full account takeover attack chain"

  steps:
    - plugin: port_scan
      config:
        target: "10.0.0.1"
        ports: "1-1024"
      delay: 5s

    - plugin: credential_stuffing
      config:
        target_service: auth-service
        userlist: ["admin", "root", "user1"]
      delay: 2s

    - plugin: privilege_escalation
      config:
        target_user: admin
```

Run a scenario:

```bash
./threatsim run scenario account_takeover
```

> **Status:** Coming in Phase 3.

---

## 🔍 Detection Engine

The detection engine analyzes the event stream and detects suspicious activity using **YAML-defined rules** — no code changes required.

### Detection Rules

```yaml
# configs/rules/brute_force.yaml
rules:
  - name: brute_force_attack
    description: "Detects brute force login attempts"
    condition:
      event_type: login_failed
      group_by: source_ip
      threshold: 20
      window: 30s
    severity: high
    risk_score: 60
```

**Translation:** If 20+ `login_failed` events from the same IP occur within 30 seconds → trigger a HIGH severity alert.

### More Rule Examples

```yaml
# Port Scan Detection
- name: port_scan_detected
  condition:
    event_type: port_probe
    group_by: source_ip
    threshold: 50
    window: 60s
  severity: medium
  risk_score: 30

# DDoS Detection
- name: ddos_burst
  condition:
    event_type: http_flood
    group_by: source_ip
    threshold: 1000
    window: 10s
  severity: critical
  risk_score: 90
```

> **Status:** Coming in Phase 2.

---

## 📊 Risk Scoring Engine

Each detected attack contributes to a cumulative risk score per source IP:

| Attack Type | Base Score |
|------------|-----------|
| Port Scan | 30 |
| Brute Force | 60 |
| Credential Stuffing | 70 |
| Privilege Escalation | 85 |
| DDoS | 90 |

**Threat Levels:**

| Score Range | Threat Level |
|------------|-------------|
| 0 — 30 | 🟢 LOW |
| 31 — 60 | 🟡 MEDIUM |
| 61 — 80 | 🟠 HIGH |
| 81 — 100 | 🔴 CRITICAL |

Example:

```
Port scan detected (30) + Brute force detected (60)
= Risk Score: 90
= Threat Level: 🔴 CRITICAL
```

> **Status:** Coming in Phase 2.

---

## 🚨 Alert System

When high-risk attacks are detected, ThreatSIM sends alerts through configured channels:

```
┌──────────────────────────────────────────┐
│         🚨 CRITICAL ALERT                │
│                                          │
│  Brute Force Attack Detected             │
│  Source IP:       10.1.2.3               │
│  Target Service:  auth-service           │
│  Events:          47 failed logins       │
│  Window:          30 seconds             │
│  Risk Score:      90 (CRITICAL)          │
└──────────────────────────────────────────┘
```

**Supported channels:**
- 💬 Slack (webhook)
- 📧 Email (SMTP)
- 🌐 Webhook (HTTP POST)
- 📊 Dashboard (WebSocket push)

> **Status:** Coming in Phase 4.

---

## 📈 Dashboard & Observability

Real-time dashboard showing:

| Widget | Description |
|--------|-------------|
| **Active Attacks** | Live feed of running simulations |
| **Threat Score** | Current system-wide threat level gauge |
| **Attack Timeline** | Chronological event visualization |
| **Top Attacker IPs** | Leaderboard of most active source IPs |
| **Detection Alerts** | Real-time alert stream |

Built with **React + WebSocket** for live updates. Metrics exposed via **Prometheus** and visualized in **Grafana**.

Example Prometheus metrics:
```
simulated_attacks_total{plugin="brute_force"} 150
alerts_triggered_total{severity="high"} 12
detection_latency_seconds{rule="brute_force_attack"} 0.023
```

> **Status:** Coming in Phase 5.

---

## ⌨️ CLI Reference

```bash
# Show help
threatsim --help

# List available attack plugins  
threatsim list

# Simulate an attack
threatsim simulate <plugin> [flags]

# Flags for simulate:
#   --target string      Target IP or hostname
#   --service string     Target service name
#   --source-ip string   Source IP to simulate from
#   -d, --duration string  How long to run (e.g., 30s, 5m)
#   -r, --rate int         Events per second
#   --redis               Use Redis Streams

# Run an attack scenario (coming soon)
threatsim run scenario <name>

# Check system status (coming soon)
threatsim status

# Print version
threatsim version
```

### Examples

```bash
# Quick brute force test (5 seconds, 3 events/sec)
./threatsim simulate brute_force -d 5s -r 3

# Port scan a specific target
./threatsim simulate port_scan --target 192.168.1.100 --rate 50

# High-intensity brute force from a specific IP
./threatsim simulate brute_force --source-ip 172.16.0.99 -d 2m -r 20
```

---

## 🗂 Project Structure

```
ThreatSIM/
├── cmd/threatsim/               # CLI entry point
│   ├── main.go                  # Main + plugin registration
│   ├── simulate.go              # "simulate" command
│   └── list.go                  # "list" command
├── internal/
│   ├── core/                    # Domain types & interfaces
│   │   ├── event.go             # Event, Alert, RiskScore
│   │   ├── plugin.go            # Plugin interface + config
│   │   ├── rule.go              # Detection rule types
│   │   ├── scenario.go          # Scenario types
│   │   └── stream.go            # EventStream interface
│   ├── plugins/                 # Attack plugin implementations
│   │   ├── registry.go          # Plugin registry
│   │   ├── brute_force/         # Brute force login attack
│   │   └── port_scan/           # Port scanning attack
│   ├── streaming/               # Event streaming layer
│   │   ├── memory/              # In-memory (development)
│   │   └── redis/               # Redis Streams (production)
│   ├── detection/               # Detection engine (Phase 2)
│   ├── risk/                    # Risk scoring (Phase 2)
│   ├── alerting/                # Alert system (Phase 4)
│   ├── api/                     # REST API + WebSocket (Phase 4)
│   └── store/                   # Database layer (Phase 4)
├── configs/
│   ├── rules/                   # Detection rule YAML files
│   └── scenarios/               # Attack scenario YAML files
├── dashboard/                   # React frontend (Phase 5)
├── deploy/                      # Docker + Kubernetes configs
├── docs/                        # Documentation
│   ├── ARCHITECTURE.md          # Architecture overview
│   ├── CI_CD_PIPELINE.md        # CI/CD integration design flow
│   └── PROJECT_CONTEXT.md       # Core context for AI LLM tools
├── go.mod
├── go.sum
├── Makefile
├── LICENSE
└── README.md
```

---

## 🗺 Roadmap

### Phase 1: Foundation ✅
- [x] Core domain types (Event, Alert, Rule, Scenario)
- [x] Plugin interface & registry
- [x] Brute Force attack plugin
- [x] Port Scan attack plugin
- [x] In-memory event streaming
- [x] Redis Streams event streaming
- [x] CLI tool (`simulate`, `list`, `version`)
- [x] Colored terminal output with live event feed

### Phase 2: Detection + Risk Engine 🔄
- [ ] Detection engine with YAML rule loading
- [ ] Sliding window event evaluator
- [ ] Risk scoring engine with threat levels
- [ ] Detection rules for brute force & port scan
- [ ] Wire detection to event stream consumer
- [ ] Alert generation from detections

### Phase 3: Scenarios + More Plugins
- [ ] Scenario engine with YAML loader
- [ ] `threatsim run scenario <name>` command
- [ ] DDoS burst attack plugin
- [ ] Credential stuffing attack plugin
- [ ] Privilege escalation attack plugin
- [ ] Sample scenarios (account_takeover, lateral_movement)

### Phase 4: Alert System + API
- [ ] Alert manager with channel interface
- [ ] Slack notification channel
- [ ] Email notification channel
- [ ] Webhook notification channel
- [ ] REST API for dashboard data
- [ ] WebSocket endpoint for live updates
- [ ] PostgreSQL storage layer
- [ ] Database migrations

### Phase 5: Dashboard
- [ ] Vite + React project setup
- [ ] Attack timeline visualization
- [ ] Threat score gauge widget
- [ ] Real-time alert feed
- [ ] Top attacker IPs leaderboard
- [ ] WebSocket integration for live data
- [ ] Dark mode responsive design

### Phase 6: Observability + Deployment
- [ ] Prometheus metrics endpoint
- [ ] Grafana dashboard templates
- [ ] Docker + Docker Compose setup
- [ ] Kubernetes manifests
- [ ] CI/CD pipeline
- [ ] Comprehensive documentation

---

## 🛠 Development

### Build

```bash
go build -o threatsim ./cmd/threatsim/
```

### Run Tests

```bash
go test ./...
```

### Run with Redis (optional)

```bash
# Start Redis
docker run -d --name redis -p 6379:6379 redis:alpine

# Run with Redis streaming
./threatsim simulate brute_force --redis
```

### Full Stack (coming soon)

```bash
docker compose up
```

---

## 🤝 Contributing

ThreatSIM is open source and contributions are welcome!

### How to Contribute

1. **Fork** the repository
2. **Create** a feature branch (`git checkout -b feature/ddos-plugin`)
3. **Implement** your changes
4. **Test** thoroughly (`go test ./...`)
5. **Submit** a pull request

### Ideas for Contribution

- 🔌 **New attack plugins** — SQL injection, XSS, DNS tunneling, etc.
- 📋 **New detection rules** — More YAML rule definitions
- 🎨 **Dashboard components** — New widgets and visualizations
- 📖 **Documentation** — Tutorials, guides, examples
- 🐛 **Bug fixes** — Found a bug? Fix it!

---

## 📄 License

This project is licensed under the MIT License — see the [LICENSE](LICENSE) file for details.

---

<p align="center">
  <strong>Built by <a href="https://github.com/Stratify-Systems">Stratify Systems</a></strong>
</p>
<p align="center">
  <em>Making security testing accessible to everyone.</em>
</p>
