package server

import (
	"fmt"
)

func (s *Server) validateQueryRequest(req *QueryRequest) error {
	if req.ContextId == "" {
		return fmt.Errorf("contextId is required")
	}

	if _, ok := s.config.Contexts[req.ContextId]; !ok {
		return fmt.Errorf("contextId '%s' not found in configuration", req.ContextId)
	}

	for _, inherit := range req.Inherits {
		if _, ok := s.config.Searches[inherit]; !ok {
			return fmt.Errorf("inherit '%s' not found in configuration", inherit)
		}
	}

	if req.Search.Size.Set && req.Search.Size.Value <= 0 {
		return fmt.Errorf("search.size must be greater than 0")
	}

	return nil
}
