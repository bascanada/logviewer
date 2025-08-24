package cloudwatch

import (
	"context"
	"log"
	"strconv"
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

	// cached results
	entries []client.LogEntry
	fields  ty.UniSet[string]
}

func (r *CloudWatchLogSearchResult) GetSearch() *client.LogSearch {
	return r.search
}

// GetEntries polls for the query results and converts them.
func (r *CloudWatchLogSearchResult) fetchEntries(ctx context.Context) error {
	if len(r.entries) > 0 { // already fetched
		return nil
	}
	var results *cloudwatchlogs.GetQueryResultsOutput
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	for {
		var err error
		results, err = r.client.GetQueryResults(ctx, &cloudwatchlogs.GetQueryResultsInput{QueryId: &r.queryId})
		if err != nil {
			return err
		}
		if results.Status == types.QueryStatusComplete || results.Status == types.QueryStatusFailed || results.Status == types.QueryStatusCancelled {
			break
		}
		select {
		case <-ticker.C:
			continue
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	for _, resultFields := range results.Results {
		entry := client.LogEntry{Fields: make(ty.MI)}
		for _, field := range resultFields {
			if field.Field == nil || field.Value == nil {
				continue
			}
			fName := *field.Field
			fVal := *field.Value
			switch fName {
			case "@timestamp":
				if ts, ok := parseCloudWatchTimestamp(fVal); ok {
					entry.Timestamp = ts
				} else {
					log.Printf("cloudwatch: failed to parse timestamp '%s'", fVal)
				}
			case "@message":
				entry.Message = fVal
			default:
				entry.Fields[fName] = fVal
			}
		}
		r.entries = append(r.entries, entry)
	}
	return nil
}

func (r *CloudWatchLogSearchResult) GetEntries(ctx context.Context) ([]client.LogEntry, chan []client.LogEntry, error) {
	if err := r.fetchEntries(ctx); err != nil {
		return nil, nil, err
	}
	return r.entries, nil, nil
}

func (r *CloudWatchLogSearchResult) GetFields() (ty.UniSet[string], chan ty.UniSet[string], error) {
	// If already computed, return cached
	if len(r.fields) > 0 {
		return r.fields, nil, nil
	}
	// Ensure entries are loaded. Use background context since interface lacks ctx.
	if len(r.entries) == 0 {
		_ = r.fetchEntries(context.Background())
	}
	fields := ty.UniSet[string]{}
	for _, e := range r.entries {
		for k, v := range e.Fields {
			if k == "@message" || k == "@timestamp" || k == "@ptr" || k == "@logStream" || k == "@log" || (len(k) > 0 && k[0] == '@') {
				continue
			}
			ty.AddField(k, v, &fields)
		}
	}
	r.fields = fields
	return fields, nil, nil
}

// parseCloudWatchTimestamp attempts to parse a CloudWatch Logs Insights timestamp.
// Common formats observed:
//  * "2006-01-02 15:04:05.000" (default in Insights results)
//  * time.RFC3339 or RFC3339Nano (defensive)
//  * Milliseconds since epoch (string of digits)
func parseCloudWatchTimestamp(v string) (time.Time, bool) {
	// Primary formats used in typical Insights outputs.
	layouts := []string{"2006-01-02 15:04:05.000", time.RFC3339Nano, time.RFC3339}
	for _, layout := range layouts {
		if ts, err := time.Parse(layout, v); err == nil {
			return ts, true
		}
	}
	// Try epoch millis (string of digits)
	if len(v) >= 13 { // at least millisecond precision length
		if ms, err := strconv.ParseInt(v, 10, 64); err == nil {
			return time.Unix(0, ms*int64(time.Millisecond)), true
		}
	}
	return time.Time{}, false
}
