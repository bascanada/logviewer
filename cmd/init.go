package cmd

import (
	httpPkg "github.com/bascanada/logviewer/pkg/http"
	"github.com/bascanada/logviewer/pkg/log"
	"github.com/bascanada/logviewer/pkg/log/impl/ssh"
	"github.com/spf13/cobra"
)

var (
	// kibana options
	endpointOpensearch string
	endpointKibana     string
	index              string

	// k8s options
	k8sNamespace string
	k8sPod       string
	k8sContainer string
	k8sPrevious  bool
	k8sTimestamp bool

	// splunk
	endpointSplunk string

	// docker
	dockerHost      string
	dockerContainer string

	// ssh options
	sshOptions ssh.SSHLogClientOptions
	cmd        string

	// cloudwatch
	cloudwatchLogGroup      string
	cloudwatchRegion        string
	cloudwatchProfile       string
	cloudwatchEndpoint      string
	cloudwatchUseInsights   bool
	cloudwatchPollInterval  string
	cloudwatchMaxPollInterval string
	cloudwatchPollBackoff   string

	// extra client fields
	headerField string
	bodyField   string

	// range
	from string
	to   string
	last string

	// fields
	fields    []string
	fieldsOps []string
	inherits  []string
	regex     string

	size int

	duration string
	refresh  bool

	template string

	contextPath string
	contextIds  []string

	logger log.MyLoggerOptions

	myLog bool
	debugHttp bool
)

func onCommandStart(cmd *cobra.Command, args []string) {
	log.ConfigureMyLogger(&logger)
	// enable HTTP debug logs when requested
	httpPkg.SetDebug(debugHttp)
}

