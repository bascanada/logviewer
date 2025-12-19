# Integration Test Environment - Transaction Simulator

This directory contains a realistic transaction simulator that demonstrates LogViewer's capabilities across multiple log sources.

## Overview

The transaction simulator mimics a microservices architecture with the following components:

- **Frontend Service** (Nginx-style logs → K8s stdout)
- **Order Service** (JSON logs → OpenSearch)
- **Payment Service** (Structured logs → Splunk)
- **Database** (Error logs → K8s stdout)
- **LocalStack DynamoDB** (Transaction metadata storage)

The simulator automatically generates realistic traffic with:
- 10% payment failures (timeout errors)
- 5% database deadlocks
- Variable latency (0-500ms normal, occasional >1s slow requests)
- Distributed tracing via `trace_id` across all services

## Quick Start

### 1. Start Infrastructure

```bash
make integration/start
```

This starts:
- Splunk Enterprise (port 8000, 8088)
- OpenSearch + Dashboards (ports 9200, 5601)
- K3s Kubernetes cluster
- LocalStack (DynamoDB on port 4566)
- SSH server for log testing

### 2. Deploy Transaction Simulator

```bash
make integration/deploy-simulation
```

This will:
1. Build the Go application as a Docker image
2. Import the image into the K3s cluster
3. Deploy the simulator as a Kubernetes pod
4. Automatically start generating transaction logs

The simulator runs on port 8081 inside the cluster and generates background traffic automatically.

### 3. Query Logs with LogViewer

#### View all logs across all sources (last 5 minutes)
```bash
logviewer query log -c ./config.yaml -i k3s-payment -i splunk-prod -i opensearch-prod --last 5m
```

#### Filter by a specific trace ID to follow a transaction
```bash
# Pick a trace_id from the output above
logviewer query log -c ./config.yaml -i k3s-payment -i splunk-prod -i opensearch-prod \
  --field trace_id=abc-def-ghi-123 --last 10m
```

#### Find all payment errors
```bash
logviewer query log -c ./config.yaml -i splunk-prod \
  --field level=ERROR --field app=payment-service --last 5m
```

#### Find slow requests
```bash
logviewer query log -c ./config.yaml -i opensearch-prod \
  --field app=api-gateway --field level=WARN --last 5m
```

## Manual Testing

You can also trigger individual transactions via HTTP:

```bash
# Get the service IP (if testing from host machine, use kubectl port-forward)
kubectl --kubeconfig=integration/k8s/k3s.yaml port-forward svc/payment-processor 8081:80

# Trigger a transaction
curl http://localhost:8081/checkout
```

## Configuration

The simulator uses these environment variables (automatically configured in Kubernetes):

- `SPLUNK_HEC_URL`: Splunk HTTP Event Collector endpoint
- `SPLUNK_HEC_TOKEN`: HEC authentication token
- `OPENSEARCH_URL`: OpenSearch index endpoint
- `LOCALSTACK_URL`: LocalStack endpoint for DynamoDB

## Simulated Scenarios

### Normal Transaction Flow
1. Frontend receives checkout request → K8s logs
2. Order service processes order → OpenSearch
3. Payment service authorizes payment → Splunk
4. Transaction complete (~100-500ms)

### Slow Checkout Incident (10% of requests)
1. Frontend receives checkout request → K8s logs
2. Order service processes order → OpenSearch
3. Payment service TIMEOUT (30s) → Splunk ERROR
4. API gateway logs slow request warning → OpenSearch
5. DynamoDB records FAILED status

### Database Deadlock (5% of requests)
1. Normal transaction flow starts
2. Database deadlock occurs → K8s ERROR logs
3. Transaction may succeed or fail depending on retry logic

## Architecture

```
┌─────────────────────────────────────────────┐
│          Kubernetes (K3s)                   │
│  ┌───────────────────────────────────────┐ │
│  │  payment-processor Pod                │ │
│  │  (Transaction Simulator)              │ │
│  │                                       │ │
│  │  - Generates traffic automatically   │ │
│  │  - Distributes logs to:              │ │
│  │    • stdout (frontend/db logs)       │ │
│  │    • Splunk (payment logs)           │ │
│  │    • OpenSearch (order logs)         │ │
│  │    • DynamoDB (transaction metadata) │ │
│  └───────────────────────────────────────┘ │
└─────────────────────────────────────────────┘
            │           │           │
            ▼           ▼           ▼
    ┌──────────┐ ┌──────────┐ ┌──────────┐
    │  K8s     │ │  Splunk  │ │OpenSearch│
    │  Logs    │ │  HEC     │ │  Index   │
    └──────────┘ └──────────┘ └──────────┘
                      │
                      ▼
                ┌──────────┐
                │LocalStack│
                │ DynamoDB │
                └──────────┘
```

