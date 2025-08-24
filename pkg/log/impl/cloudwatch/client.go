package cloudwatch

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"

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

// sanitizeQueryValue escapes single quotes in user provided values to safely embed them
// in CloudWatch Logs Insights query strings using single-quoted literals.
func sanitizeQueryValue(v string) string {
	// CloudWatch Logs Insights uses backslash as escape inside single quotes.
	// Replace single quote ' with \'
	return strings.ReplaceAll(v, "'", "\\'")
}

// isSafeFieldName ensures the field name only contains allowed runes (letters, digits, underscore, at-sign, and dot)
// to mitigate injection via crafted field names.
func isSafeFieldName(name string) bool {
	if name == "" {
		return false
	}
	for _, r := range name {
		if r == '@' || r == '_' || r == '.' || unicode.IsLetter(r) || unicode.IsDigit(r) {
			continue
		}
		return false
	}
	return true
}

// Get will be implemented in Phase 2.
func (c *CloudWatchLogClient) Get(ctx context.Context, search *client.LogSearch) (client.LogSearchResult, error) {
	logGroupName, ok := search.Options.GetStringOk("logGroupName")
	if !ok {
		return nil, errors.New("logGroupName is required in options for CloudWatch Logs")
	}

	// 1. Build the query string
	var queryParts []string
	// Always fetch the raw message and timestamp
	queryParts = append(queryParts, "fields @timestamp, @message")

	// Add filters from LogSearch.Fields with sanitization to avoid query injection.
	for key, value := range search.Fields {
		if !isSafeFieldName(key) {
			// Skip unsafe field names to avoid injection via the key itself.
			continue
		}
		sanitizedValue := sanitizeQueryValue(value)
		queryParts = append(queryParts, fmt.Sprintf(" | filter %s = '%s'", key, sanitizedValue))
	}

	// Add sorting and limits
	queryParts = append(queryParts, " | sort @timestamp desc")
	if search.Size.Set {
		queryParts = append(queryParts, " | limit "+fmt.Sprintf("%d", search.Size.Value))
	}

	queryString := strings.Join(queryParts, "")

	// 2. Determine time range using search.Range (Last takes precedence over Gte/Lte)
	endTime := time.Now()
	startTime := endTime.Add(-1 * time.Hour) // default fallback

	// Helper to parse absolute timestamps (RFC3339 or Insights-like layout)
	parseAbs := func(v string) (time.Time, error) {
		if v == "" {
			return time.Time{}, errors.New("empty time string")
		}
		layouts := []string{time.RFC3339Nano, time.RFC3339, "2006-01-02 15:04:05.000"}
		var lastErr error
		for _, l := range layouts {
			if ts, err := time.Parse(l, v); err == nil {
				return ts, nil
			} else {
				lastErr = err
			}
		}
		return time.Time{}, lastErr
	}

	if search.Range.Last.Set && search.Range.Last.Value != "" {
		if d, err := time.ParseDuration(search.Range.Last.Value); err == nil {
			startTime = endTime.Add(-d)
		}
	}
	// Absolute range overrides default when provided
	if search.Range.Gte.Set && search.Range.Gte.Value != "" {
		if gte, err := parseAbs(search.Range.Gte.Value); err == nil {
			startTime = gte
		}
	}
	if search.Range.Lte.Set && search.Range.Lte.Value != "" {
		if lte, err := parseAbs(search.Range.Lte.Value); err == nil {
			endTime = lte
		}
	}
	// Ensure start <= end; if not, swap
	if startTime.After(endTime) {
		startTime, endTime = endTime.Add(-1*time.Hour), endTime
	}

	// 3. Start the query
	startQueryOutput, err := c.client.StartQuery(ctx, &cloudwatchlogs.StartQueryInput{
		LogGroupName:  aws.String(logGroupName),
		QueryString:   aws.String(queryString),
		StartTime:     aws.Int64(startTime.UnixMilli()),
		EndTime:       aws.Int64(endTime.UnixMilli()),
	})
	if err != nil {
		return nil, err
	}

	if startQueryOutput.QueryId == nil {
		return nil, errors.New("StartQuery did not return a QueryId")
	}

	// 4. Return the result handler
	return &CloudWatchLogSearchResult{client: c.client, queryId: *startQueryOutput.QueryId, search: search}, nil
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
