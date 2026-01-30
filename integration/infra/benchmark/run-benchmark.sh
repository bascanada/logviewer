#!/bin/bash
# Benchmark script comparing hl vs native Go engine performance
# Tests both local and SSH backends with various log file sizes
#
# Prerequisites:
#   - hl installed (cargo install hl or brew install pamburus/tap/hl)
#   - logviewer built (make build)
#   - SSH key configured for SSH tests
#
# Usage: ./run-benchmark.sh [options]
#   --sizes       Log file sizes to test (default: "10000 100000 1000000")
#   --filters     Filter complexity levels (default: "simple complex")
#   --backends    Backends to test (default: "local ssh")
#   --ssh-host    SSH host for remote tests (default: localhost)
#   --ssh-user    SSH user (default: $USER)
#   --iterations  Number of iterations per test (default: 3)
#   --skip-gen    Skip log file generation (use existing files)

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(dirname "$(dirname "$SCRIPT_DIR")")"
LOGVIEWER="$ROOT_DIR/build/logviewer"

# Defaults
SIZES="10000 100000 1000000"
FILTERS="simple complex"
BACKENDS="local"
SSH_HOST="localhost"
SSH_USER="$USER"
SSH_PORT="2222"  # Docker SSH container port
ITERATIONS=3
SKIP_GEN=false
LOG_DIR="/tmp/logviewer-benchmark"
RESULTS_FILE="$SCRIPT_DIR/benchmark-results-$(date +%Y%m%d-%H%M%S).csv"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --sizes) SIZES="$2"; shift 2 ;;
        --filters) FILTERS="$2"; shift 2 ;;
        --backends) BACKENDS="$2"; shift 2 ;;
        --ssh-host) SSH_HOST="$2"; shift 2 ;;
        --ssh-user) SSH_USER="$2"; shift 2 ;;
        --ssh-port) SSH_PORT="$2"; shift 2 ;;
        --iterations) ITERATIONS="$2"; shift 2 ;;
        --skip-gen) SKIP_GEN=true; shift ;;
        -h|--help)
            echo "Usage: $0 [options]"
            echo "  --sizes       Log file sizes (default: \"10000 100000 1000000\")"
            echo "  --filters     Filter types (default: \"simple complex\")"
            echo "  --backends    Backends to test (default: \"local ssh\")"
            echo "  --ssh-host    SSH host (default: localhost)"
            echo "  --ssh-user    SSH user (default: \$USER)"
            echo "  --ssh-port    SSH port (default: 2222)"
            echo "  --iterations  Iterations per test (default: 3)"
            echo "  --skip-gen    Skip log generation"
            exit 0
            ;;
        *) echo "Unknown option: $1"; exit 1 ;;
    esac
done

# Check prerequisites
check_prerequisites() {
    echo -e "${BLUE}Checking prerequisites...${NC}"

    if [ ! -f "$LOGVIEWER" ]; then
        echo "Building logviewer..."
        cd "$ROOT_DIR" && make build
    fi
    echo "  ✓ logviewer: $LOGVIEWER"

    if command -v hl &> /dev/null; then
        HL_VERSION=$(hl --version 2>&1 | head -1)
        echo "  ✓ hl: $HL_VERSION"
        HAS_HL=true
    else
        echo -e "  ${YELLOW}⚠ hl not installed (will only test native engine)${NC}"
        HAS_HL=false
    fi

    mkdir -p "$LOG_DIR"
    echo "  ✓ Log directory: $LOG_DIR"
}

# Generate log files of various sizes
generate_logs() {
    if [ "$SKIP_GEN" = true ]; then
        echo -e "${YELLOW}Skipping log generation (--skip-gen)${NC}"
        return
    fi

    echo -e "${BLUE}Generating log files...${NC}"

    for size in $SIZES; do
        OUTPUT="$LOG_DIR/logs-${size}.json"
        if [ -f "$OUTPUT" ]; then
            echo "  Using existing: $OUTPUT"
            continue
        fi
        echo "  Generating $size entries -> $OUTPUT"
        go run "$SCRIPT_DIR/generate-logs.go" -size "$size" -output "$OUTPUT"
    done
}

# Run a single benchmark test
# Args: $1=engine, $2=backend, $3=size, $4=filter_type
run_test() {
    local engine=$1
    local backend=$2
    local size=$3
    local filter_type=$4
    local log_file="$LOG_DIR/logs-${size}.json"

    # Build filter args based on type
    local filter_args=""
    case $filter_type in
        simple)
            filter_args="-f level=ERROR"
            ;;
        complex)
            filter_args="-q '(level=ERROR OR level=WARN) AND latency_ms>=500'"
            ;;
        regex)
            filter_args="-f 'message~=.*timeout.*'"
            ;;
    esac

    # Build command based on backend and engine
    local cmd=""
    local config_snippet=""

    case $backend in
        local)
            if [ "$engine" = "hl" ]; then
                config_snippet="paths: [\"$log_file\"]"
            else
                config_snippet="cmd: \"cat $log_file\""
                config_snippet="$config_snippet
      preferNativeDriver: true"
            fi
            ;;
        ssh)
            if [ "$engine" = "hl" ]; then
                config_snippet="paths: [\"/data/logs-${size}.json\"]"
            else
                config_snippet="cmd: \"cat /data/logs-${size}.json\""
                config_snippet="$config_snippet
      preferNativeDriver: true"
            fi
            ;;
    esac

    # Create temp config
    local tmp_config=$(mktemp)
    cat > "$tmp_config" << EOF
