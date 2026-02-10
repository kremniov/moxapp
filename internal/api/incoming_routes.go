// Package api provides the HTTP API server for metrics and configuration
package api

import (
	"net/http"
	"strings"

	"moxapp/internal/config"
)

// --- Incoming Routes CRUD Handlers ---

// handleIncomingRoutesRoute routes incoming routes requests with manager check
func (s *Server) handleIncomingRoutesRoute(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/incoming/routes")

	// Handle special endpoints first
	if path == "/reload" {
		s.handleReloadIncomingRoutes(w, r)
		return
	}

	hasName := path != "" && path != "/"

	switch r.Method {
	case http.MethodGet:
		if hasName {
			s.handleGetIncomingRoute(w, r)
		} else {
			s.handleListIncomingRoutes(w, r)
		}
	case http.MethodPost:
		if !s.checkIncomingManager(w) {
			return
		}
		s.handleCreateIncomingRoute(w, r)
	case http.MethodPut:
		if !s.checkIncomingManager(w) {
			return
		}
		s.handleUpdateIncomingRoute(w, r)
	case http.MethodDelete:
		if !s.checkIncomingManager(w) {
			return
		}
		s.handleDeleteIncomingRoute(w, r)
	default:
		writeError(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// checkIncomingManager checks if the config manager is available
func (s *Server) checkIncomingManager(w http.ResponseWriter) bool {
	if s.configManager == nil {
		writeError(w, "configuration not available", http.StatusServiceUnavailable)
		return false
	}
	return true
}

// handleListIncomingRoutes lists all incoming routes
func (s *Server) handleListIncomingRoutes(w http.ResponseWriter, r *http.Request) {
	if s.configManager == nil {
		writeError(w, "configuration not available", http.StatusServiceUnavailable)
		return
	}

	routes := s.configManager.GetIncomingRoutes()
	cfg := s.configManager.GetConfig()

	response := map[string]interface{}{
		"enabled":    cfg.IncomingEnabled,
		"count":      len(routes),
		"routes":     routes,
		"sim_prefix": SimulatedRoutePrefix,
	}
	writeJSON(w, response)
}

// handleGetIncomingRoute gets a specific incoming route by name
func (s *Server) handleGetIncomingRoute(w http.ResponseWriter, r *http.Request) {
	if s.configManager == nil {
		writeError(w, "configuration not available", http.StatusServiceUnavailable)
		return
	}

	name := strings.TrimPrefix(r.URL.Path, "/api/incoming/routes/")
	if name == "" {
		writeError(w, "route name is required", http.StatusBadRequest)
		return
	}

	route, err := s.configManager.GetIncomingRoute(name)
	if err != nil {
		writeError(w, err.Error(), http.StatusNotFound)
		return
	}

	writeJSON(w, route)
}

// handleCreateIncomingRoute creates a new incoming route
func (s *Server) handleCreateIncomingRoute(w http.ResponseWriter, r *http.Request) {
	var req config.IncomingEndpointRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	route := req.ToIncomingEndpoint()

	if err := s.configManager.AddIncomingRoute(route); err != nil {
		writeError(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusCreated)
	writeJSON(w, map[string]interface{}{
		"message":  "incoming route created",
		"route":    route,
		"sim_path": SimulatedRoutePrefix + route.Path,
	})
}

// handleUpdateIncomingRoute updates an existing incoming route
func (s *Server) handleUpdateIncomingRoute(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(r.URL.Path, "/api/incoming/routes/")
	if name == "" {
		writeError(w, "route name is required", http.StatusBadRequest)
		return
	}

	var req config.IncomingEndpointRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	route := req.ToIncomingEndpoint()

	if err := s.configManager.UpdateIncomingRoute(name, route); err != nil {
		writeError(w, err.Error(), http.StatusNotFound)
		return
	}

	writeJSON(w, map[string]interface{}{
		"message":  "incoming route updated",
		"route":    route,
		"sim_path": SimulatedRoutePrefix + route.Path,
	})
}

// handleDeleteIncomingRoute deletes an incoming route
func (s *Server) handleDeleteIncomingRoute(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(r.URL.Path, "/api/incoming/routes/")
	if name == "" {
		writeError(w, "route name is required", http.StatusBadRequest)
		return
	}

	if err := s.configManager.DeleteIncomingRoute(name); err != nil {
		writeError(w, err.Error(), http.StatusNotFound)
		return
	}

	writeJSON(w, map[string]interface{}{
		"message": "incoming route deleted",
		"name":    name,
	})
}

// handleReloadIncomingRoutes reloads incoming routes from static config file
func (s *Server) handleReloadIncomingRoutes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if !s.checkIncomingManager(w) {
		return
	}

	// Reload the entire config file
	if err := s.configManager.LoadFromFile(s.configManager.GetConfigPath()); err != nil {
		writeError(w, "failed to reload config: "+err.Error(), http.StatusInternalServerError)
		return
	}

	routes := s.configManager.GetIncomingRoutes()
	cfg := s.configManager.GetConfig()
	writeJSON(w, map[string]interface{}{
		"message":                  "configuration reloaded",
		"config_path":              s.configManager.GetConfigPath(),
		"incoming_routes_count":    len(routes),
		"outgoing_endpoints_count": len(cfg.Endpoints),
	})
}

// --- Incoming Control Handlers ---

// handleIncomingControl handles enable/disable of incoming routes
func (s *Server) handleIncomingControl(w http.ResponseWriter, r *http.Request) {
	if !s.checkIncomingManager(w) {
		return
	}

	switch r.Method {
	case http.MethodGet:
		// Return current state
		cfg := s.configManager.GetConfig()
		writeJSON(w, map[string]interface{}{
			"enabled":        cfg.IncomingEnabled,
			"total_routes":   s.configManager.GetIncomingRouteCount(),
			"enabled_routes": s.configManager.GetEnabledIncomingRouteCount(),
		})

	case http.MethodPost:
		// Enable/disable incoming routes
		var req struct {
			Enabled bool `json:"enabled"`
		}
		if err := readJSON(r, &req); err != nil {
			writeError(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
			return
		}

		s.configManager.SetIncomingEnabled(req.Enabled)
		writeJSON(w, map[string]interface{}{
			"message": "incoming routes status updated",
			"enabled": req.Enabled,
		})

	default:
		writeError(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleIncomingRouteControl enables/disables a specific incoming route
func (s *Server) handleIncomingRouteControl(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if !s.checkIncomingManager(w) {
		return
	}

	var req struct {
		Name    string `json:"name"`
		Enabled bool   `json:"enabled"`
	}
	if err := readJSON(r, &req); err != nil {
		writeError(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		writeError(w, "route name is required", http.StatusBadRequest)
		return
	}

	if err := s.configManager.SetIncomingRouteEnabled(req.Name, req.Enabled); err != nil {
		writeError(w, err.Error(), http.StatusNotFound)
		return
	}

	writeJSON(w, map[string]interface{}{
		"message": "incoming route status updated",
		"name":    req.Name,
		"enabled": req.Enabled,
	})
}
