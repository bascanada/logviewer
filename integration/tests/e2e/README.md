# Go-Based Integration Tests

This directory contains Go-based end-to-end integration tests for the logviewer CLI. These tests replace the legacy shell-based tests with type-safe, structured assertions and parallel execution capabilities.

## Overview

The test framework provides:

- **Strong Typing**: Tests work with structured Go types, not raw text output
- **Fluent Assertions**: Readable, chainable assertions like `Expect(t, logs).AtLeast(5).All(IsErrorLevel())`
- **Custom Checks**: Domain-specific validations like `HasTraceID()`, `DateBetween()`, `FieldNotPresent()`
- **Parallel Execution**: Tests run concurrently for faster feedback
- **JSON Parsing**: Handles both JSON array and NDJSON (newline-delimited) formats automatically

## Directory Structure

```
integration/tests/e2e/
├── main_test.go           # TestMain setup, binary building, config resolution
├── helpers_test.go        # TestContext, command execution, JSON parsing
├── assertions.go          # Fluent assertion library with LogCheck interface
├── query_log_test.go      # Log querying tests (Splunk, OpenSearch, K8s, multi-backend)
├── query_field_test.go    # Field discovery tests
├── query_values_test.go   # Field value aggregation tests
├── native_queries_test.go # Native SPL/Lucene query tests
├── hl_queries_test.go     # HL-compatible query syntax tests
└── ssh_test.go            # SSH backend tests
```

## Prerequisites

1. **Build the binary**: `make build`
2. **Start Docker services**: `make integration/start`
3. **Wait for services to be ready** (Splunk can take 2-3 minutes)

## Running Tests

### All Integration Tests
```bash
make integration/test
```

### Specific Test Suites
```bash
# Run only log query tests
make integration/test/log

# Run only field query tests
make integration/test/field

# Run only values query tests
make integration/test/values

# Run only SSH backend tests
make integration/test/ssh

# Run only native query tests
make integration/test/native

# Run only HL query tests
make integration/test/hl

# Quick smoke test (short mode)
make integration/test/short
```

### Using Go Test Directly
```bash
# Run all integration tests
go test -v -tags=integration ./integration/tests/e2e/...

# Run specific test function
go test -v -tags=integration ./integration/tests/e2e/... -run TestQueryLog_Splunk

# Run tests matching pattern
go test -v -tags=integration ./integration/tests/e2e/... -run TestQueryLog

# Run with timeout
go test -v -tags=integration -timeout 30m ./integration/tests/e2e/...

# Run in parallel (default is GOMAXPROCS)
go test -v -tags=integration -parallel 4 ./integration/tests/e2e/...

# Run with verbose assertion logging (shows what's being validated)
VERBOSE_ASSERTIONS=1 go test -v -tags=integration ./integration/tests/e2e/...
```

## Verbose Assertion Logging

To see exactly what each assertion is validating (useful for debugging and building trust in tests), enable verbose mode:

### Global Verbose Mode (all tests)
```bash
VERBOSE_ASSERTIONS=1 go test -v -tags=integration ./integration/tests/e2e/...
```

### Per-Test Verbose Mode
```go
func TestMyFeature(t *testing.T) {
    logs := tCtx.RunAndParse(t, "query", "log", "-c", "splunk")
    
    // Enable verbose logging for this assertion chain
    Expect(t, logs).
        Verbose().  // Show what's being validated
        AtLeast(5).
        All(IsErrorLevel(), FieldContains("message", "timeout"))
}
```

### Example Verbose Output
```
=== RUN   TestQueryLog_Splunk
    assertions.go:45: ✓ Asserting at least: min=10, actual=47
    assertions.go:92: ✓ Asserting all 47 logs match checks: field 'level' equals 'ERROR', field 'message' contains 'timeout'
    assertions.go:106:   ✓ All 47 logs passed all checks
--- PASS: TestQueryLog_Splunk (2.14s)
```

Verbose mode shows:
- What assertions are being performed
- Expected vs actual values
- Number of logs/checks being validated
- Success confirmations with ✓ markers

## Writing Tests

### Basic Test Structure

```go
//go:build integration

package e2e

import "testing"

func TestMyFeature(t *testing.T) {
    t.Parallel() // Allow parallel execution
    
    logs := tCtx.RunAndParse(t,
        "query", "log",
        "-i", "splunk-all",
        "--last", "1h",
        "--size", "10",
    )
    
    Expect(t, logs).
        IsNotEmpty().
        AtMost(10).
        All(FieldPresent("timestamp"))
}
```

### Using Fluent Assertions

#### Collection Assertions
```go
Expect(t, logs).Count(10)                    // Exactly 10 logs
Expect(t, logs).AtLeast(5)                   // At least 5 logs
Expect(t, logs).AtMost(20)                   // At most 20 logs
Expect(t, logs).Between(5, 15)               // Between 5 and 15 logs
Expect(t, logs).IsEmpty()                    // No logs returned
Expect(t, logs).IsNotEmpty()                 // At least one log
```

#### Scoped Assertions
```go
// Every log must satisfy these checks
Expect(t, logs).All(
    FieldEquals("level", "ERROR"),
    FieldPresent("trace_id"),
    DateAfter("timestamp", time.Now().Add(-1*time.Hour)),
)

// At least one log must satisfy all these checks
Expect(t, logs).Any(
    FieldContains("message", "timeout"),
    IsErrorLevel(),
)

// First log must satisfy these checks
Expect(t, logs).First(
    FieldEquals("app", "payment-service"),
)

// Last log must satisfy these checks
Expect(t, logs).Last(
    FieldPresent("trace_id"),
)

// No log should satisfy all these checks
Expect(t, logs).None(
    FieldEquals("level", "DEBUG"),
    FieldContains("message", "secret"),
)
```

