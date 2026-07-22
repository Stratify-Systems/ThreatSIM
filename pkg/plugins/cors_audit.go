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

	cfg.Path = ParseString(config, "path")
	cfg.Method = ParseString(config, "method")
	cfg.CustomOrigin = ParseString(config, "custom_origin")
	cfg.TestNullOrigin = ParseBool(config, "test_null_origin", false)
	cfg.ExpectedAllowCredentials = ParseBool(config, "expected_allow_credentials", false)
	cfg.ExpectedStatusCode = ParseInt(config, "expected_status_code", 0)


	return cors.RunCORSAuditValidation(simName, cfg)
}
