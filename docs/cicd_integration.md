# ThreatSim — CI/CD Pipeline & Automation Integration Guide

ThreatSim is designed to execute as a non-blocking or blocking security gate directly within continuous integration and delivery (CI/CD) pipelines.

---

## 🚀 Supported CI/CD Integrations Overview

| Platform | Recommended Report Format | Native Feature |
|:---|:---:|:---|
| **GitHub Actions** | `sarif` | Native Security Code Scanning Tab Integration |
| **GitLab CI** | `junit` | Native Test Reports & Pipeline Artifacts |
| **Jenkins** | `junit` / `html` | JUnit Test Result Publishing Plugin |
| **Azure DevOps** | `junit` | Publish Test Results Task |
| **CircleCI** | `junit` | Store Test Results Artifacts |

---

## 🐙 1. GitHub Actions Integration (`.github/workflows/threatsim.yml`)

Integrate ThreatSim into GitHub Actions pipelines to run automated security behavior testing on every Pull Request or main branch commit. Exporting in `sarif` format uploads results directly to GitHub's **Security -> Code Scanning** alerts tab.

```yaml
name: ThreatSim Security Behavior Validation

on:
  push:
    branches: [ "main", "master" ]
  pull_request:
    branches: [ "main", "master" ]

jobs:
  security-validation:
    name: Run ThreatSim Security Audits
    runs-on: ubuntu-latest

    steps:
      - name: Checkout Code
        uses: actions/checkout@v4

      - name: Set up Go 1.24
        uses: actions/setup-go@v5
        with:
          go-version: '1.24'

      - name: Build ThreatSim Binary
        run: go build -o threatsim .

      - name: Start Staging API Application
        run: |
          go run examples/secure_mockserver/main.go &
          sleep 3

      - name: Execute ThreatSim Security Simulations
        run: |
          ./threatsim run -t http://localhost:8081 -f tests/simulations/jwt_test.yaml --output sarif --out-file reports/threatsim.sarif.json

      - name: Upload SARIF Security Audit Results to GitHub Security
        uses: github/codeql-action/upload-sarif@v3
        if: always()
        with:
          sarif_file: reports/threatsim.sarif.json
```

---

## 🦊 2. GitLab CI Integration (`.gitlab-ci.yml`)

Format ThreatSim test results as `junit` XML so GitLab CI natively displays passed and failed security controls inside Merge Request widgets.

```yaml
stages:
  - test

threatsim_security_audit:
  stage: test
  image: golang:1.24
  before_script:
    - go build -o threatsim .
    - go run examples/secure_mockserver/main.go &
    - sleep 2
  script:
    - ./threatsim run -t http://localhost:8081 -f tests/simulations/idor_test.yaml --output junit --out-file reports/threatsim_junit.xml
  artifacts:
    when: always
    reports:
      junit: reports/threatsim_junit.xml
```

---

## 🏗️ 3. Jenkins Pipeline Integration (`Jenkinsfile`)

```groovy
pipeline {
    agent any
    
    stages {
        stage('Build ThreatSim') {
            steps {
                sh 'go build -o threatsim .'
            }
        }
        
        stage('Run Security Simulations') {
            steps {
                sh './threatsim run -t http://localhost:8081 -f tests/simulations/cors_test.yaml --output junit --out-file reports/threatsim_junit.xml'
            }
        }
    }
    
    post {
        always {
            junit 'reports/threatsim_junit.xml'
        }
    }
}
```

---

## ⚡ 4. Non-Interactive AI Policy Generation & Execution in CI/CD

ThreatSim supports generating security validation suites on-the-fly from OpenAPI specifications or text prompts during build steps without requiring user interaction:

```bash
# Generate policy from OpenAPI spec and execute against staging target in one step:
./threatsim ai generate --openapi swagger.json -t https://staging.api.internal -r -y --timeout 30s
```

- **`-r, --run`**: Automatically executes generated simulations against `--target-url`.
- **`-y, --yes`**: Bypasses all interactive `[y/N]` confirmation prompts.
- **Exit Code `1`**: If any generated simulation fails against the staging target, ThreatSim exits with code `1`, causing the CI/CD pipeline step to fail safely.
