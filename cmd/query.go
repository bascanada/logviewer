package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"sort"
	"strings"
	"sync"

	"github.com/bascanada/logviewer/pkg/log/client"
	"github.com/bascanada/logviewer/pkg/log/client/config"
	"github.com/bascanada/logviewer/pkg/log/factory"
	"github.com/bascanada/logviewer/pkg/log/impl/cloudwatch"
	"github.com/bascanada/logviewer/pkg/log/impl/docker"
	"github.com/bascanada/logviewer/pkg/log/impl/elk/kibana"
	"github.com/bascanada/logviewer/pkg/log/impl/elk/opensearch"
	"github.com/bascanada/logviewer/pkg/log/impl/k8s"
	"github.com/bascanada/logviewer/pkg/log/impl/local"
	splunk "github.com/bascanada/logviewer/pkg/log/impl/splunk/logclient"
	"github.com/bascanada/logviewer/pkg/log/impl/ssh"
	"github.com/bascanada/logviewer/pkg/log/printer"
	"github.com/bascanada/logviewer/pkg/query"
	"github.com/bascanada/logviewer/pkg/ty"

	"github.com/spf13/cobra"
)

func stringArrayEnvVariable(strs []string, maps *ty.MS) error {
	for _, f := range strs {
		if strings.Contains(f, "=") {
			items := strings.SplitN(f, "=", 2)
			key := items[0]
			val := items[1]

			// empty key (e.g. "=error") is treated as a free-text token
			if key == "" {
				if prev, ok := (*maps)[""]; ok && prev != "" {
					(*maps)[""] = prev + " " + val
				} else {
					(*maps)[""] = val
				}
			} else {
				(*maps)[key] = val
			}
			continue
		}

		// No '=' present: treat the whole string as a free-text token and
		// append it to any existing free-text tokens.
		if prev, ok := (*maps)[""]; ok && prev != "" {
			(*maps)[""] = prev + " " + f
		} else {
			(*maps)[""] = f
		}
	}
	return nil
}

// buildSearchRequest creates a LogSearch from CLI flags
func buildSearchRequest() client.LogSearch {
	searchRequest := client.LogSearch{
		Fields:          ty.MS{},
		FieldsCondition: ty.MS{},
		Options:         ty.MI{},
	}

	if size > 0 {
		searchRequest.Size.S(size)
	}
	if pageToken != "" {
		searchRequest.PageToken.S(pageToken)
	}
	if duration != "" {
		searchRequest.Refresh.Duration.S(duration)
	}
	if groupRegex != "" {
		searchRequest.FieldExtraction.GroupRegex.S(groupRegex)
	}
	if kvRegex != "" {
		searchRequest.FieldExtraction.KvRegex.S(kvRegex)
	}
	if to != "" {
		normalizedTo, _ := ty.NormalizeTimeValue(to)
		searchRequest.Range.Lte.S(normalizedTo)
	}
	if from != "" {
		normalizedFrom, _ := ty.NormalizeTimeValue(from)
		searchRequest.Range.Gte.S(normalizedFrom)
	}
	if last != "" {
		searchRequest.Range.Last.S(last)
	}
	// Parse fields: auto-detect hl syntax vs legacy syntax
	if len(fields) > 0 {
		var hlFields []string
		var legacyFields []string

		for _, f := range fields {
			if query.IsHLSyntax(f) {
				hlFields = append(hlFields, f)
			} else {
				legacyFields = append(legacyFields, f)
			}
		}

		// Process legacy fields (field=value)
		if len(legacyFields) > 0 {
			stringArrayEnvVariable(legacyFields, &searchRequest.Fields)
		}

		// Process hl-syntax fields into Filter
		if len(hlFields) > 0 {
			hlFilter, err := query.ParseFilterFlags(hlFields)
			if err != nil {
				fmt.Fprintf(os.Stderr, "warning: failed to parse filter: %v\n", err)
			} else if hlFilter != nil {
				// Merge with existing filter if any
				if searchRequest.Filter == nil {
					searchRequest.Filter = hlFilter
				} else {
					// Combine existing filter with new hl filter using AND
					searchRequest.Filter = &client.Filter{
						Logic:   client.LogicAnd,
						Filters: []client.Filter{*searchRequest.Filter, *hlFilter},
					}
				}
			}
		}
	}
	if len(fieldsOps) > 0 {
		stringArrayEnvVariable(fieldsOps, &searchRequest.FieldsCondition)
	}

	// Parse -q/--query expression
	if queryExpr != "" {
		queryFilter, err := query.ParseQueryExpression(queryExpr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to parse query expression: %v\n", err)
		} else if queryFilter != nil {
			// Merge with existing filter if any
			if searchRequest.Filter == nil {
				searchRequest.Filter = queryFilter
			} else {
				// Combine existing filter with query filter using AND
				searchRequest.Filter = &client.Filter{
					Logic:   client.LogicAnd,
					Filters: []client.Filter{*searchRequest.Filter, *queryFilter},
				}
			}
		}
	}

	if index != "" {
		searchRequest.Options["index"] = index
	}
	if k8sContainer != "" {
		searchRequest.Options[k8s.FieldContainer] = k8sContainer
	}
	if k8sNamespace != "" {
		searchRequest.Options[k8s.FieldNamespace] = k8sNamespace
	}
	if k8sPod != "" {
		searchRequest.Options[k8s.FieldPod] = k8sPod
	}
	if k8sLabelSelector != "" {
		searchRequest.Options[k8s.FieldLabelSelector] = k8sLabelSelector
	}
	if k8sPrevious {
		searchRequest.Options[k8s.FieldPrevious] = k8sPrevious
	}
	if k8sTimestamp {
		searchRequest.Options[k8s.OptionsTimestamp] = k8sTimestamp
	}
	if cmd != "" {
		searchRequest.Options[local.OptionsCmd] = cmd
	}
	if sshOptions.DisablePTY {
		searchRequest.Options["disablePTY"] = true
	}
	if template != "" {
		searchRequest.PrinterOptions.Template.S(template)
	}
	if dockerContainer != "" {
		searchRequest.Options["container"] = dockerContainer
	}
	if dockerService != "" {
		searchRequest.Options["service"] = dockerService
	}
	if dockerProject != "" {
		searchRequest.Options["project"] = dockerProject
	}
	if nativeQuery != "" {
		searchRequest.NativeQuery.S(nativeQuery)
	}

	searchRequest.Follow = refresh

	return searchRequest
}

