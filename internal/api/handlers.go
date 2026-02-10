// Package api provides the HTTP API server for metrics and configuration
package api

import (
	"net/http"
	"runtime"
	"time"

	"moxapp/internal/scheduler"
)

// --- Metrics Handlers ---

// handleMetricsOverview returns a merged metrics response (summary + snapshots)
func (s *Server) handleMetricsOverview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	outgoingSnapshot := s.metrics.Snapshot()

	errorSummary := map[string]int64{
		"timeout":    0,
		"dns":        0,
		"connection": 0,
		"http":       0,
	}

	for _, ep := range outgoingSnapshot.Endpoints {
		errorSummary["timeout"] += ep.TimeoutErrors
		errorSummary["dns"] += ep.DNSErrors
		errorSummary["connection"] += ep.ConnectionErrors
		errorSummary["http"] += ep.HTTPErrors
	}

	response := map[string]interface{}{
		"timestamp":      time.Now().Format(time.RFC3339),
		"uptime_seconds": outgoingSnapshot.UptimeSeconds,
		"outgoing": map[string]interface{}{
			"total_requests":   outgoingSnapshot.TotalRequests,
			"total_failures":   outgoingSnapshot.TotalFailures,
			"requests_per_sec": outgoingSnapshot.RequestsPerSecond,
			"success_rate":     outgoingSnapshot.SuccessRate,
			"endpoint_count":   len(outgoingSnapshot.Endpoints),
			"domain_count":     len(outgoingSnapshot.DNSStatsByDomain),
			"error_summary":    errorSummary,
		},
		"outgoing_snapshot": outgoingSnapshot,
	}

	if s.incomingMetrics != nil {
		incomingSnapshot := s.incomingMetrics.Snapshot()
		response["incoming"] = map[string]interface{}{
			"available":        true,
			"total_requests":   incomingSnapshot.TotalRequests,
			"requests_per_sec": incomingSnapshot.RequestsPerSecond,
			"active_routes":    len(incomingSnapshot.Routes),
		}
		response["incoming_snapshot"] = incomingSnapshot
	} else {
		response["incoming"] = map[string]interface{}{
			"available": false,
		}
		response["incoming_snapshot"] = nil
	}

	writeJSON(w, response)
}

// handleResetAllMetrics resets both outgoing and incoming metrics
func (s *Server) handleResetAllMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Reset outgoing metrics
	s.metrics.Reset()

	// Reset incoming metrics if available
	if s.incomingMetrics != nil {
		s.incomingMetrics.Reset()
	}

	response := map[string]string{
		"status":  "success",
		"message": "All metrics have been reset (outgoing and incoming)",
	}
	writeJSON(w, response)
}

// handleGetMetrics returns current outgoing metrics
func (s *Server) handleGetMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	snapshot := s.metrics.Snapshot()
	writeJSON(w, snapshot)
}

// handleResetMetrics resets outgoing metrics
func (s *Server) handleResetMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	s.metrics.Reset()

	response := map[string]string{
		"status":  "success",
		"message": "Outgoing metrics have been reset",
	}
	writeJSON(w, response)
}

// handleGetIncomingMetrics returns metrics for incoming routes
func (s *Server) handleGetIncomingMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.incomingMetrics == nil {
		writeError(w, "incoming metrics not available", http.StatusServiceUnavailable)
		return
	}

	snapshot := s.incomingMetrics.Snapshot()
	writeJSON(w, snapshot)
}

// handleResetIncomingMetrics resets incoming route metrics
func (s *Server) handleResetIncomingMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.incomingMetrics == nil {
		writeError(w, "incoming metrics not available", http.StatusServiceUnavailable)
		return
	}

	s.incomingMetrics.Reset()

	response := map[string]string{
		"status":  "success",
		"message": "Incoming metrics have been reset",
	}
	writeJSON(w, response)
}

// handleHealth returns health check information
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	schedulerStats := scheduler.SchedulerStats{}
	if s.scheduler != nil {
		schedulerStats = s.scheduler.GetStats()
	}

	cfg := s.getConfigForHandlers()

	// Count enabled endpoints
	enabledEndpoints := 0
	for _, ep := range cfg.Endpoints {
		if ep.Enabled {
			enabledEndpoints++
		}
	}

	health := map[string]interface{}{
		"status":             "healthy",
		"app":                "moxapp",
		"version":            "1.0.0",
		"timestamp":          time.Now().Format(time.RFC3339),
		"go_version":         runtime.Version(),
		"goroutines":         runtime.NumGoroutine(),
		"memory_alloc_mb":    float64(memStats.Alloc) / 1024 / 1024,
		"memory_sys_mb":      float64(memStats.Sys) / 1024 / 1024,
		"total_requests":     s.metrics.GetTotalRequests(),
		"requests_per_sec":   s.metrics.GetRequestsPerSecond(),
		"success_rate":       s.metrics.GetSuccessRate(),
		"requests_in_flight": schedulerStats.RequestsInFlight,
		"requests_skipped":   schedulerStats.RequestsSkipped,
		"scheduler_running":  s.scheduler != nil && s.scheduler.IsRunning(),
		"scheduler_paused":   schedulerStats.Paused,
		"global_enabled":     schedulerStats.GlobalEnabled,
		"endpoint_count":     len(cfg.Endpoints),
		"enabled_endpoints":  enabledEndpoints,
		"config_manager":     s.configManager != nil,
	}

	// Add incoming routes info
	if s.configManager != nil {
		health["incoming_routes_enabled"] = s.configManager.IsIncomingEnabled()
		health["incoming_routes_count"] = s.configManager.GetIncomingRouteCount()
		health["incoming_routes_active"] = s.configManager.GetEnabledIncomingRouteCount()
	}
	if s.incomingMetrics != nil {
		health["incoming_total_requests"] = s.incomingMetrics.GetTotalRequests()
		health["incoming_requests_per_sec"] = s.incomingMetrics.GetRequestsPerSecond()
	}

	writeJSON(w, health)
}

