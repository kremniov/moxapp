// Package config handles configuration loading and endpoint definitions
package config

import (
	"fmt"
	"math"
	"strings"

	"gopkg.in/yaml.v3"
)

// IncomingEndpoint represents an incoming route configuration for traffic simulation
type IncomingEndpoint struct {
	Name       string                   `mapstructure:"name" yaml:"name" json:"name"`
	Path       string                   `mapstructure:"path" yaml:"path" json:"path"`
	Method     string                   `mapstructure:"method" yaml:"method" json:"method"`
	Responses  []IncomingResponseConfig `mapstructure:"responses" yaml:"responses" json:"responses"`
	Enabled    bool                     `mapstructure:"enabled" yaml:"enabled" json:"enabled"`
	EnabledSet bool                     `mapstructure:"enabled" yaml:"-" json:"-"`
}

// UnmarshalYAML implements custom YAML parsing to detect explicit enabled field
func (e *IncomingEndpoint) UnmarshalYAML(value *yaml.Node) error {
	var raw struct {
		Name      string                   `yaml:"name"`
		Path      string                   `yaml:"path"`
		Method    string                   `yaml:"method"`
		Responses []IncomingResponseConfig `yaml:"responses"`
		Enabled   *bool                    `yaml:"enabled"`
	}

	if err := value.Decode(&raw); err != nil {
		return err
	}

	e.Name = raw.Name
	e.Path = raw.Path
	e.Method = raw.Method
	e.Responses = raw.Responses
	if raw.Enabled != nil {
		e.Enabled = *raw.Enabled
		e.EnabledSet = true
	}

	return nil
}

// IncomingResponseConfig defines a possible response configuration with probability
type IncomingResponseConfig struct {
	StatusCode    int     `mapstructure:"status" yaml:"status" json:"status"`
	Share         float64 `mapstructure:"share" yaml:"share" json:"share"`
	MinResponseMs int     `mapstructure:"min_response_ms" yaml:"min_response_ms" json:"min_response_ms"`
	MaxResponseMs int     `mapstructure:"max_response_ms" yaml:"max_response_ms" json:"max_response_ms"`
}

// Validate checks if the incoming endpoint configuration is valid
func (e *IncomingEndpoint) Validate() []string {
	var errors []string

	if e.Name == "" {
		errors = append(errors, "name is required")
	}

	if e.Path == "" {
		errors = append(errors, fmt.Sprintf("incoming endpoint %s: path is required", e.Name))
	} else if !strings.HasPrefix(e.Path, "/") {
		errors = append(errors, fmt.Sprintf("incoming endpoint %s: path must start with /", e.Name))
	}

	if e.Method == "" {
		errors = append(errors, fmt.Sprintf("incoming endpoint %s: method is required", e.Name))
	} else if e.Method != "*" {
		validMethods := map[string]bool{"GET": true, "POST": true, "PUT": true, "DELETE": true, "PATCH": true, "HEAD": true, "OPTIONS": true}
		if !validMethods[e.Method] {
			errors = append(errors, fmt.Sprintf("incoming endpoint %s: invalid method %s (use * for any method)", e.Name, e.Method))
		}
	}

	if len(e.Responses) == 0 {
		errors = append(errors, fmt.Sprintf("incoming endpoint %s: at least one response configuration is required", e.Name))
	}

	// Validate response configurations
	var totalShare float64
	for i, resp := range e.Responses {
		respErrors := resp.Validate(e.Name, i)
		errors = append(errors, respErrors...)
		totalShare += resp.Share
	}

	// Check that shares sum to approximately 1.0 (allow small floating point errors)
	if len(e.Responses) > 0 && math.Abs(totalShare-1.0) > 0.001 {
		errors = append(errors, fmt.Sprintf("incoming endpoint %s: response shares must sum to 1.0 (got %.3f)", e.Name, totalShare))
	}

	return errors
}

// Validate checks if the response configuration is valid
func (r *IncomingResponseConfig) Validate(endpointName string, index int) []string {
	var errors []string

	if r.StatusCode < 100 || r.StatusCode > 599 {
		errors = append(errors, fmt.Sprintf("incoming endpoint %s response[%d]: status code must be between 100 and 599", endpointName, index))
	}

	if r.Share < 0 || r.Share > 1 {
		errors = append(errors, fmt.Sprintf("incoming endpoint %s response[%d]: share must be between 0 and 1", endpointName, index))
	}

	if r.MinResponseMs < 0 {
		errors = append(errors, fmt.Sprintf("incoming endpoint %s response[%d]: min_response_ms must be non-negative", endpointName, index))
	}

	if r.MaxResponseMs < 0 {
		errors = append(errors, fmt.Sprintf("incoming endpoint %s response[%d]: max_response_ms must be non-negative", endpointName, index))
	}

	if r.MaxResponseMs < r.MinResponseMs {
		errors = append(errors, fmt.Sprintf("incoming endpoint %s response[%d]: max_response_ms must be >= min_response_ms", endpointName, index))
	}

	return errors
}

// Clone creates a deep copy of the incoming endpoint
func (e *IncomingEndpoint) Clone() IncomingEndpoint {
	clone := *e
	if e.Responses != nil {
		clone.Responses = make([]IncomingResponseConfig, len(e.Responses))
		copy(clone.Responses, e.Responses)
	}
	return clone
}

// IncomingEndpointRequest represents a request to create or update an incoming endpoint
type IncomingEndpointRequest struct {
	Name      string                   `json:"name"`
	Path      string                   `json:"path"`
	Method    string                   `json:"method"`
	Responses []IncomingResponseConfig `json:"responses"`
	Enabled   bool                     `json:"enabled"`
}

// ToIncomingEndpoint converts an IncomingEndpointRequest to an IncomingEndpoint
func (r *IncomingEndpointRequest) ToIncomingEndpoint() IncomingEndpoint {
	return IncomingEndpoint{
		Name:      r.Name,
		Path:      r.Path,
		Method:    r.Method,
		Responses: r.Responses,
		Enabled:   r.Enabled,
	}
}
