# ThreatSim

[![Go Version](https://img.shields.io/badge/Go-1.24%2B-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](https://opensource.org/licenses/MIT)
[![Build Status](https://img.shields.io/badge/build-passing-brightgreen.svg)](https://github.com/suryatk2007/threatsim)

ThreatSim is a **Security Behavior Validation Engine**. 

Rather than indiscriminately scanning your infrastructure like a traditional vulnerability scanner, ThreatSim allows you to deterministically **validate that your application's security controls behave as expected**. It treats your application as a black box and runs declarative simulations to verify that your auth, rate limits, headers, and input validation are actively working.

## Why ThreatSim?

* Traditional scanners search for known vulnerabilities and often produce noisy, non-actionable alerts.
* ThreatSim validates that an application's specific security controls (like RBAC boundaries, authentication gates, and input sanitization) behave exactly as intended.
* ThreatSim is designed to be integrated into CI/CD pipelines as a deterministic security validation gate.

## Key Features

- **Security Behavior Validation:** Define exactly how your endpoint *should* respond to malicious or unauthorized input (e.g., verifying a 401 Unauthorized or 403 Forbidden status).
- **Declarative Security (Policy-as-Code):** Write test cases in simple, readable JSON or YAML formats.
- **Stateful Security Workflows:** Extract dynamic tokens, IDs, or headers from responses and inject them into subsequent API calls to validate complex, multi-step business logic.
- **Payload Expansion:** A convenience feature to automatically expand a single simulation into multiple validation requests using built-in dictionaries.
- **CI/CD Integration:** Fails fast and returns a non-zero exit code if any expected security behavior is violated, acting as a strict gatekeeper.
- **Extensible Architecture:** Advanced security logic can be encapsulated into custom Go plugins, abstracted entirely into simple YAML configs.

## Quick Start

### 1. Installation

```bash
git clone https://github.com/suryatk2007/threatsim.git
cd threatsim
go build -o threatsim
```

### 2. Configuration

Create a `threatsim.yaml` file in your project root to point the engine to your target and simulation files:

```yaml
target_url: "https://api.example.com"
simulations_path: "./simulations"
```

### 3. Write Security Validations

Create validation policies in your `simulations` directory. 

#### Example 1: Security Behavior Validation
Verify that your application correctly enforces authentication and authorization controls.

```yaml
version: "1.0"
simulations:
  - name: "Verify unauthenticated access returns 401"
    request:
      method: "GET"
      path: "/api/secure-data"
    expected:
      status_code: 401

  - name: "Verify normal users cannot access admin endpoints"
    request:
      method: "POST"
      path: "/api/admin/delete"
      headers:
        Authorization: "Bearer user-token"
    expected:
      status_code: 403
```

#### Example 2: Stateful Security Workflows
Validate multi-step logic by extracting data from one response and injecting it into the next.

```yaml
version: "1.0"
simulations:
  - name: "Step 1: Authenticate and Extract JWT"
    request:
      method: "POST"
      path: "/auth/login"
      body: '{"user":"admin", "pass":"secret"}'
    expected:
      status_code: 200
    extract:
      json:
        session_token: "data.token"

  - name: "Step 2: Fetch Secure Profile using Token"
    request:
      method: "GET"
      path: "/profile"
      headers:
        Authorization: "Bearer {{state.session_token}}"
    expected:
      status_code: 200
```

#### Example 3: Advanced Payload Expansion & Plugins
Automatically expand a validation check across multiple inputs, or hand execution off to a plugin for complex logic.

```yaml
version: "1.0"
simulations:
  - name: "Validate SQLi Input Rejection"
    plugin: "sqli"
    config:
      path: "/api/users"
      method: "POST"
      query_params:
        id: "100"
      body:
        username: "admin"
```

### 4. Execute

Run ThreatSim from your terminal. It will automatically load your configuration and execute the validations.

```bash
./threatsim run
```

## Execution Lifecycle

```text
Load Simulation
      ↓
Validate Configuration
      ↓
Expand Payloads (optional)
      ↓
Execute Requests
      ↓
Collect Responses
      ↓
Validate Expectations
      ↓
Generate Report
```

## Roadmap

- [ ] Parallel simulation execution
- [ ] HTML reports
- [ ] JSON reports
- [ ] SARIF/JUnit output
- [ ] Conditional execution
- [ ] Variable improvements
- [ ] OpenAPI import
- [ ] Simulation templates

## Documentation

For a deep dive into the architecture, design decisions, and how to extend ThreatSim, please read our [Internals Documentation](./docs/internals.md).