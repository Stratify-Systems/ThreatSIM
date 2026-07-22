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

	if v, ok := config["path"].(string); ok { cfg.Path = v }
	if v, ok := config["method"].(string); ok { cfg.Method = v }
	if v, ok := config["body"].(string); ok { cfg.Body = v }
	if v, ok := config["expected_body_contains"].(string); ok { cfg.ExpectedBodyContains = v }

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

	if reqRaw, ok := config["num_requests"]; ok {
		switch v := reqRaw.(type) {
		case int:
			cfg.NumRequests = v
		case float64:
			cfg.NumRequests = int(v)
		}
	}

	if concRaw, ok := config["concurrency"]; ok {
		switch v := concRaw.(type) {
		case int:
			cfg.Concurrency = v
		case float64:
			cfg.Concurrency = int(v)
		}
	}

	if escRaw, ok := config["expected_status_code"]; ok {
		switch v := escRaw.(type) {
		case int:
			cfg.ExpectedStatusCode = v
		case float64:
			cfg.ExpectedStatusCode = int(v)
		}
	}

	return rate_limit.RunRateLimitValidation(simName, cfg)
}
