//go:build integration

package e2e

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
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
	Fixtures []string `json:"fixtures"`         // Names of fixtures to seed
	RunID    string   `json:"run_id,omitempty"` // Unique ID to tag logs with for isolation
}

// SeedResponse is the response from /seed endpoint
type SeedResponse struct {
	Success bool           `json:"success"`
	Message string         `json:"message"`
	Seeded  map[string]int `json:"seeded"` // fixture name -> count
	Errors  []string       `json:"errors,omitempty"`
	RunID   string         `json:"run_id,omitempty"` // Echo back the run_id used
}

const (
	logGeneratorURL  = "http://localhost:8081" // Adjust if running in different environment
	indexingWaitTime = 7 * time.Second         // DEPRECATED: Use polling instead
)

// SeedTestData seeds the specified fixtures by calling the log-generator /seed endpoint
func SeedTestData(ctx context.Context, runID string, fixtures ...string) (*SeedResponse, error) {
	// If no fixtures specified, seed all
	if len(fixtures) == 0 {
		fixtures = []string{"error-logs", "payment-logs", "order-logs", "mixed-levels", "trace-logs", "slow-requests"}
	}

	req := SeedRequest{
		Fixtures: fixtures,
		RunID:    runID,
	}
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

// SeedAndWait seeds test data and intelligently polls until data is queryable
// This replaces the old static sleep approach with smart polling
func SeedAndWait(ctx context.Context, runID string, fixtures ...string) (*SeedResponse, error) {
	resp, err := SeedTestData(ctx, runID, fixtures...)
	if err != nil {
		return resp, err
	}

	// Verify each fixture is queryable - poll until data appears
	fmt.Printf("Validating seeded data is indexed and queryable...\n")
	if err := ValidateSeedData(ctx, runID, fixtures); err != nil {
		return resp, fmt.Errorf("seed validation failed: %w", err)
	}

	fmt.Printf("✓ All fixtures validated and ready for testing\n")
	return resp, nil
}

// ValidateSeedData polls until all seeded fixtures are queryable via CLI
// This ensures data is actually indexed before tests run
func ValidateSeedData(ctx context.Context, runID string, fixtures []string) error {
	if len(fixtures) == 0 {
		fixtures = []string{"error-logs", "payment-logs", "order-logs", "mixed-levels", "trace-logs", "slow-requests"}
	}

	timeout := 90 * time.Second      // Increased timeout for slow indexing
	checkInterval := 2 * time.Second // Reduced frequency to avoid overwhelming CLI

	for _, fixtureName := range fixtures {
		fixture, exists := AvailableFixtures[fixtureName]
		if !exists {
			return fmt.Errorf("unknown fixture: %s", fixtureName)
		}

		fixtureStart := time.Now()
		fmt.Printf("  Checking %s (%d logs expected)...\n", fixtureName, fixture.Count)

		// Poll until expected count is found
		found := false
		deadline := time.Now().Add(timeout)
		var lastCount int
		attempt := 0

		for time.Now().Before(deadline) {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			attempt++

			// Query using CLI to count logs
			count := countLogsInFixture(ctx, fixture, runID)
			lastCount = count

			// Show progress every attempt (every 2 seconds)
			elapsed := time.Since(fixtureStart).Seconds()
			if count > 0 {
				fmt.Printf("    [%ds] Found %d/%d logs (attempt #%d)\n", int(elapsed), count, fixture.Count, attempt)
			} else {
				fmt.Printf("    [%ds] No logs yet (attempt #%d)\n", int(elapsed), attempt)
			}

			if count >= fixture.Count {
				fmt.Printf("    ✓ %s ready with %d logs (took %.1fs)\n\n", fixtureName, count, elapsed)
				found = true
				break
			}

			// Keep polling
			time.Sleep(checkInterval)
		}

		if !found {
			fmt.Printf("    ✗ Timeout waiting for %s\n", fixtureName)
			return fmt.Errorf("timeout waiting for %s to be indexed (expected %d logs, found %d after %.1fs, context: %s)",
				fixtureName, fixture.Count, lastCount, time.Since(fixtureStart).Seconds(), getContextForIndex(fixture.Index))
		}
	}

	return nil
}

// countLogsInFixture queries the fixture index via CLI and returns log count
func countLogsInFixture(ctx context.Context, fixture TestFixtureInfo, runID string) int {
	// Use a simple query to count logs in this fixture's index
	contextName := getContextForIndex(fixture.Index)
	args := []string{
		"query", "log",
		"-i", contextName,
		"--last", "2h", // Seeded data is 1 hour old, use generous window
		"--size", "1000", // Get enough to count
		"--json",
	}

	// Try WITH run_id filter first (for new log-generator)
	if runID != "" {
		argsWithRunID := append(args, "-f", "run_id="+runID)
		output, err := runCommand(argsWithRunID...)
		if err != nil {
			// Only show error on first few attempts
			if os.Getenv("DEBUG_E2E") != "" {
				fmt.Printf("      [DEBUG] Query with run_id failed: %v\n", err)
			}
		} else {
			logs := parseJSONOutputSimple(output)
			if len(logs) > 0 {
				return len(logs)
			}
		}
	}

	// Fallback: try WITHOUT run_id filter (for backward compatibility)
	output, err := runCommand(args...)
	if err != nil {
		if os.Getenv("DEBUG_E2E") != "" {
			fmt.Printf("      [DEBUG] Query without run_id failed: %v\n", err)
		}
		return 0
	}

	logs := parseJSONOutputSimple(output)
	return len(logs)
}

// runCommand executes the logviewer CLI command
func runCommand(args ...string) (string, error) {
	// Get paths from environment (set by TestMain)
	binaryPath := os.Getenv("LOGVIEWER_BINARY")
	configPath := os.Getenv("LOGVIEWER_CONFIG")

	if binaryPath == "" || configPath == "" {
		return "", fmt.Errorf("LOGVIEWER_BINARY or LOGVIEWER_CONFIG not set")
	}

	cmd := exec.Command(binaryPath, args...)
	cmd.Env = append(os.Environ(), "LOGVIEWER_CONFIG="+configPath)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

// parseJSONOutputSimple parses NDJSON output without test assertions
// Used during validation where we don't want to fail tests on parse errors
func parseJSONOutputSimple(output string) []map[string]interface{} {
	output = strings.TrimSpace(output)
	if output == "" {
		return []map[string]interface{}{}
	}

	// Try array format first
	var logsArray []map[string]interface{}
	if err := json.Unmarshal([]byte(output), &logsArray); err == nil {
		return logsArray
	}

	// Try NDJSON format
	var logs []map[string]interface{}
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var log map[string]interface{}
		if err := json.Unmarshal([]byte(line), &log); err == nil {
			logs = append(logs, log)
		}
	}
	return logs
}

// getContextForIndex maps a fixture index to the appropriate logviewer context
func getContextForIndex(index string) string {
	// Map test indexes to their configured contexts
	contextMap := map[string]string{
		"test-e2e-errors":   "test-splunk-errors",
		"test-e2e-payments": "test-splunk-payments",
		"test-e2e-orders":   "test-opensearch-orders",
		"test-e2e-mixed":    "test-opensearch-mixed",
		"test-e2e-traces":   "test-splunk-traces",
		"test-e2e-slow":     "test-opensearch-slow",
	}

	if ctx, ok := contextMap[index]; ok {
		return ctx
	}

	// Fallback - try to infer from index name
	if contains(index, "splunk") || contains(index, "error") || contains(index, "payment") || contains(index, "trace") {
		return "splunk-all"
	}
	return "opensearch-all"
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || indexOf(s, substr) >= 0))
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
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
