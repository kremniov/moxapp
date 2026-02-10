// Package scheduler provides the request scheduling logic
package scheduler

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"moxapp/internal/client"
	"moxapp/internal/config"
)

// ResultHandler is a callback function for handling request results
type ResultHandler func(*client.RequestResult)

// Scheduler orchestrates the load test execution
type Scheduler struct {
	configManager *config.Manager
	client        *client.Client
	resultHandler ResultHandler

	nextRequestTime map[string]time.Time
	mu              sync.RWMutex

	semaphore chan struct{} // Limits concurrency
	stopChan  chan struct{}
	wg        sync.WaitGroup

	// Statistics
	requestsScheduled int64
	requestsInFlight  int64
	requestsSkipped   int64 // Skipped due to disabled state

	// State
	running   bool
	runningMu sync.Mutex

	// Big red stop button - atomic for instant access without locks
	// 0 = running (enabled), 1 = paused (disabled)
	paused int32

	// Context for cancelling in-flight requests on emergency stop
	baseCtx    context.Context
	cancelFunc context.CancelFunc
	ctx        context.Context
}

// SchedulerStats holds scheduler statistics
type SchedulerStats struct {
	RequestsScheduled int64
	RequestsInFlight  int64
	RequestsSkipped   int64
	ActiveEndpoints   int
	EnabledEndpoints  int
	Paused            bool
	GlobalEnabled     bool
}

// New creates a new scheduler with config manager
func New(configManager *config.Manager, httpClient *client.Client, handler ResultHandler) *Scheduler {
	cfg := configManager.GetConfig()

	s := &Scheduler{
		configManager:   configManager,
		client:          httpClient,
		resultHandler:   handler,
		nextRequestTime: make(map[string]time.Time),
		semaphore:       make(chan struct{}, cfg.ConcurrentRequests),
		stopChan:        make(chan struct{}),
		paused:          0, // Start in running state
	}

	// Initialize next request times (all start now)
	now := time.Now()
	for i := range cfg.Endpoints {
		s.nextRequestTime[cfg.Endpoints[i].Name] = now
	}

	return s
}

// NewWithConfig creates a new scheduler with a static config (legacy compatibility)
func NewWithConfig(cfg *config.Config, httpClient *client.Client, handler ResultHandler) *Scheduler {
	// Create a temporary manager with the config
	manager := config.NewManager()
	// Copy config values - this is a simplified approach for backwards compatibility
	manager.SetGlobalMultiplier(cfg.GlobalMultiplier)
	manager.SetConcurrentRequests(cfg.ConcurrentRequests)
	manager.SetLogAllRequests(cfg.LogAllRequests)
	manager.SetAPIPort(cfg.APIPort)
	manager.SetEnabled(cfg.Enabled)

	// Add endpoints
	for _, ep := range cfg.Endpoints {
		manager.AddEndpoint(ep)
	}

	return New(manager, httpClient, handler)
}

// Start begins the load test scheduling loop
func (s *Scheduler) Start(ctx context.Context) error {
	s.runningMu.Lock()
	if s.running {
		s.runningMu.Unlock()
		return fmt.Errorf("scheduler already running")
	}
	s.running = true

	// Create cancellable context for emergency stop
	s.baseCtx = ctx
	s.ctx, s.cancelFunc = context.WithCancel(ctx)
	s.runningMu.Unlock()

	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return s.shutdown()
		case <-s.stopChan:
			return s.shutdown()
		case <-ticker.C:
			s.tick()
		}
	}
}

// tick checks all endpoints and spawns requests for those that are due
func (s *Scheduler) tick() {
	// Check global pause state first (atomic - very fast)
	if s.IsPaused() {
		return
	}

	// Check if globally enabled via config manager
	if !s.configManager.IsEnabled() {
		return
	}

	now := time.Now()
	cfg := s.configManager.GetConfig()

	for i := range cfg.Endpoints {
		endpoint := &cfg.Endpoints[i]

		// Skip disabled endpoints
		if !endpoint.Enabled {
			continue
		}

		s.mu.RLock()
		nextTime, exists := s.nextRequestTime[endpoint.Name]
		s.mu.RUnlock()

		// Initialize next request time for new endpoints
		if !exists {
			s.mu.Lock()
			s.nextRequestTime[endpoint.Name] = now
			s.mu.Unlock()
			nextTime = now
		}

		if now.After(nextTime) || now.Equal(nextTime) {
			// Calculate next request time BEFORE spawning to avoid drift
			interval := s.calculateInterval(endpoint.FrequencyPerMin, cfg.GlobalMultiplier)

			s.mu.Lock()
			s.nextRequestTime[endpoint.Name] = now.Add(interval)
			s.mu.Unlock()

			// Spawn goroutine for request (non-blocking)
			s.wg.Add(1)
			atomic.AddInt64(&s.requestsScheduled, 1)

			// Make a copy of endpoint for the goroutine
			epCopy := *endpoint
			go s.executeRequest(&epCopy)
		}
	}
}

