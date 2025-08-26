#!/bin/bash

# This script starts a local Splunk instance for development purposes.

# --- Configuration ---
SPLUNK_HOST="localhost"
SPLUNK_API_PORT="8089"
SPLUNK_HEC_PORT="8088"
SPLUNK_USER="admin"
SPLUNK_PASSWORD="password"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# default to a token file next to this script so other integration scripts find it
HEC_TOKEN_FILE="${HEC_TOKEN_FILE:-${SCRIPT_DIR}/.hec_token}"

# --- Main Script ---

# Wait for Splunk to be ready
echo "Waiting for Splunk to be ready..."
MAX_ATTEMPTS=60
SLEEP_SECONDS=5
READY=0
TMP_RESP=$(mktemp)
for attempt in $(seq 1 $MAX_ATTEMPTS); do
  status=$(curl -s -k -u "${SPLUNK_USER}:${SPLUNK_PASSWORD}" -o "$TMP_RESP" -w "%{http_code}" "https://${SPLUNK_HOST}:${SPLUNK_API_PORT}/services/server/info" || echo "000")
  if [ "$status" = "200" ]; then
    echo "Splunk responded (HTTP 200) on attempt $attempt."
    READY=1
    break
  fi
  echo "Attempt $attempt/$MAX_ATTEMPTS: HTTP status=$status. Sleeping ${SLEEP_SECONDS}s..."
  # Optionally show a short snippet of the response for debugging on early attempts
  if [ $attempt -le 3 ] && [ -s "$TMP_RESP" ]; then
    echo "Response preview:"
    head -n 20 "$TMP_RESP" | sed -n '1,20p'
  fi
  sleep $SLEEP_SECONDS
done

if [ $READY -ne 1 ]; then
  echo "Splunk did not become ready after $((MAX_ATTEMPTS * SLEEP_SECONDS)) seconds. Last response:"
  [ -s "$TMP_RESP" ] && sed -n '1,200p' "$TMP_RESP" || echo "(no response body)"
  rm -f "$TMP_RESP"
  exit 1
fi
rm -f "$TMP_RESP"
echo "Splunk is ready."


sleep 15

# Enable HEC
echo "Enabling HEC..."
for i in $(seq 1 10); do
  status_code=$(curl -s -k -u "${SPLUNK_USER}:${SPLUNK_PASSWORD}" "https://${SPLUNK_HOST}:${SPLUNK_API_PORT}/services/data/inputs/http?output_mode=json" -d "disabled=0" -o /dev/null -w "%{http_code}")
  if [ "$status_code" = "200" ]; then
    echo "HEC enabled."
    break
  fi
  if [ $i -eq 10 ]; then
    echo "Error: Failed to enable HEC after 10 attempts. Last status: $status_code" >&2
    exit 1
  fi
  echo "Attempt $i/10 to enable HEC failed (status: $status_code), retrying in 5s..."
  sleep 5
done

