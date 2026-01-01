package client

import (
	"context"
	"testing"
	"time"

	"github.com/bascanada/logviewer/pkg/ty"
)

type MockLogSearchResult struct {
	Entries []LogEntry
	Channel chan []LogEntry
    Search *LogSearch
}

func (m *MockLogSearchResult) GetSearch() *LogSearch {
    if m.Search != nil {
        return m.Search
    }
	return &LogSearch{Options: ty.MI{"__context_id__": "test-ctx"}}
}

func (m *MockLogSearchResult) GetEntries(ctx context.Context) ([]LogEntry, chan []LogEntry, error) {
	return m.Entries, m.Channel, nil
}

func (m *MockLogSearchResult) GetFields(ctx context.Context) (ty.UniSet[string], chan ty.UniSet[string], error) {
	return nil, nil, nil
}

func (m *MockLogSearchResult) GetPaginationInfo() *PaginationInfo {
	return nil
}

func (m *MockLogSearchResult) Err() <-chan error {
	return nil
}

func TestMultiLogSearchResult_GetEntries_Streaming(t *testing.T) {
	// Setup mock results
	ch1 := make(chan []LogEntry)
	mock1 := &MockLogSearchResult{
		Entries: []LogEntry{{Message: "init1", Timestamp: time.Now()}},
		Channel: ch1,
        Search: &LogSearch{Options: ty.MI{"__context_id__": "ctx1"}},
	}

	ch2 := make(chan []LogEntry)
	mock2 := &MockLogSearchResult{
		Entries: []LogEntry{{Message: "init2", Timestamp: time.Now()}},
		Channel: ch2,
        Search: &LogSearch{Options: ty.MI{"__context_id__": "ctx2"}},
	}

	multiRes, err := NewMultiLogSearchResult(&LogSearch{})
	if err != nil {
		t.Fatalf("NewMultiLogSearchResult failed: %v", err)
	}
	multiRes.Add(mock1, nil)
	multiRes.Add(mock2, nil)

	// Call GetEntries
	ctx := context.Background()
	initialEntries, mergedCh, err := multiRes.GetEntries(ctx)

	if err != nil {
		t.Fatalf("GetEntries failed: %v", err)
	}

    // Check initial entries
    if len(initialEntries) != 2 {
        t.Errorf("Expected 2 initial entries, got %d", len(initialEntries))
    }

    // Check if channel is returned
    if mergedCh == nil {
        t.Fatal("Expected merged channel, got nil")
    }

	// Test streaming
	go func() {
		ch1 <- []LogEntry{{Message: "stream1", Timestamp: time.Now()}}
		ch2 <- []LogEntry{{Message: "stream2", Timestamp: time.Now()}}
		close(ch1)
		close(ch2)
	}()

	// Read from merged channel
	count := 0
	for entries := range mergedCh {
		count++
        for _, e := range entries {
            if e.Message == "stream1" {
                if e.ContextID != "ctx1" {
                     t.Errorf("Expected ContextID ctx1 for stream1, got %s", e.ContextID)
                }
            } else if e.Message == "stream2" {
                 if e.ContextID != "ctx2" {
                     t.Errorf("Expected ContextID ctx2 for stream2, got %s", e.ContextID)
                }
            }
        }
	}
    
    // We expect at least 1 or 2 batches depending on how the loop and go scheduler work.
    // The loop "for entries := range mergedCh" will exit when mergedChannel is closed.
    // mergedChannel is closed when ch1 and ch2 are closed.
}
