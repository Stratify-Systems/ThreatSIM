package cmd

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/suryatk2007/threatsim/pkg/ai"
	"github.com/suryatk2007/threatsim/pkg/engine"
)

var (
	aiPromptStr      string
	aiInputFile      string
	aiOpenAPIFile    string
	aiOutFile        string
	aiTargetURL      string
	aiRunImmediately bool
	aiAutoConfirm    bool
	aiTimeoutStr     string
	aiInsecure       bool
)

var aiGenerateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Convert natural language security requirements or OpenAPI specs into valid ThreatSim YAML simulations",
	Long:  `Generate translates plain-English security descriptions or OpenAPI/Swagger specification files into schema-validated ThreatSim YAML policy files using configured AI providers (Groq, OpenAI, Ollama).`,
	RunE: func(cmd *cobra.Command, args []string) error {
		var reqDescription string

		if strings.TrimSpace(aiOpenAPIFile) != "" {
			prompt, err := ai.ConvertOpenAPIToPrompt(aiOpenAPIFile)
			if err != nil {
				return fmt.Errorf("error processing OpenAPI spec file %q: %w", aiOpenAPIFile, err)
			}
			reqDescription = prompt
		} else if strings.TrimSpace(aiPromptStr) != "" {
			reqDescription = aiPromptStr
		} else if strings.TrimSpace(aiInputFile) != "" {
			data, err := os.ReadFile(aiInputFile)
			if err != nil {
				return fmt.Errorf("error reading input file %q: %w", aiInputFile, err)
			}
			reqDescription = string(data)
		} else {
			fmt.Println("Describe your application's security requirements.")
			fmt.Println("Press Ctrl+D when finished (or Ctrl+C to cancel):")
			fmt.Println("--------------------------------------------------")

			reader := bufio.NewReader(os.Stdin)
			var lines []string
			for {
				line, err := reader.ReadString('\n')
				if err != nil {
					if err == io.EOF {
						if len(line) > 0 {
							lines = append(lines, line)
						}
						break
					}
					return fmt.Errorf("error reading stdin: %w", err)
				}
				lines = append(lines, line)
			}
			reqDescription = strings.Join(lines, "")
		}

		reqDescription = strings.TrimSpace(reqDescription)
		if reqDescription == "" {
			return fmt.Errorf("no security requirements description provided")
		}

		cfg, err := ai.LoadConfigFromEnv()
		if err != nil {
			return fmt.Errorf("configuration error: %w; set THREATSIM_AI_API_KEY (see .env.example)", err)
		}

		client := ai.NewClient(*cfg)
		generator := ai.NewGenerator(client)

		fmt.Printf("\nGenerating simulations using %s (%s)...\n", strings.Title(cfg.Provider), cfg.Model)

		ctx := context.Background()
		yamlContent, def, err := generator.Generate(ctx, reqDescription)
		if err != nil {
			return fmt.Errorf("AI generated an invalid simulation: %w", err)
		}

		if strings.TrimSpace(aiOpenAPIFile) != "" {
			if err := ai.ValidateDefinitionAgainstOpenAPI(def, aiOpenAPIFile); err != nil {
				return fmt.Errorf("AI generated routes that do not match the supplied OpenAPI specification: %w", err)
			}
		}

		targetPath := aiOutFile
		if targetPath == "" {
			targetPath = "tests/simulations/generated.yaml"
		}

		// Ensure parent directory exists
		if dir := filepath.Dir(targetPath); dir != "." && dir != "" {
			if err := os.MkdirAll(dir, 0755); err != nil {
				return fmt.Errorf("error creating directory %q: %w", dir, err)
			}
		}

		if err := os.WriteFile(targetPath, []byte(yamlContent+"\n"), 0644); err != nil {
			return fmt.Errorf("error writing generated YAML to %q: %w", targetPath, err)
		}

		fmt.Printf("\n✓ Generated %d simulations\n", len(def.Simulations))
		fmt.Printf("✓ Saved to %s\n\n", targetPath)

		// Print complete policy YAML preview for review
		fmt.Println("\033[1m\033[36m==============================================================================\033[0m")
		fmt.Printf("\033[1m     GENERATED POLICY PREVIEW (\033[33m%s\033[0m\033[1m)\033[0m\n", targetPath)
		fmt.Println("\033[1m\033[36m==============================================================================\033[0m")
		fmt.Println("\033[1;33mWARNING: AI-generated endpoints may be incorrect or unavailable.\033[0m")
		fmt.Println("\033[1;33mVerify every path, HTTP method, authentication setting, and expected response before testing.\033[0m")
		fmt.Println(yamlContent)
		fmt.Println("\033[1m\033[36m==============================================================================\033[0m")
		fmt.Printf("\033[36m💡 Tip: You can open and edit '%s' in your editor before proceeding.\033[0m\n", targetPath)

		// --run requests execution; --yes only controls whether its review prompt is skipped.
		shouldRun := aiRunImmediately

		if shouldRun && !aiAutoConfirm {
			reader := bufio.NewReader(os.Stdin)
			fmt.Printf("\nHave you verified the generated endpoints and policy configuration? Proceed with execution? [y/N]: ")
			input, _ := reader.ReadString('\n')
			input = strings.TrimSpace(strings.ToLower(input))
			if input == "y" || input == "yes" {
				shouldRun = true
			}
		}

		if shouldRun {
			// Re-load simulation file in case user edited targetPath during review prompt
			loaderEng := engine.New("http://localhost")
			if reloadedDef, err := loaderEng.LoadSimulation(targetPath); err == nil {
				def = reloadedDef
			}

			finalURL := strings.TrimSpace(aiTargetURL)
			if finalURL == "" {
				if workspaceCfg, _ := loadConfig(); workspaceCfg != nil && workspaceCfg.TargetURL != "" {
					finalURL = workspaceCfg.TargetURL
				}
			}

			if finalURL == "" {
				reader := bufio.NewReader(os.Stdin)
				fmt.Print("Enter target URL (e.g. http://localhost:8080): ")
				input, _ := reader.ReadString('\n')
				finalURL = strings.TrimSpace(input)
			}

			if finalURL == "" {
				fmt.Println("Error: Target URL is required to execute simulations.")
				return fmt.Errorf("target URL is required to execute simulations")
			}

			timeoutDuration, err := time.ParseDuration(aiTimeoutStr)
			if err != nil {
				timeoutDuration = 15 * time.Second
			}

			fmt.Printf("\nExecuting ThreatSim validations against %s...\n\n", finalURL)

			eng := engine.New(finalURL, engine.WithTimeout(timeoutDuration), engine.WithInsecure(aiInsecure))
			report := eng.Execute(def)

			reporter := &engine.ConsoleReporter{}
			if err := reporter.Generate(os.Stdout, report, eng.State.GetAll()); err != nil {
				return fmt.Errorf("error generating execution report: %w", err)
			}

			if report.Failed > 0 {
				return fmt.Errorf("simulation validation failed: %d failed controls, %d setup errors", report.Failed, report.Errors)
			}
		}
		return nil
	},
}

