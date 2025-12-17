#!/bin/bash
# Test script for native query functionality (Splunk SPL, OpenSearch Lucene)
# Usage: ./integration/test-native-queries.sh [backend]
# Backends: splunk, opensearch, all (default)

source "$(dirname "${BASH_SOURCE[0]}")/test-common.sh"

test_splunk() {
    print_header "SPLUNK NATIVE QUERY TESTS"

    print_test "Test 1: Basic native SPL query (uses /events endpoint)"
    run $LOGVIEWER query log -i splunk-native-query --last 1h --size 10

    print_test "Test 2: Native query with stats command (uses /results endpoint)"
    run $LOGVIEWER query log -i splunk-native-stats --last 1h --size 20

    print_test "Test 3: Native query with timechart (uses /results endpoint)"
    run $LOGVIEWER query log -i splunk-native-timechart --last 24h --size 30

    print_test "Test 4: Native query with additional filters"
    run $LOGVIEWER query log -i splunk-native-with-filter --last 1h --size 10

    print_test "Test 5: Ad-hoc native SPL query (direct endpoint)"
    run $LOGVIEWER query log \
        --splunk-endpoint https://localhost:8089/services \
        --client-headers "$SCRIPT_DIR/splunk-headers.txt" \
        --native-query 'index=main sourcetype=httpevent | head 5' \
        --last 1h

    print_success "Splunk native query tests completed"
}

test_opensearch() {
    print_header "OPENSEARCH NATIVE QUERY TESTS"

    print_test "Test 1: Basic native Lucene query"
    run $LOGVIEWER query log -i opensearch-native-query --last 1h --size 10

    print_test "Test 2: Native query with additional filters"
    run $LOGVIEWER query log -i opensearch-native-with-filter --last 1h --size 10

    print_test "Test 3: Ad-hoc native Lucene query (direct endpoint)"
    run $LOGVIEWER query log \
        --opensearch-endpoint http://localhost:9200 \
        --elk-index app-logs \
        --native-query 'level:ERROR OR level:WARN' \
        --last 1h \
        --size 10

    print_test "Test 4: JSON output with native query"
    run $LOGVIEWER query log -i opensearch-native-query --last 1h --size 5 --json

    print_success "OpenSearch native query tests completed"
}

run_backend_tests "NATIVE QUERY INTEGRATION TESTS" "${1:-all}" splunk opensearch
