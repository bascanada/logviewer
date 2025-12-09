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

## Development

To modify the simulator:

1. Edit `integration/log-generator/main.go`
2. Rebuild and redeploy: `make integration/deploy-simulation`

Key functions:
- `simulateTransaction()`: Core transaction logic
- `startLoadGenerator()`: Background traffic generator
- `sendToSplunk()`, `sendToOpenSearch()`: Log shipping
- `logToDynamo()`: Metadata storage
