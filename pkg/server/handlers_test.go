package server

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/berlingoqc/logviewer/pkg/log/client"
	"github.com/berlingoqc/logviewer/pkg/log/client/config"
	"github.com/berlingoqc/logviewer/pkg/ty"
	"github.com/stretchr/testify/assert"
)

func newTestServer(t *testing.T, cfg *config.ContextConfig) *Server {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	if cfg == nil {
		cfg = &config.ContextConfig{}
	}
	return NewServer("localhost", "8080", cfg, logger)
}

func TestHealthHandler(t *testing.T) {
	s := newTestServer(t, nil)

	req, err := http.NewRequest("GET", "/health", nil)
	assert.NoError(t, err)

	rr := httptest.NewRecorder()
	s.router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code, "handler returned wrong status code")

	expected := `{"status":"ok"}` + "\n"
	assert.JSONEq(t, expected, rr.Body.String(), "handler returned unexpected body")
}

func TestContextsHandler_List(t *testing.T) {
	cfg := &config.ContextConfig{
		Contexts: map[string]config.SearchContext{
			"ctx1": {Client: "client1", SearchInherit: []string{"s1"}},
			"ctx2": {Client: "client2"},
		},
	}
	s := newTestServer(t, cfg)

	req, err := http.NewRequest("GET", "/contexts", nil)
	assert.NoError(t, err)

	rr := httptest.NewRecorder()
	s.router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var resp ContextsResponse
	err = json.Unmarshal(rr.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Len(t, resp.Contexts, 2)

	foundCtx1 := false
	foundCtx2 := false
	for _, c := range resp.Contexts {
		if c.Id == "ctx1" {
			foundCtx1 = true
			assert.Equal(t, "client1", c.Client)
			assert.Equal(t, []string{"s1"}, c.SearchInherit)
		}
		if c.Id == "ctx2" {
			foundCtx2 = true
			assert.Equal(t, "client2", c.Client)
		}
	}
	assert.True(t, foundCtx1, "context ctx1 not found in response")
	assert.True(t, foundCtx2, "context ctx2 not found in response")
}

func TestContextsHandler_Detail(t *testing.T) {
	cfg := &config.ContextConfig{
		Contexts: map[string]config.SearchContext{
			"ctx1": {Client: "client1", SearchInherit: []string{"s1"}},
		},
	}
	s := newTestServer(t, cfg)

	req, err := http.NewRequest("GET", "/contexts/ctx1", nil)
	assert.NoError(t, err)

	rr := httptest.NewRecorder()
	s.router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var resp ContextInfo
	err = json.Unmarshal(rr.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "ctx1", resp.Id)
	assert.Equal(t, "client1", resp.Client)
}

func TestContextsHandler_NotFound(t *testing.T) {
	s := newTestServer(t, &config.ContextConfig{})

	req, err := http.NewRequest("GET", "/contexts/nonexistent", nil)
	assert.NoError(t, err)

	rr := httptest.NewRecorder()
	s.router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

// Mocks for testing query handlers
type mockLogSearchResult struct {
	client.LogSearchResult
}

func (m *mockLogSearchResult) GetEntries(ctx context.Context) ([]client.LogEntry, chan []client.LogEntry, error) {
	return []client.LogEntry{{Message: "test log"}}, nil, nil
}
func (m *mockLogSearchResult) GetFields() (ty.UniSet[string], chan ty.UniSet[string], error) {
	return ty.UniSet[string]{"field1": []string{"value1"}}, nil, nil
}

func TestQueryLogsHandler(t *testing.T) {
	// TODO: This test needs to mock the factory.GetLogSearchFactory
	// and factory.GetLogClientFactory to return mock implementations
	// that don't make real network calls. This is a placeholder.

	t.Skip("skipping test for queryLogsHandler, requires mocking factories")

	cfg := &config.ContextConfig{
		Clients:  map[string]config.Client{"c1": {Type: "mock"}},
		Contexts: map[string]config.SearchContext{"ctx1": {Client: "c1"}},
	}
	s := newTestServer(t, cfg)

	body := `{"contextId": "ctx1"}`
	req, err := http.NewRequest("POST", "/query/logs", strings.NewReader(body))
	assert.NoError(t, err)

	rr := httptest.NewRecorder()
	s.router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}
