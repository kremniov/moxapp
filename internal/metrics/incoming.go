// Package metrics provides in-memory metrics collection
package metrics

import (
	"sync"
	"sync/atomic"
	"time"
)

// IncomingRouteMetrics holds metrics for a single incoming route
type IncomingRouteMetrics struct {
	TotalRequests     int64         `json:"total_requests"`
	ResponsesByStatus map[int]int64 `json:"responses_by_status"`

	TotalResponseMs float64     `json:"-"` // Not exported, used for avg calculation
	ResponseTimes   *RingBuffer `json:"-"` // For percentiles

	LastRequest time.Time `json:"last_request,omitempty"`

	RouteName string `json:"route_name"`
	RoutePath string `json:"route_path"`

	mu sync.Mutex
}

// NewIncomingRouteMetrics creates new incoming route metrics
func NewIncomingRouteMetrics(routeName, routePath string) *IncomingRouteMetrics {
	return &IncomingRouteMetrics{
		ResponsesByStatus: make(map[int]int64),
		ResponseTimes:     NewRingBuffer(1000),
		RouteName:         routeName,
		RoutePath:         routePath,
	}
}

// Record records a request to this incoming route
func (m *IncomingRouteMetrics) Record(statusCode int, responseTimeMs float64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.TotalRequests++
	m.ResponsesByStatus[statusCode]++
	m.TotalResponseMs += responseTimeMs
	m.ResponseTimes.Add(responseTimeMs)
	m.LastRequest = time.Now()
}

// GetStats returns a snapshot of the incoming route metrics
func (m *IncomingRouteMetrics) GetStats() IncomingRouteSnapshot {
	m.mu.Lock()
	defer m.mu.Unlock()

	snap := IncomingRouteSnapshot{
		TotalRequests:     m.TotalRequests,
		ResponsesByStatus: make(map[int]int64),
		RouteName:         m.RouteName,
		RoutePath:         m.RoutePath,
	}

	// Copy status counts
	for status, count := range m.ResponsesByStatus {
		snap.ResponsesByStatus[status] = count
	}

	if !m.LastRequest.IsZero() {
		snap.LastRequest = m.LastRequest.Format(time.RFC3339)
	}

	if m.TotalRequests > 0 {
		snap.AvgResponseMs = m.TotalResponseMs / float64(m.TotalRequests)
	}

	snap.P95ResponseMs = m.ResponseTimes.Percentile(95)
	snap.P99ResponseMs = m.ResponseTimes.Percentile(99)
	snap.MaxResponseMs = m.ResponseTimes.Max()
	snap.MinResponseMs = m.ResponseTimes.Min()

	return snap
}

// Reset clears all metrics
func (m *IncomingRouteMetrics) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.TotalRequests = 0
	m.ResponsesByStatus = make(map[int]int64)
	m.TotalResponseMs = 0
	m.LastRequest = time.Time{}
	m.ResponseTimes.Reset()
}

// IncomingRouteSnapshot is a serializable snapshot of incoming route metrics
type IncomingRouteSnapshot struct {
	TotalRequests     int64         `json:"total_requests"`
	ResponsesByStatus map[int]int64 `json:"responses_by_status"`

	AvgResponseMs float64 `json:"avg_response_ms"`
	P95ResponseMs float64 `json:"p95_response_ms"`
	P99ResponseMs float64 `json:"p99_response_ms"`
	MaxResponseMs float64 `json:"max_response_ms"`
	MinResponseMs float64 `json:"min_response_ms"`

	LastRequest string `json:"last_request,omitempty"`

	RouteName string `json:"route_name"`
	RoutePath string `json:"route_path"`
}

// IncomingCollector collects and aggregates metrics for incoming routes
type IncomingCollector struct {
	startTime     time.Time
	totalRequests int64

	routes map[string]*IncomingRouteMetrics // keyed by route name

	mu sync.RWMutex
}

// NewIncomingCollector creates a new incoming metrics collector
func NewIncomingCollector() *IncomingCollector {
	return &IncomingCollector{
		startTime: time.Now(),
		routes:    make(map[string]*IncomingRouteMetrics),
	}
}

// Record records a request to an incoming route
func (c *IncomingCollector) Record(routeName, routePath string, statusCode int, responseTimeMs float64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	atomic.AddInt64(&c.totalRequests, 1)

	// Get or create route metrics
	route, exists := c.routes[routeName]
	if !exists {
		route = NewIncomingRouteMetrics(routeName, routePath)
		c.routes[routeName] = route
	}

	route.Record(statusCode, responseTimeMs)
}

// Snapshot returns a serializable snapshot of all incoming route metrics
func (c *IncomingCollector) Snapshot() *IncomingMetricsSnapshot {
	c.mu.RLock()
	defer c.mu.RUnlock()

	uptime := time.Since(c.startTime).Seconds()

	snapshot := &IncomingMetricsSnapshot{
		UptimeSeconds: uptime,
		TotalRequests: atomic.LoadInt64(&c.totalRequests),
		Routes:        make(map[string]IncomingRouteSnapshot),
		CollectedAt:   time.Now().Format(time.RFC3339),
	}

	// Calculate requests per second
	if uptime > 0 {
		snapshot.RequestsPerSecond = float64(snapshot.TotalRequests) / uptime
	}

	// Collect route metrics
	for name, route := range c.routes {
		snapshot.Routes[name] = route.GetStats()
	}

	return snapshot
}

// Reset resets all incoming metrics
func (c *IncomingCollector) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.startTime = time.Now()
	atomic.StoreInt64(&c.totalRequests, 0)
	c.routes = make(map[string]*IncomingRouteMetrics)
}

// GetTotalRequests returns the total number of incoming requests
func (c *IncomingCollector) GetTotalRequests() int64 {
	return atomic.LoadInt64(&c.totalRequests)
}

// GetRequestsPerSecond returns the current requests per second rate
func (c *IncomingCollector) GetRequestsPerSecond() float64 {
	uptime := time.Since(c.startTime).Seconds()
	if uptime == 0 {
		return 0
	}
	return float64(atomic.LoadInt64(&c.totalRequests)) / uptime
}

// GetRouteMetrics returns metrics for a specific route
func (c *IncomingCollector) GetRouteMetrics(routeName string) (*IncomingRouteSnapshot, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	route, exists := c.routes[routeName]
	if !exists {
		return nil, false
	}

	stats := route.GetStats()
	return &stats, true
}

// IncomingMetricsSnapshot is a serializable snapshot of all incoming metrics
type IncomingMetricsSnapshot struct {
	UptimeSeconds     float64                          `json:"uptime_seconds"`
	TotalRequests     int64                            `json:"total_requests"`
	RequestsPerSecond float64                          `json:"requests_per_second"`
	CollectedAt       string                           `json:"collected_at"`
	Routes            map[string]IncomingRouteSnapshot `json:"routes"`
}
