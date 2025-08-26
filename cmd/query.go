package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strings"

	"github.com/bascanada/logviewer/pkg/log/client"
	"github.com/bascanada/logviewer/pkg/log/client/config"
	"github.com/bascanada/logviewer/pkg/log/factory"
	"github.com/bascanada/logviewer/pkg/log/impl/docker"
	"github.com/bascanada/logviewer/pkg/log/impl/elk/kibana"
	"github.com/bascanada/logviewer/pkg/log/impl/elk/opensearch"
	"github.com/bascanada/logviewer/pkg/log/impl/k8s"
	"github.com/bascanada/logviewer/pkg/log/impl/local"
	splunk "github.com/bascanada/logviewer/pkg/log/impl/splunk/logclient"
	"github.com/bascanada/logviewer/pkg/log/impl/ssh"
	"github.com/bascanada/logviewer/pkg/log/impl/cloudwatch"
	"github.com/bascanada/logviewer/pkg/log/printer"
	"github.com/bascanada/logviewer/pkg/ty"
	"github.com/bascanada/logviewer/pkg/views"

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
	if regex != "" {
		searchRequest.FieldExtraction.Regex.S(regex)
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

	if contextPath != "" {
		if len(contextIds) != 1 {
			return nil, errors.New("-i required only exactly one element when doing a query log or query tag")
		}
		config, err := config.LoadContextConfig(contextPath)
		if err != nil {
			return nil, err
		}

		clientFactory, err := factory.GetLogClientFactory(config.Clients)
		if err != nil {
			return nil, err
		}

		searchFactory, err := factory.GetLogSearchFactory(clientFactory, *config)
		if err != nil {
			return nil, err
		}

		sr, err := searchFactory.GetSearchResult(context.Background(), contextIds[0], inherits, searchRequest)

		return sr, err
	} else {
		if len(inherits) > 0 {
			return nil, errors.New("--inherits is only when using --config")
		}
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
			panic(err1)
		}
		searchResult.GetEntries(context.Background())
		fields, _, _ := searchResult.GetFields(context.Background())

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
			panic(err1)
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
		config, err := config.LoadContextConfig(contextPath)
		if err != nil {
			panic(err)
		}

		if err := views.RunQueryViewApp(*config, contextIds); err != nil {
			panic(err)
		}
	},
}
