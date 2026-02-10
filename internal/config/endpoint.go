// Package config handles configuration loading and endpoint definitions
package config

import (
	"fmt"
	"net/url"

	"gopkg.in/yaml.v3"
)

// Endpoint represents a single API endpoint to be load tested
type Endpoint struct {
	Name            string            `mapstructure:"name" yaml:"name" json:"name"`
	Method          string            `mapstructure:"method" yaml:"method" json:"method"`
	URLTemplate     string            `mapstructure:"url_template" yaml:"url_template" json:"url_template"`
	ConfigPath      string            `mapstructure:"config_path" yaml:"config_path,omitempty" json:"config_path,omitempty"`
	FrequencyPerMin float64           `mapstructure:"frequency" yaml:"frequency" json:"frequency"`
	Auth            interface{}       `mapstructure:"auth" yaml:"auth" json:"auth"` // string ref or inline object
	ResolvedAuth    *AuthConfig       `mapstructure:"-" yaml:"-" json:"-"`          // Resolved at load time
	Headers         map[string]string `mapstructure:"headers" yaml:"headers,omitempty" json:"headers,omitempty"`
	Body            interface{}       `mapstructure:"body" yaml:"body,omitempty" json:"body,omitempty"`
	Timeout         int               `mapstructure:"timeout" yaml:"timeout" json:"timeout"`
	Enabled         bool              `mapstructure:"enabled" yaml:"enabled" json:"enabled"`
	EnabledSet      bool              `mapstructure:"enabled" yaml:"-" json:"-"`
}

// UnmarshalYAML implements custom YAML parsing to detect explicit enabled field
func (e *Endpoint) UnmarshalYAML(value *yaml.Node) error {
	var raw struct {
		Name        string            `yaml:"name"`
		Method      string            `yaml:"method"`
		URLTemplate string            `yaml:"url_template"`
		ConfigPath  string            `yaml:"config_path"`
		Frequency   float64           `yaml:"frequency"`
		Auth        interface{}       `yaml:"auth"`
		Headers     map[string]string `yaml:"headers"`
		Body        interface{}       `yaml:"body"`
		Timeout     int               `yaml:"timeout"`
		Enabled     *bool             `yaml:"enabled"`
	}

	if err := value.Decode(&raw); err != nil {
		return err
	}

	e.Name = raw.Name
	e.Method = raw.Method
	e.URLTemplate = raw.URLTemplate
	e.ConfigPath = raw.ConfigPath
	e.FrequencyPerMin = raw.Frequency
	e.Auth = raw.Auth
	e.Headers = raw.Headers
	e.Body = raw.Body
	e.Timeout = raw.Timeout
	if raw.Enabled != nil {
		e.Enabled = *raw.Enabled
		e.EnabledSet = true
	}

	return nil
}

// Validate checks if the endpoint configuration is valid
func (e *Endpoint) Validate() []string {
	var errors []string

	if e.Name == "" {
		errors = append(errors, "name is required")
	}

	if e.Method == "" {
		errors = append(errors, fmt.Sprintf("endpoint %s: method is required", e.Name))
	} else {
		validMethods := map[string]bool{"GET": true, "POST": true, "PUT": true, "DELETE": true, "PATCH": true, "HEAD": true, "OPTIONS": true}
		if !validMethods[e.Method] {
			errors = append(errors, fmt.Sprintf("endpoint %s: invalid method %s", e.Name, e.Method))
		}
	}

	if e.URLTemplate == "" && e.ConfigPath == "" {
		errors = append(errors, fmt.Sprintf("endpoint %s: url_template or config_path is required", e.Name))
	}

	if e.FrequencyPerMin < 0 {
		errors = append(errors, fmt.Sprintf("endpoint %s: frequency must be non-negative", e.Name))
	}

	if e.Timeout <= 0 {
		errors = append(errors, fmt.Sprintf("endpoint %s: timeout must be positive", e.Name))
	}

	return errors
}

// GetHostname extracts the hostname from the URL template
func (e *Endpoint) GetHostname() string {
	// Try to parse the URL template (may contain template variables)
	parsedURL, err := url.Parse(e.URLTemplate)
	if err != nil {
		return ""
	}
	return parsedURL.Hostname()
}

// Clone creates a deep copy of the endpoint
func (e *Endpoint) Clone() Endpoint {
	clone := *e
	if e.Headers != nil {
		clone.Headers = make(map[string]string)
		for k, v := range e.Headers {
			clone.Headers[k] = v
		}
	}
	return clone
}

// EndpointRequest represents a request to create or update an endpoint
type EndpointRequest struct {
	Name            string            `json:"name"`
	Method          string            `json:"method"`
	URLTemplate     string            `json:"url_template"`
	ConfigPath      string            `json:"config_path,omitempty"`
	FrequencyPerMin float64           `json:"frequency"`
	Auth            interface{}       `json:"auth,omitempty"`
	Headers         map[string]string `json:"headers,omitempty"`
	Body            interface{}       `json:"body,omitempty"`
	Timeout         int               `json:"timeout,omitempty"`
	Enabled         bool              `json:"enabled"`
}

// ToEndpoint converts an EndpointRequest to an Endpoint
func (r *EndpointRequest) ToEndpoint() Endpoint {
	return Endpoint{
		Name:            r.Name,
		Method:          r.Method,
		URLTemplate:     r.URLTemplate,
		ConfigPath:      r.ConfigPath,
		FrequencyPerMin: r.FrequencyPerMin,
		Auth:            r.Auth,
		Headers:         r.Headers,
		Body:            r.Body,
		Timeout:         r.Timeout,
		Enabled:         r.Enabled,
		EnabledSet:      true,
	}
}
