// Package server provides RPC server functionality for multiplexed connections.
package server

import (
	"errors"
	"io"

	"github.com/gostdlib/base/concurrency/sync"
	"github.com/gostdlib/base/context"

	"github.com/bearlytools/claw/rpc/interceptor"
	"github.com/bearlytools/claw/rpc/internal/msgs"
)

// Common errors.
var (
	ErrClosed          = errors.New("server closed")
	ErrSessionClosed   = errors.New("session closed")
	ErrMessageTooLarge = errors.New("message size exceeds limit")
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
// Default is 4MB (maxPayloadSize).
func WithMaxRecvMsgSize(size int) Option {
	return func(s *Server) {
		s.maxRecvMsgSize = size
	}
}

// WithMaxSendMsgSize sets the maximum size for sent messages.
// Messages larger than this will cause the send to fail with ErrMessageTooLarge.
// Default is 0 (no limit, only protocol max applies).
func WithMaxSendMsgSize(size int) Option {
	return func(s *Server) {
		s.maxSendMsgSize = size
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

	maxRecvMsgSize int // Maximum size of received messages (0 = default 4MB)
	maxSendMsgSize int // Maximum size of sent messages (0 = no limit)
}

// New creates a new RPC server.
func New(opts ...Option) *Server {
	s := &Server{
		registry: NewRegistry(),
		conns:    make(map[*ServerConn]struct{}),
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

// Register registers a handler for a specific package/service/call combination.
func (s *Server) Register(pkg, service, call string, handler Handler) error {
	return s.registry.Register(pkg, service, call, handler)
}

// Serve handles a single connection, spawning session goroutines via context.Pool(ctx).
// This blocks until the connection is closed or an error occurs.
func (s *Server) Serve(ctx context.Context, transport io.ReadWriteCloser) error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		transport.Close()
		return ErrClosed
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
// It closes all active connections and waits for them to finish.
func (s *Server) Shutdown(ctx context.Context) error {
	s.mu.Lock()
	s.closed = true
	conns := make([]*ServerConn, 0, len(s.conns))
	for conn := range s.conns {
		conns = append(conns, conn)
	}
	s.mu.Unlock()

	// Send GoAway to all connections and close them.
	for _, conn := range conns {
		conn.goAway(ctx)
	}

	// Wait for all connections to close or context to be done.
	done := make(chan struct{})
	go func() {
		for _, conn := range conns {
			<-conn.closed
		}
		close(done)
	}()

	select {
	case <-ctx.Done():
		// Force close all connections.
		for _, conn := range conns {
			conn.Close()
		}
		return ctx.Err()
	case <-done:
		return nil
	}
}