// parseRuntimeVars parses --var flags into a map
func parseRuntimeVars() map[string]string {
	runtimeVars := make(map[string]string)
	for _, v := range vars {
		parts := strings.SplitN(v, "=", 2)
		if len(parts) == 2 {
			runtimeVars[parts[0]] = parts[1]
		}
	}
	return runtimeVars
}

// resolveContextIdsFromConfig resolves context IDs, using current context if none specified
func resolveContextIdsFromConfig(cfg *config.ContextConfig) []string {
	if len(contextIds) > 0 {
		return contextIds
	}
	if cfg.CurrentContext != "" {
		if _, ok := cfg.Contexts[cfg.CurrentContext]; ok {
			return []string{cfg.CurrentContext}
		}
	}
	return []string{}
}

// isAdHocQuery returns true if CLI flags indicate an ad-hoc query (no config)
func isAdHocQuery() bool {
	return endpointOpensearch != "" ||
		endpointKibana != "" ||
		cloudwatchLogGroup != "" ||
		(k8sNamespace != "" && len(contextIds) == 0 && configPath == "") ||
		(cmd != "" && len(contextIds) == 0 && configPath == "") ||
		endpointSplunk != "" ||
		((dockerContainer != "" || dockerService != "") && len(contextIds) == 0 && configPath == "")
}