func init() {
	aiCmd.AddCommand(aiGenerateCmd)

	aiGenerateCmd.Flags().StringVarP(&aiPromptStr, "prompt", "p", "", "Direct security requirements text prompt")
	aiGenerateCmd.Flags().StringVarP(&aiInputFile, "input", "i", "", "File path containing security requirements description")
	aiGenerateCmd.Flags().StringVarP(&aiOpenAPIFile, "openapi", "a", "", "File path to an OpenAPI/Swagger spec file (JSON or YAML)")
	aiGenerateCmd.Flags().StringVarP(&aiOutFile, "out-file", "o", "tests/simulations/generated.yaml", "Destination path for generated ThreatSim simulation file")

	aiGenerateCmd.Flags().StringVarP(&aiTargetURL, "target-url", "t", "", "Target application base URL for immediate simulation execution")
	aiGenerateCmd.Flags().BoolVarP(&aiRunImmediately, "run", "r", false, "Automatically run generated simulations against target after saving")
	aiGenerateCmd.Flags().BoolVarP(&aiAutoConfirm, "yes", "y", false, "Skip interactive review confirmation prompt (auto-confirm)")
	aiGenerateCmd.Flags().StringVar(&aiTimeoutStr, "timeout", "15s", "Default HTTP request timeout when running simulations")
	aiGenerateCmd.Flags().BoolVar(&aiInsecure, "insecure", false, "Skip TLS/SSL certificate verification when running simulations")
}
