package logclient

import (
	"errors"
	"log"
	"time"

	httpPkg "github.com/berlingoqc/logviewer/pkg/http"
	"github.com/berlingoqc/logviewer/pkg/log/client"
	"github.com/berlingoqc/logviewer/pkg/log/impl/splunk/restapi"
	"github.com/berlingoqc/logviewer/pkg/ty"
)

// Increase the retry limit because Splunk search jobs can take several seconds
// to dispatch on a fresh dev instance.
const maxRetryDoneJob = 30

type SplunkLogSearchClientOptions struct {
	Url string `json:"url"`

	Headers    ty.MS `json:"headers"`
	SearchBody ty.MS `json:"searchBody"`
	// Polling configuration
	PollIntervalSeconds int `json:"pollIntervalSeconds"`
	MaxRetries          int `json:"maxRetries"`
}

type SplunkLogSearchClient struct {
	client restapi.SplunkRestClient

	options SplunkLogSearchClientOptions
}

func (s SplunkLogSearchClient) Get(search *client.LogSearch) (client.LogSearchResult, error) {

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
	for {
		if tryCount >= maxRetries {
			return nil, errors.New("number of retry for splunk job failed")
		}

		time.Sleep(pollInterval)
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

	firstResult, err := s.client.GetSearchResult(searchJobResponse.Sid, 0, search.Size.Value)

	if err != nil {
		return nil, err
	}

	return SplunkLogSearchResult{
		logClient: &s,
		search:    search,
		results:   []restapi.SearchResultsResponse{firstResult},
	}, nil
}

func GetClient(options SplunkLogSearchClientOptions) (client.LogClient, error) {

	target := restapi.SplunkTarget{
		Endpoint: options.Url,
	}

	// If headers include Authorization or other fixed headers, pass them as
	// an Auth implementation so GET requests also include those headers.
	if options.Headers != nil && len(options.Headers) > 0 {
		// set the Auth on the target so Get requests include the same headers
		target.Auth = httpPkg.HeaderAuth{Headers: options.Headers}
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
