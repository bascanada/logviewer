// SPDX-License-Identifier: GPL-3.0-only
package cmd

import (
	"fmt"
	"os"

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

	rootCmd.AddCommand(queryCommand)
	rootCmd.AddCommand(versionCommand)
	rootCmd.AddCommand(serverCmd)
}
