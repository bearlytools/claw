package unix

import (
	"bufio"
	"errors"
	"net"

	"github.com/gostdlib/base/concurrency/sync"
	"github.com/gostdlib/base/context"
	"github.com/gostdlib/base/retry/exponential"

	"github.com/bearlytools/claw/rpc/transport"
)

// Common errors.
var (
	ErrClosed       = errors.New("transport closed")
	ErrNotConnected = errors.New("not connected")
)

// ClientTransport implements transport.Transport over Unix domain sockets.
// It uses buffered I/O for improved performance.
type ClientTransport struct {
	path    string
	config  *config
	backoff *exponential.Backoff

	// Read state - protected by readMu.
	// bufio.Reader is not thread-safe, so we need a mutex.
	readMu sync.Mutex
	reader *bufio.Reader

	// Write state - protected by writeMu.
	// bufio.Writer is not thread-safe, so we need a mutex.
	writeMu sync.Mutex
	writer  *bufio.Writer

	// Connection state - protected by connMu.
	connMu    sync.Mutex
	conn      net.Conn
	connected bool
	closed    bool
	connErr   error
}

// Dial creates a new Unix socket transport connection to the specified path.
//
// Example:
//
//	transport, err := unix.Dial(ctx, "/var/run/myapp.sock")
//	if err != nil {
//	    return err
//	}
//	defer transport.Close()
//	conn := client.New(ctx, transport)
func Dial(ctx context.Context, path string, opts ...Option) (*ClientTransport, error) {
	cfg := defaultConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	// Create backoff for reconnection.
	backoff, err := exponential.New(exponential.WithPolicy(cfg.retryPolicy))
	if err != nil {
		return nil, err
	}

	t := &ClientTransport{
		path:    path,
		config:  cfg,
		backoff: backoff,
	}

	// Establish initial connection.
	if err := t.connect(ctx); err != nil {
		return nil, err
	}

	return t, nil
}

// connect establishes the Unix socket connection.
func (t *ClientTransport) connect(ctx context.Context) error {
	t.connMu.Lock()
	if t.closed {
		t.connMu.Unlock()
		return ErrClosed
	}

	// Clean up existing connection if any.
	t.cleanupLocked()
	t.connMu.Unlock()

	// Create dialer with timeout.
	dialer := &net.Dialer{
		Timeout: t.config.dialTimeout,
	}

	conn, err := dialer.DialContext(ctx, "unix", t.path)
	if err != nil {
		return err
	}

	// Setup buffered I/O.
	t.connMu.Lock()
	defer t.connMu.Unlock()

	// Check if closed while connecting.
	if t.closed {
		conn.Close()
		return ErrClosed
	}

	t.conn = conn
	t.connected = true
	t.connErr = nil

	// Create new buffered reader/writer.
	t.readMu.Lock()
	t.reader = bufio.NewReaderSize(conn, t.config.readBufferSize)
	t.readMu.Unlock()

	t.writeMu.Lock()
	t.writer = bufio.NewWriterSize(conn, t.config.writeBufferSize)
	t.writeMu.Unlock()

	return nil
}

// cleanupLocked cleans up the current connection. Must hold t.connMu.
func (t *ClientTransport) cleanupLocked() {
	t.connected = false

	if t.conn != nil {
		// Attempt to flush any buffered data before closing.
		t.writeMu.Lock()
		if t.writer != nil {
			t.writer.Flush() // Best effort, ignore errors.
			t.writer = nil
		}
		t.writeMu.Unlock()

		t.readMu.Lock()
		t.reader = nil
		t.readMu.Unlock()

		t.conn.Close()
		t.conn = nil
	}
}

// Read reads data from the server.
// This method is safe to call concurrently with Write.
func (t *ClientTransport) Read(p []byte) (int, error) {
	t.connMu.Lock()
	if t.closed {
		t.connMu.Unlock()
		return 0, ErrClosed
	}
	if !t.connected {
		t.connMu.Unlock()
		return 0, ErrNotConnected
	}
	t.connMu.Unlock()

	// Lock for reading - bufio.Reader is not thread-safe.
	t.readMu.Lock()
	reader := t.reader
	t.readMu.Unlock()

	if reader == nil {
		return 0, ErrNotConnected
	}

	n, err := reader.Read(p)
	if err != nil {
		t.connMu.Lock()
		t.connErr = err
		t.connMu.Unlock()
	}
	return n, err
}

