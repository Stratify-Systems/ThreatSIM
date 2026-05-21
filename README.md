<p align="center">
  <img src="docs/assets/banner.png" alt="ThreatSIM Banner" width="800" />
</p>

<h1 align="center">ThreatSIM</h1>

<p align="center">
  <strong>Continuous Security Validation & Cyber Attack Simulation Platform</strong>
</p>

<p align="center">
  <a href="#-the-problem">The Problem</a> •
  <a href="#-how-it-works">How It Works</a> •
  <a href="#-quick-start-cli">Quick Start</a> •
  <a href="#-cicd-integration">CI/CD</a> •
  <a href="#-full-platform-dashboard">Dashboard</a>
</p>

---

## 🎯 The Problem

Security teams deploy monitoring tools, detection rules, SIEM systems, and alert pipelines — but **they rarely test whether those systems actually detect attacks**. ThreatSIM fixes this by bringing a "Security as Code" mindset to your development lifecycle.

It safely fakes attacks to prove that your security detection tools (like SIEMs, WAFs, or Datadog) are working correctly before code ever reaches production.

---

## 🧠 How It Works

ThreatSIM is a data pipeline composed of independent modules that work together to simulate and validate security detection:

```
Attack Plugin → Event Stream → Detection Engine → Risk Engine → Pass/Fail
```

1. **Attack Plugins:** Modular components defining the attack (e.g., `Brute Force`, `Port Scan`, `DDoS`).
2. **Detection Engine:** YAML-based rules evaluate events using sliding window thresholds.
3. **Risk Engine:** Accumulates risk scores and assigns threat levels (LOW → CRITICAL).
4. **Validation Gate:** Reports pass/fail based on whether alerts fired — suitable for CI/CD.

### Execution Modes

- **Internal Generation (default):** Generates mock telemetry internally to test YAML-defined detection rules.
- **Active Network Traffic (`--active`):** Sends actual (but safe) malicious network requests to test your external SIEMs and WAFs.

---

## 🚀 Quick Start (CLI)

### Prerequisites

- **Go 1.25+**

### Install & Run

```bash
# Clone the repository
git clone https://github.com/Stratify-Systems/ThreatSIM.git
cd ThreatSIM

# Build the CLI
go build -o threatsim ./cmd/threatsim/

# List available attack plugins
./threatsim list

# Run a quick security validation (the core feature!)
./threatsim validate --plugin brute_force --expect-alert

# Run a brute force simulation with live event output
./threatsim simulate brute_force -d 5s -r 10

# Run a multi-step attack scenario
./threatsim run scenario account_takeover
```

---

## ⚙️ CI/CD Integration

**This is the core purpose of ThreatSIM.** The `validate` command is designed to run inside your CI/CD pipeline as a security gate.

### The `validate` Command

```bash
# Basic validation — exits 0 if alerts fire, exits 1 if not
threatsim validate --plugin brute_force --expect-alert

# JSON output for machine parsing
threatsim validate --plugin brute_force --expect-alert --json

# Validate multiple attack detections
threatsim validate --plugin port_scan --expect-alert
threatsim validate --plugin privilege_escalation --expect-alert

# Custom rules directory
threatsim validate --plugin brute_force --expect-alert --rules ./my-rules/
```

### GitHub Actions Example

```yaml
security-validation:
  name: Security Detection Validation
  runs-on: ubuntu-latest
  steps:
    - uses: actions/checkout@v4
    - uses: actions/setup-go@v5
      with:
        go-version: "1.25"

    - name: Build ThreatSIM
      run: go build -o bin/threatsim ./cmd/threatsim/

    - name: Validate Brute Force Detection
      run: ./bin/threatsim validate --plugin brute_force --expect-alert --json

    - name: Validate Port Scan Detection
      run: ./bin/threatsim validate --plugin port_scan --expect-alert --json
```

### How It Works in Your Pipeline

1. **Deploy to Staging:** Your application and security monitoring rules are deployed.
2. **ThreatSIM Validation:** The CI runner executes `threatsim validate` to test detection.
3. **Gate Decision:**
   - ✅ **PASS** — Alerts fired, detection works → proceed to production.
   - ❌ **FAIL** — No alerts, detection is broken → block deployment.

---

## 📊 Full Platform (Dashboard + API)

ThreatSIM also includes a REST API server with WebSocket live feed:

```bash
# Start the API server (works without Postgres using in-memory store)
./threatsim server

# Or with Docker Compose for the full stack (Postgres, Prometheus, Grafana)
docker-compose up --build
```

### API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/health` | Health check |
| GET | `/api/v1/simulations` | List all simulations |
| POST | `/api/v1/simulations` | Start a new simulation |
| GET | `/api/v1/events` | List generated events |
| GET | `/api/v1/alerts` | List fired alerts |
| WS | `/ws/live` | Live WebSocket feed |
| GET | `/metrics` | Prometheus metrics |

---

## 🔧 Available Plugins

| Plugin ID | Name | What It Tests |
|-----------|------|---------------|
| `brute_force` | Brute Force Login Attack | Failed login monitoring, rate limiting |
| `port_scan` | Port Scanning Attack | Network scan detection, IDS/IPS |
| `ddos` | DDoS HTTP Burst Attack | HTTP flood detection |
| `credential_stuffing` | Credential Stuffing Attack | Automated login detection |
| `privilege_escalation` | Privilege Escalation Attack | Privilege escalation monitoring |

---

## 📁 Project Structure

```
ThreatSIM/
├── cmd/threatsim/          # CLI application (validate, simulate, server, etc.)
├── internal/
│   ├── core/               # Core types (Event, Alert, Plugin, Rule, etc.)
│   ├── plugins/            # Attack simulation plugins
│   ├── detection/          # YAML-based detection engine
│   ├── risk/               # Risk scoring engine
│   ├── streaming/          # Event stream (memory + Redis)
│   ├── alerting/           # Alert dispatcher (webhook, slack, email)
│   ├── api/                # REST API server + WebSocket
│   ├── store/              # Postgres persistence
│   └── scenario/           # Multi-step attack scenarios
├── configs/
│   ├── rules/              # Detection rule YAML files
│   └── scenarios/          # Attack scenario YAML files
├── scripts/                # CI/CD helper scripts
└── .github/workflows/      # GitHub Actions CI pipeline
```

---

## 📜 License

MIT License — See [LICENSE](LICENSE) for details.
