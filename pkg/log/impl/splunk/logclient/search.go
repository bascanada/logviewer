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
	return s.parseResults(&s.results[0]), nil, nil
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

	entries := make([]client.LogEntry, len(searchResponse.Results))

	for i, result := range searchResponse.Results {
		timestamp, err := time.Parse(time.RFC3339, result.GetString("_time"))
		if err != nil {
			log.Println("warning failed to parsed timestamp " + result.GetString("_time"))
		}

		entries[i].Message = result.GetString("_raw")
		entries[i].Timestamp = timestamp
		entries[i].Level = ""
		entries[i].Fields = ty.MI{}

		for k, v := range result {
			if k[0] != '_' {
				entries[i].Fields[k] = v
			}
		}
	}

	return entries

}

func (s SplunkLogSearchResult) Err() <-chan error {
	return nil
}
