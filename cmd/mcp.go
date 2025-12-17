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
// 11. Pagination / Cursoring: ✅ COMPLETED
//     - query_logs now supports pageToken parameter and returns nextPageToken in
//       meta when more results are available. Agent can fetch subsequent pages by
//       passing the token in the next request.
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
	"sync"
	"time"

	"github.com/bascanada/logviewer/pkg/log/client"
	"github.com/bascanada/logviewer/pkg/log/client/config"
	"github.com/bascanada/logviewer/pkg/log/factory"
	"github.com/bascanada/logviewer/pkg/ty"
	"github.com/fsnotify/fsnotify"
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
		// Centralized config handling (matching query command):
		// - If an explicit configPath is given, use it.
		// - If no configPath, attempt to load the default config.
		cfgPath := configPath

		// Pre-validate config loading to provide consistent error messages
		_, files, err := loadConfig(cfgPath)
		if err != nil {
			log.Fatal(err)
		}

		log.Printf("Starting MCP server with config: %v\n", files)

		bundle, err := BuildMCPServer(cfgPath)
		if err != nil {
			log.Fatalf("failed to build MCP server: %v", err)
		}

		if err := server.ServeStdio(bundle.Server); err != nil {
			log.Fatalf("failed to start server: %v", err)
		}
	},
}

// ConfigManager handles thread-safe configuration reloading.
type ConfigManager struct {
	mu            sync.RWMutex
	configPath    string
	loadedFiles   []string
	currentCfg    *config.ContextConfig
	searchFactory factory.SearchFactory
	watcher       *fsnotify.Watcher
	debounceTimer *time.Timer
	closeChan     chan struct{}
}

func NewConfigManager(path string) (*ConfigManager, error) {
	cm := &ConfigManager{
		configPath: path,
		closeChan:  make(chan struct{}),
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create watcher: %w", err)
	}
	cm.watcher = watcher

	if err := cm.Reload(); err != nil {
		watcher.Close()
		return nil, err
	}

	go cm.watch()

	return cm, nil
}

// NewConfigManagerForTest creates a ConfigManager from an in-memory config (for testing).
// Does not set up file watching.
func NewConfigManagerForTest(cfg *config.ContextConfig) (*ConfigManager, error) {
	clientFactory, err := factory.GetLogClientFactory(cfg.Clients)
	if err != nil {
		return nil, fmt.Errorf("failed to build client factory: %w", err)
	}

	searchFactory, err := factory.GetLogSearchFactory(clientFactory, *cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to build search factory: %w", err)
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create watcher: %w", err)
	}

	return &ConfigManager{
		currentCfg:    cfg,
		searchFactory: searchFactory,
		watcher:       watcher,
	}, nil
}

func (cm *ConfigManager) watch() {
	const debounceDelay = 100 * time.Millisecond

	for {
		select {
		case event, ok := <-cm.watcher.Events:
			if !ok {
				return
			}
			if event.Has(fsnotify.Write) || event.Has(fsnotify.Create) || event.Has(fsnotify.Remove) {
				log.Printf("Config file changed: %s", event.Name)
				// Debounce: reset timer on each event to avoid redundant reloads
				if cm.debounceTimer != nil {
					cm.debounceTimer.Stop()
				}
				cm.debounceTimer = time.AfterFunc(debounceDelay, func() {
					if err := cm.Reload(); err != nil {
						log.Printf("Error reloading config: %v", err)
					}
				})
			}
		case err, ok := <-cm.watcher.Errors:
			if !ok {
				return
			}
			log.Printf("Watcher error: %v", err)
		case <-cm.closeChan:
			return
		}
	}
}

func (cm *ConfigManager) Reload() error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if cm.configPath != "" {
		log.Printf("Reloading configuration from: %s", cm.configPath)
	} else {
		log.Printf("Reloading configuration from default locations")
	}

	// 1. Reload file from disk
	newCfg, files, err := loadConfig(cm.configPath)
	if err != nil {
		return err
	}

	// 2. Rebuild factories
	clientFactory, err := factory.GetLogClientFactory(newCfg.Clients)
	if err != nil {
		return fmt.Errorf("failed to build client factory: %w", err)
	}

	searchFactory, err := factory.GetLogSearchFactory(clientFactory, *newCfg)
	if err != nil {
		return fmt.Errorf("failed to build search factory: %w", err)
	}

	// 3. Update state
	cm.currentCfg = newCfg
	cm.searchFactory = searchFactory

	// 4. Update watcher
	// First, remove old files from watcher to prevent resource leaks
	for _, f := range cm.loadedFiles {
		// It's safe to ignore errors here, as the file might have been deleted.
		_ = cm.watcher.Remove(f)
	}
	// Then, add new files to watcher
	for _, f := range files {
		if err := cm.watcher.Add(f); err != nil {
			log.Printf("Failed to watch file %s: %v", f, err)
		}
	}
	cm.loadedFiles = files

	return nil
}

// Get returns a thread-safe snapshot of the current configuration and search factory.
func (cm *ConfigManager) Get() (*config.ContextConfig, factory.SearchFactory) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.currentCfg, cm.searchFactory
}

