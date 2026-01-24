package hl

import (
	"fmt"
	"strings"
)

// BuildSSHCommand constructs a shell command string for SSH execution.
// It creates a hybrid command that:
// 1. Checks if hl is available on the remote host
// 2. Uses hl with the provided arguments if available
// 3. Falls back to a simple cat/tail command if hl is not present
//
// This approach ensures:
// - Optimal performance when hl is installed remotely (filtering happens server-side)
// - Graceful fallback when hl is not available (client-side filtering still works)
// - Reduced bandwidth usage with hl (only filtered results are transmitted)
//
// Parameters:
//   - hlArgs: Arguments for hl (excluding the "hl" command itself)
//   - paths: File paths to read
//   - fallbackCmd: The command to use if hl is not available (e.g., "cat" or "tail -f")
//
// Returns a shell command string safe for SSH execution.
func BuildSSHCommand(hlArgs []string, paths []string, fallbackCmd string) string {
	// Build the hl command with proper escaping
	hlCmdParts := make([]string, 0, 1+len(hlArgs)+len(paths))
	hlCmdParts = append(hlCmdParts, "hl")
	for _, arg := range hlArgs {
		hlCmdParts = append(hlCmdParts, shellEscape(arg))
	}
	for _, path := range paths {
		hlCmdParts = append(hlCmdParts, shellEscape(path))
	}
	hlCmd := strings.Join(hlCmdParts, " ")

	// Build the fallback command
	var fallback string
	if fallbackCmd != "" {
		fallback = fallbackCmd
	} else {
		// Default fallback: cat the files
		fallbackParts := make([]string, 0, 1+len(paths))
		fallbackParts = append(fallbackParts, "cat")
		for _, path := range paths {
			fallbackParts = append(fallbackParts, shellEscape(path))
		}
		fallback = strings.Join(fallbackParts, " ")
	}

	// Construct the hybrid command
	// The command checks for hl availability and chooses the appropriate path
	return fmt.Sprintf(`if command -v hl >/dev/null 2>&1; then %s; else %s; fi`, hlCmd, fallback)
}

// BuildSSHCommandWithMarker is like BuildSSHCommand but adds a marker to the output
// to indicate which engine was used. This is useful for the client to know whether
// the output has been pre-filtered.
//
// The marker is printed to stderr so it doesn't interfere with log output.
func BuildSSHCommandWithMarker(hlArgs []string, paths []string, fallbackCmd string) string {
	// Build the hl command with proper escaping
	hlCmdParts := make([]string, 0, 1+len(hlArgs)+len(paths))
	hlCmdParts = append(hlCmdParts, "hl")
	for _, arg := range hlArgs {
		hlCmdParts = append(hlCmdParts, shellEscape(arg))
	}
	for _, path := range paths {
		hlCmdParts = append(hlCmdParts, shellEscape(path))
	}
	hlCmd := strings.Join(hlCmdParts, " ")

	// Build the fallback command
	var fallback string
	if fallbackCmd != "" {
		fallback = fallbackCmd
	} else {
		fallbackParts := make([]string, 0, 1+len(paths))
		fallbackParts = append(fallbackParts, "cat")
		for _, path := range paths {
			fallbackParts = append(fallbackParts, shellEscape(path))
		}
		fallback = strings.Join(fallbackParts, " ")
	}

	// Add markers to indicate which engine was used (sent to stderr)
	hlCmdWithMarker := fmt.Sprintf(`echo "HL_ENGINE=hl" >&2; %s`, hlCmd)
	fallbackWithMarker := fmt.Sprintf(`echo "HL_ENGINE=native" >&2; %s`, fallback)

	return fmt.Sprintf(`if command -v hl >/dev/null 2>&1; then %s; else %s; fi`, hlCmdWithMarker, fallbackWithMarker)
}

// BuildFollowSSHCommand constructs a command for following logs via SSH.
// When in follow mode, we use tail -f as the fallback instead of cat.
func BuildFollowSSHCommand(hlArgs []string, paths []string) string {
	// For follow mode, the fallback should use tail -f
	fallbackParts := make([]string, 0, 2+len(paths))
	fallbackParts = append(fallbackParts, "tail", "-f")
	for _, path := range paths {
		fallbackParts = append(fallbackParts, shellEscape(path))
	}
	fallback := strings.Join(fallbackParts, " ")

	return BuildSSHCommand(hlArgs, paths, fallback)
}

// shellEscape escapes a string for safe use in a shell command.
// This is critical for preventing command injection attacks.
func shellEscape(s string) string {
	// If the string contains no special characters, return as-is
	if isShellSafe(s) {
		return s
	}

	// Use single quotes for escaping, which prevents interpretation of
	// all special characters except single quotes themselves.
	// For single quotes within the string, we close the single quote,
	// add an escaped single quote, and re-open single quotes.
	escaped := strings.ReplaceAll(s, "'", `'\''`)
	return "'" + escaped + "'"
}

// isShellSafe returns true if the string contains no shell special characters
// and can be used without quoting.
func isShellSafe(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		// Allow alphanumerics, underscore, hyphen, dot, forward slash, and colon
		if !((c >= 'a' && c <= 'z') ||
			(c >= 'A' && c <= 'Z') ||
			(c >= '0' && c <= '9') ||
			c == '_' || c == '-' || c == '.' || c == '/' || c == ':') {
			return false
		}
	}
	return true
}

// ArgsToString converts a slice of arguments to a single command string.
// Each argument is properly escaped.
func ArgsToString(args []string) string {
	parts := make([]string, 0, len(args))
	for _, arg := range args {
		parts = append(parts, shellEscape(arg))
	}
	return strings.Join(parts, " ")
}
