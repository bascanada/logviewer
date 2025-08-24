#!/bin/bash
set -euo pipefail

OPENSEARCH_HOST="${OPENSEARCH_HOST:-localhost}"
OPENSEARCH_PORT="${OPENSEARCH_PORT:-9200}"
LOG_FILE="${LOG_FILE:-../logs/app.log}"
INDEX_NAME="app-logs-$(date +%Y.%m.%d)"

echo "Waiting for OpenSearch to be ready..."
until curl -s "${OPENSEARCH_HOST}:${OPENSEARCH_PORT}" > /dev/null; do
    sleep 1
done
echo "OpenSearch is ready."

echo "Sending logs from '${LOG_FILE}' to OpenSearch index '${INDEX_NAME}'..."

# Prepare the log data in bulk format
BULK_DATA=""
while IFS= read -r line; do
  if [[ -z "$line" ]]; then
    continue
  fi
  # Escape double quotes in the log line to ensure valid JSON.
  ESCAPED_LINE=$(echo "$line" | sed 's/"/\\"/g')

  JSON_PAYLOAD=$(printf '{"@timestamp": "%s", "message": "%s"}' "$(date -u +"%Y-%m-%dT%H:%M:%S.%3NZ")" "$ESCAPED_LINE")

  BULK_DATA+=$(printf '{"index": {"_index": "%s"}}\n' "$INDEX_NAME")
  BULK_DATA+=$(printf '%s\n' "$JSON_PAYLOAD")
done < "${LOG_FILE}"

# Send the data to OpenSearch using the bulk API
curl -s -X POST "${OPENSEARCH_HOST}:${OPENSEARCH_PORT}/_bulk" -H 'Content-Type: application/x-ndjson' --data-binary @- <<< "$BULK_DATA" > /dev/null

echo "Log sending complete."
