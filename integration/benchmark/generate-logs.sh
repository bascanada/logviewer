#!/bin/bash
# Log file generator for performance benchmarking
# Generates JSON log files with configurable size and characteristics
#
# Usage: ./generate-logs.sh [options]
#   -s, --size       Number of log entries (default: 100000)
#   -o, --output     Output file path (default: /tmp/benchmark-logs.json)
#   -e, --error-rate Error log percentage (default: 5)
#   -w, --warn-rate  Warning log percentage (default: 10)
#   --multiline      Include multiline stack traces (default: false)
#   --compress       Compress output with gzip (default: false)

set -e

# Defaults
SIZE=100000
OUTPUT="/tmp/benchmark-logs.json"
ERROR_RATE=5
WARN_RATE=10
MULTILINE=false
COMPRESS=false

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -s|--size)
            SIZE="$2"
            shift 2
            ;;
        -o|--output)
            OUTPUT="$2"
            shift 2
            ;;
        -e|--error-rate)
            ERROR_RATE="$2"
            shift 2
            ;;
        -w|--warn-rate)
            WARN_RATE="$2"
            shift 2
            ;;
        --multiline)
            MULTILINE=true
            shift
            ;;
        --compress)
            COMPRESS=true
            shift
            ;;
        -h|--help)
            echo "Usage: $0 [options]"
            echo "  -s, --size       Number of log entries (default: 100000)"
            echo "  -o, --output     Output file path (default: /tmp/benchmark-logs.json)"
            echo "  -e, --error-rate Error log percentage (default: 5)"
            echo "  -w, --warn-rate  Warning log percentage (default: 10)"
            echo "  --multiline      Include multiline stack traces"
            echo "  --compress       Compress output with gzip"
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            exit 1
            ;;
    esac
done

# Services and their typical messages
SERVICES=("api-gateway" "user-service" "payment-service" "order-service" "inventory-service" "notification-service")

INFO_MESSAGES=(
    "Request processed successfully"
    "User authenticated"
    "Database query executed"
    "Cache hit for key"
    "Message published to queue"
    "Health check passed"
    "Connection established"
    "Session started"
    "Configuration loaded"
    "Metrics collected"
)

WARN_MESSAGES=(
    "High latency detected"
    "Cache miss, fetching from database"
    "Retry attempt"
    "Connection pool exhausted"
    "Rate limit approaching"
    "Deprecated API called"
    "Memory usage high"
    "Queue backlog growing"
)

ERROR_MESSAGES=(
    "Connection refused"
    "Timeout waiting for response"
    "Database connection failed"
    "Authentication failed"
    "Invalid request payload"
    "Service unavailable"
    "Out of memory"
    "Disk space critical"
)

STACK_TRACE='    at com.example.service.Handler.process(Handler.java:42)
    at com.example.service.Controller.handle(Controller.java:128)
    at org.springframework.web.servlet.FrameworkServlet.service(FrameworkServlet.java:897)
    at javax.servlet.http.HttpServlet.service(HttpServlet.java:750)
    at org.apache.catalina.core.ApplicationFilterChain.doFilter(ApplicationFilterChain.java:166)
Caused by: java.net.SocketTimeoutException: Read timed out
    at java.net.SocketInputStream.socketRead0(Native Method)
    at java.net.SocketInputStream.read(SocketInputStream.java:150)'

echo "Generating $SIZE log entries to $OUTPUT..."
echo "  Error rate: ${ERROR_RATE}%"
echo "  Warn rate: ${WARN_RATE}%"
echo "  Multiline: $MULTILINE"
echo ""

# Create output directory if needed
mkdir -p "$(dirname "$OUTPUT")"

# Start time for progress reporting
START_TIME=$(date +%s)

# Generate logs
{
    for i in $(seq 1 $SIZE); do
        # Generate timestamp (spread over last 24 hours)
        OFFSET=$((RANDOM % 86400))
        TIMESTAMP=$(date -u -v-${OFFSET}S +"%Y-%m-%dT%H:%M:%S.000Z" 2>/dev/null || date -u -d "-${OFFSET} seconds" +"%Y-%m-%dT%H:%M:%S.000Z")

        # Select random service
        SERVICE=${SERVICES[$((RANDOM % ${#SERVICES[@]}))]}

        # Determine log level based on rates
        RAND=$((RANDOM % 100))
        if [ $RAND -lt $ERROR_RATE ]; then
            LEVEL="ERROR"
            MSG=${ERROR_MESSAGES[$((RANDOM % ${#ERROR_MESSAGES[@]}))]}
        elif [ $RAND -lt $((ERROR_RATE + WARN_RATE)) ]; then
            LEVEL="WARN"
            MSG=${WARN_MESSAGES[$((RANDOM % ${#WARN_MESSAGES[@]}))]}
        else
            LEVEL="INFO"
            MSG=${INFO_MESSAGES[$((RANDOM % ${#INFO_MESSAGES[@]}))]}
        fi

        # Generate random latency (1-5000ms)
        LATENCY=$((RANDOM % 5000 + 1))

        # Generate trace ID (50% of logs have it)
        TRACE_ID=""
        if [ $((RANDOM % 2)) -eq 0 ]; then
            TRACE_ID=$(printf '%08x-%04x-%04x-%04x-%012x' $RANDOM $RANDOM $RANDOM $RANDOM $RANDOM)
        fi

        # Generate user ID (for some logs)
        USER_ID=""
        if [ $((RANDOM % 3)) -eq 0 ]; then
            USER_ID="user_$((RANDOM % 10000))"
        fi

        # Build JSON
        JSON="{\"@timestamp\":\"$TIMESTAMP\",\"level\":\"$LEVEL\",\"message\":\"$MSG\",\"app\":\"$SERVICE\",\"latency_ms\":$LATENCY"

        if [ -n "$TRACE_ID" ]; then
            JSON="$JSON,\"trace_id\":\"$TRACE_ID\""
        fi

        if [ -n "$USER_ID" ]; then
            JSON="$JSON,\"user_id\":\"$USER_ID\""
        fi

        # Add stack trace for errors with multiline option
        if [ "$MULTILINE" = true ] && [ "$LEVEL" = "ERROR" ]; then
            # Escape the stack trace for JSON
            ESCAPED_TRACE=$(echo "$STACK_TRACE" | sed 's/\\/\\\\/g' | sed ':a;N;$!ba;s/\n/\\n/g')
            JSON="$JSON,\"stack_trace\":\"$ESCAPED_TRACE\""
        fi

        JSON="$JSON}"
        echo "$JSON"

        # Progress reporting every 10000 entries
        if [ $((i % 10000)) -eq 0 ]; then
            ELAPSED=$(($(date +%s) - START_TIME))
            RATE=$((i / (ELAPSED + 1)))
            echo "  Generated $i / $SIZE entries (${RATE}/s)..." >&2
        fi
    done
} > "$OUTPUT"

# Compress if requested
if [ "$COMPRESS" = true ]; then
    gzip -f "$OUTPUT"
    OUTPUT="$OUTPUT.gz"
fi

# Final stats
END_TIME=$(date +%s)
DURATION=$((END_TIME - START_TIME))
FILE_SIZE=$(ls -lh "$OUTPUT" | awk '{print $5}')

echo ""
echo "Done!"
echo "  File: $OUTPUT"
echo "  Size: $FILE_SIZE"
echo "  Entries: $SIZE"
echo "  Duration: ${DURATION}s"
echo "  Rate: $((SIZE / (DURATION + 1))) entries/s"
