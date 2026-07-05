# ThreatSIM MVP Core Workflow Guide

This document outlines the step-by-step API flow to register applications, create security validation simulations, execute runs, and query histories in ThreatSIM.

---

## Overview of the Flow

```
1. Register App  ──▶  2. Create Sim  ──▶  3. Run Sim  ──▶  4. Check History
   (baseUrl)           (payload/expect)      (status check)     (execution log)
```

---

## 1. Register an Application

Before running tests, register the target application. This stores the base URL configuration.

### HTTP Request
`POST /api/v1/applications`

```bash
curl -X POST http://localhost:8080/api/v1/applications \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Target API (HttpBin)",
    "baseUrl": "https://httpbin.org"
  }'
```

### Response
```json
{
  "success": true,
  "message": "Application registered",
  "data": {
    "id": "755e357b-2a61-4ef0-936b-2dd391b6e0f2",
    "name": "Target API (HttpBin)",
    "baseUrl": "https://httpbin.org",
    "createdAt": "2026-07-05T21:24:00"
  },
  "timestamp": "2026-07-05T21:24:00.120729"
}
```
*Note the `"id"` in `"data"`. You will use this as `applicationId` in the next step.*

---

## 2. Create a Security Simulation

Define a security validation test case. Specify the target endpoint, HTTP method, attack payload headers/body, and the HTTP response status code you expect if security layers function correctly (e.g., `403 Forbidden`).

### HTTP Request
`POST /api/v1/simulations`

```bash
curl -X POST http://localhost:8080/api/v1/simulations \
  -H "Content-Type: application/json" \
  -d '{
    "applicationId": "755e357b-2a61-4ef0-936b-2dd391b6e0f2",
    "attackType": "SQL_INJECTION",
    "httpMethod": "POST",
    "endpoint": "/post",
    "headers": {
      "Content-Type": "application/json"
    },
    "requestBody": "{\"username\": \"admin\", \"password\": \"'\'' OR 1=1 --\"}",
    "expectedStatus": 403
  }'
```

### Response
```json
{
  "success": true,
  "message": "Simulation created",
  "data": {
    "id": "2f6fe35c-d9ea-48fe-ae3b-bde87713fd9b",
    "applicationId": "755e357b-2a61-4ef0-936b-2dd391b6e0f2",
    "applicationName": "Target API (HttpBin)",
    "attackType": "SQL_INJECTION",
    "httpMethod": "POST",
    "endpoint": "/post",
    "headers": {
      "Content-Type": "application/json"
    },
    "requestBody": "{\"username\": \"admin\", \"password\": \"' OR 1=1 --\"}",
    "expectedStatus": 403,
    "enabled": true
  },
  "timestamp": "2026-07-05T21:25:04.918523"
}
```
*Note the simulation `"id"` from the response. Use this ID to trigger the test in the next step.*

---

## 3. Run the Simulation

Execute the attack simulation against the target application. The executor will:
1. Construct the target URL (`baseUrl` + `endpoint`).
2. Attach headers and body.
3. Fire a real HTTP request.
4. Record the duration, response body, and status code.
5. Determine:
   * **`PASS`**: If `actualStatus` matches `expectedStatus`.
   * **`FAIL`**: If `actualStatus` differs from `expectedStatus` (e.g. application returned `200 OK` to an attack payload instead of `403 Forbidden`).
   * **`ERROR`**: If the target server is unreachable or times out.

### HTTP Request
`POST /api/v1/simulations/{id}/run`

```bash
curl -X POST http://localhost:8080/api/v1/simulations/2f6fe35c-d9ea-48fe-ae3b-bde87713fd9b/run
```

### Response (Example of a Security Failure / FAIL)
```json
{
  "success": true,
  "message": "Simulation executed",
  "data": {
    "id": "ec6636ce-8038-4888-b03a-34e01b4b4503",
    "simulationId": "2f6fe35c-d9ea-48fe-ae3b-bde87713fd9b",
    "status": "FAIL",
    "expectedStatus": 403,
    "actualStatus": 200,
    "responseBody": "{\n  \"args\": {}, ... \n  \"url\": \"https://httpbin.org/post\"\n}\n",
    "duration": 1828,
    "executedAt": null
  },
  "timestamp": "2026-07-05T21:36:28.000401"
}
```

---

## 4. Query Execution History

Retrieve a chronologically ordered list of previous runs for a specific simulation to track security behavior over time.

### HTTP Request
`GET /api/v1/simulations/{id}/results`

```bash
curl http://localhost:8080/api/v1/simulations/2f6fe35c-d9ea-48fe-ae3b-bde87713fd9b/results
```

### Response
```json
{
  "success": true,
  "data": [
    {
      "id": "ec6636ce-8038-4888-b03a-34e01b4b4503",
      "simulationId": "2f6fe35c-d9ea-48fe-ae3b-bde87713fd9b",
      "status": "FAIL",
      "expectedStatus": 403,
      "actualStatus": 200,
      "responseBody": "...",
      "duration": 1828,
      "executedAt": "2026-07-05T21:36:27.981174"
    },
    {
      "id": "57708886-805c-40f9-9968-05255db8eb9a",
      "simulationId": "2f6fe35c-d9ea-48fe-ae3b-bde87713fd9b",
      "status": "FAIL",
      "expectedStatus": 403,
      "actualStatus": 200,
      "responseBody": "...",
      "duration": 1622,
      "executedAt": "2026-07-05T21:26:19.522919"
    }
  ],
  "timestamp": "2026-07-05T21:36:45.353684"
}
```
