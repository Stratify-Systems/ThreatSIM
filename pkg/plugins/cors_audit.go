package plugins

import (
	"github.com/suryatk2007/threatsim/pkg/plugins/utils/cors"
	"github.com/suryatk2007/threatsim/pkg/types"
)

type CORSAuditPlugin struct{}

func init() {
	Register(&CORSAuditPlugin{})
}

func (p *CORSAuditPlugin) Name() string {
	return "cors_audit"
}

func (p *CORSAuditPlugin) Description() string {
	return "Audits API endpoints for insecure CORS configurations (untrusted origin reflection, null origin reflection, wildcard * with credentials)."
}

func (p *CORSAuditPlugin) Execute(simName string, ctx Context, config map[string]interface{}) []types.SimulationResult {
	cfg := cors.CORSAuditConfig{
		BaseURL: ctx.TargetURL,
		Client:  ctx.Client,
	}

	if v, ok := config["path"].(string); ok { cfg.Path = v }
	if v, ok := config["method"].(string); ok { cfg.Method = v }
	if v, ok := config["custom_origin"].(string); ok { cfg.CustomOrigin = v }
	if v, ok := config["test_null_origin"].(bool); ok { cfg.TestNullOrigin = v }
	if v, ok := config["expected_allow_credentials"].(bool); ok { cfg.ExpectedAllowCredentials = v }

	if escRaw, ok := config["expected_status_code"]; ok {
		switch v := escRaw.(type) {
		case int:
			cfg.ExpectedStatusCode = v
		case float64:
			cfg.ExpectedStatusCode = int(v)
		}
	}

	return cors.RunCORSAuditValidation(simName, cfg)
}
