# ThreatSIM CI/CD Pipeline Integration

This document illustrates how ThreatSIM fits into a real-world CI/CD pipeline for a software team.

> **Note:** The diagram below specifically illustrates **Method 1 (Active Network Traffic)** where ThreatSIM acts as a live attacker sending requests to a Staging App. For a comprehensive look at the expected final state of the project, including **Method 2 (Log Injection)**, please refer to [`EXPECTED_OUTCOME.md`](./EXPECTED_OUTCOME.md).

This illustrates a "Security as Code" testing pipeline, where your detection rules are tested via ThreatSIM immediately after deployment to a staging environment, but before going to production.

## Architecture Flow

```mermaid
graph TD
    %% Define Styles
    classDef pipeline fill:#e1f5fe,stroke:#01579b,stroke-width:2px;
    classDef staging fill:#fff3e0,stroke:#e65100,stroke-width:2px;
    classDef validation fill:#e8f5e9,stroke:#2e7d32,stroke-width:2px;
    classDef decision fill:#fff9c4,stroke:#f57f17,stroke-width:2px;
    classDef fail fill:#ffebee,stroke:#c62828,stroke-width:2px;
    classDef success fill:#f1f8e9,stroke:#2e7d32,stroke-width:2px;

    subgraph CI/CD Pipeline
        A([1. Commit App Code & Detection Rule]) --> B([2. Build & Deploy to Staging])
        B --> C[3. Trigger Security CI Job]
    end

    subgraph Staging Environment
        C -->|Runs CLI| D{4. ThreatSIM Execution}
        D -- "threatsim simulate brute_force" --> E[(Staging App/API)]
        E -- "Raw Traffic / App Logs" --> F[SIEM / Detection Engine]
    end

    subgraph Validation Phase
        D -->|Concurrent Wait| G[5. Wait for Log Processing]
        G --> H[6. CI Script Queries SIEM API]
        F -- "Generates Alerts" --> H
        H --> I{7. Alert Match Expected?}
    end

    I -- Yes --> J([8. Pipeline Passes: Deploy to Prod])
    I -- No --> K([8. Pipeline Fails: Block Deployment])

    %% Apply Styles
    class A,B,C pipeline;
    class D,E,F staging;
    class G,H validation;
    class I decision;
    class J success;
    class K fail;
```

## Breakdown of the Flow

1. **Commit:** A security engineer or developer writes a new detection rule (e.g., "Detect 20 failed logins in 30s") and pushes it to Git.
2. **Deploy to Staging:** Both the application and the new security monitoring rules are spun up in a staging/sandbox environment.
3. **ThreatSIM Execution:** The CI runner executes the ThreatSIM CLI targeting the newly deployed staging environment:
   ```bash
   threatsim simulate brute_force --target staging.api.internal --rate 10 --duration 10s
   ```
4. **Traffic Generation:** ThreatSIM blasts the staging application with simulated malicious traffic.
5. **Detection:** The staging app generates logs/metrics, and your security backend processes them.
6. **Validation:** The CI/CD script waits briefly, then makes an API call to your security backend (or ThreatSIM's alert dashboard) to check if an alert labeled "Brute Force Detected" was successfully generated for that specific target within the last 30 seconds.
7. **Decision Gate:**
   - **Success:** If the alert fired, your detection works. The deployment continues to production.
   - **Failure:** If no alert is found, either the app isn't logging correctly or the detection rule is broken. The pipeline fails immediately, preventing blind spots in the same way a failing unit test would block a software release.
