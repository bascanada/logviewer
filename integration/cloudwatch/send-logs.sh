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

LOG_EVENTS=""
while IFS= read -r line; do
  if [[ -z "$line" ]]; then
    continue
  fi
  TIMESTAMP=$(date +%s000)
  # Escape quotes for JSON
  ESCAPED_LINE=$(echo "$line" | sed 's/"/\\"/g')
  LOG_EVENTS+=$(printf '{"timestamp": %s, "message": "%s"},' "$TIMESTAMP" "$ESCAPED_LINE")
done < "${LOG_FILE}"

# Remove trailing comma
LOG_EVENTS="[${LOG_EVENTS%,}]"

aws --endpoint-url=${AWS_ENDPOINT_URL} logs put-log-events \
  --log-group-name ${LOG_GROUP_NAME} \
  --log-stream-name ${LOG_STREAM_NAME} \
  --log-events "${LOG_EVENTS}"

echo "Log sending complete."
