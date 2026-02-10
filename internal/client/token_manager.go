// Package client provides HTTP client functionality with DNS timing
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"moxapp/internal/config"
)

// ManagedToken represents a token with its lifecycle state
type ManagedToken struct {
	Value       string
	ExpiresAt   time.Time
	RefreshAt   time.Time
	LastRefresh time.Time
	LastError   error
	ErrorCount  int
	mu          sync.RWMutex
}

// TokenManager manages JWT tokens with automatic refresh
type TokenManager struct {
	tokens            map[string]*ManagedToken // authConfigName -> token
	authConfigs       map[string]*config.AuthConfig
	httpClient        *http.Client
	envGetter         EnvGetter
	mu                sync.RWMutex
	refreshInterval   time.Duration
	stopChan          chan struct{}
	backgroundRunning bool
}

// TokenStatus provides information about a token's current state
type TokenStatus struct {
	HasToken     bool   `json:"has_token"`
	ExpiresAt    string `json:"expires_at,omitempty"`
	RefreshAt    string `json:"refresh_at,omitempty"`
	LastRefresh  string `json:"last_refresh,omitempty"`
	LastError    string `json:"last_error,omitempty"`
	ErrorCount   int    `json:"error_count"`
	IsExpired    bool   `json:"is_expired"`
	NeedsRefresh bool   `json:"needs_refresh"`
}

// NewTokenManager creates a new token manager
func NewTokenManager(authConfigs map[string]*config.AuthConfig, envGetter EnvGetter) *TokenManager {
	return &TokenManager{
		tokens:          make(map[string]*ManagedToken),
		authConfigs:     authConfigs,
		httpClient:      &http.Client{Timeout: 30 * time.Second},
		envGetter:       envGetter,
		refreshInterval: 30 * time.Second,
		stopChan:        make(chan struct{}),
	}
}

// GetToken returns the current token for an auth config, refreshing if needed
func (tm *TokenManager) GetToken(ctx context.Context, authName string) (string, error) {
	tm.mu.RLock()
	authCfg := tm.authConfigs[authName]
	token := tm.tokens[authName]
	tm.mu.RUnlock()

	if authCfg == nil {
		return "", fmt.Errorf("auth config not found: %s", authName)
	}

	// Static token from env var (no refresh needed)
	if authCfg.TokenEndpoint == nil {
		if authCfg.EnvVar == "" {
			return "", nil
		}
		return tm.envGetter.GetEnv(authCfg.EnvVar), nil
	}

	// Dynamic token - check if refresh needed
	if token == nil || time.Now().After(token.RefreshAt) {
		return tm.refreshToken(ctx, authName, authCfg)
	}

	token.mu.RLock()
	defer token.mu.RUnlock()
	return token.Value, nil
}

// refreshToken fetches a new token from the token endpoint with retry logic
func (tm *TokenManager) refreshToken(ctx context.Context, authName string, cfg *config.AuthConfig) (string, error) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	// Check if another goroutine already refreshed
	if token := tm.tokens[authName]; token != nil {
		token.mu.RLock()
		if time.Now().Before(token.RefreshAt) {
			value := token.Value
			token.mu.RUnlock()
			return value, nil
		}
		token.mu.RUnlock()
	}

	// Try to refresh with retries
	var lastErr error
	retryDelays := []time.Duration{1 * time.Second, 2 * time.Second, 3 * time.Second}

	for attempt := 0; attempt <= 3; attempt++ {
		if attempt > 0 {
			// Wait before retry
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(retryDelays[attempt-1]):
			}
			log.Printf("Retrying token refresh for %s (attempt %d/3)", authName, attempt)
		}

		tokenValue, expiresAt, err := tm.fetchToken(ctx, cfg)
		if err == nil {
			// Success - store token
			refreshBeforeExpiry := time.Duration(cfg.RefreshBeforeExpiry) * time.Second
			if refreshBeforeExpiry == 0 {
				refreshBeforeExpiry = 60 * time.Second
			}

			newToken := &ManagedToken{
				Value:       tokenValue,
				ExpiresAt:   expiresAt,
				RefreshAt:   expiresAt.Add(-refreshBeforeExpiry),
				LastRefresh: time.Now(),
				ErrorCount:  0,
			}

			tm.tokens[authName] = newToken
			log.Printf("Successfully refreshed token for %s (expires at %s)", authName, expiresAt.Format(time.RFC3339))
			return tokenValue, nil
		}

		lastErr = err
		log.Printf("Failed to refresh token for %s: %v", authName, err)
	}

	// All retries failed - keep existing token if available
	if existingToken := tm.tokens[authName]; existingToken != nil {
		existingToken.mu.Lock()
		existingToken.LastError = lastErr
		existingToken.ErrorCount++
		value := existingToken.Value
		existingToken.mu.Unlock()

		log.Printf("Token refresh failed for %s after 3 retries, keeping existing token (error count: %d)", authName, existingToken.ErrorCount)
		return value, nil
	}

	return "", fmt.Errorf("failed to refresh token after 3 retries: %w", lastErr)
}

