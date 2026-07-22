package cmd

import (
	"github.com/spf13/cobra"
)

var aiCmd = &cobra.Command{
	Use:   "ai",
	Short: "AI-powered Security-as-Code policy generation and analysis",
	Long:  `The ai subcommand group provides AI authoring tools for generating, analyzing, and enhancing ThreatSim security simulation policies.`,
}

func init() {
	rootCmd.AddCommand(aiCmd)
}
