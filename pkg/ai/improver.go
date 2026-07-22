package ai

import (
	"context"
	"fmt"
	"strings"

	"github.com/suryatk2007/threatsim/pkg/types"
	"gopkg.in/yaml.v3"
)

// Improver analyzes existing ThreatSim security policy files and generates complementary simulation entries to close security gaps.
type Improver struct {
	Client    *Client
	Generator *Generator
}

// NewImprover constructs a new Improver instance.
func NewImprover(client *Client) *Improver {
	return &Improver{
		Client:    client,
		Generator: NewGenerator(client),
	}
}

// Improve takes an existing simulation YAML policy, identifies missing security checks, and generates an expanded, schema-validated policy suite.
func (imp *Improver) Improve(ctx context.Context, existingYAML string) (string, *types.SimulationDefinition, error) {
	if strings.TrimSpace(existingYAML) == "" {
		return "", nil, fmt.Errorf("existing simulation policy content cannot be empty")
	}

	origDef, err := ValidateYAML(existingYAML)
	if err != nil {
		return "", nil, fmt.Errorf("cannot improve invalid simulation YAML: %w", err)
	}

	improvementPrompt := fmt.Sprintf(`Analyze the following existing ThreatSim security simulation policy file (%d simulations defined):

%s

Your task is to identify MISSING security boundary checks (such as unthrottled endpoints, missing IDOR isolation tests, missing JWT forgery checks, or missing CORS origin audits) and generate ADDITIONAL complementary ThreatSim YAML simulation entries to close these gaps.

Output ONLY valid ThreatSim YAML starting with version: "1.0" containing the new complementary simulation entries.`, len(origDef.Simulations), existingYAML)

	_, newDef, err := imp.Generator.Generate(ctx, improvementPrompt)

	if err != nil {
		return "", nil, fmt.Errorf("AI policy improvement generation failed: %w", err)
	}

	// Merge new simulations into existing definition
	mergedDef := &types.SimulationDefinition{
		Version:     origDef.Version,
		Simulations: append(origDef.Simulations, newDef.Simulations...),
	}

	mergedBytes, err := yaml.Marshal(mergedDef)
	if err != nil {
		return "", nil, fmt.Errorf("failed to marshal merged simulation policy: %w", err)
	}

	return string(mergedBytes), mergedDef, nil
}
