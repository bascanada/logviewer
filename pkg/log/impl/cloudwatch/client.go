package cloudwatch

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/berlingoqc/logviewer/pkg/log/client"
	"github.com/berlingoqc/logviewer/pkg/ty"
)

// CWClient defines the interface for the AWS CloudWatch Logs client.
// This is used to allow for mocking in tests.
type CWClient interface {
	StartQuery(ctx context.Context, params *cloudwatchlogs.StartQueryInput, optFns ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.StartQueryOutput, error)
	GetQueryResults(ctx context.Context, params *cloudwatchlogs.GetQueryResultsInput, optFns ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.GetQueryResultsOutput, error)
}

// CloudWatchLogClient implements the client.LogClient interface for AWS CloudWatch.
type CloudWatchLogClient struct {
	client CWClient
}

// Get will be implemented in Phase 2.
func (c *CloudWatchLogClient) Get(search *client.LogSearch) (client.LogSearchResult, error) {
	logGroupName, ok := search.Options.GetStringOk("logGroupName")
	if !ok {
		return nil, errors.New("logGroupName is required in options for CloudWatch Logs")
	}

	// 1. Build the query string
	var queryParts []string
	// Always fetch the raw message and timestamp
	queryParts = append(queryParts, "fields @timestamp, @message")

	// Add filters from LogSearch.Fields
	for key, value := range search.Fields {
		// Basic string equality filter. Can be expanded for other operators.
		queryParts = append(queryParts, fmt.Sprintf(" | filter %s = '%s'", key, value))
	}

	// Add sorting and limits
	queryParts = append(queryParts, " | sort @timestamp desc")
	if search.Size.Set {
		queryParts = append(queryParts, " | limit "+fmt.Sprintf("%d", search.Size.Value))
	}

	queryString := strings.Join(queryParts, "")

	// 2. Determine time range (simplified for brevity)
	// A full implementation should handle Gte, Lte, and Last.
	endTime := time.Now()
	startTime := endTime.Add(-1 * time.Hour) // Default to last hour

	// 3. Start the query
	startQueryOutput, err := c.client.StartQuery(context.TODO(), &cloudwatchlogs.StartQueryInput{
		LogGroupName:  aws.String(logGroupName),
		QueryString:   aws.String(queryString),
		StartTime:     aws.Int64(startTime.UnixMilli()),
		EndTime:       aws.Int64(endTime.UnixMilli()),
	})
	if err != nil {
		return nil, err
	}

	// 4. Return the result handler
	return &CloudWatchLogSearchResult{
		client:  c.client,
		queryId: *startQueryOutput.QueryId,
		search:  search,
	}, nil
}

// GetLogClient creates a new CloudWatch Logs client.
// It uses the 'region' and 'profile' from the options if provided.
func GetLogClient(options ty.MI) (client.LogClient, error) {
	var cfgOptions []func(*config.LoadOptions) error

	// If a region is specified in the config, add it to the SDK options.
	if region, ok := options.GetStringOk("region"); ok {
		cfgOptions = append(cfgOptions, config.WithRegion(region))
	}

	// If a profile is specified, add it to the SDK options.
	if profile, ok := options.GetStringOk("profile"); ok {
		cfgOptions = append(cfgOptions, config.WithSharedConfigProfile(profile))
	}

	// Load the default AWS configuration, applying our custom options.
	cfg, err := config.LoadDefaultConfig(context.TODO(), cfgOptions...)
	if err != nil {
		return nil, err
	}

	return &CloudWatchLogClient{
		client: cloudwatchlogs.NewFromConfig(cfg),
	}, nil
}