## Troubleshooting

### Simulator not generating logs
```bash
# Check pod status
kubectl --kubeconfig=integration/k8s/k3s.yaml get pods

# View pod logs
kubectl --kubeconfig=integration/k8s/k3s.yaml logs -l app=payment-processor -f
```

### Can't connect to Splunk/OpenSearch
```bash
# Verify services are running
docker ps

# Check if services are accessible from K3s network
docker exec k3s-server wget -O- http://splunk:8088/services/collector/health
docker exec k3s-server wget -O- http://opensearch:9200
```

### Image not found in K3s
```bash
# Manually import image
docker save log-generator:latest | docker exec -i k3s-server ctr images import -

# Verify image is available
docker exec k3s-server ctr images ls | grep log-generator
```

## Cleanup

```bash
# Stop simulator
kubectl --kubeconfig=integration/k8s/k3s.yaml delete -f integration/k8s/app.yaml

# Stop all infrastructure
make integration/stop
```

## MCP Agent Integration Tests

The logviewer includes LLM-driven integration tests that use a local Ollama instance to test the MCP server with an actual AI agent. These tests validate that an LLM can effectively discover contexts, query logs, and investigate issues.

### Prerequisites

1. **Install Ollama**: https://ollama.ai/download
2. **Pull a model with tool-calling support**:
   ```bash
   ollama pull mistral  # Recommended for tool calling
   # or
   ollama pull llama3
   ```
3. **Start Ollama**:
   ```bash
   ollama serve
   ```

### Running LLM Integration Tests

```bash
# Run all MCP agent tests (requires Ollama running)
go test ./cmd/... -run TestMCPAgent -v

# Run specific test scenarios
go test ./cmd/... -run TestMCPAgent_DiscoveryWorkflow -v
go test ./cmd/... -run TestMCPAgent_ErrorInvestigation -v
go test ./cmd/... -run TestMCPAgent_MultiStepReasoning -v
go test ./cmd/... -run TestMCPAgent_ContextNotFound -v
go test ./cmd/... -run TestMCPAgent_DynamicPrompts -v

# Run benchmarks
go test ./cmd/... -run=^$ -bench=BenchmarkMCPToolCall -v
```

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `OLLAMA_HOST` | `http://localhost:11434` | Ollama API endpoint |
| `OLLAMA_MODEL` | `llama3.1` | Model to use for testing |

### Recommended Models

Tool-calling capability varies by model. Recommended models in order of reliability:

1. **llama3.1** (8B) - Good balance of speed and tool-calling
2. **qwen2.5** - Excellent tool-calling support
3. **mistral** - Works but sometimes describes instead of calling tools
4. **llama3.1:70b** - Best accuracy but requires more resources

```bash
# Pull recommended model
ollama pull llama3.1

# Or for better accuracy (requires ~40GB RAM)
ollama pull llama3.1:70b
```

### Test Scenarios

1. **Discovery Workflow**: Agent lists available contexts and fields
2. **Error Investigation**: Agent finds ERROR logs using appropriate filters
3. **Multi-Step Reasoning**: Complex queries requiring multiple tool calls
4. **Context Not Found**: Error handling when context doesn't exist
5. **Dynamic Prompts**: Validates context-specific prompt generation

### Test Architecture

```
┌─────────────────────────────────────────────────────┐
│                   Test Harness                      │
│  ┌───────────────────┐    ┌─────────────────────┐  │
│  │   OllamaClient    │    │   MCP In-Process    │  │
│  │  (LLM Interface)  │◄──►│      Client         │  │
│  └───────────────────┘    └─────────────────────┘  │
│           │                         │               │
│           ▼                         ▼               │
│  ┌───────────────────┐    ┌─────────────────────┐  │
│  │  Local Ollama     │    │   MCP Server        │  │
│  │  (llama3.1)       │    │   (logviewer)       │  │
│  └───────────────────┘    └─────────────────────┘  │
└─────────────────────────────────────────────────────┘
```

The test harness:
1. Starts an in-process MCP server
2. Connects to local Ollama
3. Sends prompts to the LLM with MCP tools available
4. Executes tool calls and feeds results back to LLM
5. Validates the agent's tool usage and final response

## Development

To modify the simulator:

1. Edit `integration/log-generator/main.go`
2. Rebuild and redeploy: `make integration/deploy-simulation`

Key functions:
- `simulateTransaction()`: Core transaction logic
- `startLoadGenerator()`: Background traffic generator
- `sendToSplunk()`, `sendToOpenSearch()`: Log shipping
- `logToDynamo()`: Metadata storage
