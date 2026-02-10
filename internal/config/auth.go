// Package config handles configuration loading and endpoint definitions
package config

import (
	"fmt"
	"strings"
)

// Auth type constants
const (
	AuthTypeNone        = "none"
	AuthTypeBearer      = "bearer"
	AuthTypeAPIKey      = "api_key"
	AuthTypeAPIKeyQuery = "api_key_query"
	AuthTypeBasic       = "basic"
	AuthTypeCustom      = "custom_header"
)

// AuthConfig represents a reusable authentication configuration
type AuthConfig struct {
	Name        string `mapstructure:"name" yaml:"name" json:"name"`
	Type        string `mapstructure:"type" yaml:"type" json:"type"`
	Description string `mapstructure:"description" yaml:"description,omitempty" json:"description,omitempty"`

	// For api_key and custom_header types
	HeaderName string `mapstructure:"header_name" yaml:"header_name,omitempty" json:"header_name,omitempty"`

	// For api_key_query type
	QueryParam string `mapstructure:"query_param" yaml:"query_param,omitempty" json:"query_param,omitempty"`

	// Credential sources (env var names, not secret values)
	EnvVar      string `mapstructure:"env_var" yaml:"env_var,omitempty" json:"env_var,omitempty"`
	UsernameEnv string `mapstructure:"username_env" yaml:"username_env,omitempty" json:"username_env,omitempty"`
	PasswordEnv string `mapstructure:"password_env" yaml:"password_env,omitempty" json:"password_env,omitempty"`

	// Token endpoint configuration for JWT/OAuth (bearer type with refresh)
	TokenEndpoint *TokenEndpointConfig `mapstructure:"token_endpoint" yaml:"token_endpoint,omitempty" json:"token_endpoint,omitempty"`

	// Refresh settings (seconds before expiry to refresh token)
	RefreshBeforeExpiry int `mapstructure:"refresh_before_expiry" yaml:"refresh_before_expiry,omitempty" json:"refresh_before_expiry,omitempty"`
}

// TokenEndpointConfig defines how to obtain/refresh a bearer token
type TokenEndpointConfig struct {
	URL         string            `mapstructure:"url" yaml:"url,omitempty" json:"url,omitempty"`
	URLEnv      string            `mapstructure:"url_env" yaml:"url_env,omitempty" json:"url_env,omitempty"`
	Method      string            `mapstructure:"method" yaml:"method,omitempty" json:"method,omitempty"`
	UsernameEnv string            `mapstructure:"username_env" yaml:"username_env,omitempty" json:"username_env,omitempty"`
	PasswordEnv string            `mapstructure:"password_env" yaml:"password_env,omitempty" json:"password_env,omitempty"`
	Headers     map[string]string `mapstructure:"headers" yaml:"headers,omitempty" json:"headers,omitempty"`
	Body        interface{}       `mapstructure:"body" yaml:"body,omitempty" json:"body,omitempty"`
	TokenPath   string            `mapstructure:"token_path" yaml:"token_path,omitempty" json:"token_path,omitempty"`       // JSON path to token in response (e.g., "access_token" or "data.token")
	ExpiresPath string            `mapstructure:"expires_path" yaml:"expires_path,omitempty" json:"expires_path,omitempty"` // JSON path to expiry (seconds or timestamp)
}

// Validate validates an AuthConfig
func (a *AuthConfig) Validate() []string {
	var errors []string

	if a.Name == "" {
		errors = append(errors, "auth config name is required")
	}

	validTypes := map[string]bool{
		AuthTypeNone:        true,
		AuthTypeBearer:      true,
		AuthTypeAPIKey:      true,
		AuthTypeAPIKeyQuery: true,
		AuthTypeBasic:       true,
		AuthTypeCustom:      true,
	}

	if !validTypes[a.Type] {
		errors = append(errors, fmt.Sprintf("auth %s: invalid type '%s' (must be one of: none, bearer, api_key, api_key_query, basic, custom_header)", a.Name, a.Type))
	}

	switch a.Type {
	case AuthTypeAPIKey, AuthTypeCustom:
		if a.HeaderName == "" {
			errors = append(errors, fmt.Sprintf("auth %s: header_name required for type %s", a.Name, a.Type))
		}
		if a.EnvVar == "" && a.TokenEndpoint == nil {
			errors = append(errors, fmt.Sprintf("auth %s: env_var or token_endpoint required", a.Name))
		}

	case AuthTypeAPIKeyQuery:
		if a.QueryParam == "" {
			errors = append(errors, fmt.Sprintf("auth %s: query_param required for api_key_query", a.Name))
		}
		if a.EnvVar == "" {
			errors = append(errors, fmt.Sprintf("auth %s: env_var required for api_key_query", a.Name))
		}

	case AuthTypeBasic:
		if a.UsernameEnv == "" || a.PasswordEnv == "" {
			errors = append(errors, fmt.Sprintf("auth %s: username_env and password_env required for basic auth", a.Name))
		}

	case AuthTypeBearer:
		if a.EnvVar == "" && a.TokenEndpoint == nil {
			errors = append(errors, fmt.Sprintf("auth %s: env_var or token_endpoint required for bearer", a.Name))
		}
		if a.TokenEndpoint != nil {
			errors = append(errors, a.validateTokenEndpoint()...)
		}
	}

	return errors
}