// --- Control Handlers ---

// handleControl routes control requests
func (s *Server) handleControl(w http.ResponseWriter, r *http.Request) {
	if s.scheduler == nil {
		writeError(w, "scheduler not available", http.StatusServiceUnavailable)
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.handleGetControlStatus(w, r)
	case http.MethodPost:
		s.handleControlAction(w, r)
	default:
		writeError(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleGetControlStatus returns current scheduler control status
func (s *Server) handleGetControlStatus(w http.ResponseWriter, r *http.Request) {
	stats := s.scheduler.GetStats()

	status := map[string]interface{}{
		"global_enabled":     stats.GlobalEnabled,
		"paused":             stats.Paused,
		"scheduler_running":  s.scheduler.IsRunning(),
		"requests_scheduled": stats.RequestsScheduled,
		"requests_in_flight": stats.RequestsInFlight,
		"requests_skipped":   stats.RequestsSkipped,
		"total_endpoints":    stats.ActiveEndpoints,
		"enabled_endpoints":  stats.EnabledEndpoints,
		"disabled_endpoints": stats.ActiveEndpoints - stats.EnabledEndpoints,
	}

	writeJSON(w, status)
}

// handleControlAction handles POST requests to /api/control
func (s *Server) handleControlAction(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Action string `json:"action"`
	}

	if err := readJSON(r, &req); err != nil {
		writeError(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	switch req.Action {
	case "pause":
		s.scheduler.Pause()
		writeJSON(w, map[string]interface{}{
			"status":  "success",
			"message": "Scheduler paused - no new requests will be scheduled",
			"paused":  true,
		})

	case "resume":
		s.scheduler.Resume()
		writeJSON(w, map[string]interface{}{
			"status":  "success",
			"message": "Scheduler resumed - requests are being scheduled",
			"paused":  false,
		})

	case "emergency_stop":
		s.scheduler.EmergencyStop()
		writeJSON(w, map[string]interface{}{
			"status":  "success",
			"message": "EMERGENCY STOP - All scheduling stopped and in-flight requests cancelled",
			"paused":  true,
		})

	default:
		writeError(w, "unknown action: "+req.Action+". Valid actions: pause, resume, emergency_stop", http.StatusBadRequest)
	}
}

// handleEndpointEnable handles enabling/disabling specific endpoints
func (s *Server) handleEndpointEnable(w http.ResponseWriter, r *http.Request) {
	if s.configManager == nil {
		writeError(w, "configuration manager not available", http.StatusServiceUnavailable)
		return
	}

	if r.Method != http.MethodPost {
		writeError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Name    string `json:"name"`
		Enabled bool   `json:"enabled"`
	}

	if err := readJSON(r, &req); err != nil {
		writeError(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		writeError(w, "endpoint name is required", http.StatusBadRequest)
		return
	}

	if err := s.configManager.SetEndpointEnabled(req.Name, req.Enabled); err != nil {
		writeError(w, err.Error(), http.StatusNotFound)
		return
	}

	action := "disabled"
	if req.Enabled {
		action = "enabled"
	}

	writeJSON(w, map[string]interface{}{
		"status":   "success",
		"message":  "Endpoint " + req.Name + " " + action,
		"endpoint": req.Name,
		"enabled":  req.Enabled,
	})
}

// handleBulkEndpointEnable handles enabling/disabling multiple endpoints at once
func (s *Server) handleBulkEndpointEnable(w http.ResponseWriter, r *http.Request) {
	if s.configManager == nil {
		writeError(w, "configuration manager not available", http.StatusServiceUnavailable)
		return
	}

	if r.Method != http.MethodPost {
		writeError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Names   []string `json:"names"`
		Enabled bool     `json:"enabled"`
	}

	if err := readJSON(r, &req); err != nil {
		writeError(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	var updated []string
	var errors []string

	for _, name := range req.Names {
		if err := s.configManager.SetEndpointEnabled(name, req.Enabled); err != nil {
			errors = append(errors, name+": "+err.Error())
		} else {
			updated = append(updated, name)
		}
	}

	action := "disabled"
	if req.Enabled {
		action = "enabled"
	}

	writeJSON(w, map[string]interface{}{
		"status":  "success",
		"message": "Bulk " + action + " completed",
		"updated": updated,
		"errors":  errors,
		"summary": map[string]int{
			"total_requested": len(req.Names),
			"updated":         len(updated),
			"failed":          len(errors),
		},
	})
}

// handleEnableAll enables or disables all endpoints
func (s *Server) handleEnableAll(w http.ResponseWriter, r *http.Request) {
	if s.configManager == nil {
		writeError(w, "configuration manager not available", http.StatusServiceUnavailable)
		return
	}

	if r.Method != http.MethodPost {
		writeError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Enabled bool `json:"enabled"`
	}

	if err := readJSON(r, &req); err != nil {
		writeError(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	endpoints := s.configManager.GetEndpoints()
	var updated int

	for _, ep := range endpoints {
		if err := s.configManager.SetEndpointEnabled(ep.Name, req.Enabled); err == nil {
			updated++
		}
	}

	action := "disabled"
	if req.Enabled {
		action = "enabled"
	}

	writeJSON(w, map[string]interface{}{
		"status":  "success",
		"message": "All endpoints " + action,
		"updated": updated,
		"enabled": req.Enabled,
	})
}

// --- Settings Handlers ---

// handleGetSettings returns current runtime settings
func (s *Server) handleGetSettings(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	cfg := s.getConfigForHandlers()

	settings := map[string]interface{}{
		"global_multiplier":   cfg.GlobalMultiplier,
		"concurrent_requests": cfg.ConcurrentRequests,
		"log_all_requests":    cfg.LogAllRequests,
		"api_port":            cfg.APIPort,
		"enabled":             cfg.Enabled,
	}

	writeJSON(w, settings)
}

// handleSetMultiplier updates the global load multiplier
func (s *Server) handleSetMultiplier(w http.ResponseWriter, r *http.Request) {
	if s.configManager == nil {
		writeError(w, "configuration manager not available", http.StatusServiceUnavailable)
		return
	}

	switch r.Method {
	case http.MethodGet:
		cfg := s.getConfigForHandlers()
		writeJSON(w, map[string]interface{}{
			"global_multiplier": cfg.GlobalMultiplier,
		})

	case http.MethodPost, http.MethodPut:
		var req struct {
			Multiplier float64 `json:"multiplier"`
		}

		if err := readJSON(r, &req); err != nil {
			writeError(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
			return
		}

		if req.Multiplier < 0 {
			writeError(w, "multiplier must be non-negative", http.StatusBadRequest)
			return
		}

		oldMultiplier := s.configManager.GetConfig().GlobalMultiplier
		s.configManager.SetGlobalMultiplier(req.Multiplier)

		writeJSON(w, map[string]interface{}{
			"status":         "success",
			"message":        "Global multiplier updated",
			"old_multiplier": oldMultiplier,
			"new_multiplier": req.Multiplier,
		})

	default:
		writeError(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleSetConcurrency updates the concurrent requests limit
func (s *Server) handleSetConcurrency(w http.ResponseWriter, r *http.Request) {
	if s.configManager == nil {
		writeError(w, "configuration manager not available", http.StatusServiceUnavailable)
		return
	}

	switch r.Method {
	case http.MethodGet:
		cfg := s.getConfigForHandlers()
		writeJSON(w, map[string]interface{}{
			"concurrent_requests": cfg.ConcurrentRequests,
		})

	case http.MethodPost, http.MethodPut:
		var req struct {
			Concurrent int `json:"concurrent"`
		}

		if err := readJSON(r, &req); err != nil {
			writeError(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
			return
		}

		if req.Concurrent <= 0 {
			writeError(w, "concurrent must be positive", http.StatusBadRequest)
			return
		}

		oldConcurrent := s.configManager.GetConfig().ConcurrentRequests
		s.configManager.SetConcurrentRequests(req.Concurrent)

		writeJSON(w, map[string]interface{}{
			"status":         "success",
			"message":        "Concurrent requests limit updated",
			"old_concurrent": oldConcurrent,
			"new_concurrent": req.Concurrent,
		})

	default:
		writeError(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleSetLogRequests updates the log all requests setting
func (s *Server) handleSetLogRequests(w http.ResponseWriter, r *http.Request) {
	if s.configManager == nil {
		writeError(w, "configuration manager not available", http.StatusServiceUnavailable)
		return
	}

	switch r.Method {
	case http.MethodGet:
		cfg := s.getConfigForHandlers()
		writeJSON(w, map[string]interface{}{
			"log_all_requests": cfg.LogAllRequests,
		})

	case http.MethodPost, http.MethodPut:
		var req struct {
			LogRequests bool `json:"log_requests"`
		}

		if err := readJSON(r, &req); err != nil {
			writeError(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
			return
		}

		oldValue := s.configManager.GetConfig().LogAllRequests
		s.configManager.SetLogAllRequests(req.LogRequests)

		writeJSON(w, map[string]interface{}{
			"status":           "success",
			"message":          "Log all requests setting updated",
			"old_log_requests": oldValue,
			"new_log_requests": req.LogRequests,
		})

	default:
		writeError(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}
