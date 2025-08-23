package server

import (
	"encoding/json"
	"net/http"
)

// APIError is a standardized error response structure.
type APIError struct {
	Message string                 `json:"error"`
	Code    string                 `json:"code"`
	Details map[string]interface{} `json:"details,omitempty"`
}

const (
	ErrCodeContextNotFound = "CONTEXT_NOT_FOUND"
	ErrCodeInvalidSearch   = "INVALID_SEARCH"
	ErrCodeBackendError    = "BACKEND_ERROR"
	ErrCodeConfigError     = "CONFIG_ERROR"
	ErrCodeValidationError = "VALIDATION_ERROR"
)

// writeJSON writes a JSON response with a given status code.
func (s *Server) writeJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		s.logger.Error("failed to write json response", "err", err)
	}
}

// writeError writes a standardized APIError response.
func (s *Server) writeError(w http.ResponseWriter, statusCode int, code, message string) {
	s.writeJSON(w, statusCode, APIError{
		Code:    code,
		Message: message,
	})
}
