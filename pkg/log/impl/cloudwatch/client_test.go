package cloudwatch

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
	"github.com/berlingoqc/logviewer/pkg/log/client"
	"github.com/berlingoqc/logviewer/pkg/ty"
	"github.com/stretchr/testify/assert"
)

// mockCWClient is a mock implementation of the CWClient interface.
type mockCWClient struct {
	StartQueryFunc      func(ctx context.Context, params *cloudwatchlogs.StartQueryInput, optFns ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.StartQueryOutput, error)
	GetQueryResultsFunc func(ctx context.Context, params *cloudwatchlogs.GetQueryResultsInput, optFns ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.GetQueryResultsOutput, error)
	FilterLogEventsFunc func(ctx context.Context, params *cloudwatchlogs.FilterLogEventsInput, optFns ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.FilterLogEventsOutput, error)
}

func (m *mockCWClient) StartQuery(ctx context.Context, params *cloudwatchlogs.StartQueryInput, optFns ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.StartQueryOutput, error) {
	return m.StartQueryFunc(ctx, params, optFns...)
}

func (m *mockCWClient) GetQueryResults(ctx context.Context, params *cloudwatchlogs.GetQueryResultsInput, optFns ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.GetQueryResultsOutput, error) {
	return m.GetQueryResultsFunc(ctx, params, optFns...)
}
func (m *mockCWClient) FilterLogEvents(ctx context.Context, params *cloudwatchlogs.FilterLogEventsInput, optFns ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.FilterLogEventsOutput, error) {
	if m.FilterLogEventsFunc != nil { return m.FilterLogEventsFunc(ctx, params, optFns...) }
	return &cloudwatchlogs.FilterLogEventsOutput{}, nil
}

func TestGetLogClient(t *testing.T) {
	// This test checks that providing a profile that doesn't exist returns an error.
	// This is the most we can test without a real AWS session or extensive mocking.
	t.Run("invalid profile", func(t *testing.T) {
		options := ty.MI{
			"profile": "this-profile-does-not-exist",
		}
		_, err := GetLogClient(options)
		assert.Error(t, err)
	})
}

func TestCloudWatchLogClient_Get(t *testing.T) {
	mockClient := &mockCWClient{
		StartQueryFunc: func(ctx context.Context, params *cloudwatchlogs.StartQueryInput, optFns ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.StartQueryOutput, error) {
			assert.Equal(t, "test-group", *params.LogGroupName)
			expectedQuery := "fields @timestamp, @message | filter level = 'error' | sort @timestamp desc | limit 100"
			assert.Equal(t, expectedQuery, *params.QueryString)
			// Validate that a duration Last overrides default (we didn't set Last here, so default ~1h window)
			assert.Greater(t, *params.EndTime, *params.StartTime)
			return &cloudwatchlogs.StartQueryOutput{
				QueryId: aws.String("test-query-id"),
			}, nil
		},
	}

	logClient := &CloudWatchLogClient{client: mockClient}

	search := &client.LogSearch{
		Fields: ty.MS{"level": "error"},
		Size:   ty.Opt[int]{Set: true, Value: 100},
		Options: ty.MI{
			"logGroupName": "test-group",
		},
	}

	result, err := logClient.Get(context.Background(), search)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	cwResult, ok := result.(*CloudWatchLogSearchResult)
	assert.True(t, ok)
	assert.Equal(t, "test-query-id", cwResult.queryId)
}

func TestCloudWatchLogSearchResult_GetEntries(t *testing.T) {
	mockClient := &mockCWClient{
		GetQueryResultsFunc: func(ctx context.Context, params *cloudwatchlogs.GetQueryResultsInput, optFns ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.GetQueryResultsOutput, error) {
			return &cloudwatchlogs.GetQueryResultsOutput{
				Status: types.QueryStatusComplete,
				Results: [][]types.ResultField{
					{
						{Field: aws.String("@timestamp"), Value: aws.String("2025-08-23 21:30:00.123")},
						{Field: aws.String("@message"), Value: aws.String("test message 1")},
						{Field: aws.String("level"), Value: aws.String("INFO")},
					},
					{
						{Field: aws.String("@timestamp"), Value: aws.String("2025-08-23 21:30:05.000")},
						{Field: aws.String("@message"), Value: aws.String("test message 2")},
						{Field: aws.String("level"), Value: aws.String("DEBUG")},
					},
				},
			}, nil
		},
	}

	searchResult := &CloudWatchLogSearchResult{
		client:  mockClient,
		queryId: "test-query-id",
		search:  &client.LogSearch{},
	}

	entries, _, err := searchResult.GetEntries(context.Background())
	assert.NoError(t, err)
	assert.Len(t, entries, 2)

	assert.Equal(t, "test message 1", entries[0].Message)
	assert.False(t, entries[0].Timestamp.IsZero())
	assert.Equal(t, 123000000, entries[0].Timestamp.Nanosecond())
	assert.Equal(t, "INFO", entries[0].Fields["level"])

	assert.Equal(t, "test message 2", entries[1].Message)
	assert.Equal(t, "DEBUG", entries[1].Fields["level"])
}

