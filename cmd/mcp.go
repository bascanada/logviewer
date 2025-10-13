package cmd

// -----------------------------------------------------------------------------
// Future Improvements (MCP Server)
// -----------------------------------------------------------------------------
// 1. Streaming / Follow Mode:
//    - Add a tool (e.g. "tail_logs" or "query_logs_follow") that streams new log
//      entries for a context (server-sent incremental batches or a bounded poll
//      loop). Would require MCP extension for incremental results or chunked
//      output handling.
// 2. Summarization / Analytics Tool:
//    - Provide a "summarize_logs" tool that groups by level, extracts top error
//      signatures, counts occurrences, and surfaces anomaly hints. Could accept
//      parameters: contextId, last, groupBy (level/service), topN.
// 3. Explicit Time Range Parameters:
//    - Support gte / lte absolute timestamps (RFC3339) alongside "last" to allow
//      precise investigations and reproducibility of queries.
// 4. Aggregation / Facet Tool:
//    - Expose a "facet_fields" tool returning counts for selected fields
//      (e.g. level, service, host) to guide targeted filtering.
// 5. Structured Error Codes:
//    - Standardize JSON error envelope with machine-friendly codes
//      (e.g. CONTEXT_NOT_FOUND, BACKEND_UNAVAILABLE, VALIDATION_ERROR) instead of
//      returning plain text or heuristic detection.
// 6. Field Discovery Caching:
//    - Cache get_fields results per context + time window (LRU / TTL) to reduce
//      backend load when agents probe frequently.
// 7. Partial / Sample Queries:
//    - Allow a lightweight "sample_logs" tool that fetches a very small set
//      (e.g. size=5) quickly for faster iterative refinement.
// 8. Query DSL / Expression Language:
//    - Introduce a simple expression syntax (level=ERROR AND message~"timeout")
//      parsed into backend-specific filters to expand flexibility beyond
//      strict equality.
// 9. Security / Multi-Tenancy:
//    - Context-level ACLs and redaction hooks before returning entries
//      (mask secrets, PII, tokens).
// 10. Metrics & Instrumentation:
//     - Emit internal metrics (query latency, error rate, cache hit ratio) and
//       optionally expose via a "diagnostics" tool.
// 11. Pagination / Cursoring:
//     - Extend query_logs to return a cursor token for fetching next pages when
//       size limit is reached.
// 12. Enhanced Similarity Suggestions:
//     - Replace simple Levenshtein with weighted trigram similarity and include
//       last-used context prioritization.
// 13. README / Documentation Update:
//     - Add detailed MCP usage section, examples of natural-language prompts,
//       and troubleshooting guide for context resolution.
// 14. Pluggable Normalization Pipeline:
//     - Allow custom transformers (timestamp normalization, field remapping,
//       enrichment) prior to returning entries.
// 15. Rate Limiting / Circuit Breaking:
//     - Prevent costly repeated queries (same context/filters) in tight loops.
// 16. Advanced Prompt Templates:
//     - Additional prompts: error_investigation, performance_degradation,
//       release_regression to accelerate LLM-driven diagnostics.
// 17. Cross-Context Correlation Tool:
//     - Tool that executes the same search across multiple contexts and merges
//       aligned timelines (e.g. by traceId / requestId).
// 18. Output Formatting Options:
//     - Allow user to request minimal, pretty, or raw JSON for entries.
// 19. Pluggable Authentication to External Backends:
//     - Support dynamic credentials injection or rotation for Splunk/ELK.
// 20. Test Coverage Expansion:
//     - Add integration tests specifically for MCP tool handlers with mocked
//       factories to ensure stability.
// -----------------------------------------------------------------------------

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/bascanada/logviewer/pkg/log/client"
	"github.com/bascanada/logviewer/pkg/log/client/config"
	"github.com/bascanada/logviewer/pkg/log/factory"
	"github.com/bascanada/logviewer/pkg/ty"
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
		cfg, err := config.LoadContextConfig(configPath)
		if err != nil {
			switch {
			case errors.Is(err, config.ErrConfigParse):
				log.Fatalf("invalid configuration format: %v", err)
			case errors.Is(err, config.ErrNoClients):
				log.Fatalf("configuration missing 'clients' section: %v", err)
			case errors.Is(err, config.ErrNoContexts):
				log.Fatalf("configuration missing 'contexts' section: %v", err)
			default:
				log.Fatalf("failed to load context config: %v", err)
			}
		}

		bundle, err := BuildMCPServer(cfg)
		if err != nil {
			log.Fatalf("failed to build MCP server: %v", err)
		}

		if err := server.ServeStdio(bundle.Server); err != nil {
			log.Fatalf("failed to start server: %v", err)
		}
	},
}

