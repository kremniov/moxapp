// Package config handles configuration loading and endpoint definitions
package config

import (
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/spf13/viper"
)

// Config represents the main application configuration
type Config struct {
	Enabled            bool                   `mapstructure:"enabled" json:"enabled"`
	GlobalMultiplier   float64                `mapstructure:"global_multiplier" json:"global_multiplier"`
	ConcurrentRequests int                    `mapstructure:"concurrent_requests" json:"concurrent_requests"`
	LogAllRequests     bool                   `mapstructure:"log_all_requests" json:"log_all_requests"`
	APIPort            int                    `mapstructure:"api_port" json:"api_port"`
	AuthConfigs        map[string]*AuthConfig `mapstructure:"auth_configs" json:"auth_configs"`
	Endpoints          []Endpoint             `mapstructure:"outgoing_endpoints" json:"outgoing_endpoints"`
	IncomingEnabled    bool                   `mapstructure:"incoming_enabled" json:"incoming_enabled"`
	IncomingRoutes     []IncomingEndpoint     `mapstructure:"incoming_routes" json:"incoming_routes"`

	mu sync.RWMutex `mapstructure:"-" json:"-"`
}

// Manager handles configuration with thread-safe endpoint management
type Manager struct {
	config     *Config
	viper      *viper.Viper
	envViper   *viper.Viper
	configPath string // Path to the config file
	mu         sync.RWMutex
}

// NewManager creates a new configuration manager
func NewManager() *Manager {
	v := viper.New()

	// Set defaults for main config
	v.SetDefault("enabled", true)
	v.SetDefault("global_multiplier", 1.0)
	v.SetDefault("concurrent_requests", 30)
	v.SetDefault("log_all_requests", false)
	v.SetDefault("api_port", 8080)
	v.SetDefault("outgoing_endpoints", []Endpoint{})
	v.SetDefault("incoming_enabled", true)
	v.SetDefault("incoming_routes", []IncomingEndpoint{})

	// Enable environment variable reading for LOADTEST_ prefixed vars
	v.SetEnvPrefix("LOADTEST")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Create a separate viper instance for .env file
	envV := viper.New()
	envV.SetConfigFile(".env")
	envV.SetConfigType("env")

	// Try to read .env file (silently ignore if not found)
	_ = envV.ReadInConfig()

	return &Manager{
		config: &Config{
			Enabled:            true,
			GlobalMultiplier:   1.0,
			ConcurrentRequests: 30,
			APIPort:            8080,
			AuthConfigs:        make(map[string]*AuthConfig),
			Endpoints:          []Endpoint{},
			IncomingEnabled:    true,
			IncomingRoutes:     []IncomingEndpoint{},
		},
		viper:    v,
		envViper: envV,
	}
}

// LoadFromFile loads configuration from a YAML file
func (m *Manager) LoadFromFile(path string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.configPath = path // Store the config path
	m.viper.SetConfigFile(path)
	m.viper.SetConfigType("yaml")

	if err := m.viper.ReadInConfig(); err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	if err := m.viper.Unmarshal(m.config); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Initialize auth configs map if nil
	if m.config.AuthConfigs == nil {
		m.config.AuthConfigs = make(map[string]*AuthConfig)
	}

	// Set names for auth configs based on map keys
	for name, authCfg := range m.config.AuthConfigs {
		authCfg.Name = name
	}

	// Set default values for endpoints and resolve auth
	m.normalizeEndpoints()

	// Normalize incoming routes
	m.normalizeIncomingRoutes()

	return nil
}

// ReplaceConfig replaces the in-memory configuration entirely
func (m *Manager) ReplaceConfig(newCfg *Config) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if newCfg == nil {
		return fmt.Errorf("config cannot be nil")
	}

	// Apply defaults for required fields
	if newCfg.ConcurrentRequests <= 0 {
		newCfg.ConcurrentRequests = 30
	}
	if newCfg.APIPort <= 0 {
		newCfg.APIPort = 8080
	}
	if newCfg.GlobalMultiplier == 0 {
		newCfg.GlobalMultiplier = 1.0
	}
	if newCfg.AuthConfigs == nil {
		newCfg.AuthConfigs = make(map[string]*AuthConfig)
	}

	// Set names for auth configs based on map keys
	for name, authCfg := range newCfg.AuthConfigs {
		authCfg.Name = name
	}

	// Swap config, then normalize (ensures endpoints/routes are valid)
	m.config = newCfg
	m.normalizeEndpoints()
	m.normalizeIncomingRoutes()

	return nil
}

