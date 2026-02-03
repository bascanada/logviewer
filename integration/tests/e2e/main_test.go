//go:build integration

package e2e

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	_, filename, _, _ := runtime.Caller(0)
	currentDir := filepath.Dir(filename)
	rootDir := filepath.Join(currentDir, "../../..")

	binaryPath := filepath.Join(rootDir, "build", "logviewer")
	configPath := filepath.Join(rootDir, "integration", "infra", "config.yaml") + ":" +
		filepath.Join(rootDir, "integration", "infra", "config.hl.yaml") + ":" +
		filepath.Join(rootDir, "integration", "infra", "config.extra.yaml")

	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		fmt.Println("Building logviewer binary...")
		cmd := exec.Command("make", "build")
		cmd.Dir = rootDir
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			fmt.Printf("ERROR: Failed to build binary: %v\n", err)
			os.Exit(1)
		}
	}

	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		fmt.Printf("ERROR: Logviewer binary not found at %s\n", binaryPath)
		os.Exit(1)
	}

	fmt.Println("Verifying Docker services are running...")
	fmt.Println("(If tests fail, ensure you've run: make integration/start)")
	fmt.Println()

	// Check infrastructure health BEFORE attempting to seed
	fmt.Println("Checking infrastructure health...")
	healthCtx, healthCancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer healthCancel()

	if err := CheckAllServices(healthCtx); err != nil {
		fmt.Printf("ERROR: Infrastructure not ready: %v\n", err)
		fmt.Println("Ensure all services are running: make integration/start")
		os.Exit(1)
	}

	// Generate unique RunID for this test session
	runID := fmt.Sprintf("test-run-%d", time.Now().Unix())

	// Check if we should disable run_id filtering (for backward compatibility with old log-generator)
	disableRunIDFilter := os.Getenv("DISABLE_RUN_ID_FILTER") == "true"
	if disableRunIDFilter {
		fmt.Println("⚠️  RunID filtering disabled (backward compatibility mode)")
		runID = "" // Don't pass runID to seed endpoint
	} else {
		fmt.Printf("Test Run ID: %s\n", runID)
	}
	fmt.Println()

	tCtx = TestContext{
		BinaryPath:         binaryPath,
		ConfigPath:         configPath,
		RunID:              runID,
		DisableRunIDFilter: disableRunIDFilter,
	}

	// Set environment variables for non-test functions
	os.Setenv("LOGVIEWER_BINARY", binaryPath)
	os.Setenv("LOGVIEWER_CONFIG", configPath)

	fmt.Printf("Using binary: %s\n", binaryPath)
	fmt.Printf("Using config: %s\n", configPath)
	fmt.Println()

	// Seed test data with intelligent polling validation
	fmt.Println("Seeding test data via log-generator...")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Determine which fixtures to seed based on TEST_FIXTURES env var
	// If not set, seed all fixtures for backward compatibility
	// If set to empty string "", skip seeding entirely
	fixturesStr, fixturesEnvSet := os.LookupEnv("TEST_FIXTURES")
	var fixturesToSeed []string
	var skipSeeding bool

	if !fixturesEnvSet {
		// Env var not set - seed all by default
		fixturesToSeed = []string{} // Empty means all
		fmt.Println("Seeding all fixtures (set TEST_FIXTURES to customize)")
	} else if fixturesStr == "" {
		// Explicitly set to empty - skip seeding
		skipSeeding = true
		fmt.Println("Skipping fixture seeding (TEST_FIXTURES=\"\")")
	} else {
		// Specific fixtures requested
		fixturesToSeed = strings.Split(fixturesStr, ",")
		// Trim whitespace
		for i := range fixturesToSeed {
			fixturesToSeed[i] = strings.TrimSpace(fixturesToSeed[i])
		}
		fmt.Printf("Seeding selected fixtures: %v\n", fixturesToSeed)
	}

	var resp *SeedResponse
	var err error
	if !skipSeeding {
		resp, err = SeedAndWait(ctx, runID, fixturesToSeed...)
		if err != nil {
			fmt.Printf("ERROR: Failed to seed test data: %v\n", err)

			// If RunID was used, suggest disabling it for old log-generator
			if runID != "" && !disableRunIDFilter {
				fmt.Println("\n💡 Tip: If your log-generator doesn't support run_id yet, try:")
				fmt.Println("   DISABLE_RUN_ID_FILTER=true make integration/test")
				fmt.Println("\n   Or rebuild log-generator with: ./scripts/redeploy-log-generator.sh")
			}

			fmt.Println("\nTests cannot run without seeded data.")
			fmt.Println("Ensure log-generator is running: docker-compose up -d log-generator")
			os.Exit(1) // Fail fast - don't run tests without data
		}

		fmt.Printf("✓ Successfully seeded and validated test data: %v\n", resp.Seeded)
	}
	fmt.Println("Test environment ready!")
	fmt.Println()

	// Change to root directory so relative paths in config work
	if err := os.Chdir(rootDir); err != nil {
		fmt.Printf("ERROR: Failed to change to root dir: %v\n", err)
		os.Exit(1)
	}

	exitCode := m.Run()

	// Optional: Cleanup test data after tests complete
	// (Uncomment if you want to clean up test indexes after tests)
	// fmt.Println("Cleaning up test data...")
	// CleanupTestData(context.Background())

	os.Exit(exitCode)
}
