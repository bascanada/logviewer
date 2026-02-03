package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/google/uuid"
)

// LogEntry for structured logging
type LogEntry struct {
	Timestamp time.Time `json:"@timestamp"`
	Level     string    `json:"level"`
	Message   string    `json:"message"`
	App       string    `json:"app"`
	TraceID   string    `json:"trace_id,omitempty"`
	Latency   int64     `json:"latency_ms,omitempty"`
	RunID     string    `json:"run_id,omitempty"` // For test run isolation
}

var (
	splunkHecURL   = os.Getenv("SPLUNK_HEC_URL")
	splunkHecToken = os.Getenv("SPLUNK_HEC_TOKEN")
	opensearchURL  = os.Getenv("OPENSEARCH_URL")
	localstackURL  = os.Getenv("LOCALSTACK_URL") // e.g. http://localstack:4566
	dynamoClient   *dynamodb.Client
)

func main() {
	initDynamoDB()

	// 0. Health Check Endpoint
	http.HandleFunc("/health", handleHealth)

	// 1. Trigger Endpoint (Manual)
	http.HandleFunc("/checkout", handleCheckout)

	// 2. Test Data Seeding Endpoint
	http.HandleFunc("/seed", handleSeed)

	// 3. Load Generator (Automatic background traffic)
	go startLoadGenerator()

	log.Println("Transaction Simulator running on :8081")
	log.Println("Test data seeding available at POST /seed")
	if err := http.ListenAndServe(":8081", nil); err != nil {
		log.Fatal(err)
	}
}

// initDynamoDB connects to LocalStack
func initDynamoDB() {
	if localstackURL == "" {
		return
	}

	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion("us-east-1"),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("test", "test", "")),
		config.WithEndpointResolverWithOptions(aws.EndpointResolverWithOptionsFunc(
			func(service, region string, options ...interface{}) (aws.Endpoint, error) {
				return aws.Endpoint{URL: localstackURL}, nil
			})),
	)
	if err != nil {
		log.Printf("Failed to load AWS config: %v", err)
		return
	}
	dynamoClient = dynamodb.NewFromConfig(cfg)

	// Create table if not exists (ignoring errors for brevity)
	dynamoClient.CreateTable(context.TODO(), &dynamodb.CreateTableInput{
		TableName:            aws.String("Orders"),
		KeySchema:            []types.KeySchemaElement{{AttributeName: aws.String("OrderID"), KeyType: types.KeyTypeHash}},
		AttributeDefinitions: []types.AttributeDefinition{{AttributeName: aws.String("OrderID"), AttributeType: types.ScalarAttributeTypeS}},
		BillingMode:          types.BillingModePayPerRequest,
	})
}

func startLoadGenerator() {
	// Wait for startup
	time.Sleep(5 * time.Second)
	log.Println("Starting background load generator...")

	for {
		// Simulate traffic bursts
		go simulateTransaction()
		time.Sleep(time.Duration(rand.Intn(2000)+500) * time.Millisecond)
	}
}

func handleCheckout(w http.ResponseWriter, r *http.Request) {
	simulateTransaction()
	w.Write([]byte("Transaction simulated"))
}

func simulateTransaction() {
	traceID := uuid.New().String()
	start := time.Now()

	// --- 1. Frontend Service (Nginx Style -> Stdout/K8s Logs) ---
	// "192.168.1.5 - - [Date] "POST /api/checkout" 200 ..."
	fmt.Printf("10.0.0.%d - - [%s] \"POST /api/checkout HTTP/1.1\" 200 1024 \"Mozilla/5.0\" trace_id=%s\n",
		rand.Intn(255), time.Now().Format("02/Jan/2006:15:04:05 -0700"), traceID)

	// --- 2. Order Service (JSON -> OpenSearch) ---
	sendToOpenSearch(LogEntry{
		Timestamp: time.Now(),
		Level:     "INFO",
		App:       "order-service",
		Message:   fmt.Sprintf("Processing order for user_%d", rand.Intn(1000)),
		TraceID:   traceID,
	})

	// Introduce realistic latency
	processTime := rand.Intn(500)
	time.Sleep(time.Duration(processTime) * time.Millisecond)

	// --- 3. Payment Service (Key-Value Text -> Splunk) ---
	// Occasional "Slow" error
	if rand.Float32() < 0.1 { // 10% chance of error
		sendToSplunk(LogEntry{
			Timestamp: time.Now(),
			Level:     "ERROR",
			App:       "payment-service",
			Message:   "PaymentGatewayTimeoutException: Upstream provider failed to respond in 30s",
			TraceID:   traceID,
			Latency:   30000,
		})
		// Also log to DynamoDB about the failure
		logToDynamo(traceID, "FAILED", "Payment Timeout")
	} else {
		sendToSplunk(LogEntry{
			Timestamp: time.Now(),
			Level:     "INFO",
			App:       "payment-service",
			Message:   "Payment authorized successfully auth_code=XYZ123",
			TraceID:   traceID,
			Latency:   int64(processTime),
		})
		logToDynamo(traceID, "SUCCESS", "Authorized")
	}

	// --- 4. Database (Stdout/K8s Logs - simulating separate pod) ---
	// Occasional deadlock
	if rand.Float32() < 0.05 {
		fmt.Printf("%s [ERROR] [database-primary] Deadlock found when trying to get lock; try restarting transaction trace_id=%s\n",
			time.Now().Format(time.RFC3339), traceID)
	}

	totalLatency := time.Since(start).Milliseconds()
	if totalLatency > 1000 {
		sendToOpenSearch(LogEntry{
			Timestamp: time.Now(),
			Level:     "WARN",
			App:       "api-gateway",
			Message:   "Slow request detected",
			TraceID:   traceID,
			Latency:   totalLatency,
		})
	}
}

