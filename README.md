# logviewer

<p align="center">
  <img src="logo.svg" alt="logviewer logo" width="120" style="height:auto;" />

  <span>***this application is in early development , i'm still testing things out , if you like give me feedback***</span>
</p>

## Description

Logviewer is a cli log client for multiple source with an unified search feature and configuration.

The goal is to provide a one cli tool to read the logs from all the places they might be stored and output
them in a custom format that you can feed to another application.

Providing a unified way to build log queries that will work across all implementations or, when needed,
pass raw requests to the backend implementation.

Log source currently supported are:

* command (local or ssh)
* K8S
* Opensearch/Kibana
* Splunk
* AWS CloudWatch
* Docker

Way to use logviewer

* CLI like Curl
* MCP to integrate with your favorite LLM agent like copilot, gemini or claude.

## How to use (CLI)

![demo](demo.gif)

Example of the core functionality

```bash
# Query for log in the last 10 minutes
-> % logviewer [...] --last 10m --size 10 query log

# Query for the search field
-> % logviewer [...] --last 10m --size 10 query field

# Query with a filter on a field
-> % logviewer [...] --last 10m --size 10 -f level=INFO query log

# Query with a custom format , all fields can be used and more , it's go template
-> % logviewer [...] --last 10m --size 10 --format "{{.Fields.level}} - {{.Message}}" query log

# Use a config file and context instead of repeating the same field
-> % logviewer [...] -i cloudwatch-app-logs  --last 10m --size 10 --format "{{.Fields.level}} - {{.Message}}" query log

```

## How to install

