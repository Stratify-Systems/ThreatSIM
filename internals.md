# ThreatSim Internals & Architecture

ThreatSim is designed with a clean, extensible architecture that strictly separates the command-line interface (CLI) from the core execution engine. This ensures that the engine can be used as a library in other applications or extended without modifying the CLI layer.

## Project Structure

- `cmd/`: Contains the Cobra-based CLI commands (`root.go`, `run.go`). It acts as a thin layer that parses flags and invokes the engine.
- `pkg/engine/`: Contains the core business logic.
  - `engine.go`: Handles parsing simulation files, constructing HTTP requests, sending them, and validating the response against the expected outcome.
  - `report.go`: Formats the execution results into a human-readable summary.
- `pkg/types/`: Defines the data models used throughout the application.

## Core Components

### 1. Data Models (`pkg/types`)

The foundation of the engine is the `SimulationDefinition` which maps perfectly to the YAML/JSON simulation files.
- `Request`: Contains the HTTP method, path, headers, query parameters, and body.
- `Expected`: Defines the expected outcome. Currently, it supports `status_code`, but is designed to be easily extended with fields like `body_contains` or `headers_match`.
- `ValidationReport` and `SimulationResult`: Store the final state of the execution for reporting.

### 2. Execution Engine (`pkg/engine`)

The `Engine` struct encapsulates the HTTP client and target configuration.
- **Isolation:** The engine does not interact with `os.Stdout` or CLI arguments directly. It takes in a `TargetURL` and returns a `ValidationReport`.
- **Parsing:** `LoadSimulation` attempts to unmarshal the simulation file using YAML. Since YAML is a superset of JSON, this naturally supports both formats safely.
- **Execution:** `Execute` iterates over all simulations. The `executeSimulation` method processes an individual simulation independently, safely constructing the URL and request body, applying headers, and performing the HTTP round trip. 
- **Validation:** Currently, the engine validates the `status_code`. This logic is located at the end of `executeSimulation` and is primed for future extension.

### 3. Reporting (`pkg/engine/report.go`)

To maintain separation of concerns, the engine computes the raw `ValidationReport` data, and a separate `PrintReport` function is responsible for formatting it. This allows the CLI to easily output the results to `os.Stdout`, while a future web interface or API could consume the raw struct directly.

## Design Decisions

- **Go (Golang):** Chosen for its performance, concurrency support, robust standard library (`net/http`), and ease of distributing single-binary CLI tools.
- **Cobra CLI:** Used to provide a robust, industry-standard CLI experience with built-in help and flag parsing, keeping the `cmd` package thin.
- **Extensibility:** The `Expected` struct is explicitly designed so that new validation criteria can be added seamlessly without breaking existing simulation files.
- **Minimal Dependencies:** The project limits third-party libraries (relying mostly on standard libraries alongside `yaml.v3` and `cobra`) to ensure a stable, secure, and easily maintainable core.