func logToDynamo(id, status, msg string) {
	if dynamoClient == nil {
		return
	}
	dynamoClient.PutItem(context.TODO(), &dynamodb.PutItemInput{
		TableName: aws.String("Orders"),
		Item: map[string]types.AttributeValue{
			"OrderID": &types.AttributeValueMemberS{Value: id},
			"Status":  &types.AttributeValueMemberS{Value: status},
			"Message": &types.AttributeValueMemberS{Value: msg},
			"TTL":     &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", time.Now().Unix()+3600)},
		},
	})
}

// getSplunkHTTPClient returns an HTTP client configured for Splunk HEC with TLS settings
// from environment variables. For integration testing, SPLUNK_HEC_INSECURE=true allows
// connections to self-signed certificates.
func getSplunkHTTPClient() *http.Client {
	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}

	// Allow insecure for integration testing
	if os.Getenv("SPLUNK_HEC_INSECURE") == "true" {
		tlsConfig.InsecureSkipVerify = true
	} else if caCert := os.Getenv("SPLUNK_HEC_CA_CERT"); caCert != "" {
		// Support custom CA cert
		certPool := x509.NewCertPool()
		certPool.AppendCertsFromPEM([]byte(caCert))
		tlsConfig.RootCAs = certPool
	} else if caCertFile := os.Getenv("SPLUNK_HEC_CA_CERT_FILE"); caCertFile != "" {
		// Support custom CA cert file
		caCertPEM, err := os.ReadFile(caCertFile)
		if err == nil {
			certPool := x509.NewCertPool()
			certPool.AppendCertsFromPEM(caCertPEM)
			tlsConfig.RootCAs = certPool
		}
	}

	return &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}
}

func sendToSplunk(entry LogEntry) {
	if splunkHecURL == "" {
		return
	}
	payload, _ := json.Marshal(map[string]interface{}{"event": entry, "sourcetype": "json"})
	req, _ := http.NewRequest("POST", splunkHecURL, bytes.NewBuffer(payload))
	req.Header.Set("Authorization", "Splunk "+splunkHecToken)
	req.Header.Set("Content-Type", "application/json")

	client := getSplunkHTTPClient()
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("ERROR: failed to send log to splunk: %v", err)
		return
	}
	defer resp.Body.Close()
}

func sendToOpenSearch(entry LogEntry) {
	if opensearchURL == "" {
		return
	}
	// Transaction simulator sends to app-logs index
	url := fmt.Sprintf("%s/app-logs/_doc", opensearchURL)
	payload, _ := json.Marshal(entry)
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("ERROR: failed to send log to opensearch: %v", err)
		return
	}
	defer resp.Body.Close()
}

// === TEST DATA SEEDING ===

// TestFixture represents a set of test logs to seed
type TestFixture struct {
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Index       string     `json:"index"` // Splunk/OpenSearch index
	Logs        []LogEntry `json:"logs"`
}

// SeedRequest is the request payload for /seed endpoint
type SeedRequest struct {
	Fixtures []string `json:"fixtures"`         // Names of fixtures to seed
	RunID    string   `json:"run_id,omitempty"` // Unique ID to tag logs with
}

// SeedResponse is the response from /seed endpoint
type SeedResponse struct {
	Success bool           `json:"success"`
	Message string         `json:"message"`
	Seeded  map[string]int `json:"seeded"` // fixture name -> count
	Errors  []string       `json:"errors,omitempty"`
	RunID   string         `json:"run_id,omitempty"` // Echo back the run_id
}

