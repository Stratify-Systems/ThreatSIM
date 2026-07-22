# ThreatSim Architecture & Internals Documentation

ThreatSim is engineered with a strict separation of concerns, decoupling the CLI layer from the core execution engine and plugin framework. This ensures the engine is portable, highly testable, thread-safe, and easily extensible.

---

## 1. Project Directory Structure

```text
threatsim/
├── cmd/                        # Cobra CLI layer (flag parsing, command routing)
│   └── run.go                  # Main "run" CLI command definition
├── docs/                       # Developer and architecture documentation
│   └── internals.md            # Deep dive technical architecture documentation
├── examples/                   # Built-in test mock servers
│   ├── mockserver/             # Vulnerable mock server (Port 8080)
│   └── secure_mockserver/      # Secure mock server (Port 8081)
├── pkg/
│   ├── engine/                 # Core engine (parser, executor, concurrency, reporter)
│   │   ├── engine.go           # Engine construction, options, parallel execution
│   │   ├── options.go          # Functional options (WithTimeout, WithInsecure)
│   │   └── report/             # Output formatters (Console, JSON, HTML, PDF)
│   ├── plugins/                # Plugin registry & security workflow implementations
│   │   ├── plugin.go           # Plugin interface & global registry
│   │   ├── bruteforce.go       # Bruteforce rate-limiting plugin
│   │   ├── idor.go             # IDOR cross-tenant isolation plugin
│   │   ├── jwt_forge.go        # JWT signature forgery plugin
│   │   └── utils/              # Modular attack utilities
│   │       ├── auth/           # Shared auth & JSON path extraction helper
│   │       │   └── auth_session.go
│   │       ├── bruteforce/     # Password dictionary generators
│   │       │   └── bruteforce_gen.go
│   │       ├── idor/           # Cross-tenant IDOR attack runner
│   │       │   └── idor_runner.go
│   │       └── jwt/            # JWT attack modes & runner
│   │           ├── alg_none.go          # Mode 2: "alg": "none" header bypass
│   │           ├── jwt_runner.go        # JWT attack runner & parallel token tester
│   │           ├── signature_tamper.go  # Mode 1: Claim alteration
│   │           └── weak_secret.go       # Mode 3: HMAC-SHA256 re-signing
│   └── types/                  # Global data structures & domain models
│       └── simulation.go       # Simulation, Request, Expected, Result models
├── schemas/                    # Authoritative YAML schema definitions
│   ├── simulation.yaml         # General simulation schema reference
│   └── plugins/                # Plugin-specific config schema templates
│       ├── bruteforce.yaml
│       ├── idor.yaml
│       └── jwt_forge.yaml
├── tests/                      # Centralized test workspace
│   ├── unit/                   # Go unit tests (engine, auth, jwt tests)
│   │   ├── auth_session_test.go
│   │   ├── engine_test.go
│   │   └── jwt_test.go
│   └── simulations/            # Integration YAML test scenarios
│       ├── bruteforce.yaml
│       ├── idor_test.yaml
│       ├── jwt_test.yaml
│       ├── parallel_test.yaml
│       └── timeout_test.yaml
├── threatsim.yaml              # Workspace configuration file
└── main.go                     # Application entrance point
```

---

## 2. Core Architectural Components

### A. Data Models (`pkg/types/simulation.go`)
ThreatSim relies on strongly typed Go models:
- **`Simulation`**: Represents an individual security validation test. Supports standard HTTP validations (`request` and `expected`) or delegation to Go security modules (`plugin` and `config`).
- **`Request`**: Defines HTTP method, path, headers, query parameters, string payload, and optional per-request `timeout` overrides.
- **`Expected`**: Defines pass/fail criteria (`status_code`, `headers`, `body_contains`, `body_regex`).
- **`SimulationResult`**: Details the outcome of an execution (passed/failed status, duration, URL, reason).
- **`ValidationReport`**: Rolled-up aggregate report containing total simulation counts, pass rates, and detailed execution results.

