.PHONY: build build/all splunk/test test test/coverage

SHA=$(shell git rev-parse --short HEAD)
VERSION?=$(SHA)

# Path to the generated k3s kubeconfig (created by integration/k8s/configure-kubeconfig.sh)
K3S_KUBECONFIG=integration/k8s/k3s.yaml


# Build targets

build:
	@echo "building (debug-friendly) version $(VERSION) for current platform"
	@go build -ldflags "-X github.com/berlingoqc/logviewer/cmd.sha1ver=$(VERSION)" -o build/logviewer

# Optimized / stripped release build (smaller binary, no DWARF/debug, trimmed paths)
# Usage: make release [VERSION=...] [CGO_ENABLED=0]
release:
	@echo "building optimized release version $(VERSION)"
	@mkdir -p build
	@CGO_ENABLED=${CGO_ENABLED-0} go build -trimpath -buildvcs=false \
		-ldflags "-s -w -X github.com/berlingoqc/logviewer/cmd.sha1ver=$(VERSION)" \
		-o build/logviewer
	@echo "binary size: $$(wc -c < build/logviewer) bytes"
	@echo "(add optional compression: upx --best build/logviewer)"

build/all:
	@echo "building (debug-friendly) version $(VERSION) for all platforms"
	@GOOS=linux GOARCH=arm64 go build -ldflags "-X github.com/berlingoqc/logviewer/cmd.sha1ver=$(VERSION)" -o build/logviewer-linux-arm64
	@GOOS=linux GOARCH=amd64 go build -ldflags "-X github.com/berlingoqc/logviewer/cmd.sha1ver=$(VERSION)" -o build/logviewer-linux-amd64
	@GOOS=darwin GOARCH=arm64 go build -ldflags "-X github.com/berlingoqc/logviewer/cmd.sha1ver=$(VERSION)" -o build/logviewer-darwin-arm64
	@GOOS=darwin GOARCH=amd64 go build -ldflags "-X github.com/berlingoqc/logviewer/cmd.sha1ver=$(VERSION)" -o build/logviewer-darwin-amd64

# Optimized multi-platform build (stripped)
release/all:
	@echo "building optimized release version $(VERSION) for all platforms"
	@GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -trimpath -buildvcs=false -ldflags "-s -w -X github.com/berlingoqc/logviewer/cmd.sha1ver=$(VERSION)" -o build/logviewer-linux-arm64
	@GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -buildvcs=false -ldflags "-s -w -X github.com/berlingoqc/logviewer/cmd.sha1ver=$(VERSION)" -o build/logviewer-linux-amd64
	@GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -trimpath -buildvcs=false -ldflags "-s -w -X github.com/berlingoqc/logviewer/cmd.sha1ver=$(VERSION)" -o build/logviewer-darwin-arm64
	@GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -buildvcs=false -ldflags "-s -w -X github.com/berlingoqc/logviewer/cmd.sha1ver=$(VERSION)" -o build/logviewer-darwin-amd64


# Unit tests

test:
	@go test ./...

test/coverage:
	@command -v gocover-cobertura >/dev/null 2>&1 || { echo "Installing gocover-cobertura"; go install github.com/boumenot/gocover-cobertura@latest; }
	@go test -coverprofile=coverage.txt -covermode count ./...
	@cat coverage.txt | gocover-cobertura > coverage.xml


# Integration Environment Management
integration/start:
	@echo "Starting all integration services..."
	@bash integration/ssh/generate-keys.sh
	@cd integration && docker-compose up -d
	@./integration/splunk/wait-for-splunk.sh
	@./integration/k8s/configure-kubeconfig.sh
	@docker ps

integration/stop:
	@echo "Stopping all integration services..."
	@cd integration && docker-compose down -v

# Service-specific start/stop
integration/start/splunk:
	@echo "Starting Splunk..."
	@cd integration && docker-compose up -d splunk

integration/stop/splunk:
	@echo "Stopping Splunk..."
	@cd integration && docker-compose stop splunk && docker-compose rm -fv splunk

integration/start/opensearch:
	@echo "Starting OpenSearch and Dashboards..."
	@cd integration && docker-compose up -d opensearch opensearch-dashboards

integration/stop/opensearch:
	@echo "Stopping OpenSearch and Dashboards..."
	@cd integration && docker-compose stop opensearch opensearch-dashboards && docker-compose rm -fv opensearch opensearch-dashboards

integration/start/ssh:
	@echo "Starting SSH server..."
	@bash integration/ssh/generate-keys.sh
	@cd integration && docker-compose up -d ssh-server

integration/stop/ssh:
	@echo "Stopping SSH server..."
	@cd integration && docker-compose stop ssh-server && docker-compose rm -fv ssh-server

integration/start/k8s:
	@echo "Starting k3s server..."
	@cd integration && docker-compose up -d k3s-server

integration/stop/k8s:
	@echo "Stopping k3s server..."
	@cd integration && docker-compose stop k3s-server && docker-compose rm -fv k3s-server

# Service-specific start/stop
integration/start/cloudwatch:
	@echo "Starting LocalStack for CloudWatch..."
	@cd integration && docker-compose up -d localstack

integration/stop/cloudwatch:
	@echo "Stopping LocalStack..."
	@cd integration && docker-compose stop localstack && docker-compose rm -f localstack

# Log Generation and Uploading
integration/logs: integration/logs/splunk integration/logs/opensearch integration/logs/ssh integration/logs/cloudwatch

integration/logs/cloudwatch:
	@echo "Sending logs to CloudWatch..."
	@cd integration/cloudwatch && ./send-logs.sh

integration/logs/splunk:
	@echo "Sending logs to Splunk..."
	@cd integration/splunk && ./send-logs.sh

integration/logs/opensearch:
	@echo "Sending logs to OpenSearch..."
	@cd integration/opensearch && ./send-logs.sh

integration/logs/ssh:
	@echo "Uploading logs to SSH server..."
	@cd integration/ssh && ./upload-log.sh

integration/tests: build
	@echo "Running integration tests..."
	@echo "Querying logs for splunk"
	@build/logviewer query log -c ./config.json -i splunk-app-logs
	@echo "Querying logs for ssh"
	@build/logviewer query log -c ./config.json -i ssh-app-log
	@echo "Querying logs for opensearch"
	@build/logviewer query log -c ./config.json -i opensearch-app-logs --last 24h
	@echo "Querying logs for docker"
	@DOCKER_CID=$$(docker ps --filter name=ssh-server -q | head -n1) \
		build/logviewer query log -c ./config.json -i docker-sample-container
	@echo "Querying logs for k3s coredns"
	@COREDNS_POD=$$(KUBECONFIG=$(K3S_KUBECONFIG) kubectl get pods -n kube-system -l k8s-app=kube-dns -o jsonpath='{.items[0].metadata.name}') \
		build/logviewer query log -c ./config.json -i k3s-coredns --size 200
	@echo "Querying logs from localstack"
	@build/logviewer query log -c ./config.json -i cloudwatch-app-logs --last 24h --size 3	