// getAdHocLogClient creates a LogClient from ad-hoc CLI flags
func getAdHocLogClient(searchRequest *client.LogSearch) (client.LogClient, error) {
	var err error
	var system string

	if endpointOpensearch != "" {
		system = "opensearch"
	} else if endpointKibana != "" {
		system = "kibana"
	} else if cloudwatchLogGroup != "" {
		system = "cloudwatch"
	} else if k8sNamespace != "" {
		system = "k8s"
	} else if cmd != "" {
		if sshOptions.Addr != "" {
			system = "ssh"
		} else {
			system = "local"
		}
	} else if endpointSplunk != "" {
		system = "splunk"
	} else if dockerContainer != "" || dockerService != "" {
		system = "docker"
	} else {
		return nil, errors.New(`
        failed to select a system for logging provide one of the following:
			* --docker-container or --docker-service
			* --splunk-endpoint
			* --kibana-endpoint
            * --opensearch-endpoint
            * --k8s-namespace
            * --ssh-addr
            * --cmd
        `)
	}

	var logClient client.LogClient

	switch system {
	case "opensearch":
		logClient, err = opensearch.GetClient(opensearch.OpenSearchTarget{Endpoint: endpointOpensearch})
	case "kibana":
		logClient, err = kibana.GetClient(kibana.KibanaTarget{Endpoint: endpointKibana})
	case "cloudwatch":
		opts := ty.MI{}
		if cloudwatchRegion != "" {
			opts["region"] = cloudwatchRegion
		}
		if cloudwatchProfile != "" {
			opts["profile"] = cloudwatchProfile
		}
		if cloudwatchEndpoint != "" {
			opts["endpoint"] = cloudwatchEndpoint
		}
		if cloudwatchLogGroup != "" {
			searchRequest.Options["logGroupName"] = cloudwatchLogGroup
		}
		searchRequest.Options["useInsights"] = fmt.Sprintf("%v", cloudwatchUseInsights)
		if cloudwatchPollInterval != "" {
			searchRequest.Options["cloudwatchPollInterval"] = cloudwatchPollInterval
		}
		if cloudwatchMaxPollInterval != "" {
			searchRequest.Options["cloudwatchMaxPollInterval"] = cloudwatchMaxPollInterval
		}
		if cloudwatchPollBackoff != "" {
			searchRequest.Options["cloudwatchPollBackoff"] = cloudwatchPollBackoff
		}
		logClient, err = cloudwatch.GetLogClient(opts)
	case "k8s":
		logClient, err = k8s.GetLogClient(k8s.K8sLogClientOptions{})
	case "ssh":
		logClient, err = ssh.GetLogClient(sshOptions)
	case "docker":
		logClient, err = docker.GetLogClient(dockerHost)
	case "splunk":
		headers := ty.MS{}
		body := ty.MS{"output_mode": "json"} // Default to JSON output
		if headerField != "" {
			if err = headers.LoadMS(headerField); err != nil {
				return nil, err
			}
			headers = headers.ResolveVariables()
		}
		if bodyField != "" {
			if err = body.LoadMS(bodyField); err != nil {
				return nil, err
			}
			body = body.ResolveVariables()
		}
		logClient, err = splunk.GetClient(splunk.SplunkLogSearchClientOptions{
			Url:        endpointSplunk,
			SearchBody: body,
			Headers:    headers,
		})
	default:
		logClient, err = local.GetLogClient()
	}

	return logClient, err
}

func resolveSearch() (client.LogSearchResult, error) {
	searchRequest := buildSearchRequest()

	// Check if this is a config-based query
	if configPath != "" || len(contextIds) > 0 {
		cfg, _, err := loadConfig(configPath)
		if err != nil {
			return nil, err
		}

		clientFactory, err := factory.GetLogClientFactory(cfg.Clients)
		if err != nil {
			return nil, err
		}

		searchFactory, err := factory.GetLogSearchFactory(clientFactory, *cfg)
		if err != nil {
			return nil, err
		}

		runtimeVars := parseRuntimeVars()
		resolvedContextIds := resolveContextIdsFromConfig(cfg)

		if len(resolvedContextIds) == 0 {
			return nil, errors.New("no contexts specified for query; use -i to select one or more contexts or set a default with 'logviewer context use'")
		}

		// For single context, execute directly without MultiLogSearchResult wrapper
		if len(resolvedContextIds) == 1 {
			ctx := context.Background()
			searchRequest.Options["__context_id__"] = resolvedContextIds[0]
			return searchFactory.GetSearchResult(ctx, resolvedContextIds[0], inherits, searchRequest, runtimeVars)
		}

		// Fan-out: execute queries for each context concurrently.
		multiResult, err := client.NewMultiLogSearchResult(&searchRequest)
		if err != nil {
			return nil, err
		}
		var wg sync.WaitGroup
		ctx := context.Background()

		for _, contextId := range resolvedContextIds {
			wg.Add(1)
			go func(cid string) {
				defer wg.Done()
				reqCopy := searchRequest
				reqCopy.Options = ty.MergeM(make(ty.MI, len(searchRequest.Options)+1), searchRequest.Options)
				reqCopy.Options["__context_id__"] = cid
				reqCopy.Fields = ty.MergeM(make(ty.MS, len(searchRequest.Fields)), searchRequest.Fields)
				reqCopy.FieldsCondition = ty.MergeM(make(ty.MS, len(searchRequest.FieldsCondition)), searchRequest.FieldsCondition)
				if searchRequest.Variables != nil {
					reqCopy.Variables = make(map[string]client.VariableDefinition, len(searchRequest.Variables))
					for k, v := range searchRequest.Variables {
						reqCopy.Variables[k] = v
					}
				}
				sr, err := searchFactory.GetSearchResult(ctx, cid, inherits, reqCopy, runtimeVars)
				multiResult.Add(sr, err)
			}(contextId)
		}

		wg.Wait()

		if len(multiResult.Errors) > 0 {
			var errorStrings []string
			for _, e := range multiResult.Errors {
				errorStrings = append(errorStrings, e.Error())
			}
			fmt.Fprintf(os.Stderr, "errors encountered for some contexts:\n%s\n", strings.Join(errorStrings, "\n"))
		}
		return multiResult, nil
	}

	// Ad-hoc query (no config)
	if headerField != "" {
		headerMap := ty.MS{}
		if err := headerMap.LoadMS(headerField); err != nil {
			return nil, err
		}
	}

	logClient, err := getAdHocLogClient(&searchRequest)
	if err != nil {
		return nil, err
	}

	searchResult, err := logClient.Get(context.Background(), &searchRequest)
	if err != nil {
		return nil, err
	}

	return searchResult, nil
}

