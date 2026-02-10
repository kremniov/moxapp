// Package api provides the HTTP API server for metrics and configuration
package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"moxapp/internal/config"
)

// handleListAuthConfigs returns all auth configs
// GET /api/outgoing/auth-configs
func (s *Server) handleListAuthConfigs(w http.ResponseWriter, r *http.Request) {
	authConfigs := s.configManager.GetAuthConfigs()

	response := map[string]interface{}{
		"count":        len(authConfigs),
		"auth_configs": authConfigs,
	}
	writeJSON(w, response)
}

// handleGetAuthConfig returns a single auth config by name
// GET /api/outgoing/auth-configs/{name}
func (s *Server) handleGetAuthConfig(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(r.URL.Path, "/api/outgoing/auth-configs/")
	// Remove any additional path segments (e.g., /token, /status, /refresh)
	if idx := strings.Index(name, "/"); idx != -1 {
		name = name[:idx]
	}

	if name == "" {
		writeError(w, "auth config name is required", http.StatusBadRequest)
		return
	}

	authCfg, err := s.configManager.GetAuthConfig(name)
	if err != nil {
		writeError(w, err.Error(), http.StatusNotFound)
		return
	}

	writeJSON(w, authCfg)
}

// handleCreateAuthConfig creates a new auth config
// POST /api/outgoing/auth-configs
func (s *Server) handleCreateAuthConfig(w http.ResponseWriter, r *http.Request) {
	var authCfg config.AuthConfig
	if err := json.NewDecoder(r.Body).Decode(&authCfg); err != nil {
		writeError(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Validate auth config
	if errs := authCfg.Validate(); len(errs) > 0 {
		writeError(w, "validation failed: "+strings.Join(errs, "; "), http.StatusBadRequest)
		return
	}

	if err := s.configManager.AddAuthConfig(&authCfg); err != nil {
		if strings.Contains(err.Error(), "already exists") {
			writeError(w, err.Error(), http.StatusConflict)
		} else {
			writeError(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusCreated)
	writeJSON(w, map[string]interface{}{
		"status":      "success",
		"message":     "Auth config created successfully",
		"auth_config": authCfg,
	})
}

// handleUpdateAuthConfig updates an existing auth config
// PUT /api/outgoing/auth-configs/{name}
func (s *Server) handleUpdateAuthConfig(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(r.URL.Path, "/api/outgoing/auth-configs/")
	if name == "" {
		writeError(w, "auth config name is required", http.StatusBadRequest)
		return
	}

	var authCfg config.AuthConfig
	if err := json.NewDecoder(r.Body).Decode(&authCfg); err != nil {
		writeError(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Validate auth config
	if errs := authCfg.Validate(); len(errs) > 0 {
		writeError(w, "validation failed: "+strings.Join(errs, "; "), http.StatusBadRequest)
		return
	}

	if err := s.configManager.UpdateAuthConfig(name, &authCfg); err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeError(w, err.Error(), http.StatusNotFound)
		} else if strings.Contains(err.Error(), "already exists") {
			writeError(w, err.Error(), http.StatusConflict)
		} else {
			writeError(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	writeJSON(w, map[string]interface{}{
		"status":      "success",
		"message":     "Auth config updated successfully",
		"auth_config": authCfg,
	})
}

// handleDeleteAuthConfig deletes an auth config by name
// DELETE /api/outgoing/auth-configs/{name}
func (s *Server) handleDeleteAuthConfig(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(r.URL.Path, "/api/outgoing/auth-configs/")
	if name == "" {
		writeError(w, "auth config name is required", http.StatusBadRequest)
		return
	}

	if err := s.configManager.DeleteAuthConfig(name); err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeError(w, err.Error(), http.StatusNotFound)
		} else {
			writeError(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	writeJSON(w, map[string]interface{}{
		"status":  "success",
		"message": "Auth config deleted successfully",
	})
}

// handleSetAuthToken manually sets a token for an auth config
// POST /api/outgoing/auth-configs/{name}/token
func (s *Server) handleSetAuthToken(w http.ResponseWriter, r *http.Request) {
	name := extractAuthConfigName(r.URL.Path, "/token")
	if name == "" {
		writeError(w, "auth config name is required", http.StatusBadRequest)
		return
	}

	// Check if auth config exists
	_, err := s.configManager.GetAuthConfig(name)
	if err != nil {
		writeError(w, err.Error(), http.StatusNotFound)
		return
	}

	// Check if token manager is available
	if s.tokenManager == nil {
		writeError(w, "token manager not available", http.StatusServiceUnavailable)
		return
	}

	var req struct {
		Token     string `json:"token"`
		ExpiresIn int    `json:"expires_in"` // seconds
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	if req.Token == "" {
		writeError(w, "token is required", http.StatusBadRequest)
		return
	}

	expiresIn := time.Duration(req.ExpiresIn) * time.Second
	if err := s.tokenManager.SetToken(name, req.Token, expiresIn); err != nil {
		writeError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, map[string]interface{}{
		"status":  "success",
		"message": "Token set successfully",
	})
}

// handleRefreshAuthToken forces a token refresh for an auth config
// POST /api/outgoing/auth-configs/{name}/refresh
func (s *Server) handleRefreshAuthToken(w http.ResponseWriter, r *http.Request) {
	name := extractAuthConfigName(r.URL.Path, "/refresh")
	if name == "" {
		writeError(w, "auth config name is required", http.StatusBadRequest)
		return
	}

	// Check if auth config exists
	authCfg, err := s.configManager.GetAuthConfig(name)
	if err != nil {
		writeError(w, err.Error(), http.StatusNotFound)
		return
	}

	// Check if token manager is available
	if s.tokenManager == nil {
		writeError(w, "token manager not available", http.StatusServiceUnavailable)
		return
	}

	// Check if auth config has token endpoint
	if authCfg.TokenEndpoint == nil {
		writeError(w, "auth config does not have token endpoint configured", http.StatusBadRequest)
		return
	}

	// Force refresh
	if err := s.tokenManager.ForceRefresh(r.Context(), name); err != nil {
		writeError(w, "failed to refresh token: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Get the token to display (masked)
	token, err := s.tokenManager.GetToken(r.Context(), name)
	if err != nil {
		writeError(w, "token refreshed but unable to retrieve: "+err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, map[string]interface{}{
		"status":  "success",
		"message": "Token refreshed successfully",
		"token":   maskToken(token),
	})
}

// handleAuthTokenStatus returns the token status for an auth config
// GET /api/outgoing/auth-configs/{name}/status
func (s *Server) handleAuthTokenStatus(w http.ResponseWriter, r *http.Request) {
	name := extractAuthConfigName(r.URL.Path, "/status")
	if name == "" {
		writeError(w, "auth config name is required", http.StatusBadRequest)
		return
	}

	// Check if auth config exists
	_, err := s.configManager.GetAuthConfig(name)
	if err != nil {
		writeError(w, err.Error(), http.StatusNotFound)
		return
	}

	// Check if token manager is available
	if s.tokenManager == nil {
		writeError(w, "token manager not available", http.StatusServiceUnavailable)
		return
	}

	status := s.tokenManager.GetTokenStatus(name)
	writeJSON(w, status)
}

// handleAuthConfigs is a router for auth config CRUD operations
func (s *Server) handleAuthConfigs(w http.ResponseWriter, r *http.Request) {
	// Extract path after /api/outgoing/auth-configs
	path := strings.TrimPrefix(r.URL.Path, "/api/outgoing/auth-configs")

	// Check for token management endpoints
	if strings.Contains(path, "/token") {
		if r.Method == http.MethodPost {
			s.handleSetAuthToken(w, r)
		} else {
			writeError(w, "method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}

	if strings.Contains(path, "/refresh") {
		if r.Method == http.MethodPost {
			s.handleRefreshAuthToken(w, r)
		} else {
			writeError(w, "method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}

	if strings.Contains(path, "/status") {
		if r.Method == http.MethodGet {
			s.handleAuthTokenStatus(w, r)
		} else {
			writeError(w, "method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}

	// Check if it's a request for a specific auth config
	hasName := path != "" && path != "/"

	switch r.Method {
	case http.MethodGet:
		if hasName {
			s.handleGetAuthConfig(w, r)
		} else {
			s.handleListAuthConfigs(w, r)
		}
	case http.MethodPost:
		if hasName {
			writeError(w, "POST to specific auth config not allowed, use PUT to update", http.StatusMethodNotAllowed)
		} else {
			s.handleCreateAuthConfig(w, r)
		}
	case http.MethodPut:
		if hasName {
			s.handleUpdateAuthConfig(w, r)
		} else {
			writeError(w, "PUT requires auth config name in path", http.StatusBadRequest)
		}
	case http.MethodDelete:
		if hasName {
			s.handleDeleteAuthConfig(w, r)
		} else {
			writeError(w, "DELETE requires auth config name in path", http.StatusBadRequest)
		}
	default:
		writeError(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// extractAuthConfigName extracts the auth config name from the URL path
// removing the specified suffix (e.g., "/token", "/status", "/refresh")
func extractAuthConfigName(path, suffix string) string {
	name := strings.TrimPrefix(path, "/api/outgoing/auth-configs/")
	name = strings.TrimSuffix(name, suffix)
	return name
}

// maskToken masks a token for safe display (shows first 4 and last 4 characters)
func maskToken(token string) string {
	if len(token) <= 8 {
		return "****"
	}
	return token[:4] + "****" + token[len(token)-4:]
}
