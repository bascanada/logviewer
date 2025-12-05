package client

import (
	"context"
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
