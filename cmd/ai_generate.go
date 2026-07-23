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
	Run: func(cmd *cobra.Command, args []string) {
		var reqDescription string
		isInteractive := false

		if strings.TrimSpace(aiOpenAPIFile) != "" {
			prompt, err := ai.ConvertOpenAPIToPrompt(aiOpenAPIFile)
			if err != nil {
				fmt.Printf("Error processing OpenAPI spec file %q: %v\n", aiOpenAPIFile, err)
				os.Exit(1)
			}
			reqDescription = prompt
		} else if strings.TrimSpace(aiPromptStr) != "" {
			reqDescription = aiPromptStr
		} else if strings.TrimSpace(aiInputFile) != "" {
			data, err := os.ReadFile(aiInputFile)
			if err != nil {
				fmt.Printf("Error reading input file %q: %v\n", aiInputFile, err)
				os.Exit(1)
			}
			reqDescription = string(data)
		} else {
			isInteractive = true
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
					fmt.Printf("Error reading stdin: %v\n", err)
					os.Exit(1)
				}
				lines = append(lines, line)
			}
			reqDescription = strings.Join(lines, "")
		}

		reqDescription = strings.TrimSpace(reqDescription)
		if reqDescription == "" {
			fmt.Println("Error: No security requirements description provided.")
			os.Exit(1)
		}

		cfg, err := ai.LoadConfigFromEnv()
		if err != nil {
			fmt.Printf("Configuration Error: %v\n", err)
			fmt.Println("Please set THREATSIM_AI_API_KEY in your .env file or environment (see .env.example).")
			os.Exit(1)
		}

		client := ai.NewClient(*cfg)
		generator := ai.NewGenerator(client)

		fmt.Printf("\nGenerating simulations using %s (%s)...\n", strings.Title(cfg.Provider), cfg.Model)

		ctx := context.Background()
		yamlContent, def, err := generator.Generate(ctx, reqDescription)
		if err != nil {
			fmt.Printf("\nAI generated an invalid simulation.\n\n%v\n", err)
			os.Exit(1)
		}

		targetPath := aiOutFile
		if targetPath == "" {
			targetPath = "tests/simulations/generated.yaml"
		}

		// Ensure parent directory exists
		if dir := filepath.Dir(targetPath); dir != "." && dir != "" {
			if err := os.MkdirAll(dir, 0755); err != nil {
				fmt.Printf("Error creating directory %q: %v\n", dir, err)
				os.Exit(1)
			}
		}

		if err := os.WriteFile(targetPath, []byte(yamlContent+"\n"), 0644); err != nil {
			fmt.Printf("Error writing generated YAML to %q: %v\n", targetPath, err)
			os.Exit(1)
		}

		fmt.Printf("\n✓ Generated %d simulations\n", len(def.Simulations))
		fmt.Printf("✓ Saved to %s\n\n", targetPath)

		// Print summary preview
		fmt.Println("--------------------------------------------------")
		fmt.Print(ai.SummarizeDefinition(def))
		fmt.Println("--------------------------------------------------")

		// Determine if we should auto-run / prompt to run
		shouldRun := aiRunImmediately || aiAutoConfirm

		if !shouldRun && (isInteractive || cmd.Flags().Changed("target-url") || aiRunImmediately) {
			reader := bufio.NewReader(os.Stdin)
			fmt.Print("\nDo you want to run these simulations now against a target application? [y/N]: ")
			input, _ := reader.ReadString('\n')
			input = strings.TrimSpace(strings.ToLower(input))
			if input == "y" || input == "yes" {
				shouldRun = true
			}
		}

		if shouldRun {
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
				os.Exit(1)
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
				fmt.Printf("Error generating execution report: %v\n", err)
				os.Exit(1)
			}

			if report.Failed > 0 {
				os.Exit(1)
			}
		}
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
	aiGenerateCmd.Flags().BoolVarP(&aiAutoConfirm, "yes", "y", false, "Skip interactive review confirmation prompt")
	aiGenerateCmd.Flags().StringVar(&aiTimeoutStr, "timeout", "15s", "Default HTTP request timeout when running simulations")
	aiGenerateCmd.Flags().BoolVar(&aiInsecure, "insecure", false, "Skip TLS/SSL certificate verification when running simulations")
}
