//go:build integration

package e2e

import (
	"strings"
	"testing"
)

func TestRecursiveFilters_Splunk(t *testing.T) {
	t.Skip("Recursive filter tests need specific context configurations - skipping for core smoke tests")
	t.Parallel()

	t.Run("SimpleOrFilter", func(t *testing.T) {
		logs := tCtx.RunAndParse(t,
			"query", "log",
			"-i", "splunk-errors-or-warns",
			"--last", "1h",
			"--size", "10",
		)
		Expect(t, logs).IsNotEmpty().All(FieldOneOf("level", "ERROR", "WARN"))
	})

	t.Run("NestedAndOrFilter", func(t *testing.T) {
		logs := tCtx.RunAndParse(t,
			"query", "log",
			"-i", "splunk-critical",
			"--last", "1h",
			"--size", "10",
		)
		Expect(t, logs).IsNotEmpty().All(
			FieldOneOf("level", "ERROR", "WARN"),
			FieldEquals("app", "log-generator"),
		)
	})

	t.Run("NotFilter", func(t *testing.T) {
		logs := tCtx.RunAndParse(t,
			"query", "log",
			"-i", "splunk-no-debug",
			"--last", "1h",
			"--size", "10",
		)
		if len(logs) > 0 {
			Expect(t, logs).None(FieldEquals("level", "DEBUG"))
		}
	})

	t.Run("LegacyPlusFilter", func(t *testing.T) {
		logs := tCtx.RunAndParse(t,
			"query", "log",
			"-i", "splunk-legacy-plus-filter",
			"--last", "1h",
			"--size", "10",
		)
		Expect(t, logs).IsNotEmpty().All(
			FieldEquals("app", "log-generator"),
			FieldOneOf("level", "ERROR", "WARN"),
		)
	})

	t.Run("ExistsOperator", func(t *testing.T) {
		logs := tCtx.RunAndParse(t,
			"query", "log",
			"-i", "splunk-has-trace",
			"--last", "1h",
			"--size", "10",
		)
		Expect(t, logs).IsNotEmpty().All(HasTraceID())
	})
}

func TestRecursiveFilters_OpenSearch(t *testing.T) {
	t.Skip("Recursive filter tests need specific context configurations - skipping for core smoke tests")
	t.Parallel()

	t.Run("SimpleOrFilter", func(t *testing.T) {
		logs := tCtx.RunAndParse(t,
			"query", "log",
			"-i", "opensearch-errors-or-warns",
			"--last", "1h",
			"--size", "10",
		)
		Expect(t, logs).IsNotEmpty().All(FieldOneOf("level", "ERROR", "WARN"))
	})

	t.Run("ComplexNestedFilter", func(t *testing.T) {
		logs := tCtx.RunAndParse(t,
			"query", "log",
			"-i", "opensearch-complex",
			"--last", "1h",
			"--size", "10",
		)
		Expect(t, logs).IsNotEmpty().All(
			FieldEquals("app", "log-generator"),
			FieldOneOf("level", "ERROR", "WARN"),
		)
	})

	t.Run("RegexFilterWithOr", func(t *testing.T) {
		logs := tCtx.RunAndParse(t,
			"query", "log",
			"-i", "opensearch-regex-errors",
			"--last", "1h",
			"--size", "10",
		)
		Expect(t, logs).IsNotEmpty().All(func() LogCheck {
			return simpleCheck{
				validator: func(log map[string]interface{}) bool {
					msg := GetFieldValue(log, "message")
					msgLower := strings.ToLower(msg)
					return strings.Contains(msgLower, "error") || strings.Contains(msgLower, "fail")
				},
				desc: "message contains 'error' or 'fail' (case-insensitive)",
			}
		}())
	})

	t.Run("WildcardOperator", func(t *testing.T) {
		logs := tCtx.RunAndParse(t,
			"query", "log",
			"-i", "opensearch-wildcard",
			"--last", "1h",
			"--size", "10",
		)
		Expect(t, logs).IsNotEmpty().All(FieldMatches("app", "log-*"))
	})
}

func TestRecursiveFilters_K8s(t *testing.T) {
	t.Skip("K8s tests require K8s cluster - skipping for core smoke tests")
	t.Parallel()

	t.Run("ClientSideOrFilter", func(t *testing.T) {
		logs := tCtx.RunAndParse(t,
			"query", "log",
			"-i", "k8s-errors-or-warns",
			"--last", "1h",
			"--size", "10",
		)
		Expect(t, logs).IsNotEmpty().All(FieldOneOf("level", "ERROR", "WARN"))
	})

	t.Run("ClientSideNotFilter", func(t *testing.T) {
		logs := tCtx.RunAndParse(t,
			"query", "log",
			"-i", "k8s-no-debug",
			"--last", "1h",
			"--size", "10",
		)
		if len(logs) > 0 {
			Expect(t, logs).None(FieldEquals("level", "DEBUG"))
		}
	})
}

func TestHLCompatibleQuerySyntax_Splunk(t *testing.T) {
	t.Parallel()

	t.Run("GreaterThan", func(t *testing.T) {
		logs := tCtx.RunAndParse(t,
			"query", "log",
			"-i", "splunk-slow-requests",
			"--last", "1h",
			"--size", "10",
		)
		if len(logs) > 0 {
			Expect(t, logs).All(func() LogCheck {
				return simpleCheck{
					validator: func(log map[string]interface{}) bool {
						latency := GetFieldValueAsInt(log, "latency_ms")
						return latency > 1000
					},
					desc: "latency_ms > 1000",
				}
			}())
		}
	})

	t.Run("Negation", func(t *testing.T) {
		logs := tCtx.RunAndParse(t,
			"query", "log",
			"-i", "splunk-not-debug-negate",
			"--last", "1h",
			"--size", "10",
		)
		if len(logs) > 0 {
			Expect(t, logs).None(FieldEquals("level", "DEBUG"))
		}
	})

	t.Run("ComplexComparisonWithOr", func(t *testing.T) {
		logs := tCtx.RunAndParse(t,
			"query", "log",
			"-i", "splunk-critical-slow",
			"--last", "1h",
			"--size", "10",
		)
		if len(logs) > 0 {
			Expect(t, logs).All(func() LogCheck {
				return simpleCheck{
					validator: func(log map[string]interface{}) bool {
						level := GetFieldValue(log, "level")
						latency := GetFieldValueAsInt(log, "latency_ms")
						return level == "ERROR" || latency > 1000
					},
					desc: "level == ERROR OR latency_ms > 1000",
				}
			}())
		}
	})
}
