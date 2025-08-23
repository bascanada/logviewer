package restapi

import (
	"fmt"
	"log"
	"strconv"

	"github.com/berlingoqc/logviewer/pkg/http"
	"github.com/berlingoqc/logviewer/pkg/ty"
)

// Struct to hold search job response
type SearchJobResponse struct {
	Sid string `json:"sid"`
}

// Struct to hold search job status response
type JobStatusResponse struct {
	Entry []struct {
		Content struct {
			IsDone bool `json:"isDone"`
		} `json:"content"`
	} `json:"entry"`
}

// Struct to hold search results response
type SearchResultsResponse struct {
	Results []ty.MI `json:"results"`
}

type SplunkTarget struct {
	Endpoint string `json:"endpoint"`
	Auth     http.Auth
}

type SplunkRestClient struct {
	target SplunkTarget
	client http.HttpClient
}

func (src SplunkRestClient) CreateSearchJob(
	searchQuery string,
	earliestTime string,
	latestTime string,

	headers ty.MS,

	data ty.MS,
) (SearchJobResponse, error) {
	var searchJobResponse SearchJobResponse

	searchPath := fmt.Sprintf("/search/jobs")

	// Ensure data map is initialized
	if data == nil {
		data = ty.MS{}
	}

	// Build the form data for the search job in a small helper so tests can
	// validate its shape without performing HTTP calls.
	body := buildSearchJobData(searchQuery, earliestTime, latestTime, data)

	err := src.client.PostData(searchPath, headers, body, &searchJobResponse, src.target.Auth)

	return searchJobResponse, err

}

// buildSearchJobData returns the form body to POST to Splunk to create a
// search job. It defaults empty time ranges to -24h@h / now and only sets the
// standard earliest_time/latest_time fields (avoids custom.dispatch.*).
func buildSearchJobData(searchQuery, earliestTime, latestTime string, data ty.MS) ty.MS {
	if data == nil {
		data = ty.MS{}
	}

	if earliestTime == "" && latestTime == "" {
		earliestTime = "-24h@h"
		latestTime = "now"
	}

	if latestTime != "" {
		data["latest_time"] = latestTime
	}

	if earliestTime != "" {
		data["earliest_time"] = earliestTime
	}

	data["search"] = "search " + searchQuery
	return data
}

func (src SplunkRestClient) GetSearchStatus(
	sid string,
) (JobStatusResponse, error) {
	var response JobStatusResponse

	searchPath := fmt.Sprintf("/search/jobs/%s", sid)

	queryParams := ty.MS{
		"output_mode": "json",
	}

	err := src.client.Get(searchPath, queryParams, nil, &response, src.target.Auth)
	if err == nil {
		if len(response.Entry) > 0 {
			if http.DebugEnabled() {
				log.Printf("[SPLUNK-STATUS] entry_count=%d isDone=%v\n", len(response.Entry), response.Entry[0].Content.IsDone)
			}
		} else {
			if http.DebugEnabled() {
				log.Printf("[SPLUNK-STATUS] entry_count=0\n")
			}
		}
	}
	return response, err
}

func (src SplunkRestClient) GetSearchResult(
	sid string,
	offset int,
	count int,
) (SearchResultsResponse, error) {
	var response SearchResultsResponse

	searchPath := fmt.Sprintf("/search/jobs/%s/events", sid)

	queryParams := ty.MS{
		"output_mode": "json",
		"offset":      strconv.Itoa(offset),
		"count":       strconv.Itoa(count),
	}

	err := src.client.Get(searchPath, queryParams, nil, &response, src.target.Auth)
	return response, err

}

func GetSplunkRestClient(
	target SplunkTarget,
) (SplunkRestClient, error) {
	return SplunkRestClient{
		target: target,
		client: http.GetClient(target.Endpoint),
	}, nil
}
