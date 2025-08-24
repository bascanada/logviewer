#!/bin/bash
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
KEY_PATH="${SCRIPT_DIR}/id_rsa"
PUB_PATH="${KEY_PATH}.pub"
AUTH_KEYS="${SCRIPT_DIR}/authorized_keys"

if [[ -f "$KEY_PATH" && -f "$PUB_PATH" ]]; then
  echo "SSH key pair already exists at $KEY_PATH"
else
  echo "Generating SSH key pair..."
  ssh-keygen -t rsa -b 4096 -N "" -f "$KEY_PATH" >/dev/null
fi

if [[ ! -f "$AUTH_KEYS" ]]; then
  echo "Creating authorized_keys from public key"
  cp "$PUB_PATH" "$AUTH_KEYS"
else
  # ensure public key present only once
  grep -q "$(cat "$PUB_PATH")" "$AUTH_KEYS" || cat "$PUB_PATH" >> "$AUTH_KEYS"
fi

chmod 600 "$KEY_PATH" || true
chmod 644 "$PUB_PATH" "$AUTH_KEYS" || true

echo "SSH keys ready."