func TestCloudWatch_TimeRange_Last(t *testing.T) {
	mockClient := &mockCWClient{
		StartQueryFunc: func(ctx context.Context, params *cloudwatchlogs.StartQueryInput, optFns ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.StartQueryOutput, error) {
			// Expect approximately a 10m window
			windowMs := *params.EndTime - *params.StartTime
			assert.InDelta(t, 10*60*1000, windowMs, 5*1000) // allow 5s jitter
			return &cloudwatchlogs.StartQueryOutput{QueryId: aws.String("qid-last")}, nil
		},
	}
	c := &CloudWatchLogClient{client: mockClient}
	s := &client.LogSearch{Options: ty.MI{"logGroupName": "lg"}}
	s.Range.Last.S("10m")
	_, err := c.Get(context.Background(), s)
	assert.NoError(t, err)
}

func TestCloudWatch_TimeRange_GteLte(t *testing.T) {
	mockClient := &mockCWClient{
		StartQueryFunc: func(ctx context.Context, params *cloudwatchlogs.StartQueryInput, optFns ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.StartQueryOutput, error) {
			// We set explicit times; ensure they are respected
			expectedStart := time.Date(2025, 8, 23, 12, 0, 0, 0, time.UTC).UnixMilli()
			expectedEnd := time.Date(2025, 8, 23, 13, 0, 0, 0, time.UTC).UnixMilli()
			assert.Equal(t, expectedStart, *params.StartTime)
			assert.Equal(t, expectedEnd, *params.EndTime)
			return &cloudwatchlogs.StartQueryOutput{QueryId: aws.String("qid-abs")}, nil
		},
	}
	c := &CloudWatchLogClient{client: mockClient}
	s := &client.LogSearch{Options: ty.MI{"logGroupName": "lg"}}
	s.Range.Gte.S("2025-08-23T12:00:00Z")
	s.Range.Lte.S("2025-08-23T13:00:00Z")
	_, err := c.Get(context.Background(), s)
	assert.NoError(t, err)
}

func TestCloudWatch_GetFields(t *testing.T) {
	mockClient := &mockCWClient{
		GetQueryResultsFunc: func(ctx context.Context, params *cloudwatchlogs.GetQueryResultsInput, optFns ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.GetQueryResultsOutput, error) {
			return &cloudwatchlogs.GetQueryResultsOutput{
				Status: types.QueryStatusComplete,
				Results: [][]types.ResultField{{
					{Field: aws.String("@timestamp"), Value: aws.String("2025-08-23 21:30:00.123")},
					{Field: aws.String("@message"), Value: aws.String("log one")},
					{Field: aws.String("level"), Value: aws.String("INFO")},
					{Field: aws.String("service"), Value: aws.String("auth")},
				}, {
					{Field: aws.String("@timestamp"), Value: aws.String("2025-08-23 21:30:01.000")},
					{Field: aws.String("@message"), Value: aws.String("log two")},
					{Field: aws.String("level"), Value: aws.String("DEBUG")},
					{Field: aws.String("service"), Value: aws.String("auth")},
				}},
			}, nil
		},
	}
	sr := &CloudWatchLogSearchResult{client: mockClient, queryId: "qid-fields", search: &client.LogSearch{}}
	// Ensure entries loaded
	_, _, err := sr.GetEntries(context.Background())
	assert.NoError(t, err)
	fields, _, err := sr.GetFields(context.Background())
	assert.NoError(t, err)
	// Expect level INFO, DEBUG and service auth
	assert.Contains(t, fields["level"], "INFO")
	assert.Contains(t, fields["level"], "DEBUG")
	assert.Contains(t, fields["service"], "auth")
}
