//go:build integration

package e2e

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestK8s_QueryLog tests basic K8s log querying with label selectors
func TestK8s_QueryLog(t *testing.T) {
	t.Parallel()

	t.Run("BasicQuery", func(t *testing.T) {
		output, err := RunCommand("query", "log", "-i", "payment-processor-all", "--last", "1h", "--size", "10", "--json")
		if err != nil {
			fmt.Printf("Command Output: %s\n", output)
		}
		assert.NoError(t, err, "Query should execute successfully")

		entries := ParseJSONOutput(output)
		assert.NotEmpty(t, entries, "Should return some log entries")
		assert.LessOrEqual(t, len(entries), 10, "Should respect size limit")
	})

	t.Run("WithFilters", func(t *testing.T) {
		output, err := RunCommand("query", "log", "-i", "payment-processor-all", "--last", "1h", "-f", "level=ERROR", "--size", "10", "--json")
		if err != nil {
			fmt.Printf("Command Output: %s\n", output)
		}
		assert.NoError(t, err, "Query with filter should execute successfully")

		entries := ParseJSONOutput(output)
		if len(entries) > 0 {
			Expect(t, entries).All(FieldContains("level", "ERROR"))
		}
	})

	t.Run("TimeRange", func(t *testing.T) {
		output, err := RunCommand("query", "log", "-i", "payment-processor-all", "--last", "30m", "--size", "5", "--json")
		if err != nil {
			fmt.Printf("Command Output: %s\n", output)
		}
		assert.NoError(t, err, "Query with time range should execute successfully")

		entries := ParseJSONOutput(output)
		assert.NotEmpty(t, entries, "Should return logs from last 30 minutes")
	})
}

// TestK8s_QueryField tests field discovery in K8s logs
func TestK8s_QueryField(t *testing.T) {
	t.Parallel()

	t.Run("BasicFieldDiscovery", func(t *testing.T) {
		output, err := RunCommand("query", "field", "-i", "payment-processor-all", "--last", "1h", "--json")
		if err != nil {
			fmt.Printf("Command Output: %s\n", output)
		}
		assert.NoError(t, err, "Field discovery should execute successfully")

		fields := ParseFieldsJSON(output)
		assert.NotEmpty(t, fields, "Should discover fields from K8s logs")

		// K8s logs should have at least level field extracted
		assert.Contains(t, fields, "level", "Should discover level field")
	})

	t.Run("WithFiltering", func(t *testing.T) {
		output, err := RunCommand("query", "field", "-i", "payment-processor-all", "--last", "1h", "-f", "level=ERROR", "--json")
		if err != nil {
			fmt.Printf("Command Output: %s\n", output)
		}
		assert.NoError(t, err, "Field discovery with filter should execute successfully")

		fields := ParseFieldsJSON(output)
		assert.NotEmpty(t, fields, "Should discover fields from filtered K8s logs")
	})
}

// TestK8s_QueryValues tests value extraction from K8s logs
func TestK8s_QueryValues(t *testing.T) {
	t.Parallel()

	t.Run("SingleField", func(t *testing.T) {
		output, err := RunCommand("query", "values", "level", "-i", "payment-processor-all", "--last", "1h", "--json")
		if err != nil {
			fmt.Printf("Command Output: %s\n", output)
		}
		assert.NoError(t, err, "Values query should execute successfully")

		values := ParseValuesJSON(output)
		assert.NotEmpty(t, values, "Should extract distinct level values")

		// Should have common log levels
		levelValues := values["level"]
		assert.NotEmpty(t, levelValues, "Should have level values")
	})

	t.Run("MultipleFields", func(t *testing.T) {
		output, err := RunCommand("query", "values", "level", "app", "-i", "payment-processor-all", "--last", "1h", "--json")
		if err != nil {
			fmt.Printf("Command Output: %s\n", output)
		}
		assert.NoError(t, err, "Multi-field values query should execute successfully")

		values := ParseValuesJSON(output)
		assert.Contains(t, values, "level", "Should have level values")
		// app field may or may not be present depending on K8s log format
	})

	t.Run("WithFiltering", func(t *testing.T) {
		output, err := RunCommand("query", "values", "app", "-i", "payment-processor-all", "--last", "1h", "-f", "level=ERROR", "--json")
		if err != nil {
			fmt.Printf("Command Output: %s\n", output)
		}
		assert.NoError(t, err, "Values query with filter should execute successfully")

		values := ParseValuesJSON(output)
		// Should return values only from ERROR level logs
		assert.NotNil(t, values)
	})
}
