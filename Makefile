.PHONY: build build/all release release/all test test/coverage integration/start integration/stop integration/tests integration/logs integration/start/logs integration/stop/logs integration/deploy-simulation install uninstall

SHA=$(shell git rev-parse --short HEAD)
# Determine latest tag (fallback to '0.0.0' when repository has no tags or git fails)
LATEST_TAG:=$(shell git describe --tags --abbrev=0 2>/dev/null || echo 0.0.0)

# Default VERSION is <LATEST_TAG>-<SHA> unless overridden on the make command line
VERSION?=$(LATEST_TAG)-$(SHA)

# Installation defaults (can be overridden on the `make` command line)
PREFIX ?= /usr/local

# Path to the generated k3s kubeconfig (created by integration/k8s/configure-kubeconfig.sh)
K3S_KUBECONFIG=integration/k8s/k3s.yaml

# Build targets

build:
	@echo "building (debug-friendly) version $(VERSION) for current platform"
	@go build -ldflags "-X github.com/bascanada/logviewer/cmd.sha1ver=$(VERSION)" -o build/logviewer


build/all:
	@echo "building (debug-friendly) version $(VERSION) for all platforms"
	@GOOS=linux GOARCH=arm64 go build -ldflags "-X github.com/bascanada/logviewer/cmd.sha1ver=$(VERSION)" -o build/logviewer-linux-arm64
	@GOOS=linux GOARCH=amd64 go build -ldflags "-X github.com/bascanada/logviewer/cmd.sha1ver=$(VERSION)" -o build/logviewer-linux-amd64
	@GOOS=darwin GOARCH=arm64 go build -ldflags "-X github.com/bascanada/logviewer/cmd.sha1ver=$(VERSION)" -o build/logviewer-darwin-arm64
	@GOOS=darwin GOARCH=amd64 go build -ldflags "-X github.com/bascanada/logviewer/cmd.sha1ver=$(VERSION)" -o build/logviewer-darwin-amd64


# Optimized / stripped release build (smaller binary, no DWARF/debug, trimmed paths)
# Usage: make release [VERSION=...] [CGO_ENABLED=0] [GOOS=...] [GOARCH=...] [OUTPUT=...]
OUTPUT ?= build/logviewer
release:
	@echo "building optimized release version $(VERSION) for $(or $(GOOS),current platform)/$(or $(GOARCH),current arch)"
	@mkdir -p build
	@CGO_ENABLED=${CGO_ENABLED-0} go build -trimpath -buildvcs=false \
		-ldflags "-s -w -X github.com/bascanada/logviewer/cmd.sha1ver=$(VERSION)" \
		-o $(OUTPUT)
	@echo "binary size: $$(wc -c < $(OUTPUT)) bytes"
	@echo "(add optional compression: upx --best $(OUTPUT))"


# Optimized multi-platform build (stripped)
release/all:
	@echo "building optimized release version $(VERSION) for all platforms"
	@$(MAKE) release GOOS=linux GOARCH=arm64 OUTPUT=build/logviewer-linux-arm64
	@$(MAKE) release GOOS=linux GOARCH=amd64 OUTPUT=build/logviewer-linux-amd64
	@$(MAKE) release GOOS=darwin GOARCH=arm64 OUTPUT=build/logviewer-darwin-arm64
	@$(MAKE) release GOOS=darwin GOARCH=amd64 OUTPUT=build/logviewer-darwin-amd64
	@$(MAKE) release GOOS=windows GOARCH=amd64 OUTPUT=build/logviewer-windows-amd64.exe
	@$(MAKE) release GOOS=windows GOARCH=arm64 OUTPUT=build/logviewer-windows-arm64.exe


# Install the built binary to a system location.
# Usage: make install [PREFIX=/usr/local] [DESTDIR=]
# Example: make install PREFIX=/usr/local
install:
	@echo "Installing logviewer with: PREFIX='$(PREFIX)'"
	@mkdir -p "$$PREFIX"
	@cp -f build/logviewer "$$PREFIX"
	@chmod 0755 "$$PREFIX/logviewer"
	@echo "Installed to $$PREFIX/logviewer"

# Remove installed binary
uninstall:
	@echo "Uninstalling logviewer from: PREFIX='$(PREFIX)'"
	@rm -f "$$PREFIX/logviewer"
	@echo "Removed $$PREFIX/logviewer"





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
	@rm -rf ./integration/splunk/.hec_token

# Service-specific start/stop
integration/start/splunk:
	@echo "Starting Splunk..."
	@cd integration && docker-compose up -d splunk

integration/stop/splunk:
	@echo "Stopping Splunk..."
	@cd integration && docker-compose stop splunk && docker-compose rm -fv splunk
	@rm -f ./integration/splunk/.hec_token

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

integration/start/logs:
	@echo "Starting log-generator..."
	@export SPLUNK_HEC_TOKEN=$$(cat ./integration/splunk/.hec_token 2>/dev/null || echo "") && \
		cd integration && docker-compose -f docker-compose-log-generator.yml up -d

integration/stop/logs:
	@echo "Stopping log-generator..."
	@cd integration && docker-compose -f docker-compose-log-generator.yml down -v

integration/deploy-simulation:
	@echo "Building simulator image..."
	@docker build -t log-generator:latest integration/log-generator
	
	@echo "Importing image to k3s..."
	@docker save log-generator:latest | docker exec -i k3s-server ctr images import -

	@echo "Applying K8s manifests..."
	@if [ -f integration/splunk/.hec_token ]; then \
		export TOKEN=$$(cat integration/splunk/.hec_token) && \
		export SPLUNK_IP=$$(docker inspect splunk --format '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}') && \
		export OPENSEARCH_IP=$$(docker inspect opensearch --format '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}') && \
		export LOCALSTACK_IP=$$(docker inspect localstack --format '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}') && \
		sed -e "s/YOUR_HEC_TOKEN_HERE/$$TOKEN/g" \
		    -e "s|https://splunk:8088|https://$$SPLUNK_IP:8088|g" \
		    -e "s|http://opensearch:9200|http://$$OPENSEARCH_IP:9200|g" \
		    -e "s|http://localstack:4566|http://$$LOCALSTACK_IP:4566|g" \
		    integration/k8s/app.yaml | \
		KUBECONFIG=$(K3S_KUBECONFIG) kubectl apply -f -; \
	else \
		echo "ERROR: Splunk HEC token not found. Run 'make integration/start' first."; \
		exit 1; \
	fi

	@echo "Simulation deployed! Logs are flowing."


# Log Generation and Uploading
integration/logs: integration/logs/generator integration/logs/ssh integration/logs/cloudwatch

integration/logs/cloudwatch:
	@echo "Sending logs to CloudWatch..."
	@cd integration/cloudwatch && ./send-logs.sh

integration/logs/generator: integration/start/logs
	@echo "Deploying sample logs to Splunk and OpenSearch via log-generator..."
	@for i in $$(seq 1 30); do \
		if curl -s http://localhost:8081 >/dev/null; then break; fi; \
		echo "Waiting for log-generator..."; \
		sleep 1; \
	done
	@curl -s -o /dev/null -G --data-urlencode "message=User 'alice' logged in successfully" http://localhost:8081/log/info
	@curl -s -o /dev/null -G --data-urlencode "message=Payment failed for order #12345: Insufficient funds" http://localhost:8081/log/error
	@curl -s -o /dev/null -H "X-Request-ID: xyz-987-abc" -G --data-urlencode "message=API key is approaching expiration date" http://localhost:8081/log/warn

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

