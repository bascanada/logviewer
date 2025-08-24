.PHONY: build build/all splunk/test test test/coverage

SHA=$(shell git rev-parse --short HEAD)
VERSION?=$(SHA)



# Build targets

build:
	@echo "building version $(VERSION) for current platform"
	@go build -ldflags "-X github.com/berlingoqc/logviewer/cmd.sha1ver=$(VERSION)" -o build/logviewer

build/all:
	@echo "building version $(VERSION) for all platforms"
	@GOOS=linux GOARCH=arm64 go build -ldflags "-X github.com/berlingoqc/logviewer/cmd.sha1ver=$(VERSION)" -o build/logviewer-linux-arm64
	@GOOS=linux GOARCH=amd64 go build -ldflags "-X github.com/berlingoqc/logviewer/cmd.sha1ver=$(VERSION)" -o build/logviewer-linux-amd64
	@GOOS=darwin GOARCH=arm64 go build -ldflags "-X github.com/berlingoqc/logviewer/cmd.sha1ver=$(VERSION)" -o build/logviewer-darwin-arm64
	@GOOS=darwin GOARCH=amd64 go build -ldflags "-X github.com/berlingoqc/logviewer/cmd.sha1ver=$(VERSION)" -o build/logviewer-darwin-amd64


# Unit tests

test:
	@go test ./...

test/coverage:
	@go test -coverprofile=coverage.txt -covermode count ./... && cat coverage.txt | gocover-cobertura > coverage.xml


# Integration Environment Management
integration/start:
	@echo "Starting all integration services..."
	@cd integration && docker-compose up -d

integration/stop:
	@echo "Stopping all integration services..."
	@cd integration && docker-compose down

# Service-specific start/stop
integration/start/splunk:
	@echo "Starting Splunk..."
	@cd integration && docker-compose up -d splunk

integration/stop/splunk:
	@echo "Stopping Splunk..."
	@cd integration && docker-compose stop splunk && docker-compose rm -f splunk

integration/start/opensearch:
	@echo "Starting OpenSearch and Dashboards..."
	@cd integration && docker-compose up -d opensearch opensearch-dashboards

integration/stop/opensearch:
	@echo "Stopping OpenSearch and Dashboards..."
	@cd integration && docker-compose stop opensearch opensearch-dashboards && docker-compose rm -f opensearch opensearch-dashboards

integration/start/ssh:
	@echo "Starting SSH server..."
	@cd integration && docker-compose up -d ssh-server

integration/stop/ssh:
	@echo "Stopping SSH server..."
	@cd integration && docker-compose stop ssh-server && docker-compose rm -f ssh-server

integration/start/k3s:
	@echo "Starting k3s server..."
	@cd integration && docker-compose up -d k3s-server

integration/stop/k3s:
	@echo "Stopping k3s server..."
	@cd integration && docker-compose stop k3s-server && docker-compose rm -f k3s-server

# Log Generation and Uploading
integration/logs/splunk:
	@echo "Sending logs to Splunk..."
	@cd integration/splunk && ./send-logs.sh

integration/logs/opensearch:
	@echo "Sending logs to OpenSearch..."
	@cd integration/opensearch && ./send-logs.sh

integration/logs/ssh:
	@echo "Uploading logs to SSH server..."
	@cd integration/ssh && ./upload-log.sh

# Kubernetes Management
integration/k8s/configure:
	@echo "Configuring kubectl for k3s..."
	@chmod +x integration/k8s/configure-kubeconfig.sh
	@./integration/k8s/configure-kubeconfig.sh

integration/k8s/deploy:
	@echo "Deploying log-emitter pod to k3s..."
	@KUBECONFIG=$$HOME/.kube/k3s.yaml kubectl apply -f integration/k8s/log-emitter.yml

integration/k8s/delete:
	@echo "Deleting log-emitter pod from k3s..."
	@KUBECONFIG=$$HOME/.kube/k3s.yaml kubectl delete -f integration/k8s/log-emitter.yml

integration/k8s/logs:
	@echo "Tailing logs from log-emitter pod..."
	@KUBECONFIG=$$HOME/.kube/k3s.yaml kubectl logs -f log-emitter