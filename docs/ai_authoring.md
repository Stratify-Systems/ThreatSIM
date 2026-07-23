# ThreatSim — AI Policy Engineering & LLM Integration Guide

ThreatSim features a vendor-agnostic AI authoring and analysis suite under the `threatsim ai` subcommand group.

---

## 🏛️ AI Engine Architecture & Design Principles

1. **Strict Deterministic Boundary**: The AI engine is purely an authoring assistant. ThreatSim's Go engine ([`pkg/engine/`](file:///home/suryatk/ThreatSIM/pkg/engine/)) remains the sole arbiter of security validation.
2. **Schema-Driven Prompting**: System prompts dynamically load ThreatSim's authoritative YAML schemas from [`schemas/`](file:///home/suryatk/ThreatSIM/schemas/) so generated policies always use valid plugin definitions.
3. **Automated Self-Correction Retry Loop**: If LLM generation fails ThreatSim's strict YAML validation (`engine.LoadSimulation()`), ThreatSim automatically feeds the exact Go error back to the LLM and requests a corrected attempt (up to 2 retries).

---

## 🔑 Environment Configuration (`.env`)

ThreatSim reads secrets from `.env` files using `godotenv`. Create a `.env` file in your workspace root:

```bash
cp .env.example .env
```

### Supported Provider Configurations

#### 1. Groq API (Default - High Speed & Free Tier)
```ini
THREATSIM_AI_PROVIDER=groq
THREATSIM_AI_BASE_URL=https://api.groq.com/openai/v1
THREATSIM_AI_MODEL=llama-3.3-70b-versatile
THREATSIM_AI_API_KEY=gsk_your_groq_api_key_here
```

#### 2. OpenAI API
```ini
THREATSIM_AI_PROVIDER=openai
THREATSIM_AI_BASE_URL=https://api.openai.com/v1
THREATSIM_AI_MODEL=gpt-4o
THREATSIM_AI_API_KEY=sk-proj-your_openai_api_key_here
```

#### 3. Offline Local Ollama (Air-Gapped Privacy)
Run Ollama locally without sending data to external cloud APIs:
```bash
ollama serve
ollama pull llama3
```

Set your `.env`:
```ini
THREATSIM_AI_PROVIDER=ollama
THREATSIM_AI_BASE_URL=http://localhost:11434/v1
THREATSIM_AI_MODEL=llama3
THREATSIM_AI_API_KEY=ollama
```

---

## 🚀 AI Commands Deep Dive

### 1. `threatsim ai generate`
Translates natural language descriptions or OpenAPI specifications into schema-validated ThreatSim YAML files.

- **Interactive Mode**:
  ```bash
  ./threatsim ai generate
  # Type your requirements, then press Ctrl+D
  ```

- **Inline Prompt String (`-p`)**:
  ```bash
  ./threatsim ai generate -p "Users must only access their own invoices. Lock login after 5 failed attempts."
  ```

- **OpenAPI Specification Import (`-a, --openapi`)**:
  ```bash
  ./threatsim ai generate --openapi swagger.json -o tests/simulations/api_suite.yaml
  ```

- **Immediate Auto-Execution (`-r, -t, -y`)**:
  ```bash
  ./threatsim ai generate -p "Validate CORS security on /api/data" -t http://localhost:8081 --run --yes
  ```

---

### 2. `threatsim ai explain`
Analyzes an existing ThreatSim YAML policy and generates a plain-English, GitHub-Flavored Markdown security audit report.

- **Terminal Output (ANSI Markdown Renderer)**:
  ```bash
  ./threatsim ai explain -f tests/simulations/jwt_test.yaml
  ```

- **Save Markdown Report to File (`-o`)**:
  ```bash
  ./threatsim ai explain -f tests/simulations/jwt_test.yaml -o reports/jwt_audit_report.md
  ```

---

### 3. `threatsim ai improve`
Analyzes an existing policy file for missing attack vector coverage (IDOR, CORS, Rate Limiting, JWT forgery), generates complementary simulation entries, schema-validates them, and merges them into an expanded policy suite.

```bash
./threatsim ai improve -f tests/simulations/cors_test.yaml -o tests/simulations/improved_suite.yaml
```

---

## 🤖 Automated PR Review Bot Integration

You can integrate `threatsim ai explain` into GitHub PR workflows to automatically comment plain-English security summaries on pull requests:

```yaml
- name: Generate Security Explanation Report
  run: |
    ./threatsim ai explain -f tests/simulations/generated.yaml -o report.md

- name: Comment Security Report on PR
  uses: marooned/actions-comment-pull-request@v1
  with:
    filePath: report.md
```
