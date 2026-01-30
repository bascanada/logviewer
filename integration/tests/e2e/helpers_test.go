//go:build integration

package e2e

import (
	"bufio"
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

type TestContext struct {
	BinaryPath string
	ConfigPath string
}

var tCtx TestContext

func (c *TestContext) Run(t *testing.T, args ...string) (string, string, error) {
	t.Helper()
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
	stdout, stderr, err := c.Run(t, args...)
	require.NoError(t, err, "Command failed. Stderr: %s", stderr)
	return parseJSONOutput(t, stdout)
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
