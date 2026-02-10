// Package client provides HTTP client functionality with DNS timing
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptrace"
	"time"

	"moxapp/internal/config"
)

// RequestResult holds the result of an HTTP request
type RequestResult struct {
	EndpointName     string    `json:"endpoint_name"`
	URL              string    `json:"url"`
	Method           string    `json:"method"`
	StatusCode       int       `json:"status_code"`
	Success          bool      `json:"success"`
	Error            string    `json:"error,omitempty"`
	ErrorType        string    `json:"error_type,omitempty"`
	TotalTimeMs      float64   `json:"total_time_ms"`
	DNSTimeMs        float64   `json:"dns_time_ms"`
	ConnectTimeMs    float64   `json:"connect_time_ms"`
	TLSTimeMs        float64   `json:"tls_time_ms"`
	TimeToFirstByte  float64   `json:"time_to_first_byte_ms"`
	Hostname         string    `json:"hostname"`
	ResponseSize     int64     `json:"response_size"`
	RequestTimestamp time.Time `json:"request_timestamp"`
}

// Client is the HTTP client with DNS timing capabilities
type Client struct {
	httpClient   *http.Client
	tokenManager *TokenManager
	logRequests  bool
}

// ClientOptions configures the HTTP client
type ClientOptions struct {
	Timeout      time.Duration
	MaxConns     int
	LogRequests  bool
	EnvGetter    EnvGetter
	AuthConfigs  map[string]*config.AuthConfig
	TokenManager *TokenManager
}

// DefaultOptions returns the default client options
func DefaultOptions() ClientOptions {
	return ClientOptions{
		Timeout:     30 * time.Second,
		MaxConns:    100,
		LogRequests: false,
	}
}

// New creates a new HTTP client
func New(opts ClientOptions) *Client {
	transport := &http.Transport{
		MaxIdleConns:        opts.MaxConns,
		MaxIdleConnsPerHost: opts.MaxConns,
		MaxConnsPerHost:     opts.MaxConns,
		IdleConnTimeout:     90 * time.Second,
		DisableKeepAlives:   false,
		ForceAttemptHTTP2:   true,
	}

	client := &Client{
		httpClient: &http.Client{
			Transport: transport,
			Timeout:   opts.Timeout,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse // Don't follow redirects automatically
			},
		},
		logRequests: opts.LogRequests,
	}

	// Use provided TokenManager or create a new one
	if opts.TokenManager != nil {
		client.tokenManager = opts.TokenManager
	} else if opts.AuthConfigs != nil && opts.EnvGetter != nil {
		client.tokenManager = NewTokenManager(opts.AuthConfigs, opts.EnvGetter)
	}

	return client
}

// Execute executes an HTTP request for the given endpoint
func (c *Client) Execute(ctx context.Context, endpoint *config.Endpoint) *RequestResult {
	result := &RequestResult{
		EndpointName:     endpoint.Name,
		Method:           endpoint.Method,
		RequestTimestamp: time.Now(),
	}

	startTime := time.Now()

	// Evaluate URL template
	evaluatedURL, err := config.EvaluateTemplate(endpoint.URLTemplate)
	if err != nil {
		result.Error = fmt.Sprintf("Template error: %v", err)
		result.ErrorType = "template"
		result.TotalTimeMs = float64(time.Since(startTime).Microseconds()) / 1000.0
		return result
	}
	result.URL = evaluatedURL
	result.Hostname = ExtractHostname(evaluatedURL)

	// Prepare request body if needed
	var bodyReader io.Reader
	if endpoint.Body != nil && (endpoint.Method == "POST" || endpoint.Method == "PUT" || endpoint.Method == "PATCH") {
		// Evaluate body template
		evaluatedBody, err := config.EvaluateBodyTemplate(endpoint.Body)
		if err != nil {
			result.Error = fmt.Sprintf("Body template error: %v", err)
			result.ErrorType = "template"
			result.TotalTimeMs = float64(time.Since(startTime).Microseconds()) / 1000.0
			return result
		}

		bodyBytes, err := json.Marshal(evaluatedBody)
		if err != nil {
			result.Error = fmt.Sprintf("Body marshal error: %v", err)
			result.ErrorType = "marshal"
			result.TotalTimeMs = float64(time.Since(startTime).Microseconds()) / 1000.0
			return result
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	// Create request with context
	req, err := http.NewRequestWithContext(ctx, endpoint.Method, evaluatedURL, bodyReader)
	if err != nil {
		result.Error = fmt.Sprintf("Failed to create request: %v", err)
		result.ErrorType = "request"
		result.TotalTimeMs = float64(time.Since(startTime).Microseconds()) / 1000.0
		return result
	}

	// Set headers
	req.Header.Set("User-Agent", "moxapp/1.0")
	if bodyReader != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for key, value := range endpoint.Headers {
		// Evaluate header value template
		evaluatedValue, err := config.EvaluateTemplate(value)
		if err != nil {
			evaluatedValue = value // Use original if template fails
		}
		req.Header.Set(key, evaluatedValue)
	}

	// Apply authentication
	if endpoint.ResolvedAuth != nil && c.tokenManager != nil {
		if err := ApplyAuth(req, endpoint.ResolvedAuth, c.tokenManager); err != nil {
			result.Error = fmt.Sprintf("Auth error: %v", err)
			result.ErrorType = "auth"
			result.TotalTimeMs = float64(time.Since(startTime).Microseconds()) / 1000.0
			return result
		}
	}

	// Setup DNS/connection tracing
	var timing TimingInfo
	timing.RequestStart = time.Now()
	trace := CreateClientTrace(&timing)
	req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))

	// Execute request
	resp, err := c.httpClient.Do(req)
	timing.RequestDone = time.Now()

	// Calculate total time
	result.TotalTimeMs = float64(time.Since(startTime).Microseconds()) / 1000.0

	if err != nil {
		errorType, errorMsg := CategorizeError(err)
		result.ErrorType = errorType
		result.Error = errorMsg

		// Still capture timing info if available
		result.DNSTimeMs = timing.DNSTimeMs()
		result.ConnectTimeMs = timing.ConnectTimeMs()
		result.TLSTimeMs = timing.TLSTimeMs()
		return result
	}
	defer resp.Body.Close()

	// Read and discard body to allow connection reuse
	bodySize, _ := io.Copy(io.Discard, resp.Body)
	result.ResponseSize = bodySize

	// Set timing results
	result.DNSTimeMs = timing.DNSTimeMs()
	result.ConnectTimeMs = timing.ConnectTimeMs()
	result.TLSTimeMs = timing.TLSTimeMs()
	result.TimeToFirstByte = timing.TimeToFirstByteMs()

	// Set status and success
	result.StatusCode = resp.StatusCode
	result.Success = resp.StatusCode >= 200 && resp.StatusCode < 400

	if !result.Success {
		result.ErrorType = "http"
		result.Error = fmt.Sprintf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	return result
}

// SetLogRequests enables or disables request logging
func (c *Client) SetLogRequests(log bool) {
	c.logRequests = log
}

// GetTokenManager returns the token manager for managing dynamic tokens
func (c *Client) GetTokenManager() *TokenManager {
	return c.tokenManager
}

// SetTokenManager sets the token manager
func (c *Client) SetTokenManager(tm *TokenManager) {
	c.tokenManager = tm
}
