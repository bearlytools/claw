package tcp

import (
	"bufio"
	"crypto/tls"
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

// ClientTransport implements transport.Transport over TCP.
// It uses buffered I/O for improved performance.
type ClientTransport struct {
	addr    string
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

// Dial creates a new TCP transport connection to the specified address.
// The address should be in the form "host:port".
//
// Example:
//
//	transport, err := tcp.Dial(ctx, "localhost:8080")
//	if err != nil {
//	    return err
//	}
//	defer transport.Close()
//	conn := client.New(ctx, transport)
func Dial(ctx context.Context, addr string, opts ...Option) (*ClientTransport, error) {
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
		addr:    addr,
		config:  cfg,
		backoff: backoff,
	}

	// Establish initial connection.
	if err := t.connect(ctx); err != nil {
		return nil, err
	}

	return t, nil
}

// connect establishes the TCP connection.
func (t *ClientTransport) connect(ctx context.Context) error {
	t.connMu.Lock()
	if t.closed {
		t.connMu.Unlock()
		return ErrClosed
	}

	// Clean up existing connection if any.
	t.cleanupLocked()
	t.connMu.Unlock()

	// Create dialer with timeout and keep-alive.
	dialer := &net.Dialer{
		Timeout:   t.config.dialTimeout,
		KeepAlive: t.config.keepAlive,
	}

	var conn net.Conn
	var err error

	if t.config.tlsConfig != nil {
		// TLS connection.
		conn, err = tls.DialWithDialer(dialer, "tcp", t.addr, t.config.tlsConfig)
	} else {
		// Plain TCP connection.
		conn, err = dialer.DialContext(ctx, "tcp", t.addr)
	}

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
	// These must be protected by their respective mutexes when used.
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
		// We need to be careful here - the writer might be in use.
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
	// This is important for RPC where we want messages sent promptly.
	// The buffer still helps by coalescing small writes within a single
	// Write() call and reducing syscall overhead.
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
	return &net.TCPAddr{}
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

// Dialer implements transport.Dialer for TCP connections.
type Dialer struct {
	addr   string
	config *config
}

// NewDialer creates a new TCP dialer.
func NewDialer(addr string, opts ...Option) *Dialer {
	cfg := defaultConfig()
	for _, opt := range opts {
		opt(cfg)
	}
	return &Dialer{addr: addr, config: cfg}
}

// Dial establishes a new TCP transport connection.
func (d *Dialer) Dial(ctx context.Context) (transport.Transport, error) {
	return Dial(ctx, d.addr,
		WithTLSConfig(d.config.tlsConfig),
		WithRetryPolicy(d.config.retryPolicy),
		WithDialTimeout(d.config.dialTimeout),
		WithReadBufferSize(d.config.readBufferSize),
		WithWriteBufferSize(d.config.writeBufferSize),
		WithKeepAlive(d.config.keepAlive),
	)
}

// Verify Dialer implements transport.Dialer.
var _ transport.Dialer = (*Dialer)(nil)