// executeRequest executes a single HTTP request
func (s *Scheduler) executeRequest(endpoint *config.Endpoint) {
	defer s.wg.Done()

	// Check pause state before acquiring semaphore
	if s.IsPaused() || !s.configManager.IsEnabled() {
		atomic.AddInt64(&s.requestsSkipped, 1)
		return
	}

	// Acquire semaphore (blocks if at capacity)
	select {
	case s.semaphore <- struct{}{}:
		// Acquired
	case <-s.ctx.Done():
		// Context cancelled while waiting (emergency stop)
		atomic.AddInt64(&s.requestsSkipped, 1)
		return
	}
	defer func() { <-s.semaphore }()

	// Double-check pause state after acquiring semaphore
	if s.IsPaused() || !s.configManager.IsEnabled() {
		atomic.AddInt64(&s.requestsSkipped, 1)
		return
	}

	// Check if this specific endpoint is still enabled
	enabled, err := s.configManager.IsEndpointEnabled(endpoint.Name)
	if err != nil || !enabled {
		atomic.AddInt64(&s.requestsSkipped, 1)
		return
	}

	atomic.AddInt64(&s.requestsInFlight, 1)
	defer atomic.AddInt64(&s.requestsInFlight, -1)

	// Create timeout context for this specific request
	reqCtx, cancel := context.WithTimeout(s.ctx, time.Duration(endpoint.Timeout)*time.Second)
	defer cancel()

	// Execute the request
	result := s.client.Execute(reqCtx, endpoint)
	if result != nil && result.ErrorType == "cancelled" && !s.IsPaused() && s.configManager.IsEnabled() {
		result.ErrorType = "timeout"
		result.Error = "Request timeout"
	}

	// Report result (non-blocking)
	if s.resultHandler != nil {
		s.resultHandler(result)
	}
}

// calculateInterval calculates the time between requests for an endpoint
func (s *Scheduler) calculateInterval(freqPerMin float64, globalMultiplier float64) time.Duration {
	adjustedFreq := freqPerMin * globalMultiplier
	if adjustedFreq <= 0 {
		return 24 * time.Hour // Very long interval for disabled endpoints
	}
	secondsBetween := 60.0 / adjustedFreq
	return time.Duration(secondsBetween * float64(time.Second))
}

// Stop signals the scheduler to stop gracefully
func (s *Scheduler) Stop() {
	s.runningMu.Lock()
	defer s.runningMu.Unlock()

	if !s.running {
		return
	}

	close(s.stopChan)
}

// EmergencyStop immediately stops all scheduling and cancels in-flight requests
// This is the "big red stop button"
func (s *Scheduler) EmergencyStop() {
	// Set pause state immediately (atomic)
	atomic.StoreInt32(&s.paused, 1)

	// Cancel the context to abort all in-flight requests
	s.runningMu.Lock()
	if s.cancelFunc != nil {
		s.cancelFunc()
	}
	s.runningMu.Unlock()

	// Also disable globally in config
	s.configManager.SetEnabled(false)
}

// Pause pauses scheduling without cancelling in-flight requests
func (s *Scheduler) Pause() {
	atomic.StoreInt32(&s.paused, 1)
	s.configManager.SetEnabled(false)
}

// Resume resumes scheduling after a pause
func (s *Scheduler) Resume() {
	s.runningMu.Lock()
	if s.ctx == nil || s.ctx.Err() != nil {
		fmt.Printf("[scheduler] recreating request context (err=%v)\n", s.ctx.Err())
		parent := s.baseCtx
		if parent == nil {
			parent = context.Background()
		}
		s.ctx, s.cancelFunc = context.WithCancel(parent)
	}
	s.runningMu.Unlock()

	s.configManager.SetEnabled(true)
	atomic.StoreInt32(&s.paused, 0)
}

// IsPaused returns true if the scheduler is paused
func (s *Scheduler) IsPaused() bool {
	return atomic.LoadInt32(&s.paused) == 1
}

// shutdown performs a graceful shutdown
func (s *Scheduler) shutdown() error {
	s.runningMu.Lock()
	s.running = false
	if s.cancelFunc != nil {
		s.cancelFunc()
	}
	s.runningMu.Unlock()

	// Wait for all in-flight requests with timeout
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-time.After(30 * time.Second):
		return fmt.Errorf("shutdown timeout: %d requests still in-flight", atomic.LoadInt64(&s.requestsInFlight))
	}
}

// GetStats returns current scheduler statistics
func (s *Scheduler) GetStats() SchedulerStats {
	cfg := s.configManager.GetConfig()

	// Count enabled endpoints
	enabledCount := 0
	for _, ep := range cfg.Endpoints {
		if ep.Enabled {
			enabledCount++
		}
	}

	return SchedulerStats{
		RequestsScheduled: atomic.LoadInt64(&s.requestsScheduled),
		RequestsInFlight:  atomic.LoadInt64(&s.requestsInFlight),
		RequestsSkipped:   atomic.LoadInt64(&s.requestsSkipped),
		ActiveEndpoints:   len(cfg.Endpoints),
		EnabledEndpoints:  enabledCount,
		Paused:            s.IsPaused(),
		GlobalEnabled:     s.configManager.IsEnabled(),
	}
}

// IsRunning returns true if the scheduler is currently running
func (s *Scheduler) IsRunning() bool {
	s.runningMu.Lock()
	defer s.runningMu.Unlock()
	return s.running
}

// GetConfigManager returns the config manager (for API access)
func (s *Scheduler) GetConfigManager() *config.Manager {
	return s.configManager
}
