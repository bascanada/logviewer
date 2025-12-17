package ty

import (
	"regexp"
	"time"
)

// Time-only formats (HH:MM:SS or HH:MM)
var timeOnlyFormats = []string{
	"15:04:05",
	"15:04",
}

// Date-time formats without timezone
var dateTimeFormats = []string{
	"2006-01-02 15:04:05",
	"2006-01-02 15:04",
	"2006-01-02T15:04:05",
	"2006-01-02T15:04",
}

// durationRegex matches Go duration strings like "1h", "30m", "1h30m"
var durationRegex = regexp.MustCompile(`^-?(\d+(\.\d+)?(ns|us|µs|ms|s|m|h))+$`)

// NormalizeTimeValue attempts to normalize a time value to RFC3339 format.
// It handles:
// - Duration strings (1h, 30m) - returned as-is
// - RFC3339 timestamps - returned as-is
// - Time-only (HH:MM:SS, HH:MM) - converted to today's date at that time
// - Date-time without timezone - converted to local timezone
//
// Returns the normalized value and whether it was modified.
func NormalizeTimeValue(value string) (string, bool) {
	if value == "" {
		return value, false
	}

	// Check if it's a duration (like "1h", "30m")
	if durationRegex.MatchString(value) {
		return value, false
	}

	// Check if it's already RFC3339
	if _, err := time.Parse(time.RFC3339, value); err == nil {
		return value, false
	}
	if _, err := time.Parse(time.RFC3339Nano, value); err == nil {
		return value, false
	}

	// Try time-only formats (HH:MM:SS, HH:MM)
	for _, format := range timeOnlyFormats {
		if t, err := time.ParseInLocation(format, value, time.Local); err == nil {
			// Use today's date with the parsed time
			now := time.Now()
			fullTime := time.Date(now.Year(), now.Month(), now.Day(),
				t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), time.Local)
			return fullTime.Format(time.RFC3339), true
		}
	}

	// Try date-time formats without timezone
	for _, format := range dateTimeFormats {
		if t, err := time.ParseInLocation(format, value, time.Local); err == nil {
			return t.Format(time.RFC3339), true
		}
	}

	// Return original value if no format matched
	return value, false
}