### B. Execution Engine & Options (`pkg/engine/engine.go`)
The execution engine operates independently of CLI input and stdout:
1. **Configuration Loading**: Parses global defaults from `threatsim.yaml` or CLI arguments.
2. **Functional Options Pattern**: Engine instances are built using `engine.New(targetURL, opts...)`:
   - `WithTimeout(duration)`: Configures global HTTP client timeout.
   - `WithInsecure(bool)`: Controls TLS certificate verification (`InsecureSkipVerify`).
3. **Environment Variable Expansion**: `os.ExpandEnv` expands variables (e.g., `${API_KEY}`) prior to YAML unmarshaling.
4. **Execution Dispatch**: Simulations are processed in parallel goroutines (`go func(sim types.Simulation)`).

### C. Concurrency & Parallel Execution Engine
ThreatSim achieves exceptional speed by parallelizing workloads across multiple levels:
- **Simulation-Level Parallelism**: `Engine.Execute()` executes every simulation entry defined in a policy file concurrently using a `sync.WaitGroup` and mutex-protected result aggregation.
- **Plugin-Level Sub-Request Concurrency**:
  - `idor`: User A and User B authentications run concurrently via parallel goroutines.
  - `jwt_forge`: Candidate forged tokens (e.g., across multiple weak HMAC secrets) are evaluated concurrently across parallel worker routines.
  - `bruteforce`: Evaluates passwords across 5 parallel worker goroutines.
- **Unit Test Parallelism**: All Go unit tests in `tests/unit/` call `t.Parallel()`, running concurrently when invoked via `go test ./...`.

---

## 3. Plugin Framework & Attack Utilities (`pkg/plugins/`)

### A. Plugin Interface & Self-Registration
Plugins implement the `Plugin` interface:
```go
type Plugin interface {
    Name() string
    Description() string
    Execute(simName string, ctx Context, config map[string]interface{}) []types.SimulationResult
}
```
Plugins register themselves globally in package `init()` functions via `plugins.Register(&PluginStruct{})`.

### B. Modular Utility Packages (`pkg/plugins/utils/`)
- **`auth/auth_session.go`**: Centralized authentication helper (`auth.AuthenticateAndExtract`). Sends authentication POST requests and extracts tokens or resource IDs using dot-notation JSON paths (`ExtractJSONPath`).
- **`bruteforce/bruteforce_gen.go`**: Generates password dictionaries for brute-force rate-limiting tests.
- **`idor/idor_runner.go`**: Cross-tenant IDOR attack runner supporting multi-method (`GET`, `POST`, `PUT`, `DELETE`, `PATCH`) and `{id}` template substitution in path URLs and JSON payloads.
- **`jwt/`**: Modular JWT attack suite:
  - `signature_tamper.go`: Alters payload claims while keeping original signature.
  - `alg_none.go`: Sets header `"alg": "none"` and strips signature (`header.payload.`).
  - `weak_secret.go`: Alters payload claims and re-signs using HMAC-SHA256 with common weak keys (`secret`, `123456`, `password`, `admin`, `key`) or user-supplied secrets.
  - `jwt_runner.go`: Orchestrates JWT attack execution and evaluates responses in parallel.

---

## 4. Reporting & Secret Masking Architecture

Results are handed to the `Reporter` interface in `pkg/engine/report`.
- **Secret Masking**: Before writing reports to console, JSON, HTML, or PDF, sensitive fields (passwords, auth tokens, bearer credentials) are automatically scrubbed or masked (e.g. `Bearer [MASKED]`) to prevent accidental secret exposure in build logs and CI/CD artifacts.

---

## 5. Schema Validation System (`schemas/`)

Simulation definitions are governed by structured schema specifications in [`schemas/`](file:///home/suryatk/ThreatSIM/schemas/):
- **`schemas/simulation.yaml`**: Definitive specification for standard HTTP simulations and plugin invocations.
- **`schemas/plugins/`**: Complete parameter requirements and type guidelines for `bruteforce`, `idor`, and `jwt_forge` plugins.
