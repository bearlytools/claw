// Package pool provides a load-balanced connection pool for RPC clients.
// It manages multiple connections to backend addresses with health checking
// and automatic reconnection.
package pool

import (
	"errors"

	"github.com/gostdlib/base/concurrency/sync"
	"github.com/gostdlib/base/context"
	"github.com/gostdlib/base/retry/exponential"

	"github.com/bearlytools/claw/rpc/client"
	"github.com/bearlytools/claw/rpc/health"
	"github.com/bearlytools/claw/rpc/transport"
	"github.com/bearlytools/claw/rpc/transport/resolver"
)

// ConnState represents the state of a SubConn.
type ConnState uint8

const (
	// StateIdle indicates the SubConn is not connected and not trying to connect.
	StateIdle ConnState = iota
	// StateConnecting indicates the SubConn is establishing a connection.
	StateConnecting
	// StateReady indicates the SubConn is connected and ready for RPCs.
	StateReady
	// StateTransientFailure indicates the SubConn has failed and is backing off.
	StateTransientFailure
	// StateShutdown indicates the SubConn is shut down permanently.
	StateShutdown
)

// String implements fmt.Stringer.
func (s ConnState) String() string {
	switch s {
	case StateIdle:
		return "IDLE"
	case StateConnecting:
		return "CONNECTING"
	case StateReady:
		return "READY"
	case StateTransientFailure:
		return "TRANSIENT_FAILURE"
	case StateShutdown:
		return "SHUTDOWN"
	default:
		return "UNKNOWN"
	}
}

// Common errors for SubConn.
var (
	ErrSubConnShutdown = errors.New("subconn is shutdown")
	ErrSubConnNotReady = errors.New("subconn is not ready")
	ErrNoReadySubConns = errors.New("no ready subconns available")
)

// SubConn represents a connection to a single backend address.
// It manages connection lifecycle including connecting, health checking,
// and reconnection with exponential backoff.
type SubConn struct {
	addr       resolver.Address
	dialFunc   transport.DialFunc
	clientOpts []client.Option

	conn   *client.Conn
	state  ConnState
	health health.ServingStatus
	mu     sync.Mutex

	lastErr error
	closeCh chan struct{}
	backoff *exponential.Backoff
}

// newSubConn creates a new SubConn for the given address.
func newSubConn(addr resolver.Address, dialFunc transport.DialFunc, clientOpts []client.Option) *SubConn {
	backoff, _ := exponential.New(exponential.WithPolicy(exponential.ThirtySecondsRetryPolicy()))
	return &SubConn{
		addr:       addr,
		dialFunc:   dialFunc,
		clientOpts: clientOpts,
		state:      StateIdle,
		health:     health.Unknown,
		closeCh:    make(chan struct{}),
		backoff:    backoff,
	}
}

// Addr returns the address this SubConn connects to.
func (sc *SubConn) Addr() resolver.Address {
	return sc.addr
}

// State returns the current connection state.
func (sc *SubConn) State() ConnState {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	return sc.state
}

// HealthStatus returns the current health status.
func (sc *SubConn) HealthStatus() health.ServingStatus {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	return sc.health
}

// IsReady returns true if the SubConn is connected and healthy.
func (sc *SubConn) IsReady() bool {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	return sc.state == StateReady && sc.health == health.Serving
}

// LastError returns the last error that occurred, if any.
func (sc *SubConn) LastError() error {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	return sc.lastErr
}

// Conn returns the underlying client.Conn.
// Returns nil if not connected. Caller must check IsReady() first.
func (sc *SubConn) Conn() *client.Conn {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	return sc.conn
}

// Connect initiates a connection to the backend.
// This is non-blocking; connection happens asynchronously.
func (sc *SubConn) Connect(ctx context.Context) {
	sc.mu.Lock()
	switch sc.state {
	case StateShutdown:
		sc.mu.Unlock()
		return
	case StateConnecting, StateReady:
		sc.mu.Unlock()
		return
	}
	sc.state = StateConnecting
	sc.mu.Unlock()

	pool := context.Pool(ctx)
	pool.Submit(ctx, func() {
		sc.connectWithRetry(ctx)
	})
}

