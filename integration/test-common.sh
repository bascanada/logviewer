#!/bin/bash
# Common utilities for integration test scripts
# Source this file: source "$(dirname "${BASH_SOURCE[0]}")/test-common.sh"

set -e

# Resolve paths
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[1]}")" && pwd)"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"
LOGVIEWER="$ROOT_DIR/build/logviewer"
export LOGVIEWER_CONFIG="$SCRIPT_DIR/config.yaml:$SCRIPT_DIR/config.extra.yaml"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
MAGENTA='\033[0;35m'
NC='\033[0m'

print_header() {
    echo ""
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}========================================${NC}"
}

print_major_header() {
    echo ""
    echo -e "${MAGENTA}########################################${NC}"
    echo -e "${MAGENTA}# $1${NC}"
    echo -e "${MAGENTA}########################################${NC}"
}

print_test() {
    echo ""
    echo -e "${YELLOW}>>> $1${NC}"
}

print_cmd() {
    echo -e "${CYAN}\$ $@${NC}"
}

print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

print_error() {
    echo -e "${RED}✗ $1${NC}"
}

# Run command: prints the command then executes it
run() {
    print_cmd "$@"
    "$@"
    echo ""
}

# Ensure logviewer binary exists
ensure_binary() {
    if [ ! -f "$LOGVIEWER" ]; then
        echo "Building logviewer..."
        cd "$ROOT_DIR" && make build
    fi
}

# Print test configuration
print_config() {
    local test_name="$1"
    local backend="${2:-all}"
    print_header "$test_name"
    echo "Config: $LOGVIEWER_CONFIG"
    echo "Binary: $LOGVIEWER"
    echo "Backend: $backend"
}

# Standard main wrapper for backend-based tests
run_backend_tests() {
    local test_name="$1"
    local backend="$2"
    shift 2
    local -a backends=("$@")

    ensure_binary
    print_config "$test_name" "$backend"

    case "$backend" in
        all)
            for b in "${backends[@]}"; do
                "test_$b"
            done
            ;;
        *)
            local found=false
            for b in "${backends[@]}"; do
                if [ "$backend" = "$b" ]; then
                    "test_$b"
                    found=true
                    break
                fi
            done
            if [ "$found" = false ]; then
                echo "Unknown backend: $backend"
                echo "Available: ${backends[*]} all"
                exit 1
            fi
            ;;
    esac

    print_header "ALL TESTS COMPLETED"
}
