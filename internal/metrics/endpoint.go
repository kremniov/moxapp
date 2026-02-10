// Package metrics provides in-memory metrics collection
package metrics

import (
	"sync"
	"time"
)

// EndpointMetrics holds metrics for a single endpoint
type EndpointMetrics struct {
	TotalRequests    int64 `json:"total_requests"`
	Successful       int64 `json:"successful"`
	Failed           int64 `json:"failed"`
	TimeoutErrors    int64 `json:"timeout_errors"`
	DNSErrors        int64 `json:"dns_errors"`
	ConnectionErrors int64 `json:"connection_errors"`
	HTTPErrors       int64 `json:"http_errors"`
	OtherErrors      int64 `json:"other_errors"`

	TotalTimeMs    float64 `json:"-"` // Not exported, used for avg calculation
	TotalDNSTimeMs float64 `json:"-"`
	TotalConnectMs float64 `json:"-"`

	ResponseTimes *RingBuffer `json:"-"` // For percentiles
	DNSTimes      *RingBuffer `json:"-"`

	LastStatusCode int       `json:"last_status_code"`
	LastError      string    `json:"last_error"`
	LastSuccess    time.Time `json:"last_success,omitempty"`

	URLPattern string `json:"url_pattern"`
	Hostname   string `json:"hostname"`

	mu sync.Mutex
}

// NewEndpointMetrics creates new endpoint metrics
func NewEndpointMetrics(urlPattern, hostname string) *EndpointMetrics {
	return &EndpointMetrics{
		ResponseTimes: NewRingBuffer(1000),
		DNSTimes:      NewRingBuffer(1000),
		URLPattern:    urlPattern,
		Hostname:      hostname,
	}
}

// RecordSuccess records a successful request
func (em *EndpointMetrics) RecordSuccess(totalTimeMs, dnsTimeMs, connectTimeMs float64, statusCode int) {
	em.mu.Lock()
	defer em.mu.Unlock()

	em.TotalRequests++
	em.Successful++
	em.LastStatusCode = statusCode
	em.LastSuccess = time.Now()

	em.TotalTimeMs += totalTimeMs
	em.TotalDNSTimeMs += dnsTimeMs
	em.TotalConnectMs += connectTimeMs

	em.ResponseTimes.Add(totalTimeMs)
	if dnsTimeMs > 0 {
		em.DNSTimes.Add(dnsTimeMs)
	}
}

// RecordFailure records a failed request
func (em *EndpointMetrics) RecordFailure(totalTimeMs, dnsTimeMs, connectTimeMs float64, statusCode int, errorType, errorMsg string) {
	em.mu.Lock()
	defer em.mu.Unlock()

	em.TotalRequests++
	em.Failed++
	em.LastStatusCode = statusCode
	em.LastError = errorMsg

	em.TotalTimeMs += totalTimeMs
	em.TotalDNSTimeMs += dnsTimeMs
	em.TotalConnectMs += connectTimeMs

	em.ResponseTimes.Add(totalTimeMs)
	if dnsTimeMs > 0 {
		em.DNSTimes.Add(dnsTimeMs)
	}

	// Categorize error
	switch errorType {
	case "timeout":
		em.TimeoutErrors++
	case "dns":
		em.DNSErrors++
	case "connection":
		em.ConnectionErrors++
	case "http":
		em.HTTPErrors++
	default:
		em.OtherErrors++
	}
}

// GetStats returns a snapshot of the endpoint metrics
func (em *EndpointMetrics) GetStats() EndpointSnapshot {
	em.mu.Lock()
	defer em.mu.Unlock()

	snap := EndpointSnapshot{
		TotalRequests:    em.TotalRequests,
		Successful:       em.Successful,
		Failed:           em.Failed,
		TimeoutErrors:    em.TimeoutErrors,
		DNSErrors:        em.DNSErrors,
		ConnectionErrors: em.ConnectionErrors,
		HTTPErrors:       em.HTTPErrors,
		OtherErrors:      em.OtherErrors,
		LastStatusCode:   em.LastStatusCode,
		LastError:        em.LastError,
		URLPattern:       em.URLPattern,
		Hostname:         em.Hostname,
	}

	if !em.LastSuccess.IsZero() {
		snap.LastSuccess = em.LastSuccess.Format(time.RFC3339)
	}

	if em.TotalRequests > 0 {
		snap.SuccessRate = float64(em.Successful) / float64(em.TotalRequests) * 100
		snap.AvgTotalTimeMs = em.TotalTimeMs / float64(em.TotalRequests)
		if em.TotalDNSTimeMs > 0 {
			snap.AvgDNSTimeMs = em.TotalDNSTimeMs / float64(em.TotalRequests)
		}
		if em.TotalConnectMs > 0 {
			snap.AvgConnectTimeMs = em.TotalConnectMs / float64(em.TotalRequests)
		}
	}

	snap.P95TotalTimeMs = em.ResponseTimes.Percentile(95)
	snap.P99TotalTimeMs = em.ResponseTimes.Percentile(99)
	snap.MaxTotalTimeMs = em.ResponseTimes.Max()
	snap.P95DNSTimeMs = em.DNSTimes.Percentile(95)

	return snap
}

// Reset clears all metrics
func (em *EndpointMetrics) Reset() {
	em.mu.Lock()
	defer em.mu.Unlock()

	em.TotalRequests = 0
	em.Successful = 0
	em.Failed = 0
	em.TimeoutErrors = 0
	em.DNSErrors = 0
	em.ConnectionErrors = 0
	em.HTTPErrors = 0
	em.OtherErrors = 0
	em.TotalTimeMs = 0
	em.TotalDNSTimeMs = 0
	em.TotalConnectMs = 0
	em.LastStatusCode = 0
	em.LastError = ""
	em.LastSuccess = time.Time{}
	em.ResponseTimes.Reset()
	em.DNSTimes.Reset()
}

// EndpointSnapshot is a serializable snapshot of endpoint metrics
type EndpointSnapshot struct {
	TotalRequests    int64   `json:"total_requests"`
	Successful       int64   `json:"successful"`
	Failed           int64   `json:"failed"`
	SuccessRate      float64 `json:"success_rate"`
	TimeoutErrors    int64   `json:"timeout_errors"`
	DNSErrors        int64   `json:"dns_errors"`
	ConnectionErrors int64   `json:"connection_errors"`
	HTTPErrors       int64   `json:"http_errors"`
	OtherErrors      int64   `json:"other_errors"`

	AvgTotalTimeMs   float64 `json:"avg_total_time_ms"`
	AvgDNSTimeMs     float64 `json:"avg_dns_time_ms"`
	AvgConnectTimeMs float64 `json:"avg_connect_time_ms"`
	P95TotalTimeMs   float64 `json:"p95_total_time_ms"`
	P99TotalTimeMs   float64 `json:"p99_total_time_ms"`
	MaxTotalTimeMs   float64 `json:"max_total_time_ms"`
	P95DNSTimeMs     float64 `json:"p95_dns_time_ms"`

	LastStatusCode int    `json:"last_status_code"`
	LastError      string `json:"last_error,omitempty"`
	LastSuccess    string `json:"last_success,omitempty"`

	URLPattern string `json:"url_pattern"`
	Hostname   string `json:"hostname"`
}
