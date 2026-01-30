#!/bin/bash
set -euo pipefail

# This script sends sample logs to a local Splunk instance using the HTTP Event Collector (HEC).
# It can be executed from the project root; paths are resolved relative to the script location.

# Resolve script directory so paths are relative to this script, not the current working directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="${SCRIPT_DIR%/*}"

# --- Configuration (can be overridden with environment variables) ---
SPLUNK_HOST="${SPLUNK_HOST:-localhost}"
SPLUNK_PORT="${SPLUNK_PORT:-8088}"
HEC_TOKEN_FILE="${HEC_TOKEN_FILE:-${SCRIPT_DIR}/.hec_token}"
LOG_DIR="${LOG_DIR:-${SCRIPT_DIR}/../logs}"
HEC_INDEX="${HEC_INDEX:-main}"

# --- Validate HEC Token ---
if [[ ! -f "${HEC_TOKEN_FILE}" ]]; then
    echo "Error: HEC token file not found at '${HEC_TOKEN_FILE}'." >&2
    echo "Please create the file and paste your HEC token in it." >&2
    exit 1
fi
HEC_TOKEN="$(cat "${HEC_TOKEN_FILE}")"
if [[ -z "${HEC_TOKEN}" ]]; then
    echo "Error: HEC token file is empty." >&2
    exit 1
fi

# --- Validate Log Directory ---
if [[ ! -d "${LOG_DIR}" ]]; then
    echo "Error: Log directory not found at '${LOG_DIR}'" >&2
    exit 1
fi

echo "Sending logs from '${LOG_DIR}' to Splunk index '${HEC_INDEX}'..."

# --- Send Logs ---
# Find all .log files, then read each line and send it to Splunk
find "${LOG_DIR}" -type f -name "*.log" -print0 | while IFS= read -r -d $'\0' logfile; do
    echo "Processing log file: ${logfile}"
    while IFS= read -r line; do
        # Skip empty lines
        if [[ -z "$line" ]]; then
            continue
        fi

        # Format the log entry as a JSON object for HEC.
        # Escape double quotes in the log line to ensure valid JSON.
        json_payload="{\"event\": \"$(echo "$line" | sed 's/"/\\"/g')\", \"index\": \"${HEC_INDEX}\"}"

        # Send the data to Splunk HEC
        curl -s -k "https://${SPLUNK_HOST}:${SPLUNK_PORT}/services/collector" \
             -H "Authorization: Splunk ${HEC_TOKEN}" \
             -d "${json_payload}"
    done < "${logfile}"
done

echo "Log sending complete."