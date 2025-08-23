package cmd

import (
	"github.com/spf13/cobra"
)

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Starts a MCP server",
	Long:  `Starts a MCP server, exposing the logviewer's core functionalities as a tool.`,
	Run: func(cmd *cobra.Command, args []string) {
	},
}

func init() {
	rootCmd.AddCommand(mcpCmd)
}
