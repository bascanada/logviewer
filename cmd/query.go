package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strings"

	"github.com/bascanada/logviewer/pkg/log/client"
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
				if prev, ok := (*maps)(""); ok && prev != "" {
					(*maps)("") = prev + " " + val
				} else {
					(*maps)("") = val
				}
			} else {
				(*maps)[key] = val
			}
			continue
		}

		// No '=' present: treat the whole string as a free-text token and
		// append it to any existing free-text tokens.
		if prev, ok := (*maps)(""); ok && prev != "" {
			(*maps)("") = prev + " " + f
		} else {
			(*maps)("") = f
		}
	}
	return nil
}

func resolveSearch(cmd *cobra.Command) (client.LogSearchResult, error) {
	var err error

	// resolve this from args
	searchRequest := client.LogSearch{
		Fields:          ty.MS{},
		FieldsCondition: ty.MS{},
		Options:         ty.MI{},
	}
	if size > 0 {
		searchRequest.Size.S(size)
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

	if cmd.Use != "local" {
		searchRequest.Options[local.OptionsCmd] = cmd.Use
	}

	if template != "" {
		searchRequest.PrinterOptions.Template.S(template)
	}

	searchRequest.Refresh.Follow.S(refresh)

	if len(contextIds) > 0 {
		if len(contextIds) != 1 {
			return nil, errors.New("-i required only exactly one element when doing a query log or query tag")
		}
		config, _, err := loadConfig(cmd)
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

		return searchFactory.GetSearchResult(context.Background(), contextIds[0], inherits, searchRequest)
	}

	var system string

	if endpointOpensearch != "" {
		system = "opensearch"
	} else if endpointKibana != "" {
		system = "kibana"
	} else if cloudwatchLogGroup != "" {
		system = "cloudwatch"
	} else if k8sNamespace != "" {
		system = "k8s"
	} else if cmd.Use != "" {
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
		searchRequest.Options["Container"] = dockerContainer
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

	searchResult, err := logClient.Get(context.Background(), &searchRequest)
	if err != nil {
		return nil, err
	}

	return searchResult, nil
}

var queryFieldCommand = &cobra.Command{
	Use:    "field",
	Short:  "Dispaly available field for filtering of logs",
	PreRun: onCommandStart,
	RunE: func(cmd *cobra.Command, args []string) error {
		searchResult, err := resolveSearch(cmd)
		if err != nil {
			return err
		}
		searchResult.GetEntries(context.Background())
		fields, _, _ := searchResult.GetFields(context.Background())

		for k, b := range fields {
			fmt.Printf("%s \n", k)
			for _, r := range b {
				fmt.Println("    " + r)
			}
		}
		return nil
	},
}

var queryLogCommand = &cobra.Command{
	Use:    "log",
	Short:  "Display logs for system",
	PreRun: onCommandStart,
	RunE: func(cmd *cobra.Command, args []string) error {
		searchResult, err := resolveSearch(cmd)
		if err != nil {
			return err
		}
		outputter := printer.PrintPrinter{}
		continous, err := outputter.Display(context.Background(), searchResult)
		if err != nil {
			return err
		}
		if continous {
			c := make(chan os.Signal, 1)
			signal.Notify(c, os.Interrupt)
			<-c
		}
		return nil
	},
}

var queryCommand = &cobra.Command{
	Use:    "query",
	Short:  "Query a login system for logs and available fields",
	PreRun: onCommandStart,
	RunE: func(cmd *cobra.Command, args []string) error {
		config, path, err := loadConfig(cmd)
		if err != nil {
			return err
		}

		fmt.Fprintf(os.Stderr, "Using config file: %s\n", path)

		if err := views.RunQueryViewApp(*config, contextIds); err != nil {
			return err
		}
		return nil
	},
}

var (
	contextIds []string
	inherits   bool
)

var (
	fields     []string
	fieldsOps  []string
	size       int
	refresh    bool
	duration   string
	regex      string
	from       string
	to         string
	last       string
	template   string
	index      string
	headerField string
	bodyField   string

	// docker
	dockerHost      string
	dockerContainer string

	// splunk
	endpointSplunk string

	// kibana
	endpointKibana string

	// opensearch
	endpointOpensearch string

	// cloudwatch
	cloudwatchLogGroup        string
	cloudwatchRegion          string
	cloudwatchProfile         string
	cloudwatchEndpoint        string
	cloudwatchUseInsights     bool
	cloudwatchPollInterval    string
	cloudwatchMaxPollInterval string
	cloudwatchPollBackoff     string

	// k8s
	k8sContainer string
	k8sNamespace string
	k8sPod       string
	k8sPrevious  bool
	k8sTimestamp bool

	// local
	cmd string

	// ssh
	sshOptions ssh.SshLogClientOptions
)

func onCommandStart(cmd *cobra.Command, args []string) {
	if len(args) > 0 {
		contextIds = args
	}
}

func init() {
	queryLogCommand.Flags().StringArrayVarP(&fields, "field", "f", []string{}, "field to display")
	queryLogCommand.Flags().StringArrayVarP(&fieldsOps, "field-op", "p", []string{}, "field operation for filtering")
	queryLogCommand.Flags().IntVarP(&size, "size", "s", 200, "size of the log to received")
	queryLogCommand.Flags().BoolVarP(&refresh, "refresh", "r", false, "continuously refresh the query")
	queryLogCommand.Flags().StringVar(&duration, "duration", "", "duration of the refresh")
	queryLogCommand.Flags().StringVar(&regex, "regex", "", "regex to extract field from the message")
	queryLogCommand.Flags().StringVar(&from, "from", "", "from date for the query")
	queryLogCommand.Flags().StringVar(&to, "to", "", "to date for the query")
	queryLogCommand.Flags().StringVar(&last, "last", "15m", "last duration for the query")
	queryLogCommand.Flags().StringVarP(&template, "template", "t", "", "template to use for the output")
	query_add_docker_flags(queryLogCommand)
	query_add_splunk_flags(queryLogCommand)
	query_add_kibana_flags(queryLogCommand)
	query_add_opensearch_flags(queryLogCommand)
	query_add_cloudwatch_flags(queryLogCommand)
	query_add_k8s_flags(queryLogCommand)
	query_add_local_flags(queryLogCommand)
	query_add_ssh_flags(queryLogCommand)

	queryFieldCommand.Flags().StringArrayVarP(&contextIds, "id", "i", []string{}, "id of the context to use")
	queryFieldCommand.Flags().BoolVar(&inherits, "inherits", false, "inherits the context")
	query_add_docker_flags(queryFieldCommand)
	query_add_splunk_flags(queryFieldCommand)
	query_add_kibana_flags(queryFieldCommand)
	query_add_opensearch_flags(queryFieldCommand)
	query_add_cloudwatch_flags(queryFieldCommand)
	query_add_k8s_flags(queryFieldCommand)
	query_add_local_flags(queryFieldCommand)
	query_add_ssh_flags(queryFieldCommand)

	queryLogCommand.Flags().StringArrayVarP(&contextIds, "id", "i", []string{}, "id of the context to use")
	queryLogCommand.Flags().BoolVar(&inherits, "inherits", false, "inherits the context")

	queryCommand.Flags().StringArrayVarP(&contextIds, "id", "i", []string{}, "id of the context to use")

	rootCmd.AddCommand(queryCommand)
	queryCommand.AddCommand(queryLogCommand)
	queryCommand.AddCommand(queryFieldCommand)
}

func query_add_docker_flags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&dockerHost, "docker-host", "", "docker host")
	cmd.Flags().StringVar(&dockerContainer, "docker-container", "", "docker container")
}