// getTestFixtures returns all available test fixtures
func getTestFixtures(runID string) map[string]TestFixture {
	baseTime := time.Now().Add(-1 * time.Hour) // Start from 1 hour ago

	fixtures := make(map[string]TestFixture)

	// Fixture 1: Error logs (50 errors from payment service)
	errorLogs := TestFixture{
		Name:        "error-logs",
		Description: "50 ERROR level logs from payment-service",
		Index:       "test-e2e-errors",
		Logs:        make([]LogEntry, 50),
	}
	for i := 0; i < 50; i++ {
		errorLogs.Logs[i] = LogEntry{
			Timestamp: baseTime.Add(time.Duration(i) * time.Second),
			Level:     "ERROR",
			Message:   fmt.Sprintf("Payment failed: insufficient funds (user_%d)", 1000+i),
			App:       "payment-service",
			TraceID:   fmt.Sprintf("test-trace-%03d", i+1),
			Latency:   int64(1200 + rand.Intn(500)),
			RunID:     runID,
		}
	}
	fixtures["error-logs"] = errorLogs

	// Fixture 2: Payment logs (100 mixed payment service logs)
	paymentLogs := TestFixture{
		Name:        "payment-logs",
		Description: "100 payment-service logs (80 INFO, 20 ERROR)",
		Index:       "test-e2e-payments",
		Logs:        make([]LogEntry, 100),
	}
	for i := 0; i < 100; i++ {
		level := "INFO"
		msg := fmt.Sprintf("Payment authorized successfully auth_code=TEST%d", i)
		latency := int64(100 + rand.Intn(300))

		if i%5 == 0 { // Every 5th is an error
			level = "ERROR"
			msg = fmt.Sprintf("Payment declined: card_declined (attempt_%d)", i)
			latency = int64(1500 + rand.Intn(1000))
		}

		paymentLogs.Logs[i] = LogEntry{
			Timestamp: baseTime.Add(time.Duration(i*2) * time.Second),
			Level:     level,
			Message:   msg,
			App:       "payment-service",
			TraceID:   fmt.Sprintf("test-pay-%03d", i+1),
			Latency:   latency,
			RunID:     runID,
		}
	}
	fixtures["payment-logs"] = paymentLogs

	// Fixture 3: Order logs (75 order service logs)
	orderLogs := TestFixture{
		Name:        "order-logs",
		Description: "75 order-service logs (mixed levels)",
		Index:       "test-e2e-orders",
		Logs:        make([]LogEntry, 75),
	}
	for i := 0; i < 75; i++ {
		level := "INFO"
		msg := fmt.Sprintf("Processing order #ORD%05d for user_%d", i+1, 2000+i)

		if i%10 == 0 {
			level = "WARN"
			msg = fmt.Sprintf("Low inventory warning for product_%d", i)
		} else if i%15 == 0 {
			level = "ERROR"
			msg = fmt.Sprintf("Failed to process order #ORD%05d: inventory unavailable", i+1)
		}

		orderLogs.Logs[i] = LogEntry{
			Timestamp: baseTime.Add(time.Duration(i*3) * time.Second),
			Level:     level,
			Message:   msg,
			App:       "order-service",
			TraceID:   fmt.Sprintf("test-ord-%03d", i+1),
			Latency:   int64(50 + rand.Intn(200)),
			RunID:     runID,
		}
	}
	fixtures["order-logs"] = orderLogs

	// Fixture 4: Mixed levels (100 logs: 50 INFO, 30 WARN, 20 ERROR)
	mixedLogs := TestFixture{
		Name:        "mixed-levels",
		Description: "100 logs with mixed levels (50 INFO, 30 WARN, 20 ERROR)",
		Index:       "test-e2e-mixed",
		Logs:        make([]LogEntry, 100),
	}
	levelDist := []string{}
	for i := 0; i < 50; i++ {
		levelDist = append(levelDist, "INFO")
	}
	for i := 0; i < 30; i++ {
		levelDist = append(levelDist, "WARN")
	}
	for i := 0; i < 20; i++ {
		levelDist = append(levelDist, "ERROR")
	}

	for i := 0; i < 100; i++ {
		mixedLogs.Logs[i] = LogEntry{
			Timestamp: baseTime.Add(time.Duration(i) * time.Second),
			Level:     levelDist[i],
			Message:   fmt.Sprintf("Log message %d with level %s", i+1, levelDist[i]),
			App:       "api-gateway",
			TraceID:   fmt.Sprintf("test-mix-%03d", i+1),
			Latency:   int64(100 + rand.Intn(500)),
			RunID:     runID,
		}
	}
	fixtures["mixed-levels"] = mixedLogs

	// Fixture 5: Trace logs (30 logs with trace_ids for distributed tracing tests)
	traceLogs := TestFixture{
		Name:        "trace-logs",
		Description: "30 logs with trace_ids for distributed tracing",
		Index:       "test-e2e-traces",
		Logs:        make([]LogEntry, 30),
	}
	for i := 0; i < 30; i++ {
		traceLogs.Logs[i] = LogEntry{
			Timestamp: baseTime.Add(time.Duration(i*2) * time.Second),
			Level:     "INFO",
			Message:   fmt.Sprintf("Request processed successfully (step %d)", i+1),
			App:       "api-gateway",
			TraceID:   fmt.Sprintf("trace-distributed-%03d", i+1),
			Latency:   int64(50 + rand.Intn(150)),
			RunID:     runID,
		}
	}
	fixtures["trace-logs"] = traceLogs

	// Fixture 6: Slow requests (25 logs with high latency)
	slowLogs := TestFixture{
		Name:        "slow-requests",
		Description: "25 logs with latency > 1000ms",
		Index:       "test-e2e-slow",
		Logs:        make([]LogEntry, 25),
	}
	for i := 0; i < 25; i++ {
		slowLogs.Logs[i] = LogEntry{
			Timestamp: baseTime.Add(time.Duration(i*4) * time.Second),
			Level:     "WARN",
			Message:   fmt.Sprintf("Slow request detected: endpoint=/api/search query_time=%dms", 1000+rand.Intn(2000)),
			App:       "api-gateway",
			TraceID:   fmt.Sprintf("test-slow-%03d", i+1),
			Latency:   int64(1000 + rand.Intn(2000)),
			RunID:     runID,
		}
	}
	fixtures["slow-requests"] = slowLogs

	return fixtures
}

