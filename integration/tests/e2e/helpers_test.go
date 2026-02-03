//go:build integration

package e2e

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// RetryOptions defines the retry/polling policy
type RetryOptions struct {
	Timeout       time.Duration
	RetryInterval time.Duration
}

// DefaultRetryOptions: 30 seconds max wait, check every 500ms
var DefaultRetryOptions = RetryOptions{
	Timeout:       30 * time.Second,
	RetryInterval: 500 * time.Millisecond,
}

type TestContext struct {
	BinaryPath         string
	ConfigPath         string
	RunID              string // Unique ID for this test run to isolate data
	DisableRunIDFilter bool   // Set to true to disable automatic run_id injection
}

var tCtx TestContext

func (c *TestContext) Run(t *testing.T, args ...string) (string, string, error) {
	t.Helper()

	// Log the command for debugging purposes (visible on test failure or with -v)
	// Attempts to construct a copy-pasteable command line including the config env var
	cmdStr := fmt.Sprintf("LOGVIEWER_CONFIG=%s %s %s", c.ConfigPath, c.BinaryPath, strings.Join(args, " "))
	t.Logf("Running command: %s", cmdStr)

	cmd := exec.Command(c.BinaryPath, args...)
	cmd.Env = append(os.Environ(), "LOGVIEWER_CONFIG="+c.ConfigPath)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

func (c *TestContext) RunAndExpectSuccess(t *testing.T, args ...string) (string, string) {
	t.Helper()
	stdout, stderr, err := c.Run(t, args...)
	require.NoError(t, err, "Command failed. Stderr: %s", stderr)
	return stdout, stderr
}

func (c *TestContext) RunAndParse(t *testing.T, args ...string) []map[string]interface{} {
	t.Helper()
	hasJson := false
	for _, a := range args {
		if a == "--json" {
			hasJson = true
			break
		}
	}
	if !hasJson {
		args = append(args, "--json")
	}

	// Auto-inject run_id filter if RunID is set and not disabled
	if c.RunID != "" && !c.DisableRunIDFilter {
		args = c.injectRunIDFilter(args)
	}

	stdout, stderr, err := c.Run(t, args...)
	require.NoError(t, err, "Command failed. Stderr: %s", stderr)
	return parseJSONOutput(t, stdout)
}

// RunAndParseUntilFound executes CLI in a loop until logs are found or timeout
// This replaces static sleeps with intelligent polling
func (c *TestContext) RunAndParseUntilFound(t *testing.T, expectedCount int, args ...string) []map[string]interface{} {
	t.Helper()

	start := time.Now()
	var lastLogs []map[string]interface{}
	var lastErr error

	// Ensure JSON flag is present
	hasJson := false
	for _, a := range args {
		if a == "--json" {
			hasJson = true
			break
		}
	}
	if !hasJson {
		args = append(args, "--json")
	}

	// Auto-inject run_id filter if RunID is set and not disabled
	if c.RunID != "" && !c.DisableRunIDFilter {
		args = c.injectRunIDFilter(args)
	}

	attempts := 0
	for time.Since(start) < DefaultRetryOptions.Timeout {
		attempts++

		// Use Run (not RunAndParse) to avoid failing test on transient errors
		stdout, _, err := c.Run(t, args...)
		if err == nil {
			logs := parseJSONOutput(t, stdout)
			if len(logs) >= expectedCount {
				if attempts > 1 {
					t.Logf("Found %d logs after %d attempts (%.1fs)", len(logs), attempts, time.Since(start).Seconds())
				}
				return logs // Success!
			}
			lastLogs = logs
		} else {
			lastErr = err
		}

		time.Sleep(DefaultRetryOptions.RetryInterval)
	}

	// Timeout - fail test with detailed message
	require.Failf(t, "Timeout waiting for logs",
		"Expected at least %d logs, got %d after %d attempts in %v. Last error: %v. Command: %v",
		expectedCount, len(lastLogs), attempts, DefaultRetryOptions.Timeout, lastErr, args)

	return nil
}

// injectRunIDFilter adds run_id filter to query args if not already present
func (c *TestContext) injectRunIDFilter(args []string) []string {
	// Check if run_id filter already exists
	for i, arg := range args {
		if arg == "-f" && i+1 < len(args) && strings.Contains(args[i+1], "run_id=") {
			return args // Already has run_id filter
		}
	}

	// Inject run_id filter after the command but before other flags
	// Find insertion point (after "query log/field/values")
	insertIdx := 0
	for i, arg := range args {
		if arg == "query" || arg == "log" || arg == "field" || arg == "values" {
			insertIdx = i + 1
		}
		if strings.HasPrefix(arg, "-") {
			break
		}
	}

	// Insert at the right position
	result := make([]string, 0, len(args)+2)
	result = append(result, args[:insertIdx]...)
	result = append(result, "-f", "run_id="+c.RunID)
	result = append(result, args[insertIdx:]...)

	return result
}

func parseJSONOutput(t *testing.T, output string) []map[string]interface{} {
	t.Helper()
	output = strings.TrimSpace(output)
	if output == "" {
		return []map[string]interface{}{}
	}
	var logsArray []map[string]interface{}
	if err := json.Unmarshal([]byte(output), &logsArray); err == nil {
		return logsArray
	}
	var logs []map[string]interface{}
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var log map[string]interface{}
		err := json.Unmarshal([]byte(line), &log)
		require.NoError(t, err, "Failed to unmarshal NDJSON line: %s", line)
		logs = append(logs, log)
	}
	require.NoError(t, scanner.Err(), "Error reading NDJSON output")
	return logs
}

