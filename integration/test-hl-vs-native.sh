#!/bin/bash
# Comparative tests: hl vs native engine
# Verifies that both engines produce identical results for the same queries
#
# This script runs the same queries using both hl and native Go engine,
# then compares the output to ensure consistency.
#
# Usage: ./integration/test-hl-vs-native.sh [test_name]

set -e

source "$(dirname "${BASH_SOURCE[0]}")/test-common.sh"

LOG_DIR="/tmp/logviewer-hl-test"
LOG_FILE="$LOG_DIR/test-logs.json"
DIFF_DIR="$LOG_DIR/diffs"

# Generate test log file if needed
setup_test_logs() {
    print_header "SETUP: Generating Test Logs"

    mkdir -p "$LOG_DIR" "$DIFF_DIR"

    if [ -f "$LOG_FILE" ]; then
        echo "Using existing log file: $LOG_FILE"
        return
    fi

    echo "Generating 10000 test log entries..."

    # Generate logs with predictable content for comparison
    go run "$SCRIPT_DIR/benchmark/generate-logs.go" \
        -size 10000 \
        -output "$LOG_FILE" \
        -error 10 \
        -warn 15

    echo "Log file created: $LOG_FILE"
}

# Create temporary config for testing
# Args: $1=engine (hl|native)
create_config() {
    local engine=$1
    local config_file="$LOG_DIR/config-${engine}.yaml"

    local extra_opts=""
    if [ "$engine" = "native" ]; then
        extra_opts="preferNativeDriver: true"
    fi

    cat > "$config_file" << EOF
clients:
  test-local:
    type: local
    options:
      paths: ["$LOG_FILE"]
      cmd: "cat $LOG_FILE"
      $extra_opts

searches:
  json-extract:
    fieldExtraction:
      json: true
      jsonTimestampKey: "@timestamp"
      jsonLevelKey: "level"
      jsonMessageKey: "message"

contexts:
  test:
    client: test-local
    searchInherit: [json-extract]
    search:
      fields: {}
EOF

    echo "$config_file"
}

# Run query with specified engine and capture output
# Args: $1=engine, $2=test_name, $3...=query_args
run_query() {
    local engine=$1
    local test_name=$2
    shift 2
    local query_args="$@"

    local config=$(create_config "$engine")
    local output_file="$DIFF_DIR/${test_name}-${engine}.json"

    export LOGVIEWER_CONFIG="$config"

    # Run query and capture JSON output
    eval "$LOGVIEWER query log -i test --json $query_args" > "$output_file" 2>/dev/null || true

    # Sort output for comparison (timestamps may vary in order)
    sort "$output_file" > "${output_file}.sorted"

    echo "$output_file"
}

# Compare outputs from hl and native engines
# Args: $1=test_name
compare_outputs() {
    local test_name=$1
    local hl_file="$DIFF_DIR/${test_name}-hl.json.sorted"
    local native_file="$DIFF_DIR/${test_name}-native.json.sorted"

    if diff -q "$hl_file" "$native_file" > /dev/null 2>&1; then
        print_success "PASS: $test_name - outputs match"
        return 0
    else
        print_error "FAIL: $test_name - outputs differ"
        echo "  Diff saved to: $DIFF_DIR/${test_name}.diff"
        diff "$hl_file" "$native_file" > "$DIFF_DIR/${test_name}.diff" 2>&1 || true
        return 1
    fi
}

# Run a single comparative test
# Args: $1=test_name, $2...=query_args
run_comparative_test() {
    local test_name=$1
    shift
    local query_args="$@"

    print_test "Testing: $test_name"
    echo "  Query args: $query_args"

    # Check if hl is available
    if ! command -v hl &> /dev/null; then
        echo "  Skipping: hl not installed"
        return 0
    fi

    # Run with both engines
    echo "  Running with hl engine..."
    run_query "hl" "$test_name" "$query_args" > /dev/null

    echo "  Running with native engine..."
    run_query "native" "$test_name" "$query_args" > /dev/null

    # Compare results
    compare_outputs "$test_name"
}

# Test cases
test_simple_equals() {
    run_comparative_test "simple-equals" "-f level=ERROR" "--size 100"
}

test_not_equals() {
    run_comparative_test "not-equals" "-f 'level!=DEBUG'" "--size 100"
}

test_regex_match() {
    run_comparative_test "regex-match" "-f 'message~=.*error.*'" "--size 100"
}

test_comparison_gt() {
    run_comparative_test "comparison-gt" "-f 'latency_ms>1000'" "--size 100"
}

test_comparison_gte() {
    run_comparative_test "comparison-gte" "-f 'latency_ms>=500'" "--size 100"
}

test_comparison_lt() {
    run_comparative_test "comparison-lt" "-f 'latency_ms<100'" "--size 100"
}

test_or_logic() {
    run_comparative_test "or-logic" "-q 'level=ERROR OR level=WARN'" "--size 100"
}

test_and_logic() {
    run_comparative_test "and-logic" "-q 'level=ERROR AND app=api-gateway'" "--size 100"
}

test_not_logic() {
    run_comparative_test "not-logic" "-q 'NOT level=DEBUG'" "--size 100"
}

test_complex_nested() {
    run_comparative_test "complex-nested" "-q '(level=ERROR OR level=WARN) AND latency_ms>=500'" "--size 100"
}

test_time_range() {
    run_comparative_test "time-range" "--last 1h" "--size 100"
}

test_combined_filters() {
    run_comparative_test "combined-filters" "-f level=ERROR -f 'latency_ms>=1000'" "--size 100"
}

test_exists() {
    run_comparative_test "exists" "-q 'exists(trace_id)'" "--size 100"
}

test_all() {
    print_header "HL VS NATIVE COMPARATIVE TESTS"

    local passed=0
    local failed=0
    local skipped=0

    # Check hl availability first
    if ! command -v hl &> /dev/null; then
        echo -e "${YELLOW}WARNING: hl is not installed. Skipping all comparative tests.${NC}"
        echo "Install hl with: cargo install hl"
        echo "Or: brew install pamburus/tap/hl"
        exit 0
    fi

    setup_test_logs

    local tests=(
        test_simple_equals
        test_not_equals
        test_regex_match
        test_comparison_gt
        test_comparison_gte
        test_comparison_lt
        test_or_logic
        test_and_logic
        test_not_logic
        test_complex_nested
        test_time_range
        test_combined_filters
        test_exists
    )

    for test in "${tests[@]}"; do
        if $test; then
            ((passed++))
        else
            ((failed++))
        fi
    done

    echo ""
    print_header "TEST RESULTS"
    echo -e "  ${GREEN}Passed: $passed${NC}"
    if [ $failed -gt 0 ]; then
        echo -e "  ${RED}Failed: $failed${NC}"
    fi
    echo ""

    if [ $failed -gt 0 ]; then
        echo "Failed test diffs are in: $DIFF_DIR/"
        exit 1
    fi

    print_success "All comparative tests passed!"
}

# Main
ensure_binary

case "${1:-all}" in
    all) test_all ;;
    simple-equals) setup_test_logs; test_simple_equals ;;
    not-equals) setup_test_logs; test_not_equals ;;
    regex) setup_test_logs; test_regex_match ;;
    comparison) setup_test_logs; test_comparison_gt; test_comparison_gte; test_comparison_lt ;;
    logic) setup_test_logs; test_or_logic; test_and_logic; test_not_logic ;;
    complex) setup_test_logs; test_complex_nested ;;
    *)
        echo "Unknown test: $1"
        echo "Available: all, simple-equals, not-equals, regex, comparison, logic, complex"
        exit 1
        ;;
esac
