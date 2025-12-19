package unix

import (
	"time"

	"github.com/gostdlib/base/retry/exponential"
)

// config holds configuration for Unix socket transports.
type config struct {
	// Retry policy for reconnection (client only).
	retryPolicy exponential.Policy

	// Dial timeout for connection establishment.
	dialTimeout time.Duration

	// Read buffer size for bufio.Reader.
	readBufferSize int

	// Write buffer size for bufio.Writer.
	writeBufferSize int

	// File mode for the socket file (server only).
	// Default is 0600 (owner read/write only).
	socketMode uint32

	// Whether to unlink (remove) existing socket file before listening.
	// Default is true.
	unlinkExisting bool
}

func defaultConfig() *config {
	return &config{
		retryPolicy:     exponential.FastRetryPolicy(),
		dialTimeout:     30 * time.Second,
		readBufferSize:  64 * 1024, // 64KB
		writeBufferSize: 64 * 1024, // 64KB
		socketMode:      0600,
		unlinkExisting:  true,
	}
}

// Option configures a Unix socket transport.
type Option func(*config)

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

// WithSocketMode sets the file mode for the socket file.
// Only applies to server/listener. Default is 0600.
func WithSocketMode(mode uint32) Option {
	return func(c *config) {
		c.socketMode = mode
	}
}

// WithUnlinkExisting controls whether to remove an existing socket file
// before listening. Default is true.
// Only applies to server/listener.
func WithUnlinkExisting(unlink bool) Option {
	return func(c *config) {
		c.unlinkExisting = unlink
	}
}

// DefaultRetryPolicy returns the default retry policy for Unix transports.
func DefaultRetryPolicy() exponential.Policy {
	return exponential.FastRetryPolicy()
}
