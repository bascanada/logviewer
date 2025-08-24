#!/bin/bash
# Upload a log file to the integration SSH server.
# Fixes: incorrect relative path to private key (was ./ssh/id_rsa while already in integration/ssh).
set -euo pipefail

SSH_HOST="${SSH_HOST:-localhost}"
SSH_PORT="${SSH_PORT:-2222}"
SSH_USER="${SSH_USER:-testuser}"
LOG_FILE="${LOG_FILE:-../logs/app.log}"
REMOTE_PATH="app.log"

# Resolve script directory to allow calling from any working directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Allow override via SSH_KEY env var; default to id_rsa next to this script
SSH_KEY="${SSH_KEY:-${SCRIPT_DIR}/id_rsa}"

if [[ ! -f "${SSH_KEY}" ]]; then
	echo "ERROR: SSH private key not found at ${SSH_KEY}" >&2
	exit 1
fi

# Ensure private key has correct permissions (silence errors on non-Unix FS)
chmod 600 "${SSH_KEY}" 2>/dev/null || true

if [[ ! -f "${LOG_FILE}" ]]; then
	echo "ERROR: Log file '${LOG_FILE}' not found." >&2
	exit 2
fi

echo "Uploading '${LOG_FILE}' to ssh-server (${SSH_USER}@${SSH_HOST}:${SSH_PORT}) using key ${SSH_KEY}..."

set -x
scp -P "${SSH_PORT}" \
		-o "StrictHostKeyChecking=no" \
		-o "UserKnownHostsFile=/dev/null" \
		-i "${SSH_KEY}" \
		"${LOG_FILE}" "${SSH_USER}@${SSH_HOST}:${REMOTE_PATH}"
set +x

echo "Log upload complete."
