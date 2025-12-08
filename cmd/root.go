// SPDX-License-Identifier: GPL-3.0-only
package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var (
	configPath string
)

var rootCmd = &cobra.Command{
	Use:    "logviewer",
	Short:  "Log viewer for different backend (OpenSearch, SSH, Local Files)",
	Long:   ``,
	PreRun: onCommandStart,
	Run: func(cmd *cobra.Command, args []string) {
		// Check if config exists before showing generic help
		home, err := os.UserHomeDir()
		if err == nil {
			configPath := filepath.Join(home, ".logviewer", "config.yaml")
			if _, err := os.Stat(configPath); os.IsNotExist(err) {
				fmt.Println("Welcome to logviewer!")
				fmt.Println("\nNo configuration found.")
				fmt.Println("   Run 'logviewer configure' to get started with an interactive setup wizard.")
				fmt.Println("\nOr use 'logviewer --help' to see all available options.")
				return
			}
		}
		cmd.Help()
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {

	rootCmd.PersistentFlags().StringVarP(&configPath, "config", "c", "", "Config for preconfigure context for search")
	rootCmd.PersistentFlags().StringVar(&logger.Path, "logging-path", "", "file to output logs of the application")
	rootCmd.PersistentFlags().StringVar(&logger.Level, "logging-level", "", "logging level to output INFO WARN ERROR DEBUG TRACE")
	rootCmd.PersistentFlags().BoolVar(&logger.Stdout, "logging-stdout", false, "output appplication log in the stdout")

	// Register completion for --logging-level flag
	_ = rootCmd.RegisterFlagCompletionFunc("logging-level", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"TRACE", "DEBUG", "INFO", "WARN", "ERROR"}, cobra.ShellCompDirectiveNoFileComp
	})

	rootCmd.AddCommand(queryCommand)
	rootCmd.AddCommand(versionCommand)
	rootCmd.AddCommand(serverCmd)
}
