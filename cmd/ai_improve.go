package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/suryatk2007/threatsim/pkg/ai"
)

var (
	aiImproveFile string
	aiImproveOut  string
)

var aiImproveCmd = &cobra.Command{
	Use:   "improve",
	Short: "Analyze an existing ThreatSim security policy and generate additional simulations to close security gaps",
	Long:  `Improve reads an existing ThreatSim policy file, analyzes missing attack coverage (IDOR, CORS, Rate Limit, JWT forgery), and generates complementary simulation entries to strengthen your security suite.`,
	Run: func(cmd *cobra.Command, args []string) {
		if strings.TrimSpace(aiImproveFile) == "" {
			fmt.Println("Error: --file flag is required. Specify the path to an existing simulation YAML file.")
			os.Exit(1)
		}

		data, err := os.ReadFile(aiImproveFile)
		if err != nil {
			fmt.Printf("Error reading simulation file %q: %v\n", aiImproveFile, err)
			os.Exit(1)
		}

		cfg, err := ai.LoadConfigFromEnv()
		if err != nil {
			fmt.Printf("Configuration Error: %v\n", err)
			fmt.Println("Please set THREATSIM_AI_API_KEY in your .env file or environment (see .env.example).")
			os.Exit(1)
		}

		client := ai.NewClient(*cfg)
		improver := ai.NewImprover(client)

		fmt.Printf("Analyzing and improving policy coverage using %s (%s)...\n", strings.Title(cfg.Provider), cfg.Model)

		mergedYAML, mergedDef, err := improver.Improve(context.Background(), string(data))
		if err != nil {
			fmt.Printf("Policy improvement failed: %v\n", err)
			os.Exit(1)
		}

		targetPath := aiImproveOut
		if targetPath == "" {
			targetPath = "tests/simulations/improved.yaml"
		}

		if dir := filepath.Dir(targetPath); dir != "." && dir != "" {
			_ = os.MkdirAll(dir, 0755)
		}

		if err := os.WriteFile(targetPath, []byte(mergedYAML+"\n"), 0644); err != nil {
			fmt.Printf("Error writing improved simulation file to %q: %v\n", targetPath, err)
			os.Exit(1)
		}

		fmt.Printf("\n✓ Improved security policy suite: %d total simulations (expanded coverage)\n", len(mergedDef.Simulations))
		fmt.Printf("✓ Saved to %s\n", targetPath)
	},
}

func init() {
	aiCmd.AddCommand(aiImproveCmd)

	aiImproveCmd.Flags().StringVarP(&aiImproveFile, "file", "f", "", "Path to the existing ThreatSim simulation YAML file to improve (Required)")
	aiImproveCmd.Flags().StringVarP(&aiImproveOut, "out-file", "o", "tests/simulations/improved.yaml", "Destination path for the improved simulation suite")
	aiImproveCmd.MarkFlagRequired("file")
}
