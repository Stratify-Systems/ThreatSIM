package plugins

import (
	"github.com/suryatk2007/threatsim/pkg/plugins/utils/rate_limit"
	"github.com/suryatk2007/threatsim/pkg/types"
)

type RateLimitPlugin struct{}

func init() {
	Register(&RateLimitPlugin{})
}

func (p *RateLimitPlugin) Name() string {
	return "rate_limit"
}

func (p *RateLimitPlugin) Description() string {
	return "Tests API endpoint throttling for public endpoints (search, contact forms, checkout) with configurable concurrency bursts."
}

func (p *RateLimitPlugin) Execute(simName string, ctx Context, config map[string]interface{}) []types.SimulationResult {
	cfg := rate_limit.RateLimitConfig{
		BaseURL: ctx.TargetURL,
		Client:  ctx.Client,
	}

	cfg.Path = ParseString(config, "path")
	cfg.Method = ParseString(config, "method")
	cfg.Body = ParseString(config, "body")
	cfg.ExpectedBodyContains = ParseString(config, "expected_body_contains")
	cfg.NumRequests = ParseInt(config, "num_requests", 20)
	cfg.Concurrency = ParseInt(config, "concurrency", 10)
	cfg.ExpectedStatusCode = ParseInt(config, "expected_status_code", 429)

	if headersRaw, ok := config["headers"].(map[string]interface{}); ok {
		cfg.Headers = make(map[string]string)
		for k, val := range headersRaw {
			if strVal, ok := val.(string); ok {
				cfg.Headers[k] = strVal
			}
		}
	} else if headersRaw, ok := config["headers"].(map[interface{}]interface{}); ok {
		cfg.Headers = make(map[string]string)
		for k, val := range headersRaw {
			if strK, ok := k.(string); ok {
				if strV, ok := val.(string); ok {
					cfg.Headers[strK] = strV
				}
			}
		}
	}


	return rate_limit.RunRateLimitValidation(simName, cfg)
}