// Close gracefully shuts down the ConfigManager, stopping the watcher and cleaning up resources.
func (cm *ConfigManager) Close() error {
	close(cm.closeChan)
	if cm.debounceTimer != nil {
		cm.debounceTimer.Stop()
	}
	return cm.watcher.Close()
}

// BuildMCPServer creates an MCP server instance with all tools/resources/prompts registered.
// Exposed for testing so we can spin up the server without invoking cobra.Run path.
type MCPServerBundle struct {
	Server       *server.MCPServer
	ToolHandlers map[string]func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error)
}

func BuildMCPServer(configPath string) (*MCPServerBundle, error) {
	// Initialize config manager
	cm, err := NewConfigManager(configPath)
	if err != nil {
		return nil, err
	}
	return buildMCPServerWithManager(cm)
}

// buildMCPServerWithManager creates the MCP server with a provided ConfigManager.
// Internal function for testing.
func buildMCPServerWithManager(cm *ConfigManager) (*MCPServerBundle, error) {
	s := server.NewMCPServer(
		"logviewer",
		"1.0.0",
		server.WithToolCapabilities(true),
		server.WithRecovery(),
	)

	handlers := map[string]func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error){}

	// --- Tool: reload_config ---
	reloadTool := mcp.NewTool("reload_config",
		mcp.WithDescription("Reload the configuration file from disk. Use this if you have modified the config.yaml file."),
	)
	reloadHandler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		if err := cm.Reload(); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Reload failed: %v", err)), nil
		}
		return mcp.NewToolResultText("Configuration successfully reloaded."), nil
	}
	s.AddTool(reloadTool, reloadHandler)
	handlers["reload_config"] = reloadHandler

	listContextsTool := mcp.NewTool("list_contexts",
		mcp.WithDescription(`List all configured log contexts.

Usage: list_contexts

Returns: JSON array of context identifiers (strings) that can be used in other tools.

Note: You don't have to call this before every query. You can attempt query_logs directly; if the contextId is invalid the server will now return suggestions including available contexts.
`),
	)
	listHandler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		cfg, _ := cm.Get()
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
		_, searchFactory := cm.Get()

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
	start_time (string, optional): Absolute start time (RFC3339).
	end_time (string, optional): Absolute end time (RFC3339).
	pageToken (string, optional): Token for pagination to fetch older logs (returned in previous response meta).
	size (number, optional): Max number of log entries.
	fields (object, optional): Exact-match key/value filters.
	nativeQuery (string, optional): Raw query in the backend's native syntax (e.g. Splunk SPL, OpenSearch Lucene).
		Use this for advanced queries that leverage backend-specific features.
		The nativeQuery acts as the base search; fields filters are appended to refine results.
		Examples:
		- Splunk: "index=main sourcetype=httpevent | eval severity=if(level==\"ERROR\", \"HIGH\", \"LOW\")"
		- OpenSearch: "level:ERROR AND message:*timeout*"

Behavior improvements:
	- If contextId is invalid, the response includes suggestions (no need to pre-call list_contexts).
	- If results are empty, meta.hints will recommend next actions (e.g. broaden last, call get_fields).
	- If more results are available, meta.nextPageToken will be included for pagination.

Returns: { "entries": [...], "meta": { resultCount, contextId, queryTime, hints?, nextPageToken? } }
`),
		mcp.WithString("contextId", mcp.Required(), mcp.Description("Context identifier to query.")),
		mcp.WithString("last", mcp.Description(`Relative time window like 15m, 2h, 1d.`)),
		mcp.WithString("start_time", mcp.Description("Absolute start time (RFC3339).")),
		mcp.WithString("end_time", mcp.Description("Absolute end time (RFC3339).")),
		mcp.WithString("pageToken", mcp.Description("Token for pagination to fetch older logs (returned in previous response meta).")),
		mcp.WithObject("fields", mcp.Description("Exact match key/value filters (JSON object).")),
		mcp.WithNumber("size", mcp.Description("Maximum number of log entries to return.")),
		mcp.WithString("nativeQuery", mcp.Description("Raw query in backend's native syntax (Splunk SPL, OpenSearch Lucene). Acts as base search with filters appended.")),
		mcp.WithObject("variables", mcp.Description("Runtime variables for the context (JSON object).")),
	)
	queryLogsHandler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		cfg, searchFactory := cm.Get()
		start := time.Now()
		contextId, err := request.RequireString("contextId")
		if err != nil || contextId == "" {
			return mcp.NewToolResultError(fmt.Sprintf("invalid or missing contextId: %v", err)), nil
		}

		searchRequest := client.LogSearch{}
		if last, err := request.RequireString("last"); err == nil && last != "" {
			searchRequest.Range.Last.S(last)
		}
		if startTime, err := request.RequireString("start_time"); err == nil && startTime != "" {
			searchRequest.Range.Gte.S(startTime)
		}
		if endTime, err := request.RequireString("end_time"); err == nil && endTime != "" {
			searchRequest.Range.Lte.S(endTime)
		}
		if token, err := request.RequireString("pageToken"); err == nil && token != "" {
			searchRequest.PageToken.S(token)
		}
		if size, err := request.RequireFloat("size"); err == nil && int(size) > 0 {
			searchRequest.Size.S(int(size))
		}
		if nativeQuery, err := request.RequireString("nativeQuery"); err == nil && nativeQuery != "" {
			searchRequest.NativeQuery.S(nativeQuery)
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
		if pagination := searchResult.GetPaginationInfo(); pagination != nil && pagination.NextPageToken != "" {
			meta["nextPageToken"] = pagination.NextPageToken
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

	// --- Tool: get_field_values ---
	getFieldValuesTool := mcp.NewTool("get_field_values",
		mcp.WithDescription(`Get distinct values for specific log fields to understand data distribution or find specific values.

