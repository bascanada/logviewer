//go:build integration

package e2e

import (
	"testing"
)

func TestHLQueries(t *testing.T) {
	t.Parallel()

	t.Run("NotEqualsFilter", func(t *testing.T) {
		logs := tCtx.RunAndParse(t,
			"query", "log",
			"-i", "hl-context",
			"-f", "level!=DEBUG",
			"--last", "24h",
			"--size", "10",
		)
		if len(logs) > 0 {
			Expect(t, logs).None(FieldEquals("level", "DEBUG"))
		}
	})

	t.Run("OrQuery", func(t *testing.T) {
		logs := tCtx.RunAndParse(t,
			"query", "log",
			"-i", "hl-context",
			"-q", "level=ERROR OR level=WARN",
			"--last", "24h",
			"--size", "10",
		)
		if len(logs) > 0 {
			Expect(t, logs).All(FieldOneOf("level", "ERROR", "WARN"))
		}
	})

	t.Run("AndQuery", func(t *testing.T) {
		logs := tCtx.RunAndParse(t,
			"query", "log",
			"-i", "hl-context",
			"-q", "level=ERROR AND app=payment-service",
			"--last", "24h",
			"--size", "10",
		)
		if len(logs) > 0 {
			Expect(t, logs).All(
				IsErrorLevel(),
				IsFromApp("payment-service"),
			)
		}
	})

	t.Run("ContainsOperator", func(t *testing.T) {
		logs := tCtx.RunAndParse(t,
			"query", "log",
			"-i", "hl-context",
			"-q", "message CONTAINS timeout",
			"--last", "24h",
			"--size", "10",
		)
		if len(logs) > 0 {
			Expect(t, logs).All(FieldContains("message", "timeout"))
		}
	})

	t.Run("WildcardFilter", func(t *testing.T) {
		logs := tCtx.RunAndParse(t,
			"query", "log",
			"-i", "hl-context",
			"-f", "app=*-service",
			"--last", "24h",
			"--size", "10",
		)
		if len(logs) > 0 {
			Expect(t, logs).All(FieldMatches("app", "*-service"))
		}
	})
}
