#!/bin/bash

# This script sends sample logs to a local Splunk instance using the HTTP Event Collector (HEC).

# --- Configuration ---
SPLUNK_HOST="localhost"
SPLUNK_PORT="8088"
HEC_TOKEN="" # Replace with your HEC token

# --- Log Data ---
LOG_DATA=(
  "{\"event\": \"User logged in\", \"user\": \"alice\", \"status\": \"success\"}"
  "{\"event\": \"User failed to log in\", \"user\": \"bob\", \"status\": \"failure\"}"
  "{\"event\": \"Item added to cart\", \"user\": \"alice\", \"item\": \"apple\"}"
  "{\"event\": \"Checkout completed\", \"user\": \"alice\", \"status\": \"success\"}"
)

# --- Send Logs ---
for log in "${LOG_DATA[@]}"; do
  echo "Sending log: $log"
  curl -k "https://_SPLUNK_HOST_:_SPLUNK_PORT_/services/collector" \
    -H "Authorization: Splunk _HEC_TOKEN_" \
    -d "$log"
  echo ""
done
