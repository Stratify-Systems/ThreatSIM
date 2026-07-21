# 🛡️ ThreatSim

[![Go Version](https://img.shields.io/badge/Go-1.24%2B-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](https://opensource.org/licenses/MIT)

**ThreatSim** is a declarative security validation and fuzzing engine designed for modern CI/CD pipelines. 

Rather than indiscriminately scanning your infrastructure like a traditional vulnerability scanner, ThreatSim acts as a targeted **validation engine**. It treats your application as a black box and runs deterministic, multi-payload security simulations to guarantee your security controls (auth, rate limits, headers, input validation) are actively working as expected.

## ✨ Key Features

- 📜 **Declarative Security (Policy-as-Code):** Write test cases in simple, readable JSON or YAML formats.
- 🔌 **Plugin Architecture:** Execute complex, stateful security attacks (like Bruteforcing) using built-in or custom Go plugins, entirely abstracted into simple YAML config blocks.
- 💥 **Payload Expansion (Fuzzing):** Define an endpoint once and automatically fuzz it using built-in dictionaries (SQLi, XSS) or your own custom payload lists.
- ⚡ **Independent Execution:** Every simulation is executed independently. A failure in one test will not halt the entire suite.
- 📊 **Rich Validation Reports:** Generates human-readable, CI/CD-friendly output summarizing expected vs. actual behavior.
- 🛑 **Pipeline Native:** Fails fast and returns a non-zero exit code if any test fails, acting as a strict gatekeeper in your automated pipelines.
- ⚙️ **Zero-Config Execution:** Uses global `threatsim.yaml` configs to eliminate repetitive CLI arguments.

## 🚀 Quick Start

### 1. Installation

Ensure you have Go 1.24+ installed on your system.

```bash
git clone https://github.com/suryatk2007/threatsim.git
cd threatsim
go build -o threatsim
```

### 2. Configure Your Workspace

Create a `threatsim.yaml` configuration file in your project root to define your target and simulation path:

```yaml
target_url: "https://jsonplaceholder.typicode.com"
file: "simulations/security_tests.yaml"
```

### 3. Write Your Security Simulations

Create your test file (e.g., `simulations/security_tests.yaml`). 

```yaml
version: "1.0"
simulations:
  # Example 1: Payload Fuzzing
  - name: "SQL Injection Fuzzing (URL)"
    payload_type: "sqli" # Automatically expands into multiple requests
    request:
      method: "GET"
      path: "/posts/1?id={{payload}}"
    expected:
      status_code: 400

  # Example 2: Complex Plugin Execution
  - name: "Admin Login Bruteforce Test"
    plugin: "bruteforce" # Hands execution off to the specialized Bruteforce Go Plugin
    config:
      path: "/login"
      username: "admin"
```

### 4. Execute

Run ThreatSim from your terminal. It will automatically load your configuration and execute the simulations.

```bash
./threatsim run
```

## 🗺️ Roadmap

- [x] Basic HTTP execution and status code validation
- [x] Header and Body validations
- [x] Payload Expansion (Fuzzing)
- [x] Extensible Go Plugin Architecture
- [ ] Concurrent execution engine
- [ ] SARIF / JUnit XML reporting output

## 📚 Documentation

For a deep dive into the architecture, design decisions, and how to extend ThreatSim, please read our [Internals Documentation](./internals.md).