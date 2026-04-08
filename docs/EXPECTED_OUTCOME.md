# Expected Final Outcome: ThreatSIM in CI/CD

ThreatSIM is currently a **standalone telemetry simulation engine**, meaning it generates fake logs entirely within itself to test its own Detection and Risk engines. It does not touch your actual production servers yet.

By the end of this project (Phase 6), ThreatSIM will transform into a **Continuous Security Validation Platform**. It will run automatically inside your CI/CD pipelines (e.g., GitHub Actions, GitLab CI) to test your real application servers every time code is deployed.

If you are wondering: _"How will ThreatSIM actually attack or send logs to our real App Server in a CI/CD pipeline?"_ — there will be two primary methods.

---

## Method 1: The "Real Hacker" Method (Active Network Traffic)

> **Note:** The exact flow mapped out below is visually represented in the Architecture Diagram inside [`CI_CD_PIPELINE.md`](./CI_CD_PIPELINE.md).

In this method, ThreatSIM acts like a real attacker on your network. Instead of just generating mock logs, the plugins will be upgraded to send **real network requests**.

### How it works:

1. **Developer pushes code:** CI/CD spins up a temporary staging copy of your App Server.
2. **ThreatSIM launches:** ThreatSIM sends real HTTP requests mapping to an attack (e.g., rapidly sending `POST /login` with wrong passwords).
3. **App Server reacts:** Your real App Server naturally processes these requests, denies them, and writes failure logs to your security monitoring tool (like Datadog, Splunk, or an internal SIEM).
4. **The Validation (The Test):** The CI/CD script waits 10 seconds, then queries your security tool: _"Did you generate a CRITICAL alert for IP 10.1.2.3?"_
5. **Outcome:**
   - If **YES**: Your security rule works. Pipeline Passes. ✅
   - If **NO**: Your new code broke the security logging. Pipeline Fails. ❌

_Use this method when you want to prove that your application is correctly handling and logging bad traffic._

---

## Method 2: The "Fire Drill" Method (Log Injection)

Sometimes, you don't want to spam your App Server with millions of real HTTP requests (it might crash staging or skew analytics). Instead, ThreatSIM will skip the network attack and inject logs directly into your App Server's logging pipeline.

### How it works:

1. **ThreatSIM generates payloads:** ThreatSIM spins up and generates 1,000 highly realistic JSON "failed login" logs internally.
2. **Output Sinks (Log Forwarding):** Instead of keeping the logs to itself, ThreatSIM pushes those logs directly into your App Server's log aggregator (via an HTTP Webhook, Syslog, Kafka, or Elasticsearch).
3. **Security System wakes up:** Your App Server's security system reads the logs, assumes it is suddenly under a massive attack, and triggers an alarm.
4. **The Validation:** The CI/CD script checks if the alarm successfully fired.
5. **Outcome:** Passes or Fails the deployment based on whether the alarm triggered successfully.

_Use this method when you want to test whether your SIEM / Log Analyzer rules are working correctly without actually attacking the application._

---

## Summary of the Journey

| Phase         | Current State (Phases 1-5)                                | Expected Outcome (Phase 6)                                        |
| ------------- | --------------------------------------------------------- | ------------------------------------------------------------------- |
| **Execution** | Runs manually in your terminal or via local REST API.     | Runs automatically in GitHub Actions/GitLab CI on every code push.  |
| **Target**    | Attacks itself (internally generates mock telemetry).     | Attacks a real Staging App Server or injects logs into a real SIEM. |
| **Output**    | Writes to local PostgreSQL and prints to standard output. | Sends pass/fail signals to block or allow software deployments.     |
