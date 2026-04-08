# Phase 4: Alert System, API, & Storage

Phase 4 evolves ThreatSIM from a localized, transient CLI tool into a persistent, enterprise-ready service. It introduces state, external networking, concurrent HTTP serving, and asynchronous external alerting.

Because of the complexity, Phase 4 is broken down into five isolated steps, focusing on fast iterative wins and building the API layer rapidly with an in-memory store before introducing database persistence logic.

## Step 1: The Alerting Dispatcher (Fast Win)

Evolve the Risk Engine so it can page external systems instead of just logging to the terminal.

- **Action:** Build the `internal/alerting` package.

**Tasks:**

- [x] Define a standard `Notifier` interface: `Send(alert core.Alert) error`.
- [x] Implement the **Webhook Channel** (HTTP POSTs to generic SIEMs).
- [x] Implement the **Slack Channel** (formatted Slack blocks).
- [x] Implement the **Email Channel** (SMTP).
- [x] Wire these to trigger when the Risk Engine hits a `HIGH` or `CRITICAL` threshold.

## Step 2: The REST API Core (In-Memory First)

Build the API so external dashboards (Phase 5) can query running telemetry.

- **Action:** Build the `internal/api` package backed by temporary memory states.

**Tasks:**

- [x] Set up an HTTP router (Go 1.22+ `net/http` multiplexer or `chi`).
- [x] Create REST endpoints for historical data:
  - `GET /api/v1/simulations`
  - `GET /api/v1/alerts`
  - `GET /api/v1/events`
- [x] Hook these into an intermediate in-memory map/slice struct to get the handlers working immediately.

## Step 3: Real-Time WebSockets

The dashboard will need live data so it doesn't have to poll the REST API constantly.

- **Action:** Implement WebSocket streaming.

**Tasks:**

- [x] Introduce `gorilla/websocket` or `nhooyr/websocket`.
- [x] Create a `GET /ws/live` endpoint.
- [x] Tap into the existing `memory.Stream` and `RiskEngine.Alerts()` channels.
- [x] Broadcast any runtime event or risk alert to all connected UI clients instantly.

## Step 4: The Storage Foundation (Database & Migrations)

Finalize state by swapping the REST API's in-memory mock arrays with actual durable persistence.

- **Database:** PostgreSQL
- **Migration Tool:** `goose` (https://github.com/pressly/goose)

**Tasks:**

- [ ] Set up `goose` for schema migrations.
- [ ] Write SQL schemas for `simulations`, `events`, and `alerts` tables.
- [ ] Create an `internal/store` package with interfaces for saving and querying data.
- [ ] Refactor Step 2 REST handlers to query `store` instead of memory tracking arrays.

## Step 5: The Server Daemon CLI

Wrap the persistent systems into a new CLI command to act as a long-running service.

- **Action:** Add `threatsim server`.

**Tasks:**

- [ ] Create `cmd/threatsim/server.go`.
- [ ] Have the command boot the PostgreSQL connection pool.
- [ ] Start the Alert Manager background workers.
- [ ] Spin up the REST/WebSocket server on port `8080`.
- [ ] Sit as a persistent daemon waiting for HTTP connections or simulations to be triggered via API.