// BuildMCPServer creates an MCP server instance with all tools/resources/prompts registered.
// Exposed for testing so we can spin up the server without invoking cobra.Run path.
type MCPServerBundle struct {
	Server       *server.MCPServer
	ToolHandlers map[string]func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error)
}

func BuildMCPServer(cfg *config.ContextConfig) (*MCPServerBundle, error) {
	// Build shared factories once so every tool handler can reuse them.
	clientFactory, err := factory.GetLogClientFactory(cfg.Clients)
	if err != nil {
		return nil, fmt.Errorf("failed to create client factory: %w", err)
	}

	searchFactory, err := factory.GetLogSearchFactory(clientFactory, *cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create search factory: %w", err)
	}

	s := server.NewMCPServer(
		"logviewer",
		"1.0.0",
		server.WithToolCapabilities(true),
		server.WithRecovery(),
	)

	handlers := map[string]func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error){}

	listContextsTool := mcp.NewTool("list_contexts",
		mcp.WithDescription(`List all configured log contexts.

Usage: list_contexts

Returns: JSON array of context identifiers (strings) that can be used in other tools.

Note: You don't have to call this before every query. You can attempt query_logs directly; if the contextId is invalid the server will now return suggestions including available contexts.
`),
	)
	listHandler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		contextIDs := make([]string, 0, len(cfg.Contexts))
		for id := range cfg.Contexts {
			contextIDs = append(contextIDs, id)
		}
		sort.Strings(contextIDs)
		jsonBytes, err := json.Marshal(contextIDs)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to marshal contexts: %v", err)), nil
		}
		return mcp.NewToolResultText(string(jsonBytes)), nil
	}
	s.AddTool(listContextsTool, listHandler)
	handlers["list_contexts"] = listHandler

	getFieldsTool := mcp.NewTool("get_fields",
		mcp.WithDescription(`Discover available structured log fields for a given context.

Usage: get_fields contextId=<context>

Parameters:
  contextId (string, required): Context identifier.

Returns: JSON object mapping field names to arrays of distinct values.

You may skip this and directly call query_logs. If a query returns no results, consider then calling get_fields to validate field names or broaden the time window.
`),
		mcp.WithString("contextId", mcp.Required(), mcp.Description("Context identifier to inspect.")),
		mcp.WithString("last", mcp.Description("Optional relative time window for field discovery (e.g. 30m, 2h). Defaults to 15m.")),
	)
	getFieldsHandler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {

		// Extract required parameter contextId
		contextId, err := request.RequireString("contextId")
		if err != nil || contextId == "" {
			return mcp.NewToolResultError(fmt.Sprintf("invalid or missing contextId: %v", err)), nil
		}

		// Provide a small default time window unless user overrides with last
		search := client.LogSearch{}
		if lastVal, e2 := request.RequireString("last"); e2 == nil && lastVal != "" {
			search.Range.Last.S(lastVal)
		} else if !search.Range.Last.Set && !search.Range.Gte.Set {
			search.Range.Last.S("15m")
		}

		searchResult, err := searchFactory.GetSearchResult(ctx, contextId, []string{}, search, nil)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		fields, _, err := searchResult.GetFields(ctx)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		jsonBytes, err := json.Marshal(fields)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to marshal fields: %v", err)), nil
		}
		return mcp.NewToolResultText(string(jsonBytes)), nil
	}
	s.AddTool(getFieldsTool, getFieldsHandler)
	handlers["get_fields"] = getFieldsHandler

	queryLogsTool := mcp.NewTool("query_logs",
		mcp.WithDescription(`Query log entries for a context with optional filters and time window.

Usage: query_logs contextId=<context> [last=15m] [size=100] [fields={"level":"ERROR"}]

Parameters:
	contextId (string, required): Context identifier.
	last (string, optional): Relative duration window (e.g. 15m, 2h, 1d).
	size (number, optional): Max number of log entries.
	fields (object, optional): Exact-match key/value filters.

Behavior improvements:
	- If contextId is invalid, the response includes suggestions (no need to pre-call list_contexts).
	- If results are empty, meta.hints will recommend next actions (e.g. broaden last, call get_fields).

Returns: { "entries": [...], "meta": { resultCount, contextId, queryTime, hints? } }
`),
		mcp.WithString("contextId", mcp.Required(), mcp.Description("Context identifier to query.")),
		mcp.WithString("last", mcp.Description(`Relative time window like 15m, 2h, 1d.`)),
		mcp.WithObject("fields", mcp.Description("Exact match key/value filters (JSON object).")),
		mcp.WithNumber("size", mcp.Description("Maximum number of log entries to return.")),
		mcp.WithObject("variables", mcp.Description("Runtime variables for the context (JSON object).")),
	)
	queryLogsHandler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		start := time.Now()
		contextId, err := request.RequireString("contextId")
		if err != nil || contextId == "" {
			return mcp.NewToolResultError(fmt.Sprintf("invalid or missing contextId: %v", err)), nil
		}

		searchRequest := client.LogSearch{}
		if last, err := request.RequireString("last"); err == nil && last != "" {
			searchRequest.Range.Last.S(last)
		}
		if size, err := request.RequireFloat("size"); err == nil && int(size) > 0 {
			searchRequest.Size.S(int(size))
		}

		runtimeVars := make(map[string]string)
		args := request.GetArguments()
		if args != nil {
			// Handle 'fields'
			if rawFields, ok := args["fields"]; ok && rawFields != nil {
				if fieldMap, ok := rawFields.(map[string]any); ok {
					if searchRequest.Fields == nil {
						searchRequest.Fields = ty.MS{}
					}
					for k, v := range fieldMap {
						searchRequest.Fields[k] = fmt.Sprintf("%v", v)
					}
				}
			}
			// Handle 'variables'
			if rawVars, ok := args["variables"]; ok && rawVars != nil {
				if varMap, ok := rawVars.(map[string]any); ok {
					for k, v := range varMap {
						runtimeVars[k] = fmt.Sprintf("%v", v)
					}
				}
			}
		}

		// Pre-flight check for required variables
		mergedContext, err := searchFactory.GetSearchContext(ctx, contextId, []string{}, searchRequest, runtimeVars)
		if err != nil {
			// Handle context not found error separately
			if errors.Is(err, config.ErrContextNotFound) {
				all := make([]string, 0, len(cfg.Contexts))
				for id := range cfg.Contexts {
					all = append(all, id)
				}
				sort.Strings(all)
				suggestions := suggestSimilar(contextId, all, 3)
				payload := map[string]any{
					"code":              "CONTEXT_NOT_FOUND",
					"error":             err.Error(),
					"invalidContext":    contextId,
					"availableContexts": all,
					"suggestions":       suggestions,
					"hint":              "Use a suggested contextId or call list_contexts for enumeration.",
				}
				b, mErr := json.Marshal(payload)
				if mErr != nil {
					return mcp.NewToolResultError(fmt.Sprintf("failed to marshal error payload: %v", mErr)), nil
				}
				return mcp.NewToolResultText(string(b)), nil
			}
			return mcp.NewToolResultError(fmt.Sprintf("failed to get search context: %v", err)), nil
		}

		for name, def := range mergedContext.Search.Variables {
			if def.Required {
				if _, ok := runtimeVars[name]; !ok {
					if _, ok := os.LookupEnv(name); !ok {
						errMsg := fmt.Sprintf("Missing required variable '%s'. Please ask the user for '%s' and call the tool again.", name, def.Description)
						return mcp.NewToolResultError(errMsg), nil
					}
				}
			}
		}

		// Fallback: ensure some time window is always specified to prevent backend errors
		if !searchRequest.Range.Last.Set && !searchRequest.Range.Gte.Set {
			searchRequest.Range.Last.S("15m")
		}

		searchResult, err := searchFactory.GetSearchResult(ctx, contextId, []string{}, searchRequest, runtimeVars)
		if err != nil {
			// This logic can be simplified now as we have a pre-flight check
			return mcp.NewToolResultError(err.Error()), nil
		}

		entries, _, err := searchResult.GetEntries(ctx)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		meta := map[string]any{
			"resultCount": len(entries),
			"contextId":   contextId,
			"queryTime":   time.Since(start).String(),
		}
		if len(entries) == 0 {
			meta["hints"] = []string{
				"No results: consider broadening 'last' (e.g. last=2h)",
				"If you used filters, verify field names via get_fields",
			}
		}
		response := map[string]any{"entries": entries, "meta": meta}
		jsonBytes, err := json.Marshal(response)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to marshal response: %v", err)), nil
		}
		return mcp.NewToolResultText(string(jsonBytes)), nil
	}
	s.AddTool(queryLogsTool, queryLogsHandler)
	handlers["query_logs"] = queryLogsHandler

	getContextDetailsTool := mcp.NewTool("get_context_details",
		mcp.WithDescription("Inspect a context's details, including its variable schema."),
		mcp.WithString("contextId", mcp.Required(), mcp.Description("The context ID to inspect.")),
	)
	getContextDetailsHandler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		contextId, err := request.RequireString("contextId")
		if err != nil {
			return mcp.NewToolResultError("contextId is required"), nil
		}
		searchContext, err := searchFactory.GetSearchContext(ctx, contextId, []string{}, client.LogSearch{}, nil)
		if err != nil {
			if errors.Is(err, config.ErrContextNotFound) {
				all := make([]string, 0, len(cfg.Contexts))
				for id := range cfg.Contexts {
					all = append(all, id)
				}
				sort.Strings(all)
				suggestions := suggestSimilar(contextId, all, 3)
				payload := map[string]any{
					"code":              "CONTEXT_NOT_FOUND",
					"error":             err.Error(),
					"invalidContext":    contextId,
					"availableContexts": all,
					"suggestions":       suggestions,
					"hint":              "Use a suggested contextId or call list_contexts for enumeration.",
				}
				b, mErr := json.Marshal(payload)
				if mErr != nil {
					return mcp.NewToolResultError(fmt.Sprintf("failed to marshal error payload: %v", mErr)), nil
				}
				return mcp.NewToolResultText(string(b)), nil
			}
			return mcp.NewToolResultError(fmt.Sprintf("failed to get context details: %v", err)), nil
		}
		jsonBytes, err := json.Marshal(searchContext.Search)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to marshal context details: %v", err)), nil
		}
		return mcp.NewToolResultText(string(jsonBytes)), nil
	}
	s.AddTool(getContextDetailsTool, getContextDetailsHandler)
	handlers["get_context_details"] = getContextDetailsHandler

	// Resource providing context list (alternative to tool usage)
	contextsResource := mcp.NewResource(
		"logviewer://contexts",
		"LogViewer Context Index",
		mcp.WithResourceDescription("JSON array of available context IDs; server also suggests them on invalid context query."),
		mcp.WithMIMEType("application/json"),
	)
	s.AddResource(contextsResource, func(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		ids := make([]string, 0, len(cfg.Contexts))
		for id := range cfg.Contexts {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		b, err := json.Marshal(ids)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal context IDs: %w", err)
		}
		return []mcp.ResourceContents{mcp.TextResourceContents{URI: "logviewer://contexts", MIMEType: "application/json", Text: string(b)}}, nil
	})

	// Prompt guiding efficient investigation workflow
	investigationPrompt := mcp.NewPrompt(
		"log_investigation",
		mcp.WithPromptDescription("Guide for investigating logs: query first, broaden or discover fields only if needed."),
		mcp.WithArgument("objective", mcp.ArgumentDescription("High-level goal (e.g. detect payment errors).")),
		mcp.WithArgument("contextId", mcp.ArgumentDescription("Optional starting context.")),
	)
	s.AddPrompt(investigationPrompt, func(ctx context.Context, request mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		obj := request.Params.Arguments["objective"]
		ctxId := request.Params.Arguments["contextId"]
		if ctxId == "" {
			ids := make([]string, 0, len(cfg.Contexts))
			for id := range cfg.Contexts {
				ids = append(ids, id)
			}
			sort.Strings(ids)
			if len(ids) > 0 {
				ctxId = ids[0]
			}
		}
		text := fmt.Sprintf(`Objective: %s
Strategy:
1. query_logs contextId=%s last=15m size=20
2. If no results: increase last (e.g. 1h) or drop filters
3. Only call get_fields if filters might be invalid or repeated empty result
4. On context error: check suggestions or resource logviewer://contexts
5. Summarize anomalies, refine with additional field filters
Return a short plan then perform tool calls.
`, obj, ctxId)
		return mcp.NewGetPromptResult("Log Investigation", []mcp.PromptMessage{mcp.NewPromptMessage(mcp.RoleAssistant, mcp.NewTextContent(text))}), nil
	})
	return &MCPServerBundle{Server: s, ToolHandlers: handlers}, nil
}

