package metrics

import (
	"testing"
	"time"
)

func TestIncomingRouteMetrics_Record(t *testing.T) {
	metrics := NewIncomingRouteMetrics("test_route", "/api/test")

	// Record some requests
	metrics.Record(200, 100.0)
	metrics.Record(200, 150.0)
	metrics.Record(500, 50.0)

	stats := metrics.GetStats()

	if stats.TotalRequests != 3 {
		t.Errorf("expected 3 total requests, got %d", stats.TotalRequests)
	}

	if stats.ResponsesByStatus[200] != 2 {
		t.Errorf("expected 2 requests with status 200, got %d", stats.ResponsesByStatus[200])
	}

	if stats.ResponsesByStatus[500] != 1 {
		t.Errorf("expected 1 request with status 500, got %d", stats.ResponsesByStatus[500])
	}

	expectedAvg := (100.0 + 150.0 + 50.0) / 3.0
	if stats.AvgResponseMs != expectedAvg {
		t.Errorf("expected avg response %.2f, got %.2f", expectedAvg, stats.AvgResponseMs)
	}

	if stats.RouteName != "test_route" {
		t.Errorf("expected route name 'test_route', got '%s'", stats.RouteName)
	}

	if stats.RoutePath != "/api/test" {
		t.Errorf("expected route path '/api/test', got '%s'", stats.RoutePath)
	}
}

func TestIncomingRouteMetrics_Reset(t *testing.T) {
	metrics := NewIncomingRouteMetrics("test_route", "/api/test")

	metrics.Record(200, 100.0)
	metrics.Record(500, 50.0)

	stats := metrics.GetStats()
	if stats.TotalRequests != 2 {
		t.Errorf("expected 2 requests before reset, got %d", stats.TotalRequests)
	}

	metrics.Reset()

	stats = metrics.GetStats()
	if stats.TotalRequests != 0 {
		t.Errorf("expected 0 requests after reset, got %d", stats.TotalRequests)
	}

	if len(stats.ResponsesByStatus) != 0 {
		t.Error("expected empty status map after reset")
	}
}

func TestIncomingCollector_Record(t *testing.T) {
	collector := NewIncomingCollector()

	// Record requests to different routes
	collector.Record("route1", "/api/route1", 200, 100.0)
	collector.Record("route1", "/api/route1", 200, 150.0)
	collector.Record("route2", "/api/route2", 200, 50.0)
	collector.Record("route2", "/api/route2", 500, 200.0)

	// Check total
	if collector.GetTotalRequests() != 4 {
		t.Errorf("expected 4 total requests, got %d", collector.GetTotalRequests())
	}

	// Check snapshot
	snapshot := collector.Snapshot()

	if snapshot.TotalRequests != 4 {
		t.Errorf("expected 4 requests in snapshot, got %d", snapshot.TotalRequests)
	}

	if len(snapshot.Routes) != 2 {
		t.Errorf("expected 2 routes in snapshot, got %d", len(snapshot.Routes))
	}

	route1Stats, ok := snapshot.Routes["route1"]
	if !ok {
		t.Fatal("route1 not found in snapshot")
	}
	if route1Stats.TotalRequests != 2 {
		t.Errorf("expected 2 requests for route1, got %d", route1Stats.TotalRequests)
	}

	route2Stats, ok := snapshot.Routes["route2"]
	if !ok {
		t.Fatal("route2 not found in snapshot")
	}
	if route2Stats.TotalRequests != 2 {
		t.Errorf("expected 2 requests for route2, got %d", route2Stats.TotalRequests)
	}
	if route2Stats.ResponsesByStatus[500] != 1 {
		t.Errorf("expected 1 status 500 for route2, got %d", route2Stats.ResponsesByStatus[500])
	}
}

func TestIncomingCollector_GetRouteMetrics(t *testing.T) {
	collector := NewIncomingCollector()

	collector.Record("route1", "/api/route1", 200, 100.0)

	stats, found := collector.GetRouteMetrics("route1")
	if !found {
		t.Fatal("route1 should be found")
	}
	if stats.TotalRequests != 1 {
		t.Errorf("expected 1 request, got %d", stats.TotalRequests)
	}

	_, found = collector.GetRouteMetrics("nonexistent")
	if found {
		t.Error("nonexistent route should not be found")
	}
}

func TestIncomingCollector_Reset(t *testing.T) {
	collector := NewIncomingCollector()

	collector.Record("route1", "/api/route1", 200, 100.0)
	collector.Record("route2", "/api/route2", 200, 50.0)

	if collector.GetTotalRequests() != 2 {
		t.Errorf("expected 2 requests before reset, got %d", collector.GetTotalRequests())
	}

	collector.Reset()

	if collector.GetTotalRequests() != 0 {
		t.Errorf("expected 0 requests after reset, got %d", collector.GetTotalRequests())
	}

	snapshot := collector.Snapshot()
	if len(snapshot.Routes) != 0 {
		t.Error("expected no routes after reset")
	}
}

func TestIncomingCollector_RequestsPerSecond(t *testing.T) {
	collector := NewIncomingCollector()

	// Record some requests
	for i := 0; i < 10; i++ {
		collector.Record("route1", "/api/route1", 200, 100.0)
	}

	// Wait a short time
	time.Sleep(100 * time.Millisecond)

	rps := collector.GetRequestsPerSecond()

	// Should be positive since we recorded requests
	if rps <= 0 {
		t.Errorf("expected positive requests per second, got %.2f", rps)
	}

	// Should be reasonable (not astronomical)
	if rps > 1000 {
		t.Errorf("requests per second seems too high: %.2f", rps)
	}
}

func TestIncomingCollector_Percentiles(t *testing.T) {
	metrics := NewIncomingRouteMetrics("test_route", "/api/test")

	// Record many requests with varying response times
	for i := 1; i <= 100; i++ {
		metrics.Record(200, float64(i))
	}

	stats := metrics.GetStats()

	// P95 should be around 95
	if stats.P95ResponseMs < 90 || stats.P95ResponseMs > 100 {
		t.Errorf("P95 should be around 95, got %.2f", stats.P95ResponseMs)
	}

	// P99 should be around 99
	if stats.P99ResponseMs < 95 || stats.P99ResponseMs > 100 {
		t.Errorf("P99 should be around 99, got %.2f", stats.P99ResponseMs)
	}

	// Max should be 100
	if stats.MaxResponseMs != 100 {
		t.Errorf("Max should be 100, got %.2f", stats.MaxResponseMs)
	}

	// Min should be 1
	if stats.MinResponseMs != 1 {
		t.Errorf("Min should be 1, got %.2f", stats.MinResponseMs)
	}
}
