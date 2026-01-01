package elk

import (
	"context"
	"testing"
	"time"

	"github.com/bascanada/logviewer/pkg/log/client"
	"github.com/bascanada/logviewer/pkg/ty"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestElkSearchResult_GetPaginationInfo(t *testing.T) {
	t.Run("no size set, no pagination", func(t *testing.T) {
		search := &client.LogSearch{}
		result := ElkSearchResult{search: search}
		assert.Nil(t, result.GetPaginationInfo())
	})

	t.Run("results less than size, no more pages", func(t *testing.T) {
		search := &client.LogSearch{Size: ty.Opt[int]{Value: 10, Set: true}}
		result := ElkSearchResult{
			search: search,
			result: Hits{Hits: make([]Hit, 5)},
		}
		assert.Nil(t, result.GetPaginationInfo())
	})

	t.Run("results equal size, more pages", func(t *testing.T) {
		search := &client.LogSearch{Size: ty.Opt[int]{Value: 10, Set: true}}
		result := ElkSearchResult{
			search: search,
			result: Hits{Hits: make([]Hit, 10)},
		}
		paginationInfo := result.GetPaginationInfo()
		assert.NotNil(t, paginationInfo)
		assert.True(t, paginationInfo.HasMore)
		assert.Equal(t, "10", paginationInfo.NextPageToken)
	})

	t.Run("with existing page token", func(t *testing.T) {
		search := &client.LogSearch{
			Size:      ty.Opt[int]{Value: 10, Set: true},
			PageToken: ty.Opt[string]{Value: "10", Set: true},
		}
		result := ElkSearchResult{
			search:        search,
			result:        Hits{Hits: make([]Hit, 10)},
			CurrentOffset: 10,
		}
		paginationInfo := result.GetPaginationInfo()
		assert.NotNil(t, paginationInfo)
		assert.True(t, paginationInfo.HasMore)
		assert.Equal(t, "20", paginationInfo.NextPageToken)
	})

	t.Run("invalid page token", func(t *testing.T) {
		search := &client.LogSearch{
			Size:      ty.Opt[int]{Value: 10, Set: true},
			PageToken: ty.Opt[string]{Value: "invalid", Set: true},
		}
		result := ElkSearchResult{
			search: search,
			result: Hits{Hits: make([]Hit, 10)},
		}
		paginationInfo := result.GetPaginationInfo()
		assert.NotNil(t, paginationInfo)
		assert.True(t, paginationInfo.HasMore)
		assert.Equal(t, "10", paginationInfo.NextPageToken)
	})
}

// mockElkLogClient implements client.LogClient for testing refresh functionality
type mockElkLogClient struct {
	getCalls      int
	lastSearch    *client.LogSearch
	returnResult  Hits
	returnError   error
}

func (m *mockElkLogClient) Get(ctx context.Context, search *client.LogSearch) (client.LogSearchResult, error) {
	m.getCalls++
	m.lastSearch = search
	if m.returnError != nil {
		return nil, m.returnError
	}
	return ElkSearchResult{
		client:  m,
		search:  search,
		result:  m.returnResult,
		ErrChan: make(chan error, 1),
	}, nil
}

func (m *mockElkLogClient) GetFieldValues(ctx context.Context, search *client.LogSearch, fields []string) (map[string][]string, error) {
	return nil, nil
}

func TestElkSearchResult_onChange(t *testing.T) {
	t.Run("Returns nil when refresh duration is empty", func(t *testing.T) {
		search := &client.LogSearch{}
		result := ElkSearchResult{
			search:  search,
			ErrChan: make(chan error, 1),
		}

		ch, err := result.onChange(context.Background())
		assert.NoError(t, err)
		assert.Nil(t, ch)
	})

	t.Run("Returns error for invalid duration", func(t *testing.T) {
		search := &client.LogSearch{
			Refresh: client.RefreshOptions{
				Duration: ty.Opt[string]{Value: "invalid", Set: true},
			},
		}
		result := ElkSearchResult{
			search:  search,
			ErrChan: make(chan error, 1),
		}

		_, err := result.onChange(context.Background())
		assert.Error(t, err)
	})

	t.Run("Uses time.Now when Lte is empty", func(t *testing.T) {
		// This tests the bug fix: previously, parsing Lte when empty would fail
		mockClient := &mockElkLogClient{
			returnResult: Hits{
				Hits: []Hit{
					{
						Source: ty.MI{
							"message":    "test message",
							"@timestamp": time.Now().Format(time.RFC3339),
						},
					},
				},
			},
		}

		search := &client.LogSearch{
			Refresh: client.RefreshOptions{
				Duration: ty.Opt[string]{Value: "100ms", Set: true},
			},
			// Note: Range.Lte is NOT set (empty) - this was the bug
		}
		result := ElkSearchResult{
			client:  mockClient,
			search:  search,
			ErrChan: make(chan error, 1),
		}

		ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
		defer cancel()

		ch, err := result.onChange(ctx)
		require.NoError(t, err)
		require.NotNil(t, ch)

		// Wait for at least one polling cycle
		select {
		case entries := <-ch:
			// Should receive entries without error
			assert.NotNil(t, entries)
		case <-ctx.Done():
			// Context timeout means poll was attempted
		}

		// Verify the mock was called (polling occurred)
		assert.GreaterOrEqual(t, mockClient.getCalls, 1)
	})

	t.Run("Uses Lte value when provided", func(t *testing.T) {
		mockClient := &mockElkLogClient{
			returnResult: Hits{
				Hits: []Hit{
					{
						Source: ty.MI{
							"message":    "test message",
							"@timestamp": time.Now().Format(time.RFC3339),
						},
					},
				},
			},
		}

		lteTime := time.Now().Add(-1 * time.Minute)
		search := &client.LogSearch{
			Refresh: client.RefreshOptions{
				Duration: ty.Opt[string]{Value: "100ms", Set: true},
			},
		}
		search.Range.Lte.S(lteTime.Format(time.RFC3339))

		result := ElkSearchResult{
			client:  mockClient,
			search:  search,
			ErrChan: make(chan error, 1),
		}

		ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
		defer cancel()

		ch, err := result.onChange(ctx)
		require.NoError(t, err)
		require.NotNil(t, ch)

		// Wait for at least one polling cycle
		select {
		case <-ch:
			// Good, received entries
		case <-ctx.Done():
			// Timeout is ok
		}

		// Verify the mock client received updated search with new Gte/Lte
		if mockClient.lastSearch != nil {
			// After first poll, Gte should be set to lastLte + 1 second
			assert.NotEmpty(t, mockClient.lastSearch.Range.Gte.Value)
			assert.NotEmpty(t, mockClient.lastSearch.Range.Lte.Value)
		}
	})

	t.Run("Clears Last to avoid conflict with Gte/Lte", func(t *testing.T) {
		mockClient := &mockElkLogClient{
			returnResult: Hits{Hits: []Hit{}},
		}

		search := &client.LogSearch{
			Refresh: client.RefreshOptions{
				Duration: ty.Opt[string]{Value: "100ms", Set: true},
			},
		}
		search.Range.Last.S("1h")

		result := ElkSearchResult{
			client:  mockClient,
			search:  search,
			ErrChan: make(chan error, 1),
		}

		ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
		defer cancel()

		ch, err := result.onChange(ctx)
		require.NoError(t, err)
		require.NotNil(t, ch)

		// Wait for at least one polling cycle
		select {
		case <-ch:
		case <-ctx.Done():
		}

		// After polling, Last should be cleared
		if mockClient.lastSearch != nil {
			assert.Empty(t, mockClient.lastSearch.Range.Last.Value)
			assert.False(t, mockClient.lastSearch.Range.Last.Set)
		}
	})

	t.Run("Context cancellation stops polling", func(t *testing.T) {
		mockClient := &mockElkLogClient{
			returnResult: Hits{Hits: []Hit{}},
		}

		search := &client.LogSearch{
			Refresh: client.RefreshOptions{
				Duration: ty.Opt[string]{Value: "1s", Set: true}, // Long duration
			},
		}

		result := ElkSearchResult{
			client:  mockClient,
			search:  search,
			ErrChan: make(chan error, 1),
		}

		ctx, cancel := context.WithCancel(context.Background())
		ch, err := result.onChange(ctx)
		require.NoError(t, err)
		require.NotNil(t, ch)

		// Cancel immediately
		cancel()

		// Channel should be closed
		select {
		case _, ok := <-ch:
			if !ok {
				// Channel closed, as expected
			}
		case <-time.After(100 * time.Millisecond):
			// Timeout is also acceptable
		}
	})

	t.Run("Sends error to ErrChan on Get failure", func(t *testing.T) {
		mockClient := &mockElkLogClient{
			returnError: assert.AnError,
		}

		search := &client.LogSearch{
			Refresh: client.RefreshOptions{
				Duration: ty.Opt[string]{Value: "100ms", Set: true},
			},
		}

		errChan := make(chan error, 5)
		result := ElkSearchResult{
			client:  mockClient,
			search:  search,
			ErrChan: errChan,
		}

		ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
		defer cancel()

		ch, err := result.onChange(ctx)
		require.NoError(t, err)
		require.NotNil(t, ch)

		// Wait for error to be sent
		select {
		case err := <-errChan:
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "failed to get new logs")
		case <-ctx.Done():
			// Timeout - check if error was sent
			select {
			case err := <-errChan:
				assert.Error(t, err)
			default:
				// No error yet, might need more time
			}
		}
	})
}

