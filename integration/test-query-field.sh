#!/bin/bash
# Test script for `query field` command across all backend implementations
# Usage: ./integration/test-query-field.sh [backend]
# Backends: splunk, opensearch, k8s, cloudwatch, adhoc, all (default)

source "$(dirname "${BASH_SOURCE[0]}")/test-common.sh"

test_splunk() {
    print_header "SPLUNK FIELD TESTS"

    print_test "Test 1: Discover fields from Splunk logs"
    run $LOGVIEWER query field -i splunk-all --last 1h --size 10

    print_test "Test 2: Discover fields from payment service"
    run $LOGVIEWER query field -i payment-service --last 1h --size 10

    print_test "Test 3: JSON output"
    run $LOGVIEWER query field -i splunk-all --last 1h --size 5 --json

    print_success "Splunk field tests completed"
}

test_opensearch() {
    print_header "OPENSEARCH FIELD TESTS"

    print_test "Test 1: Discover fields from OpenSearch logs"
    run $LOGVIEWER query field -i opensearch-all --last 1h --size 10

    print_test "Test 2: Discover fields from order service"
    run $LOGVIEWER query field -i order-service --last 1h --size 10

    print_test "Test 3: Discover fields from API gateway"
    run $LOGVIEWER query field -i api-gateway --last 1h --size 10

    print_test "Test 4: JSON output"
    run $LOGVIEWER query field -i opensearch-all --last 1h --size 5 --json

    print_success "OpenSearch field tests completed"
}

test_k8s() {
    print_header "KUBERNETES FIELD TESTS"

    print_test "Test 1: Discover fields from K8s pod logs"
    run $LOGVIEWER query field -i payment-processor-all --last 1h --size 10

    print_test "Test 2: JSON output"
    run $LOGVIEWER query field -i payment-processor-all --last 1h --size 5 --json

    print_success "Kubernetes field tests completed"
}

test_cloudwatch() {
    print_header "CLOUDWATCH FIELD TESTS (LocalStack)"

    print_test "Test 1: Discover fields from CloudWatch logs"
    run $LOGVIEWER query field -i cloudwatch-orders --last 1h --size 10

    print_test "Test 2: JSON output"
    run $LOGVIEWER query field -i cloudwatch-orders --last 1h --size 5 --json

    print_success "CloudWatch field tests completed"
}

test_adhoc() {
    print_header "AD-HOC FIELD TESTS (direct endpoint)"

    print_test "Test 1: Direct OpenSearch endpoint"
    run $LOGVIEWER query field \
        --opensearch-endpoint http://localhost:9200 \
        --elk-index app-logs \
        --last 1h \
        --size 10

    print_success "Ad-hoc field tests completed"
}

run_backend_tests "QUERY FIELD INTEGRATION TESTS" "${1:-all}" splunk opensearch k8s cloudwatch adhoc
