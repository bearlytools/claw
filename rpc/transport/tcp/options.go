package tcp

import (
	"crypto/tls"
	"time"

	"github.com/gostdlib/base/retry/exponential"
)

// config holds configuration for TCP transports.
type config struct {
	// TLS configuration for secure connections.
	// If nil, plain TCP is used.
	tlsConfig *tls.Config

	// Retry policy for reconnection (client only).
	retryPolicy exponential.Policy

	// Dial timeout for connection establishment.
	dialTimeout time.Duration

	// Read buffer size for bufio.Reader.
	readBufferSize int

	// Write buffer size for bufio.Writer.
	writeBufferSize int

	// KeepAlive period for TCP connections.
	// Zero means keep-alives are disabled.
	keepAlive time.Duration
}

func defaultConfig() *config {
	return &config{
		retryPolicy:     exponential.FastRetryPolicy(),
		dialTimeout:     30 * time.Second,
		readBufferSize:  64 * 1024,  // 64KB
		writeBufferSize: 64 * 1024,  // 64KB
		keepAlive:       30 * time.Second,
	}
}

// Option configures a TCP transport.
type Option func(*config)

// WithTLSConfig sets the TLS configuration for secure connections.
// If not set, plain TCP without encryption is used.
// For clients, this configures the TLS client handshake.
// For servers, this configures the TLS server handshake.
func WithTLSConfig(cfg *tls.Config) Option {
	return func(c *config) {
		c.tlsConfig = cfg
	}
}

// WithRetryPolicy sets the retry policy for reconnection attempts.
// Only applies to client transports.
// If not set, exponential.FastRetryPolicy() is used.
func WithRetryPolicy(policy exponential.Policy) Option {
	return func(c *config) {
		c.retryPolicy = policy
	}
}

// WithDialTimeout sets the timeout for connection establishment.
// Default is 30 seconds.
func WithDialTimeout(timeout time.Duration) Option {
	return func(c *config) {
		c.dialTimeout = timeout
	}
}

// WithReadBufferSize sets the read buffer size for bufio.Reader.
// Default is 64KB.
func WithReadBufferSize(size int) Option {
	return func(c *config) {
		if size > 0 {
			c.readBufferSize = size
		}
	}
}

// WithWriteBufferSize sets the write buffer size for bufio.Writer.
// Default is 64KB.
func WithWriteBufferSize(size int) Option {
	return func(c *config) {
		if size > 0 {
			c.writeBufferSize = size
		}
	}
}

// WithKeepAlive sets the keep-alive period for TCP connections.
// Default is 30 seconds. Set to zero to disable keep-alives.
func WithKeepAlive(d time.Duration) Option {
	return func(c *config) {
		c.keepAlive = d
	}
}

// DefaultRetryPolicy returns the default retry policy for TCP transports.
func DefaultRetryPolicy() exponential.Policy {
	return exponential.FastRetryPolicy()
}

// SlowRetryPolicy returns a slower retry policy suitable for unreliable networks.
func SlowRetryPolicy() exponential.Policy {
	return exponential.SecondsRetryPolicy()
}
