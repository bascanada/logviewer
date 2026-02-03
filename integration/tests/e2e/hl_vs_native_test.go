//go:build integration

package e2e

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHLvsNative compares results between the HL-based engine and the Native Go engine
// to ensure they produce identical results for the same queries.
func TestHLvsNative(t *testing.T) {
	t.Skip("HL vs Native comparison tests need specific SSH setup - skipping for core smoke tests")
	// These tests compare ssh-hl-test (uses remote hl binary)
	// with ssh-native-test (uses client-side Go processing).

	testCases := []struct {
		name string
		args []string
	}{
		{
			name: "BasicFilter",
			args: []string{"-f", "level=ERROR", "--size", "100"},
		},
		{
			name: "NotEquals",
			args: []string{"-f", "level!=DEBUG", "--size", "100"},
		},
		{
			name: "RegexMatch",
			args: []string{"-f", "message~=.*error.*", "--size", "100"},
		},
		{
			name: "ComparisonGT",
			args: []string{"-f", "latency_ms>1000", "--size", "100"},
		},
		{
			name: "OrLogic",
			args: []string{"-q", "level=ERROR OR level=WARN", "--size", "100"},
		},
		{
			name: "AndLogic",
			args: []string{"-q", "level=ERROR AND app=log-generator", "--size", "100"},
		},
		{
			name: "NotLogic",
			args: []string{"-q", "NOT level=DEBUG", "--size", "100"},
		},
		{
			name: "Exists",
			args: []string{"-q", "exists(trace_id)", "--size", "100"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Run with HL engine
			hlArgs := append([]string{"query", "log", "-i", "ssh-hl-test", "--json", "--last", "24h"}, tc.args...)
			hlLogs := tCtx.RunAndParse(t, hlArgs...)

			// Run with Native engine
			nativeArgs := append([]string{"query", "log", "-i", "ssh-native-test", "--json", "--last", "24h"}, tc.args...)
			nativeLogs := tCtx.RunAndParse(t, nativeArgs...)

			// Compare result counts
			require.Equal(t, len(hlLogs), len(nativeLogs), "Result counts should match for %s", tc.name)

			if len(hlLogs) == 0 {
				t.Log("Warning: No logs found for test case, comparison might be trivial")
				return
			}

			// Extract and sort IDs for comparison to ensure identical content
			hlIDs := extractAndSortLogIDs(hlLogs)
			nativeIDs := extractAndSortLogIDs(nativeLogs)

			assert.Equal(t, hlIDs, nativeIDs, "Log content should match for %s", tc.name)
		})
	}
}

// extractAndSortLogIDs creates a comparable representation of log entries
func extractAndSortLogIDs(logs []map[string]interface{}) []string {
	ids := make([]string, len(logs))
	for i, log := range logs {
		// Use timestamp + message + level as a pseudo-ID
		ts := GetFieldValue(log, "timestamp")
		if ts == "" {
			ts = GetFieldValue(log, "@timestamp")
		}
		msg := GetFieldValue(log, "message")
		level := GetFieldValue(log, "level")
		ids[i] = ts + "|" + level + "|" + msg
	}
	sort.Strings(ids)
	return ids
}
