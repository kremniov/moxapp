// Package api provides the HTTP API server for metrics and configuration
package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"moxapp/internal/config"
)

// handleListEndpoints returns all endpoints
// GET /api/endpoints
func (s *Server) handleListEndpoints(w http.ResponseWriter, r *http.Request) {
	endpoints := s.configManager.GetEndpoints()

	response := map[string]interface{}{
		"count":     len(endpoints),
		"endpoints": endpoints,
	}
	writeJSON(w, response)
}

// handleGetEndpoint returns a single endpoint by name
// GET /api/endpoints/{name}
func (s *Server) handleGetEndpoint(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(r.URL.Path, "/api/outgoing/endpoints/")
	if name == "" {
		writeError(w, "endpoint name is required", http.StatusBadRequest)
		return
	}

	endpoint, err := s.configManager.GetEndpoint(name)
	if err != nil {
		writeError(w, err.Error(), http.StatusNotFound)
		return
	}

	writeJSON(w, endpoint)
}

// handleCreateEndpoint creates a new endpoint
// POST /api/endpoints
func (s *Server) handleCreateEndpoint(w http.ResponseWriter, r *http.Request) {
	var req config.EndpointRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	endpoint := req.ToEndpoint()

	if err := s.configManager.AddEndpoint(endpoint); err != nil {
		if strings.Contains(err.Error(), "already exists") {
			writeError(w, err.Error(), http.StatusConflict)
		} else if strings.Contains(err.Error(), "validation failed") {
			writeError(w, err.Error(), http.StatusBadRequest)
		} else {
			writeError(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusCreated)
	writeJSON(w, map[string]interface{}{
		"status":   "success",
		"message":  "Endpoint created successfully",
		"endpoint": endpoint,
	})
}

// handleUpdateEndpoint updates an existing endpoint
// PUT /api/endpoints/{name}
func (s *Server) handleUpdateEndpoint(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(r.URL.Path, "/api/outgoing/endpoints/")
	if name == "" {
		writeError(w, "endpoint name is required", http.StatusBadRequest)
		return
	}

	var req config.EndpointRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	endpoint := req.ToEndpoint()

	if err := s.configManager.UpdateEndpoint(name, endpoint); err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeError(w, err.Error(), http.StatusNotFound)
		} else if strings.Contains(err.Error(), "already exists") {
			writeError(w, err.Error(), http.StatusConflict)
		} else if strings.Contains(err.Error(), "validation failed") {
			writeError(w, err.Error(), http.StatusBadRequest)
		} else {
			writeError(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	writeJSON(w, map[string]interface{}{
		"status":   "success",
		"message":  "Endpoint updated successfully",
		"endpoint": endpoint,
	})
}

// handleDeleteEndpoint deletes an endpoint by name
// DELETE /api/endpoints/{name}
func (s *Server) handleDeleteEndpoint(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(r.URL.Path, "/api/outgoing/endpoints/")
	if name == "" {
		writeError(w, "endpoint name is required", http.StatusBadRequest)
		return
	}

	if err := s.configManager.DeleteEndpoint(name); err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeError(w, err.Error(), http.StatusNotFound)
		} else {
			writeError(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	writeJSON(w, map[string]interface{}{
		"status":  "success",
		"message": "Endpoint deleted successfully",
	})
}

// handleEndpoints is a router for endpoint CRUD operations
func (s *Server) handleEndpoints(w http.ResponseWriter, r *http.Request) {
	// Check if it's a request for a specific endpoint
	path := strings.TrimPrefix(r.URL.Path, "/api/outgoing/endpoints")
	hasName := path != "" && path != "/"

	switch r.Method {
	case http.MethodGet:
		if hasName {
			s.handleGetEndpoint(w, r)
		} else {
			s.handleListEndpoints(w, r)
		}
	case http.MethodPost:
		if hasName {
			writeError(w, "POST to specific endpoint not allowed, use PUT to update", http.StatusMethodNotAllowed)
		} else {
			s.handleCreateEndpoint(w, r)
		}
	case http.MethodPut:
		if hasName {
			s.handleUpdateEndpoint(w, r)
		} else {
			writeError(w, "PUT requires endpoint name in path", http.StatusBadRequest)
		}
	case http.MethodDelete:
		if hasName {
			s.handleDeleteEndpoint(w, r)
		} else {
			writeError(w, "DELETE requires endpoint name in path", http.StatusBadRequest)
		}
	default:
		writeError(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleBulkEndpoints handles bulk operations on endpoints
// POST /api/endpoints/bulk - create multiple endpoints
// DELETE /api/endpoints/bulk - delete multiple endpoints
func (s *Server) handleBulkEndpoints(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		s.handleBulkCreateEndpoints(w, r)
	case http.MethodDelete:
		s.handleBulkDeleteEndpoints(w, r)
	default:
		writeError(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleBulkCreateEndpoints creates multiple endpoints at once
func (s *Server) handleBulkCreateEndpoints(w http.ResponseWriter, r *http.Request) {
	var requests []config.EndpointRequest
	if err := json.NewDecoder(r.Body).Decode(&requests); err != nil {
		writeError(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	var created []string
	var errors []string

	for _, req := range requests {
		endpoint := req.ToEndpoint()
		if err := s.configManager.AddEndpoint(endpoint); err != nil {
			errors = append(errors, endpoint.Name+": "+err.Error())
		} else {
			created = append(created, endpoint.Name)
		}
	}

	status := http.StatusOK
	if len(created) == 0 && len(errors) > 0 {
		status = http.StatusBadRequest
	} else if len(errors) > 0 {
		status = http.StatusPartialContent
	}

	w.WriteHeader(status)
	writeJSON(w, map[string]interface{}{
		"created": created,
		"errors":  errors,
		"summary": map[string]int{
			"total_requested": len(requests),
			"created":         len(created),
			"failed":          len(errors),
		},
	})
}

// handleBulkDeleteEndpoints deletes multiple endpoints by name
func (s *Server) handleBulkDeleteEndpoints(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Names []string `json:"names"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	var deleted []string
	var errors []string

	for _, name := range req.Names {
		if err := s.configManager.DeleteEndpoint(name); err != nil {
			errors = append(errors, name+": "+err.Error())
		} else {
			deleted = append(deleted, name)
		}
	}

	status := http.StatusOK
	if len(deleted) == 0 && len(errors) > 0 {
		status = http.StatusBadRequest
	} else if len(errors) > 0 {
		status = http.StatusPartialContent
	}

	w.WriteHeader(status)
	writeJSON(w, map[string]interface{}{
		"deleted": deleted,
		"errors":  errors,
		"summary": map[string]int{
			"total_requested": len(req.Names),
			"deleted":         len(deleted),
			"failed":          len(errors),
		},
	})
}