// resolveFieldValues handles both config-based and ad-hoc queries for field values
func resolveFieldValues(fieldNames []string) (map[string][]string, error) {
	searchRequest := buildSearchRequest()
	ctx := context.Background()

	// Check if this is an ad-hoc query
	if isAdHocQuery() {
		logClient, err := getAdHocLogClient(&searchRequest)
		if err != nil {
			return nil, err
		}
		return logClient.GetFieldValues(ctx, &searchRequest, fieldNames)
	}

	// Config-based query
	if configPath == "" && len(contextIds) == 0 {
		return nil, errors.New("no config or context specified; use -i to select a context or provide endpoint flags for ad-hoc query")
	}

	cfg, _, err := loadConfig(configPath)
	if err != nil {
		return nil, err
	}

	clientFactory, err := factory.GetLogClientFactory(cfg.Clients)
	if err != nil {
		return nil, err
	}

	searchFactory, err := factory.GetLogSearchFactory(clientFactory, *cfg)
	if err != nil {
		return nil, err
	}

	runtimeVars := parseRuntimeVars()
	resolvedContextIds := resolveContextIdsFromConfig(cfg)

	if len(resolvedContextIds) == 0 {
		return nil, errors.New("no context specified; use -i to select a context")
	}

	// For multiple contexts, aggregate results using sets for efficiency
	allResultsSet := make(map[string]map[string]struct{})
	var hasError bool

	for _, contextId := range resolvedContextIds {
		if len(resolvedContextIds) > 1 {
			fmt.Fprintf(os.Stderr, "=== Context: %s ===\n", contextId)
		}

		fieldValues, err := searchFactory.GetFieldValues(ctx, contextId, inherits, searchRequest, fieldNames, runtimeVars)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error getting field values for %s: %v\n", contextId, err)
			hasError = true
			continue
		}

		// Merge results into a set for uniqueness
		for field, values := range fieldValues {
			if _, ok := allResultsSet[field]; !ok {
				allResultsSet[field] = make(map[string]struct{})
			}
			for _, v := range values {
				allResultsSet[field][v] = struct{}{}
			}
		}
	}

	if hasError && len(allResultsSet) == 0 {
		return nil, errors.New("failed to get field values from all contexts")
	}

	// Convert sets to sorted slices for deterministic output
	allResults := make(map[string][]string, len(allResultsSet))
	for field, valuesSet := range allResultsSet {
		values := make([]string, 0, len(valuesSet))
		for v := range valuesSet {
			values = append(values, v)
		}
		sort.Strings(values)
		allResults[field] = values
	}

	return allResults, nil
}

var queryFieldCommand = &cobra.Command{
	Use:    "field",
	Short:  "Dispaly available field for filtering of logs",
	PreRun: onCommandStart,
	Run: func(cmd *cobra.Command, args []string) {
		searchResult, err1 := resolveSearch()

		if err1 != nil {
			fmt.Fprintln(os.Stderr, "error:", err1)
			os.Exit(1)
		}
		searchResult.GetEntries(context.Background())
		fields, _, err := searchResult.GetFields(context.Background())
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}

		for k, b := range fields {
			fmt.Printf("%s \n", k)
			for _, r := range b {
				fmt.Println("    " + r)
			}
		}

	},
}

