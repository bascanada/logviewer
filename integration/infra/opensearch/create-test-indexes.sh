#!/bin/bash

# Creates test indexes in OpenSearch for E2E tests
# This ensures indexes exist before seeding data

set -e

OPENSEARCH_HOST="${OPENSEARCH_HOST:-localhost}"
OPENSEARCH_PORT="${OPENSEARCH_PORT:-9200}"

# Test indexes needed for E2E tests
TEST_INDEXES=(
  "test-e2e-orders"
  "test-e2e-mixed"
  "test-e2e-slow"
)

echo "Creating OpenSearch test indexes..."

for index in "${TEST_INDEXES[@]}"; do
  echo "  Creating index: $index"

  # Create index with mapping for timestamp field
  response=$(curl -s -X PUT "http://${OPENSEARCH_HOST}:${OPENSEARCH_PORT}/${index}" \
    -H 'Content-Type: application/json' \
    -d '{
      "mappings": {
        "properties": {
          "@timestamp": {
            "type": "date"
          },
          "timestamp": {
            "type": "date"
          },
          "level": {
            "type": "keyword"
          },
          "message": {
            "type": "text"
          },
          "app": {
            "type": "keyword"
          },
          "trace_id": {
            "type": "keyword"
          },
          "latency_ms": {
            "type": "long"
          },
          "run_id": {
            "type": "keyword"
          }
        }
      }
    }' 2>&1)

  # Check response
  if echo "$response" | grep -qi '"acknowledged":true'; then
    echo "    ✓ Index $index created"
  elif echo "$response" | grep -qi "resource_already_exists"; then
    echo "    ✓ Index $index already exists"
  else
    echo "    ⚠️  Response: $response"
  fi
done

echo "✓ All OpenSearch test indexes are ready"
