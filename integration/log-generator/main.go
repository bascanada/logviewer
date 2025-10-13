package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

// Simple struct for our structured logs
type LogEntry struct {
	Timestamp time.Time `json:"@timestamp"`
	Level     string    `json:"level"`
	Message   string    `json:"message"`
	App       string    `json:"app"`
	RequestID string    `json:"request_id,omitempty"`
}

var (
	splunkHecURL   = "https://splunk:8088/services/collector"
	splunkHecToken = os.Getenv("SPLUNK_HEC_TOKEN") // Your HEC token
	opensearchURL  = "http://opensearch:9200/app-logs/_doc"
)

func main() {
	if splunkHecURL == "" || splunkHecToken == "" || opensearchURL == "" {
		log.Fatal("SPLUNK_HEC_URL, SPLUNK_HEC_TOKEN, and OPENSEARCH_URL environment variables are required")
	}

	http.HandleFunc("/log/info", handleLogRequest("INFO"))
	http.HandleFunc("/log/error", handleLogRequest("ERROR"))
	http.HandleFunc("/log/warn", handleLogRequest("WARN"))

	log.Println("Log generator server starting on :8081")
	if err := http.ListenAndServe(":8081", nil); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func handleLogRequest(level string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		msg := r.URL.Query().Get("message")
		if msg == "" {
			msg = fmt.Sprintf("This is a sample %s message.", level)
		}
		requestID := r.Header.Get("X-Request-ID")

		entry := LogEntry{
			Timestamp: time.Now(),
			Level:     level,
			Message:   msg,
			App:       "log-generator",
			RequestID: requestID,
		}

		sendToSplunk(entry)
		sendToOpenSearch(entry)

		fmt.Fprintf(w, "Logged: %s\n", msg)
	}
}

func sendToSplunk(entry LogEntry) {
	// Splunk HEC expects a specific JSON format
	payload := map[string]interface{}{
		"event": entry,
	}
	jsonData, err := json.Marshal(payload)
	if err != nil {
		log.Printf("ERROR: could not marshal splunk log: %v", err)
		return
	}

	req, _ := http.NewRequest("POST", splunkHecURL, bytes.NewBuffer(jsonData))
	req.Header.Set("Authorization", "Splunk "+splunkHecToken)
	req.Header.Set("Content-Type", "application/json")

	// In a real app, use a shared HTTP client, but for this example, this is fine
	client := &http.Client{
		// In a real test env, you might need to skip verification for self-signed certs
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("ERROR: failed to send log to splunk: %v", err)
		return
	}
	defer resp.Body.Close()
	log.Printf("Sent to Splunk, status: %s", resp.Status)
}

func sendToOpenSearch(entry LogEntry) {
	jsonData, err := json.Marshal(entry)
	if err != nil {
		log.Printf("ERROR: could not marshal opensearch log: %v", err)
		return
	}

	req, _ := http.NewRequest("POST", opensearchURL, bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("ERROR: failed to send log to opensearch: %v", err)
		return
	}
	defer resp.Body.Close()
	log.Printf("Sent to OpenSearch, status: %s", resp.Status)
}