// fetchToken makes a single attempt to fetch a token from the token endpoint
func (tm *TokenManager) fetchToken(ctx context.Context, cfg *config.AuthConfig) (string, time.Time, error) {
	endpoint := cfg.TokenEndpoint
	if endpoint == nil {
		return "", time.Time{}, fmt.Errorf("no token endpoint configured")
	}

	// Build URL
	url := endpoint.URL
	if endpoint.URLEnv != "" {
		url = tm.envGetter.GetEnv(endpoint.URLEnv)
	}
	if url == "" {
		return "", time.Time{}, fmt.Errorf("token endpoint URL not configured")
	}

	// Build request body (evaluate templates if needed)
	var bodyReader io.Reader
	if endpoint.Body != nil {
		evaluatedBody, err := config.EvaluateBodyTemplate(endpoint.Body)
		if err != nil {
			return "", time.Time{}, fmt.Errorf("failed to evaluate body template: %w", err)
		}

		bodyBytes, err := json.Marshal(evaluatedBody)
		if err != nil {
			return "", time.Time{}, fmt.Errorf("failed to marshal body: %w", err)
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	// Create request
	method := endpoint.Method
	if method == "" {
		method = "POST"
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	for key, value := range endpoint.Headers {
		req.Header.Set(key, value)
	}

	// Set credentials (basic auth if provided)
	if endpoint.UsernameEnv != "" && endpoint.PasswordEnv != "" {
		username := tm.envGetter.GetEnv(endpoint.UsernameEnv)
		password := tm.envGetter.GetEnv(endpoint.PasswordEnv)
		req.SetBasicAuth(username, password)
	}

	// Execute request
	resp, err := tm.httpClient.Do(req)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", time.Time{}, fmt.Errorf("token endpoint returned status %d: %s", resp.StatusCode, string(respBody))
	}

	// Parse JSON response
	var respData map[string]interface{}
	if err := json.Unmarshal(respBody, &respData); err != nil {
		return "", time.Time{}, fmt.Errorf("failed to parse JSON response: %w", err)
	}

	// Extract token using path
	tokenValue, err := config.ExtractJSONPath(respData, endpoint.TokenPath)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to extract token from response: %w", err)
	}

	tokenStr, ok := tokenValue.(string)
	if !ok {
		return "", time.Time{}, fmt.Errorf("token value is not a string: %T", tokenValue)
	}

	// Extract expiry if configured
	var expiresAt time.Time
	if endpoint.ExpiresPath != "" {
		expiresValue, err := config.ExtractJSONPath(respData, endpoint.ExpiresPath)
		if err != nil {
			// Default to 1 hour if expiry not found
			log.Printf("Warning: Could not extract expiry for %s: %v, defaulting to 1 hour", cfg.Name, err)
			expiresAt = time.Now().Add(1 * time.Hour)
		} else {
			// Try to parse as seconds (int or float) or timestamp
			switch v := expiresValue.(type) {
			case float64:
				if v > 1000000000000 { // Timestamp in milliseconds
					expiresAt = time.Unix(0, int64(v)*int64(time.Millisecond))
				} else if v > 1000000000 { // Timestamp in seconds
					expiresAt = time.Unix(int64(v), 0)
				} else { // Seconds from now
					expiresAt = time.Now().Add(time.Duration(v) * time.Second)
				}
			case int:
				expiresAt = time.Now().Add(time.Duration(v) * time.Second)
			default:
				log.Printf("Warning: Unrecognized expiry format for %s: %T, defaulting to 1 hour", cfg.Name, v)
				expiresAt = time.Now().Add(1 * time.Hour)
			}
		}
	} else {
		// Default to 1 hour if no expiry path configured
		expiresAt = time.Now().Add(1 * time.Hour)
	}

	return tokenStr, expiresAt, nil
}

// SetToken manually sets a token (for API updates)
func (tm *TokenManager) SetToken(authName, token string, expiresIn time.Duration) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if _, exists := tm.authConfigs[authName]; !exists {
		return fmt.Errorf("auth config not found: %s", authName)
	}

	expiresAt := time.Now().Add(expiresIn)
	refreshAt := expiresAt.Add(-60 * time.Second)

	tm.tokens[authName] = &ManagedToken{
		Value:       token,
		ExpiresAt:   expiresAt,
		RefreshAt:   refreshAt,
		LastRefresh: time.Now(),
		ErrorCount:  0,
	}

	return nil
}

