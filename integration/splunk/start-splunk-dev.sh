#!/bin/bash

# This script starts a local Splunk instance for development purposes.

# --- Configuration ---
SPLUNK_HOST="localhost"
SPLUNK_API_PORT="8089"
SPLUNK_HEC_PORT="8088"
SPLUNK_USER="admin"
SPLUNK_PASSWORD="password"
HEC_TOKEN_FILE=".hec_token"

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

# Create HEC token if it doesn't exist
if [ ! -f "$HEC_TOKEN_FILE" ]; then
  echo "Creating HEC token..."
  HEC_TOKEN=$(curl -s -k -u "_SPLUNK_USER_:_SPLUNK_PASSWORD_" "https://_SPLUNK_HOST_:_SPLUNK_API_PORT_/services/data/inputs/http/http?output_mode=json" -d "name=my-dev-token" -d "description=Token for development" | grep -o '"token": "[^"]*' | grep -o '[^"]*$')
  if [ -z "$HEC_TOKEN" ]; then
    echo "Failed to create HEC token."
    exit 1
  fi
  echo "$HEC_TOKEN" > "$HEC_TOKEN_FILE"
  echo "HEC token created and saved to $HEC_TOKEN_FILE"
else
  HEC_TOKEN=$(cat "$HEC_TOKEN_FILE")
  echo "Using existing HEC token from $HEC_TOKEN_FILE"
fi

echo ""
echo "Splunk is running and ready for development."
echo ""
echo "  Splunk UI: http://localhost:8000"
_ECHO "  HEC Token: HEC_TOKEN"
echo ""
echo "To stop the Splunk instance, run: docker-compose down"
