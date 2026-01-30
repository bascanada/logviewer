#!/bin/bash
# Test script for `query log` command across all backend implementations
# Usage: ./integration/test-query-log.sh [backend]
# Backends: splunk, opensearch, k8s, cloudwatch, multi, adhoc, all (default)

source "$(dirname "${BASH_SOURCE[0]}")/test-common.sh"

test_splunk() {
    print_header "SPLUNK BACKEND TESTS"

    print_test "Test 1: Basic log query (splunk-all)"
    run $LOGVIEWER query log -i splunk-all --last 1h --size 5

    print_test "Test 2: Payment service logs"
    run $LOGVIEWER query log -i payment-service --last 1h --size 5

    print_test "Test 3: With field filter (level=ERROR)"
    run $LOGVIEWER query log -i splunk-all -f level=ERROR --last 1h --size 5

    print_test "Test 4: JSON output"
    run $LOGVIEWER query log -i splunk-all --last 1h --size 3 --json

    print_success "Splunk tests completed"
}

test_opensearch() {
    print_header "OPENSEARCH BACKEND TESTS"

    print_test "Test 1: Basic log query (opensearch-all)"
    run $LOGVIEWER query log -i opensearch-all --last 1h --size 5

    print_test "Test 2: Order service logs"
    run $LOGVIEWER query log -i order-service --last 1h --size 5

    print_test "Test 3: API Gateway logs"
    run $LOGVIEWER query log -i api-gateway --last 1h --size 5

    print_test "Test 4: With field filter (level=ERROR)"
    run $LOGVIEWER query log -i opensearch-all -f level=ERROR --last 1h --size 5

    print_test "Test 5: JSON output"
    run $LOGVIEWER query log -i opensearch-all --last 1h --size 3 --json

    print_success "OpenSearch tests completed"
}

test_k8s() {
    print_header "KUBERNETES BACKEND TESTS"

    print_test "Test 1: Payment processor pods (label selector)"
    run $LOGVIEWER query log -i payment-processor-all --last 1h --size 5

    print_test "Test 2: JSON output"
    run $LOGVIEWER query log -i payment-processor-all --last 1h --size 3 --json

    print_success "Kubernetes tests completed"
}

test_cloudwatch() {
    print_header "CLOUDWATCH BACKEND TESTS (LocalStack)"

    print_test "Test 1: Basic log query"
    run $LOGVIEWER query log -i cloudwatch-orders --last 1h --size 5

    print_test "Test 2: JSON output"
    run $LOGVIEWER query log -i cloudwatch-orders --last 1h --size 3 --json

    print_success "CloudWatch tests completed"
}

test_multi() {
    print_header "MULTI-CONTEXT TESTS"

    print_test "Test 1: Query multiple contexts (splunk + opensearch)"
    run $LOGVIEWER query log -i splunk-all -i opensearch-all --last 1h --size 3

    print_success "Multi-context tests completed"
}

test_adhoc() {
    print_header "AD-HOC TESTS (direct endpoint)"

    print_test "Test 1: Direct OpenSearch endpoint"
    run $LOGVIEWER query log \
        --opensearch-endpoint http://localhost:9200 \
        --elk-index app-logs \
        --last 1h \
        --size 5

    print_success "Ad-hoc tests completed"
}

run_backend_tests "QUERY LOG INTEGRATION TESTS" "${1:-all}" splunk opensearch k8s cloudwatch multi adhoc
