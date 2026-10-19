package cmd

import (
	"fmt"
	"net/http"
	"os"

	"github.com/spf13/cobra"
	"github.com/suryatk2007/threatsim/pkg/api"
)

var (
	serveHost string
	servePort int
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the local ThreatSim dashboard API",
	RunE: func(cmd *cobra.Command, _ []string) error {
		root, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("resolve workspace: %w", err)
		}
		address := fmt.Sprintf("%s:%d", serveHost, servePort)
		fmt.Printf("ThreatSim API listening on http://%s\n", address)
		return http.ListenAndServe(address, api.NewServer(root).Handler())
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
	serveCmd.Flags().StringVar(&serveHost, "host", "127.0.0.1", "API bind host")
	serveCmd.Flags().IntVarP(&servePort, "port", "p", 8090, "API port")
}
