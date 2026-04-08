package main

import (
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/Stratify-Systems/ThreatSIM/internal/plugins"
	bruteforce "github.com/Stratify-Systems/ThreatSIM/internal/plugins/brute_force"
	credstuff "github.com/Stratify-Systems/ThreatSIM/internal/plugins/credential_stuffing"
	ddos "github.com/Stratify-Systems/ThreatSIM/internal/plugins/ddos"
	portscan "github.com/Stratify-Systems/ThreatSIM/internal/plugins/port_scan"
	privesc "github.com/Stratify-Systems/ThreatSIM/internal/plugins/privilege_escalation"
)

// Version info (set at build time)
var version = "0.1.0"

// Global plugin registry — shared across all commands
var registry *plugins.Registry

func main() {
	// Initialize plugin registry with all available plugins
	registry = plugins.NewRegistry()
	registry.Register(&bruteforce.Plugin{})
	registry.Register(&portscan.Plugin{})
	registry.Register(&ddos.Plugin{})
	registry.Register(&credstuff.Plugin{})
	registry.Register(&privesc.Plugin{})

	// Build and execute the CLI
	rootCmd := newRootCmd()
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "threatsim",
		Short: "ThreatSIM — Cyber Attack Simulation Platform",
		Long: banner() + `
ThreatSIM is an open-source platform to simulate cyber attacks
and test whether your security detection systems actually work.

Use it to:
  • Simulate attacks (brute force, port scan, DDoS, etc.)
  • Run multi-step attack scenarios
  • Evaluate detection rules, alert pipelines, and SIEM systems`,
	}

	// Register sub-commands
	root.AddCommand(newSimulateCmd())
	root.AddCommand(newRunCmd())
	root.AddCommand(newListCmd())
	root.AddCommand(newVersionCmd())

	return root
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the version of ThreatSIM",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("ThreatSIM v%s\n", version)
		},
	}
}

func banner() string {
	cyan := color.New(color.FgCyan, color.Bold).SprintFunc()
	return cyan(`
  _____ _                    _   ____ ___ __  __ 
 |_   _| |__  _ __ ___  __ _| |_/ ___|_ _|  \/  |
   | | | '_ \| '__/ _ \/ _' | __\___ \| || |\/| |
   | | | | | | | |  __/ (_| | |_ ___) | || |  | |
   |_| |_| |_|_|  \___|\__,_|\__|____/___|_|  |_|
`) + "\n"
}