Usage: get_field_values contextId=<context> fields=["level","error_code"] [last=15m]

Parameters:
  contextId (string, required): Context identifier.
  fields (array of strings, required): Field names to get distinct values for.
  last (string, optional): Relative time window (e.g. 15m, 2h). Defaults to 15m.
  start_time (string, optional): Absolute start time (RFC3339).
  end_time (string, optional): Absolute end time (RFC3339).
  filters (object, optional): Additional key/value filters to apply.

Returns: JSON object mapping field names to arrays of distinct values.

Example response:
{
  "level": ["ERROR", "WARN", "INFO"],
  "error_code": ["TIMEOUT", "AUTH_FAILURE", "DB_CONN_ERR"]
}
`),
		mcp.WithString("contextId", mcp.Required(), mcp.Description("Context identifier to query.")),
		mcp.WithArray("fields", mcp.Required(), mcp.Description("Field names to get distinct values for (array of strings).")),
		mcp.WithString("last", mcp.Description("Relative time window like 15m, 2h, 1d.")),
		mcp.WithString("start_time", mcp.Description("Absolute start time (RFC3339).")),
		mcp.WithString("end_time", mcp.Description("Absolute end time (RFC3339).")),
		mcp.WithObject("filters", mcp.Description("Additional key/value filters to apply (JSON object).")),
	)
	getFieldValuesHandler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		cfg, searchFactory := cm.Get()
		contextId, err := request.RequireString("contextId")
		if err != nil || contextId == "" {
			return mcp.NewToolResultError(fmt.Sprintf("invalid or missing contextId: %v", err)), nil
		}

		// Extract fields array
		var fieldNames []string
		args := request.GetArguments()
		if args != nil {
			if rawFields, ok := args["fields"]; ok && rawFields != nil {
				switch v := rawFields.(type) {
				case []interface{}:
					for _, f := range v {
						if s, ok := f.(string); ok {
							fieldNames = append(fieldNames, s)
						}
					}
				case []string:
					fieldNames = v
				}
			}
		}
		if len(fieldNames) == 0 {
			return mcp.NewToolResultError("fields parameter is required and must be a non-empty array of field names"), nil
		}

		searchRequest := client.LogSearch{}
		if last, err := request.RequireString("last"); err == nil && last != "" {
			searchRequest.Range.Last.S(last)
		}
		if startTime, err := request.RequireString("start_time"); err == nil && startTime != "" {
			searchRequest.Range.Gte.S(startTime)
		}
		if endTime, err := request.RequireString("end_time"); err == nil && endTime != "" {
			searchRequest.Range.Lte.S(endTime)
		}

		// Handle filters
		if args != nil {
			if rawFilters, ok := args["filters"]; ok && rawFilters != nil {
				if filterMap, ok := rawFilters.(map[string]any); ok {
					if searchRequest.Fields == nil {
						searchRequest.Fields = ty.MS{}
					}
					for k, v := range filterMap {
						searchRequest.Fields[k] = fmt.Sprintf("%v", v)
					}
				}
			}
		}

		// Fallback: ensure some time window is always specified
		if !searchRequest.Range.Last.Set && !searchRequest.Range.Gte.Set {
			searchRequest.Range.Last.S("15m")
		}

		// Pre-flight check for context existence
		_, err = searchFactory.GetSearchContext(ctx, contextId, []string{}, searchRequest, nil)
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
			return mcp.NewToolResultError(fmt.Sprintf("failed to get search context: %v", err)), nil
		}

		fieldValues, err := searchFactory.GetFieldValues(ctx, contextId, []string{}, searchRequest, fieldNames, nil)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to get field values: %v", err)), nil
		}

		jsonBytes, err := json.Marshal(fieldValues)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to marshal field values: %v", err)), nil
		}
		return mcp.NewToolResultText(string(jsonBytes)), nil
	}
	s.AddTool(getFieldValuesTool, getFieldValuesHandler)
	handlers["get_field_values"] = getFieldValuesHandler

	getContextDetailsTool := mcp.NewTool("get_context_details",
		mcp.WithDescription("Inspect a context's details, including its variable schema."),
		mcp.WithString("contextId", mcp.Required(), mcp.Description("The context ID to inspect.")),
	)
	getContextDetailsHandler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		cfg, searchFactory := cm.Get()
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
		cfg, _ := cm.Get()
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
		cfg, _ := cm.Get()
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
