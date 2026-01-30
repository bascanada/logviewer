//go:build integration

package e2e

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
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

	tCtx = TestContext{
		BinaryPath: binaryPath,
		ConfigPath: configPath,
	}

	fmt.Printf("Using binary: %s\n", binaryPath)
	fmt.Printf("Using config: %s\n", configPath)
	fmt.Println()

	// Seed test data before running tests
	fmt.Println("Seeding test data via log-generator...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := SeedAndWait(ctx) // Seeds all fixtures by default
	if err != nil {
		fmt.Printf("WARNING: Failed to seed test data: %v\n", err)
		fmt.Println("Tests may fail if test data is not available.")
		fmt.Println("Ensure log-generator is running: docker-compose up -d log-generator")
	} else {
		fmt.Printf("Successfully seeded test data: %v\n", resp.Seeded)
		fmt.Println("Test data ready for E2E tests!")
	}
	fmt.Println()

	exitCode := m.Run()

	// Optional: Cleanup test data after tests complete
	// (Uncomment if you want to clean up test indexes after tests)
	// fmt.Println("Cleaning up test data...")
	// CleanupTestData(context.Background())

	os.Exit(exitCode)
}
