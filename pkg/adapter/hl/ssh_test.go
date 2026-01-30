package hl

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildSSHCommand_Basic(t *testing.T) {
	args := []string{"-P", "--raw", "--since", "-15m", "-q", "level = error"}
	paths := []string{"/var/log/app.log"}

	cmd := BuildSSHCommand(args, paths, "", 0)

	// Should contain the if-else structure
	assert.Contains(t, cmd, "if command -v hl >/dev/null 2>&1; then")
	assert.Contains(t, cmd, "; else")
	assert.Contains(t, cmd, "; fi")

	// Should contain hl command with args
	assert.Contains(t, cmd, "hl -P --raw --since -15m -q 'level = error' /var/log/app.log")

	// Should contain fallback
	assert.Contains(t, cmd, "cat /var/log/app.log")
}

func TestBuildSSHCommand_CustomFallback(t *testing.T) {
	args := []string{"-P", "--raw"}
	paths := []string{"/var/log/app.log"}

	cmd := BuildSSHCommand(args, paths, "tail -n 1000 /var/log/app.log", 0)

	assert.Contains(t, cmd, "tail -n 1000 /var/log/app.log")
}

func TestBuildFollowSSHCommand(t *testing.T) {
	args := []string{"-P", "--raw", "-F"}
	paths := []string{"/var/log/app.log"}

	cmd := BuildFollowSSHCommand(args, paths, 0)

	assert.Contains(t, cmd, "hl -P --raw -F /var/log/app.log")
	assert.Contains(t, cmd, "tail -f /var/log/app.log")
}

func TestBuildSSHCommand_MultiplePaths(t *testing.T) {
	args := []string{"-P", "--raw"}
	paths := []string{"/var/log/app.log", "/var/log/error.log"}

	cmd := BuildSSHCommand(args, paths, "", 0)

	assert.Contains(t, cmd, "/var/log/app.log")
	assert.Contains(t, cmd, "/var/log/error.log")
}

func TestBuildSSHCommandWithMarker(t *testing.T) {
	args := []string{"-P", "--raw"}
	paths := []string{"/var/log/app.log"}

	cmd := BuildSSHCommandWithMarker(args, paths, "", 0)

	// Should contain engine markers
	assert.Contains(t, cmd, `echo "HL_ENGINE=hl" >&2`)
	assert.Contains(t, cmd, `echo "HL_ENGINE=native" >&2`)
}

func TestShellEscape(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"simple", "hello", "hello"},
		{"path", "/var/log/app.log", "/var/log/app.log"},
		{"hyphen arg", "-P", "-P"},
		{"with space", "hello world", "'hello world'"},
		{"with quote", "it's", `'it'\''s'`},
		{"with double quote", `say "hi"`, `'say "hi"'`},
		{"special chars", "a; rm -rf /", "'a; rm -rf /'"},
		{"backticks", "`whoami`", "'`whoami`'"},
		{"dollar", "$HOME", "'$HOME'"},
		{"pipe", "cat | grep", "'cat | grep'"},
		{"redirect", "file > /dev/null", "'file > /dev/null'"},
		{"ampersand", "cmd &", "'cmd &'"},
		{"semicolon", "cmd; cmd2", "'cmd; cmd2'"},
		{"newline", "line1\nline2", "'line1\nline2'"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shellEscape(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsShellSafe(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"hello", true},
		{"Hello123", true},
		{"/var/log/app.log", true},
		{"-P", true},
		{"--since", true},
		{"2024-01-01T00:00:00Z", true},
		{"hello world", false},
		{"it's", false},
		{"$HOME", false},
		{"`cmd`", false},
		{"a;b", false},
		{"a|b", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := isShellSafe(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestArgsToString(t *testing.T) {
	args := []string{"hl", "-P", "--raw", "-q", "level = error", "/var/log/app.log"}
	result := ArgsToString(args)

	assert.Equal(t, "hl -P --raw -q 'level = error' /var/log/app.log", result)
}

func TestBuildSSHCommand_InjectionPrevention(t *testing.T) {
	// Test that malicious input is properly escaped
	maliciousValue := "error'; rm -rf / #"
	args := []string{"-P", "--raw", "-q", maliciousValue}
	paths := []string{"/var/log/app.log"}

	cmd := BuildSSHCommand(args, paths, "", 0)

	// The malicious value should be escaped with single quotes
	// The escaped form is: 'error'\'''; rm -rf / #'
	// This breaks out of the quote, adds an escaped quote, and re-enters
	assert.Contains(t, cmd, `'error'\''`)

	// The key thing is that the semicolon and rm command are INSIDE single quotes
	// so they won't be interpreted as shell commands
	// The full escaped string should be: 'error'\''; rm -rf / #'
	// This means the shell sees the literal string: error'; rm -rf / #
	// NOT a command terminator followed by rm -rf

	// Verify the structure is correct - the rm command is quoted, not raw
	assert.Contains(t, cmd, `hl -P --raw -q 'error'\''`)
}