// ForceRefresh forces an immediate token refresh
func (tm *TokenManager) ForceRefresh(ctx context.Context, authName string) error {
	tm.mu.RLock()
	authCfg := tm.authConfigs[authName]
	tm.mu.RUnlock()

	if authCfg == nil {
		return fmt.Errorf("auth config not found: %s", authName)
	}

	if authCfg.TokenEndpoint == nil {
		return fmt.Errorf("auth config %s does not have a token endpoint", authName)
	}

	_, err := tm.refreshToken(ctx, authName, authCfg)
	return err
}

// GetTokenStatus returns the status of a token
func (tm *TokenManager) GetTokenStatus(authName string) *TokenStatus {
	tm.mu.RLock()
	token := tm.tokens[authName]
	authCfg := tm.authConfigs[authName]
	tm.mu.RUnlock()

	status := &TokenStatus{
		HasToken: token != nil,
	}

	if token != nil {
		token.mu.RLock()
		defer token.mu.RUnlock()

		status.ExpiresAt = token.ExpiresAt.Format(time.RFC3339)
		status.RefreshAt = token.RefreshAt.Format(time.RFC3339)
		status.LastRefresh = token.LastRefresh.Format(time.RFC3339)
		status.ErrorCount = token.ErrorCount
		status.IsExpired = time.Now().After(token.ExpiresAt)
		status.NeedsRefresh = time.Now().After(token.RefreshAt)

		if token.LastError != nil {
			status.LastError = token.LastError.Error()
		}
	}

	// Check if this auth config has a token endpoint
	if authCfg != nil && authCfg.TokenEndpoint == nil {
		// Static token from env - always available
		status.HasToken = true
	}

	return status
}

// UpdateAuthConfigs updates the auth configs (called when config is reloaded)
func (tm *TokenManager) UpdateAuthConfigs(configs map[string]*config.AuthConfig) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	tm.authConfigs = configs
}

// StartBackgroundRefresh starts a goroutine that proactively refreshes tokens
func (tm *TokenManager) StartBackgroundRefresh(ctx context.Context) {
	tm.mu.Lock()
	if tm.backgroundRunning {
		tm.mu.Unlock()
		return
	}
	tm.backgroundRunning = true
	tm.mu.Unlock()

	go func() {
		ticker := time.NewTicker(tm.refreshInterval)
		defer ticker.Stop()

		log.Printf("Token manager background refresh started (interval: %s)", tm.refreshInterval)

		for {
			select {
			case <-ctx.Done():
				log.Println("Token manager background refresh stopped")
				return
			case <-tm.stopChan:
				log.Println("Token manager background refresh stopped")
				return
			case <-ticker.C:
				tm.refreshExpiringTokens(ctx)
			}
		}
	}()
}

// StopBackgroundRefresh stops the background refresh goroutine
func (tm *TokenManager) StopBackgroundRefresh() {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if tm.backgroundRunning {
		close(tm.stopChan)
		tm.backgroundRunning = false
	}
}

// refreshExpiringTokens checks all tokens and refreshes those approaching expiry
func (tm *TokenManager) refreshExpiringTokens(ctx context.Context) {
	tm.mu.RLock()
	authConfigsSnapshot := make(map[string]*config.AuthConfig)
	tokensSnapshot := make(map[string]*ManagedToken)
	for name, cfg := range tm.authConfigs {
		authConfigsSnapshot[name] = cfg
	}
	for name, token := range tm.tokens {
		tokensSnapshot[name] = token
	}
	tm.mu.RUnlock()

	for authName, authCfg := range authConfigsSnapshot {
		if authCfg.TokenEndpoint == nil {
			continue
		}

		token, exists := tokensSnapshot[authName]
		if !exists {
			continue
		}

		token.mu.RLock()
		needsRefresh := time.Now().After(token.RefreshAt)
		token.mu.RUnlock()

		if needsRefresh {
			log.Printf("Background refresh triggered for %s", authName)
			_, _ = tm.refreshToken(ctx, authName, authCfg)
		}
	}
}

// GetEnv is a helper to access environment variables (implements EnvGetter for itself)
func (tm *TokenManager) GetEnv(key string) string {
	if tm.envGetter == nil {
		return ""
	}
	return tm.envGetter.GetEnv(key)
}
