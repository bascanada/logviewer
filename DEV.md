
# Development

## Integration environment

You can start a integration environment with docker and docker-compose that start:

* splunk
* opensearch
* aws LocalStack
* k3s
* ssh server

Dependencies:

* docker
* docker-compose
* awscli
* kubectl
* jq

### Start all instance supported

```bash
make integration/start
```

### Start instance to forward logs to opensearch and splunk

```bash
make integration/start/logs
```

### Deploy logs everywhere

```bash
make integration/logs
```

### Now you can use the default config to query the instance in docker

```bash
go run . query  -c ./config.yaml -i splunk-app-logs log
```