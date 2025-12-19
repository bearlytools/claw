// Package server provides RPC server functionality for multiplexed connections.
package server

import (
	"errors"
	"io"

	"github.com/gostdlib/base/concurrency/sync"
	"github.com/gostdlib/base/context"
)

// Common errors.
var (
	ErrClosed        = errors.New("server closed")
	ErrSessionClosed = errors.New("session closed")
)

// Option configures a Server.
type Option func(*Server)

// Server handles RPC connections and dispatches to registered handlers.
type Server struct {
	registry *Registry
	conns    map[*ServerConn]struct{}
	mu       sync.Mutex
	closed   bool
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

	conn := newServerConn(ctx, s, transport)
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
