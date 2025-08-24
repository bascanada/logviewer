package logclient

import (
	"context"
	"log"
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
