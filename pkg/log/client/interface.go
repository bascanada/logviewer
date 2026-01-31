package client

import (
	"context"
)

// LogClient is the behavioral contract for all log sources.
type LogClient interface {
	Query(ctx context.Context, search LogSearch) ([]LogEntry, error)
	GetFields(ctx context.Context, search LogSearch) (map[string][]string, error)
	GetValues(ctx context.Context, search LogSearch, field string) ([]string, error)
}
