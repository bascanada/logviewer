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
		logs := tCtx.RunAndParse(t,
			"query", "field",
			"-i", "ssh-logs",
			"--last", "1h",
			"--size", "5",
		)
		if len(logs) > 0 {
			Expect(t, logs).All(FieldPresent("fields"))
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