// connectWithRetry attempts to connect with exponential backoff.
func (sc *SubConn) connectWithRetry(ctx context.Context) {
	// Create a context that can be cancelled by shutdown
	connectCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Watch for shutdown
	pool := context.Pool(ctx)
	pool.Submit(ctx, func() {
		select {
		case <-sc.closeCh:
			cancel()
		case <-connectCtx.Done():
		}
	})

	err := sc.backoff.Retry(connectCtx, func(retryCtx context.Context, r exponential.Record) error {
		err := sc.tryConnect(retryCtx)
		if err != nil {
			sc.mu.Lock()
			if sc.state == StateShutdown {
				sc.mu.Unlock()
				return exponential.ErrRetryCanceled
			}
			sc.state = StateTransientFailure
			sc.lastErr = err
			sc.mu.Unlock()
		}
		return err
	})

	if err != nil && !errors.Is(err, exponential.ErrRetryCanceled) {
		sc.mu.Lock()
		if sc.state != StateShutdown {
			sc.state = StateTransientFailure
			sc.lastErr = err
		}
		sc.mu.Unlock()
	}
}

// tryConnect attempts a single connection.
func (sc *SubConn) tryConnect(ctx context.Context) error {
	// Dial the transport
	t, err := sc.dialFunc(ctx, sc.addr.Addr)
	if err != nil {
		return err
	}

	// Create client.Conn
	conn := client.New(ctx, t, sc.clientOpts...)

	sc.mu.Lock()
	if sc.state == StateShutdown {
		sc.mu.Unlock()
		conn.Close()
		return ErrSubConnShutdown
	}
	sc.conn = conn
	sc.state = StateReady
	sc.health = health.Serving // Assume healthy until checked
	sc.lastErr = nil
	sc.mu.Unlock()

	return nil
}

// CheckHealth performs a health check on the connection.
// Returns the health status.
func (sc *SubConn) CheckHealth(ctx context.Context) health.ServingStatus {
	sc.mu.Lock()
	conn := sc.conn
	state := sc.state
	sc.mu.Unlock()

	if state != StateReady || conn == nil {
		return health.Unknown
	}

	status, err := health.Check(ctx, conn, "")
	if err != nil {
		sc.mu.Lock()
		sc.health = health.NotServing
		sc.lastErr = err
		sc.mu.Unlock()
		return health.NotServing
	}

	sc.mu.Lock()
	sc.health = status
	sc.mu.Unlock()
	return status
}

// setHealth updates the health status.
func (sc *SubConn) setHealth(status health.ServingStatus) {
	sc.mu.Lock()
	sc.health = status
	sc.mu.Unlock()
}

// disconnect closes the current connection and transitions to idle.
func (sc *SubConn) disconnect() {
	sc.mu.Lock()
	conn := sc.conn
	sc.conn = nil
	if sc.state != StateShutdown {
		sc.state = StateIdle
	}
	sc.health = health.Unknown
	sc.mu.Unlock()

	if conn != nil {
		conn.Close()
	}
}

// shutdown permanently shuts down the SubConn immediately.
func (sc *SubConn) shutdown() {
	sc.mu.Lock()
	if sc.state == StateShutdown {
		sc.mu.Unlock()
		return
	}
	sc.state = StateShutdown
	conn := sc.conn
	sc.conn = nil

	select {
	case <-sc.closeCh:
	default:
		close(sc.closeCh)
	}
	sc.mu.Unlock()

	if conn != nil {
		conn.Close()
	}
}

// gracefulShutdown gracefully shuts down the SubConn, waiting for in-flight
// RPCs to complete. The context controls how long to wait.
func (sc *SubConn) gracefulShutdown(ctx context.Context) error {
	sc.mu.Lock()
	if sc.state == StateShutdown {
		sc.mu.Unlock()
		return nil
	}
	sc.state = StateShutdown
	conn := sc.conn
	sc.conn = nil

	select {
	case <-sc.closeCh:
	default:
		close(sc.closeCh)
	}
	sc.mu.Unlock()

	if conn != nil {
		return conn.GracefulClose(ctx)
	}
	return nil
}

// handleConnectionFailure is called when an RPC fails due to connection issues.
// It marks the connection as failed and triggers reconnection.
func (sc *SubConn) handleConnectionFailure(ctx context.Context, err error) {
	sc.mu.Lock()
	if sc.state == StateShutdown {
		sc.mu.Unlock()
		return
	}

	sc.lastErr = err
	conn := sc.conn
	sc.conn = nil
	sc.state = StateConnecting
	sc.health = health.Unknown
	sc.mu.Unlock()

	if conn != nil {
		conn.Close()
	}

	// Start reconnection
	pool := context.Pool(ctx)
	pool.Submit(ctx, func() {
		sc.connectWithRetry(ctx)
	})
}
