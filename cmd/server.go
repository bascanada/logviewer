package cmd

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"

	"github.com/bascanada/logviewer/pkg/api"
	
	"github.com/bascanada/logviewer/pkg/server"
	"github.com/spf13/cobra"
)

var (
	port       int
	host       string
	
)

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Start the logviewer server",
	Long:  `Starts an HTTP server to query logs, providing a programmatic API.`,
	PreRun: onCommandStart,
	RunE: func(cmd *cobra.Command, args []string) error {
		// NOTE: This implementation assumes a logger is configured and available via `onCommandStart`.
		// A basic logger is created here as an example. You should integrate this with your application's logging strategy.
		logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

		cfg, path, err := loadConfig(cmd)
		if err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}
		logger.Info("loading configuration", "path", path)

		s, err := server.NewServer(host, strconv.Itoa(port), cfg, logger, api.OpenAPISpec)
		if err != nil {
			return fmt.Errorf("failed to create server: %w", err)
		}

		if err := s.Start(); err != nil {
			return fmt.Errorf("server failed to start: %w", err)
		}
		return nil
	},
}

func init() {
	serverCmd.Flags().IntVarP(&port, "port", "p", 8080, "Port to listen on")
	serverCmd.Flags().StringVarP(&host, "host", "H", "0.0.0.0", "Host to bind to")
	addConfigFlag(serverCmd)
	serverCmd.MarkFlagRequired("config")
}
