package plugins

import (
	"github.com/suryatk2007/threatsim/pkg/plugins/utils"
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
	var cfg utils.IDORConfig
	cfg.BaseURL = ctx.TargetURL
	cfg.Client = ctx.Client

	// Parse configuration
	if v, ok := config["auth_path"].(string); ok { cfg.AuthPath = v }
	if v, ok := config["user_a_payload"].(string); ok { cfg.UserAPayload = v }
	if v, ok := config["user_b_payload"].(string); ok { cfg.UserBPayload = v }
	if v, ok := config["token_json_path"].(string); ok { cfg.TokenJSONPath = v }
	if v, ok := config["id_json_path"].(string); ok { cfg.IDJSONPath = v }
	if v, ok := config["target_path"].(string); ok { cfg.TargetPath = v }
	if v, ok := config["expected_body_contains"].(string); ok { cfg.ExpectedBodyContains = v }

	if escRaw, ok := config["expected_status_code"]; ok {
		switch v := escRaw.(type) {
		case int:
			cfg.ExpectedStatusCode = v
		case float64:
			cfg.ExpectedStatusCode = int(v)
		}
	}

	if cfg.AuthPath == "" || cfg.UserAPayload == "" || cfg.UserBPayload == "" || cfg.TargetPath == "" {
		return []types.SimulationResult{{
			SimulationName: simName,
			Passed:         false,
			ExpectedResult: "Valid plugin configuration",
			ActualResult:   "Missing fields",
			Reason:         "Missing required fields (auth_path, user_a_payload, user_b_payload, target_path)",
		}}
	}

	if cfg.ExpectedStatusCode == 0 {
		cfg.ExpectedStatusCode = 403 // Default to expecting Forbidden
	}

	return utils.RunIDORValidation(simName, cfg)
}
