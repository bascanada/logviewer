#!/bin/bash
# Master test script that runs all integration tests
# Usage: ./integration/test-all.sh [test-suite]
# Test suites: log, field, values, filters, native, hl, all (default)

source "$(dirname "${BASH_SOURCE[0]}")/test-common.sh"

# Track test results
TESTS_PASSED=0
TESTS_FAILED=0
FAILED_TESTS=""

run_test() {
    local script="$1"
    local name="$2"

    print_major_header "Running: $name"

    if "$SCRIPT_DIR/$script" all; then
        TESTS_PASSED=$((TESTS_PASSED + 1))
        print_success "$name - PASSED"
    else
        TESTS_FAILED=$((TESTS_FAILED + 1))
        FAILED_TESTS="$FAILED_TESTS\n  - $name"
        print_error "$name - FAILED"
    fi
}

test_log()     { run_test "test-query-log.sh" "Query Log Tests"; }
test_field()   { run_test "test-query-field.sh" "Query Field Tests"; }
test_values()  { run_test "test-query-values.sh" "Query Values Tests"; }
test_filters() { run_test "test-recursive-filters.sh" "Recursive Filter Tests"; }
test_native()  { run_test "test-native-queries.sh" "Native Query Tests"; }
test_hl()      { run_test "test-hl-queries.sh" "HL-Compatible Query Tests"; }

print_summary() {
    print_major_header "TEST SUMMARY"
    echo -e "  Passed: ${GREEN}$TESTS_PASSED${NC}"
    echo -e "  Failed: ${RED}$TESTS_FAILED${NC}"

    if [ $TESTS_FAILED -gt 0 ]; then
        echo ""
        echo -e "${RED}Failed tests:${NC}"
        echo -e "$FAILED_TESTS"
        exit 1
    else
        echo ""
        print_success "All tests passed!"
    fi
}

main() {
    local suite="${1:-all}"

    ensure_binary
    print_major_header "LOGVIEWER INTEGRATION TEST SUITE"
    echo "Config: $LOGVIEWER_CONFIG"
    echo "Binary: $LOGVIEWER"
    echo "Test Suite: $suite"

    case "$suite" in
        log)     test_log ;;
        field)   test_field ;;
        values)  test_values ;;
        filters) test_filters ;;
        native)  test_native ;;
        hl)      test_hl ;;
        all)
            test_log
            test_field
            test_values
            test_filters
            test_native
            test_hl
            ;;
        *)
            echo "Unknown test suite: $suite"
            echo "Usage: $0 [log|field|values|filters|native|hl|all]"
            exit 1
            ;;
    esac

    print_summary
}

main "$@"
