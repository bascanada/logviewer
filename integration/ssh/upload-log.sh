#!/bin/bash
set -euo pipefail

SSH_HOST="${SSH_HOST:-localhost}"
SSH_PORT="${SSH_PORT:-2222}"
SSH_USER="${SSH_USER:-testuser}"
LOG_FILE="${LOG_FILE:-../logs/app.log}"
REMOTE_PATH="/home/testuser/app.log"

echo "Uploading '${LOG_FILE}' to ssh-server..."

scp -P "${SSH_PORT}" -o "StrictHostKeyChecking=no" -o "UserKnownHostsFile=/dev/null" -i "./ssh/id_rsa" "${LOG_FILE}" "${SSH_USER}@${SSH_HOST}:${REMOTE_PATH}"

echo "Log upload complete."