### Available LogCheck Functions

#### Field Presence
```go
FieldPresent("field_name")           // Field exists (even if null)
FieldNotPresent("field_name")        // Field is completely missing
FieldNotEmpty("field_name")          // Field exists and is not empty string
```

#### Field Value Matching
```go
FieldEquals("level", "ERROR")                    // Exact match
FieldContains("message", "timeout")              // Substring match
FieldMatches("app", "*-service")                 // Wildcard pattern
FieldOneOf("level", "ERROR", "WARN", "INFO")    // Match any of these
```

#### Date/Time Checks
```go
DateBetween("timestamp", startTime, endTime)     // Within range
DateAfter("timestamp", time.Now().Add(-1*time.Hour))   // After time
DateBefore("timestamp", time.Now())               // Before time
```

#### Domain-Specific Checks
```go
HasTraceID()                         // Log has non-empty trace_id
IsErrorLevel()                       // level == "ERROR"
IsFromApp("payment-service")         // app == "payment-service"
IsFromContext("splunk-all")          // context_id == "splunk-all"
HasAllFields("field1", "field2")     // Multiple fields present
```

### Testing Field Values

```go
values := tCtx.RunAndParseValues(t,
    "query", "values",
    "-i", "splunk-all",
    "level", "app",
    "--last", "1h",
)

ExpectValues(t, values).
    HasFields("level", "app").
    FieldHasValues("level", "INFO", "ERROR").
    FieldValueCount("level", 4)
```

## Custom Assertions

You can create your own LogCheck functions:

```go
func HasValidTraceID() LogCheck {
    return simpleCheck{
        validator: func(log map[string]interface{}) bool {
            val, ok := log["trace_id"]
            if !ok {
                return false
            }
            traceID, isStr := val.(string)
            return isStr && len(traceID) == 32 // Example: 32-char hex
        },
        desc: "has valid 32-character trace ID",
    }
}

// Usage
Expect(t, logs).All(HasValidTraceID())
```

## Test Execution Flow

1. **TestMain** (in `main_test.go`):
   - Resolves binary path (`build/logviewer`)
   - Combines config files (`config.yaml` + `config.hl.yaml`)
   - Builds binary if missing
   - Initializes global `tCtx` (TestContext)

2. **Individual Tests**:
   - Use `tCtx.Run()` to execute binary with args
   - Use `tCtx.RunAndParse()` to get structured JSON logs
   - Use `Expect()` to make fluent assertions
   - Tests marked with `t.Parallel()` run concurrently

3. **Environment**:
   - Binary reads `LOGVIEWER_CONFIG` env var (set automatically)
   - Config points to `integration/infra/config.yaml:integration/infra/config.hl.yaml`
   - Tests assume Docker services are running

## Debugging Tests

### Verbose Output
```bash
go test -v -tags=integration ./integration/tests/e2e/... -run TestQueryLog_Splunk
```

### See Command Output
```bash
# Add logging in test:
stdout, stderr := tCtx.RunAndExpectSuccess(t, "query", "log", "-i", "splunk-all")
t.Logf("stdout: %s", stdout)
t.Logf("stderr: %s", stderr)
```

### Run Single Test
```bash
go test -v -tags=integration ./integration/tests/e2e/... -run TestQueryLog_Splunk/BasicQuery
```

### Skip Slow Tests
```bash
go test -v -tags=integration -short ./integration/tests/e2e/...
```

## Comparison with Legacy Shell Tests

| Feature | Legacy (Shell) | New (Go) |
|---------|---------------|----------|
| **Type Safety** | ❌ String manipulation | ✅ Strongly typed |
| **Assertions** | ❌ Exit codes only | ✅ Structured validation |
| **JSON Parsing** | ❌ Manual jq/grep | ✅ Automatic |
| **Parallel Execution** | ❌ Sequential | ✅ Concurrent |
| **Field Presence** | ❌ Can't distinguish missing vs empty | ✅ Precise detection |
| **Readability** | ⚠️ Bash verbosity | ✅ Fluent API |
| **IDE Support** | ❌ Limited | ✅ Full autocomplete |
| **Debugging** | ⚠️ Echo statements | ✅ Go debugger |

## Migration Status

✅ Implemented:
- Query log tests (Splunk, OpenSearch, K8s, multi-backend)
- Query field tests
- Query values tests
- Native query tests (SPL, Lucene)
- HL query syntax tests
- SSH backend tests
- Fluent assertion library
- Parallel execution support

📝 Future enhancements:
- Docker backend tests
- CloudWatch backend tests
- Performance benchmarking
- Coverage reporting integration
- CI/CD pipeline integration

## Contributing

When adding new tests:

1. Use `//go:build integration` build tag
2. Add `t.Parallel()` if test is independent
3. Use fluent assertions for readability
4. Create custom `LogCheck` functions for reusable validations
5. Document expected behavior in test name
6. Add new test files to `integration/test/*` Makefile targets

## Legacy Tests

Legacy shell-based tests remain in `integration/legacy/` and can still be run:

```bash
# Run legacy tests
make integration/tests

# Or directly
bash integration/legacy/test-all.sh
```

These will be deprecated once full migration to Go tests is complete.
