package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
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

func resolveSearch() (client.LogSearchResult, error) {

	// resolve this from args
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
		searchRequest.Range.Lte.S(to)
	}

	if from != "" {
		searchRequest.Range.Gte.S(from)
	}

	if last != "" {
		searchRequest.Range.Last.S(last)
	}

	if len(fields) > 0 {
		stringArrayEnvVariable(fields, &searchRequest.Fields)
	}

	if len(fieldsOps) > 0 {
		stringArrayEnvVariable(fieldsOps, &searchRequest.FieldsCondition)
	}

	if index != "" {
		// use lowercase `index` consistently with splunk mapper and tests
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

	if k8sPrevious {
		searchRequest.Options[k8s.FieldPrevious] = k8sPrevious
	}

	if k8sTimestamp {
		searchRequest.Options[k8s.OptionsTimestamp] = k8sTimestamp
	}

	if cmd != "" {
		searchRequest.Options[local.OptionsCmd] = cmd
	}

	if template != "" {
		searchRequest.PrinterOptions.Template.S(template)
	}

	searchRequest.Refresh.Follow.S(refresh)

	// Centralized config handling:
	// - If an explicit configPath is given, use it.
	// - If no configPath but context ids (-i) are provided, attempt to load the default config.
	// - If no configPath and no -i, do not load any config and continue with non-config flow.
	if configPath != "" || len(contextIds) > 0 {
		var cfg *config.ContextConfig
		var err error
		if configPath != "" {
			cfg, err = config.LoadContextConfig(configPath)
		} else {
			cfg, err = config.LoadContextConfig("")
		}

		if err != nil {
			// Handle all config loading errors uniformly.
			errorMsg := "failed to load context config"
			switch {
			case errors.Is(err, config.ErrConfigParse):
				errorMsg = "invalid configuration file format"
			case errors.Is(err, config.ErrNoClients):
				errorMsg = "configuration missing 'clients' section"
			case errors.Is(err, config.ErrNoContexts):
				errorMsg = "configuration missing 'contexts' section"
			}
			if configPath != "" {
				return nil, fmt.Errorf("%s %s: %w", errorMsg, configPath, err)
			}
			return nil, fmt.Errorf("%s: %w", errorMsg, err)
		}

		clientFactory, err := factory.GetLogClientFactory(cfg.Clients)
		if err != nil {
			return nil, err
		}

		searchFactory, err := factory.GetLogSearchFactory(clientFactory, *cfg)
		if err != nil {
			return nil, err
		}

		// Parse --var flags into a map for runtime variable substitution.
		runtimeVars := make(map[string]string)
		for _, v := range vars {
			parts := strings.SplitN(v, "=", 2)
			if len(parts) == 2 {
				runtimeVars[parts[0]] = parts[1]
			}
		}

		// If no contexts are specified via -i, and we are not in an interactive view,
		// there's nothing to query.
		if len(contextIds) == 0 {
			// This check is to prevent trying to query nothing when in non-interactive mode.
			// The interactive view has its own logic for handling context selection.
			// Note: This part of the logic might need adjustment depending on the exact desired CLI behavior
			// when no contexts are provided.
			return nil, errors.New("no contexts specified for query; use -i to select one or more contexts")
		}

		// For single context, execute directly without MultiLogSearchResult wrapper
		if len(contextIds) == 1 {
			ctx := context.Background()
			return searchFactory.GetSearchResult(ctx, contextIds[0], inherits, searchRequest, runtimeVars)
		}

		// Fan-out: execute queries for each context concurrently.
		multiResult, err := client.NewMultiLogSearchResult(&searchRequest)
		if err != nil {
			return nil, err
		}
		var wg sync.WaitGroup
		ctx := context.Background()

		for _, contextId := range contextIds {
			wg.Add(1)
			go func(cid string) {
				defer wg.Done()
				// The search request is copied to avoid data races.
				reqCopy := searchRequest
				// Deep copy map fields to avoid concurrent map writes.
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

		// Print errors for failed contexts to stderr.
		if len(multiResult.Errors) > 0 {
			var errorStrings []string
			for _, e := range multiResult.Errors {
				errorStrings = append(errorStrings, e.Error())
			}
			fmt.Fprintf(os.Stderr, "errors encountered for some contexts:\n%s\n", strings.Join(errorStrings, "\n"))
		}
		return multiResult, nil
	}

	if headerField != "" {
		headerMap := ty.MS{}

		if err := headerMap.LoadMS(headerField); err != nil {
			return nil, err
		}

	}

	if dockerContainer != "" {

		searchRequest.Options["Container"] = dockerContainer
	}

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
	} else if dockerContainer != "" {
		system = "docker"
	} else {
		return nil, errors.New(`
        failed to select a system for logging provide one of the following:
			* --docker-container
			* --splunk-endpoint
			* --kibana-endpoint
            * --openseach-endpoint
            * --k8s-namespace
            * --ssh-addr
            * --cmd
        `)
	}

	var logClient client.LogClient

	if system == "opensearch" {
		logClient, err = opensearch.GetClient(opensearch.OpenSearchTarget{Endpoint: endpointOpensearch})
	} else if system == "kibana" {
		logClient, err = kibana.GetClient(kibana.KibanaTarget{Endpoint: endpointKibana})
	} else if system == "cloudwatch" {
		// Build options map expected by cloudwatch.GetLogClient
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
		// These options are per-search rather than client creation, push into search.Options
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
	} else if system == "k8s" {
		logClient, err = k8s.GetLogClient(k8s.K8sLogClientOptions{})
	} else if system == "ssh" {
		logClient, err = ssh.GetLogClient(sshOptions)
	} else if system == "docker" {
		logClient, err = docker.GetLogClient(dockerHost)
	} else if system == "splunk" {
		headers := ty.MS{}
		body := ty.MS{}
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
	} else {
		logClient, err = local.GetLogClient()
	}
	if err != nil {
		return nil, err
	}

	searchResult, err2 := logClient.Get(context.Background(), &searchRequest)
	if err2 != nil {
		return nil, err2
	}

	return searchResult, nil

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

		outputter := printer.PrintPrinter{}
		continous, err := outputter.Display(context.Background(), searchResult)
		if err != nil {
			panic(err)
		}
		if continous {
			c := make(chan os.Signal, 1)
			signal.Notify(c, os.Interrupt)
			<-c
		}
	},
}

var queryCommand = &cobra.Command{
	Use:    "query",
	Short:  "Query a login system for logs and available fields",
	PreRun: onCommandStart,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Please use 'logviewer query log' to stream logs or 'logviewer query field' to inspect fields.")
		cmd.Help()
	},
}
