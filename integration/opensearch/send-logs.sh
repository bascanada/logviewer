#!/bin/bash
set -euo pipefail

# Ultra-simple OpenSearch bulk ingestion script (clean version).
# Escapes quotes, backslashes, tabs, and control characters so JSON stays valid.

OPENSEARCH_HOST="${OPENSEARCH_HOST:-localhost}"
OPENSEARCH_PORT="${OPENSEARCH_PORT:-9200}"
INDEX_NAME="${INDEX_NAME:-app-logs}"   # single static index
LOG_FILE="${LOG_FILE:-../logs/app.log}"
VERBOSE="${VERBOSE:-0}"
[[ "$VERBOSE" == "1" ]] && set -x

if [[ ! -f "$LOG_FILE" ]]; then
  echo "ERROR: Log file '$LOG_FILE' not found" >&2
  exit 1
fi

echo "Waiting for OpenSearch (${OPENSEARCH_HOST}:${OPENSEARCH_PORT})..."
until curl -s "${OPENSEARCH_HOST}:${OPENSEARCH_PORT}" > /dev/null; do sleep 1; done
echo "OpenSearch reachable."

# Create index if missing
if ! curl -s -o /dev/null -w '%{http_code}' "${OPENSEARCH_HOST}:${OPENSEARCH_PORT}/${INDEX_NAME}" | grep -q '^200$'; then
  echo "Creating index '${INDEX_NAME}'..."
  curl -s -X PUT "${OPENSEARCH_HOST}:${OPENSEARCH_PORT}/${INDEX_NAME}" -H 'Content-Type: application/json' -d '{
    "settings": {"number_of_shards":1, "number_of_replicas":0},
    "mappings": {"properties": {"@timestamp": {"type": "date"}, "message": {"type": "text"}}}
  }' > /dev/null
else
  echo "Index '${INDEX_NAME}' already exists."
fi

TMP_BULK=$(mktemp)
LINE_COUNT=0
while IFS= read -r line || [[ -n "$line" ]]; do
  [[ -z "$line" ]] && continue

  # Escape JSON special chars and control chars.
  esc=$line
  # Backslash first
  esc=${esc//\\/\\\\}
  # Quote
  esc=${esc//"/\\"}
  # Tab -> \t (code 9 appears in stack traces indentation)
  esc=${esc//$'\t'/\\t}
  # Carriage return
  esc=${esc//$'\r'/}
  # Other control chars (0x00-0x08,0x0B,0x0C,0x0E-0x1F) -> space
  esc=$(printf '%s' "$esc" | perl -pe 's/[\x00-\x08\x0B\x0C\x0E-\x1F]/ /g')

  printf '{"index":{"_index":"%s"}}\n' "$INDEX_NAME" >> "$TMP_BULK"
  # Portable RFC3339 / RFC3339Nano-ish timestamp: prefer gdate (GNU) for millis, fallback to seconds.
  if command -v gdate >/dev/null 2>&1; then
    TS=$(gdate -u +%Y-%m-%dT%H:%M:%S.%3NZ)
  else
    # macOS / BSD date lacks %N; we emit second precision.
    TS=$(date -u +%Y-%m-%dT%H:%M:%SZ)
  fi
  printf '{"@timestamp":"%s","message":"%s"}\n' "$TS" "$esc" >> "$TMP_BULK"
  LINE_COUNT=$((LINE_COUNT+1))
done < "$LOG_FILE"

if [[ $LINE_COUNT -eq 0 ]]; then
  echo "No lines to ingest (all empty)."; rm -f "$TMP_BULK"; exit 0
fi
echo "Built bulk file with $LINE_COUNT actions at $TMP_BULK"

RESPONSE=$(curl -s -H 'Content-Type: application/x-ndjson' \
  --data-binary @"$TMP_BULK" \
  -X POST "${OPENSEARCH_HOST}:${OPENSEARCH_PORT}/_bulk?refresh=wait_for")

# Basic validation
if command -v jq >/dev/null 2>&1; then
  ERRORS=$(printf '%s' "$RESPONSE" | jq -r '.errors') || ERRORS="true"
  if [[ "$ERRORS" == "true" ]]; then
    echo "Bulk reported errors:" >&2
    printf '%s' "$RESPONSE" | jq '.items[] | select(.index.status>=300) | .index.error' | head -n 20 >&2
  fi
  CREATED=$(printf '%s' "$RESPONSE" | jq '[.items[].index.status] | map(select(.==201 or .==200)) | length')
else
  # Fallback simple parsing
  echo "$RESPONSE" | grep -q '"errors":true' && echo "Bulk reported errors (raw):" >&2 && echo "$RESPONSE" | head -n 40 >&2
  CREATED=$(echo "$RESPONSE" | grep -E '"status":201|"status":200' | wc -l | tr -d ' ' || echo 0)
fi

echo "Indexed $CREATED / $LINE_COUNT documents into ${INDEX_NAME}."
if [[ "$CREATED" -ne "$LINE_COUNT" ]]; then
  echo "WARNING: Mismatch between lines ($LINE_COUNT) and successes ($CREATED). Showing first 10 item entries:" >&2
  if command -v jq >/dev/null 2>&1; then
    printf '%s' "$RESPONSE" | jq '.items[0:10]' >&2
  else
    echo "$RESPONSE" | head -n 60 >&2
  fi
fi

echo "Done."
rm -f "$TMP_BULK"
