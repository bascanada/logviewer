#!/bin/bash
# SSH + HL Integration Test Script
# Tests hl engine over SSH connection comparing with native engine
#
# Prerequisites:
#   - Docker and docker-compose installed
#   - SSH container running (docker-compose up ssh-server)
#   - Log files generated on remote
#
# Usage: ./test-ssh-hl.sh [options]
#   --setup       Set up SSH container and generate logs
#   --size N      Number of log entries to generate (default: 10000)
#   --skip-setup  Skip setup, use existing logs
#   --rebuild     Force rebuild of SSH container

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"
LOGVIEWER="$ROOT_DIR/build/logviewer"

# Configuration
SSH_HOST="localhost"
SSH_PORT="2222"
SSH_USER="testuser"
SSH_KEY="$SCRIPT_DIR/ssh/id_rsa"
LOG_SIZE=10000
SKIP_SETUP=false
REBUILD=false

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
        --setup) SKIP_SETUP=false; shift ;;
        --size) LOG_SIZE="$2"; shift 2 ;;
        --skip-setup) SKIP_SETUP=true; shift ;;
        --rebuild) REBUILD=true; shift ;;
        -h|--help)
            echo "Usage: $0 [options]"
            echo "  --setup       Set up SSH container and generate logs"
            echo "  --size N      Number of log entries (default: 10000)"
            echo "  --skip-setup  Skip setup, use existing logs"
            echo "  --rebuild     Force rebuild of SSH container"
            exit 0
            ;;
        *) echo "Unknown option: $1"; exit 1 ;;
    esac
done

# Test results
PASSED=0
FAILED=0