func init() {

	// CONFIG

	queryCommand.PersistentFlags().StringVarP(&contextPath, "config", "c", "", "Config for preconfigure context for search")
	queryCommand.PersistentFlags().StringArrayVarP(&contextIds, "id", "i", []string{}, "Context id to execute")

	// IMPL SPECIFIQUE

	// ME
	queryCommand.PersistentFlags().BoolVar(&myLog, "mylog", false, "read from logviewer logs file")
	queryCommand.PersistentFlags().BoolVar(&debugHttp, "debug-http", false, "enable HTTP debug logs (prints request bodies and masked headers)")

	// K8S
	queryCommand.PersistentFlags().StringVar(&k8sNamespace, "k8s-namespace", "", "K8s namespace")
	queryCommand.PersistentFlags().StringVar(&k8sPod, "k8s-pod", "", "K8s pod")
	queryCommand.PersistentFlags().StringVar(&k8sContainer, "k8s-container", "", "K8s container")
	queryCommand.PersistentFlags().BoolVar(&k8sPrevious, "k8s-previous", false, "K8s log of previous container")
	queryCommand.PersistentFlags().BoolVar(&k8sTimestamp, "k8s-timestamp", false, "K8s include RFC3339 timestamp")
	// ELK
	queryCommand.PersistentFlags().StringVar(&endpointOpensearch, "opensearch-endpoint", "", "Opensearch endpoint")
	queryCommand.PersistentFlags().StringVar(&endpointKibana, "kibana-endpoint", "", "Kibana endpoint")
	queryCommand.PersistentFlags().StringVar(&index, "elk-index", "", "Elk index to search")
	// SPLUNK
	queryCommand.PersistentFlags().StringVar(&endpointSplunk, "splunk-endpoint", "", "Splunk endpoint")
	// DOCKER
	queryCommand.PersistentFlags().StringVar(&dockerHost, "docker-host", "", "Docker context")
	queryCommand.PersistentFlags().StringVar(&dockerContainer, "docker-container", "", "Docker container")

	// SSH
	queryCommand.PersistentFlags().StringVar(&sshOptions.Addr, "ssh-addr", "", "SSH address and port localhost:22")
	queryCommand.PersistentFlags().StringVar(&sshOptions.User, "ssh-user", "", "SSH user")
	queryCommand.PersistentFlags().StringVar(&sshOptions.PrivateKey, "ssh-identifiy", "", "SSH private key , by default $HOME/.ssh/id_rsa")

	// CLOUDWATCH
	queryCommand.PersistentFlags().StringVar(&cloudwatchLogGroup, "cloudwatch-log-group", "", "CloudWatch Logs log group name")
	queryCommand.PersistentFlags().StringVar(&cloudwatchRegion, "cloudwatch-region", "", "AWS region for CloudWatch Logs (overrides SDK default)")
	queryCommand.PersistentFlags().StringVar(&cloudwatchProfile, "cloudwatch-profile", "", "AWS shared config profile to use for CloudWatch Logs")
	queryCommand.PersistentFlags().StringVar(&cloudwatchEndpoint, "cloudwatch-endpoint", "", "Custom endpoint for CloudWatch Logs (useful for LocalStack)")
	queryCommand.PersistentFlags().BoolVar(&cloudwatchUseInsights, "cloudwatch-use-insights", true, "Use CloudWatch Logs Insights (set to false to fallback to FilterLogEvents)")

	// CloudWatch polling tuning (affects Insights async polling)
	queryCommand.PersistentFlags().StringVar(&cloudwatchPollInterval, "cloudwatch-poll-interval", "", "Base poll interval (e.g. 1s) for CloudWatch Insights polling")
	queryCommand.PersistentFlags().StringVar(&cloudwatchMaxPollInterval, "cloudwatch-max-poll-interval", "", "Max poll interval (e.g. 30s) for CloudWatch Insights polling")
	queryCommand.PersistentFlags().StringVar(&cloudwatchPollBackoff, "cloudwatch-poll-backoff", "", "Backoff factor (e.g. 2s) for CloudWatch Insights polling")

	// ADDITIONAL CLIENT INFO
	queryCommand.PersistentFlags().StringVar(&headerField, "client-headers", "", "File containings list of headers to be used by the underlying client")
	queryCommand.PersistentFlags().StringVar(&bodyField, "client-body", "", "File containing base body to be used by the underlying client")

	// COMMAND
	queryCommand.PersistentFlags().StringVar(&cmd, "cmd", "", "If using ssh or local , manual command to run")

	// RANGE
	queryCommand.PersistentFlags().StringVar(&from, "from", "", "Get entry gte datetime date >= from")
	queryCommand.PersistentFlags().StringVar(&to, "to", "", "Get entry lte datetime date <= to")
	queryCommand.PersistentFlags().StringVar(&last, "last", "", "Get entry in the last duration")

	// SIZE
	queryCommand.PersistentFlags().IntVar(&size, "size", 0, "Get entry max size")

	// FIELD validation
	queryCommand.PersistentFlags().StringArrayVarP(&fields, "fields", "f", []string{}, "Field for selection field=value")
	queryCommand.PersistentFlags().StringArrayVar(
		&fieldsOps, "fields-condition", []string{}, "Field Ops for selection field=value (match, exists, wildcard, regex)",
	)
	queryCommand.PersistentFlags().StringVar(
		&regex, "fields-regex", "",
		"Regex to extract field from log text, using named group \".*(?P<Level>INFO|WARN|ERROR).*\"")

	// LIVE DATA OPTIONS
	queryLogCommand.PersistentFlags().StringVar(
		&duration, "refresh-rate", "", "If provide refresh log at the rate provide (ex: 30s)")
	queryLogCommand.PersistentFlags().BoolVar(&refresh, "refresh", false, "If provide activate live data")

	// OUTPUT FORMATTING
	queryLogCommand.PersistentFlags().StringVar(
		&template,
		"format",
		"[{{.Timestamp.Format \"15:04:05\" }}][{{.Level}}] {{.Message}}", "Format for the log entry")

	queryCommand.PersistentFlags().StringArrayVar(&inherits, "inherits", []string{}, "When using config , list of inherits to execute on top of the one configure for the search")

	queryCommand.AddCommand(queryLogCommand)
	queryCommand.AddCommand(queryFieldCommand)

}
