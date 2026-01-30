#!/bin/bash
# Test script for hl-compatible query syntax (-f with operators, -q flag)
# Usage: ./integration/test-hl-queries.sh [backend]
# Backends: splunk, opensearch, all (default)

source "$(dirname "${BASH_SOURCE[0]}")/test-common.sh"

# Use 24h to accommodate stale test data
TIME_RANGE="--last 24h"

test_splunk() {
    print_header "SPLUNK HL-COMPATIBLE QUERY TESTS"

    # === Context-based tests (YAML config) ===
    print_test "Test 1: Greater than operator (latency > 1000ms) - context"
    run $LOGVIEWER query log -i splunk-slow-requests $TIME_RANGE --size 10

    print_test "Test 2: Greater or equal operator (latency >= 500ms) - context"
    run $LOGVIEWER query log -i splunk-latency-gte $TIME_RANGE --size 10

    print_test "Test 3: Less than operator (latency < 100ms) - context"
    run $LOGVIEWER query log -i splunk-fast-requests $TIME_RANGE --size 10

    print_test "Test 4: Negate field (level != DEBUG) - context"
    run $LOGVIEWER query log -i splunk-not-debug-negate $TIME_RANGE --size 10

    print_test "Test 5: Negate + regex (message !~= .*success.*) - context"
    run $LOGVIEWER query log -i splunk-not-success $TIME_RANGE --size 10

    print_test "Test 6: OR with comparison (ERROR OR latency > 1000) - context"
    run $LOGVIEWER query log -i splunk-critical-slow $TIME_RANGE --size 10

    print_test "Test 7: AND with negation (ERROR AND NOT payment-service) - context"
    run $LOGVIEWER query log -i splunk-errors-not-payment $TIME_RANGE --size 10

    # === CLI flag tests (-f with hl syntax) ===
    print_test "Test 8: -f flag with != operator (auto-detection)"
    run $LOGVIEWER query log -i splunk-errors-or-warns -f "level!=DEBUG" $TIME_RANGE --size 10

    print_test "Test 9: -f flag with >= operator (auto-detection)"
    run $LOGVIEWER query log -i splunk-errors-or-warns -f "latency_ms>=500" $TIME_RANGE --size 10

    print_test "Test 10: -f flag with ~= regex operator (auto-detection)"
    run $LOGVIEWER query log -i splunk-errors-or-warns -f "message~=.*error.*" $TIME_RANGE --size 10

    print_test "Test 11: Multiple -f flags combined (AND logic)"
    run $LOGVIEWER query log -i splunk-errors-or-warns -f "level!=DEBUG" -f "app=log-generator" $TIME_RANGE --size 10

    # === -q flag tests (complex expressions) ===
    print_test "Test 12: -q flag with simple condition"
    run $LOGVIEWER query log -i splunk-errors-or-warns -q "level=ERROR" $TIME_RANGE --size 10

    print_test "Test 13: -q flag with OR expression"
    run $LOGVIEWER query log -i splunk-errors-or-warns -q "level=ERROR OR level=WARN" $TIME_RANGE --size 10

    print_test "Test 14: -q flag with AND expression"
    run $LOGVIEWER query log -i splunk-errors-or-warns -q "level=ERROR AND app=log-generator" $TIME_RANGE --size 10

    print_test "Test 15: -q flag with NOT expression"
    run $LOGVIEWER query log -i splunk-errors-or-warns -q "NOT level=DEBUG" $TIME_RANGE --size 10

    print_test "Test 16: -q flag with parenthesized expression"
    run $LOGVIEWER query log -i splunk-errors-or-warns -q "(level=ERROR OR level=WARN) AND app=log-generator" $TIME_RANGE --size 10

    print_test "Test 17: -q flag with comparison operators"
    run $LOGVIEWER query log -i splunk-errors-or-warns -q "latency_ms>=500 AND latency_ms<2000" $TIME_RANGE --size 10

    print_test "Test 18: -q flag with exists function"
    run $LOGVIEWER query log -i splunk-errors-or-warns -q "exists(trace_id)" $TIME_RANGE --size 10

    print_test "Test 19: -q flag with exists and other conditions"
    run $LOGVIEWER query log -i splunk-errors-or-warns -q "exists(trace_id) AND level=ERROR" $TIME_RANGE --size 10

    print_test "Test 20: -q flag with symbolic operators (&& ||)"
    run $LOGVIEWER query log -i splunk-errors-or-warns -q "level=ERROR || level=WARN" $TIME_RANGE --size 10

    print_success "Splunk hl-compatible query tests completed"
}

test_opensearch() {
    print_header "OPENSEARCH HL-COMPATIBLE QUERY TESTS"

    # === Context-based tests (YAML config) ===
    print_test "Test 1: Greater than operator (latency > 1000ms) - context"
    run $LOGVIEWER query log -i opensearch-slow-requests $TIME_RANGE --size 10

    print_test "Test 2: Less than or equal operator (latency <= 200ms) - context"
    run $LOGVIEWER query log -i opensearch-latency-lte $TIME_RANGE --size 10

    print_test "Test 3: Negate field (level != INFO) - context"
    run $LOGVIEWER query log -i opensearch-not-info $TIME_RANGE --size 10

    # === CLI flag tests (-f with hl syntax) ===
    print_test "Test 4: -f flag with != operator (auto-detection)"
    run $LOGVIEWER query log -i opensearch-errors-or-warns -f "level!=DEBUG" $TIME_RANGE --size 10

    print_test "Test 5: -f flag with >= operator (auto-detection)"
    run $LOGVIEWER query log -i opensearch-errors-or-warns -f "latency_ms>=500" $TIME_RANGE --size 10

    print_test "Test 6: Multiple -f flags combined (AND logic)"
    run $LOGVIEWER query log -i opensearch-errors-or-warns -f "level!=DEBUG" -f "app=log-generator" $TIME_RANGE --size 10

    # === -q flag tests (complex expressions) ===
    print_test "Test 7: -q flag with OR expression"
    run $LOGVIEWER query log -i opensearch-errors-or-warns -q "level=ERROR OR level=WARN" $TIME_RANGE --size 10

    print_test "Test 8: -q flag with AND expression"
    run $LOGVIEWER query log -i opensearch-errors-or-warns -q "level=ERROR AND app=log-generator" $TIME_RANGE --size 10

    print_test "Test 9: -q flag with NOT expression"
    run $LOGVIEWER query log -i opensearch-errors-or-warns -q "NOT level=DEBUG" $TIME_RANGE --size 10

    print_test "Test 10: -q flag with parenthesized expression"
    run $LOGVIEWER query log -i opensearch-errors-or-warns -q "(level=ERROR OR level=WARN) AND app=log-generator" $TIME_RANGE --size 10

    print_test "Test 11: -q flag with comparison operators"
    run $LOGVIEWER query log -i opensearch-errors-or-warns -q "latency_ms>=500 AND latency_ms<2000" $TIME_RANGE --size 10

    print_test "Test 12: -q flag with exists function"
    run $LOGVIEWER query log -i opensearch-errors-or-warns -q "exists(trace_id)" $TIME_RANGE --size 10

    print_success "OpenSearch hl-compatible query tests completed"
}

run_backend_tests "HL-COMPATIBLE QUERY SYNTAX INTEGRATION TESTS" "${1:-all}" splunk opensearch
