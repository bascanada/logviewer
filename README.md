
# logviewer

Terminal based log viewer for multiple log source with search feature and view configuration.

***this application is in early development , i'm still testing things out , if you like give me feedback***

Log source at the moment are:

* Command local or by ssh
* Kubectl logs
* Opensearch/Kibana logs
* Splunk logs
* AWS CloudWatch
* Docker

## How to install

You can check [the release folder](https://github.com/bascanada/logviewer/releases) for prebuild binary.
You can use the development build or the standard release.

Other option is to use docker to run the application

```bash
logviewer() {
   docker run -it -v $HOME/.logviewer/config.json:/config.json -v $HOME/.ssh:/.ssh ghcr.io/bascanada/logviewer:latest "$@"
}
logviewer_update() {
   docker pull ghcr.io/bascanada/logviewer:latest
}
```

## How to use

There is main way to access the log

* Via the stdout , outputting directly in the terminal like curl
* With the TUI , creating tmux like views for multiple log query like k9s
* Http server , query remotely for the log configure in your context
* MCP integrate , allow llm to query and analyze your log


## Implementations


### Opensearch

```bash
# Query max of 10 logs entry in the last 10 minute for an index in an instance
-> % logviewer --opensearch-endpoint "..." --elk-index "...*" --last 10m --size 10 query log

# Query for the field with the same restrictions
-> % logviewer --opensearch-endpoint "..." --elk-index "...*" --last 10m --size 10 query field

# Query with a filter on a field
-> % logviewer --opensearch-endpoint "..." --elk-index "...*" --last 10m --size 10 -f level=INFO query log

# Query with a custom format , all fields can be used and more , it's go template
-> % logviewer --opensearch-endpoint "https://logs-dev.elk.eu-west-1.nonprod.aws.eu" --elk-index "gfx*" --last 10m --size 10 --format "{{.Fields.level}} - {{.Message}}" query log
```


### Kubernetes

***still in early development and usure if it will stay on the long term***
***may be replace by using the kubectl command instead***

```bash
-> % logviewer --k8s-container frontend-dev-75fb7b89bb-9msbl --k8s-namespace growbe-prod  query log
```

### Docker

Will used your `$DOCKER_HOST` if `--docker-host` is not provided , only required arguments is the name
of the container to query log for.

```bash
logviewer query log --docker-host "unix:///Users/William.Quintal/.colima/lol/docker.sock" --docker-container "growbe-portal" --refresh --last 42h
```


```
{
    "clients": {
      "docker-local": {
        "type": "docker",
        "options": {
          "Host": "unix:///Users/William.Quintal/.colima/lol/docker.sock"
        }
      }
    },
    "contexts": {
      "growbe-portal": {
        "client": "docker-local",
        "search": {
          "range": {
            "last": "24h"
          },
          "fields": {
            
          },
          "options": {
            "Container": "growbe-portal"
          }
        }
      }
    }
  }
```

### Local/SSH

Query from local and ssh don't use mutch of the search field like for opensearch
in the future with the command builder depending on the context it may be used but for
now you have to configure the command to run yourself.

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


#### AWS CloudWatch

You can query CloudWatch Logs either by providing flags on the CLI or by creating a client in `config.json`.

Example: query the last 30 minutes from a specific log group using Insights

```bash
-> % logviewer --cloudwatch-log-group "/aws/lambda/my-func" --cloudwatch-region "us-east-1" --last 30m --size 50 query log
[12:01:34][INFO] ...log message...
```

If you need to target a LocalStack or custom endpoint set `--cloudwatch-endpoint` and to force the use of FilterLogEvents (instead of Insights) use `--cloudwatch-use-insights=false`.

Example `config.json` client entry for CloudWatch:

```json
"clients": {
  "local-cloudwatch": {
    "type": "cloudwatch",
    "options": {
      "region": "us-east-1",
      "profile": "default",
      "endpoint": "http://localhost:4566"
    }
  }
}
```

You can then reference the client in a context and set `logGroupName` in the `options` of the search if you prefer to keep the group inside the context.





## TUI

The TUI work only with configuration and it's really early development.
The inspiration was k9s and i want to do something similar in look to be able
to switch quickly between different preconfigured view to access logs and easily
do operation on them like filtering across multiple datasource.

For exemple you have two logs source from two application and you want to filter
both based on the correlation id. You could enter it once and filter both of your
request.

```bash
# You can specify many context to be executed and the TUI for now will
# create a split screen and pressing Ctrl+b with the selected panel
# will display the field
-> % logviewer -c ./config.json -i growbe-odoo -i growbe-ingress query
```




## MCP Server

LogViewer can also be run as an MCP server, exposing its core functionalities as a tool for Large Language Models (LLMs) and other AI agents. This enables programmatic access to log contexts, fields, and querying capabilities through natural language or structured commands.

### Starting the MCP Server

To start the MCP server, use the `mcp` command and provide a path to your configuration file:

```bash
logviewer mcp --config /path/to/your/config.json
```

By default, the server will listen on port `8081`. You can change this with the `--port` flag.

### Interacting with the MCP Server

Once the server is running, you can interact with it using an MCP client or any tool that can communicate with an MCP server.

**Example:**

```bash
# In one terminal, start the server
logviewer mcp --config ./config.json
```

## Server Mode

LogViewer can be run as a server, exposing its log querying capabilities via an HTTP API. This allows for programmatic access to the log aggregation engine.

### Starting the Server

To start the server, use the `server` command and provide a path to your configuration file:

```bash
logviewer server --config /path/to/your/config.json
```

By default, the server will listen on `0.0.0.0:8080`. You can change this with the `--host` and `--port` flags.








### API Endpoints

The server provides the following endpoints:

#### `GET /health`

Checks the health of the server.

```bash
curl -X GET http://localhost:8080/health
```

#### `GET /contexts`

Lists all available contexts from the configuration file.

```bash
curl -X GET http://localhost:8080/contexts
```

#### `GET /contexts/{contextId}`

Retrieves details for a specific context.

```bash
curl -X GET http://localhost:8080/contexts/my-context-id
```

#### `POST /query/logs`

Queries for log entries, equivalent to `logviewer query log`.

**Example:** Get the last 10 log entries for the `growbe-odoo` context.

```bash
curl -X POST http://localhost:8080/query/logs \
  -H "Content-Type: application/json" \
  -d '{
    "contextId": "growbe-odoo",
    "search": {
      "size": 10
    }
  }'
```

#### `POST /query/fields`

Queries for available fields, equivalent to `logviewer query field`.

**Example:** Get all available fields for the `growbe-odoo` context.

```bash
curl -X POST http://localhost:8080/query/fields \
  -H "Content-Type: application/json" \
  -d '{
    "contextId": "growbe-odoo"
  }'
```

## LLM Tool Integration

For AI tools that can only make GET requests (like Gemini CLI), use the GET endpoints:

### Discovery Pattern
```bash
# 1. Find available contexts
GET /contexts

# 2. Discover fields for a context
GET /query/fields?contextId=my-context

# 3. Query logs using discovered field names
GET /query/logs?contextId=my-context&fields=field_name=field_value&last=1h
```

### Example: Find Error Logs
```bash
# Using curl (what Gemini CLI does internally)
curl "http://localhost:8080/query/logs?contextId=nonprod-api&fields=level=ERROR&last=30m&size=20"
```
