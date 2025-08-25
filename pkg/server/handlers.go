package server

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/bascanada/logviewer/pkg/log/client"
)

// Base request structure for query endpoints
type QueryRequest struct {
	ContextId string           `json:"contextId"`           // Required
	Inherits  []string         `json:"inherits,omitempty"`  // Optional search inherits
	Search    client.LogSearch `json:"search"`              // Search overrides
	Variables map[string]string `json:"variables,omitempty"` // Variables for substitution
}

// Response for /query/logs endpoint
type LogsResponse struct {
	Logs []client.LogEntry `json:"logs,omitempty"`
	Meta QueryMetadata     `json:"meta,omitempty"`
}

// Response for /query/fields endpoint
type FieldsResponse struct {
	Fields map[string][]string `json:"fields,omitempty"` // field_name -> [possible_values]
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

func (s *Server) queryLogsGETHandler(w http.ResponseWriter, r *http.Request) {
	// Parse required contextId
	contextId := r.URL.Query().Get("contextId")
	if contextId == "" {
		s.writeError(w, http.StatusBadRequest, ErrCodeContextNotFound, "contextId query parameter is required")
		return
	}

	// Parse optional inherits: "inherit1,inherit2"
	var inherits []string
	if inheritsParam := r.URL.Query().Get("inherits"); inheritsParam != "" {
		inherits = strings.Split(inheritsParam, ",")
	}

	// Build LogSearch from query parameters
	search := client.LogSearch{}

	// Parse fields: "field1=value1,field2=value2"
	if fieldsParam := r.URL.Query().Get("fields"); fieldsParam != "" {
		fields := make(map[string]string)
		for _, pair := range strings.Split(fieldsParam, ",") {
			if kv := strings.SplitN(pair, "=", 2); len(kv) == 2 {
				key := strings.TrimSpace(kv[0])
				value := strings.TrimSpace(kv[1])
				fields[key] = value
			}
		}
		search.Fields = fields
	}

	// Parse time range
	if last := r.URL.Query().Get("last"); last != "" {
		search.Range.Last.S(last)
	}
	if gte := r.URL.Query().Get("from"); gte != "" {
		search.Range.Gte.S(gte)
	}
	if lte := r.URL.Query().Get("to"); lte != "" {
		search.Range.Lte.S(lte)
	}

	// Parse size
	if sizeStr := r.URL.Query().Get("size"); sizeStr != "" {
		if size, err := strconv.Atoi(sizeStr); err == nil && size > 0 {
			search.Size.S(size)
		}
	}

	// Parse variables: "key1=val1,key2=val2"
	variables := make(map[string]string)
	if varsParam := r.URL.Query().Get("vars"); varsParam != "" {
		for _, pair := range strings.Split(varsParam, ",") {
			if kv := strings.SplitN(pair, "=", 2); len(kv) == 2 {
				key := strings.TrimSpace(kv[0])
				value := strings.TrimSpace(kv[1])
				variables[key] = value
			}
		}
	}

	// Create QueryRequest and reuse existing logic
	req := QueryRequest{
		ContextId: contextId,
		Inherits:  inherits,
		Search:    search,
		Variables: variables,
	}

	// Log the GET request
	s.logger.Info("GET query logs request",
		"contextId", contextId,
		"remote_addr", r.RemoteAddr)

	// Process using existing POST handler logic
	s.processQueryLogsRequest(w, r, &req)
}

// GET version of /query/fields
func (s *Server) queryFieldsGETHandler(w http.ResponseWriter, r *http.Request) {
	// Parse required contextId
	contextId := r.URL.Query().Get("contextId")
	if contextId == "" {
		s.writeError(w, http.StatusBadRequest, ErrCodeContextNotFound, "contextId query parameter is required")
		return
	}

	// Parse optional inherits
	var inherits []string
	if inheritsParam := r.URL.Query().Get("inherits"); inheritsParam != "" {
		inherits = strings.Split(inheritsParam, ",")
	}

	// Build basic LogSearch for field discovery
	search := client.LogSearch{}

	// Parse time range for field discovery
	if last := r.URL.Query().Get("last"); last != "" {
		search.Range.Last.S(last)
	}

	// Parse variables: "key1=val1,key2=val2"
	variables := make(map[string]string)
	if varsParam := r.URL.Query().Get("vars"); varsParam != "" {
		for _, pair := range strings.Split(varsParam, ",") {
			if kv := strings.SplitN(pair, "=", 2); len(kv) == 2 {
				key := strings.TrimSpace(kv[0])
				value := strings.TrimSpace(kv[1])
				variables[key] = value
			}
		}
	}

	// Create QueryRequest
	req := QueryRequest{
		ContextId: contextId,
		Inherits:  inherits,
		Search:    search,
		Variables: variables,
	}

	s.logger.Info("GET query fields request", "contextId", contextId)

	// Process using existing POST handler logic
	s.processQueryFieldsRequest(w, r, &req)
}

func (s *Server) processQueryLogsRequest(w http.ResponseWriter, r *http.Request, req *QueryRequest) {
	if err := s.validateQueryRequest(req); err != nil {
		s.writeError(w, http.StatusBadRequest, ErrCodeValidationError, err.Error())
		return
	}

	startTime := time.Now()

	searchResult, err := s.searchFactory.GetSearchResult(r.Context(), req.ContextId, req.Inherits, req.Search, req.Variables)
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

	sc, err := s.config.GetSearchContext(req.ContextId, req.Inherits, req.Search, req.Variables)
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

func (s *Server) queryLogsRouter(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.queryLogsGETHandler(w, r)
	case http.MethodPost:
		s.queryLogsHandler(w, r)
	default:
		s.writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Only GET and POST methods are allowed")
	}
}

func (s *Server) queryLogsHandler(w http.ResponseWriter, r *http.Request) {
	var req QueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, ErrCodeInvalidSearch, "Invalid request body")
		return
	}
	s.processQueryLogsRequest(w, r, &req)
}

func (s *Server) queryFieldsRouter(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.queryFieldsGETHandler(w, r)
	case http.MethodPost:
		s.queryFieldsHandler(w, r)
	default:
		s.writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Only GET and POST methods are allowed")
	}
}

func (s *Server) processQueryFieldsRequest(w http.ResponseWriter, r *http.Request, req *QueryRequest) {
	if err := s.validateQueryRequest(req); err != nil {
		s.writeError(w, http.StatusBadRequest, ErrCodeValidationError, err.Error())
		return
	}

	startTime := time.Now()

	searchResult, err := s.searchFactory.GetSearchResult(r.Context(), req.ContextId, req.Inherits, req.Search, req.Variables)
	if err != nil {
		s.logger.Error("failed to get search result", "err", err, "contextId", req.ContextId)
		s.writeError(w, http.StatusBadRequest, ErrCodeInvalidSearch, err.Error())
		return
	}

	fields, _, err := searchResult.GetFields(r.Context())
	if err != nil {
		s.logger.Error("failed to get fields", "err", err)
		s.writeError(w, http.StatusInternalServerError, ErrCodeBackendError, "Failed to retrieve fields from backend")
		return
	}

	sc, err := s.config.GetSearchContext(req.ContextId, req.Inherits, req.Search, req.Variables)
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

func (s *Server) queryFieldsHandler(w http.ResponseWriter, r *http.Request) {
	var req QueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, ErrCodeInvalidSearch, "Invalid request body")
		return
	}
	s.processQueryFieldsRequest(w, r, &req)
}

func (s *Server) openapiHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/yaml")
	w.WriteHeader(http.StatusOK)
	w.Write(s.openapiSpec)
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
