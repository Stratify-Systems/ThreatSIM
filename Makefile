.PHONY: build run test lint clean list simulate validate server docker-up docker-down fmt

# Build the CLI binary
build:
	go build -o bin/threatsim ./cmd/threatsim/

# Run tests
test:
	go test -v ./...

# Run tests with coverage
test-cover:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Format code
fmt:
	gofmt -w ./

# Lint (requires golangci-lint)
lint:
	golangci-lint run ./...

# Clean build artifacts
clean:
	rm -rf bin/ coverage.out coverage.html

# List available plugins
list: build
	./bin/threatsim list

# Run a brute force simulation (default)
simulate: build
	./bin/threatsim simulate brute_force -d 5s -r 3

# ═══════════════════════════════════════
# CORE FEATURE: Security Validation Gate
# ═══════════════════════════════════════

# Validate that brute force detection works
validate: build
	./bin/threatsim validate --plugin brute_force --expect-alert

# Validate all attack detections
validate-all: build
	./bin/threatsim validate --plugin brute_force --expect-alert
	./bin/threatsim validate --plugin port_scan --expect-alert
	./bin/threatsim validate --plugin privilege_escalation --expect-alert

# Validate with JSON output (for CI/CD)
validate-json: build
	./bin/threatsim validate --plugin brute_force --expect-alert --json

# ═══════════════════════════════════════

# Start the API server (falls back to in-memory store if no Postgres)
server: build
	./bin/threatsim server

# Start full stack with Docker Compose
docker-up:
	docker compose up -d --build

# Stop Docker Compose
docker-down:
	docker compose down

# Build the dashboard
dashboard:
	cd dashboard && bun install && bun run build

# Run the dashboard in dev mode
dashboard-dev:
	cd dashboard && bun run dev

# Show help
help:
	@echo "ThreatSIM Makefile"
	@echo ""
	@echo "Core Commands:"
	@echo "  make validate       Run security detection validation (core feature)"
	@echo "  make validate-all   Validate all attack detections"
	@echo "  make validate-json  Validate with JSON output (CI/CD)"
	@echo ""
	@echo "Development:"
	@echo "  make build          Build the CLI binary"
	@echo "  make test           Run tests"
	@echo "  make test-cover     Run tests with coverage report"
	@echo "  make fmt            Format Go code"
	@echo "  make lint           Lint Go code"
	@echo "  make clean          Clean build artifacts"
	@echo ""
	@echo "Simulation:"
	@echo "  make list           List available plugins"
	@echo "  make simulate       Run a quick brute force simulation"
	@echo "  make server         Start the API server"
	@echo ""
	@echo "Docker:"
	@echo "  make docker-up      Start full stack with Docker Compose"
	@echo "  make docker-down    Stop Docker Compose"
