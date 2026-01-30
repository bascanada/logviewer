//go:build integration

package e2e

import (
	"testing"
)

func TestQueryField_Splunk(t *testing.T) {
	t.Parallel()

	t.Run("BasicFieldDiscovery", func(t *testing.T) {
		logs := tCtx.RunAndParse(t,
			"query", "field",
			"-i", "splunk-all",
			"--last", "1h",
			"--size", "10",
		)
		Expect(t, logs).IsNotEmpty().AtMost(10)
		if len(logs) > 0 {
			Expect(t, logs).All(FieldPresent("fields"))
		}
	})

	t.Run("WithFiltering", func(t *testing.T) {
		logs := tCtx.RunAndParse(t,
			"query", "field",
			"-i", "splunk-all",
			"-f", "level=ERROR",
			"--last", "1h",
			"--size", "10",
		)
		if len(logs) > 0 {
			Expect(t, logs).All(
				FieldEquals("level", "ERROR"),
				FieldPresent("fields"),
			)
		}
	})
}

func TestQueryField_OpenSearch(t *testing.T) {
	t.Parallel()

	t.Run("BasicFieldDiscovery", func(t *testing.T) {
		logs := tCtx.RunAndParse(t,
			"query", "field",
			"-i", "opensearch-all",
			"--last", "1h",
			"--size", "10",
		)
		Expect(t, logs).IsNotEmpty().AtMost(10)
		if len(logs) > 0 {
			Expect(t, logs).All(FieldPresent("fields"))
		}
	})
}
