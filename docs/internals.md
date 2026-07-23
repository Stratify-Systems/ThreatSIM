# ThreatSim Architecture & Internals Documentation

ThreatSim is engineered with a strict separation of concerns, decoupling the CLI layer from the core execution engine, plugin framework, and AI authoring pipeline. This ensures the engine is portable, highly testable, thread-safe, and easily extensible.

---

## 1. Project Directory Structure

```text
threatsim/
├── cmd/                        # Cobra CLI layer (flag parsing, command routing)
│   ├── root.go                 # Root CLI command definition
│   ├── run.go                  # "run" subcommand (simulation execution & reporting)
│   ├── plugins.go              # "plugins" subcommand (schema & plugin discovery)
│   ├── ai.go                   # "ai" subcommand group
│   ├── ai_generate.go          # "ai generate" subcommand (natural language & OpenAPI policy generator)
│   ├── ai_explain.go           # "ai explain" subcommand (plain-English policy explainer)
│   └── ai_improve.go           # "ai improve" subcommand (security coverage gap analyzer)
├── docs/                       # Developer and architecture documentation
│   ├── ai_authoring.md         # AI policy engineering & LLM setup guide
│   ├── cicd_integration.md     # CI/CD integration guide (GitHub Actions, GitLab CI, Jenkins)
│   ├── internals.md            # Technical engine architecture documentation
│   ├── plugin_authoring.md     # Go security plugin development guide
│   ├── plugins_reference.md    # Complete schema reference manual for all plugins
│   └── troubleshooting.md     # Operational FAQ & troubleshooting guide

├── examples/                   # Built-in test mock servers and OpenAPI samples
│   ├── mockserver/             # Vulnerable mock server (Port 8080)
│   ├── secure_mockserver/      # Secure mock server (Port 8081)
│   └── openapi_sample.json     # Sample OpenAPI v3 specification file
├── pkg/
│   ├── ai/                     # AI authoring engine (OpenAI/Groq client, prompts, validation)
│   │   ├── client.go           # OpenAI-compatible API client (.env, Groq defaults)
│   │   ├── explainer.go        # Security policy explanation generator
│   │   ├── generator.go        # Generation orchestrator, sanitizer, self-correction retry loop
│   │   ├── improver.go         # Security policy coverage gap analyzer & builder
│   │   ├── models.go           # Chat completion DTO models
│   │   ├── openapi.go          # OpenAPI / Swagger spec parser
│   │   └── prompt.go           # Dynamic schema-driven system prompt builder
│   ├── engine/                 # Core engine (parser, executor, concurrency, multi-reporters)
│   │   ├── engine.go           # Engine construction, functional options, parallel execution
│   │   ├── report.go           # Console & JSON reporter implementations + secret masking
│   │   ├── report_html.go      # HTML reporter implementation
│   │   ├── report_pdf.go       # PDF reporter implementation
│   │   ├── report_sarif.go     # OASIS SARIF v2.1.0 reporter (GitHub Security)
│   │   └── report_junit.go     # JUnit XML reporter (GitLab CI, Jenkins, CircleCI)
│   ├── plugins/                # Plugin registry & security workflow implementations
│   │   ├── plugin.go           # Plugin interface, global registry, ValidateConfig & config helpers
│   │   ├── bruteforce.go       # Bruteforce rate-limiting plugin
│   │   ├── cors_audit.go       # CORS origin reflection audit plugin
│   │   ├── idor.go             # IDOR cross-tenant isolation plugin
│   │   ├── jwt_forge.go        # JWT signature forgery plugin
│   │   ├── rate_limit.go       # Endpoint rate-limiting burst plugin
│   │   └── utils/              # Modular attack utilities
│   │       ├── auth/           # Shared auth & JSON path extraction helper
│   │       ├── bruteforce/     # Password dictionary generators
│   │       ├── cors/           # CORS origin audit runner
│   │       ├── idor/           # Cross-tenant IDOR attack runner
│   │       ├── jwt/            # JWT attack modes & runner
│   │       └── rate_limit/     # API endpoint rate-limiting burst runner
│   └── types/                  # Global data structures & domain models
│       └── simulation.go       # Simulation, Request, Expected, Result models
├── schemas/                    # Authoritative YAML schema definitions
│   ├── simulation.yaml         # General simulation schema reference
│   └── plugins/                # Plugin-specific config schema templates
│       ├── bruteforce.yaml
│       ├── cors_audit.yaml
│       ├── idor.yaml
│       ├── jwt_forge.yaml
│       └── rate_limit.yaml
├── tests/                      # Centralized test workspace
│   ├── unit/                   # Go unit tests (engine, plugins, AI, reporters)
│   │   ├── ai_test.go
│   │   ├── auth_session_test.go
│   │   ├── bruteforce_test.go
│   │   ├── cors_audit_test.go
│   │   ├── engine_test.go
│   │   ├── idor_test.go
│   │   ├── jwt_test.go
│   │   ├── plugin_framework_test.go
│   │   ├── rate_limit_test.go
│   │   └── report_cicd_test.go
│   └── simulations/            # Integration YAML test scenarios
│       ├── bruteforce.yaml
│       ├── cors_test.yaml
│       ├── generated.yaml
│       ├── idor_test.yaml
│       ├── improved.yaml
│       ├── jwt_test.yaml
│       ├── openapi_generated.yaml
│       ├── parallel_test.yaml
│       ├── rate_limit_test.yaml
│       └── timeout_test.yaml
├── fallback/
│   └── threatsim.yaml          # Default fallback workspace configuration file
├── .env.example                # Template AI credentials file (Groq default)
├── LICENSE                     # MIT License
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
1. **Configuration Loading**: Parses global defaults from `threatsim.yaml`, `.threatsim.yaml`, or `fallback/threatsim.yaml` when CLI flags are omitted.

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
  - `rate_limit`: Fires concurrent burst requests across parallel worker routines.
  - `cors_audit`: Preflight OPTIONS and actual origin reflection checks run concurrently.
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

### B. Schema Validation & Thread-Safe State Writeback
- **`ValidateConfig(pluginName, config)`**: Automatically validates plugin parameters against YAML schemas in `schemas/plugins/*.yaml` or built-in field validation rules prior to executing network requests.
- **Config Helper Functions**: `ParseString()`, `ParseInt()`, and `ParseBool()` provide safe type coercion for plugin configuration maps.
- **`StateMap` Container (`ctx.SetState`, `ctx.GetState`)**: Provides a thread-safe mutex-protected state storage engine (`RWMutex`) allowing plugins to persist extracted tokens and session variables across executions.

### C. Modular Utility Packages (`pkg/plugins/utils/`)
- **`auth/auth_session.go`**: Centralized authentication helper (`auth.AuthenticateAndExtract`). Sends authentication POST requests and extracts tokens or resource IDs using dot-notation JSON paths (`ExtractJSONPath`).
- **`bruteforce/bruteforce_gen.go`**: Password generation helper supporting custom wordlist files (`wordlist_path`) and custom JSON payload field keys (`username_field`, `password_field`).
- **`cors/cors_runner.go`**: CORS origin audit runner testing preflight and actual request origin reflection (`https://attacker.com`, `null`) and wildcard credentials.
- **`idor/idor_runner.go`**: Cross-tenant IDOR attack runner supporting multi-method (`GET`, `POST`, `PUT`, `DELETE`, `PATCH`) and `{id}` template substitution in path URLs and JSON payloads.
- **`jwt/`**: Modular JWT attack suite (`signature_tamper`, `alg_none`, `weak_secret`).
- **`rate_limit/rate_limit_runner.go`**: Public endpoint throttling audit runner firing configurable concurrent request bursts.

---

## 4. Reporting & Secret Masking Architecture

Results are handed to the `Reporter` interface implementations (`ConsoleReporter`, `JSONReporter`, `HTMLReporter`, `PDFReporter`, `SARIFReporter`, `JUnitReporter`).
- **Setup/Auth Error Differentiation**: `types.SimulationResult` includes an `IsError bool` field and `types.ValidationReport` includes an `Errors int` counter. Pre-flight authentication or test configuration setup failures are flagged as `⚠️ [ERROR] Test Setup Failure` with diagnostic guidance rather than false security vulnerabilities (`✗ [FAIL]`).
- **Interactive Terminal Reporter (`ConsoleReporter`)**: Prints an ANSI-styled ASCII logo banner (`PrintBanner`), an Executive Validation Box summarizing pass rates, failed controls, setup errors, and execution times, and detailed simulation breakdowns (Target Endpoint, Assertion Match, Root Cause Analysis, Diagnostic Tips, and Latency).
- **SARIF v2.1.0 (`SARIFReporter`)**: Formats security control failures into OASIS SARIF v2.1.0 JSON format for native integration into GitHub Actions Security Code Scanning tabs (`upload-sarif`).
- **JUnit XML (`JUnitReporter`)**: Formats test results into standard JUnit XML schema (`<testsuites>`, `<testcase>`, `<failure>`) for GitLab CI, Jenkins, Azure DevOps, and CircleCI.
- **Secret Masking**: Before writing reports to console, JSON, HTML, PDF, SARIF, or JUnit, sensitive fields (passwords, auth tokens, bearer credentials) are automatically scrubbed or masked (e.g. `Bearer ********`) to prevent accidental secret exposure in build logs and CI/CD artifacts.

---

## 5. Schema Validation System (`schemas/`)

Simulation definitions are governed by structured schema specifications in [`schemas/`](file:///home/suryatk/ThreatSIM/schemas/):
- **`schemas/simulation.yaml`**: Definitive specification for standard HTTP simulations and plugin invocations.
- **`schemas/plugins/`**: Complete parameter requirements and type guidelines for `bruteforce`, `cors_audit`, `idor`, `jwt_forge`, and `rate_limit` plugins.

---

## 6. AI Engine Architecture (`pkg/ai/`)

ThreatSim includes an AI authoring and analysis suite under `threatsim ai`:
- **Deterministic Boundary**: The AI engine is purely an authoring assistant. The core execution engine (`pkg/engine/`) remains the sole arbiter of security validation.
- **Vendor-Agnostic HTTP Client (`pkg/ai/client.go`)**: Communicates with any OpenAI-compatible API endpoint (Groq, OpenAI, Ollama, custom proxies) configured via `.env` (`THREATSIM_AI_PROVIDER`, `THREATSIM_AI_BASE_URL`, `THREATSIM_AI_MODEL`, `THREATSIM_AI_API_KEY`). Defaults to Groq (`llama-3.3-70b-versatile`).
- **Schema-Driven Prompting (`pkg/ai/prompt.go`)**: System prompts dynamically incorporate authoritative YAML schemas from `schemas/` so LLM outputs stay synchronized with available plugin definitions.
- **Policy Generation, Interactive Review & Auto-Execution (`pkg/ai/generator.go` & `cmd/ai_generate.go`)**:
  - Sanitizes output by stripping markdown code fences (` ```yaml ... ``` `).
  - Validates generated output against ThreatSim's engine parser (`engine.LoadSimulation()`).
  - Automatically feeds validation errors back to the LLM for a corrected attempt if validation fails (up to 2 retries).
  - **Interactive Policy Preview & Review Pause**: Displays the full generated YAML content (`GENERATED POLICY PREVIEW`) and pauses execution with an interactive prompt `[y/N]`. Allows developers to open, inspect, or edit `tests/simulations/generated.yaml` in an external editor before confirming. Upon user confirmation (`y`), ThreatSim automatically re-loads the updated YAML file and executes the simulations against the target application. Supports CLI flag `--yes` (`-y`) to bypass the confirmation prompt for automated CI/CD pipelines.
- **OpenAPI Specification Import (`pkg/ai/openapi.go`)**:

  - `threatsim ai generate --openapi <spec-file>`: Parses OpenAPI v2/v3 JSON/YAML specs and automatically formats a structured requirement suite for LLM generation.

- **Policy Explanation Engine (`pkg/ai/explainer.go` & `cmd/ai_explain.go`)**:
  - `threatsim ai explain -f <file>`: Analyzes simulation YAML definitions and generates plain-English audit summaries detailing tested attack vectors and security control boundaries.
  - **Terminal ANSI Markdown Rendering**: Renders rich ANSI-styled Markdown (colored headers, bold text, styled bullets) for interactive terminal viewing (`RenderTerminalMarkdown`).
  - **Clean Markdown File Output**: Saves pure GitHub-Flavored Markdown to disk when `-o <file.md>` is specified.
- **Policy Coverage Improvement Engine (`pkg/ai/improver.go` & `cmd/ai_improve.go`)**:
  - `threatsim ai improve -f <file>`: Analyzes existing simulation files for security coverage gaps and generates complementary, schema-validated simulation entries to strengthen the suite.