// normalizeEndpoints sets default values for endpoints and resolves auth
func (m *Manager) normalizeEndpoints() {
	for i := range m.config.Endpoints {
		if m.config.Endpoints[i].Timeout == 0 {
			m.config.Endpoints[i].Timeout = 30
		}
		if m.config.Endpoints[i].Method == "" {
			m.config.Endpoints[i].Method = "GET"
		}
		// Default enabled to true when not explicitly set
		if m.config.Endpoints[i].Enabled == false && m.config.Endpoints[i].EnabledSet == false {
			m.config.Endpoints[i].Enabled = true
		}

		// Resolve auth config
		if m.config.Endpoints[i].Auth == nil {
			m.config.Endpoints[i].Auth = "none"
		}
		resolvedAuth, err := ResolveEndpointAuth(m.config.Endpoints[i].Auth, m.config.AuthConfigs)
		if err != nil {
			// Log error but don't fail - set to none
			fmt.Printf("Warning: Failed to resolve auth for endpoint %s: %v\n", m.config.Endpoints[i].Name, err)
			m.config.Endpoints[i].ResolvedAuth = &AuthConfig{Type: AuthTypeNone}
		} else {
			m.config.Endpoints[i].ResolvedAuth = resolvedAuth
		}
	}
}

// normalizeIncomingRoutes sets default values for incoming routes
func (m *Manager) normalizeIncomingRoutes() {
	for i := range m.config.IncomingRoutes {
		if m.config.IncomingRoutes[i].Method == "" {
			m.config.IncomingRoutes[i].Method = "GET"
		}
		if m.config.IncomingRoutes[i].Enabled == false && m.config.IncomingRoutes[i].EnabledSet == false {
			m.config.IncomingRoutes[i].Enabled = true
		}
	}
}

// GetEnv returns an environment variable value from the .env file
func (m *Manager) GetEnv(key string) string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.envViper.GetString(key)
}

// GetAPIPortFromEnv returns the API port from .env file, or default 8080
func (m *Manager) GetAPIPortFromEnv() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if port := m.envViper.GetInt("API_PORT"); port > 0 {
		return port
	}

	return 8080
}

// GetEnvMap returns all environment variables from .env file as a map
func (m *Manager) GetEnvMap() map[string]string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	envMap := make(map[string]string)
	for _, key := range m.envViper.AllKeys() {
		envMap[strings.ToUpper(key)] = m.envViper.GetString(key)
	}
	return envMap
}

// GetConfig returns a copy of the current configuration
func (m *Manager) GetConfig() *Config {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Return a copy to prevent external modification
	cfg := *m.config
	cfg.Endpoints = make([]Endpoint, len(m.config.Endpoints))
	copy(cfg.Endpoints, m.config.Endpoints)
	return &cfg
}

// SetGlobalMultiplier updates the global multiplier
func (m *Manager) SetGlobalMultiplier(multiplier float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.config.GlobalMultiplier = multiplier
}

// SetConcurrentRequests updates the concurrent requests limit
func (m *Manager) SetConcurrentRequests(concurrent int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.config.ConcurrentRequests = concurrent
}

// SetAPIPort updates the API port
func (m *Manager) SetAPIPort(port int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.config.APIPort = port
}

// SetLogAllRequests updates the log all requests setting
func (m *Manager) SetLogAllRequests(log bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.config.LogAllRequests = log
}

// SetEnabled sets the global enabled flag (big red stop button)
func (m *Manager) SetEnabled(enabled bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.config.Enabled = enabled
}

// IsEnabled returns the current global enabled state
func (m *Manager) IsEnabled() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config.Enabled
}

// SetEndpointEnabled enables or disables a specific endpoint
func (m *Manager) SetEndpointEnabled(name string, enabled bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i := range m.config.Endpoints {
		if m.config.Endpoints[i].Name == name {
			m.config.Endpoints[i].Enabled = enabled
			return nil
		}
	}
	return fmt.Errorf("endpoint not found: %s", name)
}

// IsEndpointEnabled returns whether a specific endpoint is enabled
func (m *Manager) IsEndpointEnabled(name string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for i := range m.config.Endpoints {
		if m.config.Endpoints[i].Name == name {
			return m.config.Endpoints[i].Enabled, nil
		}
	}
	return false, fmt.Errorf("endpoint not found: %s", name)
}

// --- Endpoint CRUD Operations ---

// GetEndpoints returns all endpoints
func (m *Manager) GetEndpoints() []Endpoint {
	m.mu.RLock()
	defer m.mu.RUnlock()

	endpoints := make([]Endpoint, len(m.config.Endpoints))
	copy(endpoints, m.config.Endpoints)
	return endpoints
}

