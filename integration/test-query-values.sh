#!/bin/bash
# Test script for `query values` command across all backend implementations
# Usage: ./integration/test-query-values.sh [backend]
# Backends: splunk, opensearch, k8s, cloudwatch, all (default)

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"
LOGVIEWER="$ROOT_DIR/build/logviewer"
export LOGVIEWER_CONFIG="$SCRIPT_DIR/config.yaml:$SCRIPT_DIR/config.extra.yaml"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

print_header() {
    echo ""
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}========================================${NC}"
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

# Run command: prints the command then executes it
# Exits on failure due to set -e
run() {
    print_cmd "$@"
    "$@"
    echo ""
}

# Check if logviewer binary exists
if [ ! -f "$LOGVIEWER" ]; then
    echo "Building logviewer..."
    cd "$ROOT_DIR" && make build
fi

# =============================================================================
# SPLUNK TESTS
# =============================================================================
test_splunk() {
    print_header "SPLUNK BACKEND TESTS"

    print_test "Test 1: Single field (level)"
    run $LOGVIEWER query values -i splunk-all level --last 1h

    print_test "Test 2: Multiple fields (level, app)"
    run $LOGVIEWER query values -i splunk-all level app --last 1h

    print_test "Test 3: With filter applied (trace_id where level=ERROR)"
    run $LOGVIEWER query values -i splunk-all trace_id -f level=ERROR --last 1h

    print_test "Test 4: JSON output"
    run $LOGVIEWER query values -i splunk-all level app --last 1h --json

    print_test "Test 5: Using payment-service context"
    run $LOGVIEWER query values -i payment-service level --last 1h

    print_success "Splunk tests completed"
}

# =============================================================================
# OPENSEARCH TESTS
# =============================================================================
test_opensearch() {
    print_header "OPENSEARCH BACKEND TESTS"

    print_test "Test 1: Single field (level)"
    run $LOGVIEWER query values -i opensearch-all level --last 1h

    print_test "Test 2: Multiple fields (level, app)"
    run $LOGVIEWER query values -i opensearch-all level app --last 1h

    print_test "Test 3: Using order-service context"
    run $LOGVIEWER query values -i order-service level --last 1h

    print_test "Test 4: Using api-gateway context"
    run $LOGVIEWER query values -i api-gateway level app --last 1h

    print_test "Test 5: JSON output"
    run $LOGVIEWER query values -i opensearch-all level --last 1h --json

    print_success "OpenSearch tests completed"
}

# =============================================================================
# KUBERNETES TESTS
# =============================================================================
test_k8s() {
    print_header "KUBERNETES BACKEND TESTS"

    print_test "Test 1: Single field with label selector (level)"
    run $LOGVIEWER query values -i payment-processor-all level --last 1h

    print_test "Test 2: Multiple fields (level, trace_id)"
    run $LOGVIEWER query values -i payment-processor-all level trace_id --last 1h

    print_test "Test 3: JSON output"
    run $LOGVIEWER query values -i payment-processor-all level --last 1h --json

    print_success "Kubernetes tests completed"
}

# =============================================================================
# CLOUDWATCH TESTS (via LocalStack)
# =============================================================================
test_cloudwatch() {
    print_header "CLOUDWATCH BACKEND TESTS (LocalStack)"

    print_test "Test 1: Single field (level)"
    run $LOGVIEWER query values -i cloudwatch-orders level --last 1h

    print_test "Test 2: JSON output"
    run $LOGVIEWER query values -i cloudwatch-orders level --last 1h --json

    print_success "CloudWatch tests completed"
}

# =============================================================================
# COMPARISON TESTS (query field vs query values)
# =============================================================================
test_comparison() {
    print_header "COMPARISON: query field vs query values"

    print_test "query field output (for reference):"
    run $LOGVIEWER query field -i splunk-all --last 1h --size 20

    print_test "query values output (should have same format):"
    run $LOGVIEWER query values -i splunk-all level app --last 1h

    print_success "Comparison tests completed"
}

# =============================================================================
# AD-HOC TESTS (without config)
# =============================================================================
test_adhoc() {
    print_header "AD-HOC TESTS (direct endpoint)"

    print_test "Test 1: Direct OpenSearch endpoint"
    run $LOGVIEWER query values level app \
        --opensearch-endpoint http://localhost:9200 \
        --elk-index app-logs \
        --last 1h

    print_success "Ad-hoc tests completed"
}

# =============================================================================
# MAIN
# =============================================================================
main() {
    local backend="${1:-all}"

    print_header "QUERY VALUES INTEGRATION TESTS"
    echo "Config: $LOGVIEWER_CONFIG"
    echo "Binary: $LOGVIEWER"
    echo "Backend: $backend"

    case "$backend" in
        splunk)
            test_splunk
            ;;
        opensearch)
            test_opensearch
            ;;
        k8s)
            test_k8s
            ;;
        cloudwatch)
            test_cloudwatch
            ;;
        comparison)
            test_comparison
            ;;
        adhoc)
            test_adhoc
            ;;
        all)
            test_splunk
            test_opensearch
            test_k8s
            test_cloudwatch
            test_comparison
            ;;
        *)
            echo "Unknown backend: $backend"
            echo "Usage: $0 [splunk|opensearch|k8s|cloudwatch|comparison|adhoc|all]"
            exit 1
            ;;
    esac

    print_header "ALL TESTS COMPLETED"
}

main "$@"
