package main

import (
	"bytes"
	"context"
	"crypto/tls"
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

	// 1. Trigger Endpoint (Manual)
	http.HandleFunc("/checkout", handleCheckout)

	// 2. Load Generator (Automatic background traffic)
	go startLoadGenerator()

	log.Println("Transaction Simulator running on :8081")
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

func sendToSplunk(entry LogEntry) {
	if splunkHecURL == "" {
		return
	}
	payload, _ := json.Marshal(map[string]interface{}{"event": entry, "sourcetype": "json"})
	req, _ := http.NewRequest("POST", splunkHecURL, bytes.NewBuffer(payload))
	req.Header.Set("Authorization", "Splunk "+splunkHecToken)
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}}
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
	payload, _ := json.Marshal(entry)
	req, _ := http.NewRequest("POST", opensearchURL, bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("ERROR: failed to send log to opensearch: %v", err)
		return
	}
	defer resp.Body.Close()
}
