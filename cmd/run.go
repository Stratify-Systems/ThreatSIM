package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/suryatk2007/threatsim/pkg/engine"
)

var (
	targetURL string
	simFile   string
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Execute security simulations against a target application",
	Long:  `Run loads a simulation file (YAML or JSON) and executes all defined test cases against the specified target URL.`,
	Run: func(cmd *cobra.Command, args []string) {
		if targetURL == "" {
			fmt.Println("Error: --target-url is required")
			os.Exit(1)
		}
		if simFile == "" {
			fmt.Println("Error: --file is required")
			os.Exit(1)
		}

		eng := engine.New(targetURL)
		
		// Load the simulation file
		def, err := eng.LoadSimulation(simFile)
		if err != nil {
			fmt.Printf("Error loading simulation: %v\n", err)
			os.Exit(1) // Stop execution if invalid
		}

		// Execute the simulations
		report := eng.Execute(def)
		
		// Generate the report
		engine.PrintReport(os.Stdout, report)
		
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
	
	runCmd.MarkFlagRequired("target-url")
	runCmd.MarkFlagRequired("file")
}
