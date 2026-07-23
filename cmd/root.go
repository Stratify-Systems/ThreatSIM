/*
Copyright © 2026 Surya TK
*/
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/suryatk2007/threatsim/pkg/engine"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "threatsim",
	Short: "ThreatSim is a security validation execution engine",
	Long: fmt.Sprintf("%s%s%s\n%s", "\033[1m\033[36m", engine.ASCIIBanner, "\033[0m",
		`ThreatSim validates an application's security behavior by executing
predefined security simulations against a running application and verifying
that the application's response matches the expected behavior.

It is designed to be easily extensible and integrated into CI/CD pipelines.`),
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
