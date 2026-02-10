// Package api provides the HTTP API server for metrics and configuration
package api

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"
	"time"

	"moxapp/internal/client"
	"moxapp/internal/config"
	"moxapp/internal/metrics"
	"moxapp/internal/scheduler"
	"moxapp/internal/web"
)

// Server is the HTTP API server
type Server struct {
	server        *http.Server
	metrics       *metrics.Collector
	config        *config.Config  // Legacy - kept for compatibility
	configManager *config.Manager // Config manager with both outgoing and incoming routes
	scheduler     *scheduler.Scheduler
	tokenManager  *client.TokenManager // Token manager for auth configs

	// Incoming routes simulation metrics
	incomingMetrics *metrics.IncomingCollector
}

// NewServer creates a new API server (legacy - uses Config directly)
func NewServer(addr string, metricsCollector *metrics.Collector, cfg *config.Config) *Server {
	s := &Server{
		metrics: metricsCollector,
		config:  cfg,
	}

	mux := http.NewServeMux()
	s.setupRoutes(mux)

	// Wrap with middleware
	handler := corsMiddleware(jsonMiddleware(mux))

	s.server = &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return s
}

// NewServerWithManager creates a new API server with config manager
func NewServerWithManager(addr string, metricsCollector *metrics.Collector, configManager *config.Manager) *Server {
	s := &Server{
		metrics:       metricsCollector,
		configManager: configManager,
		config:        configManager.GetConfig(), // For legacy compatibility
	}

	mux := http.NewServeMux()
	s.setupRoutes(mux)

	// Wrap with middleware
	handler := corsMiddleware(jsonMiddleware(mux))

	s.server = &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return s
}

// SetScheduler sets the scheduler reference for health checks
func (s *Server) SetScheduler(sched *scheduler.Scheduler) {
	s.scheduler = sched
}

// SetConfigManager sets the config manager for dynamic endpoint management
func (s *Server) SetConfigManager(manager *config.Manager) {
	s.configManager = manager
}

// SetIncomingMetrics sets the incoming routes metrics collector
func (s *Server) SetIncomingMetrics(collector *metrics.IncomingCollector) {
	s.incomingMetrics = collector
}

// SetTokenManager sets the token manager for auth config operations
func (s *Server) SetTokenManager(tm *client.TokenManager) {
	s.tokenManager = tm
}

// setupRoutes configures the API routes
func (s *Server) setupRoutes(mux *http.ServeMux) {
	staticRegistered := s.staticFrontend(mux)

	// API Documentation
	mux.HandleFunc("/api/docs", s.handleDocsRoute)
	mux.HandleFunc("/api/docs/", s.handleDocsRoute)

	// Metrics endpoints - unified under /api/metrics
	mux.HandleFunc("/api/metrics", s.handleMetricsOverview)
	mux.HandleFunc("/api/metrics/reset", s.handleResetAllMetrics)
	mux.HandleFunc("/api/metrics/outgoing", s.handleGetMetrics)
	mux.HandleFunc("/api/metrics/outgoing/reset", s.handleResetMetrics)
	mux.HandleFunc("/api/metrics/incoming", s.handleGetIncomingMetrics)
	mux.HandleFunc("/api/metrics/incoming/reset", s.handleResetIncomingMetrics)

	// Outgoing traffic management - settings, endpoints, control
	mux.HandleFunc("/api/outgoing/settings", s.handleGetSettings)
	mux.HandleFunc("/api/outgoing/settings/multiplier", s.handleSetMultiplier)
	mux.HandleFunc("/api/outgoing/settings/concurrency", s.handleSetConcurrency)
	mux.HandleFunc("/api/outgoing/settings/log-requests", s.handleSetLogRequests)

	// Config import/export
	mux.HandleFunc("/api/config/export", s.handleExportConfig)
	mux.HandleFunc("/api/config/import", s.handleImportConfig)

	mux.HandleFunc("/api/outgoing/endpoints", s.handleEndpointsRoute)
	mux.HandleFunc("/api/outgoing/endpoints/", s.handleEndpointsRoute)
	mux.HandleFunc("/api/outgoing/endpoints/bulk", s.handleBulkEndpointsRoute)

	mux.HandleFunc("/api/outgoing/auth-configs", s.handleAuthConfigs)
	mux.HandleFunc("/api/outgoing/auth-configs/", s.handleAuthConfigs)

	mux.HandleFunc("/api/outgoing/control", s.handleControl)
	mux.HandleFunc("/api/outgoing/control/endpoint", s.handleEndpointEnable)
	mux.HandleFunc("/api/outgoing/control/endpoints/bulk", s.handleBulkEndpointEnable)
	mux.HandleFunc("/api/outgoing/control/endpoints/all", s.handleEnableAll)

	// Incoming routes management API
	mux.HandleFunc("/api/incoming/routes", s.handleIncomingRoutesRoute)
	mux.HandleFunc("/api/incoming/routes/", s.handleIncomingRoutesRoute)
	mux.HandleFunc("/api/incoming/control", s.handleIncomingControl)
	mux.HandleFunc("/api/incoming/control/route", s.handleIncomingRouteControl)

	// Simulated routes endpoint - handles /sim/*
	mux.HandleFunc(SimulatedRoutePrefix+"/", s.handleSimulatedRoute)
	mux.HandleFunc(SimulatedRoutePrefix, s.handleSimulatedRouteInfo)

	// Health check
	mux.HandleFunc("/health", s.handleHealth)

	// Root handler - API info (only when frontend is not embedded)
	if !staticRegistered {
		mux.HandleFunc("/", s.handleRoot)
	}
}

