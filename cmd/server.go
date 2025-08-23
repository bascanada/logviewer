package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	port       int
	host       string
	configPath string
)

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Start the logviewer server",
	Long:  `Starts an HTTP server to query logs, providing a programmatic API.`,
	PreRun: onCommandStart,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Starting server on %s:%d\n", host, port)
		fmt.Printf("Using config file: %s\n", configPath)
		// Server startup logic will go here
	},
}

func init() {
	serverCmd.Flags().IntVar(&port, "port", 8080, "Port to listen on")
	serverCmd.Flags().StringVar(&host, "host", "0.0.0.0", "Host to bind to")
	serverCmd.Flags().StringVar(&configPath, "config", "", "Path to the config.json file (required)")
	serverCmd.MarkFlagRequired("config")
}
