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
  <a href="#-ci-cd-integration">CI/CD</a> •
  <a href="#-full-platform-dashboard">Dashboard</a>
</p>

---

## 🎯 The Problem

Security teams deploy monitoring tools, detection rules, SIEM systems, and alert pipelines — but **they rarely test whether those systems actually detect attacks**. ThreatSIM fixes this by bringing a "Security as Code" mindset to your development lifecycle.

It safely fakes attacks to prove that your security detection tools (like SIEMs, WAFs, or Datadog) are working correctly before code ever reaches production.

---

## 🧠 How It Works

ThreatSIM is a massive data pipeline composed of independent modules that work together to simulate and detect cyber attacks:

1. **Attack Plugins:** Modular components defining the attack (e.g., `Brute Force`, `Port Scan`, `DDoS`).
2. **Scenario Engine:** Chains plugins together to perform complex, multi-step attacks.
3. **Execution Modes:**
   - **Internal Generation:** Generates mock telemetry logic completely internally to test its own built-in YAML-defined detection engines.
   - **Active Network Traffic (`--active`):** Sends actual (but safe) malicious network requests (e.g. hundreds of incorrect `POST /login` requests) over the network to test your external SIEMs and WAFs.
4. **Dashboard:** A full-stack React frontend showing a live feed of attacks, events, and risk scores.

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

# Run an active brute force simulation against a local endpoint
./threatsim simulate brute_force --target http://localhost:8080/login --active --rate 10 --duration 5s
```

---

## 📊 Full Platform Dashboard

ThreatSIM isn't just a CLI script; it includes a full React dashboard, a Go API Backend, PostgreSQL state storage, and Prometheus/Grafana observability.

To boot the entire universe and see attacks visualized in real-time:

```bash
# Ensure Docker is running
docker-compose up --build
```

Once booted, open `http://localhost:5173` in your web browser. When you run an attack in your terminal, the web UI will instantly light up with live event timelines and risk scoring!

---

## ⚙️ CI/CD Integration

The core goal of ThreatSIM is to run automatically inside your Deployment Pipeline (e.g., GitHub Actions, GitLab CI) to test your real application servers every time code is deployed.

1. **Deploy to Staging:** Both the application and the new security monitoring rules are spun up in a staging environment.
2. **ThreatSIM Execution:** The CI runner executes the ThreatSIM CLI targeting the newly deployed staging environment:
   ```bash
   threatsim simulate brute_force --target http://staging.api.internal/login --active
   ```
3. **Validation:** The CI/CD script queries your security backend (e.g., Datadog, Splunk) asking, _"Did anyone trigger an alarm in the last 10 seconds?"_
   - **Success:** If an alarm fired, your detection works! Deployment continues to production.
   - **Failure:** If no alarm fired, your new code broke security logging. The pipeline fails immediately, blocking deployment.

_(Check out `scripts/ci-validation.sh` for an example of a CI/CD validator script)._
