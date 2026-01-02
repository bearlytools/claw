// Package server provides RPC server functionality for multiplexed connections.
package server

import (
	"io"
	"net"
	stdsync "sync"

	"github.com/gostdlib/base/concurrency/sync"
	"github.com/gostdlib/base/context"
	"github.com/gostdlib/base/values/sizes"

	rpcctx "github.com/bearlytools/claw/rpc/context"
	"github.com/bearlytools/claw/rpc/errors"
	"github.com/bearlytools/claw/rpc/interceptor"
	"github.com/bearlytools/claw/rpc/internal/msgs"
)

// Common errors.
var (
	ErrClosed             = errors.New("server closed")
	ErrSessionClosed      = errors.New("session closed")
	ErrMessageTooLarge    = errors.New("message size exceeds limit")
	ErrTooManyConnections = errors.New("too many connections")
)

// Option configures a Server.
type Option func(*Server)

// WithCompression sets the default compression algorithm for server responses.
// Use msgs.CmpNone to disable compression (default).
func WithCompression(alg msgs.Compression) Option {
	return func(s *Server) {
		s.defaultCompression = alg
	}
}

// WithUnaryInterceptor adds unary interceptors to the server.
// Multiple calls chain the interceptors; they execute in the order provided.
func WithUnaryInterceptor(interceptors ...interceptor.UnaryServerInterceptor) Option {
	return func(s *Server) {
		if s.unaryInterceptor == nil {
			s.unaryInterceptor = interceptor.ChainUnaryServer(interceptors...)
		} else {
			s.unaryInterceptor = interceptor.ChainUnaryServer(append([]interceptor.UnaryServerInterceptor{s.unaryInterceptor}, interceptors...)...)
		}
	}
}

// WithStreamInterceptor adds stream interceptors to the server.
// Multiple calls chain the interceptors; they execute in the order provided.
func WithStreamInterceptor(interceptors ...interceptor.StreamServerInterceptor) Option {
	return func(s *Server) {
		if s.streamInterceptor == nil {
			s.streamInterceptor = interceptor.ChainStreamServer(interceptors...)
		} else {
			s.streamInterceptor = interceptor.ChainStreamServer(append([]interceptor.StreamServerInterceptor{s.streamInterceptor}, interceptors...)...)
		}
	}
}

// WithMaxRecvMsgSize sets the maximum size for received messages.
// Messages larger than this will be rejected with ErrMessageTooLarge.
// Default is 4 MiB.
func WithMaxRecvMsgSize(size int) Option {
	return func(s *Server) {
		s.maxRecvMsgSize = size
	}
}

// WithMaxSendMsgSize sets the maximum size for sent messages.
// Messages larger than this will cause the send to fail with ErrMessageTooLarge.
// Default is 4 MiB.
func WithMaxSendMsgSize(size int) Option {
	return func(s *Server) {
		s.maxSendMsgSize = size
	}
}

// WithMaxConnections sets the maximum number of concurrent connections.
// New connections are rejected with ErrTooManyConnections when at limit.
// Default is 0 (no limit).
func WithMaxConnections(max int) Option {
	return func(s *Server) {
		s.maxConnections = max
	}
}

// WithMaxConcurrentRPCs sets the maximum number of concurrent RPC handlers.
// This uses a limited worker pool to restrict concurrency.
// Default is 0 (no limit, uses the context's pool directly).
func WithMaxConcurrentRPCs(max int) Option {
	return func(s *Server) {
		s.maxConcurrentRPCs = max
	}
}

// WithPacking enables Cap'n Proto-style message packing for connections.
// When enabled, the server will agree to packing if requested by the client.
// Packing can significantly reduce message size by eliminating zero bytes.
func WithPacking(enabled bool) Option {
	return func(s *Server) {
		s.allowPacking = enabled
	}
}

// Server handles RPC connections and dispatches to registered handlers.
type Server struct {
	registry           *Registry
	conns              map[*ServerConn]struct{}
	mu                 sync.Mutex
	closed             bool
	defaultCompression msgs.Compression

	unaryInterceptor  interceptor.UnaryServerInterceptor
	streamInterceptor interceptor.StreamServerInterceptor

	maxRecvMsgSize    int  // Maximum size of received messages (default 4 MiB)
	maxSendMsgSize    int  // Maximum size of sent messages (default 4 MiB)
	maxConnections    int  // Maximum concurrent connections (0 = no limit)
	maxConcurrentRPCs int  // Maximum concurrent RPC handlers (0 = no limit)
	allowPacking      bool // Whether to allow packing if client requests it
}

// New creates a new RPC server.
func New(opts ...Option) *Server {
	s := &Server{
		registry:       NewRegistry(),
		conns:          make(map[*ServerConn]struct{}),
		maxRecvMsgSize: 4 * sizes.MiB,
		maxSendMsgSize: 4 * sizes.MiB,
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

// Register registers a handler for a specific package/service/call combination.
func (s *Server) Register(ctx context.Context, pkg, service, call string, handler Handler) error {
	return s.registry.Register(ctx, pkg, service, call, handler)
}

// Registry returns the server's handler registry.
// This is useful for reflection and introspection of registered services.
func (s *Server) Registry() *Registry {
	return s.registry
}

// Serve handles a single connection, spawning session goroutines via context.Pool(ctx).
// This blocks until the connection is closed or an error occurs.
func (s *Server) Serve(ctx context.Context, transport io.ReadWriteCloser) error {
	// Inject remote IP into context if transport provides it.
	if t, ok := transport.(interface{ RemoteAddr() net.Addr }); ok {
		if addr := t.RemoteAddr(); addr != nil {
			ctx = rpcctx.WithRemoteAddr(ctx, addr)
		}
	}

	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		transport.Close()
		return errors.E(ctx, errors.Unavailable, ErrClosed)
	}

	// Check connection limit.
	if s.maxConnections > 0 && len(s.conns) >= s.maxConnections {
		s.mu.Unlock()
		transport.Close()
		return errors.E(ctx, errors.ResourceExhausted, ErrTooManyConnections)
	}

	conn := newServerConn(ctx, s, transport, s.defaultCompression)
	s.conns[conn] = struct{}{}
	s.mu.Unlock()

	err := conn.serve(ctx)

	s.mu.Lock()
	delete(s.conns, conn)
	s.mu.Unlock()

	return err
}

// Shutdown gracefully shuts down the server.
// It stops accepting new connections, sends GoAway to all active connections,
// waits for in-flight RPCs to complete, then closes all connections.
// The context controls how long to wait; if cancelled, remaining connections
// are forcefully closed.
func (s *Server) Shutdown(ctx context.Context) error {
	s.mu.Lock()
	s.closed = true
	conns := make([]*ServerConn, 0, len(s.conns))
	for conn := range s.conns {
		conns = append(conns, conn)
	}
	s.mu.Unlock()

	if len(conns) == 0 {
		return nil
	}

	// Gracefully close all connections in parallel.
	done := make(chan struct{})
	var lastErr error
	var errMu stdsync.Mutex

	go func() {
		var wg stdsync.WaitGroup
		for _, conn := range conns {
			wg.Add(1)
			conn := conn
			go func() {
				defer wg.Done()
				if err := conn.GracefulClose(ctx); err != nil {
					errMu.Lock()
					lastErr = err
					errMu.Unlock()
				}
			}()
		}
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return lastErr
	case <-ctx.Done():
		// Force close all remaining connections.
		for _, conn := range conns {
			conn.Close()
		}
		return ctx.Err()
	}
}

// IsDraining returns true if the server is shutting down (no new connections accepted).
func (s *Server) IsDraining() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.closed
}
