package ai

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/suryatk2007/threatsim/pkg/engine"
	"github.com/suryatk2007/threatsim/pkg/types"
)

// Generator orchestrates LLM prompt generation, output sanitization, schema validation, and self-correction retries.
type Generator struct {
	Client     *Client
	MaxRetries int
}

// NewGenerator constructs a new Generator instance.
func NewGenerator(client *Client) *Generator {
	return &Generator{
		Client:     client,
		MaxRetries: 2, // Allow up to 2 self-correction retries on validation failure
	}
}

// SanitizeYAML strips markdown code fences (```yaml ... ```) and conversational filler from LLM output.
func SanitizeYAML(raw string) string {
	cleaned := strings.TrimSpace(raw)

	// Remove markdown backtick code blocks (```yaml ... ``` or ``` ...)
	reCodeFence := regexp.MustCompile("(?s)```(?:yaml|yml)?\\s*\n?(.*?)\n?\\s*```")
	if matches := reCodeFence.FindStringSubmatch(cleaned); len(matches) > 1 {
		cleaned = strings.TrimSpace(matches[1])
	} else if strings.HasPrefix(cleaned, "```") {
		cleaned = strings.TrimPrefix(cleaned, "```yaml")
		cleaned = strings.TrimPrefix(cleaned, "```yml")
		cleaned = strings.TrimPrefix(cleaned, "```")
		cleaned = strings.TrimSuffix(cleaned, "```")
		cleaned = strings.TrimSpace(cleaned)
	}

	// Locate version: "1.0" or version: '1.0' if there is leading commentary text
	if idx := strings.Index(cleaned, "version:"); idx > 0 {
		cleaned = cleaned[idx:]
	}

	return strings.TrimSpace(cleaned)
}

// ValidateYAML verifies the YAML syntax and ThreatSim schema rules by attempting to parse it with the core engine.
func ValidateYAML(yamlContent string) (*types.SimulationDefinition, error) {
	tmpFile, err := os.CreateTemp("", "threatsim_gen_*.yaml")
	if err != nil {
		return nil, fmt.Errorf("failed to create temporary validation file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(yamlContent); err != nil {
		tmpFile.Close()
		return nil, fmt.Errorf("failed to write validation file: %w", err)
	}
	tmpFile.Close()

	eng := engine.New("http://localhost")
	def, err := eng.LoadSimulation(tmpFile.Name())
	if err != nil {
		return nil, fmt.Errorf("schema validation failed: %w", err)
	}

	return def, nil
}

// Generate sends user requirements to the LLM, sanitizes the response, validates it, and self-corrects if invalid.
func (g *Generator) Generate(ctx context.Context, description string) (string, *types.SimulationDefinition, error) {
	if strings.TrimSpace(description) == "" {
		return "", nil, fmt.Errorf("description cannot be empty")
	}

	systemPrompt := BuildSystemPrompt()
	messages := []ChatMessage{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: BuildUserPrompt(description)},
	}

	var lastErr error
	for attempt := 0; attempt <= g.MaxRetries; attempt++ {
		req := ChatCompletionRequest{
			Model:       g.Client.Config.Model,
			Messages:    messages,
			Temperature: 0.1, // Low temperature for deterministic schema generation
		}

		resp, err := g.Client.CreateChatCompletion(ctx, req)
		if err != nil {
			return "", nil, fmt.Errorf("LLM completion request failed: %w", err)
		}

		rawOutput := resp.Choices[0].Message.Content
		sanitizedYAML := SanitizeYAML(rawOutput)

		def, valErr := ValidateYAML(sanitizedYAML)
		if valErr == nil {
			return sanitizedYAML, def, nil // Success!
		}

		lastErr = valErr

		// If retries remain, append failure context to messages for self-correction retry
		if attempt < g.MaxRetries {
			messages = append(messages, ChatMessage{
				Role:    "assistant",
				Content: rawOutput,
			})
			messages = append(messages, ChatMessage{
				Role:    "user",
				Content: fmt.Sprintf("The generated YAML failed ThreatSim schema validation with error:\n%v\n\nPlease fix the YAML syntax/schema and return ONLY valid ThreatSim YAML starting with version: \"1.0\".", valErr),
			})
		}
	}

	return "", nil, fmt.Errorf("AI generated an invalid simulation after %d retries.\nValidation error: %v", g.MaxRetries, lastErr)
}
