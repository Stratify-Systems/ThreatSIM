# ThreatSim — Operational Troubleshooting & FAQ Guide

This document provides solutions for common operational issues, environment setup errors, and troubleshooting scenarios.

---

## ❓ Frequently Encountered Issues & Solutions

### 1. `Configuration Error: THREATSIM_AI_API_KEY environment variable is not set`

#### Cause:
You executed `threatsim ai generate`, `ai explain`, or `ai improve` without setting an API key in your `.env` file or environment.

#### Solution:
1. Copy `.env.example` to `.env`:
   ```bash
   cp .env.example .env
   ```
2. Edit `.env` and enter your Groq or OpenAI API key:
   ```ini
   THREATSIM_AI_API_KEY=gsk_your_actual_groq_api_key_here
   ```
3. If using local offline **Ollama**, set `THREATSIM_AI_PROVIDER=ollama` and `THREATSIM_AI_API_KEY=ollama`.

---

### 2. `x509: certificate signed by unknown authority` or `tls: bad certificate`

#### Cause:
You executed simulations against a local staging server, development environment, or test mock server that uses self-signed SSL/TLS certificates.

#### Solution:
Pass the `--insecure` flag on the command line or set `insecure: true` in your configuration file:

- **CLI Flag**:
  ```bash
  ./threatsim run -t https://staging.local --insecure -f tests/simulations/jwt_test.yaml
  ```

- **Configuration (`threatsim.yaml` / `fallback/threatsim.yaml`)**:
  ```yaml
  target_url: "https://staging.local"
  insecure: true
  ```

---

### 3. `Auth Failed: token path "token" not found in response`

#### Cause:
The `idor` or `jwt_forge` plugin attempted to authenticate to `auth_path`, but could not extract the token from the HTTP response using the specified `token_json_path`.

#### Solution:
1. Verify that your target application's authentication endpoint (`auth_path`) is running and returning HTTP `200 OK`.
2. Inspect the login JSON response structure. Use dot-notation syntax for nested fields:
   - If response is `{"token": "xyz"}`, set `token_json_path: "token"`.
   - If response is `{"data": {"token": "xyz"}}`, set `token_json_path: "data.token"`.
   - If response is `{"result": {"auth": {"bearer": "xyz"}}}`, set `token_json_path: "result.auth.bearer"`.

---

### 4. `Executed 50 burst requests but rate limiting was never triggered (Expected status 429)`

#### Cause:
The `rate_limit` or `bruteforce` plugin fired requests against an endpoint, but the target application returned status `404 Not Found` or did not trigger throttling.

#### Solution:
1. Ensure the endpoint path exists on your target application (e.g. `/api/search` vs `/search`).
2. Increase the `num_requests` and `concurrency` parameters in your YAML config if your target application's rate-limiting threshold is higher (e.g. set `num_requests: 100` and `concurrency: 20`).

---

### 5. `address already in use` when launching Mock Server

#### Cause:
Port 8080 or 8081 is already bound by a running mock server or another local application.

#### Solution:
Find and terminate the process using port 8080/8081:

```bash
# Find process on port 8080:
lsof -i :8080
# Kill process:
kill -9 <PID>
```

---

### 6. How Configuration File Fallbacks Work

When executing `./threatsim run` or `./threatsim ai generate` without explicitly passing `--target-url` (`-t`) or `--file` (`-f`), ThreatSim automatically searches for defaults in this exact order:

```text
1. threatsim.yaml             (Current workspace root)
2. .threatsim.yaml            (Hidden root configuration)
3. fallback/threatsim.yaml    (Default fallback directory)
```

If none of these configuration files exist and no CLI flags are supplied, ThreatSim will prompt for the required input interactively.
