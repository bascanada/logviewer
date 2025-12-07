# logviewer

<p align="center">
  <img src="https://raw.githubusercontent.com/bascanada/logviewer/main/logo.svg" alt="logviewer logo" width="120" style="height:auto;" />

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

## Why LogViewer?

**The Problem:** Modern applications spread logs across multiple systems—Kubernetes pods, CloudWatch, Splunk, Docker containers, SSH servers. Each system has its own CLI tool, query syntax, and authentication method. Debugging issues that span multiple services means juggling multiple terminals and commands.

**The Solution:** LogViewer provides a single, unified CLI to query all your log sources with consistent syntax:

* **One tool, multiple sources** - Query Kubernetes, CloudWatch, Splunk, OpenSearch, Docker, and local files with the same commands
* **Unified query syntax** - Learn once, use everywhere. No need to remember SPL, KQL, or different CLIs
* **Powerful field extraction** - Extract structured data from unstructured logs using regex patterns
* **Flexible formatting** - Custom templates let you format output for humans or pipe to other tools
* **Config-driven** - Save common queries as reusable contexts instead of maintaining complex shell scripts
* **Multi-context search** - Query multiple environments (dev/staging/prod) simultaneously and get merged, time-sorted results
* **AI integration** - Use as an MCP server with Claude, Copilot, or Gemini for natural language log queries

**Perfect for:**
* DevOps engineers managing logs across multiple platforms
* SREs debugging distributed systems
* Developers who need quick access to logs without learning platform-specific tools
* Teams wanting to standardize log access across their infrastructure

## How to use (CLI)

