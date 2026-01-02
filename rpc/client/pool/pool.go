package pool

import (
	"errors"
	"fmt"

	"github.com/gostdlib/base/concurrency/sync"
	"github.com/gostdlib/base/context"

	"github.com/bearlytools/claw/rpc/client"
	"github.com/bearlytools/claw/rpc/transport"
	"github.com/bearlytools/claw/rpc/transport/resolver"
)

// Common errors for Pool.
var (
	ErrPoolClosed    = errors.New("pool is closed")
	ErrNoAddresses   = errors.New("resolver returned no addresses")
	ErrResolverNil   = errors.New("resolver is nil")
)

// Pool manages connections to multiple backend addresses with load balancing.
// It provides the same RPC interface as client.Conn but distributes calls
// across multiple connections.
type Pool struct {
	target   string
	cfg      *config
	dialFunc transport.DialFunc
	resolver resolver.Resolver

	subConns      map[string]*SubConn // addr -> SubConn
	readySubConns []*SubConn          // only ready SubConns
	mu            sync.Mutex

	// readyBroadcast is closed when a new SubConn becomes ready.
	// After closing, it is replaced with a new channel for future broadcasts.
	// This allows multiple goroutines to wait for ready SubConns.
	readyBroadcast chan struct{}

	closed chan struct{}
	ctx    context.Context
}

// New creates a new connection pool.
// The target is parsed according to the scheme://authority/endpoint format.
// If no scheme is specified, "passthrough" is used.
//
// Example targets:
//   - "dns:///myservice.namespace:8080"
//   - "passthrough:///localhost:8080"
//   - "localhost:8080" (uses passthrough)
func New(ctx context.Context, target string, dialFunc transport.DialFunc, opts ...Option) (*Pool, error) {
	cfg := defaultConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	var r resolver.Resolver
	if cfg.resolver != nil {
		r = cfg.resolver
	} else {
		// Parse target and build resolver
		t, err := resolver.Parse(target)
		if err != nil {
			return nil, fmt.Errorf("parse target: %w", err)
		}

		b, ok := resolver.Get(t.Scheme)
		if !ok {
			return nil, fmt.Errorf("unknown resolver scheme: %s", t.Scheme)
		}

		r, err = b.Build(t, resolver.BuildOptions{})
		if err != nil {
			return nil, fmt.Errorf("build resolver: %w", err)
		}
	}

	p := &Pool{
		target:         target,
		cfg:            cfg,
		dialFunc:       dialFunc,
		resolver:       r,
		subConns:       make(map[string]*SubConn),
		readyBroadcast: make(chan struct{}),
		closed:         make(chan struct{}),
		ctx:            ctx,
	}

	// Initial address resolution
	if err := p.resolveAndUpdate(ctx); err != nil {
		r.Close()
		return nil, err
	}

	// Start health checker
	p.startHealthChecker(ctx)

	return p, nil
}

// resolveAndUpdate resolves addresses and updates SubConns.
func (p *Pool) resolveAndUpdate(ctx context.Context) error {
	addrs, err := p.resolver.Resolve(ctx)
	if err != nil {
		return fmt.Errorf("resolve addresses: %w", err)
	}

	if len(addrs) == 0 {
		return ErrNoAddresses
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	// Track which addresses are still valid
	validAddrs := make(map[string]bool)
	for _, addr := range addrs {
		validAddrs[addr.Addr] = true
	}

	// Remove SubConns for addresses no longer in the list
	for addr, sc := range p.subConns {
		if !validAddrs[addr] {
			sc.shutdown()
			delete(p.subConns, addr)
		}
	}

	// Add SubConns for new addresses
	for _, addr := range addrs {
		if _, exists := p.subConns[addr.Addr]; !exists {
			sc := newSubConn(addr, p.dialFunc, p.cfg.clientOpts)
			p.subConns[addr.Addr] = sc
			sc.Connect(ctx)
		}
	}

	return nil
}

// getSubConn picks a ready SubConn using the configured balancer.
func (p *Pool) getSubConn() (*SubConn, error) {
	p.mu.Lock()
	ready := p.readySubConns
	p.mu.Unlock()

	if len(ready) == 0 {
		return nil, ErrNoReadySubConns
	}

	return p.cfg.balancer.Pick(ready)
}

// getSubConnWait picks a ready SubConn, optionally waiting for one to become ready.
// If waitForReady is false, it behaves like getSubConn and fails immediately.
// If waitForReady is true, it blocks until a SubConn is ready or context is done.
func (p *Pool) getSubConnWait(ctx context.Context, waitForReady bool) (*SubConn, error) {
	for {
		// Check if pool is closed
		select {
		case <-p.closed:
			return nil, ErrPoolClosed
		default:
		}

		// Try to get a ready SubConn
		p.mu.Lock()
		ready := p.readySubConns
		broadcast := p.readyBroadcast
		p.mu.Unlock()

		if len(ready) > 0 {
			return p.cfg.balancer.Pick(ready)
		}

		// No ready SubConns - fail fast or wait
		if !waitForReady {
			return nil, ErrNoReadySubConns
		}

		// Wait for a SubConn to become ready
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-p.closed:
			return nil, ErrPoolClosed
		case <-broadcast:
			// A SubConn became ready, loop back to try again
		}
	}
}

