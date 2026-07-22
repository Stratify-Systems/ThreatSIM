package plugins

import (
	"github.com/suryatk2007/threatsim/pkg/plugins/utils/jwt"
	"github.com/suryatk2007/threatsim/pkg/types"
)

type JWTForgePlugin struct{}

func init() {
	Register(&JWTForgePlugin{})
}

func (p *JWTForgePlugin) Name() string {
	return "jwt_forge"
}

func (p *JWTForgePlugin) Description() string {
	return "Validates that the application properly verifies JWT signatures and rejects forged payloads (supports signature_tamper, alg_none, and weak_secret attack modes)."
}

func (p *JWTForgePlugin) Execute(simName string, ctx Context, config map[string]interface{}) []types.SimulationResult {
	cfg := jwt.JWTForgeConfig{
		BaseURL: ctx.TargetURL,
		Client:  ctx.Client,
	}

	cfg.AuthPath = ParseString(config, "auth_path")
	cfg.AuthPayload = ParseString(config, "auth_payload")
	cfg.TokenJSONPath = ParseString(config, "token_json_path")
	cfg.TargetPath = ParseString(config, "target_path")
	cfg.AttackMode = ParseString(config, "attack_mode")
	cfg.WeakSecret = ParseString(config, "weak_secret")
	cfg.ExpectedBodyContains = ParseString(config, "expected_body_contains")
	cfg.ExpectedStatusCode = ParseInt(config, "expected_status_code", 401)


	if claimsRaw, ok := config["forge_claims"].(map[string]interface{}); ok {
		cfg.ForgeClaims = claimsRaw
	} else if claimsRaw, ok := config["forge_claims"].(map[interface{}]interface{}); ok {
		cfg.ForgeClaims = make(map[string]interface{})
		for k, v := range claimsRaw {
			if strK, ok := k.(string); ok {
				cfg.ForgeClaims[strK] = v
			}
		}
	}

	cfg.StateSetter = &ctx
	return jwt.RunJWTForgeValidation(simName, cfg)
}
