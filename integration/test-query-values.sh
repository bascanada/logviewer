#!/bin/bash
# Test script for `query values` command across all backend implementations
# Usage: ./integration/test-query-values.sh [backend]
# Backends: splunk, opensearch, k8s, cloudwatch, comparison, adhoc, all (default)

source "$(dirname "${BASH_SOURCE[0]}")/test-common.sh"

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

test_cloudwatch() {
    print_header "CLOUDWATCH BACKEND TESTS (LocalStack)"

    print_test "Test 1: Single field (level)"
    run $LOGVIEWER query values -i cloudwatch-orders level --last 1h

    print_test "Test 2: JSON output"
    run $LOGVIEWER query values -i cloudwatch-orders level --last 1h --json

    print_success "CloudWatch tests completed"
}

test_comparison() {
    print_header "COMPARISON: query field vs query values"

    print_test "query field output (for reference):"
    run $LOGVIEWER query field -i splunk-all --last 1h --size 20

    print_test "query values output (should have same format):"
    run $LOGVIEWER query values -i splunk-all level app --last 1h

    print_success "Comparison tests completed"
}

test_adhoc() {
    print_header "AD-HOC TESTS (direct endpoint)"

    print_test "Test 1: Direct OpenSearch endpoint"
    run $LOGVIEWER query values level app \
        --opensearch-endpoint http://localhost:9200 \
        --elk-index app-logs \
        --last 1h

    print_success "Ad-hoc tests completed"
}

run_backend_tests "QUERY VALUES INTEGRATION TESTS" "${1:-all}" splunk opensearch k8s cloudwatch comparison adhoc