func (s *Server) staticFrontend(mux *http.ServeMux) bool {
	fsys, err := web.FS()
	if err != nil {
		return false
	}

	fsysHTTP := http.FS(fsys)
	fileServer := http.FileServer(fsysHTTP)

	mux.Handle("/assets/", http.StripPrefix("/", fileServer))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			if strings.HasPrefix(r.URL.Path, "/api/") || r.URL.Path == "/health" || strings.HasPrefix(r.URL.Path, "/sim/") {
				http.NotFound(w, r)
				return
			}
			if fileExists(fsysHTTP, strings.TrimPrefix(r.URL.Path, "/")) {
				fileServer.ServeHTTP(w, r)
				return
			}
		}

		indexFile, err := fsys.Open("index.html")
		if err != nil {
			http.NotFound(w, r)
			return
		}
		defer indexFile.Close()

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = io.Copy(w, indexFile)
	})

	return true
}

func fileExists(fsys http.FileSystem, name string) bool {
	if name == "" || name == "." {
		return false
	}
	file, err := fsys.Open(path.Clean(name))
	if err != nil {
		return false
	}
	file.Close()
	return true
}

// handleRoot provides API information
func (s *Server) handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	info := map[string]interface{}{
		"app":     "moxapp",
		"version": "1.0.0",
		"docs": map[string]string{
			"swagger": "/api/docs/swagger",
			"redoc":   "/api/docs/redoc",
			"openapi": "/api/docs/openapi.yaml",
		},
		"endpoints": map[string]string{
			// Documentation
			"GET /api/docs/swagger":      "Swagger UI - Interactive API documentation",
			"GET /api/docs/redoc":        "ReDoc - Alternative API documentation",
			"GET /api/docs/openapi.yaml": "OpenAPI specification (YAML)",

			// Health
			"GET /health": "Health check",

			// Metrics - unified under /api/metrics
			"GET /api/metrics":                 "Get metrics (summary + snapshots)",
			"POST /api/metrics/reset":          "Reset all metrics (outgoing and incoming)",
			"GET /api/metrics/outgoing":        "Get outgoing traffic metrics",
			"POST /api/metrics/outgoing/reset": "Reset outgoing metrics",
			"GET /api/metrics/incoming":        "Get incoming traffic metrics",
			"POST /api/metrics/incoming/reset": "Reset incoming metrics",

			// Outgoing - settings, endpoints, control
			"GET /api/outgoing/settings":                     "Get all outgoing settings",
			"GET /api/outgoing/settings/multiplier":          "Get global multiplier",
			"POST /api/outgoing/settings/multiplier":         "Set global multiplier",
			"GET /api/outgoing/settings/concurrency":         "Get concurrent requests limit",
			"POST /api/outgoing/settings/concurrency":        "Set concurrent requests limit",
			"GET /api/outgoing/settings/log-requests":        "Get log all requests setting",
			"POST /api/outgoing/settings/log-requests":       "Set log all requests setting",
			"GET /api/outgoing/endpoints":                    "List all outgoing endpoints",
			"GET /api/outgoing/endpoints/{name}":             "Get outgoing endpoint by name",
			"POST /api/outgoing/endpoints":                   "Create new outgoing endpoint",
			"PUT /api/outgoing/endpoints/{name}":             "Update outgoing endpoint",
			"DELETE /api/outgoing/endpoints/{name}":          "Delete outgoing endpoint",
			"POST /api/outgoing/endpoints/bulk":              "Bulk create outgoing endpoints",
			"DELETE /api/outgoing/endpoints/bulk":            "Bulk delete outgoing endpoints",
			"GET /api/outgoing/auth-configs":                 "List all auth configs",
			"GET /api/outgoing/auth-configs/{name}":          "Get auth config by name",
			"POST /api/outgoing/auth-configs":                "Create new auth config",
			"PUT /api/outgoing/auth-configs/{name}":          "Update auth config",
			"DELETE /api/outgoing/auth-configs/{name}":       "Delete auth config",
			"POST /api/outgoing/auth-configs/{name}/token":   "Manually set token for auth config",
			"POST /api/outgoing/auth-configs/{name}/refresh": "Force refresh token for auth config",
			"GET /api/outgoing/auth-configs/{name}/status":   "Get token status for auth config",
			"GET /api/outgoing/control":                      "Get scheduler control status",
			"POST /api/outgoing/control":                     "Control scheduler (pause, resume, emergency_stop)",
			"POST /api/outgoing/control/endpoint":            "Enable/disable specific outgoing endpoint",
			"POST /api/outgoing/control/endpoints/bulk":      "Enable/disable multiple outgoing endpoints",
			"POST /api/outgoing/control/endpoints/all":       "Enable/disable all outgoing endpoints",
			"GET /api/config/export":                         "Export full config as YAML",
			"POST /api/config/import":                        "Import full config from YAML",

			// Incoming Routes CRUD
			"GET /api/incoming/routes":           "List all incoming routes",
			"GET /api/incoming/routes/{name}":    "Get incoming route by name",
			"POST /api/incoming/routes":          "Create new incoming route",
			"PUT /api/incoming/routes/{name}":    "Update incoming route",
			"DELETE /api/incoming/routes/{name}": "Delete incoming route",
			"POST /api/incoming/routes/reload":   "Reload incoming routes from static config",

			// Incoming Routes Control
			"GET /api/incoming/control":        "Get incoming routes status",
			"POST /api/incoming/control":       "Enable/disable all incoming routes",
			"POST /api/incoming/control/route": "Enable/disable specific incoming route",

			// Simulated Routes
			"* /sim/*": "Simulated incoming routes (responds based on configured patterns)",
		},
	}
	writeJSON(w, info)
}