func init() {
	mcpCmd.Flags().IntVar(&mcpPort, "port", 8081, "Port for the MCP server")
	rootCmd.AddCommand(mcpCmd)
}

// suggestSimilar returns up to max suggestions ranked by simple edit distance (Levenshtein) and substring match boost.
func suggestSimilar(target string, candidates []string, max int) []string {
	type scored struct {
		v     string
		d     int
		boost bool
	}
	scoredList := make([]scored, 0, len(candidates))
	for _, c := range candidates {
		if c == target {
			continue
		}
		boost := strings.Contains(strings.ToLower(c), strings.ToLower(target))
		scoredList = append(scoredList, scored{v: c, d: levenshtein(target, c), boost: boost})
	}
	sort.Slice(scoredList, func(i, j int) bool {
		if scoredList[i].d != scoredList[j].d {
			return scoredList[i].d < scoredList[j].d
		}
		return scoredList[i].boost && !scoredList[j].boost
	})
	out := make([]string, 0, max)
	for _, s := range scoredList {
		out = append(out, s.v)
		if len(out) >= max {
			break
		}
	}
	return out
}

// levenshtein computes Levenshtein distance between two strings.
func levenshtein(a, b string) int {
	r1, r2 := []rune(a), []rune(b)
	n, m := len(r1), len(r2)
	if n == 0 {
		return m
	}
	if m == 0 {
		return n
	}
	dp := make([]int, m+1)
	for j := 0; j <= m; j++ {
		dp[j] = j
	}
	for i := 1; i <= n; i++ {
		prev := dp[0]
		dp[0] = i
		for j := 1; j <= m; j++ {
			cost := 0
			if r1[i-1] != r2[j-1] {
				cost = 1
			}
			insert := dp[j] + 1
			delete := dp[j-1] + 1
			subst := prev + cost
			prev = dp[j]
			min := insert
			if delete < min {
				min = delete
			}
			if subst < min {
				min = subst
			}
			dp[j] = min
		}
	}
	return dp[m]
}
