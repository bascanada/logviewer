package server

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/berlingoqc/logviewer/pkg/log/client"
	"github.com/berlingoqc/logviewer/pkg/log/factory"
)

// Base request structure for query endpoints
type QueryRequest struct {
	ContextId string           `json:"contextId"`           // Required
	Inherits  []string         `json:"inherits,omitempty"`  // Optional search inherits
	Search    client.LogSearch `json:"search"`              // Search overrides
}

// Response for /query/logs endpoint
type LogsResponse struct {
	Logs  []client.LogEntry `json:"logs,omitempty"`
	Error string            `json:"error,omitempty"`
	Meta  QueryMetadata     `json:"meta,omitempty"`
}

// Response for /query/fields endpoint
type FieldsResponse struct {
	Fields map[string][]string `json:"fields,omitempty"` // field_name -> [possible_values]
	Error  string              `json:"error,omitempty"`
	Meta   QueryMetadata       `json:"meta,omitempty"`
}

// Response for /contexts endpoint
type ContextsResponse struct {
	Contexts []ContextInfo `json:"contexts"`
}

type ContextInfo struct {
	Id            string   `json:"id"`
	Client        string   `json:"client"`
	Description   string   `json:"description,omitempty"`
	SearchInherit []string `json:"searchInherit,omitempty"`
}

// Metadata about query execution
type QueryMetadata struct {
	QueryTime   string `json:"queryTime"`   // How long the query took
	ResultCount int    `json:"resultCount"` // Number of results returned
	ContextUsed string `json:"contextUsed"` // Which context was used
	ClientType  string `json:"clientType"`  // opensearch, splunk, k8s, etc.
}

func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	s.writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) queryLogsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Only POST method is allowed")
		return
	}

	var req QueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, ErrCodeInvalidSearch, "Invalid request body")
		return
	}

	if err := s.validateQueryRequest(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, ErrCodeValidationError, err.Error())
		return
	}

	startTime := time.Now()

	clientFactory, err := factory.GetLogClientFactory(s.config.Clients)
	if err != nil {
		s.logger.Error("failed to get log client factory", "err", err)
		s.writeError(w, http.StatusInternalServerError, ErrCodeConfigError, "Internal server error")
		return
	}

	searchFactory, err := factory.GetLogSearchFactory(clientFactory, *s.config)
	if err != nil {
		s.logger.Error("failed to get log search factory", "err", err)
		s.writeError(w, http.StatusInternalServerError, ErrCodeConfigError, "Internal server error")
		return
	}

	searchResult, err := searchFactory.GetSearchResult(req.ContextId, req.Inherits, req.Search)
	if err != nil {
		s.logger.Error("failed to get search result", "err", err, "contextId", req.ContextId)
		s.writeError(w, http.StatusBadRequest, ErrCodeInvalidSearch, err.Error())
		return
	}

	entries, _, err := searchResult.GetEntries(r.Context())
	if err != nil {
		s.logger.Error("failed to get log entries", "err", err)
		s.writeError(w, http.StatusInternalServerError, ErrCodeBackendError, "Failed to retrieve logs from backend")
		return
	}

	sc, err := s.config.GetSearchContext(req.ContextId, req.Inherits, req.Search)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, ErrCodeContextNotFound, "Could not get search context")
		return
	}

	resp := LogsResponse{
		Logs: entries,
		Meta: QueryMetadata{
			QueryTime:   time.Since(startTime).String(),
			ResultCount: len(entries),
			ContextUsed: req.ContextId,
			ClientType:  s.config.Clients[sc.Client].Type,
		},
	}

	s.writeJSON(w, http.StatusOK, resp)
}

func (s *Server) queryFieldsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Only POST method is allowed")
		return
	}

	var req QueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, ErrCodeInvalidSearch, "Invalid request body")
		return
	}

	if err := s.validateQueryRequest(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, ErrCodeValidationError, err.Error())
		return
	}

	startTime := time.Now()

	clientFactory, err := factory.GetLogClientFactory(s.config.Clients)
	if err != nil {
		s.logger.Error("failed to get log client factory", "err", err)
		s.writeError(w, http.StatusInternalServerError, ErrCodeConfigError, "Internal server error")
		return
	}

	searchFactory, err := factory.GetLogSearchFactory(clientFactory, *s.config)
	if err != nil {
		s.logger.Error("failed to get log search factory", "err", err)
		s.writeError(w, http.StatusInternalServerError, ErrCodeConfigError, "Internal server error")
		return
	}

	searchResult, err := searchFactory.GetSearchResult(req.ContextId, req.Inherits, req.Search)
	if err != nil {
		s.logger.Error("failed to get search result", "err", err, "contextId", req.ContextId)
		s.writeError(w, http.StatusBadRequest, ErrCodeInvalidSearch, err.Error())
		return
	}

	_, _, _ = searchResult.GetEntries(r.Context())

	fields, _, err := searchResult.GetFields()
	if err != nil {
		s.logger.Error("failed to get fields", "err", err)
		s.writeError(w, http.StatusInternalServerError, ErrCodeBackendError, "Failed to retrieve fields from backend")
		return
	}

	sc, err := s.config.GetSearchContext(req.ContextId, req.Inherits, req.Search)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, ErrCodeContextNotFound, "Could not get search context")
		return
	}

	resp := FieldsResponse{
		Fields: fields,
		Meta: QueryMetadata{
			QueryTime:   time.Since(startTime).String(),
			ResultCount: len(fields),
			ContextUsed: req.ContextId,
			ClientType:  s.config.Clients[sc.Client].Type,
		},
	}

	s.writeJSON(w, http.StatusOK, resp)
}

func (s *Server) contextsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Only GET method is allowed")
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/contexts")
	path = strings.Trim(path, "/")

	if path == "" {
		// List all contexts
		var contexts []ContextInfo
		for id, context := range s.config.Contexts {
			contexts = append(contexts, ContextInfo{
				Id:            id,
				Client:        context.Client,
				SearchInherit: context.SearchInherit,
			})
		}
		resp := ContextsResponse{Contexts: contexts}
		s.writeJSON(w, http.StatusOK, resp)
		return
	}

	// Get a specific context
	contextId := path
	context, ok := s.config.Contexts[contextId]
	if !ok {
		s.writeError(w, http.StatusNotFound, ErrCodeContextNotFound, "Context not found")
		return
	}

	info := ContextInfo{
		Id:            contextId,
		Client:        context.Client,
		SearchInherit: context.SearchInherit,
	}
	s.writeJSON(w, http.StatusOK, info)
}
