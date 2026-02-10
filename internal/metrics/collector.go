// Package metrics provides in-memory metrics collection
package metrics

import (
	"sync"
	"sync/atomic"
	"time"

	"moxapp/internal/client"
)

// Collector collects and aggregates metrics from all requests
type Collector struct {
	startTime      time.Time
	totalRequests  int64
	totalSuccesses int64
	totalFailures  int64

	endpoints map[string]*EndpointMetrics
	domains   map[string]*DomainMetrics

	mu sync.RWMutex
}

// NewCollector creates a new metrics collector
func NewCollector() *Collector {
	return &Collector{
		startTime: time.Now(),
		endpoints: make(map[string]*EndpointMetrics),
		domains:   make(map[string]*DomainMetrics),
	}
}

// Record records the result of an HTTP request
func (c *Collector) Record(result *client.RequestResult) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Update global counters
	atomic.AddInt64(&c.totalRequests, 1)
	if result.Success {
		atomic.AddInt64(&c.totalSuccesses, 1)
	} else {
		atomic.AddInt64(&c.totalFailures, 1)
	}

	// Get or create endpoint metrics
	ep, exists := c.endpoints[result.EndpointName]
	if !exists {
		ep = NewEndpointMetrics(result.URL, result.Hostname)
		c.endpoints[result.EndpointName] = ep
	}

	// Update endpoint metrics
	if result.Success {
		ep.RecordSuccess(result.TotalTimeMs, result.DNSTimeMs, result.ConnectTimeMs, result.StatusCode)
	} else {
		ep.RecordFailure(result.TotalTimeMs, result.DNSTimeMs, result.ConnectTimeMs, result.StatusCode, result.ErrorType, result.Error)
	}

	// Update domain metrics only when we actually performed DNS work
	if result.Hostname != "" {
		// DNS success if we got a positive DNS time and no DNS error
		if result.DNSTimeMs > 0 && result.ErrorType != "dns" {
			domain, exists := c.domains[result.Hostname]
			if !exists {
				domain = NewDomainMetrics()
				c.domains[result.Hostname] = domain
			}
			domain.RecordSuccess(result.DNSTimeMs)
		} else if result.ErrorType == "dns" {
			domain, exists := c.domains[result.Hostname]
			if !exists {
				domain = NewDomainMetrics()
				c.domains[result.Hostname] = domain
			}
			domain.RecordFailure(result.Error)
		}
	}
}

// Snapshot returns a serializable snapshot of all metrics
func (c *Collector) Snapshot() *MetricsSnapshot {
	c.mu.RLock()
	defer c.mu.RUnlock()

	uptime := time.Since(c.startTime).Seconds()

	snapshot := &MetricsSnapshot{
		UptimeSeconds:    uptime,
		TotalRequests:    atomic.LoadInt64(&c.totalRequests),
		TotalSuccesses:   atomic.LoadInt64(&c.totalSuccesses),
		TotalFailures:    atomic.LoadInt64(&c.totalFailures),
		Endpoints:        make(map[string]EndpointSnapshot),
		DNSStatsByDomain: make(map[string]DomainSnapshot),
		CollectedAt:      time.Now().Format(time.RFC3339),
	}

	// Calculate rates
	if uptime > 0 {
		snapshot.RequestsPerSecond = float64(snapshot.TotalRequests) / uptime
	}
	if snapshot.TotalRequests > 0 {
		snapshot.SuccessRate = float64(snapshot.TotalSuccesses) / float64(snapshot.TotalRequests) * 100
	}

	// Collect endpoint metrics
	for name, ep := range c.endpoints {
		snapshot.Endpoints[name] = ep.GetStats()
	}

	// Collect domain metrics
	for hostname, domain := range c.domains {
		snapshot.DNSStatsByDomain[hostname] = domain.GetStats()
	}

	return snapshot
}

// Reset resets all metrics
func (c *Collector) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.startTime = time.Now()
	atomic.StoreInt64(&c.totalRequests, 0)
	atomic.StoreInt64(&c.totalSuccesses, 0)
	atomic.StoreInt64(&c.totalFailures, 0)
	c.endpoints = make(map[string]*EndpointMetrics)
	c.domains = make(map[string]*DomainMetrics)
}

// GetTotalRequests returns the total number of requests
func (c *Collector) GetTotalRequests() int64 {
	return atomic.LoadInt64(&c.totalRequests)
}

// GetSuccessRate returns the current success rate as a percentage
func (c *Collector) GetSuccessRate() float64 {
	total := atomic.LoadInt64(&c.totalRequests)
	if total == 0 {
		return 100.0
	}
	successes := atomic.LoadInt64(&c.totalSuccesses)
	return float64(successes) / float64(total) * 100
}

// GetRequestsPerSecond returns the current requests per second rate
func (c *Collector) GetRequestsPerSecond() float64 {
	uptime := time.Since(c.startTime).Seconds()
	if uptime == 0 {
		return 0
	}
	return float64(atomic.LoadInt64(&c.totalRequests)) / uptime
}

// MetricsSnapshot is a serializable snapshot of all metrics
type MetricsSnapshot struct {
	UptimeSeconds     float64                     `json:"uptime_seconds"`
	TotalRequests     int64                       `json:"total_requests"`
	TotalSuccesses    int64                       `json:"total_successes"`
	TotalFailures     int64                       `json:"total_failures"`
	SuccessRate       float64                     `json:"success_rate"`
	RequestsPerSecond float64                     `json:"requests_per_second"`
	CollectedAt       string                      `json:"collected_at"`
	Endpoints         map[string]EndpointSnapshot `json:"endpoints"`
	DNSStatsByDomain  map[string]DomainSnapshot   `json:"dns_stats_by_domain"`
}
