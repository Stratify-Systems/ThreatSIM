package unit

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/suryatk2007/threatsim/pkg/plugins/utils/rate_limit"
)

func TestRunRateLimitValidation_ThrottledEndpoint(t *testing.T) {
	t.Parallel()

	var reqCount int
	var mu sync.Mutex

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		reqCount++
		current := reqCount
		mu.Unlock()

		if current > 5 {
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte(`{"error":"too many requests"}`))
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"result":"search results"}`))
	}))
	defer ts.Close()

	cfg := rate_limit.RateLimitConfig{
		BaseURL:            ts.URL,
		Path:               "/api/search",
		NumRequests:        20,
		Concurrency:        5,
		ExpectedStatusCode: 429,
		Client:             ts.Client(),
	}

	results := rate_limit.RunRateLimitValidation("Rate Limit Audit Test", cfg)
	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}

	if !results[0].Passed {
		t.Errorf("Expected rate limit test to pass on throttled server, failed with reason: %s", results[0].Reason)
	}
}

func TestRunRateLimitValidation_UnthrottledEndpoint(t *testing.T) {
	t.Parallel()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// VULNERABLE! Server never rate-limits requests
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"result":"unthrottled"}`))
	}))
	defer ts.Close()

	cfg := rate_limit.RateLimitConfig{
		BaseURL:            ts.URL,
		Path:               "/api/search",
		NumRequests:        15,
		Concurrency:        5,
		ExpectedStatusCode: 429,
		Client:             ts.Client(),
	}

	results := rate_limit.RunRateLimitValidation("Rate Limit Audit Test", cfg)
	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}

	if results[0].Passed {
		t.Errorf("Expected rate limit test to fail on unthrottled server, but passed")
	}
}
