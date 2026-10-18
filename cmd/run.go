package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/suryatk2007/threatsim/pkg/engine"
	"gopkg.in/yaml.v3"
)

// Config represents the structure of the threatsim.yaml configuration file
type Config struct {
	TargetURL string `yaml:"target_url"`
	File      string `yaml:"file"`
	Timeout   string `yaml:"timeout"`
	Insecure  bool   `yaml:"insecure"`
}

var (
	targetURL  string
	simFile    string
	outputFmt  string
	outFile    string
	timeoutStr string
	insecure   bool
)

// loadConfig attempts to read threatsim.yaml or fallback/threatsim.yaml
func loadConfig() (*Config, error) {
	configPaths := []string{
		"threatsim.yaml",
		".threatsim.yaml",
		filepath.Join("fallback", "threatsim.yaml"),
	}
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
	RunE: func(cmd *cobra.Command, args []string) error {
		// Attempt to load from config file
		cfg, err := loadConfig()
		if err != nil {
			return fmt.Errorf("error reading config: %w", err)
		}

		// Fallback to config file if flags are not provided
		finalURL := targetURL
		finalFile := simFile
		finalTimeoutStr := timeoutStr
		finalInsecure := insecure

		if cfg != nil {
			if finalURL == "" {
				finalURL = cfg.TargetURL
			}
			if finalFile == "" {
				finalFile = cfg.File
			}
			if !cmd.Flags().Changed("timeout") && cfg.Timeout != "" {
				finalTimeoutStr = cfg.Timeout
			}
			if !cmd.Flags().Changed("insecure") && cfg.Insecure {
				finalInsecure = cfg.Insecure
			}
		}

		if finalURL == "" {
			return fmt.Errorf("target URL is required; provide it via --target-url or threatsim.yaml")
		}
		if finalFile == "" {
			return fmt.Errorf("simulation file is required; provide it via --file or threatsim.yaml")
		}

		timeoutDuration, err := time.ParseDuration(finalTimeoutStr)
		if err != nil {
			return fmt.Errorf("invalid timeout value %q: %w", finalTimeoutStr, err)
		}

		eng := engine.New(finalURL, engine.WithTimeout(timeoutDuration), engine.WithInsecure(finalInsecure))

		// Load the simulation file
		def, err := eng.LoadSimulation(finalFile)
		if err != nil {
			return fmt.Errorf("error loading simulation: %w", err)
		}

		// Execute the simulations
		report := eng.Execute(def)

		// Generate the report
		var reporter engine.Reporter
		switch strings.ToLower(outputFmt) {
		case "json":
			reporter = &engine.JSONReporter{}
		case "html":
			reporter = &engine.HTMLReporter{}
		case "pdf":
			reporter = &engine.PDFReporter{}
		case "sarif":
			reporter = &engine.SARIFReporter{}
		case "junit", "xml":
			reporter = &engine.JUnitReporter{}
		default:
			reporter = &engine.ConsoleReporter{}
		}

		if (outputFmt == "html" || outputFmt == "pdf" || outputFmt == "sarif" || outputFmt == "junit" || outputFmt == "xml") && outFile == "" {
			if err := os.MkdirAll("reports", 0755); err != nil {
				return fmt.Errorf("error creating reports directory: %w", err)
			}
			ext := outputFmt
			if ext == "sarif" {
				ext = "sarif.json"
			} else if ext == "junit" {
				ext = "xml"
			}
			outFile = fmt.Sprintf("reports/threatsim_report.%s", ext)
			fmt.Printf("Generating %s report at: %s\n", outputFmt, outFile)
		}

		outWriter := os.Stdout
		if outFile != "" {
			f, err := os.Create(outFile)
			if err != nil {
				return fmt.Errorf("error creating output file: %w", err)
			}
			defer f.Close()
			outWriter = f
		}

		err = reporter.Generate(outWriter, report, eng.State.GetAll())
		if err != nil {
			return fmt.Errorf("error generating report: %w", err)
		}

		// Exit with failure code if any simulations failed
		if report.Failed > 0 {
			return fmt.Errorf("simulation validation failed: %d failed controls, %d setup errors", report.Failed, report.Errors)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(runCmd)

	// Local flags for the run command
	runCmd.Flags().StringVarP(&targetURL, "target-url", "t", "", "Target application base URL (e.g., http://localhost:8080)")
	runCmd.Flags().StringVarP(&simFile, "file", "f", "", "Path to the simulation file (YAML or JSON)")
	runCmd.Flags().StringVarP(&outputFmt, "output", "o", "console", "Output format for the report (console, json, html, pdf)")
	runCmd.Flags().StringVar(&outFile, "out-file", "", "Write report to the specified file (useful for pdf or html)")
	runCmd.Flags().StringVar(&timeoutStr, "timeout", "15s", "Default HTTP request timeout (e.g., 5s, 15s, 1m)")
	runCmd.Flags().BoolVar(&insecure, "insecure", false, "Skip TLS/SSL certificate verification for HTTPS requests")
}
