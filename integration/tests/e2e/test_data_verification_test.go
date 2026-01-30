//go:build integration

package e2e

import (
	"testing"
)

// TestDataSeeding_ErrorLogs verifies the error-logs fixture was seeded correctly
func TestDataSeeding_ErrorLogs(t *testing.T) {
	t.Parallel()

	fixture, err := GetFixtureInfo("error-logs")
	if err != nil {
		t.Fatalf("Failed to get fixture info: %v", err)
	}

	logs := tCtx.RunAndParse(t,
		"query", "log",
		"-i", "test-splunk-errors",
		"--size", "100", // Request more than the 50 we seeded
	)

	// Verify we got exactly the expected count
	Expect(t, logs).
		Count(fixture.Count). // Exactly 50 logs
		All(
			IsErrorLevel(),
			FieldPresent("trace_id"),
			FieldEquals("app", "payment-service"),
			FieldPresent("latency_ms"),
		)
}

// TestDataSeeding_PaymentLogs verifies the payment-logs fixture was seeded correctly
func TestDataSeeding_PaymentLogs(t *testing.T) {
	t.Parallel()

	fixture, err := GetFixtureInfo("payment-logs")
	if err != nil {
		t.Fatalf("Failed to get fixture info: %v", err)
	}

	logs := tCtx.RunAndParse(t,
		"query", "log",
		"-i", "test-splunk-payments",
		"--size", "150", // Request more than the 100 we seeded
	)

	// Verify we got exactly the expected count
	Expect(t, logs).
		Count(fixture.Count). // Exactly 100 logs
		All(
			FieldEquals("app", "payment-service"),
			FieldPresent("trace_id"),
		)

	// Verify we have both INFO and ERROR levels (80 INFO, 20 ERROR)
	errorCount := 0
	infoCount := 0
	for _, log := range logs {
		level := GetFieldValue(log, "level")
		if level == "ERROR" {
			errorCount++
		} else if level == "INFO" {
			infoCount++
		}
	}

	if errorCount != 20 {
		t.Errorf("Expected 20 ERROR logs, got %d", errorCount)
	}
	if infoCount != 80 {
		t.Errorf("Expected 80 INFO logs, got %d", infoCount)
	}
}

// TestDataSeeding_MixedLevels verifies the mixed-levels fixture was seeded correctly
func TestDataSeeding_MixedLevels(t *testing.T) {
	t.Parallel()

	fixture, err := GetFixtureInfo("mixed-levels")
	if err != nil {
		t.Fatalf("Failed to get fixture info: %v", err)
	}

	logs := tCtx.RunAndParse(t,
		"query", "log",
		"-i", "test-opensearch-mixed",
		"--size", "150",
	)

	// Verify we got exactly the expected count
	Expect(t, logs).
		Count(fixture.Count). // Exactly 100 logs
		All(
			FieldEquals("app", "api-gateway"),
			FieldPresent("trace_id"),
		)

	// Verify level distribution: 50 INFO, 30 WARN, 20 ERROR
	levelCounts := make(map[string]int)
	for _, log := range logs {
		level := GetFieldValue(log, "level")
		levelCounts[level]++
	}

	if levelCounts["INFO"] != 50 {
		t.Errorf("Expected 50 INFO logs, got %d", levelCounts["INFO"])
	}
	if levelCounts["WARN"] != 30 {
		t.Errorf("Expected 30 WARN logs, got %d", levelCounts["WARN"])
	}
	if levelCounts["ERROR"] != 20 {
		t.Errorf("Expected 20 ERROR logs, got %d", levelCounts["ERROR"])
	}
}

// TestDataSeeding_TraceLogs verifies the trace-logs fixture was seeded correctly
func TestDataSeeding_TraceLogs(t *testing.T) {
	t.Parallel()

	fixture, err := GetFixtureInfo("trace-logs")
	if err != nil {
		t.Fatalf("Failed to get fixture info: %v", err)
	}

	logs := tCtx.RunAndParse(t,
		"query", "log",
		"-i", "test-splunk-traces",
		"--size", "50",
	)

	// Verify we got exactly the expected count
	Expect(t, logs).
		Count(fixture.Count). // Exactly 30 logs
		All(
			FieldPresent("trace_id"),
			FieldEquals("app", "api-gateway"),
			FieldEquals("level", "INFO"),
		)

	// Verify all trace_ids follow the expected pattern
	for i, log := range logs {
		traceID := GetFieldValue(log, "trace_id")
		if traceID == "" {
			t.Errorf("Log #%d missing trace_id", i)
		}
		// Trace IDs should match pattern: trace-distributed-XXX
		if len(traceID) < 10 {
			t.Errorf("Log #%d has invalid trace_id format: %s", i, traceID)
		}
	}
}

// TestDataSeeding_SlowRequests verifies the slow-requests fixture was seeded correctly
func TestDataSeeding_SlowRequests(t *testing.T) {
	t.Parallel()

	fixture, err := GetFixtureInfo("slow-requests")
	if err != nil {
		t.Fatalf("Failed to get fixture info: %v", err)
	}

	logs := tCtx.RunAndParse(t,
		"query", "log",
		"-i", "test-opensearch-slow",
		"--size", "50",
	)

	// Verify we got exactly the expected count
	Expect(t, logs).
		Count(fixture.Count). // Exactly 25 logs
		All(
			FieldEquals("level", "WARN"),
			FieldEquals("app", "api-gateway"),
			FieldPresent("latency_ms"),
		)

	// Verify all logs have latency > 1000ms
	for i, log := range logs {
		latency := GetFieldValueAsInt(log, "latency_ms")
		if latency < 1000 {
			t.Errorf("Log #%d has latency %d ms, expected >= 1000ms", i, latency)
		}
	}
}

// TestDataSeeding_FieldValues verifies field value queries work with seeded data
func TestDataSeeding_FieldValues(t *testing.T) {
	t.Parallel()

	// Query values from error logs fixture
	values := tCtx.RunAndParseValues(t,
		"query", "values",
		"-i", "test-splunk-errors",
		"level", "app",
	)

	ExpectValues(t, values).
		HasFields("level", "app").
		FieldHasExactValues("level", "ERROR"). // All should be ERROR
		FieldHasExactValues("app", "payment-service") // All should be payment-service
}