// validateTokenEndpoint validates the token endpoint configuration
func (a *AuthConfig) validateTokenEndpoint() []string {
	var errors []string
	te := a.TokenEndpoint

	if te.URL == "" && te.URLEnv == "" {
		errors = append(errors, fmt.Sprintf("auth %s: token_endpoint.url or token_endpoint.url_env required", a.Name))
	}

	if te.Method == "" {
		errors = append(errors, fmt.Sprintf("auth %s: token_endpoint.method required", a.Name))
	}

	if te.TokenPath == "" {
		errors = append(errors, fmt.Sprintf("auth %s: token_endpoint.token_path required (e.g., 'access_token' or 'data.token')", a.Name))
	}

	return errors
}

// HasTokenEndpoint returns true if this auth config has a token endpoint for auto-refresh
func (a *AuthConfig) HasTokenEndpoint() bool {
	return a.TokenEndpoint != nil
}

// ResolveEndpointAuth resolves an endpoint's auth to a complete AuthConfig
// auth can be:
// - nil/empty -> none
// - string -> reference to named auth config
// - map[string]interface{} -> inline auth config or override
func ResolveEndpointAuth(auth interface{}, configs map[string]*AuthConfig) (*AuthConfig, error) {
	if auth == nil {
		return &AuthConfig{Type: AuthTypeNone}, nil
	}

	// String reference (e.g., "example_api" or "none")
	if ref, ok := auth.(string); ok {
		if ref == "" || ref == AuthTypeNone {
			return &AuthConfig{Type: AuthTypeNone}, nil
		}
		cfg, exists := configs[ref]
		if !exists {
			return nil, fmt.Errorf("auth config not found: %s", ref)
		}
		return cfg, nil
	}

	// Map (inline definition or override)
	if authMap, ok := auth.(map[string]interface{}); ok {
		return parseAuthConfigMap(authMap, configs)
	}

	return nil, fmt.Errorf("invalid auth format: expected string or object, got %T", auth)
}

// parseAuthConfigMap parses an inline auth config or auth with override
func parseAuthConfigMap(authMap map[string]interface{}, configs map[string]*AuthConfig) (*AuthConfig, error) {
	// Check if it's a reference with overrides
	if ref, ok := authMap["ref"].(string); ok && ref != "" {
		baseCfg, exists := configs[ref]
		if !exists {
			return nil, fmt.Errorf("auth config not found: %s", ref)
		}
		// Clone base config
		cfg := *baseCfg
		// Apply overrides
		if headerName, ok := authMap["header_name"].(string); ok {
			cfg.HeaderName = headerName
		}
		if queryParam, ok := authMap["query_param"].(string); ok {
			cfg.QueryParam = queryParam
		}
		return &cfg, nil
	}

	// Inline auth config
	cfg := &AuthConfig{}
	if authType, ok := authMap["type"].(string); ok {
		cfg.Type = authType
	}
	if headerName, ok := authMap["header_name"].(string); ok {
		cfg.HeaderName = headerName
	}
	if queryParam, ok := authMap["query_param"].(string); ok {
		cfg.QueryParam = queryParam
	}
	if envVar, ok := authMap["env_var"].(string); ok {
		cfg.EnvVar = envVar
	}
	if usernameEnv, ok := authMap["username_env"].(string); ok {
		cfg.UsernameEnv = usernameEnv
	}
	if passwordEnv, ok := authMap["password_env"].(string); ok {
		cfg.PasswordEnv = passwordEnv
	}

	if cfg.Type == "" {
		return nil, fmt.Errorf("inline auth config missing required field: type")
	}

	return cfg, nil
}

// ExtractJSONPath extracts a value from nested map using dot-notation path
// Examples: "access_token", "data.token", "result.access_token"
func ExtractJSONPath(data map[string]interface{}, path string) (interface{}, error) {
	if path == "" {
		return nil, fmt.Errorf("json path is empty")
	}

	parts := strings.Split(path, ".")
	current := interface{}(data)

	for i, part := range parts {
		currentMap, ok := current.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("expected object at path segment '%s', got %T", strings.Join(parts[:i], "."), current)
		}

		value, exists := currentMap[part]
		if !exists {
			return nil, fmt.Errorf("path segment '%s' not found", part)
		}

		current = value
	}

	return current, nil
}
