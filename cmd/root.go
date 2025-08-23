package cmd

import (
	"fmt"
	"os"

	"github.com/rivo/tview"
	"github.com/spf13/cobra"
)

var ()

var rootCmd = &cobra.Command{
	Use:    "logviewer",
	Short:  "Log viewer for different backend (OpenSearch, SSH, Local Files)",
	Long:   ``,
	PreRun: onCommandStart,
	Run: func(cmd *cobra.Command, args []string) {

		box := tview.NewBox().SetBorder(true).SetTitle("logviewer")
		if err := tview.NewApplication().SetRoot(box, true).Run(); err != nil {
			panic(err)
		}
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {

	rootCmd.PersistentFlags().StringVar(&logger.Path, "logging-path", "", "file to output logs of the application")
	rootCmd.PersistentFlags().StringVar(&logger.Level, "logging-level", "", "logging level to output INFO WARN ERROR DEBUG TRACE")
	rootCmd.PersistentFlags().BoolVar(&logger.Stdout, "logging-stdout", false, "output appplication log in the stdout")

	rootCmd.AddCommand(queryCommand)
	rootCmd.AddCommand(versionCommand)
	rootCmd.AddCommand(serverCmd)
}
