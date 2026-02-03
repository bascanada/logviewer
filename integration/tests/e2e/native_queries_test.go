//go:build integration

package e2e

import (
	"testing"
)

func TestNativeQueries_Splunk(t *testing.T) {
	t.Parallel()

	t.Run("BasicNativeQuery", func(t *testing.T) {
		logs := tCtx.RunAndParse(t,
			"query", "log",
			"-i", "splunk-native-query",
			"--last", "1h",
			"--size", "10",
		)
		Expect(t, logs).AtMost(10)
	})

	t.Run("ErrorLevelQuery", func(t *testing.T) {
		logs := tCtx.RunAndParse(t,
			"query", "log",
			"-i", "splunk-native-query",
			"-f", "level=ERROR",
			"--last", "1h",
			"--size", "10",
		)
		if len(logs) > 0 {
			Expect(t, logs).All(IsErrorLevel())
		}
	})
}

func TestNativeQueries_OpenSearch(t *testing.T) {
	t.Parallel()

	t.Run("BasicNativeQuery", func(t *testing.T) {
		logs := tCtx.RunAndParse(t,
			"query", "log",
			"-i", "opensearch-native-query",
			"--last", "1h",
			"--size", "10",
		)
		Expect(t, logs).AtMost(10)
	})
}
