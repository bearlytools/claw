package unix

import (
	"bufio"
	"net"
	"os"

	"github.com/gostdlib/base/concurrency/sync"
	"github.com/gostdlib/base/context"

	"github.com/bearlytools/claw/rpc/server"
	"github.com/bearlytools/claw/rpc/transport"
)

// Listener implements transport.Listener for Unix socket connections.
// It accepts incoming Unix socket connections and wraps them in buffered transports.
type Listener struct {
	listener net.Listener
	config   *config
	path     string

	mu     sync.Mutex
	closed bool
}

// Listen creates a new Unix socket listener on the specified path.
//
// Example:
//
//	listener, err := unix.Listen(ctx, "/var/run/myapp.sock")
//	if err != nil {
//	    return err
//	}
//	defer listener.Close()
//
//	for {
//	    trans, err := listener.Accept(ctx)
//	    if err != nil {
//	        break
//	    }
//	    go server.Serve(ctx, trans)
//	}
func Listen(ctx context.Context, path string, opts ...Option) (*Listener, error) {
	cfg := defaultConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	// Remove existing socket file if configured.
	if cfg.unlinkExisting {
		// Check if path exists and is a socket.
		if info, err := os.Stat(path); err == nil {
			if info.Mode()&os.ModeSocket != 0 {
				if err := os.Remove(path); err != nil {
					return nil, err
				}
			}
		}
	}

	// Listen on Unix socket.
	listener, err := net.Listen("unix", path)
	if err != nil {
		return nil, err
	}

	// Set socket file permissions.
	if err := os.Chmod(path, os.FileMode(cfg.socketMode)); err != nil {
		listener.Close()
		os.Remove(path)
		return nil, err
	}

	return &Listener{
		listener: listener,
		config:   cfg,
		path:     path,
	}, nil
}

// Accept waits for and returns the next connection as a transport.
func (l *Listener) Accept(ctx context.Context) (transport.Transport, error) {
	l.mu.Lock()
	if l.closed {
		l.mu.Unlock()
		return nil, ErrClosed
	}
	listener := l.listener
	l.mu.Unlock()

	// Accept with context cancellation support.
	type acceptResult struct {
		conn net.Conn
		err  error
	}
	resultCh := make(chan acceptResult, 1)

	go func() {
		conn, err := listener.Accept()
		resultCh <- acceptResult{conn, err}
	}()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case result := <-resultCh:
		if result.err != nil {
			return nil, result.err
		}
		return newServerTransport(result.conn, l.config), nil
	}
}

// Close closes the listener and removes the socket file.
func (l *Listener) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.closed {
		return nil
	}
	l.closed = true

	err := l.listener.Close()

	// Remove the socket file.
	os.Remove(l.path)

	return err
}

// Addr returns the listener's network address.
func (l *Listener) Addr() net.Addr {
	return l.listener.Addr()
}

// Path returns the socket file path.
func (l *Listener) Path() string {
	return l.path
}

// Verify Listener implements transport.Listener.
var _ transport.Listener = (*Listener)(nil)

// ServerTransport wraps an accepted Unix socket connection with buffered I/O.
type ServerTransport struct {
	conn   net.Conn
	config *config

	// Read state - protected by readMu.
	readMu sync.Mutex
	reader *bufio.Reader

	// Write state - protected by writeMu.
	writeMu sync.Mutex
	writer  *bufio.Writer

	// Connection state.
	connMu sync.Mutex
	closed bool
}

// newServerTransport creates a new server-side transport from an accepted connection.
func newServerTransport(conn net.Conn, cfg *config) *ServerTransport {
	return &ServerTransport{
		conn:   conn,
		config: cfg,
		reader: bufio.NewReaderSize(conn, cfg.readBufferSize),
		writer: bufio.NewWriterSize(conn, cfg.writeBufferSize),
	}
}

// Read reads data from the client.
// This method is safe to call concurrently with Write.
func (t *ServerTransport) Read(p []byte) (int, error) {
	t.connMu.Lock()
	if t.closed {
		t.connMu.Unlock()
		return 0, ErrClosed
	}
	t.connMu.Unlock()

	t.readMu.Lock()
	reader := t.reader
	t.readMu.Unlock()

	if reader == nil {
		return 0, ErrClosed
	}

	return reader.Read(p)
}

// Write writes data to the client.
// Data is buffered and flushed after each write to ensure
// complete messages are sent promptly.
// This method is safe to call concurrently with Read.
func (t *ServerTransport) Write(p []byte) (int, error) {
	t.connMu.Lock()
	if t.closed {
		t.connMu.Unlock()
		return 0, ErrClosed
	}
	t.connMu.Unlock()

	t.writeMu.Lock()
	defer t.writeMu.Unlock()

	if t.writer == nil {
		return 0, ErrClosed
	}

	n, err := t.writer.Write(p)
	if err != nil {
		return n, err
	}

	// Flush after write to ensure data is sent promptly.
	if err := t.writer.Flush(); err != nil {
		return n, err
	}

	return n, nil
}

