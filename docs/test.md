# Testing Alert Dispatcher (Phase 4, Step 1)

This guide provides the exact commands used to verify the Alert Dispatcher's deduplication, state tracking, and payload generation.

## 1. Start a Local Webhook Receiver
Use `http-echo-server` to spin up a quick, temporary local server that prints all incoming POST requests and their JSON payloads to your terminal.

Open **Terminal 1**:
```bash
# If not installed, you can run it via npx
npx http-echo-server 9000
```
This will start listening on `http://localhost:9000`.

## 2. Run the Brute Force Simulation
In a separate terminal, export the webhook URL environment variable so the ThreatSIM dispatcher registers the `WebhookNotifier`. Then, run a simulation that is guaranteed to escalate the Risk Score quickly.

Open **Terminal 2**:
```bash
# Export the URL where the echo server is listening
export THREATSIM_WEBHOOK_URL=http://localhost:9000

# Run a brute force simulation for 5 seconds at 10 requests per second
./threatsim simulate brute_force -d 5s -r 10
```

## 3. Verify the Expected Output
In **Terminal 1** (the echo server), you should see precisely **two** JSON payloads arrive:
1. The first alert when the Threat Level transitions to `HIGH` (Score >= 61).
2. The second alert when the Threat Level transitions to `CRITICAL` (Score >= 81).

Because of the state-based deduplication logic, you will **not** see an alert for every single event that keeps the score at `CRITICAL`. The `factors` array should also show uniquely tripped rules (no duplicate `"brute_force_attack"` strings).

## Testing REST API Core (Phase 4, Step 2)

This tests the new embedded in-memory HTTP API layer routing via `chi`. (Note this applies when the application exposes the HTTP listener via `threatsim server` step later in Phase 4).

### 1. Boot the JSON endpoints
Assuming you wire `api.NewServer()` properly, boot the API:
```bash
# E.g. when Step 5 wraps the listener 
go run ./cmd/threatsim server
```

### 2. cURL the in-memory states
You can poll the REST endpoints with standard JSON queries using HTTPie, curl, or Postman:

```bash
# Fetch running or completed simulations
curl -s http://localhost:8080/api/v1/simulations | jq

# Fetch deduplicated live alerts that fired hooks
curl -s http://localhost:8080/api/v1/alerts | jq

# Fetch all atomic streaming events
curl -s http://localhost:8080/api/v1/events | jq
```
