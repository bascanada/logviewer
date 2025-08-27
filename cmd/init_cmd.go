package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/bascanada/logviewer/pkg/log/client"
	"github.com/bascanada/logviewer/pkg/log/client/config"
	"github.com/bascanada/logviewer/pkg/ty"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var format string

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Create a default config file",
	RunE: func(cmd *cobra.Command, args []string) error {
		defaultConfig := config.ContextConfig{
			Clients: config.Clients{
				"local-splunk": {
					Type: "splunk",
					Options: ty.MI{
						"Url": "https://localhost:8089/services",
						"Headers": ty.MS{
							"Authorization": "Basic <base64-encoded-username:password>",
						},
						"SearchBody": ty.MS{
							"output_mode": "json",
						},
					},
				},
			},
			Contexts: config.Contexts{
				"splunk-app-logs": {
					Client: "local-splunk",
					Search: client.LogSearch{
						Fields: ty.MS{
							"sourcetype": "httpevent",
						},
						Options: ty.MI{
							"index": "main",
						},
					},
				},
			},
		}

		var (
			data []byte
			err  error
		)

		fileName := "config." + format

		switch format {
		case "json":
			data, err = json.MarshalIndent(defaultConfig, "", "  ")
		case "yaml":
			data, err = yaml.Marshal(defaultConfig)
		default:
			return fmt.Errorf("unsupported format: %s", format)
		}

		if err != nil {
			return fmt.Errorf("failed to marshal config: %w", err)
		}

		if err := os.WriteFile(fileName, data, 0644); err != nil {
			return fmt.Errorf("failed to write config file: %w", err)
		}

		fmt.Printf("created config file: %s\n", fileName)
		return nil
	},
}

func init() {
	initCmd.Flags().StringVar(&format, "format", "json", "config file format (json or yaml)")
	rootCmd.AddCommand(initCmd)
}
