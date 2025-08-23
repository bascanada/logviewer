
# logviewer

Terminal based log viewer for multiple log source with search feature and view configuration.

***this application is in early development , i'm still testing things out , if you like give me feedback***

Log source at the moment are:

* Command local or by ssh
* Kubectl logs
* Opensearch/Kibana logs
* Splunk logs

Possible future source:

* AWS CloudWatch
* Docker
* Command builder ( for local and ssh logs based on what you want to look at)


## How to install

You can check [the release folder](https://github.com/berlingoqc/logviewer/releases) for prebuild binary.
You can use the development build or the standard release.

Other option is to use docker to run the application

```bash
logviewer() {
   docker run -it -v $HOME/.logviewer/config.json:/config.json -v $HOME/.ssh:/.ssh ghcr.io/berlingoqc/logviewer:latest "$@"
}
logviewer_update() {
   docker pull ghcr.io/berlingoqc/logviewer:latest
}
```

## How to use

There is main way to access the log

* Via the stdout , outputting directly in the terminal
* With the TUI , creating tmux like views for multiple log query


### Basic log query

```bash
-> % logviewer query --help
Query a login system for logs and available fields

Usage:
  logviewer query [flags]
  logviewer query [command]

Available Commands:
  field       Dispaly available field for filtering of logs
  log         Display logs for system
			  Without subcommand open the query TUI

Flags:
      --client-body string             File containing base body to be used by the underlying client
      --client-headers string          File containings list of headers to be used by the underlying client
      --cmd string                     If using ssh or local , manual command to run
  -c, --config string                  Config for preconfigure context for search
      --docker-container string        Docker container
      --docker-host string             Docker context
      --elk-index string               Elk index to search
  -f, --fields stringArray             Field for selection field=value
      --fields-condition stringArray   Field Ops for selection field=value (match, exists, wildcard, regex)
      --fields-regex string            Regex to extract field from log text, using named group ".*(?P<Level>INFO|WARN|ERROR).*"
      --from string                    Get entry gte datetime date >= from
  -h, --help                           help for query
  -i, --id stringArray                 Context id to execute
      --inherits stringArray           When using config , list of inherits to execute on top of the one configure for the search
      --k8s-container string           K8s container
      --k8s-namespace string           K8s namespace
      --k8s-pod string                 K8s pod
      --k8s-previous                   K8s log of previous container
      --k8s-timestamp                  K8s include RFC3339 timestamp
      --kibana-endpoint string         Kibana endpoint
      --last string                    Get entry in the last duration
      --mylog                          read from logviewer logs file
      --opensearch-endpoint string     Opensearch endpoint
      --size int                       Get entry max size
      --splunk-endpoint string         Splunk endpoint
      --ssh-addr string                SSH address and port localhost:22
      --ssh-identifiy string           SSH private key , by default $HOME/.ssh/id_rsa
      --ssh-user string                SSH user
      --to string                      Get entry lte datetime date <= to

Global Flags:
      --logging-level string   logging level to output INFO WARN ERROR DEBUG TRACE
      --logging-path string    file to output logs of the application
      --logging-stdout         output appplication log in the stdout

Use "logviewer query [command] --help" for more information about a command.
```

#### Query from opensearch

```bash
# Query max of 10 logs entry in the last 10 minute for an index in an instance
-> % logviewer --opensearch-endpoint "..." --elk-index "...*" --last 10m --size 10 query log
[19:51:34][INFO] name='/health/healthcheck' total=1 
[19:51:34][INFO] Getting pending jobs to schedule...
[19:51:34][INFO] Job(0) scheduled in 0(ms)
[19:51:34][WARN] Select expired jobs, expiration max delay 1, returned 0 jobs
[19:51:35][INFO] name='/health/healthcheck' total=1 
[19:51:35][INFO] name='/health/healthcheck' total=1 
[19:51:35][INFO] name='/health/healthcheck' total=1 
[19:51:35][INFO] name='/health/healthcheck' total=1 
[19:51:36][INFO] name='/health/healthcheck' total=1 
[19:51:36][INFO] name='/health/healthcheck' total=1 

# Query for the field with the same restrictions
-> % logviewer --opensearch-endpoint "..." --elk-index "...*" --last 10m --size 10 query field
thread 
    http-nio-8080-exec-5
    http-nio-8080-exec-9
    health.check-thread-5
    health.check-thread-2
    health.check-thread-4
    http-nio-8080-exec-7
    http-nio-8080-exec-10
environment 
    dev
    intqa
level 
    INFO

# Query with a filter on a field
-> % logviewer --opensearch-endpoint "..." --elk-index "...*" --last 10m --size 10 -f level=INFO query log
[19:51:34][WARN] Select expired jobs, expiration max delay 1, returned 0 jobs

# Query with a custom format , all fields can be used and more , it's go template
-> % logviewer --opensearch-endpoint "https://logs-dev.elk.eu-west-1.nonprod.aws.eu" --elk-index "gfx*" --last 10m --size 10 --format "{{.Fields.level}} - {{.Message}}" query log
INFO - Message sent to SQS with SQS-assigned messageId: a64e36bf-9418-4c06-93d7-311424dee65c
INFO - Message sent to SQS with SQS-assigned messageId: 6da1de09-aa1e-4295-abe4-c8eb457775ad
INFO - Shutting down SessionCallBackScheduler executor
INFO - Shutting down SessionCallBackScheduler executor
INFO - Shutting down SessionCallBackScheduler executor
INFO - Message sent to SQS with SQS-assigned messageId: 9d741e5d-1abf-4ae0-3554-798a0cc7f1b9
INFO - name='/health/healthcheck' total=0 
INFO - name='/health/healthcheck' total=1 
INFO - name='/health/healthcheck' total=0 
INFO - name='/health/healthcheck' total=1 
```

#### Query from kubernetes

***still in early development and usure if it will stay on the long term***
***may be replace by using the kubectl command instead***

```bash
-> % logviewer --k8s-container frontend-dev-75fb7b89bb-9msbl --k8s-namespace growbe-prod  query log
```

#### Query from docker

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



#### Query from command local or ssh

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


### Creating configuration file to save client and log search

For more conveniance you can create configuration for the client and search you want to do.

Create a configuration json file with the following format.

```json
{
  // Map of all client configuration
  "clients": {
    "local": {
      "type": "local",
      "options": {}
    },
    "growbe": {
      "type": "kibana",
      "options": {
        "Endpoint": "http://myinstance"
      }
    },
	"growbe-os": {
      "type": "opensearch",
      "options": {
        "Endpoint": "http://myinstance"
      }
    }
  },
  // You can define search object that will be used to create a query
  "searches": {
    "growbe": {
      "range": {
        "last": "15m"
      },
      "refresh": {
        "duration": "3s"
      },
      "size": 100,
      "options": {
        "Index": "logstash-*"
      },
      "printerOptions": {
        "template": "{{.Message}}"
      }
    }
  },
  // Map of all search you can use
  "contexts": {
	// Name of the search
    "growbe-ingress": {
	  // client to be used
      "client": "growbe",
	  // array of search object, will be overwritten from first to last finish with
	  // this search object
	  "searchInherit": ["growbe"],
	  // search object
      "search": {
        "fields": {
          "kubernetes.container_image": "k8s.gcr.io/ingress-nginx/controller:v1.2.0"
        }
      }
    },
    "growbe-odoo": {
      "client": "growbe",
      "search": {
        "range": {
          "last": "20m"
        },
        "fields": {
          "kubernetes.container_image": "docker.io/bitnami/odoo:16.0.20221115-debian-11-r13"
        }
      }
    }
  }
}
```

You can then use this configuration file to do some of the query.

```bash
-> % logviewer -c ./config.json -i growbe-odoo query log
...

# You can also used the command line options to overwrite some settings
-> % logviewer -c ./config.json -i growbe-odoo query log --size 300
...
```

### Using the TUI

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

## Todo

* Mix multiple datasource in the same flow (maybe be complicated to order them correctly)
* Expand JSON model if present
* Create custom function for template to display multiline pretty things.
* Add color
* PrinterOptions template dont work in config
* Automatic refresh if need be
* Supporter template dans les expression regex ou autre moyens pour simplifier l'exctractions