// GetEndpoint returns an endpoint by name
func (m *Manager) GetEndpoint(name string) (*Endpoint, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for i := range m.config.Endpoints {
		if m.config.Endpoints[i].Name == name {
			ep := m.config.Endpoints[i] // Copy
			return &ep, nil
		}
	}
	return nil, fmt.Errorf("endpoint not found: %s", name)
}

// AddEndpoint adds a new endpoint
func (m *Manager) AddEndpoint(endpoint Endpoint) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check for duplicate name
	for _, ep := range m.config.Endpoints {
		if ep.Name == endpoint.Name {
			return fmt.Errorf("endpoint already exists: %s", endpoint.Name)
		}
	}

	// Set defaults
	if endpoint.Timeout == 0 {
		endpoint.Timeout = 30
	}
	if endpoint.Auth == nil {
		endpoint.Auth = "none"
	}
	if endpoint.Method == "" {
		endpoint.Method = "GET"
	}

	// Resolve auth
	resolvedAuth, err := ResolveEndpointAuth(endpoint.Auth, m.config.AuthConfigs)
	if err != nil {
		return fmt.Errorf("failed to resolve auth: %w", err)
	}
	endpoint.ResolvedAuth = resolvedAuth

	// Validate
	if errors := endpoint.Validate(); len(errors) > 0 {
		return fmt.Errorf("validation failed: %s", strings.Join(errors, "; "))
	}

	m.config.Endpoints = append(m.config.Endpoints, endpoint)
	return nil
}

// UpdateEndpoint updates an existing endpoint by name
func (m *Manager) UpdateEndpoint(name string, endpoint Endpoint) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i := range m.config.Endpoints {
		if m.config.Endpoints[i].Name == name {
			// If name is being changed, check for duplicate
			if endpoint.Name != name {
				for j, ep := range m.config.Endpoints {
					if ep.Name == endpoint.Name && i != j {
						return fmt.Errorf("endpoint with name %s already exists", endpoint.Name)
					}
				}
			}

			// Set defaults
			if endpoint.Timeout == 0 {
				endpoint.Timeout = 30
			}
			if endpoint.Auth == nil {
				endpoint.Auth = "none"
			}
			if endpoint.Method == "" {
				endpoint.Method = "GET"
			}

			// Resolve auth
			resolvedAuth, err := ResolveEndpointAuth(endpoint.Auth, m.config.AuthConfigs)
			if err != nil {
				return fmt.Errorf("failed to resolve auth: %w", err)
			}
			endpoint.ResolvedAuth = resolvedAuth

			// Validate
			if errors := endpoint.Validate(); len(errors) > 0 {
				return fmt.Errorf("validation failed: %s", strings.Join(errors, "; "))
			}

			m.config.Endpoints[i] = endpoint
			return nil
		}
	}
	return fmt.Errorf("endpoint not found: %s", name)
}

