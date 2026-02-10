// Package api provides the HTTP API server for metrics and configuration
package api

import (
	"encoding/json"
	"io"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"moxapp/internal/config"
)

// SimulatedRoutePrefix is the prefix for all simulated incoming routes
const SimulatedRoutePrefix = "/sim"

// EchoResponse represents the response body for simulated routes
type EchoResponse struct {
	Timestamp    string       `json:"timestamp"`
	MatchedRoute MatchedRoute `json:"matched_route"`
	Request      RequestEcho  `json:"request"`
	Response     ResponseInfo `json:"response"`
}

// MatchedRoute contains information about the matched route configuration
type MatchedRoute struct {
	Name   string `json:"name"`
	Path   string `json:"path"`
	Method string `json:"method"`
}

// RequestEcho contains the echoed request details
type RequestEcho struct {
	Method      string              `json:"method"`
	Path        string              `json:"path"`
	PathSuffix  string              `json:"path_suffix,omitempty"`
	Headers     map[string][]string `json:"headers"`
	QueryParams map[string][]string `json:"query_params,omitempty"`
	Body        interface{}         `json:"body,omitempty"`
	RemoteAddr  string              `json:"remote_addr"`
}

// ResponseInfo contains information about the simulated response
type ResponseInfo struct {
	Status           int     `json:"status"`
	SimulatedDelayMs float64 `json:"simulated_delay_ms"`
}

// handleSimulatedRoute handles all requests to /sim/* and routes them to configured incoming routes
func (s *Server) handleSimulatedRoute(w http.ResponseWriter, r *http.Request) {
	// Extract the path after /sim prefix
	path := strings.TrimPrefix(r.URL.Path, SimulatedRoutePrefix)
	if path == "" {
		path = "/"
	}

	// Check if config manager is available
	if s.configManager == nil {
		writeError(w, "configuration not available", http.StatusServiceUnavailable)
		return
	}

	// Match the route
	route, pathSuffix, matched := s.configManager.MatchIncomingRoute(path, r.Method)
	if !matched {
		writeError(w, "no matching route found for path: "+path, http.StatusNotFound)
		return
	}

	// Select response based on weighted probability
	selectedResponse := selectWeightedResponse(route.Responses)

	// Calculate simulated delay
	delayMs := randomDuration(selectedResponse.MinResponseMs, selectedResponse.MaxResponseMs)

	// Sleep to simulate response time
	if delayMs > 0 {
		time.Sleep(time.Duration(delayMs) * time.Millisecond)
	}

	// Record metrics
	if s.incomingMetrics != nil {
		s.incomingMetrics.Record(route.Name, route.Path, selectedResponse.StatusCode, float64(delayMs))
	}

	// Build echo response
	echoResponse := buildEchoResponse(r, route, path, pathSuffix, selectedResponse.StatusCode, float64(delayMs))

	// Write response
	w.WriteHeader(selectedResponse.StatusCode)
	writeJSON(w, echoResponse)
}

// selectWeightedResponse selects a response based on weighted probability (share)
func selectWeightedResponse(responses []config.IncomingResponseConfig) config.IncomingResponseConfig {
	if len(responses) == 0 {
		// Fallback - should not happen if validation is working
		return config.IncomingResponseConfig{
			StatusCode:    500,
			Share:         1.0,
			MinResponseMs: 0,
			MaxResponseMs: 0,
		}
	}

	if len(responses) == 1 {
		return responses[0]
	}

	// Generate random number between 0 and 1
	randVal := rand.Float64()

	// Cumulative probability selection
	cumulative := 0.0
	for _, resp := range responses {
		cumulative += resp.Share
		if randVal < cumulative {
			return resp
		}
	}

	// Fallback to last response (handles floating point rounding)
	return responses[len(responses)-1]
}

// randomDuration returns a random duration between min and max milliseconds
func randomDuration(minMs, maxMs int) int {
	if minMs >= maxMs {
		return minMs
	}
	return minMs + rand.Intn(maxMs-minMs+1)
}

// buildEchoResponse constructs the echo response with full request details
func buildEchoResponse(r *http.Request, route *config.IncomingEndpoint, path, pathSuffix string, statusCode int, delayMs float64) EchoResponse {
	// Parse request body if present
	var body interface{}
	if r.Body != nil && r.ContentLength > 0 {
		bodyBytes, err := io.ReadAll(r.Body)
		if err == nil && len(bodyBytes) > 0 {
			// Try to parse as JSON
			var jsonBody interface{}
			if err := json.Unmarshal(bodyBytes, &jsonBody); err == nil {
				body = jsonBody
			} else {
				// Return as string if not valid JSON
				body = string(bodyBytes)
			}
		}
	}

	// Copy headers (excluding some sensitive ones)
	headers := make(map[string][]string)
	for key, values := range r.Header {
		// Optionally filter sensitive headers
		lowerKey := strings.ToLower(key)
		if lowerKey == "authorization" {
			headers[key] = []string{"[REDACTED]"}
		} else {
			headers[key] = values
		}
	}

	// Copy query parameters
	var queryParams map[string][]string
	if len(r.URL.Query()) > 0 {
		queryParams = r.URL.Query()
	}

	return EchoResponse{
		Timestamp: time.Now().Format(time.RFC3339Nano),
		MatchedRoute: MatchedRoute{
			Name:   route.Name,
			Path:   route.Path,
			Method: route.Method,
		},
		Request: RequestEcho{
			Method:      r.Method,
			Path:        path,
			PathSuffix:  pathSuffix,
			Headers:     headers,
			QueryParams: queryParams,
			Body:        body,
			RemoteAddr:  r.RemoteAddr,
		},
		Response: ResponseInfo{
			Status:           statusCode,
			SimulatedDelayMs: delayMs,
		},
	}
}

// handleSimulatedRouteInfo provides information about available simulated routes
func (s *Server) handleSimulatedRouteInfo(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != SimulatedRoutePrefix && r.URL.Path != SimulatedRoutePrefix+"/" {
		// This is an actual simulated route request
		s.handleSimulatedRoute(w, r)
		return
	}

	// Provide information about /sim endpoint
	if s.configManager == nil {
		writeError(w, "configuration not available", http.StatusServiceUnavailable)
		return
	}

	routes := s.configManager.GetIncomingRoutes()
	enabledRoutes := make([]map[string]interface{}, 0)

	for _, route := range routes {
		if route.Enabled {
			enabledRoutes = append(enabledRoutes, map[string]interface{}{
				"name":      route.Name,
				"path":      SimulatedRoutePrefix + route.Path,
				"method":    route.Method,
				"responses": len(route.Responses),
			})
		}
	}

	response := map[string]interface{}{
		"description":    "Simulated incoming routes endpoint",
		"prefix":         SimulatedRoutePrefix,
		"enabled_routes": len(enabledRoutes),
		"routes":         enabledRoutes,
		"usage":          "Send requests to " + SimulatedRoutePrefix + "/{path} to trigger simulated responses",
	}

	writeJSON(w, response)
}
