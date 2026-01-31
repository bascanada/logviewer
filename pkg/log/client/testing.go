package client

import (
	"context"
)

// MockLogClient implements the behavioral interface for testing.
type MockLogClient struct {
	LastSearch LogSearch
	OnQuery    func(search LogSearch) ([]LogEntry, error)
	OnFields   func(search LogSearch) (map[string][]string, error)
	OnValues   func(search LogSearch, field string) ([]string, error)
}

func (m *MockLogClient) Query(ctx context.Context, s LogSearch) ([]LogEntry, error) {
	m.LastSearch = s
	if m.OnQuery != nil {
		return m.OnQuery(s)
	}
	return []LogEntry{}, nil
}

func (m *MockLogClient) GetFields(ctx context.Context, s LogSearch) (map[string][]string, error) {
	if m.OnFields != nil {
		return m.OnFields(s)
	}
	return map[string][]string{"level": {"INFO"}, "message": {"foo"}}, nil
}

func (m *MockLogClient) GetValues(ctx context.Context, s LogSearch, field string) ([]string, error) {
	if m.OnValues != nil {
		return m.OnValues(s, field)
	}
	return []string{}, nil
}
