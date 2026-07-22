/*
Copyright © 2026 Surya TK
*/
package cmd

import (
	"os"

	"github.com/spf13/cobra"
)



// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "threatsim",
	Short: "ThreatSim is a security validation execution engine",
	Long: `ThreatSim validates an application's security behavior by executing
predefined security simulations against a running application and verifying
that the application's response matches the expected behavior.

It is designed to be easily extensible and integrated into CI/CD pipelines.`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	// Run: func(cmd *cobra.Command, args []string) { },
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// Disable default completion command
	rootCmd.CompletionOptions.DisableDefaultCmd = true
}