// Close closes the transport.
func (t *ServerTransport) Close() error {
	t.connMu.Lock()
	defer t.connMu.Unlock()

	if t.closed {
		return nil
	}
	t.closed = true

	// Flush any buffered data before closing.
	t.writeMu.Lock()
	if t.writer != nil {
		t.writer.Flush() // Best effort.
		t.writer = nil
	}
	t.writeMu.Unlock()

	t.readMu.Lock()
	t.reader = nil
	t.readMu.Unlock()

	return t.conn.Close()
}

// LocalAddr returns the local network address.
func (t *ServerTransport) LocalAddr() net.Addr {
	return t.conn.LocalAddr()
}

// RemoteAddr returns the remote network address.
func (t *ServerTransport) RemoteAddr() net.Addr {
	return t.conn.RemoteAddr()
}

// Verify ServerTransport implements transport.Transport.
var _ transport.Transport = (*ServerTransport)(nil)

// Server is a high-level Unix socket server that manages a listener and handles
// incoming connections using a goroutine pool. It is similar in design
// to Go's http.Server.
//
// Example:
//
//	rpcSrv := server.New()
//	rpcSrv.Register("myapp", "UserService", "GetUser", handler)
//
//	unixSrv := unix.NewServer(rpcSrv, "/var/run/myapp.sock")
//	if err := unixSrv.ListenAndServe(ctx); err != nil {
//	    log.Fatal(err)
//	}
type Server struct {
	rpcServer *server.Server
	path      string
	config    *config

	mu       sync.Mutex
	listener *Listener
	closed   bool
}

// NewServer creates a new Unix socket server that will serve RPC connections.
// The server does not start listening until ListenAndServe or Serve is called.
func NewServer(rpcServer *server.Server, path string, opts ...Option) *Server {
	cfg := defaultConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	return &Server{
		rpcServer: rpcServer,
		path:      path,
		config:    cfg,
	}
}

// ListenAndServe creates a Unix socket listener on the configured path and
// starts accepting connections. It blocks until the server is closed
// or an error occurs.
//
// Connections are handled in goroutines from the context's pool.
func (s *Server) ListenAndServe(ctx context.Context) error {
	listener, err := Listen(ctx, s.path,
		WithSocketMode(s.config.socketMode),
		WithUnlinkExisting(s.config.unlinkExisting),
		WithReadBufferSize(s.config.readBufferSize),
		WithWriteBufferSize(s.config.writeBufferSize),
	)
	if err != nil {
		return err
	}

	return s.Serve(ctx, listener)
}

// Serve accepts connections on the provided listener and handles them.
// It blocks until the server is closed or an error occurs.
//
// Connections are handled in goroutines from the context's pool.
func (s *Server) Serve(ctx context.Context, listener *Listener) error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		listener.Close()
		return ErrClosed
	}
	s.listener = listener
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		s.listener = nil
		s.mu.Unlock()
		listener.Close()
	}()

	pool := context.Pool(ctx)

	for {
		trans, err := listener.Accept(ctx)
		if err != nil {
			s.mu.Lock()
			closed := s.closed
			s.mu.Unlock()

			if closed {
				return nil
			}
			return err
		}

		pool.Submit(ctx, func() {
			s.rpcServer.Serve(ctx, trans)
		})
	}
}

// Shutdown gracefully shuts down the server. It first closes the listener
// to stop accepting new connections, then waits for all existing connections
// to complete by calling Shutdown on the underlying RPC server.
//
// If the context is cancelled, Shutdown returns the context error.
func (s *Server) Shutdown(ctx context.Context) error {
	s.mu.Lock()
	s.closed = true
	listener := s.listener
	s.mu.Unlock()

	if listener != nil {
		listener.Close()
	}

	return s.rpcServer.Shutdown(ctx)
}

// Close immediately closes the server and all connections.
// For graceful shutdown, use Shutdown instead.
func (s *Server) Close() error {
	s.mu.Lock()
	s.closed = true
	listener := s.listener
	s.mu.Unlock()

	if listener != nil {
		return listener.Close()
	}
	return nil
}

// Addr returns the listener's address, or nil if not listening.
func (s *Server) Addr() net.Addr {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.listener != nil {
		return s.listener.Addr()
	}
	return nil
}

// Path returns the configured socket path.
func (s *Server) Path() string {
	return s.path
}
