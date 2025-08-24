package cmd

import (
	"context"
	"encoding/json"
	"log"
	"os"

	"github.com/berlingoqc/logviewer/pkg/log/client/config"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/spf13/cobra"
)

var mcpPort int

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Starts a MCP server",
	Long:  `Starts a MCP server, exposing the logviewer's core functionalities as a tool.`,
	Run: func(cmd *cobra.Command, args []string) {
		if contextPath == "" {
			log.Fatal("config file is required")
		}
		cfg, err := config.LoadContextConfig(contextPath)
		if err != nil {
			log.Fatalf("failed to load context config: %v", err)
		}

		s := server.NewMCPServer(
			"logviewer",
			"1.0.0",
			server.WithToolCapabilities(true),
			server.WithRecovery(),
		)

		listContextsTool := mcp.NewTool("list_contexts",
			mcp.WithDescription("Lists all available log contexts from the configuration file."),
		)
		s.AddTool(listContextsTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			contextIDs := make([]string, 0, len(cfg.Contexts))
			for id := range cfg.Contexts {
				contextIDs = append(contextIDs, id)
			}
			jsonBytes, err := json.Marshal(contextIDs)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			return mcp.NewToolResultText(string(jsonBytes)), nil
		})

		getFieldsTool := mcp.NewTool("get_fields",
			mcp.WithDescription("Retrieves the available fields for a specified context."),
			mcp.WithString("contextId", mcp.Required(), mcp.Description("The ID of the context to query.")),
		)
		s.AddTool(getFieldsTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			// TODO: The implementation of this function is blocked by an unknown API for `mcp-go`.
			// The following code is a guess and does not compile.
			//
			// contextId, err := request.RequireString("contextId")
			// if err != nil {
			// 	return mcp.NewToolResultError(err.Error()), nil
			// }

			// clientFactory, err := factory.GetLogClientFactory(cfg.Clients)
			// if err != nil {
			// 	log.Fatalf("failed to create client factory: %v", err)
			// }

			// searchFactory, err := factory.GetLogSearchFactory(clientFactory, *cfg)
			// if err != nil {
			// 	log.Fatalf("failed to create search factory: %v", err)
			// }

			// searchResult, err := searchFactory.GetSearchResult(contextId, []string{}, client.LogSearch{})
			// if err != nil {
			// 	return mcp.NewToolResultError(err.Error()), nil
			// }
			// fields, _, err := searchResult.GetFields()
			// if err != nil {
			// 	return mcp.NewToolResultError(err.Error()), nil
			// }
			// jsonBytes, err := json.Marshal(fields)
			// if err != nil {
			// 	return mcp.NewToolResultError(err.Error()), nil
			// }
			// return mcp.NewToolResultText(string(jsonBytes)), nil
			return mcp.NewToolResultError("not implemented"), nil
		})

		queryLogsTool := mcp.NewTool("query_logs",
			mcp.WithDescription("Queries for log entries based on a context and optional filters."),
			mcp.WithString("contextId", mcp.Required(), mcp.Description("The ID of the context to query.")),
			mcp.WithString("last", mcp.Description(`A duration string (e.g., "15m", "2h").`)),
			mcp.WithObject("fields", mcp.Description("A map of field-value pairs for filtering.")),
			mcp.WithNumber("size", mcp.Description("The maximum number of log entries to return.")),
		)
		s.AddTool(queryLogsTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			// TODO: The implementation of this function is blocked by an unknown API for `mcp-go`.
			// The following code is a guess and does not compile.
			//
			// contextId, err := request.RequireString("contextId")
			// if err != nil {
			// 	return mcp.NewToolResultError(err.Error()), nil
			// }

			// searchRequest := client.LogSearch{}
			// if last, err := request.RequireString("last"); err == nil {
			// 	searchRequest.Range.Last.S(last)
			// }
			// if fields, err := request.RequireMap("fields"); err == nil {
			// 	searchRequest.Fields = make(map[string]string)
			// 	for k, v := range fields {
			// 		searchRequest.Fields[k] = fmt.Sprintf("%v", v)
			// 	}
			// }
			// if size, err := request.RequireFloat("size"); err == nil {
			// 	searchRequest.Size.S(int(size))
			// }

			// clientFactory, err := factory.GetLogClientFactory(cfg.Clients)
			// if err != nil {
			// 	log.Fatalf("failed to create client factory: %v", err)
			// }

			// searchFactory, err := factory.GetLogSearchFactory(clientFactory, *cfg)
			// if err != nil {
			// 	log.Fatalf("failed to create search factory: %v", err)
			// }

			// searchResult, err := searchFactory.GetSearchResult(contextId, []string{}, searchRequest)
			// if err != nil {
			// 	return mcp.NewToolResultError(err.Error()), nil
			// }

			// entries, _, err := searchResult.GetEntries(cmd.Context())
			// if err != nil {
			// 	return mcp.NewToolResultError(err.Error()), nil
			// }
			// jsonBytes, err := json.Marshal(entries)
			// if err != nil {
			// 	return mcp.NewToolResultError(err.Error()), nil
			// }
			// return mcp.NewToolResultText(string(jsonBytes)), nil
			return mcp.NewToolResultError("not implemented"), nil
		})

		if err := server.ServeStdio(s); err != nil {
			log.Fatalf("failed to start server: %v", err)
		}
	},
}

func init() {
	mcpCmd.Flags().IntVar(&mcpPort, "port", 8081, "Port for the MCP server")
	mcpCmd.PersistentFlags().StringVarP(&contextPath, "config", "c", os.Getenv("CONTEXT_PATH"), "Path to the context file")
	rootCmd.AddCommand(mcpCmd)
}