var queryLogCommand = &cobra.Command{
	Use:    "log",
	Short:  "Display logs for system",
	PreRun: onCommandStart,
	Run: func(cmd *cobra.Command, args []string) {
		searchResult, err1 := resolveSearch()

		if err1 != nil {
			fmt.Fprintln(os.Stderr, "error:", err1)
			os.Exit(1)
		}

		if paginationInfo := searchResult.GetPaginationInfo(); paginationInfo != nil && paginationInfo.HasMore {
			fmt.Fprintf(os.Stderr, "More results available. To fetch the next page, run the same command with --page-token \"%s\"\n", paginationInfo.NextPageToken)
		}

		if jsonOutput {
			// Machine Mode (NDJSON for lnav/jq)
			enc := json.NewEncoder(os.Stdout)
			entries, c, err := searchResult.GetEntries(context.Background())

			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}

			// Helper to encode a slice of entries
			printJSON := func(es []client.LogEntry) error {
				for i := range es {
					// Extract JSON fields if enabled
					client.ExtractJSONFromEntry(&es[i], searchResult.GetSearch())
					if err := enc.Encode(es[i]); err != nil {
						return err
					}
				}
				return nil
			}

			if err := printJSON(entries); err != nil {
				fmt.Fprintf(os.Stderr, "Error writing initial JSON output: %v\n", err)
				os.Exit(1)
			}

			// Handle live/follow mode
			if c != nil {
				for newEntries := range c {
					if err := printJSON(newEntries); err != nil {
						fmt.Fprintf(os.Stderr, "Error writing streaming JSON output: %v\n", err)
						break
					}
				}
			}
			return // End execution for this mode
		}

		outputter := printer.PrintPrinter{}
		onError := func(err error) {
			fmt.Fprintf(os.Stderr, "Error displaying logs: %v\n", err)
			os.Exit(1)
		}
		continous, err := outputter.Display(context.Background(), searchResult, onError)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error displaying logs: %v\n", err)
			os.Exit(1)
		}
		if continous {
			c := make(chan os.Signal, 1)
			signal.Notify(c, os.Interrupt)
			<-c
			// try to close the search result, if it supports it
			if closer, ok := searchResult.(interface{ Close() error }); ok {
				if err := closer.Close(); err != nil {
					fmt.Fprintln(os.Stderr, "error closing search:", err)
				}
			}
		}
	},
}

var queryValuesCommand = &cobra.Command{
	Use:   "values [field...]",
	Short: "Get distinct values for specific fields from logs",
	Long: `Get distinct values for one or more specific fields from logs.

This command efficiently retrieves distinct values for the specified fields,
respecting current filters and time range.

Examples:
  # Get distinct values for a single field
  logviewer query values -i prod-logs error_code --last 1h

  # Get distinct values for multiple fields
  logviewer query values -i prod-logs level service error_code --last 2h

  # With filters applied
  logviewer query values -i prod-logs error_code -f level=ERROR --last 1h

  # Ad-hoc query (without config)
  logviewer query values level app --opensearch-endpoint http://localhost:9200 --elk-index app-logs --last 1h`,
	PreRun: onCommandStart,
	Args:   cobra.MinimumNArgs(1), // Require at least one field
	Run: func(cmd *cobra.Command, args []string) {
		fieldNames := args

		fieldValues, err := resolveFieldValues(fieldNames)
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}

		if jsonOutput {
			enc := json.NewEncoder(os.Stdout)
			if err := enc.Encode(fieldValues); err != nil {
				fmt.Fprintf(os.Stderr, "error encoding JSON: %v\n", err)
				os.Exit(1)
			}
		} else {
			// Output as formatted text (same format as query field)
			for _, field := range fieldNames {
				values, ok := fieldValues[field]
				if !ok || len(values) == 0 {
					fmt.Printf("%s \n", field)
					fmt.Println("    (no values found)")
					continue
				}
				fmt.Printf("%s \n", field)
				for _, v := range values {
					fmt.Println("    " + v)
				}
			}
		}
	},
}

var queryCommand = &cobra.Command{
	Use:    "query",
	Short:  "Query a login system for logs and available fields",
	PreRun: onCommandStart,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Println("Please use 'logviewer query log' to stream logs, 'logviewer query field' to inspect fields, or 'logviewer query values' to get distinct values.")
		cmd.Help()
	},
}