![demo](https://raw.githubusercontent.com/bascanada/logviewer/main/demo.gif)

LogViewer works with or without a config file. Start simple and add complexity as needed.

### 1. Basic Usage - No Config Required

Query local files directly using the `--cmd` flag:

```bash
# Read last 100 lines from a log file
logviewer --cmd "tail -n 100 /var/log/app.log" query log

# Read from Docker container
logviewer --docker-container my-app --docker-host "unix:///var/run/docker.sock" query log

# Read from SSH server
logviewer --ssh-user admin --ssh-addr "server.com:22" --cmd "tail -n 100 /var/log/app.log" query log
```

### 2. Add Time Filtering

Narrow down results to a specific time window:

```bash
# Last 10 minutes
logviewer --cmd "tail -n 1000 /var/log/app.log" --last 10m query log

# Last hour, limit to 50 entries
logviewer --cmd "tail -n 5000 /var/log/app.log" --last 1h --size 50 query log

# Specific time range
logviewer --cmd "cat /var/log/app.log" --from "2025-12-05T10:00:00Z" --to "2025-12-05T11:00:00Z" query log
```

### 3. Custom Output Formatting

Format log output using Go templates:

```bash
# Simple format - just the message
logviewer --cmd "tail /var/log/app.log" --format "{{.Message}}" query log

# Add timestamp
logviewer --cmd "tail /var/log/app.log" --format "{{.Timestamp.Format \"15:04:05\"}} {{.Message}}" query log

# Multi-line format with fields
logviewer --cmd "tail /var/log/app.log" --format "[{{.Timestamp.Format \"15:04:05\"}}] {{.Level}}\n{{.Message}}" query log
```

### 4. Create a Config File

Save common queries in `~/.logviewer/config.yaml`:

```yaml
clients:
  local:
    type: local

contexts:
  app-logs:
    client: local
    search:
      options:
        cmd: 'tail -n {{or .Size.Value 200}} /var/log/app.log'
```

Now use contexts instead of repeating flags:

```bash
# Query using context
logviewer -i app-logs query log

# Override size from config
logviewer -i app-logs --size 50 query log

# Add time range
logviewer -i app-logs --last 1h query log
```

### 5. Extract and Filter by Fields

Add field extraction to make logs searchable (see [Field Extraction](#field-extraction) for details):

```yaml
searches:
  spring-boot-logs:
    fieldExtraction:
      timestampRegex: '^(\d{4}-\d{2}-\d{2}\s+\d{2}:\d{2}:\d{2}(\.\d+)?)'
      groupRegex: '.*(?P<level>INFO|WARN|ERROR|DEBUG).*---\s*\[(?P<thread>[^\]]+)\]'
      kvRegex: '([A-Z]{2,}):\s([^\s,]+)'

contexts:
  app-logs:
    client: local
    searchInherit: ["spring-boot-logs"]
    search:
      options:
        cmd: 'tail -n {{or .Size.Value 500}} /var/log/app.log'
```

Now filter by extracted fields:

```bash
# Filter by log level
logviewer -i app-logs -f level=ERROR query log

# Filter by multiple fields
logviewer -i app-logs -f level=ERROR -f thread=nio-8080-exec-1 query log

# Discover available fields
logviewer -i app-logs query field
```

### 6. Add Custom Templates

Combine field extraction with templates for formatted output:

```yaml
searches:
  spring-boot-logs:
    fieldExtraction:
      groupRegex: '.*(?P<level>INFO|WARN|ERROR|DEBUG).*---\s*\[(?P<thread>[^\]]+)\]'
    printerOptions:
      template: '[{{.Timestamp.Format "15:04:05"}}] [{{.Field "level"}}] [{{.Field "thread"}}] {{.Message}}'
```

```bash
logviewer -i app-logs query log
# Output: [10:30:45] [ERROR] [nio-8080-exec-7] Payment failed for order ID: 12345
```

### 7. Query Multiple Sources

Search across multiple environments simultaneously:

```yaml
contexts:
  app-prod:
    client: k8s-prod
    search:
      options:
        namespace: production
        pod: app-*
  
  app-staging:
    client: k8s-staging
    search:
      options:
        namespace: staging
        pod: app-*
```

```bash
# Query both contexts, results merged and sorted by timestamp
logviewer -i app-prod -i app-staging --last 10m query log
```

### Common Patterns

```bash
# Real-time log tailing (refreshes every 2 seconds)
logviewer -i app-logs --refresh 2s query log

# Save logs to file
logviewer -i app-logs --last 1h query log > logs.txt

# Pipe to other tools
logviewer -i app-logs query log | grep "error" | wc -l

# Use with environment variables for dynamic values
export LOG_FILE="/var/log/app-$(date +%Y%m%d).log"
logviewer --cmd "tail -n 100 $LOG_FILE" query log
```

## Installation

### Quick Install (Linux & macOS)

Run the following command to download the latest release and install it to `/usr/local/bin`:

```bash
curl -L "https://github.com/bascanada/logviewer/releases/latest/download/logviewer-$(uname -s | tr '[:upper:]' '[:lower:]')-$(uname -m | sed 's/x86_64/amd64/')" -o ./logviewer && chmod +x ./logviewer
sudo mv ./logviewer /usr/local/bin/
```

### Docker

Run LogViewer without installing it on your host system. Add this function to your shell configuration (`.zshrc` or `.bashrc`):

```bash
# Add to ~/.zshrc or ~/.bashrc
logviewer() {
   docker run -it --rm \
     -v $HOME/.logviewer/config.yaml:/config.yaml \
     -v $HOME/.ssh:/.ssh \
     -v /var/run/docker.sock:/var/run/docker.sock \
     ghcr.io/bascanada/logviewer:latest "$@"
}
```

### Build from Source

```bash
git clone https://github.com/bascanada/logviewer.git
cd logviewer
make install PREFIX=$HOME/.local/bin
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

#### Field Extraction

LogViewer can extract structured fields from unstructured log lines using regex patterns. This is particularly useful for text-based log sources (local files, Docker, Kubernetes, SSH).

**Why Field Extraction?**

Turn this:
```
2025-08-22 10:01:19.200  ERROR 18748 --- [nio-8080-exec-7] com.example.service.PaymentService : Payment failed for order ID: 12345
```

Into searchable fields: `level=ERROR`, `thread=nio-8080-exec-7`, `class=com.example.service.PaymentService`, `ID=12345`

**Configuration:**

```yaml
searches:
  spring-boot-logs:
    fieldExtraction:
      # Extract timestamp at the beginning of the line
      timestampRegex: '^(\d{4}-\d{2}-\d{2}\s+\d{2}:\d{2}:\d{2}(\.\d+)?)'
      
      # Extract named groups (Level, Thread, Class)
      groupRegex: '.*(?P<level>INFO|WARN|ERROR|DEBUG).*---\s*\[(?P<thread>[^\]]+)\]\s+(?P<class>[\w.$-]+)\s+:'
      
      # Extract key-value pairs (ID: 12345, IP: 192.168.1.1)
      kvRegex: '([A-Z]{2,}):\s([^\s,]+)'
```

**Regex Types:**

1. **`timestampRegex`**: Extracts and parses timestamps from the beginning of each log line
   - Supports multiple formats (RFC3339, space-separated dates)
   - Removed from message text after extraction

2. **`groupRegex`**: Extracts named fields using `(?P<fieldname>...)` syntax
   - Use for structured patterns that appear in consistent positions
   - Field names can be lowercase (automatically handled)
   - Example: `(?P<level>INFO|WARN|ERROR)` creates a `level` field

3. **`kvRegex`**: Extracts key-value pairs scattered throughout the message
   - Matches patterns like `key=value` or `KEY: value`
   - Useful for extracting IDs, IPs, counts, etc.
   - Example: `([A-Z]{2,}):\s([^\s,]+)` matches `ID: 12345`

4. **`json`**: Enables native JSON parsing
   - Set `json: true` to enable
   - Automatically flattens JSON objects into fields
   - Configurable keys for standard fields:
     - `jsonMessageKey`: Key for the main message (default: "message")
     - `jsonLevelKey`: Key for the log level (default: "level")
     - `jsonTimestampKey`: Key for the timestamp (default: "timestamp")
   - Supports numeric timestamps (Unix epoch) and string formats

**Using Extracted Fields:**

```bash
# Filter by extracted fields
logviewer -i app-logs -f level=ERROR query log

# Access in templates
logviewer -i app-logs --format "{{.Field \"level\"}} {{.Field \"thread\"}}: {{.Message}}" query log

# Discover available fields
logviewer -i app-logs query field
```

**Tips:**

* Start simple: extract one field at a time and test with `query field`
* Use online regex testers (regex101.com) with sample log lines
* Field values are automatically trimmed of whitespace
* Both uppercase and lowercase field names work in templates
* Use `(?P<name>...)` for named groups that become searchable fields

#### Template Functions

LogViewer templates use Go's `text/template` engine with additional helper functions:

**Built-in Fields:**

* `{{.Timestamp}}` - Log entry timestamp (time.Time)
* `{{.Message}}` - Log message text
* `{{.Level}}` - Log level (if extracted)
* `{{.ContextID}}` - Source context (in multi-context queries)

**Helper Functions:**

* **`{{.Timestamp.Format "layout"}}`** - Format timestamp
  ```yaml
  # Common layouts
  {{.Timestamp.Format "15:04:05"}}           # 10:30:45
  {{.Timestamp.Format "2006-01-02 15:04:05"}} # 2025-12-05 10:30:45
  {{.Timestamp.Format "Jan _2 15:04:05"}}     # Dec  5 10:30:45
  ```

* **`{{.Field "name"}}`** - Access fields case-insensitively
  ```yaml
  {{.Field "level"}}   # Access level field (lowercase or uppercase)
  {{.Field "thread"}}  # Access thread field
  {{.Field "ID"}}      # Access extracted ID
  ```

* **`{{MultiLine .Fields}}`** - Format all fields as multi-line list
  ```yaml
  Message: {{.Message}}{{MultiLine .Fields}}
  # Output:
  # Message: Payment failed
  #  * level=ERROR
  #  * ID=12345
  ```

* **`{{KV .Fields}}`** - Format fields as key=value pairs
  ```yaml
  [{{.Level}}] {{KV .Fields}} {{.Message}}
  # Output: [INFO] logger=http duration=5ms Request processed
  ```

* **`{{ExpandJson .Message}}`** - Pretty-print JSON in message
  ```yaml
  {{ExpandJson .Message}}
  # Finds JSON objects and formats them with color and indentation
  ```

* **`{{.Fields}}`** - Map of all extracted fields (useful for debugging)

**Template Examples:**

```yaml
# Compact one-liner
template: '[{{.Timestamp.Format "15:04:05"}}] {{.Field "level"}} {{.Message}}'

# Detailed multi-line
template: |
  [{{.Timestamp.Format "2006-01-02 15:04:05"}}] [{{.Field "level"}}] [{{.ContextID}}]
  Thread: {{.Field "thread"}}
  Class: {{.Field "class"}}
  {{.Message}}

# JSON extraction
template: |
  {{.Timestamp.Format "15:04:05"}} {{.Field "level"}}
  {{.Message}}
  {{ExpandJson .Message}}
```

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

Supports connecting to local and remote Docker hosts. Uses your `$DOCKER_HOST` if `--docker-host` is not provided. Remote hosts are supported via SSH using Docker's connhelper (e.g., `ssh://user@host:port`).

For Docker Compose environments, you can specify a `service` name instead of a `container` ID/name. This dynamically resolves the container using Docker Compose labels. Optionally, restrict to a specific `project` to avoid conflicts when multiple Compose projects are running.

```bash
# Local Docker
logviewer query log --docker-host "unix:///var/run/docker.sock" --docker-container "my-app"

# Remote Docker via SSH
logviewer query log --docker-host "ssh://user@remote-host:22" --docker-container "my-app"

# Docker Compose service discovery
logviewer query log --docker-host "unix:///var/run/docker.sock" --docker-service "api" --docker-project "my-project"
```

```yaml
clients:
  local-docker:
    type: docker
    options:
      host: unix:///var/run/docker.sock
  remote-docker:
    type: docker
    options:
      host: ssh://user@remote-host:22
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
  docker-compose-service:
    client: local-docker
    search:
      options:
        service: "api"
        project: "my-project"  # Optional
        showStdout: true
        showStderr: true
        timestamps: true
```

**Docker Options:**

* `host`: Docker daemon socket or URL (supports SSH via connhelper)
* `container`: Container ID or name (for direct specification)
* `service`: Docker Compose service name (for dynamic resolution)
* `project`: Docker Compose project name (optional, for service filtering)
* `showStdout`, `showStderr`, `timestamps`, `details`: Log output options

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

### MCP Server (AI Integration)

Turn your favorite LLM into an expert DevOps assistant. LogViewer implements the **Model Context Protocol (MCP)**, allowing AI agents (like Claude Desktop, GitHub Copilot, or custom agents) to directly query and analyze your logs.

**Why use LogViewer with AI?**

* **Autonomous Investigation**: The AI can explore logs, refine searches, and drill down into errors without your constant input.
* **Context Aware**: It understands your environments (dev, staging, prod) through your configuration.
* **Smart Filtering**: It discovers available fields and applies precise filters to cut through the noise.
* **Root Cause Analysis**: Ask "Why did the payment fail?" and watch it query multiple services, correlate timestamps, and present the evidence.

#### Available Tools

The MCP server exposes the following tools to the AI:

* **`list_contexts`**: Discovers all available log sources (e.g., `k8s-prod`, `aws-lambda`, `local-dev`).
* **`query_logs`**: The core tool. Fetches logs with powerful filtering:
  * Time windows (`last=15m`, `last=24h`)
  * Field filtering (`fields={"level": "ERROR", "service": "payment"}`)
  * Pagination and sizing
* **`get_fields`**: Introspects logs to find searchable fields (e.g., "Is there a `requestId` field I can filter on?").
* **`get_context_details`**: Inspects configuration and required variables for specific contexts.

#### Built-in Prompts

* **`log_investigation`**: A guided workflow that instructs the AI on the best strategy to investigate an incident (Query -> Analyze -> Refine).

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

## Troubleshooting

### Common Issues

**"Config file not found"**
```bash
# Check if config exists
ls ~/.logviewer/config.yaml

# Specify explicit path
logviewer -c /path/to/config.yaml -i context query log

# Or use environment variable
export LOGVIEWER_CONFIG=/path/to/config.yaml
```

**"Context not found"**
```bash
# List available contexts in your config
grep -A1 "contexts:" config.yaml

# Check context name matches exactly (case-sensitive)
logviewer -i exact-context-name query log
```

**"No logs returned"**

* Check time range: `--last 24h` instead of `--last 10m`
* Verify client connectivity (credentials, network, endpoints)
* Test without filters first: remove `-f` flags
* For field filters, ensure fields exist: `logviewer -i context query field`

**"Timestamp shows 00:00:00"**

* Your `timestampRegex` is extracting but not parsing correctly
* Check timestamp format in logs matches the regex
* LogViewer supports: RFC3339, `YYYY-MM-DD HH:MM:SS.mmm`, and `YYYY-MM-DD HH:MM:SS`

**"Field extraction not working"**

* Test regex on sample log lines using regex101.com
* Verify named groups use `(?P<name>...)` syntax (not `(?<name>...)`)
* Check that field names are alphanumeric
* Run `query field` to see what's actually extracted

**"Template error"**

* Ensure fields exist before accessing: `{{.Field "name"}}`
* Use `.Field` method for case-insensitive access
* Check for typos in field names and template syntax

### Debug Tips

**Enable verbose logging:**
```bash
# Set log level (not implemented yet - coming soon)
# For now, check stderr for error messages
logviewer -i context query log 2>errors.log
```

**Test queries incrementally:**
```bash
# 1. Start simple
logviewer -i context query log --size 5

# 2. Add time range
logviewer -i context --last 1h query log --size 5

# 3. Add filters
logviewer -i context --last 1h -f level=ERROR query log --size 5

# 4. Add formatting
logviewer -i context --last 1h -f level=ERROR --format "{{.Message}}" query log
```

**Verify regex patterns:**
```bash
# Use a small sample first
logviewer --cmd "head -n 10 /var/log/app.log" query field

# Then test filters
logviewer --cmd "head -n 50 /var/log/app.log" -f level=ERROR query log
```

## FAQ

**Q: Can LogViewer tail logs in real-time?**

A: Yes! Use the `--refresh` flag with a duration:
```bash
logviewer -i context --refresh 2s query log
```

**Q: How do I authenticate to Splunk/CloudWatch/Kubernetes?**

A: Authentication varies by source:
* **Kubernetes**: Uses kubeconfig file (same as kubectl)
* **CloudWatch**: Uses AWS credentials (AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY, or IAM role)
* **Splunk**: Pass auth token in headers via config
* **OpenSearch**: Basic auth or API keys in endpoint URL
* **SSH**: Private key file (default: `~/.ssh/id_rsa`)

**Q: Can I pipe LogViewer output to other tools?**

A: Absolutely! LogViewer outputs to stdout:
```bash
# Pipe to grep
logviewer -i context query log | grep "error"

# Pipe to jq (if using JSON format)
logviewer -i context --format "{{json .}}" query log | jq '.message'

# Save to file
logviewer -i context query log > logs.txt
```

**Q: How do I query logs from multiple Kubernetes namespaces?**

A: Create multiple contexts:
```yaml
contexts:
  k8s-namespace1:
    client: k8s-prod
    search:
      options:
        namespace: namespace1
  k8s-namespace2:
    client: k8s-prod
    search:
      options:
        namespace: namespace2
```

Then use multi-context search:
```bash
logviewer -i k8s-namespace1 -i k8s-namespace2 query log
```

**Q: What's the difference between `searchInherit` and inline config?**

A: `searchInherit` lets you reuse common settings:
```yaml
searches:
  common-formatting:
    printerOptions:
      template: "{{.Timestamp.Format \"15:04:05\"}} {{.Message}}"

contexts:
  app-logs:
    searchInherit: ["common-formatting"]  # Reuse template
    search:
      options:
        cmd: "tail app.log"
```

Inline config is specific to one context. Use `searchInherit` for shared settings across multiple contexts.

**Q: Does LogViewer store or send my logs anywhere?**

A: No. LogViewer is a CLI tool that queries your log sources directly and displays results locally. Nothing is stored or transmitted except to your configured log sources.

**Q: Can I contribute or request features?**

A: Yes! This project is open source. Open an issue or PR on GitHub.

## License

This project is licensed under the GNU General Public License v3.0 - see the [LICENSE](LICENSE) file for details.