// Write writes data to the server.
// Data is buffered and flushed after each write to ensure
// complete messages are sent promptly.
// This method is safe to call concurrently with Read.
func (t *ClientTransport) Write(p []byte) (int, error) {
	t.connMu.Lock()
	if t.closed {
		t.connMu.Unlock()
		return 0, ErrClosed
	}
	if !t.connected {
		t.connMu.Unlock()
		return 0, ErrNotConnected
	}
	t.connMu.Unlock()

	// Lock for writing - bufio.Writer is not thread-safe.
	t.writeMu.Lock()
	defer t.writeMu.Unlock()

	if t.writer == nil {
		return 0, ErrNotConnected
	}

	n, err := t.writer.Write(p)
	if err != nil {
		t.connMu.Lock()
		t.connErr = err
		t.connMu.Unlock()
		return n, err
	}

	// Flush after write to ensure data is sent.
	if err := t.writer.Flush(); err != nil {
		t.connMu.Lock()
		t.connErr = err
		t.connMu.Unlock()
		return n, err
	}

	return n, nil
}

// Close closes the transport.
func (t *ClientTransport) Close() error {
	t.connMu.Lock()
	defer t.connMu.Unlock()

	if t.closed {
		return nil
	}
	t.closed = true
	t.cleanupLocked()

	return nil
}

// LocalAddr returns the local network address.
func (t *ClientTransport) LocalAddr() net.Addr {
	t.connMu.Lock()
	defer t.connMu.Unlock()

	if t.conn != nil {
		return t.conn.LocalAddr()
	}
	return nil
}

// RemoteAddr returns the remote network address.
func (t *ClientTransport) RemoteAddr() net.Addr {
	t.connMu.Lock()
	defer t.connMu.Unlock()

	if t.conn != nil {
		return t.conn.RemoteAddr()
	}
	return &net.UnixAddr{Name: t.path, Net: "unix"}
}

// Reconnect attempts to reconnect with exponential backoff.
func (t *ClientTransport) Reconnect(ctx context.Context) error {
	t.connMu.Lock()
	if t.closed {
		t.connMu.Unlock()
		return ErrClosed
	}
	t.connMu.Unlock()

	return t.backoff.Retry(ctx, func(retryCtx context.Context, r exponential.Record) error {
		return t.connect(retryCtx)
	})
}

// Err returns any connection error.
func (t *ClientTransport) Err() error {
	t.connMu.Lock()
	defer t.connMu.Unlock()
	return t.connErr
}

// Connected returns true if the transport is connected.
func (t *ClientTransport) Connected() bool {
	t.connMu.Lock()
	defer t.connMu.Unlock()
	return t.connected && !t.closed
}

// Verify ClientTransport implements transport.Transport.
var _ transport.Transport = (*ClientTransport)(nil)

// Dialer implements transport.Dialer for Unix socket connections.
type Dialer struct {
	path   string
	config *config
}

// NewDialer creates a new Unix socket dialer.
func NewDialer(path string, opts ...Option) *Dialer {
	cfg := defaultConfig()
	for _, opt := range opts {
		opt(cfg)
	}
	return &Dialer{path: path, config: cfg}
}

// Dial establishes a new Unix socket transport connection.
func (d *Dialer) Dial(ctx context.Context) (transport.Transport, error) {
	return Dial(ctx, d.path,
		WithRetryPolicy(d.config.retryPolicy),
		WithDialTimeout(d.config.dialTimeout),
		WithReadBufferSize(d.config.readBufferSize),
		WithWriteBufferSize(d.config.writeBufferSize),
	)
}

// Verify Dialer implements transport.Dialer.
var _ transport.Dialer = (*Dialer)(nil)

// NewResolvingDialer creates a Unix socket dialer with name resolution.
// The target is parsed according to the scheme://authority/endpoint format.
// If no scheme is specified, "passthrough" is used.
//
// For Unix sockets, the endpoint is the socket path.
//
// Example targets:
//   - "passthrough:////var/run/app.sock" - direct path
//   - "/var/run/app.sock" - direct path (uses passthrough)
//
// This function requires importing the resolver packages you want to use:
//
//	import _ "github.com/bearlytools/claw/rpc/transport/resolver/passthrough"
func NewResolvingDialer(target string, opts ...Option) (*transport.ResolvingDialer, error) {
	cfg := defaultConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	dialFunc := func(ctx context.Context, addr string) (transport.Transport, error) {
		return Dial(ctx, addr,
			WithRetryPolicy(cfg.retryPolicy),
			WithDialTimeout(cfg.dialTimeout),
			WithReadBufferSize(cfg.readBufferSize),
			WithWriteBufferSize(cfg.writeBufferSize),
		)
	}

	return transport.NewResolvingDialer(target, dialFunc)
}
