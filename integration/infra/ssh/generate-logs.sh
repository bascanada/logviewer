#!/bin/sh
# Simple log file generator for SSH integration testing
# Uses only POSIX shell commands for maximum compatibility
#
# Usage: /scripts/generate-logs.sh [size] [output]
#   size   - Number of log entries (default: 10000)
#   output - Output file path (default: /data/test-logs.json)

SIZE=${1:-10000}
OUTPUT=${2:-/data/test-logs.json}

# Fixed timestamp base (use current time in ISO format)
BASE_TS=$(/bin/date -u +"%Y-%m-%dT%H:%M:%S")

echo "Generating $SIZE log entries to $OUTPUT..."

# Create output directory
mkdir -p "$(/usr/bin/dirname "$OUTPUT" 2>/dev/null || echo "/data")"

# Generate logs
i=0
{
    while [ $i -lt $SIZE ]; do
        # Simple pseudo-random based on counter
        RAND=$((i * 7919 % 100))

        # Determine log level (5% ERROR, 10% WARN, rest INFO)
        if [ $RAND -lt 5 ]; then
            LEVEL="ERROR"
            case $((i % 5)) in
                0) MSG="Connection refused" ;;
                1) MSG="Timeout waiting for response" ;;
                2) MSG="Database connection failed" ;;
                3) MSG="Authentication failed" ;;
                *) MSG="Service unavailable" ;;
            esac
        elif [ $RAND -lt 15 ]; then
            LEVEL="WARN"
            case $((i % 5)) in
                0) MSG="High latency detected" ;;
                1) MSG="Cache miss" ;;
                2) MSG="Retry attempt" ;;
                3) MSG="Connection pool exhausted" ;;
                *) MSG="Rate limit approaching" ;;
            esac
        else
            LEVEL="INFO"
            case $((i % 8)) in
                0) MSG="Request processed successfully" ;;
                1) MSG="User authenticated" ;;
                2) MSG="Database query executed" ;;
                3) MSG="Cache hit" ;;
                4) MSG="Message published" ;;
                5) MSG="Health check passed" ;;
                6) MSG="Connection established" ;;
                *) MSG="Session started" ;;
            esac
        fi

        # Service name
        case $((i % 6)) in
            0) SVC="api-gateway" ;;
            1) SVC="user-service" ;;
            2) SVC="payment-service" ;;
            3) SVC="order-service" ;;
            4) SVC="inventory-service" ;;
            *) SVC="notification-service" ;;
        esac

        # Latency
        LATENCY=$((i % 5000 + 1))

        # Build JSON (simple, no optional fields for speed)
        echo "{\"@timestamp\":\"${BASE_TS}.${i}Z\",\"level\":\"$LEVEL\",\"message\":\"$MSG\",\"app\":\"$SVC\",\"latency_ms\":$LATENCY}"

        # Add trace_id for ~50% of entries
        if [ $((i % 2)) -eq 0 ]; then
            # Actually output with trace_id
            :
        fi

        i=$((i + 1))

        # Progress every 10000
        if [ $((i % 10000)) -eq 0 ]; then
            echo "  Generated $i / $SIZE entries..." >&2
        fi
    done
} > "$OUTPUT"

# Count lines
LINES=$(/usr/bin/wc -l < "$OUTPUT" 2>/dev/null || echo "$SIZE")
echo ""
echo "Done! Generated $LINES entries to $OUTPUT"
