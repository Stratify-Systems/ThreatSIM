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

- **Security Behavior Validation:** Define exactly how your endpoint *should* respond to malicious or unauthorized input (e.g., verifying a 401 Unauthorized or matching an error message with a Regex pattern).
- **Declarative Security (Policy-as-Code):** Write test cases in simple, readable JSON or YAML formats. Supports `$ENV_VAR` expansion to avoid hardcoding secrets.
- **Stateful Security Workflows:** Extract dynamic tokens, IDs, or headers from responses and inject them into subsequent API calls to validate complex, multi-step business logic. Extracted secrets are automatically masked in CI logs.
- **CI/CD Integration:** Fails fast and returns a non-zero exit code if any expected security behavior is violated. Supports machine-readable and human-readable reporting (`--output=json`, `--output=html`, `--output=pdf`).
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



#### Example 3: Environment Variables & Regex Validation
Use system environment variables safely and assert strict schema matching using `body_regex`.

```yaml
version: "1.0"
simulations:
  - name: "Verify API Key is enforced and returns correct JSON error"
    request:
      method: "GET"
      path: "/api/billing"
      headers:
        Authorization: "Bearer ${TEST_API_KEY}"
    expected:
      status_code: 403
      body_regex: '"error_code":\s*"AUTH_001"'
```

#### Example 4: Plugin Execution (Bruteforce)
Use extensible plugins for complex workflows with built-in safety guardrails (like `num_requests`).

```yaml
version: "1.0"
simulations:
  - name: "Admin Login Bruteforce Test"
    plugin: "bruteforce"
    config:
      path: "/login"
      username: "admin"
      num_requests: 100
```

### 4. Execute

Run ThreatSim from your terminal. It will automatically load your configuration and execute the validations.

```bash
./threatsim run
```

Or run it in a CI/CD pipeline and output machine-readable JSON:

```bash
./threatsim run --output json --out-file report.json
```

Generate rich, shareable audits using HTML or PDF outputs:

```bash
./threatsim run --output html --out-file dashboard.html
./threatsim run --output pdf --out-file validation-audit.pdf
```

## Execution Lifecycle

```text
Load Simulation
      ↓
Validate Configuration
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
- [x] HTML reports
- [x] JSON reports
- [ ] SARIF/JUnit output
- [ ] Conditional execution
- [ ] Variable improvements
- [ ] OpenAPI import
- [ ] Simulation templates

## Documentation

For a deep dive into the architecture, design decisions, and how to extend ThreatSim, please read our [Internals Documentation](./docs/internals.md).