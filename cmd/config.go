package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/bascanada/logviewer/pkg/log/client/config"
	"github.com/spf13/cobra"
)

var configPath string

func addConfigFlag(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&configPath, "config", "c", "config.json", "config file (json or yaml)")
}

func loadConfig(cmd *cobra.Command) (*config.ContextConfig, string, error) {
	configPath, _ := cmd.Flags().GetString("config")
	var loadedPath string

	if !cmd.Flags().Changed("config") {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, "", fmt.Errorf("failed to get user home directory: %w", err)
		}

		for _, ext := range []string{".json", ".yaml", ".yml"} {
			path := filepath.Join(home, ".logviewer", "config"+ext)
			if _, err := os.Stat(path); err == nil {
				loadedPath = path
				break
			}
		}
	}

	if loadedPath != "" {
		fmt.Fprintf(os.Stderr, "using default config file: %s
", loadedPath)
		configPath = loadedPath
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		return nil, "", err
	}

	return cfg, configPath, nil
}
