package http

import (
	"crypto/tls"
	"net/http"
	"time"

	"github.com/gostdlib/base/retry/exponential"
)

// config holds configuration for HTTP transports.
type config struct {
	// TLS configuration for HTTPS connections.
	tlsConfig *tls.Config

	// HTTP client for making requests (client-side only).
	httpClient *http.Client

	// Custom headers to include in requests.
	headers http.Header

	// Retry policy for reconnection.
	retryPolicy exponential.Policy

	// Path for the RPC endpoint (server-side only).
	path string

	// Timeout for initial connection establishment.
	dialTimeout time.Duration
}

func defaultConfig() *config {
	return &config{
		headers:     make(http.Header),
		retryPolicy: exponential.FastRetryPolicy(),
		path:        "/rpc",
		dialTimeout: 30 * time.Second,
	}
}

// Option configures an HTTP transport.
type Option func(*config)

// WithTLSConfig sets the TLS configuration for HTTPS connections.
// If not set, the default TLS configuration is used for HTTPS URLs.
func WithTLSConfig(cfg *tls.Config) Option {
	return func(c *config) {
		c.tlsConfig = cfg
	}
}

// WithHTTPClient sets a custom HTTP client for making requests.
// This allows customization of timeouts, transport settings, etc.
// Only applies to client-side transports.
func WithHTTPClient(client *http.Client) Option {
	return func(c *config) {
		c.httpClient = client
	}
}

// WithHeaders adds custom headers to include in HTTP requests.
// These headers are sent with every request.
func WithHeaders(headers http.Header) Option {
	return func(c *config) {
		for k, v := range headers {
			c.headers[k] = v
		}
	}
}

// WithHeader adds a single header to include in HTTP requests.
func WithHeader(key, value string) Option {
	return func(c *config) {
		c.headers.Set(key, value)
	}
}

// WithRetryPolicy sets the retry policy for reconnection attempts.
// If not set, exponential.FastRetryPolicy() is used.
func WithRetryPolicy(policy exponential.Policy) Option {
	return func(c *config) {
		c.retryPolicy = policy
	}
}

// WithPath sets the RPC endpoint path for the server handler.
// Default is "/rpc".
func WithPath(path string) Option {
	return func(c *config) {
		c.path = path
	}
}

// WithDialTimeout sets the timeout for initial connection establishment.
// Default is 30 seconds.
func WithDialTimeout(timeout time.Duration) Option {
	return func(c *config) {
		c.dialTimeout = timeout
	}
}

// DefaultRetryPolicy returns the default retry policy for HTTP transports.
// Uses exponential backoff starting at 100ms, doubling up to 60s.
func DefaultRetryPolicy() exponential.Policy {
	return exponential.FastRetryPolicy()
}

// SlowRetryPolicy returns a slower retry policy suitable for unreliable networks.
// Uses exponential backoff starting at 1s, doubling up to 60s.
func SlowRetryPolicy() exponential.Policy {
	return exponential.SecondsRetryPolicy()
}
