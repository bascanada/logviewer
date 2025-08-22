.PHONY: build build/all splunk/dev/start splunk/dev/stop splunk/test test test/coverage

SHA=$(shell git rev-parse --short HEAD)
VERSION?=$(SHA)

build:
	@echo "building version $(VERSION) for current platform"
	@go build -ldflags "-X github.com/berlingoqc/logviewer/cmd.sha1ver=$(VERSION)" -o build/logviewer

build/all:
	@echo "building version $(VERSION) for all platforms"
	@GOOS=linux GOARCH=arm64 go build -ldflags "-X github.com/berlingoqc/logviewer/cmd.sha1ver=$(VERSION)" -o build/logviewer-linux-arm64
	@GOOS=linux GOARCH=amd64 go build -ldflags "-X github.com/berlingoqc/logviewer/cmd.sha1ver=$(VERSION)" -o build/logviewer-linux-amd64
	@GOOS=darwin GOARCH=arm64 go build -ldflags "-X github.com/berlingoqc/logviewer/cmd.sha1ver=$(VERSION)" -o build/logviewer-darwin-arm64
	@GOOS=darwin GOARCH=amd64 go build -ldflags "-X github.com/berlingoqc/logviewer/cmd.sha1ver=$(VERSION)" -o build/logviewer-darwin-amd64

test:
	go test ./...

test/coverage:
	go test -coverprofile=coverage.txt -covermode count ./... && cat coverage.txt | gocover-cobertura > coverage.xml

splunk/dev/start:
	@echo "Starting Splunk for development..."
	@cd integration/splunk && ./start-splunk-dev.sh

splunk/dev/stop:
	@echo "Stopping Splunk for development..."
	@cd integration/splunk && docker-compose down

splunk/test:
	@echo "Running Splunk integration tests..."
	@cd integration/splunk && ./run-integration-tests.sh
