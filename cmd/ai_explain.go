package cmd

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/suryatk2007/threatsim/pkg/ai"
)

var (
	aiExplainFile string
	aiExplainOut  string
)

var aiExplainCmd = &cobra.Command{
	Use:   "explain",
	Short: "Explain a ThreatSim security policy file in plain English markdown",
	Long:  `Explain analyzes an existing ThreatSim YAML policy file and uses AI to generate a detailed, human-readable GitHub-Flavored Markdown security audit report detailing tested attack vectors and security boundaries.`,
	Run: func(cmd *cobra.Command, args []string) {
		var yamlContent string

		if strings.TrimSpace(aiExplainFile) != "" {
			data, err := os.ReadFile(aiExplainFile)
			if err != nil {
				fmt.Printf("Error reading simulation file %q: %v\n", aiExplainFile, err)
				os.Exit(1)
			}
			yamlContent = string(data)
		} else {
			// Read from stdin if piped or no file flag provided
			stat, _ := os.Stdin.Stat()
			if (stat.Mode() & os.ModeCharDevice) == 0 {
				reader := bufio.NewReader(os.Stdin)
				data, err := io.ReadAll(reader)
				if err != nil {
					fmt.Printf("Error reading stdin: %v\n", err)
					os.Exit(1)
				}
				yamlContent = string(data)
			}
		}

		yamlContent = strings.TrimSpace(yamlContent)
		if yamlContent == "" {
			fmt.Println("Error: No simulation policy file provided. Pass -f <path> or pipe YAML content to stdin.")
			os.Exit(1)
		}

		cfg, err := ai.LoadConfigFromEnv()
		if err != nil {
			fmt.Printf("Configuration Error: %v\n", err)
			fmt.Println("Please set THREATSIM_AI_API_KEY in your .env file or environment (see .env.example).")
			os.Exit(1)
		}

		client := ai.NewClient(*cfg)
		explainer := ai.NewExplainer(client)

		fmt.Printf("Analyzing security policy using %s (%s)...\n\n", strings.Title(cfg.Provider), cfg.Model)

		explanation, err := explainer.Explain(context.Background(), yamlContent)
		if err != nil {
			fmt.Printf("Error generating policy explanation: %v\n", err)
			os.Exit(1)
		}

		if strings.TrimSpace(aiExplainOut) != "" {
			if err := os.WriteFile(aiExplainOut, []byte(explanation+"\n"), 0644); err != nil {
				fmt.Printf("Error writing explanation markdown to %q: %v\n", aiExplainOut, err)
				os.Exit(1)
			}
			fmt.Printf("✓ Security policy explanation saved to %s\n", aiExplainOut)
		} else {
			// Print clean Markdown text directly
			fmt.Println(explanation)
		}
	},
}

func init() {
	aiCmd.AddCommand(aiExplainCmd)

	aiExplainCmd.Flags().StringVarP(&aiExplainFile, "file", "f", "", "Path to the ThreatSim simulation YAML file to explain")
	aiExplainCmd.Flags().StringVarP(&aiExplainOut, "out-file", "o", "", "Optional target file path to save markdown explanation report")
}