# Helper functions
log_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
log_success() { echo -e "${GREEN}[PASS]${NC} $1"; }
log_fail() { echo -e "${RED}[FAIL]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }

ssh_cmd() {
    ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null \
        -i "$SSH_KEY" -p "$SSH_PORT" "${SSH_USER}@${SSH_HOST}" "$@" 2>/dev/null
}

wait_for_ssh() {
    log_info "Waiting for SSH server to be ready..."
    local max_attempts=30
    local attempt=0
    while [ $attempt -lt $max_attempts ]; do
        if ssh_cmd "echo ready" >/dev/null 2>&1; then
            log_success "SSH server is ready"
            return 0
        fi
        attempt=$((attempt + 1))
        sleep 1
    done
    log_fail "SSH server not ready after $max_attempts seconds"
    return 1
}

# Setup function
setup_ssh_container() {
    log_info "Setting up SSH container..."

    cd "$SCRIPT_DIR"

    # Generate SSH keys if needed
    if [ ! -f "$SSH_KEY" ]; then
        log_info "Generating SSH keys..."
        ./ssh/generate-keys.sh
    fi

    # Build/rebuild container
    if [ "$REBUILD" = true ]; then
        log_info "Rebuilding SSH container..."
        docker-compose build --no-cache ssh-server
    fi

    # Start container
    log_info "Starting SSH container..."
    docker-compose up -d ssh-server

    # Wait for SSH to be ready
    wait_for_ssh

    # Generate logs on remote
    log_info "Generating $LOG_SIZE log entries on remote..."
    ssh_cmd "/scripts/generate-logs.sh $LOG_SIZE /data/test-logs.json"

    # Verify hl is available
    log_info "Checking hl installation on remote..."
    if ssh_cmd "hl --version" >/dev/null 2>&1; then
        HL_VERSION=$(ssh_cmd "hl --version" 2>/dev/null | head -1)
        log_success "hl is installed: $HL_VERSION"
    else
        log_fail "hl is not available on remote"
        exit 1
    fi

    # Verify logs were generated
    LOG_COUNT=$(ssh_cmd "wc -l < /data/test-logs.json" 2>/dev/null | tr -d ' ')
    log_success "Generated $LOG_COUNT log entries on remote"

    cd "$ROOT_DIR"
}

# Build logviewer if needed
build_logviewer() {
    if [ ! -f "$LOGVIEWER" ]; then
        log_info "Building logviewer..."
        cd "$ROOT_DIR" && make build
    fi
}

# Create test config
create_test_config() {
    cat > /tmp/test-ssh-hl-config.yaml << EOF
clients:
  ssh-hl:
    type: ssh
    options:
      addr: "${SSH_HOST}:${SSH_PORT}"
      user: "${SSH_USER}"
      privateKey: "${SSH_KEY}"
      disablePTY: true
      paths:
        - /data/test-logs.json
      cmd: "cat /data/test-logs.json"

  ssh-native:
    type: ssh
    options:
      addr: "${SSH_HOST}:${SSH_PORT}"
      user: "${SSH_USER}"
      privateKey: "${SSH_KEY}"
      disablePTY: true
      paths:
        - /data/test-logs.json
      cmd: "cat /data/test-logs.json"
      preferNativeDriver: true

searches:
  json-extract:
    fieldExtraction:
      json: true
      jsonTimestampKey: "@timestamp"
      jsonLevelKey: "level"
      jsonMessageKey: "message"

contexts:
  ssh-hl-test:
    client: ssh-hl
    searchInherit: [json-extract]
    search:
      fields: {}

  ssh-native-test:
    client: ssh-native
    searchInherit: [json-extract]
    search:
      fields: {}
EOF
}

# Run a comparison test
run_comparison_test() {
    local test_name=$1
    local query_args=$2

    echo ""
    echo -e "${CYAN}>>> Testing: $test_name${NC}"
    echo "  Query args: $query_args"

    # Run with HL engine
    echo "  Running with SSH + hl engine..."
    local hl_output=$(mktemp)
    local hl_start=$(date +%s%N)
    LOGVIEWER_CONFIG=/tmp/test-ssh-hl-config.yaml $LOGVIEWER query log -i ssh-hl-test $query_args --json 2>/dev/null > "$hl_output" || true
    local hl_end=$(date +%s%N)
    local hl_time=$(( (hl_end - hl_start) / 1000000 ))
    local hl_count=$(wc -l < "$hl_output" | tr -d ' ')

    # Run with Native engine
    echo "  Running with SSH + native engine..."
    local native_output=$(mktemp)
    local native_start=$(date +%s%N)
    LOGVIEWER_CONFIG=/tmp/test-ssh-hl-config.yaml $LOGVIEWER query log -i ssh-native-test $query_args --json 2>/dev/null > "$native_output" || true
    local native_end=$(date +%s%N)
    local native_time=$(( (native_end - native_start) / 1000000 ))
    local native_count=$(wc -l < "$native_output" | tr -d ' ')

    # Compare results
    if diff -q "$hl_output" "$native_output" >/dev/null 2>&1; then
        local speedup="1.00"
        if [ "$hl_time" -gt 0 ]; then
            speedup=$(echo "scale=2; $native_time / $hl_time" | bc 2>/dev/null || echo "N/A")
        fi
        log_success "$test_name - outputs match (hl: ${hl_time}ms, native: ${native_time}ms, speedup: ${speedup}x, entries: $hl_count)"
        PASSED=$((PASSED + 1))
    else
        log_fail "$test_name - outputs differ (hl: $hl_count entries, native: $native_count entries)"
        echo "  HL entries: $hl_count, Native entries: $native_count"
        # Save diff for debugging
        mkdir -p /tmp/ssh-hl-test-diffs
        diff "$hl_output" "$native_output" > "/tmp/ssh-hl-test-diffs/${test_name}.diff" 2>&1 || true
        echo "  Diff saved to: /tmp/ssh-hl-test-diffs/${test_name}.diff"
        FAILED=$((FAILED + 1))
    fi

    rm -f "$hl_output" "$native_output"
}

# Main
main() {
    echo ""
    echo -e "${BLUE}========================================"
    echo "SSH + HL Integration Test"
    echo -e "========================================${NC}"
    echo ""

    # Setup if needed
    if [ "$SKIP_SETUP" = false ]; then
        setup_ssh_container
    else
        log_info "Skipping setup (--skip-setup)"
        wait_for_ssh || exit 1
    fi

    # Build logviewer
    build_logviewer

    # Create config
    create_test_config

    echo ""
    echo -e "${BLUE}Running comparison tests...${NC}"

    # Run tests
    run_comparison_test "no-filter" "--size 1000"
    run_comparison_test "simple-filter" "-f level=ERROR --size 1000"
    run_comparison_test "or-logic" "-q 'level=ERROR OR level=WARN' --size 1000"
    run_comparison_test "and-logic" "-q 'level=ERROR AND app=api-gateway' --size 1000"
    run_comparison_test "not-logic" "-q 'NOT level=DEBUG' --size 1000"
    run_comparison_test "complex-nested" "-q '(level=ERROR OR level=WARN) AND latency_ms>=500' --size 1000"
    run_comparison_test "combined-filters" "-f level=ERROR -f 'latency_ms>=1000' --size 1000"
    run_comparison_test "exists" "-q 'exists(trace_id)' --size 1000"
    run_comparison_test "regex" "-f 'message~=.*failed.*' --size 1000"
    run_comparison_test "comparison-gte" "-f 'latency_ms>=2000' --size 1000"
    run_comparison_test "comparison-lt" "-f 'latency_ms<1000' --size 1000"

    # Performance test with larger result set
    echo ""
    echo -e "${BLUE}Running performance test (full dataset)...${NC}"
    run_comparison_test "full-dataset-filter" "-f level=ERROR"

    # Summary
    echo ""
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}TEST RESULTS${NC}"
    echo -e "${BLUE}========================================${NC}"
    echo -e "  ${GREEN}Passed: $PASSED${NC}"
    echo -e "  ${RED}Failed: $FAILED${NC}"
    echo ""

    if [ $FAILED -gt 0 ]; then
        echo "Failed test diffs are in: /tmp/ssh-hl-test-diffs/"
        exit 1
    fi

    exit 0
}

main "$@"
