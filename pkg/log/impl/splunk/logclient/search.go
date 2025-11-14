package logclient

import (
	"context"
	"log"
	"strconv"
	"time"

	"github.com/bascanada/logviewer/pkg/log/client"
	"github.com/bascanada/logviewer/pkg/log/impl/splunk/restapi"
	"github.com/bascanada/logviewer/pkg/ty"
)

type SplunkLogSearchResult struct {
	logClient *SplunkLogSearchClient
	sid       string
	search    *client.LogSearch

	results []restapi.SearchResultsResponse

	entriesChan chan ty.UniSet[string]
	// parsed offset from the incoming page token (set by client.Get)
	CurrentOffset int
}

func (s SplunkLogSearchResult) GetSearch() *client.LogSearch {
	return s.search
}

func (s SplunkLogSearchResult) GetEntries(context context.Context) ([]client.LogEntry, chan []client.LogEntry, error) {

	initialEntries := s.parseResults(&s.results[0])

	if s.search.Refresh.Follow.Value {
		logEntriesChan := make(chan []client.LogEntry, 1)

		// Use the new polling strategy if configured
		if s.logClient.options.UsePollingFollow {
			go func() {
				defer close(logEntriesChan)

				pollInterval := 5 * time.Second
				if s.logClient.options.PollIntervalSeconds > 0 {
					pollInterval = time.Duration(s.logClient.options.PollIntervalSeconds) * time.Second
				}

				lastTimestamp := time.Now()
				if len(initialEntries) > 0 {
					lastTimestamp = initialEntries[len(initialEntries)-1].Timestamp
				}

				ticker := time.NewTicker(pollInterval)
				defer ticker.Stop()

				for {
					select {
					case <-context.Done():
						return
					case <-ticker.C:
						// Create a new search for the time window since our last event
						newSearch := *s.search
						newSearch.Range.Gte.S(lastTimestamp.Format(time.RFC3339)) // From last event
						newSearch.Range.Lte.S("now")                              // To now
						newSearch.Range.Last.U()                                  // Unset "last" to prefer Gte/Lte
						newSearch.PageToken.U()                                   // Not paginating
						newSearch.Refresh.Follow.U()                              // Unset follow to prevent recursion

						newResult, err := s.logClient.Get(context, &newSearch)
						if err != nil {
							log.Printf("error during splunk follow-poll: %v", err)
							continue
						}

						newEntries, _, err := newResult.GetEntries(context)
						if err != nil {
							log.Printf("error getting entries from follow-poll: %v", err)
							continue
						}

						if len(newEntries) > 0 {
							lastTimestamp = newEntries[len(newEntries)-1].Timestamp
							logEntriesChan <- newEntries
						}
					}
				}
			}()
		} else {
			// Use the original real-time (rt_search) polling logic
			go func() {
				defer close(logEntriesChan)
				offset := s.CurrentOffset + len(s.results[0].Results)
				pollInterval := 5 * time.Second
				if s.logClient.options.PollIntervalSeconds > 0 {
					pollInterval = time.Duration(s.logClient.options.PollIntervalSeconds) * time.Second
				}
				for {
					select {
					case <-context.Done():
						return
					case <-time.After(pollInterval):
						results, err := s.logClient.client.GetSearchResult(s.sid, offset, 0)
						if err != nil {
							log.Printf("error fetching splunk results: %v", err)
							continue
						}
						if len(results.Results) > 0 {
							logEntriesChan <- s.parseResults(&results)
							offset += len(results.Results)
						}
					}
				}
			}()
		}

		return initialEntries, logEntriesChan, nil
	}
	return initialEntries, nil, nil
}

func (s SplunkLogSearchResult) GetFields(ctx context.Context) (ty.UniSet[string], chan ty.UniSet[string], error) {
	fields := ty.UniSet[string]{}

	for _, resultEntry := range s.results {
		for _, result := range resultEntry.Results {
			for k, v := range result {
				if k[0] == '_' {
					continue
				}

				ty.AddField(k, v, &fields)
			}
		}
	}

	return fields, nil, nil
}

func (s SplunkLogSearchResult) GetPaginationInfo() *client.PaginationInfo {
	if !s.search.Size.Set {
		return nil
	}

	// Use the offset parsed and stored by the client.Get implementation. If the
	// result was constructed manually (e.g. in tests) the default is 0 which
	// preserves previous behavior.
	currentOffset := s.CurrentOffset

	numResults := len(s.results[0].Results)

	// If we got fewer results than requested, this is the last page
	if numResults < s.search.Size.Value {
		return nil
	}

	return &client.PaginationInfo{
		HasMore:       true,
		NextPageToken: strconv.Itoa(currentOffset + numResults),
	}
}

func (s SplunkLogSearchResult) parseResults(searchResponse *restapi.SearchResultsResponse) []client.LogEntry {

	n := len(searchResponse.Results)
	entries := make([]client.LogEntry, n)

	for i, result := range searchResponse.Results {
		timestamp, err := time.Parse(time.RFC3339, result.GetString("_time"))
		if err != nil {
			log.Println("warning failed to parsed timestamp " + result.GetString("_time"))
		}

		// Create the LogEntry struct first
		entry := client.LogEntry{
			Message:   result.GetString("_raw"),
			Timestamp: timestamp,
			Level:     "",
			Fields:    ty.MI{},
		}

		for k, v := range result {
			if k[0] != '_' {
				entry.Fields[k] = v
			}
		}

		// FIX: Splunk's results are newest-first (i=0 is newest).
		// We place our newest item (i=0) at the end of our slice (index n-1-i)
		// so the final 'entries' slice is oldest-first.
		entries[n-1-i] = entry
	}

	return entries
}
