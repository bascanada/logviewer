package cloudwatch

import (
	"context"
	"testing"

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
}

func (m *mockCWClient) StartQuery(ctx context.Context, params *cloudwatchlogs.StartQueryInput, optFns ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.StartQueryOutput, error) {
	return m.StartQueryFunc(ctx, params, optFns...)
}

func (m *mockCWClient) GetQueryResults(ctx context.Context, params *cloudwatchlogs.GetQueryResultsInput, optFns ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.GetQueryResultsOutput, error) {
	return m.GetQueryResultsFunc(ctx, params, optFns...)
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

	result, err := logClient.Get(search)
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
						{Field: aws.String("@timestamp"), Value: aws.String("2025-08-23 21:30:00.000")},
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
	assert.Equal(t, "INFO", entries[0].Fields["level"])

	assert.Equal(t, "test message 2", entries[1].Message)
	assert.Equal(t, "DEBUG", entries[1].Fields["level"])
}
