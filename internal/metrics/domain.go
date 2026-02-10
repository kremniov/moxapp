// Package metrics provides in-memory metrics collection
package metrics

import (
	"sync"
)

// DomainMetrics holds DNS metrics for a single domain
type DomainMetrics struct {
	TotalLookups      int64 `json:"total_lookups"`
	SuccessfulLookups int64 `json:"successful_lookups"`
	FailedLookups     int64 `json:"failed_lookups"`

	TotalDNSTimeMs float64     `json:"-"` // Not exported, used for avg calculation
	DNSTimes       *RingBuffer `json:"-"` // For percentiles

	LastError string `json:"last_error,omitempty"`

	mu sync.Mutex
}

// NewDomainMetrics creates new domain metrics
func NewDomainMetrics() *DomainMetrics {
	return &DomainMetrics{
		DNSTimes: NewRingBuffer(1000),
	}
}

// RecordSuccess records a successful DNS lookup
func (dm *DomainMetrics) RecordSuccess(dnsTimeMs float64) {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	dm.TotalLookups++
	dm.SuccessfulLookups++
	dm.TotalDNSTimeMs += dnsTimeMs
	dm.DNSTimes.Add(dnsTimeMs)
}

// RecordFailure records a failed DNS lookup
func (dm *DomainMetrics) RecordFailure(errorMsg string) {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	dm.TotalLookups++
	dm.FailedLookups++
	dm.LastError = errorMsg
}

// GetStats returns a snapshot of the domain metrics
func (dm *DomainMetrics) GetStats() DomainSnapshot {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	snap := DomainSnapshot{
		TotalLookups:      dm.TotalLookups,
		SuccessfulLookups: dm.SuccessfulLookups,
		FailedLookups:     dm.FailedLookups,
		LastError:         dm.LastError,
	}

	if dm.SuccessfulLookups > 0 && dm.TotalDNSTimeMs > 0 {
		snap.AvgResolutionMs = dm.TotalDNSTimeMs / float64(dm.SuccessfulLookups)
	}

	snap.P95ResolutionMs = dm.DNSTimes.Percentile(95)
	snap.MaxResolutionMs = dm.DNSTimes.Max()
	snap.MinResolutionMs = dm.DNSTimes.Min()

	return snap
}

// Reset clears all metrics
func (dm *DomainMetrics) Reset() {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	dm.TotalLookups = 0
	dm.SuccessfulLookups = 0
	dm.FailedLookups = 0
	dm.TotalDNSTimeMs = 0
	dm.LastError = ""
	dm.DNSTimes.Reset()
}

// DomainSnapshot is a serializable snapshot of domain metrics
type DomainSnapshot struct {
	TotalLookups      int64   `json:"total_lookups"`
	SuccessfulLookups int64   `json:"successful_lookups"`
	FailedLookups     int64   `json:"failed_lookups"`
	AvgResolutionMs   float64 `json:"avg_resolution_ms"`
	P95ResolutionMs   float64 `json:"p95_resolution_ms"`
	MaxResolutionMs   float64 `json:"max_resolution_ms"`
	MinResolutionMs   float64 `json:"min_resolution_ms"`
	LastError         string  `json:"last_error,omitempty"`
}

// DNSStats aggregates DNS statistics across all domains
type DNSStats struct {
	TotalLookups      int64                      `json:"total_lookups"`
	SuccessfulLookups int64                      `json:"successful_lookups"`
	FailedLookups     int64                      `json:"failed_lookups"`
	AvgResolutionMs   float64                    `json:"avg_resolution_ms"`
	ByDomain          map[string]*DomainSnapshot `json:"by_domain"`
}

// CalculateDNSStats calculates aggregate DNS statistics from domain snapshots
func CalculateDNSStats(domains map[string]DomainSnapshot) DNSStats {
	stats := DNSStats{
		ByDomain: make(map[string]*DomainSnapshot),
	}

	var totalDNSTime float64

	for hostname, snap := range domains {
		snapCopy := snap // Create a copy to avoid pointer issues
		stats.ByDomain[hostname] = &snapCopy
		stats.TotalLookups += snap.TotalLookups
		stats.SuccessfulLookups += snap.SuccessfulLookups
		stats.FailedLookups += snap.FailedLookups
		totalDNSTime += snap.AvgResolutionMs * float64(snap.SuccessfulLookups)
	}

	if stats.SuccessfulLookups > 0 {
		stats.AvgResolutionMs = totalDNSTime / float64(stats.SuccessfulLookups)
	}

	return stats
}