// Start starts the API server
func (s *Server) Start() error {
	return s.server.ListenAndServe()
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}

// Addr returns the server address
func (s *Server) Addr() string {
	return s.server.Addr
}

// GetListenAddr returns a formatted listen address string
func (s *Server) GetListenAddr() string {
	return fmt.Sprintf("http://localhost%s", s.server.Addr)
}

// getConfigForHandlers returns the config from manager if available, otherwise legacy config
func (s *Server) getConfigForHandlers() *config.Config {
	if s.configManager != nil {
		return s.configManager.GetConfig()
	}
	return s.config
}

// handleEndpoints router helper - checks if config manager is available
func (s *Server) checkConfigManager(w http.ResponseWriter) bool {
	if s.configManager == nil {
		writeError(w, "configuration manager not available - endpoint CRUD operations disabled", http.StatusServiceUnavailable)
		return false
	}
	return true
}

// handleEndpointsRoute routes endpoint requests, checking config manager for write operations
func (s *Server) handleEndpointsRoute(w http.ResponseWriter, r *http.Request) {
	// Check if it's a request for a specific endpoint
	path := strings.TrimPrefix(r.URL.Path, "/api/outgoing/endpoints")
	hasName := path != "" && path != "/"

	// For GET requests, we can work without config manager (fallback to legacy)
	if r.Method == http.MethodGet {
		if hasName {
			// Get specific endpoint
			name := strings.TrimPrefix(path, "/")
			if s.configManager != nil {
				endpoint, err := s.configManager.GetEndpoint(name)
				if err != nil {
					writeError(w, err.Error(), http.StatusNotFound)
					return
				}
				writeJSON(w, endpoint)
			} else {
				// Fallback to legacy config
				cfg := s.getConfigForHandlers()
				for _, ep := range cfg.Endpoints {
					if ep.Name == name {
						writeJSON(w, ep)
						return
					}
				}
				writeError(w, "endpoint not found: "+name, http.StatusNotFound)
			}
		} else {
			// List all endpoints
			cfg := s.getConfigForHandlers()
			response := map[string]interface{}{
				"count":     len(cfg.Endpoints),
				"endpoints": cfg.Endpoints,
			}
			writeJSON(w, response)
		}
		return
	}

	// For write operations, require config manager
	if !s.checkConfigManager(w) {
		return
	}
	s.handleEndpoints(w, r)
}

// handleBulkEndpointsRoute routes bulk endpoint requests with config manager check
func (s *Server) handleBulkEndpointsRoute(w http.ResponseWriter, r *http.Request) {
	if !s.checkConfigManager(w) {
		return
	}
	s.handleBulkEndpoints(w, r)
}
