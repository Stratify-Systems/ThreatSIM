package main

import (
	"github.com/spf13/cobra"
)

func newRunCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run complex simulations (e.g., scenarios)",
	}

	cmd.AddCommand(newScenarioCmd())
	return cmd
}
