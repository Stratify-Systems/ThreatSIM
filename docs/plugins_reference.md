# ThreatSim — Security Plugins Reference Manual

ThreatSim features 5 native stateful security validation plugins in [`pkg/plugins/`](file:///home/suryatk/ThreatSIM/pkg/plugins/). This document is the definitive schema and configuration reference for all plugins.

---

## 📑 Quick Plugin Index

| Plugin Name | Primary Security Domain | Tested Vulnerabilities / Controls |
|:---|:---|:---|
| **[`idor`](#1-idor--cross-tenant-authorization-isolation-plugin)** | Authorization / Access Control | BOLA / IDOR, Cross-tenant data leakage, Body IDOR |
| **[`jwt_forge`](#2-jwt_forge--jwt-signature--header-tampering-plugin)** | Authentication / Cryptography | Tampered signatures, `alg=none` header bypass, weak HMAC secrets |
| **[`bruteforce`](#3-bruteforce--authentication-brute-force--lockout-plugin)** | Rate Limiting / Lockout | Password brute-force, account soft-lockouts |
| **[`cors_audit`](#4-cors_audit--cors-origin-reflection--credentials-audit-plugin)** | Browser Security / CORS | Origin reflection (`attacker.com`, `null`), wildcard `*` with credentials |
| **[`rate_limit`](#5-rate_limit--api-endpoint-throttling-burst-plugin)** | Availability / DDoS | API endpoint burst throttling (`429 Too Many Requests`) |

---

## 1. `idor` — Cross-Tenant Authorization Isolation Plugin

Evaluates Broken Object Level Authorization (BOLA) by authenticating as two distinct users in parallel (User A and User B), extracting User A's resource token and ID, and attempting to access or modify User A's private data using User B's authentication credentials.

### Configuration Parameters

| Parameter | Type | Required? | Default | Description |
|:---|:---:|:---:|:---:|:---|
| `auth_path` | String | **Required** | — | Endpoint path to authenticate users (e.g. `/auth/login`). |
| `user_a_payload` | String | **Required** | — | JSON payload string to authenticate User A (resource owner). |
| `user_b_payload` | String | **Required** | — | JSON payload string to authenticate User B (unauthorized guest). |
| `token_json_path` | String | **Required** | — | Dot-notation JSON path to extract Bearer token from login response (e.g. `data.token`). |
| `id_json_path` | String | **Required** | — | Dot-notation JSON path to extract resource/user ID from User A login response (e.g. `data.user.id`). |
| `target_path` | String | **Required** | — | Target path containing `{id}` template variable (e.g. `/api/users/{id}/private-data`). |
| `target_method` | String | Optional | `"GET"` | HTTP Method (`GET`, `POST`, `PUT`, `DELETE`, `PATCH`). |
| `target_payload` | String | Optional | `""` | Optional HTTP body string for `PUT`/`POST` IDOR tests. Supports `{id}` substitution. |
| `expected_status_code` | Integer | Optional | `403` | Status code that indicates successful access rejection (e.g. `403` or `401`). |

### YAML Example

```yaml
version: "1.0"
simulations:
  - name: "Cross-Tenant GET Path IDOR Validation"
    plugin: "idor"
    config:
      auth_path: "/auth/login"
      user_a_payload: '{"username":"admin", "password":"secret123"}'
      user_b_payload: '{"username":"guest", "password":"password123"}'
      token_json_path: "data.token"
      id_json_path: "data.user.id"
      target_method: "GET"
      target_path: "/api/users/{id}/private-data"
      expected_status_code: 403
```

---

## 2. `jwt_forge` — JWT Signature & Header Tampering Plugin

Evaluates JWT authentication enforcement across three distinct attack vectors:
1. `signature_tamper`: Modifies the JWT signature bytes.
2. `alg_none`: Sets header `"alg": "none"` and strips the signature.
3. `weak_secret`: Re-signs the token payload using a weak secret key dictionary (`secret`, `123456`, `password`).

### Configuration Parameters

| Parameter | Type | Required? | Default | Description |
|:---|:---:|:---:|:---:|:---|
| `auth_path` | String | **Required** | — | Endpoint path to authenticate and obtain a valid baseline JWT token. |
| `auth_payload` | String | **Required** | — | JSON payload string for login authentication. |
| `target_path` | String | **Required** | — | Protected administrative endpoint path to attempt unauthorized access. |
| `attack_mode` | String | **Required** | — | Attack vector: `signature_tamper`, `alg_none`, or `weak_secret`. |
| `token_json_path` | String | Optional | `"token"` | Dot-notation JSON path to extract JWT token from login response. |
| `target_method` | String | Optional | `"GET"` | HTTP Method used when accessing protected target path. |
| `expected_status_code` | Integer | Optional | `401` | Expected status code indicating token rejection. |
| `expected_body_contains` | String | Optional | `""` | Body string assertion indicating token rejection (e.g. `Unauthorized`). |

### YAML Example

```yaml
version: "1.0"
simulations:
  - name: "JWT Alg None Header Bypass Test"
    plugin: "jwt_forge"
    config:
      auth_path: "/auth/login"
      auth_payload: '{"username":"guest", "password":"password123"}'
      target_path: "/api/admin/secrets"
      attack_mode: "alg_none"
      token_json_path: "data.token"
      expected_status_code: 401
```

---

## 3. `bruteforce` — Authentication Brute-Force & Lockout Plugin

Rapidly fires authentication attempts using concurrent worker pools to evaluate whether the application enforces password brute-force protections or account soft-lockouts.

### Configuration Parameters

| Parameter | Type | Required? | Default | Description |
|:---|:---:|:---:|:---:|:---|
| `path` | String | **Required** | — | Authentication endpoint path (e.g. `/auth/login`). |
| `username` | String | **Required** | — | Target username/email to test against. |
| `num_requests` | Integer | **Required** | — | Total number of invalid login attempts to generate. |
| `wordlist_path` | String | Optional | Built-in | Path to custom password wordlist file. Defaults to built-in 10-password list. |
| `username_field` | String | Optional | `"username"` | Custom JSON payload key for username. |
| `password_field` | String | Optional | `"password"` | Custom JSON payload key for password. |
| `expected_status_code` | Integer | Optional | `429` | Expected HTTP status code indicating rate-limiting or lockout (e.g. `429`). |
| `expected_body_contains` | String | Optional | `""` | Body string assertion indicating lockout (e.g. `locked`). |

### YAML Example

```yaml
version: "1.0"
simulations:
  - name: "Login Lockout Rate Limit Test"
    plugin: "bruteforce"
    config:
      path: "/auth/login"
      username: "admin@example.com"
      num_requests: 15
      username_field: "user_email"
      password_field: "user_pass"
      expected_status_code: 429
      expected_body_contains: "locked"
```

---

## 4. `cors_audit` — CORS Origin Reflection & Credentials Audit Plugin

Audits Cross-Origin Resource Sharing (CORS) header policies by sending preflight `OPTIONS` and actual HTTP requests with untrusted Origin headers (`https://attacker.com`, `null`). Identifies dangerous origin reflections combined with `Access-Control-Allow-Credentials: true`.

### Configuration Parameters

| Parameter | Type | Required? | Default | Description |
|:---|:---:|:---:|:---:|:---|
| `path` | String | **Required** | — | Target API endpoint path to audit. |
| `method` | String | Optional | `"GET"` | HTTP Method to send with CORS origin requests. |
| `untrusted_origins` | Array | Optional | `["https://attacker.com", "null"]` | Custom list of untrusted origins to test. |

### YAML Example

```yaml
version: "1.0"
simulations:
  - name: "CORS Untrusted Origin Reflection Audit"
    plugin: "cors_audit"
    config:
      path: "/api/users/100/private-data"
      method: "GET"
      untrusted_origins:
        - "https://attacker.com"
        - "null"
```

---

## 5. `rate_limit` — API Endpoint Throttling Burst Plugin

Fires high-concurrency bursts of HTTP requests against public API endpoints to verify that rate limiting and DDoS protection middleware enforce HTTP `429 Too Many Requests`.

### Configuration Parameters

| Parameter | Type | Required? | Default | Description |
|:---|:---:|:---:|:---:|:---|
| `path` | String | **Required** | — | API endpoint path to test throttling. |
| `num_requests` | Integer | **Required** | `50` | Total number of burst requests to fire. |
| `method` | String | Optional | `"GET"` | HTTP Method (`GET`, `POST`, `PUT`, `DELETE`). |
| `concurrency` | Integer | Optional | `10` | Number of parallel worker goroutines. |
| `expected_status_code` | Integer | Optional | `429` | Expected HTTP status code indicating rate limit enforcement (`429`). |
| `expected_body_contains` | String | Optional | `""` | Body string assertion indicating throttling (e.g. `rate limit`). |

### YAML Example

```yaml
version: "1.0"
simulations:
  - name: "Search Query Endpoint Throttling"
    plugin: "rate_limit"
    config:
      path: "/api/search"
      method: "GET"
      num_requests: 50
      concurrency: 10
      expected_status_code: 429
```
