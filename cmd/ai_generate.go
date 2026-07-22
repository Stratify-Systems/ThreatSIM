package cmd

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/suryatk2007/threatsim/pkg/ai"
)

var (
	aiPromptStr  string
	aiInputFile  string
	aiOutFile    string
)

var aiGenerateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Convert natural language security requirements into valid ThreatSim YAML simulations",
	Long:  `Generate translates plain-English application security descriptions into schema-validated ThreatSim YAML policy files using configured AI providers (Groq, OpenAI, Ollama).`,
	Run: func(cmd *cobra.Command, args []string) {
		var reqDescription string

		if strings.TrimSpace(aiPromptStr) != "" {
			reqDescription = aiPromptStr
		} else if strings.TrimSpace(aiInputFile) != "" {
			data, err := os.ReadFile(aiInputFile)
			if err != nil {
				fmt.Printf("Error reading input file %q: %v\n", aiInputFile, err)
				os.Exit(1)
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
		fmt.Printf("✓ Saved to %s\n", targetPath)
	},
}

func init() {
	aiCmd.AddCommand(aiGenerateCmd)

	aiGenerateCmd.Flags().StringVarP(&aiPromptStr, "prompt", "p", "", "Direct security requirements text prompt")
	aiGenerateCmd.Flags().StringVarP(&aiInputFile, "input", "i", "", "File path containing security requirements description")
	aiGenerateCmd.Flags().StringVarP(&aiOutFile, "out-file", "o", "tests/simulations/generated.yaml", "Destination path for generated ThreatSim simulation file")
}
