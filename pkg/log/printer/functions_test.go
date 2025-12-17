package printer_test

import (
	"testing"
	"time"

	"github.com/bascanada/logviewer/pkg/log/printer"
	"github.com/stretchr/testify/assert"
)

func TestExpandJson(t *testing.T) {

	logEntries := []string{
		"get data from json : {\"dadaad\": 2244 }",
	}

	for _, v := range logEntries {
		expandedJson := printer.ExpandJson(v)

		assert.NotEqual(t, "", expandedJson)
	}

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