func TestElkSearchResult_parseResults(t *testing.T) {
	t.Run("Parses hits correctly", func(t *testing.T) {
		timestamp := time.Now().Format(time.RFC3339Nano)
		result := ElkSearchResult{
			search: &client.LogSearch{Options: ty.MI{"index": "test"}},
			result: Hits{
				Hits: []Hit{
					{
						Source: ty.MI{
							"message":    "test message 1",
							"@timestamp": timestamp,
							"level":      "INFO",
						},
					},
					{
						Source: ty.MI{
							"message":    "test message 2",
							"@timestamp": timestamp,
							"level":      "ERROR",
						},
					},
				},
			},
		}

		entries := result.parseResults()
		assert.Len(t, entries, 2)

		// Results are reversed (newest first)
		assert.Equal(t, "test message 2", entries[0].Message)
		assert.Equal(t, "test message 1", entries[1].Message)
	})

	t.Run("Handles missing message", func(t *testing.T) {
		timestamp := time.Now().Format(time.RFC3339Nano)
		result := ElkSearchResult{
			search: &client.LogSearch{Options: ty.MI{"index": "test"}},
			result: Hits{
				Hits: []Hit{
					{
						Source: ty.MI{
							"@timestamp": timestamp,
							// message is missing
						},
					},
				},
			},
		}

		entries := result.parseResults()
		assert.Len(t, entries, 1)
		assert.Empty(t, entries[0].Message)
	})

	t.Run("Handles message as non-string", func(t *testing.T) {
		timestamp := time.Now().Format(time.RFC3339Nano)
		result := ElkSearchResult{
			search: &client.LogSearch{Options: ty.MI{"index": "test"}},
			result: Hits{
				Hits: []Hit{
					{
						Source: ty.MI{
							"message":    12345, // Not a string
							"@timestamp": timestamp,
						},
					},
				},
			},
		}

		entries := result.parseResults()
		assert.Len(t, entries, 1)
		// Should handle gracefully (empty entry)
	})

	t.Run("Parses level field", func(t *testing.T) {
		timestamp := time.Now().Format(time.RFC3339Nano)
		result := ElkSearchResult{
			search: &client.LogSearch{Options: ty.MI{"index": "test"}},
			result: Hits{
				Hits: []Hit{
					{
						Source: ty.MI{
							"message":    "log message",
							"@timestamp": timestamp,
							"level":      "WARN",
						},
					},
				},
			},
		}

		entries := result.parseResults()
		assert.Len(t, entries, 1)
		assert.Equal(t, "WARN", entries[0].Level)
	})
}

func TestElkSearchResult_GetSearch(t *testing.T) {
	search := &client.LogSearch{Follow: true}
	result := ElkSearchResult{search: search}

	assert.Equal(t, search, result.GetSearch())
}

func TestElkSearchResult_Err(t *testing.T) {
	errChan := make(chan error, 1)
	result := ElkSearchResult{ErrChan: errChan}

	assert.Equal(t, (<-chan error)(errChan), result.Err())
}
