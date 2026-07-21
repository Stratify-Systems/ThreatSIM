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
- **Complex Stateful Plugins:** Advanced security logic, like testing IDOR boundaries across tenants or rapidly iterating through bruteforce dictionaries, is encapsulated into custom Go plugins and abstracted entirely into simple YAML configs.
- **CI/CD Integration:** Fails fast and returns a non-zero exit code if any expected security behavior is violated. Supports machine-readable and human-readable reporting (`--output=json`, `--output=html`, `--output=pdf`).

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

#### Example 2: Environment Variables & Regex Validation
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

#### Example 3: Plugin Execution (IDOR)
Use plugins for complex, multi-step stateful workflows like tenant isolation validation.

```yaml
version: "1.0"
simulations:
  - name: "Cross-Tenant IDOR Validation"
    plugin: "idor"
    config:
      auth_path: "/auth/login"
      user_a_payload: '{"username":"admin", "password":"secret123"}'
      user_b_payload: '{"username":"guest", "password":"password123"}'
      token_json_path: "data.token"
      id_json_path: "data.user.id"
      target_path: "/api/users/{id}/private-data"
      expected_status_code: 403
```

#### Example 4: Plugin Guardrails (Bruteforce)
Plugins include built-in safety guardrails and support for both status code and body matching.

```yaml
version: "1.0"
simulations:
  - name: "Admin Login Bruteforce Test"
    plugin: "bruteforce"
    config:
      path: "/login"
      username: "admin"
      num_requests: 100
      expected_status_code: 429
      expected_body_contains: "locked"
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

### 5. Testing with the Built-in Mock Server

ThreatSim includes an intentionally vulnerable mock API to test complex plugin scenarios like IDOR.

1. Start the mock server in a new terminal:
   ```bash
   go run cmd/mockserver/main.go
   ```
2. Run the IDOR simulation against it to see ThreatSim catch the vulnerability:
   ```bash
   ./threatsim run -t http://localhost:8080 -f simulations/idor_test.yaml
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