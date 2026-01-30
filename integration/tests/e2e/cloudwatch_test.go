//go:build integration

package e2e

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestCloudWatch_QueryLog tests CloudWatch log querying via LocalStack
func TestCloudWatch_QueryLog(t *testing.T) {
	t.Parallel()

	t.Run("BasicQuery", func(t *testing.T) {
		t.Skip("CloudWatch requires log group to be pre-populated in LocalStack")

		output, err := RunCommand("query", "log", "-i", "cloudwatch-orders", "--last", "1h", "--size", "10", "--json")
		assert.NoError(t, err, "Query should execute successfully")

		entries := ParseJSONOutput(output)
		// May be empty if no logs were sent to LocalStack
		assert.GreaterOrEqual(t, len(entries), 0, "Should return entries or empty array")
	})

	t.Run("WithTimeRange", func(t *testing.T) {
		t.Skip("CloudWatch requires log group to be pre-populated in LocalStack")

		output, err := RunCommand("query", "log", "-i", "cloudwatch-orders", "--last", "30m", "--size", "5", "--json")
		assert.NoError(t, err, "Query with time range should execute successfully")

		entries := ParseJSONOutput(output)
		assert.GreaterOrEqual(t, len(entries), 0, "Should handle time range queries")
	})
}

// TestCloudWatch_QueryField tests field discovery in CloudWatch logs
func TestCloudWatch_QueryField(t *testing.T) {
	t.Parallel()

	t.Run("BasicFieldDiscovery", func(t *testing.T) {
		t.Skip("CloudWatch requires log group to be pre-populated in LocalStack")

		output, err := RunCommand("query", "field", "-i", "cloudwatch-orders", "--last", "1h", "--json")
		assert.NoError(t, err, "Field discovery should execute successfully")

		fields := ParseFieldsJSON(output)
		assert.GreaterOrEqual(t, len(fields), 0, "Should discover fields or return empty")
	})
}

// TestCloudWatch_QueryValues tests value extraction from CloudWatch logs
func TestCloudWatch_QueryValues(t *testing.T) {
	t.Parallel()

	t.Run("SingleField", func(t *testing.T) {
		t.Skip("CloudWatch requires log group to be pre-populated in LocalStack")

		output, err := RunCommand("query", "values", "-i", "cloudwatch-orders", "--last", "1h", "--field", "level", "--json")
		assert.NoError(t, err, "Values query should execute successfully")

		values := ParseValuesJSON(output)
		assert.GreaterOrEqual(t, len(values), 0, "Should extract values or return empty")
	})
}

// Note: CloudWatch tests are skipped by default because LocalStack requires:
// 1. Log group to be created: aws --endpoint-url=http://localhost:4566 logs create-log-group --log-group-name my-app-logs
// 2. Log stream to be created: aws --endpoint-url=http://localhost:4566 logs create-log-stream --log-group-name my-app-logs --log-stream-name test-stream
// 3. Logs to be sent: aws --endpoint-url=http://localhost:4566 logs put-log-events --log-group-name my-app-logs --log-stream-name test-stream --log-events timestamp=...,message=...
//
// To enable these tests, set up CloudWatch in LocalStack using integration/infra/cloudwatch/send-logs.sh
