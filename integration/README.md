# Integration Testing Environment

Complete guide for testing LogViewer across all supported backends.

## Table of Contents

- [Quick Reference](#quick-reference)
- [Starting Infrastructure](#starting-infrastructure)
- [Pushing Logs to Backends](#pushing-logs-to-backends)
- [Testing Live Follow/Refresh](#testing-live-followrefresh)
- [Running Integration Tests](#running-integration-tests)
- [Configuration Contexts](#configuration-contexts)
- [Troubleshooting](#troubleshooting)
- [MCP Agent Tests](#mcp-agent-tests)

---

## Quick Reference

### Ports & Services

| Service | Port | UI/API | Credentials |
|---------|------|--------|-------------|
| **Splunk** | 8000 | Web UI | admin/changeme |
| **Splunk HEC** | 8088 | Log ingestion | Token in `splunk/.hec_token` |
| **Splunk API** | 8089 | REST API | admin/changeme |
| **OpenSearch** | 9200 | REST API | - |
| **OpenSearch Dashboards** | 5601 | Web UI | - |
| **K3s** | 6443 | Kubernetes API | kubeconfig: `k8s/k3s.yaml` |
| **SSH** | 2222 | SSH access | user: `testuser`, key: `ssh/id_rsa` |
| **LocalStack** | 4566 | AWS emulation | - |

### Essential Commands

```bash
# Start/stop everything
make integration/start
make integration/stop

# Build logviewer
make build

# Run all integration tests
make integration/tests

# Query with config
LOGVIEWER_CONFIG="integration/config.yaml" ./build/logviewer query log -i <context> --last 5m
```

---

## Starting Infrastructure

### Start All Services

```bash
make integration/start
```

### Start Individual Services

```bash
make integration/start/splunk       # Splunk only
make integration/start/opensearch   # OpenSearch only
make integration/start/k8s          # K3s Kubernetes cluster
make integration/start/ssh          # SSH server
```

### Or Use Docker Compose Directly

```bash
cd integration
docker-compose up -d                           # All services
docker-compose up -d splunk                    # Splunk only
docker-compose up -d opensearch opensearch-dashboards  # OpenSearch only
docker-compose up -d k3s-server                # K3s only
docker-compose up -d ssh-server                # SSH only
```

### Verify Services Are Running

```bash
# Check all containers
docker-compose -f integration/docker-compose.yml ps

# Test Splunk
curl -s http://localhost:8089/services/server/health | head -5

# Test OpenSearch
curl -s http://localhost:9200/_cluster/health | jq .

# Test K3s
kubectl --kubeconfig=integration/k8s/k3s.yaml get nodes

# Test SSH
ssh -i integration/ssh/id_rsa -p 2222 testuser@localhost "echo OK"
```

---

## Pushing Logs to Backends

### 1. Splunk (via HEC)

```bash
# Option A: Use the send-logs script
./integration/splunk/send-logs.sh

# Option B: Send manually via curl
HEC_TOKEN=$(cat integration/splunk/.hec_token)
curl -k "http://localhost:8088/services/collector/event" \
  -H "Authorization: Splunk $HEC_TOKEN" \
  -d '{"event": {"message": "test log", "level": "INFO"}, "sourcetype": "json"}'

# Option C: Send a JSON file
cat mylog.json | while read line; do
  curl -k "http://localhost:8088/services/collector/event" \
    -H "Authorization: Splunk $HEC_TOKEN" \
    -d "{\"event\": $line}"
done
```

### 2. OpenSearch

```bash
# Option A: Use the send-logs script
./integration/opensearch/send-logs.sh

# Option B: Send manually
curl -X POST "http://localhost:9200/logs-test/_doc" \
  -H "Content-Type: application/json" \
  -d '{"@timestamp": "2024-01-15T10:30:00Z", "message": "test log", "level": "INFO"}'

# Option C: Bulk insert
curl -X POST "http://localhost:9200/_bulk" \
  -H "Content-Type: application/x-ndjson" \
  --data-binary @- << 'EOF'
{"index": {"_index": "logs-test"}}
{"@timestamp": "2024-01-15T10:30:00Z", "message": "log 1", "level": "INFO"}
{"index": {"_index": "logs-test"}}
{"@timestamp": "2024-01-15T10:30:01Z", "message": "log 2", "level": "ERROR"}
EOF
```

### 3. Kubernetes (Pod Logs)

```bash
# Deploy the transaction simulator (generates logs automatically)
make integration/deploy-simulation

# Or deploy a simple logging pod
kubectl --kubeconfig=integration/k8s/k3s.yaml run log-test \
  --image=busybox \
  --restart=Never \
  -- sh -c 'while true; do echo "{\"timestamp\":\"$(date -Iseconds)\",\"level\":\"INFO\",\"message\":\"test\"}"; sleep 1; done'

# View pod logs
kubectl --kubeconfig=integration/k8s/k3s.yaml logs -f log-test
```

### 4. Docker Container Logs

```bash
# Start the log generator container
docker-compose -f integration/docker-compose-log-generator.yml up -d

# Or run any container with logging
docker run -d --name log-test alpine sh -c \
  'while true; do echo "{\"level\":\"INFO\",\"message\":\"test $(date)\"}"; sleep 1; done'

# View logs
docker logs -f log-test
```

### 5. SSH (Remote Files)

```bash
# Generate test logs on the SSH server
./integration/ssh/generate-logs.sh

# Or upload existing logs
./integration/ssh/upload-log.sh

# Or manually copy logs
scp -i integration/ssh/id_rsa -P 2222 mylog.json testuser@localhost:/home/testuser/logs/

# Verify logs exist
ssh -i integration/ssh/id_rsa -p 2222 testuser@localhost "ls -la /home/testuser/logs/"
```

### 6. Continuous Log Generation (Transaction Simulator)

The transaction simulator generates realistic microservice logs across multiple backends:

```bash
# Deploy to K8s (recommended)
make integration/deploy-simulation

# Or run locally
cd integration/log-generator
SPLUNK_HEC_URL=http://localhost:8088 \
SPLUNK_HEC_TOKEN=$(cat ../splunk/.hec_token) \
OPENSEARCH_URL=http://localhost:9200 \
go run main.go

# Trigger manual transactions
curl http://localhost:8081/checkout
```

**Log Distribution:**
- Frontend logs → K8s stdout
- Order logs → OpenSearch (`orders` index)
- Payment logs → Splunk (HEC)
- Database logs → K8s stdout

---

## Testing Live Follow/Refresh

### Understanding Refresh Types

| Backend | Refresh Type | Flag | How It Works |
|---------|--------------|------|--------------|
| Local, SSH, Docker, K8s | **Streaming** | `--refresh` | Real-time tail/follow |
| Splunk, OpenSearch | **Polling** | `--refresh-rate <duration>` | Periodic re-query |
| CloudWatch | No refresh | N/A | Query-only |

### Test Commands

```bash
# Build first
make build

# ─────────────────────────────────────────────────────────
# SPLUNK - Uses polling refresh
# ─────────────────────────────────────────────────────────
LOGVIEWER_CONFIG="integration/config.yaml" ./build/logviewer query log \
  -i payment-service --last 5m --refresh-rate 5s

# ─────────────────────────────────────────────────────────
# OPENSEARCH - Uses polling refresh
# ─────────────────────────────────────────────────────────
LOGVIEWER_CONFIG="integration/config.yaml" ./build/logviewer query log \
  -i order-service --last 5m --refresh-rate 5s

# ─────────────────────────────────────────────────────────
# KUBERNETES - Uses streaming refresh
# ─────────────────────────────────────────────────────────
LOGVIEWER_CONFIG="integration/config.yaml" ./build/logviewer query log \
  -i payment-processor-pod --last 5m --refresh

# ─────────────────────────────────────────────────────────
# DOCKER - Uses streaming refresh
# ─────────────────────────────────────────────────────────
LOGVIEWER_CONFIG="integration/config.yaml" ./build/logviewer query log \
  -i docker-test --refresh

# ─────────────────────────────────────────────────────────
# SSH - Uses streaming refresh (with HL support)
# ─────────────────────────────────────────────────────────
LOGVIEWER_CONFIG="integration/config.yaml:integration/config.hl.yaml" \
  ./build/logviewer query log -i ssh-json --last 5m --refresh

# ─────────────────────────────────────────────────────────
# LOCAL FILES - Uses streaming refresh
# ─────────────────────────────────────────────────────────
LOGVIEWER_CONFIG="integration/config.yaml" ./build/logviewer query log \
  -i local-json --refresh
```

### Test Follow with Filters

```bash
# Follow only ERROR logs in Splunk
LOGVIEWER_CONFIG="integration/config.yaml" ./build/logviewer query log \
  -i payment-service -f level=ERROR --refresh-rate 3s

# Follow a specific trace across backends
LOGVIEWER_CONFIG="integration/config.yaml" ./build/logviewer query log \
  -i payment-service -i order-service \
  -f trace_id=abc-123 --refresh-rate 5s
```

---

## Running Integration Tests

### Run All Tests

```bash
make integration/tests
# or
./integration/test-all.sh all
```

### Run Specific Test Suites

```bash
# Basic log querying
./integration/test-all.sh log

# Field extraction
./integration/test-all.sh field

# Field value enumeration
./integration/test-all.sh values

# Complex nested filters
./integration/test-all.sh filters

# Native query syntax (SPL, Lucene)
./integration/test-all.sh native

# HL-compatible queries
./integration/test-all.sh hl
```

### Individual Test Scripts

```bash
./integration/test-query-log.sh        # Basic queries
./integration/test-query-field.sh      # Field filtering
./integration/test-query-values.sh     # Value enumeration
./integration/test-recursive-filters.sh # Complex filters
./integration/test-native-queries.sh   # Native syntax
./integration/test-hl-queries.sh       # HL syntax
./integration/test-ssh-hl.sh           # SSH with HL
./integration/test-hl-vs-native.sh     # Engine comparison
```

### Run Benchmarks

```bash
./integration/benchmark/run-benchmark.sh \
  --sizes "1000,10000" \
  --filters "simple,complex" \
  --iterations 5
```

---

## Configuration Contexts

### Available Contexts (in `config.yaml`)

| Context | Backend | Index/Source | Description |
|---------|---------|--------------|-------------|
| `payment-service` | Splunk | main | Payment processor logs |
| `order-service` | OpenSearch | orders | Order service logs |
| `api-gateway` | OpenSearch | api-logs | API gateway logs |
| `payment-processor-pod` | K8s | - | Pod logs in K3s |
| `cloudwatch-test` | CloudWatch | /aws/test | LocalStack CloudWatch |
| `ssh-json` | SSH | ~/logs/app.log | Remote JSON logs |
| `local-json` | Local | logs/app.log | Local JSON logs |

### Using Multiple Configs

```bash
# Combine multiple config files
LOGVIEWER_CONFIG="integration/config.yaml:integration/config.extra.yaml"

# With HL support
LOGVIEWER_CONFIG="integration/config.yaml:integration/config.hl.yaml"
```

### Config File Locations

```
integration/
├── config.yaml           # Main configuration
├── config.extra.yaml     # Additional contexts
├── config.hl.yaml        # HL-specific configs
└── config.hl-benchmark.yaml  # Benchmark configs
```

---

## Troubleshooting

### Splunk Issues

```bash
# Check if Splunk is healthy
curl -s http://localhost:8089/services/server/health

# Get HEC token
cat integration/splunk/.hec_token

# Test HEC endpoint
curl -k "http://localhost:8088/services/collector/health"

# View Splunk container logs
docker logs splunk
```

### OpenSearch Issues

```bash
# Check cluster health
curl -s http://localhost:9200/_cluster/health | jq .

# List indices
curl -s http://localhost:9200/_cat/indices?v

# Check index exists
curl -s http://localhost:9200/orders/_count

# View container logs
docker logs opensearch
```

### Kubernetes Issues

```bash
# Check nodes
kubectl --kubeconfig=integration/k8s/k3s.yaml get nodes

# Check pods
kubectl --kubeconfig=integration/k8s/k3s.yaml get pods -A

# View pod logs
kubectl --kubeconfig=integration/k8s/k3s.yaml logs -l app=payment-processor -f

# Describe failing pod
kubectl --kubeconfig=integration/k8s/k3s.yaml describe pod <pod-name>

# Reconfigure kubeconfig
./integration/k8s/configure-kubeconfig.sh
```

### SSH Issues

```bash
# Test SSH connection
ssh -v -i integration/ssh/id_rsa -p 2222 testuser@localhost "echo OK"

# Check SSH server logs
docker logs ssh-server

# Regenerate logs on server
./integration/ssh/generate-logs.sh
```

### Docker Issues

```bash
# List containers
docker-compose -f integration/docker-compose.yml ps

# Restart a service
docker-compose -f integration/docker-compose.yml restart splunk

# View all logs
docker-compose -f integration/docker-compose.yml logs -f

# Full reset
make integration/stop
docker-compose -f integration/docker-compose.yml down -v
make integration/start
```

### Log Generator Issues

```bash
# Check if image exists in K3s
docker exec k3s-server ctr images ls | grep log-generator

# Manually import image
docker save log-generator:latest | docker exec -i k3s-server ctr images import -

# Rebuild and redeploy
make integration/deploy-simulation

# Check pod status
kubectl --kubeconfig=integration/k8s/k3s.yaml get pods -l app=payment-processor
kubectl --kubeconfig=integration/k8s/k3s.yaml logs -l app=payment-processor
```

---

## MCP Agent Tests

LLM-driven integration tests using local Ollama.

### Prerequisites

```bash
# Install Ollama
# https://ollama.ai/download

# Pull a model
ollama pull llama3.1   # Recommended
ollama pull mistral    # Alternative

# Start Ollama
ollama serve
```

### Run MCP Tests

```bash
# All MCP tests
go test ./cmd/... -run TestMCPAgent -v

# Specific scenarios
go test ./cmd/... -run TestMCPAgent_DiscoveryWorkflow -v
go test ./cmd/... -run TestMCPAgent_ErrorInvestigation -v
go test ./cmd/... -run TestMCPAgent_MultiStepReasoning -v
```

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `OLLAMA_HOST` | `http://localhost:11434` | Ollama API endpoint |
| `OLLAMA_MODEL` | `llama3.1` | Model for testing |

---

## Directory Structure

```
integration/
├── config.yaml              # Main config
├── config.extra.yaml        # Extra contexts
├── config.hl.yaml           # HL configs
├── docker-compose.yml       # Infrastructure
├── docker-compose-log-generator.yml
│
├── benchmark/               # Performance tests
│   ├── run-benchmark.sh
│   └── generate-logs.go
│
├── k8s/                     # Kubernetes
│   ├── app.yaml             # Simulator deployment
│   ├── k3s.yaml             # Kubeconfig
│   └── configure-kubeconfig.sh
│
├── log-generator/           # Transaction simulator
│   ├── main.go
│   └── Dockerfile
│
├── logs/                    # Sample log files
│   └── app.log
│
├── opensearch/              # OpenSearch scripts
│   └── send-logs.sh
│
├── splunk/                  # Splunk scripts
│   ├── send-logs.sh
│   └── .hec_token
│
├── ssh/                     # SSH testing
│   ├── id_rsa / id_rsa.pub
│   ├── generate-logs.sh
│   └── upload-log.sh
│
└── test-*.sh                # Test scripts
```

---

## Cleanup

```bash
# Stop all services
make integration/stop

# Full cleanup (remove volumes)
docker-compose -f integration/docker-compose.yml down -v

# Remove generated logs
rm -f integration/logs/*.log
```
