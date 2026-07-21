package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/suryatk2007/threatsim/pkg/engine"
	"gopkg.in/yaml.v3"
)

// Config represents the structure of the threatsim.yaml configuration file
type Config struct {
	TargetURL string `yaml:"target_url"`
	File      string `yaml:"file"`
}

var (
	targetURL string
	simFile   string
	outputFmt string
)

// loadConfig attempts to read threatsim.yaml from the current directory
func loadConfig() (*Config, error) {
	configPaths := []string{"threatsim.yaml", ".threatsim.yaml"}
	for _, path := range configPaths {
		if data, err := os.ReadFile(path); err == nil {
			var cfg Config
			if err := yaml.Unmarshal(data, &cfg); err != nil {
				return nil, fmt.Errorf("invalid config file format in %s: %w", path, err)
			}
			return &cfg, nil
		}
	}
	return nil, nil // No config file found
}

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Execute security simulations against a target application",
	Long:  `Run loads a simulation file (YAML or JSON) and executes all defined test cases against the specified target URL.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Attempt to load from config file
		cfg, err := loadConfig()
		if err != nil {
			fmt.Printf("Error reading config: %v\n", err)
			os.Exit(1)
		}

		// Fallback to config file if flags are not provided
		finalURL := targetURL
		finalFile := simFile

		if cfg != nil {
			if finalURL == "" {
				finalURL = cfg.TargetURL
			}
			if finalFile == "" {
				finalFile = cfg.File
			}
		}

		if finalURL == "" {
			fmt.Println("Error: target URL is required. Provide it via --target-url flag or in threatsim.yaml")
			os.Exit(1)
		}
		if finalFile == "" {
			fmt.Println("Error: simulation file is required. Provide it via --file flag or in threatsim.yaml")
			os.Exit(1)
		}

		eng := engine.New(finalURL)
		
		// Load the simulation file
		def, err := eng.LoadSimulation(finalFile)
		if err != nil {
			fmt.Printf("Error loading simulation: %v\n", err)
			os.Exit(1) // Stop execution if invalid
		}

		// Execute the simulations
		report := eng.Execute(def)
		
		// Generate the report
		var reporter engine.Reporter
		if outputFmt == "json" {
			reporter = &engine.JSONReporter{}
		} else {
			reporter = &engine.ConsoleReporter{}
		}

		err = reporter.Generate(os.Stdout, report, eng.State)
		if err != nil {
			fmt.Printf("Error generating report: %v\n", err)
			os.Exit(1)
		}
		
		// Exit with failure code if any simulations failed
		if report.Failed > 0 {
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(runCmd)

	// Local flags for the run command
	runCmd.Flags().StringVarP(&targetURL, "target-url", "t", "", "Target application base URL (e.g., http://localhost:8080)")
	runCmd.Flags().StringVarP(&simFile, "file", "f", "", "Path to the simulation file (YAML or JSON)")
	runCmd.Flags().StringVarP(&outputFmt, "output", "o", "console", "Output format for the report (console, json)")
}