// handleHealth handles the /health endpoint for health checks
func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "ok",
		"service": "log-generator",
	})
}

// handleSeed handles the /seed endpoint for test data seeding
func handleSeed(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req SeedRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Generate RunID if not provided
	if req.RunID == "" {
		req.RunID = fmt.Sprintf("auto-%d", time.Now().UnixNano())
	}

	log.Printf("Received seed request for fixtures: %v with RunID: %s", req.Fixtures, req.RunID)

	allFixtures := getTestFixtures(req.RunID)
	response := SeedResponse{
		Success: true,
		Seeded:  make(map[string]int),
		Errors:  []string{},
		RunID:   req.RunID,
	}

	// If no fixtures specified, seed all
	if len(req.Fixtures) == 0 {
		req.Fixtures = []string{"error-logs", "payment-logs", "order-logs", "mixed-levels", "trace-logs", "slow-requests"}
	}

	for _, fixtureName := range req.Fixtures {
		fixture, exists := allFixtures[fixtureName]
		if !exists {
			response.Errors = append(response.Errors, fmt.Sprintf("Fixture '%s' not found", fixtureName))
			response.Success = false
			continue
		}

		log.Printf("Seeding fixture '%s': %d logs to index '%s'", fixture.Name, len(fixture.Logs), fixture.Index)

		// Send logs to Splunk with custom index
		seededCount := 0
		for _, logEntry := range fixture.Logs {
			if sendToSplunkWithIndex(logEntry, fixture.Index) {
				seededCount++
			}
		}

		// Also send to OpenSearch with custom index
		for _, logEntry := range fixture.Logs {
			sendToOpenSearchWithIndex(logEntry, fixture.Index)
		}

		response.Seeded[fixtureName] = seededCount
		log.Printf("Successfully seeded %d logs for fixture '%s'", seededCount, fixture.Name)
	}

	if len(response.Errors) == 0 {
		response.Message = fmt.Sprintf("Successfully seeded %d fixtures", len(response.Seeded))
	} else {
		response.Message = fmt.Sprintf("Seeded %d fixtures with %d errors", len(response.Seeded), len(response.Errors))
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// sendToSplunkWithIndex sends a log entry to Splunk HEC with custom index
func sendToSplunkWithIndex(entry LogEntry, index string) bool {
	if splunkHecURL == "" {
		return false
	}

	payload, _ := json.Marshal(map[string]interface{}{
		"event":      entry,
		"sourcetype": "json",
		"index":      index,
	})

	req, _ := http.NewRequest("POST", splunkHecURL, bytes.NewBuffer(payload))
	req.Header.Set("Authorization", "Splunk "+splunkHecToken)
	req.Header.Set("Content-Type", "application/json")

	client := getSplunkHTTPClient()
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("ERROR: failed to send log to Splunk: %v", err)
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		log.Printf("ERROR: Splunk returned status %d", resp.StatusCode)
		return false
	}

	return true
}

// sendToOpenSearchWithIndex sends a log entry to OpenSearch with custom index
func sendToOpenSearchWithIndex(entry LogEntry, index string) {
	if opensearchURL == "" {
		return
	}

	// OpenSearch bulk API format
	url := fmt.Sprintf("%s/%s/_doc", opensearchURL, index)
	payload, _ := json.Marshal(entry)

	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("ERROR: failed to send log to OpenSearch: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		log.Printf("ERROR: OpenSearch returned status %d", resp.StatusCode)
	}
}
