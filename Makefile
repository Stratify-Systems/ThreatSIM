.PHONY: build run test lint clean list simulate server docker-up docker-down fmt

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

# Start the API server
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
	@echo "Usage:"
	@echo "  make build          Build the CLI binary"
	@echo "  make test           Run tests"
	@echo "  make test-cover     Run tests with coverage report"
	@echo "  make fmt            Format Go code"
	@echo "  make lint           Lint Go code"
	@echo "  make clean          Clean build artifacts"
	@echo "  make list           List available plugins"
	@echo "  make simulate       Run a quick brute force simulation"
	@echo "  make server         Start the API server"
	@echo "  make docker-up      Start full stack with Docker Compose"
	@echo "  make docker-down    Stop Docker Compose"
	@echo "  make dashboard      Build the React dashboard"
	@echo "  make dashboard-dev  Run dashboard in dev mode"
