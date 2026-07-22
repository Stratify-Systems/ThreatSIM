package plugins

import (
	"github.com/suryatk2007/threatsim/pkg/plugins/utils/idor"
	"github.com/suryatk2007/threatsim/pkg/types"
)

func init() {
	Register(&IDORPlugin{})
}

type IDORPlugin struct{}

func (p *IDORPlugin) Name() string {
	return "idor"
}

func (p *IDORPlugin) Description() string {
	return "Validates Insecure Direct Object Reference (IDOR) by authenticating as two users and attempting cross-tenant access."
}

func (p *IDORPlugin) Execute(simName string, ctx Context, config map[string]interface{}) []types.SimulationResult {
	var cfg idor.IDORConfig
	cfg.BaseURL = ctx.TargetURL
	cfg.Client = ctx.Client

	// Parse configuration
	cfg.AuthPath = ParseString(config, "auth_path")
	cfg.UserAPayload = ParseString(config, "user_a_payload")
	cfg.UserBPayload = ParseString(config, "user_b_payload")
	cfg.TokenJSONPath = ParseString(config, "token_json_path")
	cfg.IDJSONPath = ParseString(config, "id_json_path")
	cfg.TargetMethod = ParseString(config, "target_method")
	cfg.TargetPath = ParseString(config, "target_path")
	cfg.TargetPayload = ParseString(config, "target_payload")
	cfg.ExpectedBodyContains = ParseString(config, "expected_body_contains")
	cfg.ExpectedStatusCode = ParseInt(config, "expected_status_code", 403)

	if cfg.AuthPath == "" || cfg.UserAPayload == "" || cfg.UserBPayload == "" || cfg.TargetPath == "" {
		return []types.SimulationResult{{
			SimulationName: simName,
			Passed:         false,
			ExpectedResult: "Valid plugin configuration",
			ActualResult:   "Missing fields",
			Reason:         "Missing required fields (auth_path, user_a_payload, user_b_payload, target_path)",
		}}
	}


	return idor.RunIDORValidation(simName, cfg)
}
