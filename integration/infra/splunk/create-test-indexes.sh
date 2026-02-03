#!/bin/bash

# Creates test indexes in Splunk for E2E tests
# This ensures indexes exist before seeding data

SPLUNK_HOST="${SPLUNK_HOST:-localhost}"
SPLUNK_API_PORT="${SPLUNK_API_PORT:-8089}"
SPLUNK_USER="${SPLUNK_USER:-admin}"
SPLUNK_PASSWORD="${SPLUNK_PASSWORD:-password}"

# Test indexes needed for E2E tests
TEST_INDEXES=(
  "test-e2e-errors"
  "test-e2e-payments"
  "test-e2e-traces"
)

echo "Creating Splunk test indexes..."

for index in "${TEST_INDEXES[@]}"; do
  echo "  Creating index: $index"

  # First check if index already exists
  check_response=$(curl -s -k -u "${SPLUNK_USER}:${SPLUNK_PASSWORD}" \
    -o /dev/null -w "%{http_code}" \
    "https://${SPLUNK_HOST}:${SPLUNK_API_PORT}/services/data/indexes/${index}" \
    2>&1)

  if [ "$check_response" = "200" ]; then
    echo "    ✓ Index $index already exists"
    continue
  fi

  # Try to create the index (suppress output, check status code)
  create_response=$(curl -s -k -u "${SPLUNK_USER}:${SPLUNK_PASSWORD}" \
    -o /dev/null -w "%{http_code}" \
    "https://${SPLUNK_HOST}:${SPLUNK_API_PORT}/services/data/indexes" \
    -d "name=${index}" \
    -d "datatype=event" \
    -d "maxTotalDataSizeMB=500" \
    -d "frozenTimePeriodInSecs=86400" \
    2>&1)

  # HTTP 201 = created successfully, 409 = already exists, 200 = success
  if [ "$create_response" = "201" ] || [ "$create_response" = "200" ]; then
    echo "    ✓ Index $index created successfully"
  elif [ "$create_response" = "409" ]; then
    echo "    ✓ Index $index already exists"
  else
    echo "    ⚠️  Unexpected response code: $create_response (continuing anyway)"
  fi
done

echo "✓ All Splunk test indexes are ready"
