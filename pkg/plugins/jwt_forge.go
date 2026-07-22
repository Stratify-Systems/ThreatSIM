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

	if v, ok := config["auth_path"].(string); ok { cfg.AuthPath = v }
	if v, ok := config["auth_payload"].(string); ok { cfg.AuthPayload = v }
	if v, ok := config["token_json_path"].(string); ok { cfg.TokenJSONPath = v }
	if v, ok := config["target_path"].(string); ok { cfg.TargetPath = v }
	if v, ok := config["attack_mode"].(string); ok { cfg.AttackMode = v }
	if v, ok := config["weak_secret"].(string); ok { cfg.WeakSecret = v }
	if v, ok := config["expected_body_contains"].(string); ok { cfg.ExpectedBodyContains = v }

	if escRaw, ok := config["expected_status_code"]; ok {
		switch v := escRaw.(type) {
		case int:
			cfg.ExpectedStatusCode = v
		case float64:
			cfg.ExpectedStatusCode = int(v)
		}
	}

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

	return jwt.RunJWTForgeValidation(simName, cfg)
}