// DeleteEndpoint removes an endpoint by name
func (m *Manager) DeleteEndpoint(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i := range m.config.Endpoints {
		if m.config.Endpoints[i].Name == name {
			// Remove endpoint by swapping with last and truncating
			m.config.Endpoints = append(m.config.Endpoints[:i], m.config.Endpoints[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("endpoint not found: %s", name)
}

// FilterEndpoints returns endpoints matching the given filter patterns
func (m *Manager) FilterEndpoints(filter string) []Endpoint {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if filter == "" {
		endpoints := make([]Endpoint, len(m.config.Endpoints))
		copy(endpoints, m.config.Endpoints)
		return endpoints
	}

	patterns := strings.Split(filter, ",")
	var filtered []Endpoint

	for _, ep := range m.config.Endpoints {
		for _, pattern := range patterns {
			pattern = strings.TrimSpace(pattern)
			if pattern == "" {
				continue
			}
			// Simple substring matching
			if strings.Contains(strings.ToLower(ep.Name), strings.ToLower(pattern)) {
				filtered = append(filtered, ep)
				break
			}
		}
	}

	return filtered
}

// --- Auth Config CRUD Operations ---

// GetAuthConfigs returns all auth configs
func (m *Manager) GetAuthConfigs() map[string]*AuthConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()

	configs := make(map[string]*AuthConfig)
	for name, cfg := range m.config.AuthConfigs {
		configCopy := *cfg
		configs[name] = &configCopy
	}
	return configs
}

// GetAuthConfig returns an auth config by name
func (m *Manager) GetAuthConfig(name string) (*AuthConfig, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	cfg, exists := m.config.AuthConfigs[name]
	if !exists {
		return nil, fmt.Errorf("auth config not found: %s", name)
	}

	configCopy := *cfg
	return &configCopy, nil
}

// AddAuthConfig adds a new auth config
func (m *Manager) AddAuthConfig(authCfg *AuthConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if authCfg.Name == "" {
		return fmt.Errorf("auth config name is required")
	}

	// Check for duplicate name
	if _, exists := m.config.AuthConfigs[authCfg.Name]; exists {
		return fmt.Errorf("auth config already exists: %s", authCfg.Name)
	}

	// Validate
	if errors := authCfg.Validate(); len(errors) > 0 {
		return fmt.Errorf("validation failed: %s", strings.Join(errors, "; "))
	}

	m.config.AuthConfigs[authCfg.Name] = authCfg
	return nil
}

// UpdateAuthConfig updates an existing auth config
func (m *Manager) UpdateAuthConfig(name string, authCfg *AuthConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.config.AuthConfigs[name]; !exists {
		return fmt.Errorf("auth config not found: %s", name)
	}

	// If name is being changed, check for duplicate
	if authCfg.Name != name {
		if _, exists := m.config.AuthConfigs[authCfg.Name]; exists {
			return fmt.Errorf("auth config with name %s already exists", authCfg.Name)
		}
		// Remove old name
		delete(m.config.AuthConfigs, name)
	}

	// Validate
	if errors := authCfg.Validate(); len(errors) > 0 {
		return fmt.Errorf("validation failed: %s", strings.Join(errors, "; "))
	}

	m.config.AuthConfigs[authCfg.Name] = authCfg
	return nil
}

// DeleteAuthConfig removes an auth config by name
func (m *Manager) DeleteAuthConfig(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.config.AuthConfigs[name]; !exists {
		return fmt.Errorf("auth config not found: %s", name)
	}

	// Check if any endpoint is using this auth config
	for _, ep := range m.config.Endpoints {
		if authRef, ok := ep.Auth.(string); ok && authRef == name {
			return fmt.Errorf("cannot delete auth config %s: used by endpoint %s", name, ep.Name)
		}
	}

	delete(m.config.AuthConfigs, name)
	return nil
}

// --- Incoming Routes Management ---

// IsIncomingEnabled returns whether incoming routes are globally enabled
func (m *Manager) IsIncomingEnabled() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config.IncomingEnabled
}

// SetIncomingEnabled sets the global enabled flag for incoming routes
func (m *Manager) SetIncomingEnabled(enabled bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.config.IncomingEnabled = enabled
}

// GetIncomingRoutes returns all incoming routes
func (m *Manager) GetIncomingRoutes() []IncomingEndpoint {
	m.mu.RLock()
	defer m.mu.RUnlock()

	routes := make([]IncomingEndpoint, len(m.config.IncomingRoutes))
	for i, route := range m.config.IncomingRoutes {
		routes[i] = route.Clone()
	}
	return routes
}

// GetIncomingRoute returns an incoming route by name
func (m *Manager) GetIncomingRoute(name string) (*IncomingEndpoint, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, route := range m.config.IncomingRoutes {
		if route.Name == name {
			routeCopy := route.Clone()
			return &routeCopy, nil
		}
	}
	return nil, fmt.Errorf("incoming route not found: %s", name)
}

// AddIncomingRoute adds a new incoming route
func (m *Manager) AddIncomingRoute(route IncomingEndpoint) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check for duplicate name
	for _, r := range m.config.IncomingRoutes {
		if r.Name == route.Name {
			return fmt.Errorf("incoming route already exists: %s", route.Name)
		}
	}

	// Set defaults
	if route.Method == "" {
		route.Method = "GET"
	}

	// Validate
	if errors := route.Validate(); len(errors) > 0 {
		return fmt.Errorf("validation failed: %s", strings.Join(errors, "; "))
	}

	m.config.IncomingRoutes = append(m.config.IncomingRoutes, route)
	return nil
}

// UpdateIncomingRoute updates an existing incoming route by name
func (m *Manager) UpdateIncomingRoute(name string, route IncomingEndpoint) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i := range m.config.IncomingRoutes {
		if m.config.IncomingRoutes[i].Name == name {
			// If name is being changed, check for duplicate
			if route.Name != name {
				for j, r := range m.config.IncomingRoutes {
					if r.Name == route.Name && i != j {
						return fmt.Errorf("incoming route with name %s already exists", route.Name)
					}
				}
			}

			// Set defaults
			if route.Method == "" {
				route.Method = "GET"
			}

			// Validate
			if errors := route.Validate(); len(errors) > 0 {
				return fmt.Errorf("validation failed: %s", strings.Join(errors, "; "))
			}

			m.config.IncomingRoutes[i] = route
			return nil
		}
	}
	return fmt.Errorf("incoming route not found: %s", name)
}

