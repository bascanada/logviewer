#!/bin/bash

# This script automates the process of running integration tests against a local Splunk instance.

# --- Configuration ---
SPLUNK_HOST="localhost"
SPLUNK_API_PORT="8089"
SPLUNK_HEC_PORT="8088"
SPLUNK_USER="admin"
SPLUNK_PASSWORD="password"

# --- Helper Functions ---
function cleanup() {
  echo "Stopping Splunk..."
  docker-compose down
}

# --- Main Script ---

# Start Splunk
echo "Starting Splunk..."
docker-compose up -d

# Wait for Splunk to be ready
echo "Waiting for Splunk to be ready..."
while ! curl -s -k -u "_SPLUNK_USER_:_SPLUNK_PASSWORD_" "https://_SPLUNK_HOST_:_SPLUNK_API_PORT_/services/server/info" > /dev/null; do
  sleep 5
done
echo "Splunk is ready."

# Enable HEC
echo "Enabling HEC..."
curl -k -u "_SPLUNK_USER_:_SPLUNK_PASSWORD_" "https://_SPLUNK_HOST_:_SPLUNK_API_PORT_/services/data/inputs/http?output_mode=json" -d "disabled=0" > /dev/null

# Create HEC token
echo "Creating HEC token..."
HEC_TOKEN=$(curl -s -k -u "_SPLUNK_USER_:_SPLUNK_PASSWORD_" "https://_SPLUNK_HOST_:_SPLUNK_API_PORT_/services/data/inputs/http/http?output_mode=json" -d "name=my-test-token" -d "description=Token for integration tests" | grep -o '"token": "[^"']\*' | grep -o '[^"']*')

if [ -z "$HEC_TOKEN" ]; then
  echo "Failed to create HEC token."
  cleanup
  exit 1
fi

echo "HEC token created: $HEC_TOKEN"

# Update send-logs.sh with the new token
sed -i.bak "s/HEC_TOKEN=".*/HEC_TOKEN=\"$HEC_TOKEN\"/" send-logs.sh

# Send sample data
./send-logs.sh

# Run integration tests
# (Replace this with your actual test command)
echo "Running integration tests..."
(cd ../.. && go test ./pkg/log/impl/splunk/logclient)

# Clean up
cleanup
