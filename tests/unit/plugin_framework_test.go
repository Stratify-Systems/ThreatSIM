package unit

import (
	"fmt"
	"sync"
	"testing"

	"github.com/suryatk2007/threatsim/pkg/plugins"
)

func TestContext_StateWriteback(t *testing.T) {
	t.Parallel()

	ctx := plugins.Context{
		State: plugins.NewStateMap(),
	}

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			key := fmt.Sprintf("key_%d", idx)
			val := fmt.Sprintf("val_%d", idx)
			ctx.SetState(key, val)
		}(i)
	}

	wg.Wait()

	for i := 0; i < 50; i++ {
		key := fmt.Sprintf("key_%d", i)
		val, ok := ctx.GetState(key)
		if !ok || val != fmt.Sprintf("val_%d", i) {
			t.Errorf("Expected state for key %q to be %q, got %q (ok=%v)", key, fmt.Sprintf("val_%d", i), val, ok)
		}
	}
}

func TestValidateConfig_Schemas(t *testing.T) {
	t.Parallel()

	// 1. Valid IDOR config
	validIDOR := map[string]interface{}{
		"auth_path":      "/auth/login",
		"user_a_payload": `{"user":"a"}`,
		"user_b_payload": `{"user":"b"}`,
		"target_path":    "/api/data/{id}",
	}
	if err := plugins.ValidateConfig("idor", validIDOR); err != nil {
		t.Errorf("Expected valid IDOR config to pass schema validation, got error: %v", err)
	}

	// 2. Invalid IDOR config (missing required target_path)
	invalidIDOR := map[string]interface{}{
		"auth_path":      "/auth/login",
		"user_a_payload": `{"user":"a"}`,
	}
	if err := plugins.ValidateConfig("idor", invalidIDOR); err == nil {
		t.Errorf("Expected invalid IDOR config to fail schema validation, got nil error")
	}

	// 3. Valid JWT Forge config
	validJWT := map[string]interface{}{
		"auth_path":       "/auth/login",
		"auth_payload":    `{"user":"test"}`,
		"token_json_path": "token",
		"target_path":     "/api/secrets",
	}
	if err := plugins.ValidateConfig("jwt_forge", validJWT); err != nil {
		t.Errorf("Expected valid JWT config to pass schema validation, got error: %v", err)
	}

	// 4. Invalid Bruteforce config (missing num_requests)
	invalidBrute := map[string]interface{}{
		"path":     "/login",
		"username": "admin",
	}
	if err := plugins.ValidateConfig("bruteforce", invalidBrute); err == nil {
		t.Errorf("Expected invalid bruteforce config to fail schema validation, got nil error")
	}
}
