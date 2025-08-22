#!/bin/bash

# This script sends sample logs to a local Splunk instance using the HTTP Event Collector (HEC).

# --- Configuration ---
SPLUNK_HOST="localhost"
SPLUNK_PORT="8088"
HEC_TOKEN="" # This will be replaced by the run-integration-tests.sh script

# --- Send Logs ---
LOG_DIR="../logs"

for file in $LOG_DIR/*; do
  echo "Sending logs from $file"
  while IFS= read -r line; do
    echo "Sending log: $line"
    curl -k "https://_SPLUNK_HOST_:_SPLUNK_PORT_/services/collector"
      -H "Authorization: Splunk _HEC_TOKEN_"
      -d "$line"
    echo ""
  done < "$file"
done
