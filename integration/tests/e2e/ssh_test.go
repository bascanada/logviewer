//go:build integration

package e2e

import (
	"testing"
)

func TestSSH(t *testing.T) {
	t.Parallel()

	t.Run("BasicSSHQuery", func(t *testing.T) {
		logs := tCtx.RunAndParse(t,
			"query", "log",
			"-i", "ssh-logs",
			"--last", "1h",
			"--size", "10",
		)
		Expect(t, logs).AtMost(10)
	})

	t.Run("SSHWithFilters", func(t *testing.T) {
		logs := tCtx.RunAndParse(t,
			"query", "log",
			"-i", "ssh-logs",
			"-f", "level=ERROR",
			"--last", "1h",
			"--size", "10",
		)
		if len(logs) > 0 {
			Expect(t, logs).All(IsErrorLevel())
		}
	})

	t.Run("SSHFieldDiscovery", func(t *testing.T) {
		output, err := RunCommand("query", "field", "-i", "ssh-logs", "--last", "1h", "--json")
		if err != nil {
			t.Logf("Command Output: %s\n", output)
		}
		if err != nil {
			t.Fatalf("Field discovery should execute successfully: %v", err)
		}

		fields := ParseFieldsJSON(output)
		if len(fields) > 0 {
			// SSH logs using json-extract should have level field
			_, hasLevel := fields["level"]
			if !hasLevel {
				t.Errorf("Expected 'level' field in SSH logs, got fields: %v", fields)
			}
		}
	})
}

func TestSSH_HL(t *testing.T) {
	t.Parallel()

	t.Run("NotEqualsFilter", func(t *testing.T) {
		logs := tCtx.RunAndParse(t,
			"query", "log",
			"-i", "ssh-logs",
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
			"-i", "ssh-logs",
			"-q", "level=ERROR OR level=WARN",
			"--last", "24h",
			"--size", "10",
		)
		if len(logs) > 0 {
			Expect(t, logs).All(FieldOneOf("level", "ERROR", "WARN"))
		}
	})
}
