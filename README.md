# ThreatSIM

**Security as Code — Validate your application's security behavior before deployment.**

ThreatSIM is a platform that simulates predefined cyber attacks against your applications using safe HTTP requests and verifies whether they respond correctly. Think of it as **unit tests for your security layer**.

> This is **not** a vulnerability scanner or penetration testing tool.
> ThreatSIM only sends predefined, safe HTTP requests and compares the actual response with the expected result — PASS or FAIL.

---

## How It Works

```
         ┌────────────┐
         │  React UI  │
         └─────┬──────┘
               │
         ┌─────▼──────────────┐
         │  Spring Boot REST  │
         │       API          │
         └─────┬──────────────┘
               │
         ┌─────▼──────────────┐
         │ Simulation Service │
         └─────┬──────────────┘
               │
         ┌─────▼──────────────┐
         │ Simulation Engine  │
         └─────┬──────────────┘
               │
     ┌─────────▼─────────┐
     │  Attack Executor   │──────▶ Target Application (any HTTP API)
     └─────────┬─────────┘              │
               │                        │
     ┌─────────▼─────────┐    ◀────────┘
     │  Result Analyzer   │   (HTTP Response)
     └─────────┬─────────┘
               │
         ┌─────▼──────┐
         │ PostgreSQL  │
         └────────────┘
```

---

## Example

**1. Register an application:**

| Field    | Value                      |
|----------|----------------------------|
| Name     | Shopping API               |
| Base URL | `https://shopping-api.com` |

**2. Create a simulation:**

| Field           | Value              |
|-----------------|--------------------|
| Attack Type     | SQL Injection      |
| Method          | POST               |
| Endpoint        | `/login`           |
| Request Body    | `{"username": "admin", "password": "' OR 1=1 --"}` |
| Expected Status | `403 Forbidden`    |

**3. Run it.** ThreatSIM sends the request and compares the response:

```
✅ PASS — Expected 403, got 403 (142ms)
```

or

```
❌ FAIL — Expected 403, got 200 (89ms)
```

If your login endpoint returns `200 OK` to a SQL injection payload, your security is broken — and ThreatSIM catches it before production.

---

## Tech Stack

| Layer    | Technology                                     |
|----------|------------------------------------------------|
| Backend  | Spring Boot, Spring Web, Spring Data JPA, Lombok |
| Database | PostgreSQL                                     |
| Frontend | React                                          |
| Auth     | JWT (planned)                                  |

---

## Core Modules

### Application Management
Register and manage the external applications you want to test. Each application has a name, base URL, and active status.

### Simulation Management
Define attack simulations — the HTTP method, endpoint, headers, request body, and the expected response status. Simulations are tied to a registered application.

### Attack Executor
Builds and sends HTTP requests to the target application. Handles timeouts, connection failures, and collects response metadata.

### Result Analyzer
Compares the expected response with the actual response. Determines PASS or FAIL and records execution duration.

### Report Service
Stores execution history and provides access to past simulation results, allowing you to track security behavior over time.

---

## Database Schema

### `application`

| Column      | Type        | Description                     |
|-------------|-------------|---------------------------------|
| id          | UUID        | Primary key                     |
| name        | VARCHAR     | Application name                |
| base_url    | VARCHAR     | Base URL of the target API      |
| created_at  | TIMESTAMP   | When the application was added  |

### `simulation`

| Column          | Type        | Description                          |
|-----------------|-------------|--------------------------------------|
| id              | UUID        | Primary key                          |
| application_id  | UUID        | FK → application                     |
| attack_type     | VARCHAR     | e.g. SQL_INJECTION, XSS, AUTH_BYPASS |
| http_method     | VARCHAR     | GET, POST, PUT, DELETE, PATCH        |
| endpoint        | VARCHAR     | Target endpoint path                 |
| headers         | JSON        | Custom request headers               |
| request_body    | TEXT        | Request payload                      |
| expected_status | INT         | Expected HTTP status code            |
| enabled         | BOOLEAN     | Whether this simulation is active    |

### `simulation_result`

| Column          | Type        | Description                     |
|-----------------|-------------|---------------------------------|
| id              | UUID        | Primary key                     |
| simulation_id   | UUID        | FK → simulation                 |
| status          | VARCHAR     | PASS or FAIL                    |
| actual_status   | INT         | HTTP status code received       |
| response_body   | TEXT        | Response from target            |
| duration        | BIGINT      | Execution time in milliseconds  |
| executed_at     | TIMESTAMP   | When the simulation was run     |

---

## API Endpoints (MVP)

### Applications

| Method   | Endpoint                | Description              |
|----------|-------------------------|--------------------------|
| `POST`   | `/api/v1/applications`  | Register an application  |
| `GET`    | `/api/v1/applications`  | List all applications    |
| `GET`    | `/api/v1/applications/{id}` | Get application details |
| `PUT`    | `/api/v1/applications/{id}` | Update an application   |
| `DELETE` | `/api/v1/applications/{id}` | Delete an application   |

### Simulations

| Method   | Endpoint                          | Description                |
|----------|-----------------------------------|----------------------------|
| `POST`   | `/api/v1/simulations`             | Create a simulation        |
| `GET`    | `/api/v1/simulations`             | List all simulations       |
| `GET`    | `/api/v1/simulations/{id}`        | Get simulation details     |
| `PUT`    | `/api/v1/simulations/{id}`        | Update a simulation        |
| `DELETE` | `/api/v1/simulations/{id}`        | Delete a simulation        |
| `POST`   | `/api/v1/simulations/{id}/run`    | **Run a simulation**       |

### Results

| Method   | Endpoint                                    | Description                      |
|----------|---------------------------------------------|----------------------------------|
| `GET`    | `/api/v1/simulations/{id}/results`          | Get results for a simulation     |
| `GET`    | `/api/v1/results/{id}`                      | Get a specific result            |

### Health

| Method | Endpoint   | Description  |
|--------|------------|--------------|
| `GET`  | `/health`  | Health check |

---

## Design Principles

- **Clean Architecture** — Controllers are thin. Business logic lives in services. Execution logic is separate from result analysis.
- **SOLID Principles** — Single responsibility across modules. Depend on abstractions (e.g., the execution strategy is pluggable).
- **Extensibility** — The architecture is designed so future execution methods (Docker containers, remote agents, Kubernetes jobs) can be added without major refactoring.
- **Production-Ready Mindset** — Proper error handling, validation, logging, and structured API responses from day one.

---

## Roadmap

### Version 1 (MVP) — Current
- [x] Application registration and management
- [x] Simulation CRUD
- [x] HTTP-based attack execution
- [x] Result comparison (expected vs actual status)
- [x] Execution history and reports
- [x] React dashboard
- [x] PostgreSQL persistence

### Version 2
- [ ] JWT authentication
- [ ] Scheduled simulations (cron)
- [ ] Email / Slack notifications on failure
- [ ] Batch execution (run all simulations for an app)

### Version 3
- [ ] GitHub Actions integration
- [ ] Jenkins plugin
- [ ] CI/CD pipeline security gate

### Version 4
- [ ] Docker-based execution environment
- [ ] Agent-based remote execution
- [ ] Kubernetes job execution

### Version 5
- [ ] SIEM integration
- [ ] Live execution logs (WebSocket)
- [ ] Advanced reporting dashboard
- [ ] Multi-tenant support

---

## License

MIT License — See [LICENSE](LICENSE) for details.
