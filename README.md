# logviewer

***this application is in early development , i'm still testing things out , if you like give me feedback***

Terminal based log client for multiple source with search feature and configuration.
The goal is to provide a one time cli tool to read the logs from all the places you
decided to store them.

Providing a unified way to build log queries that will work across all implementations or, when needed,
pass raw requests to the backend implementation.

Log source at the moment are:

* Command local or by ssh
* K8S
* Opensearch/Kibana
* Splunk
* AWS CloudWatch
* Docker


Way to use logviewer

* CLI like Curl
* TUI like K9S
* MCP to integrate with your favorite LLM agent like copilot, gemini or claude.
* HTTP server if you want the app remotely

Example of the core functionality

```
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

To handle a lot of differents log servers and log context , you can use configuration files to store
all your configurations. See the `config.json` for exemple configuration. The parts are:

1. clients: connection to an endpoint
2. searches: block of fields and options that can be inherit by a context as base value
3. contexts: a search context client + a list of searches + override values.

```
{
  ...
  "searches": {
    "app1": {
      "fields": {
        "applicationName": "com.myapp"
      },
      "options": {
        "index": "com.*"
      }
    }
  },
  "contexts": {
    "app1-errors": {
      "client": "local-splunk",
      "searchInherit": ["app1"],
      "search": {
        "fields": {
          "level": "ERROR"
        }
      }
    }
  }
}
```

## Config file discovery & formats

logviewer supports JSON and YAML config files and looks for the configuration in the following order:

1. Explicit path provided with the CLI flag `-c, --config /path/to/config` (highest precedence).
2. The environment variable `LOGVIEWER_CONFIG` if set (e.g. `export LOGVIEWER_CONFIG=/path/to/config.yaml`).
3. The default file at `$HOME/.logviewer/config.yaml` (only used when no `-c` and no env var are provided).

Notes:
- Supported formats: JSON (`.json`) and YAML (`.yaml`, `.yml`). The loader will detect by extension, and as a fallback try JSON then YAML when the extension is missing or unknown.
- Command-specific behavior: the `query` command will only auto-load the default/env config when you also provide a context id with `-i/--id` (since the config is only meaningful when using contexts). The `server` and `mcp` commands attempt to load a configuration at startup (they will print actionable error messages if the config is invalid or missing required sections).
- Error messages: when a config file is invalid or missing required sections the CLI/server/mcp will return clear, actionable errors (invalid format, missing `clients`, missing `contexts`, etc.).

Examples:

```bash
# explicit path
logviewer --config /etc/logviewer/config.yaml query -i my-context --last 10m

# use env var
export LOGVIEWER_CONFIG=$HOME/.logviewer/config.yaml
logviewer query -i my-context --last 1h

# default location (if present) is $HOME/.logviewer/config.yaml
logviewer server
```




## How to install

You can check [the release folder](https://github.com/bascanada/logviewer/releases) for prebuild binary.
You can use the development build or the standard release.

Build manually

```
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
```

```
{
  "clients": {
    "local-opensearch": {
      "type": "opensearch",
      "options": {
        "endpoint": "http://localhost:9200"
      }
    }
  },
  "contexts": {
    "opensearch-app-logs": {
      "client": "local-opensearch",
      "searchInherit": [],
      "search": {
        "fields": {},
        "options": {
          "index": "app-logs"
        }
      }
    }
  }
}
```

### Splunk

```bash
# Query max of 10 logs entry in the last 10 minute for an index in an instance
-> % logviewer --splunk-url "..." --splunk-index "..." --last 10m --size 10 query log
```

```json
{
  "clients": {
    "local-splunk": {
      "type": "splunk",
      "options": {
        "url": "http://localhost:8088"
      }
    }
  },
  "contexts": {
    "splunk-app-logs": {
      "client": "local-splunk",
      "searchInherit": [],
      "search": {
        "fields": {},
        "options": {
          "index": "app-logs",
          "fields": [
            "field1",
            "field2"
          ]
        }
      }
    }
  }
}
```

#### Fields

In Splunk, some fields are not indexed by default and you need to use the `| fields + <field>` syntax to include them in your search.
You can use the `fields` option in your `search.options` to add a list of fields to be added to the search query.


### K8S

```bash
-> % logviewer --k8s-container frontend-dev-75fb7b89bb-9msbl --k8s-namespace growbe-prod  query log
```

```
{
  "clients": {
    "local-k3s": {
      "type": "k8s",
      "options": {
        "kubeConfig": "integration/k8s/k3s.yaml"
      }
    }
  },
  "contexts": {
    "k3s-coredns": {
      "client": "local-k3s",
      "searchInherit": [],
      "search": {
        "fields": {},
        "options": {
          "namespace": "kube-system",
          "pod": "${COREDNS_POD}",
          "timestamp": true
        }
      }
    }
  }
}
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
    "local-docker": {
      "type": "docker",
      "options": {
        "host": "unix:///var/run/docker.sock"
      }
    }
  },
  "contexts": {
    "docker-sample-container": {
      "client": "local-docker",
      "searchInherit": [],
      "search": {
        "fields": {},
        "options": {
          "container": "${DOCKER_CID}",
          "showStdout": true,
          "showStderr": true,
          "timestamps": true,
          "details": false
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


```
{
  "clients": {
    "local-ssh": {
      "type": "ssh",
      "options": {
        "user": "testuser",
        "addr": "127.0.0.1:2222",
        "privateKey": "integration/ssh/id_rsa"
      }
    }
  },
  "contexts": {
    "ssh-app-log": {
      "client": "local-ssh",
      "searchInherit": [],
      "search": {
        "fields": {},
        "options": {
          "cmd": "tail -n 200 app.log"
        }
      }
    }
  }
}
```



#### AWS CloudWatch

You can query CloudWatch Logs either by providing flags on the CLI or by creating a client in `config.json`.

Example: query the last 30 minutes from a specific log group using Insights

```bash
-> % logviewer --cloudwatch-log-group "/aws/lambda/my-func" --cloudwatch-region "us-east-1" --last 30m --size 50 query log
[12:01:34][INFO] ...log message...
```

If you need to target a LocalStack or custom endpoint set `--cloudwatch-endpoint` and to force the use of FilterLogEvents (instead of Insights) use `--cloudwatch-use-insights=false`.

```
{
  "clients": {
    "local-cloudwatch": {
      "type": "cloudwatch",
      "options": {
        "region": "us-east-1",
        "endpoint": "http://localhost:4566"
      }
    }
  },
  "contexts": {
    "cloudwatch-app-logs": {
      "client": "local-cloudwatch",
      "searchInherit": [],
      "search": {
        "fields": {},
        "options": {
          "logGroupName": "my-app-logs",
          "useInsights": "false"
        }
      }
    }
  }
}
```


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