package ai

import (
	"context"
	"fmt"
	"strings"

	"github.com/suryatk2007/threatsim/pkg/types"
)

// Explainer analyzes ThreatSim simulation policy files and generates human-readable security documentation.
type Explainer struct {
	Client *Client
}

// NewExplainer constructs a new Explainer instance.
func NewExplainer(client *Client) *Explainer {
	return &Explainer{
		Client: client,
	}
}

// Explain converts a ThreatSim YAML simulation policy into a clear, professional security analysis report.
func (e *Explainer) Explain(ctx context.Context, yamlContent string) (string, error) {
	if strings.TrimSpace(yamlContent) == "" {
		return "", fmt.Errorf("YAML simulation content cannot be empty")
	}

	def, err := ValidateYAML(yamlContent)
	if err != nil {
		return "", fmt.Errorf("cannot explain invalid simulation YAML: %w", err)
	}

	systemPrompt := `You are ThreatSim Security Analyst. Your task is to analyze ThreatSim security simulation YAML files and produce a clear, highly professional security explanation report formatted as clean GitHub-Flavored Markdown text.

Use proper Markdown headers (#, ##, ###), bullet lists (-), bold text (**text**), and inline code blocks ('code').

Structure your markdown explanation with the following sections:
# ThreatSim Security Policy Explanation


## 1. Executive Summary & Policy Goal
## 2. Detailed Breakdown of Security Simulations
(For each simulation, detail: Sim Name, Attack Vector / Plugin Used, Method & Target URL/Path, Pass Criteria)
## 3. Target Security Control Boundary
## 4. Audit & Compliance Alignment (e.g. OWASP API Top 10, RBAC, JWT verification)`


	userPrompt := fmt.Sprintf("Analyze and explain the following ThreatSim security policy file (%d total simulations defined):\n\n%s", len(def.Simulations), yamlContent)

	req := ChatCompletionRequest{
		Model: e.Client.Config.Model,
		Messages: []ChatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		Temperature: 0.2,
	}

	resp, err := e.Client.CreateChatCompletion(ctx, req)
	if err != nil {
		return "", fmt.Errorf("AI explanation request failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("AI returned an empty explanation")
	}

	return strings.TrimSpace(resp.Choices[0].Message.Content), nil
}

// SummarizeDefinition provides a quick non-AI inline summary of a simulation definition.
func SummarizeDefinition(def *types.SimulationDefinition) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("ThreatSim Security Policy (Version %s)\n", def.Version))
	sb.WriteString(fmt.Sprintf("Total Simulations: %d\n", len(def.Simulations)))
	for i, sim := range def.Simulations {
		if sim.Plugin != "" {
			sb.WriteString(fmt.Sprintf("  %d. [Plugin: %s] %s\n", i+1, sim.Plugin, sim.Name))
		} else {
			sb.WriteString(fmt.Sprintf("  %d. [HTTP: %s] %s\n", i+1, sim.Request.Method, sim.Name))
		}
	}
	return sb.String()
}
