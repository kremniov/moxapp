// Package api provides the HTTP API server for metrics and configuration
package api

import (
	"io"
	"net/http"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"moxapp/internal/config"
)

// handleExportConfig returns the full in-memory config as YAML
func (s *Server) handleExportConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.configManager == nil {
		writeError(w, "configuration manager not available", http.StatusServiceUnavailable)
		return
	}

	cfg := s.configManager.GetConfig()
	data, err := yaml.Marshal(cfg)
	if err != nil {
		writeError(w, "failed to serialize config", http.StatusInternalServerError)
		return
	}

	filename := "moxapp-config-" + time.Now().Format("20060102-150405") + ".yaml"
	withAttachment(w, filename)
	setContentType(w, "application/x-yaml")
	_, _ = w.Write(data)
}

// handleImportConfig replaces the in-memory config with uploaded YAML
func (s *Server) handleImportConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.configManager == nil {
		writeError(w, "configuration manager not available", http.StatusServiceUnavailable)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, "failed to read request body", http.StatusBadRequest)
		return
	}

	if len(body) == 0 {
		writeError(w, "empty request body", http.StatusBadRequest)
		return
	}

	var newCfg config.Config
	if err := yaml.Unmarshal(body, &newCfg); err != nil {
		writeError(w, "invalid YAML", http.StatusBadRequest)
		return
	}

	// Validate before replacing
	manager := config.NewManager()
	if err := manager.ReplaceConfig(&newCfg); err != nil {
		writeError(w, err.Error(), http.StatusBadRequest)
		return
	}
	if errors := manager.Validate(); len(errors) > 0 {
		writeError(w, "validation failed: "+strings.Join(errors, "; "), http.StatusBadRequest)
		return
	}

	if err := s.configManager.ReplaceConfig(&newCfg); err != nil {
		writeError(w, err.Error(), http.StatusBadRequest)
		return
	}

	writeJSON(w, map[string]string{
		"status":  "success",
		"message": "config imported",
	})
}

func withAttachment(w http.ResponseWriter, filename string) {
	w.Header().Set("Content-Disposition", "attachment; filename=\""+filename+"\"")
}

func setContentType(w http.ResponseWriter, contentType string) {
	w.Header().Set("Content-Type", contentType)
}