func (c *TestContext) RunAndParseValues(t *testing.T, args ...string) map[string][]string {
	t.Helper()
	hasJson := false
	for _, a := range args {
		if a == "--json" {
			hasJson = true
			break
		}
	}
	if !hasJson {
		args = append(args, "--json")
	}
	stdout, stderr, err := c.Run(t, args...)
	require.NoError(t, err, "Command failed. Stderr: %s", stderr)
	var values map[string][]interface{}
	err = json.Unmarshal([]byte(strings.TrimSpace(stdout)), &values)
	require.NoError(t, err, "Failed to unmarshal values JSON: %s", stdout)
	result := make(map[string][]string)
	for field, vals := range values {
		strVals := make([]string, len(vals))
		for i, v := range vals {
			strVals[i] = v.(string)
		}
		result[field] = strVals
	}
	return result
}

// RunCommand is a convenience wrapper for running commands
func RunCommand(args ...string) (string, error) {
	cmd := exec.Command(tCtx.BinaryPath, args...)
	cmd.Env = append(os.Environ(), "LOGVIEWER_CONFIG="+tCtx.ConfigPath)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

// ParseJSONOutput parses NDJSON output
func ParseJSONOutput(output string) []map[string]interface{} {
	output = strings.TrimSpace(output)
	if output == "" {
		return []map[string]interface{}{}
	}

	// Try array format first
	var logsArray []map[string]interface{}
	if err := json.Unmarshal([]byte(output), &logsArray); err == nil {
		return logsArray
	}

	// Try NDJSON format
	var logs []map[string]interface{}
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var log map[string]interface{}
		if err := json.Unmarshal([]byte(line), &log); err == nil {
			logs = append(logs, log)
		}
	}
	return logs
}

// ParseFieldsJSON parses field discovery JSON output
func ParseFieldsJSON(output string) map[string]interface{} {
	var fields map[string]interface{}
	json.Unmarshal([]byte(strings.TrimSpace(output)), &fields)
	return fields
}

// ParseValuesJSON parses values query JSON output
func ParseValuesJSON(output string) map[string][]string {
	var values map[string][]interface{}
	json.Unmarshal([]byte(strings.TrimSpace(output)), &values)

	result := make(map[string][]string)
	for field, vals := range values {
		strVals := make([]string, len(vals))
		for i, v := range vals {
			if str, ok := v.(string); ok {
				strVals[i] = str
			}
		}
		result[field] = strVals
	}
	return result
}

// GetFieldValue extracts a field value from a log entry
func GetFieldValue(entry map[string]interface{}, fieldName string) string {
	if val, ok := entry[fieldName]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	// Check nested fields structure (for Splunk)
	if fields, ok := entry["fields"].(map[string]interface{}); ok {
		if val, ok := fields[fieldName]; ok {
			if str, ok := val.(string); ok {
				return str
			}
		}
	}
	return ""
}

// GetFieldValueAsInt extracts a field value as integer
func GetFieldValueAsInt(entry map[string]interface{}, fieldName string) int {
	if val, ok := entry[fieldName]; ok {
		switch v := val.(type) {
		case float64:
			return int(v)
		case int:
			return v
		}
	}
	// Check nested fields
	if fields, ok := entry["fields"].(map[string]interface{}); ok {
		if val, ok := fields[fieldName]; ok {
			switch v := val.(type) {
			case float64:
				return int(v)
			case int:
				return v
			}
		}
	}
	return 0
}
