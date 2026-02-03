//go:build integration

package e2e

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestQueryField_Splunk(t *testing.T) {
	t.Parallel()

	t.Run("BasicFieldDiscovery", func(t *testing.T) {
		stdout, stderr := tCtx.RunAndExpectSuccess(t,
			"query", "field",
			"-i", "test-splunk-payments",
			"--last", "2h",
			"--json",
		)

		if stdout == "" {
			t.Logf("Stderr: %s", stderr)
			t.Fatal("Expected field output, got empty string")
		}

		fields := ParseFieldsJSON(stdout)
		assert.NotEmpty(t, fields, "Should discover fields from Splunk logs")

		// Splunk field discovery returns metadata fields (not extracted JSON fields)
		// The actual log content fields (level, app, etc.) are available in query results
		// but not in the field discovery API
		assert.Contains(t, fields, "index", "Should discover index field")
		assert.Contains(t, fields, "sourcetype", "Should discover sourcetype field")
	})

	t.Run("WithFiltering", func(t *testing.T) {
		stdout, _ := tCtx.RunAndExpectSuccess(t,
			"query", "field",
			"-i", "test-splunk-payments",
			"-f", "level=ERROR",
			"--last", "2h",
			"--json",
		)

		fields := ParseFieldsJSON(stdout)
		// Should still discover fields even with filtering
		assert.NotEmpty(t, fields, "Should discover fields from filtered Splunk logs")
	})
}

func TestQueryField_OpenSearch(t *testing.T) {
	t.Parallel()

	t.Run("BasicFieldDiscovery", func(t *testing.T) {
		stdout, stderr := tCtx.RunAndExpectSuccess(t,
			"query", "field",
			"-i", "test-opensearch-orders",
			"--last", "2h",
			"--json",
		)

		if stdout == "" {
			t.Logf("Stderr: %s", stderr)
			t.Fatal("Expected field output, got empty string")
		}

		fields := ParseFieldsJSON(stdout)
		assert.NotEmpty(t, fields, "Should discover fields from OpenSearch logs")

		// Seeded order logs should have these fields
		assert.Contains(t, fields, "level", "Should discover level field")
		assert.Contains(t, fields, "app", "Should discover app field")
		// Note: message field is treated specially and may not appear in field discovery
	})

	t.Run("WithFiltering", func(t *testing.T) {
		stdout, stderr, err := tCtx.Run(t,
			"query", "field",
			"-i", "test-opensearch-orders",
			"-f", "level=INFO",
			"--last", "2h",
			"--json",
		)

		if err != nil {
			fmt.Printf("Command failed - stdout: %s, stderr: %s, error: %v\n", stdout, stderr, err)
		}
		assert.NoError(t, err, "Field discovery with filter should execute successfully")

		fields := ParseFieldsJSON(stdout)
		// Should still discover fields even with filtering
		assert.NotEmpty(t, fields, "Should discover fields from filtered OpenSearch logs")
	})
}