clients:
  bench-local:
    type: local
    options:
      $config_snippet

  bench-ssh:
    type: ssh
    options:
      addr: "${SSH_HOST}:${SSH_PORT}"
      user: "${SSH_USER}"
      privateKey: "$SCRIPT_DIR/../ssh/id_rsa"
      disablePTY: true
      $config_snippet

searches:
  json-extract:
    fieldExtraction:
      json: true
      jsonTimestampKey: "@timestamp"
      jsonLevelKey: "level"
      jsonMessageKey: "message"

contexts:
  bench:
    client: bench-${backend}
    searchInherit: [json-extract]
    search:
      fields: {}
EOF

    export LOGVIEWER_CONFIG="$tmp_config"

    # Measure execution time
    local total_time=0
    local times=()

    for i in $(seq 1 $ITERATIONS); do
        local start_ns=$(date +%s%N)

        # Run the query (suppress output, capture time)
        eval "$LOGVIEWER query log -i bench $filter_args --size 1000" > /dev/null 2>&1 || true

        local end_ns=$(date +%s%N)
        local duration_ms=$(( (end_ns - start_ns) / 1000000 ))
        times+=($duration_ms)
        total_time=$((total_time + duration_ms))
    done

    # Calculate average
    local avg_time=$((total_time / ITERATIONS))

    # Cleanup
    rm -f "$tmp_config"

    # Return result
    echo "$avg_time"
}

# Run all benchmarks
run_benchmarks() {
    echo -e "${BLUE}Running benchmarks...${NC}"
    echo ""

    # CSV header
    echo "engine,backend,size,filter,avg_ms,speedup" > "$RESULTS_FILE"

    for backend in $BACKENDS; do
        echo -e "${CYAN}=== Backend: $backend ===${NC}"

        for size in $SIZES; do
            echo -e "${YELLOW}  Size: $size entries${NC}"

            for filter in $FILTERS; do
                echo -n "    Filter: $filter ... "

                # Test native engine
                native_time=$(run_test "native" "$backend" "$size" "$filter")

                # Test hl engine (if available)
                if [ "$HAS_HL" = true ]; then
                    hl_time=$(run_test "hl" "$backend" "$size" "$filter")
                    speedup=$(echo "scale=2; $native_time / $hl_time" | bc 2>/dev/null || echo "N/A")
                else
                    hl_time="N/A"
                    speedup="N/A"
                fi

                echo "native=${native_time}ms, hl=${hl_time}ms, speedup=${speedup}x"

                # Write to CSV
                echo "native,$backend,$size,$filter,$native_time,1.00" >> "$RESULTS_FILE"
                if [ "$HAS_HL" = true ]; then
                    echo "hl,$backend,$size,$filter,$hl_time,$speedup" >> "$RESULTS_FILE"
                fi
            done
        done
        echo ""
    done
}

# Print summary
print_summary() {
    echo -e "${GREEN}========================================${NC}"
    echo -e "${GREEN}Benchmark Complete${NC}"
    echo -e "${GREEN}========================================${NC}"
    echo ""
    echo "Results saved to: $RESULTS_FILE"
    echo ""
    echo "Summary:"
    echo "--------"

    if [ "$HAS_HL" = true ]; then
        echo ""
        echo "Performance comparison (hl vs native):"
        # Simple stats from CSV
        awk -F, 'NR>1 && $1=="hl" {
            sum+=$6; count++;
            if($6>max) max=$6;
            if(min=="" || $6<min) min=$6
        } END {
            if(count>0) printf "  Average speedup: %.2fx\n  Max speedup: %.2fx\n  Min speedup: %.2fx\n", sum/count, max, min
        }' "$RESULTS_FILE"
    fi

    echo ""
    echo "To visualize results:"
    echo "  cat $RESULTS_FILE | column -t -s,"
}

# Main
main() {
    echo -e "${BLUE}╔══════════════════════════════════════════╗${NC}"
    echo -e "${BLUE}║  LogViewer HL vs Native Benchmark        ║${NC}"
    echo -e "${BLUE}╚══════════════════════════════════════════╝${NC}"
    echo ""

    check_prerequisites
    echo ""

    generate_logs
    echo ""

    run_benchmarks
    echo ""

    print_summary
}

main "$@"
