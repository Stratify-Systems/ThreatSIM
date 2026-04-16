# Phase 6: Observability & Deployment Plan

This document outlines the step-by-step tasks required to complete Phase 6 of the ThreatSIM project. The goal of this phase is to transition ThreatSIM from a local development environment into a fully containerized, observable, and easily deployable tool suite.

## 🎯 Objectives

- Containerize the backend and frontend.
- Provide unified orchestration via Docker Compose.
- Expose application metrics for monitoring.
- Visualize system performance and simulation telemetry.
- Prepare the application for cloud-native deployment.

---

## 📋 Task Breakdown

### Task 1: Application Containerization (Docker)

**Goal:** Package the backend and frontend into lightweight, reproducible Docker images.

- [x] Create `Dockerfile` for the Go backend API.
  - _Details:_ Use multi-stage builds (builder image for compilation, scratch/alpine image for runtime) to minimize size.
- [x] Create `Dockerfile` for the Vite/React dashboard.
  - _Details:_ Build the static assets and serve them using a lightweight web server like Nginx.
- [x] Ensure proper environment variable handling for database strings, ports, and API URLs inside both containers.

### Task 2: Unified Orchestration (Docker Compose)

**Goal:** Run the entire stack (Database, Backend, Frontend, and future Obeservability tools) from a single command.

- [x] Update the existing `docker-compose.yml` to include the new `backend` and `frontend` services.
- [x] Configure networking so the backend can seamlessly connect to the PostgreSQL `db` service.
- [x] Map appropriate ports for host access (e.g., `8080` for backend, `5173`/`80` for frontend, `5432` for DB).

### Task 3: Application Metrics (Prometheus)

**Goal:** Instrument the Go backend to expose performance and telemetry metrics.

- [x] Implement the Prometheus Go client library (`github.com/prometheus/client_golang/prometheus`) in the backend.
- [x] Create a dedicated HTTP endpoint (`GET /metrics`) in the Go router.
- [x] Define and record custom metrics, such as:
  - `threatsim_simulations_total` (Counter)
  - `threatsim_events_generated_total` (Counter)
  - `threatsim_active_alerts` (Gauge)
  - HTTP request durations (Histogram)
- [x] Add the Prometheus server to `docker-compose.yml` to scrape the backend.

### Task 4: Visualization (Grafana)

**Goal:** Visualize the metrics captured by Prometheus.

- [ ] Add a Grafana service to `docker-compose.yml`, linked to the Prometheus instance.
- [ ] Design and export Grafana dashboard JSON templates that show:
  - System performance (CPU/Memory of the ThreatSIM engine).
  - Threat detection rates and simulation statuses.
- [ ] Automate Grafana dashboard provisioning by binding a local `./dashboards` folder to the Grafana container.

### Task 5: Cloud-Native Deployment (Kubernetes Manifests)

**Goal:** Provide the necessary `.yaml` configs to deploy ThreatSIM on any standard K8s cluster.

- [ ] Create `kubernetes/` directory.
- [ ] Write `Deployment` and `Service` manifests for:
  - PostgreSQL Database
  - ThreatSIM Go Backend
  - ThreatSIM React Dashboard
- [ ] Add ConfigMaps and Secrets for state management and configuration.

---

## 🚀 Execution Strategy

We will tackle these linearly, starting with **Task 1 (Dockerizing the code)**, to ensure we have a robust environment before adding the heavy observability stack in Tasks 3 and 4.
