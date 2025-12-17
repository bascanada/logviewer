package ty

import (
	"strings"
	"testing"
	"time"
)

func TestNormalizeTimeValue(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantChanged bool
		wantPrefix  string // Expected prefix for date-based results
	}{
		{
			name:        "empty string",
			input:       "",
			wantChanged: false,
		},
		{
			name:        "duration 1h",
			input:       "1h",
			wantChanged: false,
		},
		{
			name:        "duration 30m",
			input:       "30m",
			wantChanged: false,
		},
		{
			name:        "duration 1h30m",
			input:       "1h30m",
			wantChanged: false,
		},
		{
			name:        "RFC3339",
			input:       "2024-01-15T10:30:00Z",
			wantChanged: false,
		},
		{
			name:        "RFC3339 with timezone",
			input:       "2024-01-15T10:30:00-05:00",
			wantChanged: false,
		},
		{
			name:        "time only HH:MM:SS",
			input:       "10:30:45",
			wantChanged: true,
			wantPrefix:  time.Now().Format("2006-01-02"),
		},
		{
			name:        "time only HH:MM",
			input:       "10:30",
			wantChanged: true,
			wantPrefix:  time.Now().Format("2006-01-02"),
		},
		{
			name:        "date-time without timezone (space)",
			input:       "2024-01-15 10:30:00",
			wantChanged: true,
			wantPrefix:  "2024-01-15",
		},
		{
			name:        "date-time without timezone (T)",
			input:       "2024-01-15T10:30:00",
			wantChanged: true,
			wantPrefix:  "2024-01-15",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, changed := NormalizeTimeValue(tt.input)

			if changed != tt.wantChanged {
				t.Errorf("NormalizeTimeValue(%q) changed = %v, want %v", tt.input, changed, tt.wantChanged)
			}

			if !tt.wantChanged {
				if got != tt.input {
					t.Errorf("NormalizeTimeValue(%q) = %q, want %q (unchanged)", tt.input, got, tt.input)
				}
			} else {
				if tt.wantPrefix != "" && !strings.HasPrefix(got, tt.wantPrefix) {
					t.Errorf("NormalizeTimeValue(%q) = %q, want prefix %q", tt.input, got, tt.wantPrefix)
				}
				// Verify the result is valid RFC3339
				if _, err := time.Parse(time.RFC3339, got); err != nil {
					t.Errorf("NormalizeTimeValue(%q) = %q, not valid RFC3339: %v", tt.input, got, err)
				}
			}
		})
	}
}
