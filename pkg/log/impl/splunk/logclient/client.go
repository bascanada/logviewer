package logclient

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"
	"time"

	httpPkg "github.com/bascanada/logviewer/pkg/http"
	"github.com/bascanada/logviewer/pkg/log/client"
	"github.com/bascanada/logviewer/pkg/log/impl/splunk/restapi"
	"github.com/bascanada/logviewer/pkg/ty"
)

// Increase the retry limit because Splunk search jobs can take several seconds
// to dispatch on a fresh dev instance.
const maxRetryDoneJob = 30

type SplunkAuthOptions struct {
	Header ty.MS `json:"header" yaml:"header"`
}

type SplunkLogSearchClientOptions struct {
	Url string `json:"url" yaml:"url"`

	Auth       SplunkAuthOptions `json:"auth" yaml:"auth"`
	Headers    ty.MS             `json:"headers" yaml:"headers"`
	SearchBody ty.MS             `json:"searchBody" yaml:"searchBody"`
	// Polling configuration
	PollIntervalSeconds int `json:"pollIntervalSeconds" yaml:"pollIntervalSeconds"`
	MaxRetries          int `json:"maxRetries" yaml:"maxRetries"`
}

type SplunkLogSearchClient struct {
	client restapi.SplunkRestClient

	options SplunkLogSearchClientOptions
}

func (s SplunkLogSearchClient) Get(ctx context.Context, search *client.LogSearch) (client.LogSearchResult, error) {

	// initiate the things and wait for query to be done

	if s.options.Headers == nil {
		s.options.Headers = ty.MS{}
	}

	if s.options.SearchBody == nil {
		s.options.SearchBody = ty.MS{}
	}

	searchRequest, err := getSearchRequest(search)
	if err != nil {
		return nil, err
	}

	searchJobResponse, err := s.client.CreateSearchJob(searchRequest["search"], searchRequest["earliest_time"], searchRequest["latest_time"], s.options.Headers, s.options.SearchBody)
	if err != nil {
		return nil, err
	}

	// configure polling
	pollInterval := time.Duration(1) * time.Second
	maxRetries := maxRetryDoneJob
	if s.options.PollIntervalSeconds > 0 {
		pollInterval = time.Duration(s.options.PollIntervalSeconds) * time.Second
	}
	if s.options.MaxRetries > 0 {
		maxRetries = s.options.MaxRetries
	}

	isDone := false
	tryCount := 0

	// wait until job is done or retries exhausted
	if !search.Refresh.Follow.Value {
		for {
			if tryCount >= maxRetries {
				return nil, errors.New("number of retry for splunk job failed")
			}

			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(pollInterval):
			}
			log.Printf("waiting for splunk job %s to complete (try %d/%d)", searchJobResponse.Sid, tryCount+1, maxRetries)

			status, err := s.client.GetSearchStatus(searchJobResponse.Sid)

			if err != nil {
				return nil, err
			}

			// Guard against responses with no entry
			if len(status.Entry) > 0 {
				isDone = status.Entry[0].Content.IsDone
			} else {
				isDone = false
			}

			if isDone {
				break
			}

			tryCount += 1
		}
	}

	offset := 0
	if search.PageToken.Value != "" {
		var err error
		offset, err = strconv.Atoi(search.PageToken.Value)
		if err != nil {
			return nil, fmt.Errorf("invalid page token: %w", err)
		}
	}

	firstResult, err := s.client.GetSearchResult(searchJobResponse.Sid, offset, search.Size.Value)

	if err != nil {
		return nil, err
	}

	return SplunkLogSearchResult{
		logClient:     &s,
		sid:           searchJobResponse.Sid,
		search:        search,
		results:       []restapi.SearchResultsResponse{firstResult},
		CurrentOffset: offset,
	}, nil
}

func GetClient(options SplunkLogSearchClientOptions) (client.LogClient, error) {

	if options.Url == "" {
		return nil, fmt.Errorf("splunk client Url is empty; set the Url option in config or pass --splunk-endpoint")
	}

	target := restapi.SplunkTarget{
		Endpoint: options.Url,
		Headers:  options.Headers,
	}

	// If headers include Authorization or other fixed headers, pass them as
	// an Auth implementation so GET requests also include those headers.
	if options.Auth.Header != nil && len(options.Auth.Header) > 0 {
		// set the Auth on the target so Get requests include the same headers
		target.Auth = httpPkg.HeaderAuth{Headers: options.Auth.Header}
	}

	restClient, err := restapi.GetSplunkRestClient(target)
	if err != nil {
		return nil, err
	}

	client := SplunkLogSearchClient{
		client:  restClient,
		options: options,
	}

	return client, nil
}
