package printer

import (
	"io"
	"os"

	"github.com/fatih/color"
	"github.com/mattn/go-isatty"
)

// ColorState manages global color output settings for the printer
type ColorState struct {
	enabled bool
}

var globalColorState = &ColorState{}

// InitColorState initializes color support based on configuration and environment.
// Priority order (highest to lowest):
//  1. Explicit user setting (via CLI flag or config)
//  2. NO_COLOR environment variable
//  3. TTY detection (auto-detect terminal)
//  4. Default to disabled (for unknown writers)
func InitColorState(explicitSetting *bool, writer io.Writer) {
	// Priority 1: Explicit user override
	if explicitSetting != nil {
		color.NoColor = !*explicitSetting
		globalColorState.enabled = *explicitSetting
		return
	}

	// Priority 2: NO_COLOR environment variable (standard)
	if os.Getenv("NO_COLOR") != "" {
		color.NoColor = true
		globalColorState.enabled = false
		return
	}

	// Priority 3: Auto-detect TTY
	if f, ok := writer.(*os.File); ok {
		globalColorState.enabled = isatty.IsTerminal(f.Fd())
		color.NoColor = !globalColorState.enabled
		return
	}

	// Priority 4: Default to disabled for unknown writers
	color.NoColor = true
	globalColorState.enabled = false
}

// IsColorEnabled returns whether color output is currently enabled.
// This is used by template functions to determine if they should apply colors.
func IsColorEnabled() bool {
	return globalColorState.enabled
}
