#!/bin/bash
set -euo pipefail

LOG_GROUP_NAME="my-app-logs"
LOG_STREAM_NAME="my-app-instance-1"
LOG_FILE="${LOG_FILE:-../logs/app.log}"
AWS_ENDPOINT_URL="http://localhost:4566"

echo "Configuring AWS CLI for LocalStack..."
aws configure set aws_access_key_id "test"
aws configure set aws_secret_access_key "test"
aws configure set region "us-east-1"

echo "Creating CloudWatch log group..."
aws --endpoint-url=${AWS_ENDPOINT_URL} logs create-log-group --log-group-name ${LOG_GROUP_NAME} || echo "Log group already exists."

echo "Creating CloudWatch log stream..."
aws --endpoint-url=${AWS_ENDPOINT_URL} logs create-log-stream --log-group-name ${LOG_GROUP_NAME} --log-stream-name ${LOG_STREAM_NAME} || echo "Log stream already exists."

echo "Sending logs to CloudWatch..."

if ! command -v jq >/dev/null 2>&1; then
  echo "Error: jq is required for JSON encoding log events." >&2
  exit 1
fi

events_file=$(mktemp)
trap 'rm -f "$events_file"' EXIT

echo '[' > "$events_file"
first=1
while IFS= read -r line; do
  [[ -z "$line" ]] && continue
  ts=$(date +%s000)
  # Use jq for safe JSON encoding of the message
  evt=$(jq -cn --arg ts "$ts" --arg msg "$line" '{timestamp: ($ts|tonumber), message: $msg}')
  if [[ $first -eq 1 ]]; then
    printf '%s' "$evt" >> "$events_file"
    first=0
  else
    printf ',%s' "$evt" >> "$events_file"
  fi
done < "${LOG_FILE}"
echo ']' >> "$events_file"

LOG_EVENTS=$(cat "$events_file")

put_logs() {
  if [[ -n "$1" ]]; then
    aws --endpoint-url=${AWS_ENDPOINT_URL} logs put-log-events \
      --log-group-name ${LOG_GROUP_NAME} \
      --log-stream-name ${LOG_STREAM_NAME} \
      --sequence-token "$1" \
      --log-events "$LOG_EVENTS"
  else
    aws --endpoint-url=${AWS_ENDPOINT_URL} logs put-log-events \
      --log-group-name ${LOG_GROUP_NAME} \
      --log-stream-name ${LOG_STREAM_NAME} \
      --log-events "$LOG_EVENTS"
  fi
}

if ! out=$(put_logs "" 2>&1); then
  # Handle invalid sequence token (if stream already had events)
  if echo "$out" | grep -q 'The next expected sequenceToken is'; then
    token=$(echo "$out" | sed -n 's/.*The next expected sequenceToken is: \([A-Za-z0-9]\+\).*/\1/p')
    if [[ -n "$token" ]]; then
      echo "Retrying with sequence token $token" >&2
      put_logs "$token" || { echo "$out" >&2; exit 1; }
    else
      echo "$out" >&2
      exit 1
    fi
  else
    echo "$out" >&2
    exit 1
  fi
fi

echo "Log sending complete."
