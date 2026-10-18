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
	RunE: func(cmd *cobra.Command, args []string) error {
		if strings.TrimSpace(aiImproveFile) == "" {
			return fmt.Errorf("--file flag is required; specify an existing simulation YAML file")
		}

		data, err := os.ReadFile(aiImproveFile)
		if err != nil {
			return fmt.Errorf("error reading simulation file %q: %w", aiImproveFile, err)
		}

		cfg, err := ai.LoadConfigFromEnv()
		if err != nil {
			return fmt.Errorf("configuration error: %w; set THREATSIM_AI_API_KEY (see .env.example)", err)
		}

		client := ai.NewClient(*cfg)
		improver := ai.NewImprover(client)

		fmt.Printf("Analyzing and improving policy coverage using %s (%s)...\n", strings.Title(cfg.Provider), cfg.Model)

		mergedYAML, mergedDef, err := improver.Improve(context.Background(), string(data))
		if err != nil {
			return fmt.Errorf("policy improvement failed: %w", err)
		}

		targetPath := aiImproveOut
		if targetPath == "" {
			targetPath = "tests/simulations/improved.yaml"
		}

		if dir := filepath.Dir(targetPath); dir != "." && dir != "" {
			if err := os.MkdirAll(dir, 0755); err != nil {
				return fmt.Errorf("error creating directory %q: %w", dir, err)
			}
		}

		if err := os.WriteFile(targetPath, []byte(mergedYAML+"\n"), 0644); err != nil {
			return fmt.Errorf("error writing improved simulation file to %q: %w", targetPath, err)
		}

		fmt.Printf("\n✓ Improved security policy suite: %d total simulations (expanded coverage)\n", len(mergedDef.Simulations))
		fmt.Printf("✓ Saved to %s\n", targetPath)
		return nil
	},
}

func init() {
	aiCmd.AddCommand(aiImproveCmd)

	aiImproveCmd.Flags().StringVarP(&aiImproveFile, "file", "f", "", "Path to the existing ThreatSim simulation YAML file to improve (Required)")
	aiImproveCmd.Flags().StringVarP(&aiImproveOut, "out-file", "o", "tests/simulations/improved.yaml", "Destination path for the improved simulation suite")
	aiImproveCmd.MarkFlagRequired("file")
}
