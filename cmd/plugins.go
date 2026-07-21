package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/suryatk2007/threatsim/pkg/plugins"
)

var pluginsCmd = &cobra.Command{
	Use:   "plugins",
	Short: "List all available security validation plugins",
	Long:  `Lists all installed ThreatSim plugins and provides a brief description of their functionality and schema requirements.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("==================================================")
		fmt.Println("        ThreatSim Validation Plugins")
		fmt.Println("==================================================")

		allPlugins := plugins.List()
		if len(allPlugins) == 0 {
			fmt.Println("No plugins installed.")
			return
		}

		for name, p := range allPlugins {
			fmt.Printf("\n🔹 Plugin: %s\n", name)
			fmt.Printf("   Description: %s\n", p.Description())
			
			schemaPath := fmt.Sprintf("schemas/plugins/%s.yaml", name)
			if _, err := os.Stat(schemaPath); err == nil {
				fmt.Printf("   Schema:      %s\n", schemaPath)
			} else {
				fmt.Printf("   Schema:      No YAML schema found.\n")
			}
		}
		fmt.Println("\n==================================================")
	},
}

func init() {
	rootCmd.AddCommand(pluginsCmd)
}
