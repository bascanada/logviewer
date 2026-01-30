//go:build integration

package e2e

import (
	"testing"
)

func TestQueryValues_Splunk(t *testing.T) {
	t.Parallel()

	t.Run("SingleField", func(t *testing.T) {
		values := tCtx.RunAndParseValues(t,
			"query", "values",
			"-i", "splunk-all",
			"level",
			"--last", "1h",
		)
		ExpectValues(t, values).
			HasField("level").
			FieldHasValues("level", "INFO", "ERROR")
	})

	t.Run("MultipleFields", func(t *testing.T) {
		values := tCtx.RunAndParseValues(t,
			"query", "values",
			"-i", "splunk-all",
			"level", "app",
			"--last", "1h",
		)
		ExpectValues(t, values).HasFields("level", "app")
	})

	t.Run("WithFiltering", func(t *testing.T) {
		values := tCtx.RunAndParseValues(t,
			"query", "values",
			"-i", "splunk-all",
			"app",
			"-f", "level=ERROR",
			"--last", "1h",
		)
		ExpectValues(t, values).HasField("app")
		if appVals, ok := values["app"]; ok && len(appVals) > 0 {
			t.Logf("Apps with ERROR logs: %v", appVals)
		}
	})
}

func TestQueryValues_OpenSearch(t *testing.T) {
	t.Parallel()

	t.Run("SingleField", func(t *testing.T) {
		values := tCtx.RunAndParseValues(t,
			"query", "values",
			"-i", "opensearch-all",
			"level",
			"--last", "1h",
		)
		ExpectValues(t, values).HasField("level")
	})

	t.Run("MultipleFields", func(t *testing.T) {
		values := tCtx.RunAndParseValues(t,
			"query", "values",
			"-i", "opensearch-all",
			"level", "app",
			"--last", "1h",
		)
		ExpectValues(t, values).HasFields("level", "app")
	})
}

func TestQueryValues_MultiBackend(t *testing.T) {
	values := tCtx.RunAndParseValues(t,
		"query", "values",
		"-i", "splunk-all",
		"-i", "opensearch-all",
		"level",
		"--last", "15m",
	)
	ExpectValues(t, values).HasField("level")
	if levelVals, ok := values["level"]; ok {
		t.Logf("Combined backends found %d distinct log levels: %v", len(levelVals), levelVals)
	}
}
