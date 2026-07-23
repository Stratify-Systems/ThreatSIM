# ThreatSim — Plugin Development & Authoring Guide

This guide details how to extend ThreatSim by creating custom, stateful security validation plugins in Go.

---

## 📐 Plugin Architecture Overview

ThreatSim plugins decouple complex, multi-step attack scenarios from the core execution engine. Every plugin implements the standard `Plugin` interface defined in [`pkg/plugins/plugin.go`](file:///home/suryatk/ThreatSIM/pkg/plugins/plugin.go):

```go
type Plugin interface {
    // Name returns the unique string identifier for the plugin (e.g. "idor", "jwt_forge").
    Name() string

    // ValidateConfig verifies that the plugin configuration map matches the plugin's schema requirements.
    ValidateConfig(config map[string]interface{}) error

    // Execute runs the stateful security simulation and returns a SimulationResult.
    Execute(ctx context.Context, req types.Request, targetURL string, state map[string]string) types.SimulationResult
}
```

---

## 🛠️ Step-by-Step Tutorial: Writing a Custom Plugin (`header_audit`)

In this tutorial, we will build a custom plugin called `header_audit` that verifies whether a target endpoint enforces mandatory security headers (`Strict-Transport-Security`, `X-Content-Type-Options`, `X-Frame-Options`).

### Step 1: Create the Plugin File (`pkg/plugins/header_audit.go`)

Create `pkg/plugins/header_audit.go` and implement the `Plugin` interface:

```go
package plugins

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/suryatk2007/threatsim/pkg/types"
)

// HeaderAuditPlugin verifies mandatory security response headers.
type HeaderAuditPlugin struct{}

func (p *HeaderAuditPlugin) Name() string {
	return "header_audit"
}

func (p *HeaderAuditPlugin) ValidateConfig(config map[string]interface{}) error {
	path := ParseString(config, "path")
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("plugin %q requires a non-empty 'path' string parameter", p.Name())
	}
	return nil
}

func (p *HeaderAuditPlugin) Execute(ctx context.Context, req types.Request, targetURL string, state map[string]string) types.SimulationResult {
	start := time.Now()

	path := ParseString(req.Config, "path")
	method := ParseString(req.Config, "method")
	if method == "" {
		method = "GET"
	}

	fullURL := strings.TrimRight(targetURL, "/") + "/" + strings.TrimLeft(path, "/")

	httpReq, err := http.NewRequestWithContext(ctx, method, fullURL, nil)
	if err != nil {
		return types.SimulationResult{
			SimulationName: req.Name,
			Passed:         false,
			Duration:       time.Since(start),
			URL:            fullURL,
			Method:         method,
			Reason:         fmt.Errorf("failed to construct HTTP request: %w", err).Error(),
		}
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return types.SimulationResult{
			SimulationName: req.Name,
			Passed:         false,
			Duration:       time.Since(start),
			URL:            fullURL,
			Method:         method,
			Reason:         fmt.Errorf("HTTP request failed: %w", err).Error(),
		}
	}
	defer resp.Body.Close()

	// Validate Security Headers
	missing := []string{}
	if resp.Header.Get("X-Content-Type-Options") != "nosniff" {
		missing = append(missing, "X-Content-Type-Options: nosniff")
	}
	if resp.Header.Get("X-Frame-Options") == "" {
		missing = append(missing, "X-Frame-Options")
	}

	if len(missing) > 0 {
		return types.SimulationResult{
			SimulationName: req.Name,
			Passed:         false,
			Duration:       time.Since(start),
			URL:            fullURL,
			Method:         method,
			ActualResult:   fmt.Sprintf("Missing Headers: %s", strings.Join(missing, ", ")),
			Reason:         "Target endpoint failed mandatory HTTP security header audit",
		}
	}

	return types.SimulationResult{
		SimulationName: req.Name,
		Passed:         true,
		Duration:       time.Since(start),
		URL:            fullURL,
		Method:         method,
		ActualResult:   "All mandatory security headers enforced",
	}
}
```

---

### Step 2: Register the Plugin in `pkg/plugins/plugin.go`

Add your plugin instance to `RegisterPlugin()` inside `pkg/plugins/plugin.go`:

```go
func init() {
    RegisterPlugin(&IDORPlugin{})
    RegisterPlugin(&JWTForgePlugin{})
    RegisterPlugin(&BruteforcePlugin{})
    RegisterPlugin(&CORSAuditPlugin{})
    RegisterPlugin(&RateLimitPlugin{})
    RegisterPlugin(&HeaderAuditPlugin{}) // Add your new plugin here
}
```

---

### Step 3: Create Authoritative YAML Schema (`schemas/plugins/header_audit.yaml`)

Create `schemas/plugins/header_audit.yaml` so the ThreatSim AI engine (`pkg/ai/prompt.go`) and schema validator (`plugins.ValidateConfig`) automatically gain awareness of your new plugin:

```yaml
# Schema Template for Header Audit Plugin
name: "Security Header Verification"
plugin: "header_audit"
config:
  path: "/api/secure-endpoint"  # (Required) Path to audit
  method: "GET"                 # (Optional) HTTP Method (Defaults to GET)
```

---

### Step 4: Write Unit Tests (`tests/unit/header_audit_test.go`)

Create unit tests with a mock HTTP server to verify your plugin:

```go
package unit

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/suryatk2007/threatsim/pkg/plugins"
	"github.com/suryatk2007/threatsim/pkg/types"
)

func TestHeaderAuditPlugin(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	plugin := &plugins.HeaderAuditPlugin{}
	req := types.Request{
		Name: "Header Audit Unit Test",
		Config: map[string]interface{}{
			"path": "/api/secure-endpoint",
		},
	}

	res := plugin.Execute(context.Background(), req, ts.URL, make(map[string]string))
	if !res.Passed {
		t.Fatalf("Expected header audit plugin to pass, got failure reason: %s", res.Reason)
	}
}
```

Run unit tests to verify:
```bash
go test ./tests/unit/... -v -run TestHeaderAuditPlugin
```

---

## 🛡️ Plugin Development Best Practices

1. **Use Config Helpers**: Use `ParseString(config, key)`, `ParseInt(config, key, defaultVal)`, and `ParseBool(config, key, defaultVal)` from `pkg/plugins/plugin.go` for safe, type-asserted config extraction.
2. **Thread Safety**: Plugins are executed concurrently across parallel goroutines. Do not write to global mutable variables without mutex locks. Use the local `state` map passed into `Execute()`.
3. **Secret Masking**: Sensitive tokens extracted during authentication flows should be written to `state` (e.g. `state["token"] = bearerToken`) so ThreatSim's `MaskSensitiveData()` automatically scrubs them from terminal outputs and build logs.