// Sync creates a new synchronous RPC client.
// The connection is picked from the pool using the configured balancer.
// If WithWaitForReady(true) is passed, blocks until a connection is ready.
func (p *Pool) Sync(ctx context.Context, pkg, service, call string, opts ...client.CallOption) (*client.SyncClient, error) {
	waitForReady := client.GetWaitForReady(opts...)

	sc, err := p.getSubConnWait(ctx, waitForReady)
	if err != nil {
		return nil, err
	}

	conn := sc.Conn()
	if conn == nil {
		return nil, ErrSubConnNotReady
	}

	syncClient, err := conn.Sync(ctx, pkg, service, call, opts...)
	if err != nil {
		// Check if this is a connection failure
		if isConnectionError(err) {
			sc.handleConnectionFailure(ctx, err)
			p.updateReadySubConns()
		}
		return nil, err
	}

	return syncClient, nil
}

// BiDir creates a new bidirectional streaming RPC client.
// The connection is picked from the pool using the configured balancer.
// If WithWaitForReady(true) is passed, blocks until a connection is ready.
func (p *Pool) BiDir(ctx context.Context, pkg, service, call string, opts ...client.CallOption) (*client.BiDirClient, error) {
	waitForReady := client.GetWaitForReady(opts...)

	sc, err := p.getSubConnWait(ctx, waitForReady)
	if err != nil {
		return nil, err
	}

	conn := sc.Conn()
	if conn == nil {
		return nil, ErrSubConnNotReady
	}

	biDirClient, err := conn.BiDir(ctx, pkg, service, call, opts...)
	if err != nil {
		if isConnectionError(err) {
			sc.handleConnectionFailure(ctx, err)
			p.updateReadySubConns()
		}
		return nil, err
	}

	return biDirClient, nil
}

// Send creates a new send-only streaming RPC client.
// The connection is picked from the pool using the configured balancer.
// If WithWaitForReady(true) is passed, blocks until a connection is ready.
func (p *Pool) Send(ctx context.Context, pkg, service, call string, opts ...client.CallOption) (*client.SendClient, error) {
	waitForReady := client.GetWaitForReady(opts...)

	sc, err := p.getSubConnWait(ctx, waitForReady)
	if err != nil {
		return nil, err
	}

	conn := sc.Conn()
	if conn == nil {
		return nil, ErrSubConnNotReady
	}

	sendClient, err := conn.Send(ctx, pkg, service, call, opts...)
	if err != nil {
		if isConnectionError(err) {
			sc.handleConnectionFailure(ctx, err)
			p.updateReadySubConns()
		}
		return nil, err
	}

	return sendClient, nil
}

// Recv creates a new receive-only streaming RPC client.
// The connection is picked from the pool using the configured balancer.
// If WithWaitForReady(true) is passed, blocks until a connection is ready.
func (p *Pool) Recv(ctx context.Context, pkg, service, call string, opts ...client.CallOption) (*client.RecvClient, error) {
	waitForReady := client.GetWaitForReady(opts...)

	sc, err := p.getSubConnWait(ctx, waitForReady)
	if err != nil {
		return nil, err
	}

	conn := sc.Conn()
	if conn == nil {
		return nil, ErrSubConnNotReady
	}

	recvClient, err := conn.Recv(ctx, pkg, service, call, opts...)
	if err != nil {
		if isConnectionError(err) {
			sc.handleConnectionFailure(ctx, err)
			p.updateReadySubConns()
		}
		return nil, err
	}

	return recvClient, nil
}

// Close closes the pool and all connections immediately.
// For graceful shutdown that waits for in-flight RPCs, use GracefulClose.
func (p *Pool) Close() error {
	select {
	case <-p.closed:
		return nil
	default:
		close(p.closed)
	}

	p.mu.Lock()
	for _, sc := range p.subConns {
		sc.shutdown()
	}
	p.subConns = nil
	p.readySubConns = nil
	p.mu.Unlock()

	return p.resolver.Close()
}

// GracefulClose stops accepting new RPCs and waits for in-flight RPCs on all
// SubConns to complete before closing. The context controls how long to wait.
//
// Returns nil if all connections closed gracefully, or an error if the context
// was cancelled before all connections finished draining.
func (p *Pool) GracefulClose(ctx context.Context) error {
	select {
	case <-p.closed:
		return nil
	default:
		close(p.closed)
	}

	p.mu.Lock()
	subConns := make([]*SubConn, 0, len(p.subConns))
	for _, sc := range p.subConns {
		subConns = append(subConns, sc)
	}
	p.mu.Unlock()

	// Gracefully close all SubConns in parallel
	var lastErr error
	done := make(chan struct{})

	go func() {
		for _, sc := range subConns {
			if err := sc.gracefulShutdown(ctx); err != nil {
				lastErr = err
			}
		}
		close(done)
	}()

	select {
	case <-done:
		// All SubConns closed gracefully
	case <-ctx.Done():
		// Timeout - force close remaining
		for _, sc := range subConns {
			sc.shutdown()
		}
		lastErr = ctx.Err()
	}

	p.mu.Lock()
	p.subConns = nil
	p.readySubConns = nil
	p.mu.Unlock()

	p.resolver.Close()
	return lastErr
}

// ReadyCount returns the number of ready connections.
func (p *Pool) ReadyCount() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.readySubConns)
}

// SubConnCount returns the total number of SubConns (all states).
func (p *Pool) SubConnCount() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.subConns)
}

// isConnectionError returns true if the error indicates a connection failure
// that warrants reconnection.
func isConnectionError(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, client.ErrClosed) ||
		errors.Is(err, client.ErrFatalError)
}