// DeleteIncomingRoute removes an incoming route by name
func (m *Manager) DeleteIncomingRoute(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i := range m.config.IncomingRoutes {
		if m.config.IncomingRoutes[i].Name == name {
			m.config.IncomingRoutes = append(m.config.IncomingRoutes[:i], m.config.IncomingRoutes[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("incoming route not found: %s", name)
}

// SetIncomingRouteEnabled enables or disables a specific incoming route
func (m *Manager) SetIncomingRouteEnabled(name string, enabled bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i := range m.config.IncomingRoutes {
		if m.config.IncomingRoutes[i].Name == name {
			m.config.IncomingRoutes[i].Enabled = enabled
			return nil
		}
	}
	return fmt.Errorf("incoming route not found: %s", name)
}

// MatchIncomingRoute finds the best matching route for a given path and method
// Returns the matched route, the path suffix (portion after matched prefix), and whether a match was found
func (m *Manager) MatchIncomingRoute(path, method string) (*IncomingEndpoint, string, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.config.IncomingEnabled {
		return nil, "", false
	}

	// Build sorted routes for prefix matching (longest first) on-the-fly
	// For better performance, could cache this
	sortedRoutes := make([]IncomingEndpoint, len(m.config.IncomingRoutes))
	copy(sortedRoutes, m.config.IncomingRoutes)

	// Sort by path length descending (longest prefix first)
	sort.Slice(sortedRoutes, func(i, j int) bool {
		return len(sortedRoutes[i].Path) > len(sortedRoutes[j].Path)
	})

	// Try to match against sorted routes
	for _, route := range sortedRoutes {
		if !route.Enabled {
			continue
		}

		// Check if method matches
		if route.Method != "*" && route.Method != method {
			continue
		}

		// Check if path matches (prefix matching)
		if strings.HasPrefix(path, route.Path) {
			// Get the suffix (remainder after prefix)
			suffix := strings.TrimPrefix(path, route.Path)

			// Ensure we're matching at a path boundary
			if suffix == "" || strings.HasPrefix(suffix, "/") {
				routeCopy := route.Clone()
				return &routeCopy, suffix, true
			}
		}
	}

	return nil, "", false
}

// GetIncomingRouteCount returns the number of configured incoming routes
func (m *Manager) GetIncomingRouteCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.config.IncomingRoutes)
}

// GetEnabledIncomingRouteCount returns the number of enabled incoming routes
func (m *Manager) GetEnabledIncomingRouteCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	count := 0
	for _, route := range m.config.IncomingRoutes {
		if route.Enabled {
			count++
		}
	}
	return count
}

// GetConfigPath returns the path to the loaded config file
func (m *Manager) GetConfigPath() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.configPath
}

// --- Statistics ---

// GetTotalBaseRequestsPerMin returns the sum of all endpoint frequencies
func (m *Manager) GetTotalBaseRequestsPerMin() float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var total float64
	for _, ep := range m.config.Endpoints {
		total += ep.FrequencyPerMin
	}
	return total
}

// GetAdjustedRequestsPerMin returns the total requests per minute after applying multiplier
func (m *Manager) GetAdjustedRequestsPerMin() float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var total float64
	for _, ep := range m.config.Endpoints {
		total += ep.FrequencyPerMin
	}
	return total * m.config.GlobalMultiplier
}

// --- Validation ---

// Validate validates the entire configuration
func (m *Manager) Validate() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var errors []string

	if m.config.GlobalMultiplier < 0 {
		errors = append(errors, "global_multiplier must be non-negative")
	}

	if m.config.ConcurrentRequests <= 0 {
		errors = append(errors, "concurrent_requests must be positive")
	}

	if len(m.config.Endpoints) == 0 {
		errors = append(errors, "at least one endpoint must be defined")
	}

	// Check for duplicate endpoint names
	seen := make(map[string]bool)
	for _, ep := range m.config.Endpoints {
		if seen[ep.Name] {
			errors = append(errors, fmt.Sprintf("duplicate endpoint name: %s", ep.Name))
		}
		seen[ep.Name] = true

		// Validate each endpoint
		epErrors := ep.Validate()
		errors = append(errors, epErrors...)
	}

	return errors
}
