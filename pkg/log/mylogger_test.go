package log

import (
	"bytes"
	"log"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfigureMyLogger(t *testing.T) {
	// Redirect stdout/stderr to capture output?
	// ConfigureMyLogger sets log.SetOutput.
	
	// Test Stdout
	t.Run("stdout", func(t *testing.T) {
		opts := &MyLoggerOptions{
			Stdout: true,
			Level:  "INFO",
		}
		
		// Capture output
		var buf bytes.Buffer
		log.SetOutput(&buf)
		
		// Note: ConfigureMyLogger sets log.SetOutput to os.Stdout if Stdout is true.
		// We can't easily capture os.Stdout in parallel tests without swapping os.Stdout.
		// Instead, we verify state if possible, or just execution.
		
		// Since ConfigureMyLogger modifies global state (log package), 
		// we should be careful. 
		// We can just run it to ensure no panic.
		ConfigureMyLogger(opts)
		assert.Equal(t, LevelInfo, currentLevel)
	})

	t.Run("levels", func(t *testing.T) {
		tests := []struct {
			levelStr string
			expected int
		}{
			{"TRACE", LevelTrace},
			{"DEBUG", LevelDebug},
			{"INFO", LevelInfo},
			{"WARN", LevelWarn},
			{"ERROR", LevelError},
			{"UNKNOWN", LevelInfo},
		}

		for _, tt := range tests {
			ConfigureMyLogger(&MyLoggerOptions{Level: tt.levelStr})
			assert.Equal(t, tt.expected, currentLevel)
		}
	})
	
	t.Run("logging", func(t *testing.T) {
		// Set a custom writer
		var buf bytes.Buffer
		log.SetOutput(&buf)
		currentLevel = LevelDebug
		
		Debug("debug msg")
		Warn("warn msg")
		
		assert.Contains(t, buf.String(), "debug msg")
		assert.Contains(t, buf.String(), "warn msg")
		
		// Reset
		buf.Reset()
		currentLevel = LevelError
		Debug("debug msg") // Should not log
		Warn("warn msg") // Should not log
		
		assert.NotContains(t, buf.String(), "debug msg")
		assert.NotContains(t, buf.String(), "warn msg")
	})
	
	// Clean up - set output to devnull
	f, _ := os.Open(os.DevNull)
	log.SetOutput(f)
}
