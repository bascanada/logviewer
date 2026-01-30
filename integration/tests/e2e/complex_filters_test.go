//go:build integration

package e2e

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestComplexFilters tests complex nested AND/OR/NOT filter combinations
func TestComplexFilters(t *testing.T) {
	t.Parallel()

	t.Run("NestedAndOr_Splunk", func(t *testing.T) {
		// Test: (level=ERROR OR level=WARN) AND app=payment-service
		output, err := RunCommand("query", "log", "-i", "splunk-all", "--last", "1h",
			"-q", "(level=ERROR OR level=WARN) AND app=payment-service",
			"--size", "20", "--json")
		assert.NoError(t, err, "Complex nested filter should execute successfully")

		entries := ParseJSONOutput(output)
		assert.NotEmpty(t, entries, "Should find logs matching complex filter")

		// Verify each entry matches the filter
		for _, entry := range entries {
			level := GetFieldValue(entry, "level")
			app := GetFieldValue(entry, "app")

			assert.Contains(t, []string{"ERROR", "WARN"}, level, "Level should be ERROR or WARN")
			assert.NotEmpty(t, app, "App field should be present")
		}
	})

	t.Run("NestedAndOr_OpenSearch", func(t *testing.T) {
		// Test: (level=ERROR OR level=WARN) AND (app=order-service OR app=api-gateway)
		output, err := RunCommand("query", "log", "-i", "opensearch-all", "--last", "1h",
			"-q", "(level=ERROR OR level=WARN) AND (app=order-service OR app=api-gateway)",
			"--size", "20", "--json")
		assert.NoError(t, err, "Complex nested filter should execute successfully")

		entries := ParseJSONOutput(output)
		assert.NotEmpty(t, entries, "Should find logs matching complex filter")

		for _, entry := range entries {
			level := GetFieldValue(entry, "level")
			assert.Contains(t, []string{"ERROR", "WARN"}, level, "Level should be ERROR or WARN")
		}
	})

	t.Run("NotWithAnd", func(t *testing.T) {
		// Test: level=ERROR AND NOT app=payment-service
		output, err := RunCommand("query", "log", "-i", "splunk-all", "--last", "1h",
			"-q", "level=ERROR AND NOT app=payment-service",
			"--size", "20", "--json")
		assert.NoError(t, err, "NOT with AND should execute successfully")

		entries := ParseJSONOutput(output)
		// Verify all entries have level=ERROR
		if len(entries) > 0 {
			Expect(t, entries).All(FieldEquals("level", "ERROR"))
			// Verify none of the results have app=payment-service
			for _, entry := range entries {
				app := GetFieldValue(entry, "app")
				if app != "" {
					assert.NotEqual(t, "payment-service", app, "Should exclude payment-service")
				}
			}
		}
	})
}

// TestExistsOperator tests the EXISTS operator for field presence
func TestExistsOperator(t *testing.T) {
	t.Parallel()

	t.Run("FieldExists_Splunk", func(t *testing.T) {
		// Test: exists(trace_id)
		output, err := RunCommand("query", "log", "-i", "splunk-all", "--last", "1h",
			"-q", "exists(trace_id)",
			"--size", "20", "--json")
		assert.NoError(t, err, "EXISTS operator should execute successfully")

		entries := ParseJSONOutput(output)
		// Verify all entries have trace_id field
		for _, entry := range entries {
			traceID := GetFieldValue(entry, "trace_id")
			assert.NotEmpty(t, traceID, "trace_id field should exist and not be empty")
		}
	})

	t.Run("FieldExists_OpenSearch", func(t *testing.T) {
		// Test: exists(trace_id) AND level=ERROR
		output, err := RunCommand("query", "log", "-i", "opensearch-all", "--last", "1h",
			"-q", "exists(trace_id) AND level=ERROR",
			"--size", "20", "--json")
		assert.NoError(t, err, "EXISTS with AND should execute successfully")

		entries := ParseJSONOutput(output)
		if len(entries) > 0 {
			Expect(t, entries).All(FieldEquals("level", "ERROR"))
			for _, entry := range entries {
				traceID := GetFieldValue(entry, "trace_id")
				assert.NotEmpty(t, traceID, "trace_id should exist")
			}
		}
	})
}

// TestComparisonOperators tests all comparison operators (>, >=, <, <=)
func TestComparisonOperators(t *testing.T) {
	t.Parallel()

	t.Run("GreaterThan", func(t *testing.T) {
		// Test: latency_ms > 1000
		output, err := RunCommand("query", "log", "-i", "splunk-all", "--last", "1h",
			"-f", "latency_ms>1000",
			"--size", "20", "--json")
		assert.NoError(t, err, "Greater than operator should execute successfully")

		entries := ParseJSONOutput(output)
		// Verify all entries have latency_ms > 1000
		for _, entry := range entries {
			latency := GetFieldValueAsInt(entry, "latency_ms")
			if latency > 0 {
				assert.Greater(t, latency, 1000, "latency_ms should be > 1000")
			}
		}
	})

	t.Run("GreaterThanOrEqual", func(t *testing.T) {
		// Test: latency_ms >= 500
		output, err := RunCommand("query", "log", "-i", "opensearch-all", "--last", "1h",
			"-f", "latency_ms>=500",
			"--size", "20", "--json")
		assert.NoError(t, err, "Greater than or equal operator should execute successfully")

		entries := ParseJSONOutput(output)
		for _, entry := range entries {
			latency := GetFieldValueAsInt(entry, "latency_ms")
			if latency > 0 {
				assert.GreaterOrEqual(t, latency, 500, "latency_ms should be >= 500")
			}
		}
	})
}