# Create HEC token if it doesn't exist
if [ ! -f "$HEC_TOKEN_FILE" ]; then
  echo "Creating HEC token..."
  TMP_CREATE_RESP=$(mktemp)
  HEC_JSON=""

  # Try a couple endpoints for creation; capture HTTP status and response body for diagnostics
  create_endpoints=( \
    "https://${SPLUNK_HOST}:${SPLUNK_API_PORT}/servicesNS/nobody/splunk_httpinput/data/inputs/http?output_mode=json" \
    "https://${SPLUNK_HOST}:${SPLUNK_API_PORT}/services/data/inputs/http/http?output_mode=json" \
  )

  for ep in "${create_endpoints[@]}"; do
    echo "Attempting to create token via: $ep"
    status=$(curl -s -k -u "${SPLUNK_USER}:${SPLUNK_PASSWORD}" -o "$TMP_CREATE_RESP" -w "%{http_code}" "$ep" -d "name=my-dev-token" -d "description=Token for development" -d "disabled=0" || echo "000")
    body=$(cat "$TMP_CREATE_RESP" 2>/dev/null || true)
    echo "  HTTP status: $status"
    if [ -n "$(printf '%s' "$body" | head -n 1)" ]; then
      echo "  Response preview: $(printf '%s' "$body" | head -n 5 | tr -d '\n' )..."
    fi

    # Accept 200/201 or 'already exists' responses and attempt extraction
    if [ "$status" = "201" ] || [ "$status" = "200" ] || printf '%s' "$body" | grep -qi 'already exists'; then
      HEC_JSON="$body"
      break
    fi
  done

  rm -f "$TMP_CREATE_RESP"

  # Extract token from HEC_JSON (try jq, python, grep as fallbacks)
  if [ -n "$HEC_JSON" ]; then
    if command -v jq >/dev/null 2>&1; then
      HEC_TOKEN=$(printf '%s' "$HEC_JSON" | jq -r '.entry[0].content.token // empty')
    else
      HEC_TOKEN=$(python3 - <<PY
import sys, json
try:
    j=json.loads(sys.stdin.read() or '{}')
    entry=j.get('entry') or []
    if entry:
        content=entry[0].get('content', {})
        print(content.get('token',''))
    else:
        print('')
except Exception:
    print('')
PY
<<<"$HEC_JSON")
    fi
  fi

  # Fallback: simple grep if previous methods didn't work
  if [ -z "$HEC_TOKEN" ] && [ -n "$HEC_JSON" ]; then
    HEC_TOKEN=$(printf '%s' "$HEC_JSON" | grep -o '"token"[[:space:]]*:[[:space:]]*"[^"]*"' | head -n1 | sed -E 's/.*"token"[[:space:]]*:[[:space:]]*"([^"]*)".*/\1/')
  fi

  if [ -z "$HEC_TOKEN" ]; then
    # Try listing endpoints to find an existing token by name
    LIST_JSON=$(curl -s -k -u "${SPLUNK_USER}:${SPLUNK_PASSWORD}" "https://${SPLUNK_HOST}:${SPLUNK_API_PORT}/services/data/inputs/http/http?output_mode=json" || true)
    LIST_JSON_NS=$(curl -s -k -u "${SPLUNK_USER}:${SPLUNK_PASSWORD}" "https://${SPLUNK_HOST}:${SPLUNK_API_PORT}/servicesNS/nobody/splunk_httpinput/data/inputs/http?output_mode=json" || true)
    combined="$LIST_JSON\n$LIST_JSON_NS"

    if command -v jq >/dev/null 2>&1; then
      HEC_TOKEN=$(printf '%s' "$combined" | jq -r '.entry[]? | select((.name // "") | test("my-dev-token")) | .content.token // empty' | head -n1)
    else
      HEC_TOKEN=$(python3 - <<PY
import sys, json
try:
    s=sys.stdin.read()
    parts=[p for p in s.split('\n') if p.strip()]
    for p in parts:
        try:
            j=json.loads(p)
        except Exception:
            continue
        for e in j.get('entry', []):
            name = e.get('name','') or ''
            if 'my-dev-token' in name:
                print(e.get('content', {}).get('token',''))
                raise SystemExit
except SystemExit:
    pass
except Exception:
    pass
print('')
PY
<<<"$combined")
    fi

    # final grep fallback: search for token string near the name
    if [ -z "$HEC_TOKEN" ]; then
      HEC_TOKEN=$(printf '%s' "$combined" | awk '/my-dev-token/{for(i=NR;i<=NR+6;i++){getline; print}}' 2>/dev/null | grep -o '"token"[[:space:]]*:[[:space:]]*"[^"]*"' | head -n1 | sed -E 's/.*"token"[[:space:]]*:[[:space:]]*"([^"]*)".*/\1/')
    fi
  fi

  if [ -z "$HEC_TOKEN" ]; then
    echo "Failed to create or locate HEC token. Last response:"
    printf '%s\n' "$HEC_JSON" | sed -n '1,200p'
    echo "Possible causes: incorrect Splunk admin credentials, Splunk not ready for management API calls, or the Splunk image requires additional setup (accept license/ui init)."
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
echo "  HEC Token: $HEC_TOKEN"
echo ""
echo "To stop the Splunk instance, run: docker-compose down"