You can check [the release folder](https://github.com/bascanada/logviewer/releases) for prebuild binary.
You can use the development build or the standard release.

Build manually

```bash
make release
make install PREFIX=$HOME/.local/bin
```

Use docker to run the application , for exemple with zsh function

```bash
logviewer() {
   docker run -it -v $HOME/.logviewer/config.json:/config.json -v $HOME/.ssh:/.ssh ghcr.io/bascanada/logviewer:latest "$@"
}
logviewer_update() {
   docker pull ghcr.io/bascanada/logviewer:latest
}
```

## Features

### Configuration

logviewer supports JSON and YAML config files and looks for the configuration in the following order:

1. Explicit path provided with the CLI flag `-c, --config /path/to/config` (highest precedence).
2. The environment variable `LOGVIEWER_CONFIG` if set (e.g. `export LOGVIEWER_CONFIG=/path/to/config.yaml`).
3. The default file at `$HOME/.logviewer/config.yaml` (only used when no `-c` and no env var are provided).

The configuration files is made of 3 parts:

1. Clients: definitions of log sources to fetch from
2. Searches: search definition the arguments to build the query
3. Contexts: client + []search + custom search

The context is what you use to execute a log search.

The log file support the following for dynamic value

1. Go template: `{{or .Size.Value 200}}` to access searches fields
2. Env variable: `${DOCKER_CID}`

```yaml
searches:
  pretty-print:
    fieldExtraction:
      groupRegex: '.*(?P<Level>INFO|WARN|ERROR|DEBUG).*'
      kvRegex: '(\w+)=(".*?"|[^\s,{}]+)'
    printerOptions:
      messageRegex: '^.*--- \\[[^]]+\\]\\s*(.*)$'
      template: |
        [{{.Timestamp.Format "15:04:05" }}] [{{.Level}}]
        {{.Message}}

contexts:
  local-test:
    client: local
    search:
      fields: {}
      options:
        cmd: 'tail -n {{or .Size.Value 200}} integration/logs/app.log'
```

```bash
-> % export LOGVIEWER_CONFIG=./config.yaml
-> % logviewer -i local-test query log
... display log without formatting
-> % logviewer -i local-test --inherits pretty-print query log
... inherit the search pretty-print to add templating and field extraction
```

#### Context Variables

You can define variables in your search contexts to make them more dynamic and reusable. Variables are defined in the `variables` section of a `search` block.

**Example:**

```yaml
contexts:
  user-session-logs:
    client: local-opensearch
    search:
      fields:
        sessionId: "${sessionId}"
      variables:
        sessionId:
          description: "The user session ID to filter logs for."
          type: string
          required: true
```

```bash
logviewer query -i user-session-logs --var "sessionId=abc-123" log
```

#### Multi-Context Search

LogViewer supports querying multiple log contexts simultaneously, allowing you to search across different log sources and merge the results into a unified, chronologically sorted view.

**Usage:**

```bash
# Query multiple contexts at once
logviewer -i context1 -i context2 -i context3 query log --last 10m --size 50

# Example: Search across dev, staging, and production environments
logviewer -i app-dev -i app-staging -i app-prod query log -f level=ERROR
```

**How it works:**

* **Single Context**: Executes directly with full feature support (field queries, pagination, filtering)
* **Multiple Contexts**: 
  * Queries are executed concurrently (fan-out pattern) for better performance
  * Results are merged and sorted by timestamp
  * Each log entry is tagged with its `ContextID` to identify the source
  * Errors from individual contexts don't block successful ones

**Current Limitations:**

* ❌ **Field discovery** (`query field`) is not supported with multiple contexts
* ❌ **Pagination** is not available — all results up to `--size` limit are returned at once
* ✅ **Field filtering** (`-f level=ERROR`) works and is validated per-context
* ✅ **Time ranges** and all other search parameters are supported

**Use Cases:**

* Compare logs across multiple environments (dev, staging, production)
* Search multiple related microservices simultaneously
* Correlate events across different log sources by timestamp

### Implementations

#### Text based implementation

All those source don't natively support all query building you have to do some manually
with regex to extract timestamp and fields if you want to directly filtering or templating.

* Docker
* K8s
* Command (local/ssh)

##### K8S

```bash
-> % logviewer --k8s-container frontend-dev-75fb7b89bb-9msbl --k8s-namespace growbe-prod  query log
```

```yaml
clients:
  local-k3s:
    type: k8s
    options:
      kubeConfig: integration/k8s/k3s.yaml
contexts:
  k3s-coredns:
    client: local-k3s
    searchInherit: []
    search:
      fields: {}
      options:
        namespace: kube-system
        pod: "${COREDNS_POD}"
        timestamp: true
```

##### Docker

Will used your `$DOCKER_HOST` if `--docker-host` is not provided , only required arguments is the name
of the container to query log for.

```bash
logviewer query log --docker-host "unix:///Users/William.Quintal/.colima/lol/docker.sock" --docker-container "growbe-portal" --refresh --last 42h
```

```yaml
clients:
  local-docker:
    type: docker
    options:
      host: unix:///var/run/docker.sock
contexts:
  docker-sample-container:
    client: local-docker
    searchInherit: []
    search:
      fields: {}
      options:
        container: "${DOCKER_CID}"
        showStdout: true
        showStderr: true
        timestamps: true
        details: false
```

##### Local/SSH

The `cmd` option for local and ssh clients now functions as a Go template, allowing you to inject search parameters directly into your command. The entire `LogSearch` object is available as the data context for the template, allowing access to fields like `{{.Size.Value}}`, `{{.Range.Last.Value}}`, etc.

This allows flags like `--size 50` or `--from "2023-10-27T10:00:00Z"` to dynamically alter the command.

Example:

```yaml
cmd: 'grep "{{.Range.Gte.Value}}" my-app.log | tail -n {{or .Size.Value 100}}'
```

By default no field are extracted but you can use a multiple regex to extract some field
from the log entry and use this as a filter (like using grep)

```bash
# Read a log file , if your command does not return to prompt like tail -f you need to put something
# as refresh-rate but it wont be use (need to fix)
-> % logviewer --cmd "tail -f ./logviewer.log" --format "{{.Message}}" --refresh-rate "1" query log
2023/05/14 21:07:07 [POST]http://kibana.elk.inner.wquintal.ca/internal/search/es
2023/05/18 16:50:37 [GET]https://opensearch.qc/logstash-*/_search 

-> % logviewer --cmd "tail ./logviewer.log" query field
# Nothing by default is return
-> % logviewer --cmd "cat ./logviewer.log" query field --fields-regex ".*\[(?P<httpmethod>GET|POST|DELETE)\].*"
httpmethod 
    POST
    GET
-> % logviewer --cmd "tail -f ./logviewer.log" --refresh-rate "1" query log --format "{{.Message}}" --fields-regex ".*\[(?P<httpmethod>GET|POST|DELETE)\].*" -f httpmethod=GET
2023/05/18 16:50:37 [GET]https://opensearch.qc/logstash-*/_search 
```

```bash
# SSH work in the same way you just need to add ssh flags
--ssh-addr string                SSH address and port, localhost:22
--ssh-identifiy string           SSH private key , by default $HOME/.ssh/id_rsa
--ssh-user string                SSH user
```

```yaml
clients:
  local-ssh:
    type: ssh
    options:
      user: testuser
      addr: 127.0.0.1:2222
      privateKey: integration/ssh/id_rsa
contexts:
  ssh-app-log:
    client: local-ssh
    searchInherit: []
    search:
      fields: {}
      options:
        cmd: "tail -n 200 app.log"
```

#### Opensearch

```bash
# Query max of 10 logs entry in the last 10 minute for an index in an instance
-> % logviewer --opensearch-endpoint "..." --elk-index "...*" --last 10m --size 10 query log
```

```yaml
clients:
  local-opensearch:
    type: opensearch
    options:
      endpoint: http://localhost:9200
contexts:
  opensearch-app-logs:
    client: local-opensearch
    searchInherit: []
    search:
      fields: {}
      options:
        index: app-logs
```

#### Splunk

```bash
# Query max of 10 logs entry in the last 10 minute for an index in an instance
-> % logviewer --splunk-url "..." --splunk-index "..." --last 10m --size 10 query log
```

```yaml
clients:
  local-splunk:
    type: splunk
    options:
      url: http://localhost:8088
contexts:
  splunk-app-logs:
    client: local-splunk
    searchInherit: []
    search:
      fields: {}
      options:
        index: app-logs
        fields:
          - field1
          - field2
```

##### Fields

In Splunk, some fields are not indexed by default and you need to use the `| fields + <field>` syntax to include them in your search.
You can use the `fields` option in your `search.options` to add a list of fields to be added to the search query.

#### AWS CloudWatch

You can query CloudWatch Logs either by providing flags on the CLI or by creating a client in `config.json`.

Example: query the last 30 minutes from a specific log group using Insights

```bash
-> % logviewer --cloudwatch-log-group "/aws/lambda/my-func" --cloudwatch-region "us-east-1" --last 30m --size 50 query log
[12:01:34][INFO] ...log message...
```

If you need to target a LocalStack or custom endpoint set `--cloudwatch-endpoint` and to force the use of FilterLogEvents (instead of Insights) use `--cloudwatch-use-insights=false`.

```yaml
clients:
  local-cloudwatch:
    type: cloudwatch
    options:
      region: us-east-1
      endpoint: http://localhost:4566
contexts:
  cloudwatch-app-logs:
    client: local-cloudwatch
    searchInherit: []
    search:
      fields: {}
      options:
        logGroupName: my-app-logs
        useInsights: false
```

### MCP Server

LogViewer can also be run as an MCP server, exposing its core functionalities as a tool for Large Language Models (LLMs) and other AI agents.
This enables programmatic access to log contexts, fields, and querying capabilities through natural language or structured commands.

It support the following type:

* stdio

#### Starting the MCP Server

To start the MCP server, use the `mcp` command and provide a path to your configuration file:

```bash
logviewer mcp --config /path/to/your/config.json
```

Exemple of a configuration in github copilot

```json
{
 "servers": {
  "logviewer-integration": {
   "type": "stdio",
   "command": "logviewer",
   "args": [
    "mcp",
    "--config",
    "./config.yaml"
   ]
  }
 },
 "inputs": []
}
```

```bash
2025-12-04 16:53:48.897 [info] Connection state: Starting
2025-12-04 16:53:48.897 [info] Connection state: Running
2025-12-04 16:53:48.930 [info] Discovered 4 tools
```

## License

This project is licensed under the GNU General Public License v3.0 - see the [LICENSE](LICENSE) file for details.
