package cloudwatch

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
	"github.com/berlingoqc/logviewer/pkg/log/client"
	"github.com/berlingoqc/logviewer/pkg/ty"
)

type CloudWatchLogSearchResult struct {
	client  CWClient
	queryId string
	search  *client.LogSearch
}

func (r *CloudWatchLogSearchResult) GetSearch() *client.LogSearch {
	return r.search
}

// GetEntries polls for the query results and converts them.
func (r *CloudWatchLogSearchResult) GetEntries(ctx context.Context) ([]client.LogEntry, chan []client.LogEntry, error) {
	// Poll for results until the query is complete.
	var results *cloudwatchlogs.GetQueryResultsOutput
	for {
		var err error
		results, err = r.client.GetQueryResults(ctx, &cloudwatchlogs.GetQueryResultsInput{
			QueryId: &r.queryId,
		})
		if err != nil {
			return nil, nil, err
		}
		if results.Status == types.QueryStatusComplete || results.Status == types.QueryStatusFailed || results.Status == types.QueryStatusCancelled {
			break
		}
		time.Sleep(1 * time.Second) // Poll every second
	}

	// Convert results to LogEntry format
	var entries []client.LogEntry
	for _, resultFields := range results.Results {
		entry := client.LogEntry{Fields: make(ty.MI)}
		for _, field := range resultFields {
			if *field.Field == "@timestamp" {
				// TODO: Parse timestamp
			} else if *field.Field == "@message" {
				entry.Message = *field.Value
			} else {
				entry.Fields[*field.Field] = *field.Value
			}
		}
		entries = append(entries, entry)
	}

	return entries, nil, nil // Live tailing (chan) is not implemented here for simplicity.
}

func (r *CloudWatchLogSearchResult) GetFields() (ty.UniSet[string], chan ty.UniSet[string], error) {
    // This can be implemented by parsing the fields from the GetQueryResults output.
	return make(ty.UniSet[string]), nil, nil
}
