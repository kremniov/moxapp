// Package client provides HTTP client functionality with DNS timing
package client

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http/httptrace"
	"net/url"
	"strings"
	"time"
)

// TimingInfo holds the timing information for a request
type TimingInfo struct {
	DNSStart     time.Time
	DNSDone      time.Time
	ConnectStart time.Time
	ConnectDone  time.Time
	TLSStart     time.Time
	TLSDone      time.Time
	FirstByte    time.Time
	RequestStart time.Time
	RequestDone  time.Time

	DNSError     error
	ConnectError error
}

// DNSTimeMs returns the DNS resolution time in milliseconds
func (t *TimingInfo) DNSTimeMs() float64 {
	if t.DNSDone.IsZero() || t.DNSStart.IsZero() {
		return 0
	}
	return float64(t.DNSDone.Sub(t.DNSStart).Microseconds()) / 1000.0
}

// ConnectTimeMs returns the TCP connect time in milliseconds
func (t *TimingInfo) ConnectTimeMs() float64 {
	if t.ConnectDone.IsZero() || t.ConnectStart.IsZero() {
		return 0
	}
	return float64(t.ConnectDone.Sub(t.ConnectStart).Microseconds()) / 1000.0
}

// TLSTimeMs returns the TLS handshake time in milliseconds
func (t *TimingInfo) TLSTimeMs() float64 {
	if t.TLSDone.IsZero() || t.TLSStart.IsZero() {
		return 0
	}
	return float64(t.TLSDone.Sub(t.TLSStart).Microseconds()) / 1000.0
}

// TimeToFirstByteMs returns the time to first byte in milliseconds
func (t *TimingInfo) TimeToFirstByteMs() float64 {
	if t.FirstByte.IsZero() || t.RequestStart.IsZero() {
		return 0
	}
	return float64(t.FirstByte.Sub(t.RequestStart).Microseconds()) / 1000.0
}

// CreateClientTrace creates an httptrace.ClientTrace that populates TimingInfo
func CreateClientTrace(timing *TimingInfo) *httptrace.ClientTrace {
	return &httptrace.ClientTrace{
		DNSStart: func(info httptrace.DNSStartInfo) {
			timing.DNSStart = time.Now()
		},
		DNSDone: func(info httptrace.DNSDoneInfo) {
			timing.DNSDone = time.Now()
			timing.DNSError = info.Err
		},
		ConnectStart: func(network, addr string) {
			timing.ConnectStart = time.Now()
		},
		ConnectDone: func(network, addr string, err error) {
			timing.ConnectDone = time.Now()
			timing.ConnectError = err
		},
		TLSHandshakeStart: func() {
			timing.TLSStart = time.Now()
		},
		TLSHandshakeDone: func(state tls.ConnectionState, err error) {
			timing.TLSDone = time.Now()
		},
		GotFirstResponseByte: func() {
			timing.FirstByte = time.Now()
		},
	}
}

// ExtractHostname extracts the hostname from a URL
func ExtractHostname(rawURL string) string {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		// Try to extract hostname manually
		rawURL = strings.TrimPrefix(rawURL, "http://")
		rawURL = strings.TrimPrefix(rawURL, "https://")
		if idx := strings.Index(rawURL, "/"); idx > 0 {
			rawURL = rawURL[:idx]
		}
		if idx := strings.Index(rawURL, ":"); idx > 0 {
			rawURL = rawURL[:idx]
		}
		return rawURL
	}
	return parsedURL.Hostname()
}

// CategorizeError categorizes an error into a specific type
func CategorizeError(err error) (errorType string, errorMsg string) {
	if err == nil {
		return "", ""
	}

	errStr := err.Error()

	// Check for context errors
	if err == context.DeadlineExceeded || strings.Contains(errStr, "context deadline exceeded") {
		return "timeout", "Request timeout"
	}
	if err == context.Canceled || strings.Contains(errStr, "context canceled") {
		return "cancelled", "Request cancelled"
	}

	// Check for DNS errors
	if strings.Contains(errStr, "no such host") ||
		strings.Contains(errStr, "lookup") ||
		strings.Contains(errStr, "dns") ||
		strings.Contains(errStr, "getaddrinfo") ||
		strings.Contains(errStr, "Temporary failure in name resolution") {
		return "dns", fmt.Sprintf("DNS Error: %s", errStr)
	}

	// Check for connection errors
	if strings.Contains(errStr, "connection refused") ||
		strings.Contains(errStr, "connection reset") ||
		strings.Contains(errStr, "no route to host") ||
		strings.Contains(errStr, "network is unreachable") ||
		strings.Contains(errStr, "dial tcp") {
		return "connection", fmt.Sprintf("Connection Error: %s", errStr)
	}

	// Check for TLS errors
	if strings.Contains(errStr, "tls") ||
		strings.Contains(errStr, "certificate") ||
		strings.Contains(errStr, "x509") {
		return "tls", fmt.Sprintf("TLS Error: %s", errStr)
	}

	// Check for timeout patterns
	if strings.Contains(errStr, "timeout") ||
		strings.Contains(errStr, "deadline") {
		return "timeout", fmt.Sprintf("Timeout: %s", errStr)
	}

	// Generic error
	return "unknown", errStr
}

// IsDNSError checks if an error message indicates a DNS error
func IsDNSError(errMsg string) bool {
	if errMsg == "" {
		return false
	}
	errLower := strings.ToLower(errMsg)
	return strings.Contains(errLower, "dns") ||
		strings.Contains(errLower, "no such host") ||
		strings.Contains(errLower, "lookup") ||
		strings.Contains(errLower, "name resolution")
}
