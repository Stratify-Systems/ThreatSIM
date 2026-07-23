package unit

import (
	"net/http"
	"testing"
	"time"

	"github.com/suryatk2007/threatsim/pkg/plugins/utils/auth"
	"github.com/suryatk2007/threatsim/pkg/plugins/utils/cors"
	"github.com/suryatk2007/threatsim/pkg/plugins/utils/idor"
	"github.com/suryatk2007/threatsim/pkg/plugins/utils/jwt"
	"github.com/suryatk2007/threatsim/pkg/plugins/utils/rate_limit"
)

func TestUnreachableTarget_AllPluginsReturnError(t *testing.T) {
	unreachableURL := "http://127.0.0.1:59999"
	client := &http.Client{Timeout: 500 * time.Millisecond}

	t.Run("CORS Audit Plugin Unreachable Target", func(t *testing.T) {
		results := cors.RunCORSAuditValidation("Unreachable CORS Test", cors.CORSAuditConfig{
			BaseURL: unreachableURL,
			Path:    "/api/data",
			Client:  client,
		})

		if len(results) == 0 {
			t.Fatalf("Expected results from CORS audit runner")
		}
		res := results[0]
		if res.Passed {
			t.Errorf("Expected Passed=false when target server is unreachable, got true")
		}
		if !res.IsError {
			t.Errorf("Expected IsError=true when target server is unreachable, got false")
		}
	})

	t.Run("Rate Limit Plugin Unreachable Target", func(t *testing.T) {
		results := rate_limit.RunRateLimitValidation("Unreachable Rate Limit Test", rate_limit.RateLimitConfig{
			BaseURL:     unreachableURL,
			Path:        "/api/search",
			NumRequests: 5,
			Concurrency: 2,
			Client:      client,
		})

		if len(results) == 0 {
			t.Fatalf("Expected results from Rate Limit runner")
		}
		res := results[0]
		if res.Passed {
			t.Errorf("Expected Passed=false when target server is unreachable, got true")
		}
		if !res.IsError {
			t.Errorf("Expected IsError=true when target server is unreachable, got false")
		}
	})

	t.Run("IDOR Plugin Unreachable Target", func(t *testing.T) {
		results := idor.RunIDORValidation("Unreachable IDOR Test", idor.IDORConfig{
			BaseURL:       unreachableURL,
			AuthPath:      "/auth/login",
			UserAPayload:  `{"username":"admin"}`,
			UserBPayload:  `{"username":"guest"}`,
			TokenJSONPath: "token",
			TargetPath:    "/api/users/{id}/private-data",
			Client:        client,
		})

		if len(results) == 0 {
			t.Fatalf("Expected results from IDOR runner")
		}
		res := results[0]
		if res.Passed {
			t.Errorf("Expected Passed=false when target server is unreachable, got true")
		}
		if !res.IsError {
			t.Errorf("Expected IsError=true when target server is unreachable, got false")
		}
	})

	t.Run("JWT Forge Plugin Unreachable Target", func(t *testing.T) {
		results := jwt.RunJWTForgeValidation("Unreachable JWT Test", jwt.JWTForgeConfig{
			BaseURL:       unreachableURL,
			AuthPath:      "/auth/login",
			AuthPayload:   `{"username":"guest"}`,
			TokenJSONPath: "token",
			TargetPath:    "/api/admin/secrets",
			Client:        client,
		})

		if len(results) == 0 {
			t.Fatalf("Expected results from JWT runner")
		}
		res := results[0]
		if res.Passed {
			t.Errorf("Expected Passed=false when target server is unreachable, got true")
		}
		if !res.IsError {
			t.Errorf("Expected IsError=true when target server is unreachable, got false")
		}
	})

	t.Run("Auth Helper Unreachable Target", func(t *testing.T) {
		_, err := auth.AuthenticateAndExtract(client, auth.AuthConfig{
			URL:           unreachableURL + "/auth/login",
			Payload:       `{"username":"test"}`,
			TokenJSONPath: "token",
		})
		if err == nil {
			t.Errorf("Expected error when authenticating to unreachable server, got nil")
		}
	})
}
