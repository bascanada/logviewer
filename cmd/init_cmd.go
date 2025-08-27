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
	Run: func(cmd *cobra.Command, args []string) {
		defaultConfig := config.ContextConfig{
			Clients: config.Clients{
				"local-splunk": {
					Type: "splunk",
					Options: ty.MI{
						"Url": "https://localhost:8089/services",
						"Headers": ty.MS{
							"Authorization": "Basic YWRtaW46cGFzc3dvcmQ=",
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
			fmt.Printf("unsupported format: %s\n", format)
			os.Exit(1)
		}

		if err != nil {
			fmt.Printf("failed to marshal config: %v\n", err)
			os.Exit(1)
		}

		if err := os.WriteFile(fileName, data, 0644); err != nil {
			fmt.Printf("failed to write config file: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("created config file: %s\n", fileName)
	},
}

func init() {
	initCmd.Flags().StringVar(&format, "format", "json", "config file format (json or yaml)")
	rootCmd.AddCommand(initCmd)
}
