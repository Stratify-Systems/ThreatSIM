# ThreatSim

ThreatSim validates an application's security behavior by executing predefined security simulations against a running application and verifying that the application's response matches the expected behavior. 

It is **not** a vulnerability scanner. Its core purpose is to guarantee that the security mechanisms you have built are functioning correctly by treating the application as a black box and observing its responses.

## Features

- **Declarative Simulations:** Write test cases in simple JSON or YAML formats.
- **Independent Execution:** Every simulation is executed independently so that a failure in one test does not stop the suite.
- **Comprehensive Validation Reports:** Generates a human-readable, CI/CD-friendly validation report.
- **Fail-Fast for Pipelines:** Returns a non-zero exit code if any test fails, making it perfect for automated pipeline gates.
- **Payload Expansion (Fuzzing):** Automatically expand a single test into multiple requests using built-in security dictionaries (e.g. SQLi, XSS) or custom payloads.
- **Extensible Foundation:** Built on a clean Go architecture designed for future validation expansions.

## Installation

Ensure you have Go 1.24+ installed.

```bash
git clone https://github.com/suryatk2007/threatsim.git
cd threatsim
go build -o threatsim
```

## Quick Start

1. Create a `threatsim.yaml` configuration file in your project root to avoid passing CLI flags:
```yaml
target_url: "https://jsonplaceholder.typicode.com"
file: "simulations/security_tests.yaml"
```

2. Create a `simulations/security_tests.yaml` file to define your tests. You can inject payloads dynamically using `{{payload}}`:

```yaml
version: "1.0"
simulations:
  - name: "SQLi Injection in URL"
    payload_type: "sqli" # Automatically tests predefined SQL injection payloads
    request:
      method: "GET"
      path: "/posts/1?id={{payload}}"
    expected:
      status_code: 400
  - name: "Verify Secure API Response"
    request:
      method: "POST"
      path: "/secure-login"
      headers:
        "Authorization": "Bearer invalid_token"
    expected:
      status_code: 401
      headers:
        "Content-Type": "application/json"
        "X-Content-Type-Options": "nosniff"
      body_contains: "invalid or expired token"
```

3. Run ThreatSim:

```bash
./threatsim run
```

## Documentation

For a deep dive into the architecture and internal design of ThreatSim, please read [internals.md](./internals.md).