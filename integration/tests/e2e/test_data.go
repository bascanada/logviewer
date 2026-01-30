//go:build integration

package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// TestFixtureInfo describes a test fixture
type TestFixtureInfo struct {
	Name        string // e.g., "error-logs"
	Description string
	Index       string // Splunk/OpenSearch index
	Count       int    // Expected number of logs
}

// AvailableFixtures lists all available test fixtures
var AvailableFixtures = map[string]TestFixtureInfo{
	"error-logs": {
		Name:        "error-logs",
		Description: "50 ERROR level logs from payment-service",
		Index:       "test-e2e-errors",
		Count:       50,
	},
	"payment-logs": {
		Name:        "payment-logs",
		Description: "100 payment-service logs (80 INFO, 20 ERROR)",
		Index:       "test-e2e-payments",
		Count:       100,
	},
	"order-logs": {
		Name:        "order-logs",
		Description: "75 order-service logs (mixed levels)",
		Index:       "test-e2e-orders",
		Count:       75,
	},
	"mixed-levels": {
		Name:        "mixed-levels",
		Description: "100 logs with mixed levels (50 INFO, 30 WARN, 20 ERROR)",
		Index:       "test-e2e-mixed",
		Count:       100,
	},
	"trace-logs": {
		Name:        "trace-logs",
		Description: "30 logs with trace_ids for distributed tracing",
		Index:       "test-e2e-traces",
		Count:       30,
	},
	"slow-requests": {
		Name:        "slow-requests",
		Description: "25 logs with latency > 1000ms",
		Index:       "test-e2e-slow",
		Count:       25,
	},
}

// SeedRequest is the request payload for /seed endpoint
type SeedRequest struct {
	Fixtures []string `json:"fixtures"` // Names of fixtures to seed
}

// SeedResponse is the response from /seed endpoint
type SeedResponse struct {
	Success bool           `json:"success"`
	Message string         `json:"message"`
	Seeded  map[string]int `json:"seeded"`   // fixture name -> count
	Errors  []string       `json:"errors,omitempty"`
}

const (
	logGeneratorURL = "http://localhost:8081" // Adjust if running in different environment
	indexingWaitTime = 7 * time.Second        // Time to wait for logs to be indexed
)

// SeedTestData seeds the specified fixtures by calling the log-generator /seed endpoint
func SeedTestData(ctx context.Context, fixtures ...string) (*SeedResponse, error) {
	// If no fixtures specified, seed all
	if len(fixtures) == 0 {
		fixtures = []string{"error-logs", "payment-logs", "order-logs", "mixed-levels", "trace-logs", "slow-requests"}
	}

	req := SeedRequest{Fixtures: fixtures}
	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal seed request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", logGeneratorURL+"/seed", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to call /seed endpoint: %w (is log-generator running at %s?)", err, logGeneratorURL)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("seed endpoint returned status %d: %s", resp.StatusCode, string(body))
	}

	var seedResp SeedResponse
	if err := json.NewDecoder(resp.Body).Decode(&seedResp); err != nil {
		return nil, fmt.Errorf("failed to decode seed response: %w", err)
	}

	if !seedResp.Success {
		return &seedResp, fmt.Errorf("seed operation had errors: %v", seedResp.Errors)
	}

	return &seedResp, nil
}

// SeedAndWait seeds test data and waits for it to be indexed
func SeedAndWait(ctx context.Context, fixtures ...string) (*SeedResponse, error) {
	resp, err := SeedTestData(ctx, fixtures...)
	if err != nil {
		return resp, err
	}

	// Wait for indexing to complete
	select {
	case <-time.After(indexingWaitTime):
		return resp, nil
	case <-ctx.Done():
		return resp, ctx.Err()
	}
}

// GetFixtureInfo returns information about a specific fixture
func GetFixtureInfo(fixtureName string) (TestFixtureInfo, error) {
	info, exists := AvailableFixtures[fixtureName]
	if !exists {
		return TestFixtureInfo{}, fmt.Errorf("fixture '%s' not found", fixtureName)
	}
	return info, nil
}

// WaitForIndexing waits for logs to be indexed in the backend
// This is useful if you seed data and need to ensure it's searchable before running tests
func WaitForIndexing(ctx context.Context, duration time.Duration) error {
	select {
	case <-time.After(duration):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
