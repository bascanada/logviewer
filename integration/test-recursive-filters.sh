#!/bin/bash
# Test script for recursive filter functionality (AND/OR/NOT)
# Usage: ./integration/test-recursive-filters.sh [backend]
# Backends: splunk, opensearch, k8s, all (default)

source "$(dirname "${BASH_SOURCE[0]}")/test-common.sh"

test_splunk() {
    print_header "SPLUNK RECURSIVE FILTER TESTS"

    print_test "Test 1: Simple OR filter (ERROR OR WARN)"
    run $LOGVIEWER query log -i splunk-errors-or-warns --last 1h --size 10

    print_test "Test 2: Nested AND/OR filter ((ERROR OR WARN) AND app=log-generator)"
    run $LOGVIEWER query log -i splunk-critical --last 1h --size 10

    print_test "Test 3: NOT filter (exclude DEBUG logs)"
    run $LOGVIEWER query log -i splunk-no-debug --last 1h --size 10

    print_test "Test 4: Legacy fields + new filter combined"
    run $LOGVIEWER query log -i splunk-legacy-plus-filter --last 1h --size 10

    print_test "Test 5: EXISTS operator (logs with trace_id)"
    run $LOGVIEWER query log -i splunk-has-trace --last 1h --size 10

    print_success "Splunk recursive filter tests completed"
}

test_opensearch() {
    print_header "OPENSEARCH RECURSIVE FILTER TESTS"

    print_test "Test 1: Simple OR filter (ERROR OR WARN)"
    run $LOGVIEWER query log -i opensearch-errors-or-warns --last 1h --size 10

    print_test "Test 2: Complex nested filter (AND + nested OR)"
    run $LOGVIEWER query log -i opensearch-complex --last 1h --size 10

    print_test "Test 3: Regex filter with OR"
    run $LOGVIEWER query log -i opensearch-regex-errors --last 1h --size 10

    print_test "Test 4: Wildcard operator"
    run $LOGVIEWER query log -i opensearch-wildcard --last 1h --size 10

    print_success "OpenSearch recursive filter tests completed"
}

test_k8s() {
    print_header "K8S CLIENT-SIDE FILTER TESTS"

    print_test "Test 1: Client-side OR filter (ERROR OR WARN)"
    run $LOGVIEWER query log -i k8s-errors-or-warns --last 1h --size 10

    print_test "Test 2: Client-side NOT filter (exclude DEBUG)"
    run $LOGVIEWER query log -i k8s-no-debug --last 1h --size 10

    print_success "K8s client-side filter tests completed"
}

run_backend_tests "RECURSIVE FILTER INTEGRATION TESTS" "${1:-all}" splunk opensearch k8s
