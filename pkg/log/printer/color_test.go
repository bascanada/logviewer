package printer

import (
	"bytes"
	"os"
	"testing"

	"github.com/fatih/color"
	"github.com/stretchr/testify/assert"
)

func TestInitColorState_ExplicitTrue(t *testing.T) {
	// Reset color state before test
	color.NoColor = false
	globalColorState.enabled = false

	enabled := true
	InitColorState(&enabled, os.Stdout)

	assert.True(t, IsColorEnabled())
	assert.False(t, color.NoColor)
}

func TestInitColorState_ExplicitFalse(t *testing.T) {
	// Reset color state before test
	color.NoColor = false
	globalColorState.enabled = false

	enabled := false
	InitColorState(&enabled, os.Stdout)

	assert.False(t, IsColorEnabled())
	assert.True(t, color.NoColor)
}

func TestInitColorState_NOCOLOREnv(t *testing.T) {
	// Set NO_COLOR environment variable
	originalValue := os.Getenv("NO_COLOR")
	defer func() {
		if originalValue != "" {
			_ = os.Setenv("NO_COLOR", originalValue)
		} else {
			_ = os.Unsetenv("NO_COLOR")
		}
	}()

	_ = os.Setenv("NO_COLOR", "1")

	// Reset color state before test
	color.NoColor = false
	globalColorState.enabled = false

	InitColorState(nil, os.Stdout)

	assert.False(t, IsColorEnabled())
	assert.True(t, color.NoColor)
}

func TestInitColorState_UnknownWriter(t *testing.T) {
	// Reset color state before test
	color.NoColor = false
	globalColorState.enabled = false

	// Unset NO_COLOR to test unknown writer behavior
	originalValue := os.Getenv("NO_COLOR")
	defer func() {
		if originalValue != "" {
			_ = os.Setenv("NO_COLOR", originalValue)
		} else {
			_ = os.Unsetenv("NO_COLOR")
		}
	}()
	_ = os.Unsetenv("NO_COLOR")

	// Use a buffer (not a file) as unknown writer
	buf := &bytes.Buffer{}
	InitColorState(nil, buf)

	assert.False(t, IsColorEnabled())
	assert.True(t, color.NoColor)
}

func TestColorLevel(t *testing.T) {
	tests := []struct {
		name          string
		level         string
		colorEnabled  bool
		shouldContain string
	}{
		{"ERROR with color", "ERROR", true, "ERROR"},
		{"ERROR without color", "ERROR", false, "ERROR"},
		{"WARN with color", "WARN", true, "WARN"},
		{"INFO with color", "INFO", true, "INFO"},
		{"DEBUG with color", "DEBUG", true, "DEBUG"},
		{"TRACE with color", "TRACE", true, "TRACE"},
		{"FATAL with color", "FATAL", true, "FATAL"},
		{"CRITICAL with color", "CRITICAL", true, "CRITICAL"},
		{"WARNING with color", "WARNING", true, "WARNING"},
		{"lowercase error", "error", true, "error"},
		{"unknown level", "CUSTOM", true, "CUSTOM"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set color state
			globalColorState.enabled = tt.colorEnabled
			color.NoColor = !tt.colorEnabled

			result := ColorLevel(tt.level)

			// Result should contain the original level text
			assert.Contains(t, result, tt.shouldContain)

			// When colors are disabled, result should equal input exactly
			if !tt.colorEnabled {
				assert.Equal(t, tt.level, result)
			}
		})
	}
}

func TestColorTimestamp(t *testing.T) {
	timestamp := "12:34:56"

	// Test with colors enabled
	globalColorState.enabled = true
	color.NoColor = false
	result := ColorTimestamp(timestamp)
	assert.Contains(t, result, timestamp)

	// Test with colors disabled
	globalColorState.enabled = false
	color.NoColor = true
	result = ColorTimestamp(timestamp)
	assert.Equal(t, timestamp, result)
}

func TestColorContext(t *testing.T) {
	contextID := "prod-logs"

	// Test with colors enabled
	globalColorState.enabled = true
	color.NoColor = false
	result := ColorContext(contextID)
	assert.Contains(t, result, contextID)

	// Test with colors disabled
	globalColorState.enabled = false
	color.NoColor = true
	result = ColorContext(contextID)
	assert.Equal(t, contextID, result)
}

func TestColorString(t *testing.T) {
	text := "test message"

	colors := []string{
		"red", "green", "yellow", "blue",
		"magenta", "cyan", "white", "black",
		"dim", "gray", "grey",
	}

	for _, colorName := range colors {
		t.Run("color_"+colorName, func(t *testing.T) {
			// Test with colors enabled
			globalColorState.enabled = true
			color.NoColor = false
			result := ColorString(colorName, text)
			assert.Contains(t, result, text)

			// Test with colors disabled
			globalColorState.enabled = false
			color.NoColor = true
			result = ColorString(colorName, text)
			assert.Equal(t, text, result)
		})
	}

	// Test unknown color
	t.Run("unknown_color", func(t *testing.T) {
		globalColorState.enabled = true
		color.NoColor = false
		result := ColorString("invalid", text)
		assert.Equal(t, text, result)
	})
}

func TestBold(t *testing.T) {
	text := "important"

	// Test with colors enabled
	globalColorState.enabled = true
	color.NoColor = false
	result := Bold(text)
	assert.Contains(t, result, text)

	// Test with colors disabled
	globalColorState.enabled = false
	color.NoColor = true
	result = Bold(text)
	assert.Equal(t, text, result)
}

func TestColorFunctionsThreadSafety(_ *testing.T) {
	// This test ensures color functions can be called concurrently
	// without panicking (fatih/color is thread-safe)

	done := make(chan bool, 3)

	go func() {
		for i := 0; i < 100; i++ {
			ColorLevel("ERROR")
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			ColorTimestamp("12:34:56")
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			ColorString("red", "message")
		}
		done <- true
	}()

	// Wait for all goroutines to complete
	<-done
	<-done
	<-done
}
