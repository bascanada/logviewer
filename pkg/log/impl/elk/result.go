package elk

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/bascanada/logviewer/pkg/log/client"
	"github.com/bascanada/logviewer/pkg/ty"
)

type Hit struct {
	Index  string `json:"_index"`
	Type   string `json:"_type"`
	Id     string `json:"_id"`
	Score  int32  `json:"_score"`
	Source ty.MI  `json:"_source"`
}

type Hits struct {
	// total
	// max_score
	Hits []Hit `json:"hits"`
}

type ElkSearchResult struct {
	client client.LogClient
	search *client.LogSearch
	result Hits

	entriesChan chan ty.UniSet[string]
	// store loaded entries

	// store extracted fields
	// parsed offset from the incoming page token (set by client.Get)
	CurrentOffset int
	ErrChan       chan error
}

func GetSearchResult(client client.LogClient, search *client.LogSearch, hits Hits) ElkSearchResult {
	return ElkSearchResult{
		client:  client,
		search:  search,
		result:  hits,
		ErrChan: make(chan error, 1),
	}
}

func (sr ElkSearchResult) GetSearch() *client.LogSearch {
	return sr.search
}

func (sr ElkSearchResult) GetEntries(context context.Context) ([]client.LogEntry, chan []client.LogEntry, error) {

	entries := sr.parseResults()

	c, err := sr.onChange(context)

	return entries, c, err
}

func (sr ElkSearchResult) GetFields(ctx context.Context) (ty.UniSet[string], chan ty.UniSet[string], error) {

	fields := ty.UniSet[string]{}

	for _, h := range sr.result.Hits {
		for k, v := range h.Source {
			if k == "message" || k == "@timestamp" {
				continue
			}
			ty.AddField(k, v, &fields)
		}
	}
	return fields, nil, nil
}

func (sr ElkSearchResult) parseResults() []client.LogEntry {
	size := len(sr.result.Hits)

	entries := make([]client.LogEntry, size)

	log.Printf("receive %d for %s"+ty.LB, len(entries), sr.search.Options.GetString("index"))

	for i, h := range sr.result.Hits {
		message, b := h.Source["message"].(string)
		if !b {
			fmt.Printf("message is not string : %+v \n", h.Source["message"])
			entries[size-i-1] = client.LogEntry{}
			continue
		}
		if timestamp, b1 := h.Source["@timestamp"].(string); b1 {
			// Try high precision first, then standard RFC3339
			var date time.Time
			var err error
			if date, err = time.Parse(time.RFC3339Nano, timestamp); err != nil {
				if date, err = time.Parse(ty.Format, timestamp); err != nil {
					// Fallback: leave zero-value time (or set to now)
					date = time.Time{}
				}
			}

			var level string
			if h.Source["level"] != nil {
				level, _ = h.Source["level"].(string)
			}

			entries[size-i-1] = client.LogEntry{
				Message:   message,
				Timestamp: date,
				Level:     level, Fields: h.Source}
		} else {
			fmt.Printf("timestamp is not string : %+v \n", h.Source["@timestamp"])
		}
	}

	return entries
}

func (sr ElkSearchResult) GetPaginationInfo() *client.PaginationInfo {
	if !sr.search.Size.Set {
		return nil
	}

	// Use the offset parsed and stored by the client.Get implementation. If
	// the result was constructed manually (e.g. in tests) the default is 0.
	currentOffset := sr.CurrentOffset

	numResults := len(sr.result.Hits)

	// If we got fewer results than requested, this is the last page
	if numResults < sr.search.Size.Value {
		return nil
	}

	return &client.PaginationInfo{
		HasMore:       true,
		NextPageToken: strconv.Itoa(currentOffset + numResults),
	}
}

func (sr ElkSearchResult) Err() <-chan error {
	return sr.ErrChan
}

func (sr ElkSearchResult) onChange(ctx context.Context) (chan []client.LogEntry, error) {
	if sr.search.Refresh.Duration.Value == "" {
		return nil, nil
	}

	duration, err := time.ParseDuration(sr.search.Refresh.Duration.Value)
	if err != nil {
		return nil, err
	}

	c := make(chan []client.LogEntry, 5)
	go func() {
		for {
			select {
			case <-time.After(duration):
				{
					date, err := time.Parse(time.RFC3339, sr.search.Range.Lte.Value)
					if err != nil {
						sr.ErrChan <- fmt.Errorf("error parsing Gte.Value: %w", err)
						continue
					}
					date = date.Add(time.Second * 1)
					sr.search.Range.Gte.Value = date.Format(time.RFC3339)
					sr.search.Range.Lte.Value = time.Now().Format(time.RFC3339)
					result, err1 := sr.client.Get(ctx, sr.search)
					if err1 != nil {
						sr.ErrChan <- fmt.Errorf("failed to get new logs: %w", err1)
					}
					c <- result.(ElkSearchResult).parseResults()
				}
			case <-ctx.Done():
				close(c)
				return
			}
		}
	}()
	return c, nil
}
