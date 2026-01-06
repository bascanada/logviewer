package printer_test

import (
	"strings"
	"testing"
	"time"

	"github.com/bascanada/logviewer/pkg/log/printer"
	"github.com/stretchr/testify/assert"
)

func TestExpandJson(t *testing.T) {
	t.Run("expands simple JSON object", func(t *testing.T) {
		input := "get data from json: {\"key\": \"value\", \"num\": 42}"
		result := printer.ExpandJson(input)
		assert.NotEmpty(t, result)
		assert.Contains(t, result, "key")
		assert.Contains(t, result, "value")
		assert.Contains(t, result, "42")
	})

	t.Run("expands JSON array", func(t *testing.T) {
		input := "Response: [\"item1\", \"item2\", \"item3\"]"
		result := printer.ExpandJson(input)
		assert.NotEmpty(t, result)
		assert.Contains(t, result, "item1")
		assert.Contains(t, result, "item2")
	})

	t.Run("expands nested JSON", func(t *testing.T) {
		input := "Payload: {\"user\": {\"name\": \"John\", \"id\": 123}, \"active\": true}"
		result := printer.ExpandJson(input)
		assert.NotEmpty(t, result)
		assert.Contains(t, result, "user")
		assert.Contains(t, result, "name")
		assert.Contains(t, result, "John")
	})

	t.Run("expands multiple JSON objects", func(t *testing.T) {
		input := "First: {\"a\": 1} Second: {\"b\": 2}"
		result := printer.ExpandJson(input)
		assert.NotEmpty(t, result)
		assert.Contains(t, result, "a")
		assert.Contains(t, result, "b")
	})

	t.Run("ignores empty JSON objects", func(t *testing.T) {
		input := "Empty object: {}"
		result := printer.ExpandJson(input)
		assert.Empty(t, result)
	})

	t.Run("ignores empty JSON arrays", func(t *testing.T) {
		input := "Empty array: []"
		result := printer.ExpandJson(input)
		assert.Empty(t, result)
	})

	t.Run("returns empty for no JSON", func(t *testing.T) {
		input := "This is just a plain log message"
		result := printer.ExpandJson(input)
		assert.Empty(t, result)
	})

	t.Run("handles JSON with special characters", func(t *testing.T) {
		input := "Message: {\"url\": \"https://example.com?param=value&other=123\"}"
		result := printer.ExpandJson(input)
		assert.NotEmpty(t, result)
		assert.Contains(t, result, "url")
	})

	t.Run("handles JSON with escaped quotes", func(t *testing.T) {
		input := "Data: {\"message\": \"He said \\\"hello\\\" to me\"}"
		result := printer.ExpandJson(input)
		assert.NotEmpty(t, result)
		assert.Contains(t, result, "message")
	})

	t.Run("handles real-world checkout log", func(t *testing.T) {
		input := "Outbound: {\"redirectUrl\":\"https://payments.example.com\",\"sessionId\":\"ABC123\"}"
		result := printer.ExpandJson(input)
		assert.NotEmpty(t, result)
		assert.Contains(t, result, "redirectUrl")
		assert.Contains(t, result, "sessionId")
	})
}

func TestExpandJsonLimit(t *testing.T) {
	t.Run("respects line limit", func(t *testing.T) {
		// Create a JSON with many fields (will be many lines when formatted)
		input := `Data: {"field1": "value1", "field2": "value2", "field3": "value3", "field4": "value4", "field5": "value5"}`
		result := printer.ExpandJsonLimit(input, 3)

		lines := len(strings.Split(result, "\n"))
		// Should have at most 4 lines (3 + truncation message)
		assert.LessOrEqual(t, lines, 5)
		assert.Contains(t, result, "truncated")
	})

	t.Run("no truncation for short JSON", func(t *testing.T) {
		input := "Data: {\"key\": \"value\"}"
		result := printer.ExpandJsonLimit(input, 10)

		assert.NotEmpty(t, result)
		assert.NotContains(t, result, "truncated")
	})

	t.Run("returns empty for no JSON", func(t *testing.T) {
		input := "No JSON here"
		result := printer.ExpandJsonLimit(input, 5)
		assert.Empty(t, result)
	})
}

func TestExpandJsonCompact(t *testing.T) {
	t.Run("formats JSON on single line", func(t *testing.T) {
		input := "Data: {\"key\": \"value\", \"num\": 42}"
		result := printer.ExpandJsonCompact(input)

		assert.NotEmpty(t, result)
		// Should have minimal newlines (only leading newline per JSON)
		lines := strings.Split(strings.TrimSpace(result), "\n")
		assert.LessOrEqual(t, len(lines), 2) // At most 2 lines for single JSON
	})

	t.Run("formats array on single line", func(t *testing.T) {
		input := "Array: [\"a\", \"b\", \"c\"]"
		result := printer.ExpandJsonCompact(input)

		assert.NotEmpty(t, result)
		assert.Contains(t, result, "a")
		lines := strings.Split(strings.TrimSpace(result), "\n")
		assert.LessOrEqual(t, len(lines), 2)
	})

	t.Run("returns empty for no JSON", func(t *testing.T) {
		input := "No JSON here"
		result := printer.ExpandJsonCompact(input)
		assert.Empty(t, result)
	})
}

func TestFormatTimestamp(t *testing.T) {
	t.Run("formats valid timestamp in local time", func(t *testing.T) {
		// Use local time to ensure test works regardless of timezone
		ts := time.Date(2025, 12, 17, 10, 30, 45, 0, time.Local)
		result := printer.FormatTimestamp(ts, "15:04:05")
		assert.Equal(t, "10:30:45", result)
	})

	t.Run("converts UTC to local time", func(t *testing.T) {
		ts := time.Date(2025, 12, 17, 10, 30, 45, 0, time.UTC)
		result := printer.FormatTimestamp(ts, "15:04:05")
		// Result should be the local time equivalent
		expected := ts.Local().Format("15:04:05")
		assert.Equal(t, expected, result)
	})

	t.Run("returns N/A for zero timestamp", func(t *testing.T) {
		var zeroTime time.Time
		result := printer.FormatTimestamp(zeroTime, "15:04:05")
		assert.Equal(t, "N/A", result)
	})

	t.Run("returns N/A for time.Time{}", func(t *testing.T) {
		result := printer.FormatTimestamp(time.Time{}, "2006-01-02 15:04:05")
		assert.Equal(t, "N/A", result)
	})

	t.Run("formats with different layouts", func(t *testing.T) {
		ts := time.Date(2025, 12, 17, 10, 30, 45, 0, time.Local)
		assert.Equal(t, "2025-12-17", printer.FormatTimestamp(ts, "2006-01-02"))
		assert.Equal(t, "10:30", printer.FormatTimestamp(ts, "15:04"))
		assert.Equal(t, "Dec 17 10:30:45", printer.FormatTimestamp(ts, "Jan 02 15:04:05"))
	})
}
