//go:build integration

package e2e

import (
	"testing"
	"time"
)

func TestQueryLog_Splunk(t *testing.T) {
	t.Parallel()

	t.Run("BasicQuery", func(t *testing.T) {
		logs := tCtx.RunAndParse(t,
			"query", "log",
			"-i", "splunk-all",
			"--last", "1h",
			"--size", "10",
		)
		Expect(t, logs).IsNotEmpty().AtMost(10)
	})

	t.Run("PaymentService", func(t *testing.T) {
		logs := tCtx.RunAndParse(t,
			"query", "log",
			"-i", "payment-service",
			"--last", "1h",
			"--size", "5",
		)
		Expect(t, logs).IsNotEmpty().AtMost(5).All(FieldEquals("app", "payment-service"))
	})

	t.Run("ErrorFiltering", func(t *testing.T) {
		logs := tCtx.RunAndParse(t,
			"query", "log",
			"-i", "splunk-all",
			"-f", "level=ERROR",
			"--last", "1h",
			"--size", "10",
		)
		if len(logs) > 0 {
			Expect(t, logs).All(IsErrorLevel())
		}
	})

	t.Run("TimeRange", func(t *testing.T) {
		now := time.Now()
		oneHourAgo := now.Add(-1 * time.Hour)
		logs := tCtx.RunAndParse(t,
			"query", "log",
			"-i", "splunk-all",
			"--last", "1h",
			"--size", "10",
		)
		if len(logs) > 0 {
			Expect(t, logs).All(DateAfter("timestamp", oneHourAgo))
		}
	})
}

func TestQueryLog_OpenSearch(t *testing.T) {
	t.Parallel()

	t.Run("BasicQuery", func(t *testing.T) {
		logs := tCtx.RunAndParse(t,
			"query", "log",
			"-i", "opensearch-all",
			"--last", "1h",
			"--size", "10",
		)
		Expect(t, logs).IsNotEmpty().AtMost(10)
	})

	t.Run("ErrorFiltering", func(t *testing.T) {
		logs := tCtx.RunAndParse(t,
			"query", "log",
			"-i", "opensearch-all",
			"-f", "level=ERROR",
			"--last", "1h",
			"--size", "10",
		)
		if len(logs) > 0 {
			Expect(t, logs).All(IsErrorLevel())
		}
	})
}

func TestQueryLog_MultiBackend(t *testing.T) {
	logs := tCtx.RunAndParse(t,
		"query", "log",
		"-i", "splunk-all",
		"-i", "opensearch-all",
		"--last", "15m",
		"--size", "20",
	)
	Expect(t, logs).IsNotEmpty().AtMost(20)
}

func TestQueryLog_JSONOutput(t *testing.T) {
	logs := tCtx.RunAndParse(t,
		"query", "log",
		"-i", "splunk-all",
		"--last", "1h",
		"--size", "3",
	)
	if len(logs) > 0 {
		Expect(t, logs).All(
			FieldPresent("timestamp"),
			FieldPresent("message"),
		)
	}
}
