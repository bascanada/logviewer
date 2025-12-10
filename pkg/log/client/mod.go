package client

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/bascanada/logviewer/pkg/ty"
)

type LogEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Message   string    `json:"message"`
	Level     string    `json:"level"`
	Fields    ty.MI     `json:"fields"`
	ContextID string    `json:"context_id"`
}

// Field provides case-insensitive field access for templates.
// Usage: {{.Field "level"}} or {{.Field "thread"}}
func (e LogEntry) Field(key string) interface{} {
	// Check struct fields first
	switch key {
	case "level", "Level":
		return e.Level
	case "message", "Message":
		return e.Message
	case "timestamp", "Timestamp":
		return e.Timestamp
	}
	// Try Fields map with exact match
	if val, ok := e.Fields[key]; ok {
		return val
	}
	// Try capitalized version
	if len(key) > 0 && key[0] >= 'a' && key[0] <= 'z' {
		capKey := string(key[0]-32) + key[1:]
		if val, ok := e.Fields[capKey]; ok {
			return val
		}
	}
	return ""
}

// Result of the search , may be used to get more log
// or keep updated
type LogSearchResult interface {
	GetSearch() *LogSearch
	GetEntries(context context.Context) ([]LogEntry, chan []LogEntry, error)
	GetFields(context context.Context) (ty.UniSet[string], chan ty.UniSet[string], error)
	GetPaginationInfo() *PaginationInfo
	Err() <-chan error
}

type PaginationInfo struct {
	HasMore       bool
	NextPageToken string
}

// Client to start a log search
type LogClient interface {
	Get(ctx context.Context, search *LogSearch) (LogSearchResult, error)
}

// ExtractJSONFromEntry extracts JSON fields from the entry's Message and populates
// entry.Fields, entry.Level, entry.Message, and entry.Timestamp based on the search
// configuration. This is used by both the reader and printer to avoid code duplication.
// This function is idempotent - it's safe to call multiple times on the same entry.
func ExtractJSONFromEntry(entry *LogEntry, search *LogSearch) {
	if !search.FieldExtraction.Json.Value {
		return
	}

	// Skip if JSON already extracted (idempotency check)
	// If message doesn't contain '{', it's either already extracted or not JSON
	if !strings.Contains(entry.Message, "{") {
		return
	}

	var jsonMap map[string]interface{}
	jsonContent := entry.Message
	// Find the last occurrence of '{' to extract JSON from mixed content
	if idx := strings.LastIndex(entry.Message, "{"); idx != -1 {
		jsonContent = entry.Message[idx:]
	} else {
		return // No JSON found
	}

	decoder := json.NewDecoder(strings.NewReader(jsonContent))
	if err := decoder.Decode(&jsonMap); err != nil {
		return // Not valid JSON, leave entry unchanged
	}

	// Get configured key names or use defaults
	msgKey := "message"
	if search.FieldExtraction.JsonMessageKey.Set {
		msgKey = search.FieldExtraction.JsonMessageKey.Value
	}
	levelKey := "level"
	if search.FieldExtraction.JsonLevelKey.Set {
		levelKey = search.FieldExtraction.JsonLevelKey.Value
	}
	tsKey := "timestamp"
	if search.FieldExtraction.JsonTimestampKey.Set {
		tsKey = search.FieldExtraction.JsonTimestampKey.Value
	}

	// Extract all fields except the special ones
	if entry.Fields == nil {
		entry.Fields = make(ty.MI)
	}
	for k, v := range jsonMap {
		if k == msgKey || k == levelKey || k == tsKey {
			continue
		}
		entry.Fields[k] = v
	}

	// Extract message
	if v, ok := jsonMap[msgKey]; ok {
		if s, ok := v.(string); ok {
			entry.Message = s
		}
	}

	// Extract level
	if v, ok := jsonMap[levelKey]; ok {
		if s, ok := v.(string); ok {
			entry.Level = s
		}
	}

	// Extract timestamp
	if v, ok := jsonMap[tsKey]; ok {
		if parsed, err := parseTimestamp(v); err == nil && !parsed.IsZero() {
			entry.Timestamp = parsed
		}
	}
}

// parseTimestamp attempts to parse various timestamp formats
func parseTimestamp(value interface{}) (time.Time, error) {
	var timeStr string
	switch v := value.(type) {
	case string:
		timeStr = v
	case float64:
		// Unix timestamp
		return time.Unix(int64(v), 0), nil
	default:
		return time.Time{}, fmt.Errorf("unsupported timestamp type: %T", value)
	}

	// Try common timestamp formats
	formats := []string{
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02T15:04:05.999999999Z07:00",
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02 15:04:05",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, timeStr); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse timestamp: %s", timeStr)
}
