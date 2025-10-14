package client

import (
	"context"
	"sort"
	"sync"

	"github.com/bascanada/logviewer/pkg/ty"
)

// MultiLogSearchResult aggregates multiple LogSearchResult objects into a single,
// unified result. It is designed for multi-context queries where logs from
// different sources need to be merged and presented as a single stream.
type MultiLogSearchResult struct {
	// a slice of the individual LogSearchResult objects from each queried context.
	Results []LogSearchResult
	// a slice of errors encountered during the concurrent query execution.
	Errors []error
	// the original LogSearch request that initiated the multi-context query.
	Search *LogSearch
	// mutex to protect concurrent access to Results and Errors slices.
	mutex sync.Mutex
}

// ensure MultiLogSearchResult implements the LogSearchResult interface.
var _ LogSearchResult = (*MultiLogSearchResult)(nil)

// NewMultiLogSearchResult creates and returns a new MultiLogSearchResult.
func NewMultiLogSearchResult(search *LogSearch) *MultiLogSearchResult {
	return &MultiLogSearchResult{
		Search:  search,
		Results: []LogSearchResult{},
		Errors:  []error{},
	}
}

// Add appends a search result and an associated error to the aggregator.
// This method is safe for concurrent use.
func (m *MultiLogSearchResult) Add(result LogSearchResult, err error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if err != nil {
		m.Errors = append(m.Errors, err)
	}
	if result != nil {
		m.Results = append(m.Results, result)
	}
}

// GetSearch returns the original LogSearch request.
func (m *MultiLogSearchResult) GetSearch() *LogSearch {
	return m.Search
}

// GetEntries merges log entries from all successful search results, sorts them
// by timestamp, and returns them. It also populates the ContextID for each entry.
func (m *MultiLogSearchResult) GetEntries(ctx context.Context) ([]LogEntry, chan []LogEntry, error) {
	var allEntries []LogEntry
	var mutex sync.Mutex
	var wg sync.WaitGroup

	for _, result := range m.Results {
		wg.Add(1)
		go func(r LogSearchResult) {
			defer wg.Done()
			entries, _, err := r.GetEntries(ctx)
			if err != nil {
				// In a real-world scenario, you might want to handle this error more gracefully.
				// For now, we'll just skip the results from this source.
				return
			}

			// Populate ContextID for each entry
			contextID := r.GetSearch().Options["__context_id__"].(string)
			for i := range entries {
				entries[i].ContextID = contextID
			}

			mutex.Lock()
			allEntries = append(allEntries, entries...)
			mutex.Unlock()
		}(result)
	}

	wg.Wait()

	// Sort the combined slice of log entries by timestamp.
	sort.Slice(allEntries, func(i, j int) bool {
		return allEntries[i].Timestamp.Before(allEntries[j].Timestamp)
	})

	return allEntries, nil, nil
}

// GetFields is not implemented for MultiLogSearchResult as it's ambiguous
// how to merge fields from different sources.
func (m *MultiLogSearchResult) GetFields(context.Context) (ty.UniSet[string], chan ty.UniSet[string], error) {
	// Returning an empty set as merging fields from multiple contexts is not supported.
	return make(ty.UniSet[string]), nil, nil
}

// GetPaginationInfo returns nil as pagination is not supported for multi-context search results.
func (m *MultiLogSearchResult) GetPaginationInfo() *PaginationInfo {
	// Pagination is not supported for merged results.
	return nil
}