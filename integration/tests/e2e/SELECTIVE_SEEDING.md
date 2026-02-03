# Selective Fixture Seeding

## Overview

The E2E test framework now supports **selective fixture seeding** to speed up test execution by only seeding the data needed for specific test suites.

## How It Works

The `TEST_FIXTURES` environment variable controls which fixtures are seeded before tests run:

| TEST_FIXTURES Value | Behavior | Use Case |
|---------------------|----------|----------|
| Not set (default) | Seeds all 6 fixtures | Running full test suite |
| `"error-logs,payment-logs"` | Seeds only specified fixtures | Running specific test subset |
| `""` (empty string) | Skips seeding entirely | Tests that don't need fixtures (e.g., SSH) |

## Fixture Map

| Fixture Name | Backend | Index | Size | Used By |
|--------------|---------|-------|------|---------|
| `error-logs` | Splunk | `test-e2e-errors` | 50 logs | Log queries, Native queries |
| `payment-logs` | Splunk | `test-e2e-payments` | 100 logs | Log queries, Field queries, Values queries |
| `order-logs` | OpenSearch | `test-e2e-orders` | 75 logs | Log queries, Field queries |
| `mixed-levels` | OpenSearch | `test-e2e-mixed` | 100 logs | Values queries, Filter tests |
| `trace-logs` | Splunk | `test-e2e-traces` | 30 logs | Trace ID tests |
| `slow-requests` | OpenSearch | `test-e2e-slow` | 25 logs | Latency tests |

## Makefile Targets

The Makefile has been updated to automatically use minimal fixtures for each test suite:

```bash
# Log query tests - seeds 3 fixtures (error-logs, payment-logs, order-logs)
make integration/test/log

# Field discovery tests - seeds 2 fixtures (payment-logs, order-logs)
make integration/test/field

# Values tests - seeds 3 fixtures (payment-logs, order-logs, mixed-levels)
make integration/test/values

# Native query tests - seeds 3 fixtures (error-logs, payment-logs, order-logs)
make integration/test/native

# HL syntax tests - seeds 3 fixtures (error-logs, payment-logs, order-logs)
make integration/test/hl

# SSH tests - skips seeding (uses separate SSH fixtures)
make integration/test/ssh

# Full test suite - seeds all 6 fixtures
make integration/test
```

## Manual Control

You can override the fixtures for any test run:

```bash
# Seed only specific fixtures
TEST_FIXTURES="error-logs,payment-logs" make integration/test/native

# Skip seeding entirely
TEST_FIXTURES="" go test -v -tags=integration ./integration/tests/e2e/... -run TestSSH

# Seed all fixtures (override Makefile default)
TEST_FIXTURES="error-logs,payment-logs,order-logs,mixed-levels,trace-logs,slow-requests" \
  make integration/test/log
```

## Performance Impact

**Time savings per test run:**

| Scenario | Fixtures Seeded | Validation Time | Time Saved |
|----------|----------------|-----------------|------------|
| Full suite (old) | 6 | ~6-8 seconds | Baseline |
| Native tests (new) | 3 | ~3-4 seconds | **~3-4 seconds** |
| Field tests (new) | 2 | ~2-3 seconds | **~4-5 seconds** |
| SSH tests (new) | 0 | 0 seconds | **~6-8 seconds** |

**Over 100 test runs during development:** ~5-8 minutes saved

## Implementation Details

### Environment Variable Handling

The `TestMain` function in `main_test.go` handles three cases:

```go
TEST_FIXTURES not set         → Seeds all fixtures (backward compatible)
TEST_FIXTURES="foo,bar"        → Seeds only specified fixtures
TEST_FIXTURES=""               → Skips seeding entirely
```

### Validation

Only the selected fixtures are validated:

```
Seeding selected fixtures: [error-logs payment-logs order-logs]
Validating seeded data is indexed and queryable...
  Checking error-logs (50 logs expected)...
    [1s] Found 50/50 logs (attempt #1)
    ✓ error-logs ready with 50 logs (took 1.6s)

  Checking payment-logs (100 logs expected)...
    [1s] Found 100/100 logs (attempt #1)
    ✓ payment-logs ready with 100 logs (took 1.1s)

  Checking order-logs (75 logs expected)...
    [0s] Found 75/75 logs (attempt #1)
    ✓ order-logs ready with 75 logs (took 0.0s)

✓ Successfully seeded and validated test data: map[error-logs:50 order-logs:75 payment-logs:100]
```

## Adding New Fixtures

When adding a new test fixture:

1. **Add to `AvailableFixtures`** in `test_data.go`:
   ```go
   "my-new-fixture": {
       Name:        "my-new-fixture",
       Description: "Description of fixture",
       Index:       "test-e2e-mynew",
       Count:       100,
   }
   ```

2. **Add to log-generator** `getTestFixtures()` in `main.go`

3. **Create the index** in Splunk/OpenSearch setup scripts

4. **Update Makefile targets** that need this fixture:
   ```makefile
   integration/test/mytest: build
       @TEST_FIXTURES="error-logs,my-new-fixture" go test ...
   ```

## Troubleshooting

### "Timeout waiting for fixture to be indexed"

If a test times out validating a fixture:
1. Check if fixture is in `TEST_FIXTURES` list
2. Verify log-generator seeded the data (check logs)
3. Confirm index exists in Splunk/OpenSearch
4. Try increasing validation timeout in `test_data.go`

### "Expected logs, but got none"

If tests can't find data:
1. Ensure correct fixtures are seeded for the test
2. Check if `run_id` filtering is working
3. Verify time range in queries (`--last 2h`)
4. Look at actual CLI query output with `DEBUG_E2E=true`

### Performance Still Slow

If tests are still slow despite selective seeding:
1. Check if you're seeding more fixtures than needed
2. Consider further splitting test files
3. Run with `-short` flag for development
4. Use cached data from previous runs (advanced)

## Future Improvements

Potential enhancements:

1. **Caching**: Reuse seeded data across test runs if run_id matches
2. **Parallel seeding**: Seed Splunk and OpenSearch fixtures in parallel
3. **Lazy seeding**: Seed fixtures on-demand when first accessed
4. **Test tagging**: Auto-detect required fixtures from test tags
5. **Shared state**: Allow tests to share seeded data within a suite

## Related Files

- `integration/tests/e2e/main_test.go` - TestMain with selective seeding logic
- `integration/tests/e2e/test_data.go` - Fixture definitions and seeding
- `Makefile` - Test targets with optimal fixture sets
- `integration/infra/log-generator/main.go` - Fixture generation
