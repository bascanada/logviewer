//go:build integration

package e2e

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"
)

// ServiceHealthCheck represents a service health verification
type ServiceHealthCheck struct {
	Name    string
	URL     string
	Timeout time.Duration
}

// CheckServiceHealth polls a service health endpoint until it's ready or timeout
func CheckServiceHealth(ctx context.Context, check ServiceHealthCheck) error {
	fmt.Printf("  Checking %s at %s...\n", check.Name, check.URL)

	deadline := time.Now().Add(check.Timeout)

	// Build TLS config
	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}

	// Allow insecure for integration testing
	if os.Getenv("LOGVIEWER_TLS_INSECURE") == "true" {
		tlsConfig.InsecureSkipVerify = true
	}

	client := &http.Client{
		Timeout: 5 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}

	var lastErr error
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		req, err := http.NewRequestWithContext(ctx, "GET", check.URL, nil)
		if err != nil {
			lastErr = err
			time.Sleep(1 * time.Second)
			continue
		}

		resp, err := client.Do(req)
		if err == nil {
			resp.Body.Close()
			// Service is accessible - even 401/403 means it's running
			if resp.StatusCode < 500 {
				fmt.Printf("  ✓ %s is ready (status: %d)\n", check.Name, resp.StatusCode)
				return nil
			}
			lastErr = fmt.Errorf("status %d", resp.StatusCode)
		} else {
			lastErr = err
		}

		time.Sleep(1 * time.Second)
	}

	return fmt.Errorf("%s did not become ready in %v: %v", check.Name, check.Timeout, lastErr)
}

// CheckAllServices verifies all infrastructure services are ready
func CheckAllServices(ctx context.Context) error {
	fmt.Println("Checking infrastructure health...")

	checks := []ServiceHealthCheck{
		{
			Name:    "OpenSearch",
			URL:     "http://localhost:9200/_cluster/health",
			Timeout: 60 * time.Second,
		},
		{
			Name:    "Splunk API",
			URL:     "https://localhost:8089/services/server/health",
			Timeout: 120 * time.Second, // Splunk can be slow to start
		},
		{
			Name:    "Splunk HEC",
			URL:     "https://localhost:8088/services/collector/health",
			Timeout: 120 * time.Second,
		},
		{
			Name:    "Log Generator",
			URL:     "http://localhost:8081/health",
			Timeout: 30 * time.Second,
		},
	}

	// Run all health checks in parallel for speed
	var wg sync.WaitGroup
	errChan := make(chan error, len(checks))

	for _, check := range checks {
		wg.Add(1)
		go func(c ServiceHealthCheck) {
			defer wg.Done()
			if err := CheckServiceHealth(ctx, c); err != nil {
				errChan <- err
			}
		}(check)
	}

	// Wait for all checks to complete
	wg.Wait()
	close(errChan)

	// Collect any errors
	var errors []error
	for err := range errChan {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		fmt.Println("\n❌ Infrastructure health check failed:")
		for _, err := range errors {
			fmt.Printf("  - %v\n", err)
		}
		return fmt.Errorf("%d service(s) are not ready", len(errors))
	}

	fmt.Println("✓ All infrastructure services are healthy")
	return nil
}