func query_add_splunk_flags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&endpointSplunk, "splunk-endpoint", "", "splunk endpoint")
	cmd.Flags().StringVar(&headerField, "header", "", "header to add to the request")
	cmd.Flags().StringVar(&bodyField, "body", "", "body to add to the request")
}

func query_add_kibana_flags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&endpointKibana, "kibana-endpoint", "", "kibana endpoint")
	cmd.Flags().StringVar(&index, "index", "", "index to use for the query")
}

func query_add_opensearch_flags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&endpointOpensearch, "opensearch-endpoint", "", "opensearch endpoint")
}

func query_add_cloudwatch_flags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&cloudwatchLogGroup, "cloudwatch-log-group", "", "cloudwatch log group")
	cmd.Flags().StringVar(&cloudwatchRegion, "cloudwatch-region", "", "cloudwatch region")
	cmd.Flags().StringVar(&cloudwatchProfile, "cloudwatch-profile", "", "cloudwatch profile")
	cmd.Flags().StringVar(&cloudwatchEndpoint, "cloudwatch-endpoint", "", "cloudwatch endpoint")
	cmd.Flags().BoolVar(&cloudwatchUseInsights, "cloudwatch-use-insights", true, "cloudwatch use insights")
	cmd.Flags().StringVar(&cloudwatchPollInterval, "cloudwatch-poll-interval", "", "cloudwatch poll interval")
	cmd.Flags().StringVar(&cloudwatchMaxPollInterval, "cloudwatch-max-poll-interval", "", "cloudwatch max poll interval")
	cmd.Flags().StringVar(&cloudwatchPollBackoff, "cloudwatch-poll-backoff", "", "cloudwatch poll backoff")
}

func query_add_k8s_flags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&k8sContainer, "k8s-container", "", "k8s container")
	cmd.Flags().StringVar(&k8sNamespace, "k8s-namespace", "", "k8s namespace")
	cmd.Flags().StringVar(&k8sPod, "k8s-pod", "", "k8s pod")
	cmd.Flags().BoolVar(&k8sPrevious, "k8s-previous", false, "k8s previous")
	cmd.Flags().BoolVar(&k8sTimestamp, "k8s-timestamp", false, "k8s timestamp")
}

func query_add_local_flags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&cmd, "cmd", "", "command to execute")
}

func query_add_ssh_flags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&sshOptions.Addr, "ssh-addr", "", "ssh address")
	cmd.Flags().StringVar(&sshOptions.User, "ssh-user", "", "ssh user")
	cmd.Flags().StringVar(&sshOptions.Password, "ssh-password", "", "ssh password")
	cmd.Flags().StringVar(&sshOptions.Key, "ssh-key", "", "ssh key")
}