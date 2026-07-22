# ThreatSim Internals & Architecture

ThreatSim is engineered with a strict separation of concerns, decoupling the CLI layer from the core execution engine. This ensures the engine is portable, highly testable, and primed for future expansion.

## Project Structure

```text
threatsim/
├── cmd/             # Cobra CLI layer (flag parsing, command routing)
├── pkg/
│   ├── engine/      # Core business logic (loading, expanding, executing, reporting)
│   ├── plugins/     # Extensible security workflow plugins (e.g., bruteforce)
│   │   └── utils/   # Decoupled utility generators
│   └── types/       # Global data models and schemas
├── simulations/     # User-defined YAML/JSON test files
└── threatsim.yaml   # Global workspace configuration
```

## Architecture Overview

```mermaid
graph TD
    A[CLI / cmd] -->|Reads| B(threatsim.yaml)
    A -->|Passes URL & File| C[Engine]
    C -->|1. Parse| D[SimulationDefinition]
    D -->|2. Route| E{Has Plugin?}
    E -->|Yes| F[Plugin Engine]
    E -->|No| I[HTTP Client]
    I -->|Execute & Validate| J[Validation Logic]
    J -->|Compare| K[Expected Struct]
    K -->|Merge| L
    F -->|Merge| L[ValidationReport]
    L -->|Output| M[CLI stdout]
```

## Core Components

### 1. Data Models (`pkg/types/simulation.go`)
The foundation of ThreatSim is its flexible schema:
- **`Simulation`**: Represents a test scenario. It natively supports standard HTTP validations (via `request` and `expected`), or handing execution off to a complex Go module (via `plugin` and `config`).
- **`Request`**: Defines the HTTP method, path, headers, query parameters, raw string body, and an optional per-request `timeout` override.
- **`Expected`**: The criteria for success. Validations include `status_code`, exact `headers` matching, `body_contains` substrings, and `body_regex` pattern matching.
- **`ValidationReport`**: A rolled-up aggregate of all `SimulationResult` executions.

### 2. Execution Engine (`pkg/engine/engine.go`)
The engine operates entirely independently of `os.Stdout` or CLI contexts, making it an ideal library for external orchestration.

1. **Configuration Loading:** If no CLI flags are supplied, the tool parses a local `threatsim.yaml` (including default `timeout` and `insecure` settings).
2. **Engine Construction & Options:** `New(targetURL, opts...)` initializes an Engine instance using functional options like `WithTimeout(duration)` and `WithInsecure(bool)` for customizing TLS client behavior (`InsecureSkipVerify`).
3. **Parsing:** `LoadSimulation` expands OS environment variables (e.g., `${API_KEY}`) to prevent hardcoding secrets, then unmarshals the YAML/JSON file and ensures structural integrity.
4. **Execution Routing:** The engine loops over the simulation definitions:
   - If a simulation has a `plugin` defined, standard HTTP validation is bypassed, and execution context is handed to the Plugin architecture.
   - Otherwise, standard HTTP round-trips occur using `context.WithTimeout` (evaluating per-request timeout overrides or global defaults) to prevent hanging.
5. **Reporting**: Results are handed to a `Reporter` interface. Any sensitive plugin variables are automatically masked from the final output (Console, JSON, HTML, or PDF) to prevent credential leakage in CI/CD pipelines.

### 3. Plugin Architecture (`pkg/plugins/`)
The plugin system transforms ThreatSim from a declarative HTTP validation tool into a robust, Turing-complete security suite.
- **The Interface**: Any struct implementing `Name() string`, `Description() string`, and `Execute(simName string, ctx Context, config map[string]interface{}) []types.SimulationResult` can be registered as a plugin.
- **The Registry**: Plugins register themselves globally in their `init()` functions.
- **Capabilities**: Because plugins are native Go code, they can implement highly complex, stateful workflows. 
  - **`bruteforce`**: Takes a username and `num_requests`, generates a dictionary using a decoupled utility, and safely iterates against the target endpoint while strictly enforcing safety guardrails (aborting if limits are exceeded to prevent accidental DoS). It asserts both status codes and body content (e.g. soft lockouts).
  - **`idor`**: Automates cross-tenant authorization checks. It authenticates as two distinct users, dynamically parses their tokens and IDs via JSON paths, and performs a cross-tenant resource fetch to validate 403 Forbidden expectations.
  - **`jwt_forge`**: Validates API authentication boundaries. It intercepts a valid token, decodes it, maliciously manipulates the payload (e.g., injecting administrative roles), and attempts to use the forged token to ensure the backend verifies the cryptographic signature.

## Design Philosophy
- **Test-Driven Foundation:** The core Engine logic, YAML unmarshaling, and end-to-end execution flow are rigorously validated by tests in `engine_test.go`.
- **Go (Golang):** Chosen for its concurrency support, robust `net/http` standard library, and ease of distributing cross-platform, single-binary CLI tools.
- **Minimal Dependencies:** By relying almost exclusively on the Go standard library (with the exception of `yaml.v3` and `cobra`), ThreatSim remains incredibly lightweight, secure, and easy to maintain.
- **Extensibility First:** The validation logic checks against an `Expected` struct rather than hardcoded rules, making it trivial to add features like `RegexMatch` or `MaxLatency` in the future.
