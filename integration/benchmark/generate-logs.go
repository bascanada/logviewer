// Log generator for performance benchmarking
// Generates large JSON log files quickly for testing hl vs native performance
//
// Usage: go run generate-logs.go [options]
//   -size      Number of log entries (default: 1000000)
//   -output    Output file path (default: /tmp/benchmark-logs.json)
//   -error     Error log percentage (default: 5)
//   -warn      Warning log percentage (default: 10)

package main

import (
	"bufio"
	"compress/gzip"
	"encoding/json"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type LogEntry struct {
	Timestamp  string `json:"@timestamp"`
	Level      string `json:"level"`
	Message    string `json:"message"`
	App        string `json:"app"`
	LatencyMs  int    `json:"latency_ms"`
	TraceID    string `json:"trace_id,omitempty"`
	UserID     string `json:"user_id,omitempty"`
	RequestID  string `json:"request_id,omitempty"`
	StatusCode int    `json:"status_code,omitempty"`
	Method     string `json:"method,omitempty"`
	Path       string `json:"path,omitempty"`
}

var (
	services = []string{
		"api-gateway", "user-service", "payment-service",
		"order-service", "inventory-service", "notification-service",
		"auth-service", "search-service", "analytics-service",
	}

	infoMessages = []string{
		"Request processed successfully",
		"User authenticated",
		"Database query executed in %dms",
		"Cache hit for key user:%d",
		"Message published to queue orders",
		"Health check passed",
		"Connection established to %s",
		"Session started for user_%d",
		"Configuration loaded from %s",
		"Metrics collected: %d datapoints",
		"Background job completed",
		"Rate limit check passed",
		"Token validated successfully",
		"Feature flag %s evaluated to true",
	}

	warnMessages = []string{
		"High latency detected: %dms",
		"Cache miss, fetching from database",
		"Retry attempt %d of 3",
		"Connection pool exhausted, waiting",
		"Rate limit approaching: %d%% used",
		"Deprecated API called: %s",
		"Memory usage high: %d%%",
		"Queue backlog growing: %d messages",
		"Slow query detected: %dms",
		"Circuit breaker half-open",
	}

	errorMessages = []string{
		"Connection refused to %s:%d",
		"Timeout waiting for response after %dms",
		"Database connection failed: %s",
		"Authentication failed for user_%d",
		"Invalid request payload: missing field %s",
		"Service unavailable: %s",
		"Out of memory: heap usage %d%%",
		"Disk space critical: %d%% used",
		"Transaction failed: %s",
		"Circuit breaker open for %s",
	}

	methods = []string{"GET", "POST", "PUT", "DELETE", "PATCH"}
	paths   = []string{
		"/api/v1/users", "/api/v1/orders", "/api/v1/products",
		"/api/v1/payments", "/api/v1/auth/login", "/api/v1/search",
		"/api/v2/users/%d", "/api/v2/orders/%d/status",
	}
)

func main() {
	size := flag.Int("size", 1000000, "Number of log entries to generate")
	output := flag.String("output", "/tmp/benchmark-logs.json", "Output file path")
	errorRate := flag.Int("error", 5, "Error log percentage")
	warnRate := flag.Int("warn", 10, "Warning log percentage")
	compress := flag.Bool("gzip", false, "Compress output with gzip")
	flag.Parse()

	fmt.Printf("Generating %d log entries to %s...\n", *size, *output)
	fmt.Printf("  Error rate: %d%%\n", *errorRate)
	fmt.Printf("  Warn rate: %d%%\n", *warnRate)
	fmt.Printf("  Compress: %v\n", *compress)
	fmt.Println()

	// Create output directory
	if err := os.MkdirAll(filepath.Dir(*output), 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating directory: %v\n", err)
		os.Exit(1)
	}

	// Open output file
	file, err := os.Create(*output)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating file: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	var writer *bufio.Writer
	var gzWriter *gzip.Writer

	if *compress {
		gzWriter = gzip.NewWriter(file)
		defer gzWriter.Close()
		writer = bufio.NewWriterSize(gzWriter, 1024*1024) // 1MB buffer
	} else {
		writer = bufio.NewWriterSize(file, 1024*1024)
	}
	defer writer.Flush()

	start := time.Now()
	baseTime := time.Now().Add(-24 * time.Hour) // Spread over last 24h

	// Pre-allocate JSON encoder buffer
	enc := json.NewEncoder(writer)

	for i := 0; i < *size; i++ {
		entry := generateEntry(baseTime, *errorRate, *warnRate, i, *size)

		if err := enc.Encode(entry); err != nil {
			fmt.Fprintf(os.Stderr, "Error encoding entry: %v\n", err)
			os.Exit(1)
		}

		// Progress reporting
		if (i+1)%100000 == 0 {
			elapsed := time.Since(start).Seconds()
			rate := float64(i+1) / elapsed
			fmt.Printf("  Generated %d / %d entries (%.0f/s)...\n", i+1, *size, rate)
		}
	}

	writer.Flush()
	if gzWriter != nil {
		gzWriter.Close()
	}
	file.Close()

	// Final stats
	stat, _ := os.Stat(*output)
	elapsed := time.Since(start)
	fmt.Println()
	fmt.Println("Done!")
	fmt.Printf("  File: %s\n", *output)
	fmt.Printf("  Size: %s\n", formatSize(stat.Size()))
	fmt.Printf("  Entries: %d\n", *size)
	fmt.Printf("  Duration: %v\n", elapsed.Round(time.Millisecond))
	fmt.Printf("  Rate: %.0f entries/s\n", float64(*size)/elapsed.Seconds())
}

func generateEntry(baseTime time.Time, errorRate, warnRate, index, total int) LogEntry {
	// Spread timestamps across the time range
	offset := time.Duration(float64(index) / float64(total) * float64(24*time.Hour))
	timestamp := baseTime.Add(offset).Add(time.Duration(rand.Intn(1000)) * time.Millisecond)

	service := services[rand.Intn(len(services))]

	// Determine level
	var level, message string
	r := rand.Intn(100)
	if r < errorRate {
		level = "ERROR"
		msg := errorMessages[rand.Intn(len(errorMessages))]
		message = fmt.Sprintf(msg, randomArgs()...)
	} else if r < errorRate+warnRate {
		level = "WARN"
		msg := warnMessages[rand.Intn(len(warnMessages))]
		message = fmt.Sprintf(msg, randomArgs()...)
	} else {
		level = "INFO"
		msg := infoMessages[rand.Intn(len(infoMessages))]
		message = fmt.Sprintf(msg, randomArgs()...)
	}

	entry := LogEntry{
		Timestamp: timestamp.UTC().Format("2006-01-02T15:04:05.000Z"),
		Level:     level,
		Message:   message,
		App:       service,
		LatencyMs: rand.Intn(5000) + 1,
	}

	// Add optional fields randomly
	if rand.Intn(2) == 0 {
		entry.TraceID = fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
			rand.Uint32(), rand.Intn(65536), rand.Intn(65536), rand.Intn(65536), rand.Uint64()&0xffffffffffff)
	}

	if rand.Intn(3) == 0 {
		entry.UserID = fmt.Sprintf("user_%d", rand.Intn(10000))
	}

	if rand.Intn(4) == 0 {
		entry.RequestID = fmt.Sprintf("req_%08x", rand.Uint32())
	}

	if rand.Intn(3) == 0 {
		entry.StatusCode = []int{200, 201, 400, 401, 403, 404, 500, 502, 503}[rand.Intn(9)]
		entry.Method = methods[rand.Intn(len(methods))]
		path := paths[rand.Intn(len(paths))]
		if strings.Contains(path, "%d") {
			path = fmt.Sprintf(path, rand.Intn(10000))
		}
		entry.Path = path
	}

	return entry
}

func randomArgs() []interface{} {
	return []interface{}{
		rand.Intn(10000),
		rand.Intn(100),
		services[rand.Intn(len(services))],
		fmt.Sprintf("field_%d", rand.Intn(10)),
	}
}

func formatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
