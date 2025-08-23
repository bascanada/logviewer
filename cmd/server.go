package cmd

import (
	"log/slog"
	"os"
	"strconv"

	"github.com/berlingoqc/logviewer/pkg/log/client/config"
	"github.com/berlingoqc/logviewer/pkg/server"
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
		// NOTE: This implementation assumes a logger is configured and available via `onCommandStart`.
		// A basic logger is created here as an example. You should integrate this with your application's logging strategy.
		logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

		logger.Info("loading configuration", "path", configPath)
		cfg, err := config.LoadContextConfig(configPath)
		if err != nil {
			logger.Error("failed to load configuration", "err", err)
			os.Exit(1)
		}

		s, err := server.NewServer(host, strconv.Itoa(port), cfg, logger)
		if err != nil {
			logger.Error("failed to create server", "err", err)
			os.Exit(1)
		}

		logger.Info("starting server", "host", host, "port", port)
		if err := s.Start(); err != nil {
			logger.Error("server failed to start", "err", err)
			os.Exit(1)
		}
	},
}

func init() {
	serverCmd.Flags().IntVar(&port, "port", 8080, "Port to listen on")
	serverCmd.Flags().StringVar(&host, "host", "0.0.0.0", "Host to bind to")
	serverCmd.Flags().StringVar(&configPath, "config", "", "Path to the config.json file (required)")
	serverCmd.MarkFlagRequired("config")
}
