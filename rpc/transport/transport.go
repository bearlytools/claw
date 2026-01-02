// Package transport provides transport abstractions for RPC connections.
package transport

import (
	"fmt"
	"io"
	"net"

	"github.com/gostdlib/base/context"

	"github.com/bearlytools/claw/rpc/transport/resolver"
)

// Transport extends io.ReadWriteCloser with connection state information.
// All RPC transports must implement this interface.
type Transport interface {
	io.ReadWriteCloser

	// LocalAddr returns the local network address, if known.
	// Returns nil if not applicable (e.g., for non-network transports).
	LocalAddr() net.Addr

	// RemoteAddr returns the remote network address, if known.
	// Returns nil if not applicable (e.g., for non-network transports).
	RemoteAddr() net.Addr
}

// Dialer creates new transport connections to a remote endpoint.
type Dialer interface {
	// Dial establishes a new transport connection.
	// The returned Transport is ready for use with rpc/client.New().
	Dial(ctx context.Context) (Transport, error)
}

// Listener accepts incoming transport connections.
type Listener interface {
	// Accept waits for and returns the next incoming connection.
	// The returned Transport is ready for use with rpc/server.Serve().
	Accept(ctx context.Context) (Transport, error)

	// Close stops the listener from accepting new connections.
	// Already accepted connections are not affected.
	Close() error

	// Addr returns the listener's network address.
	Addr() net.Addr
}

// netConnTransport wraps a net.Conn to implement Transport.
type netConnTransport struct {
	net.Conn
}

// NetConnTransport wraps a net.Conn to implement the Transport interface.
// This is useful for using standard network connections with the RPC framework.
func NetConnTransport(conn net.Conn) Transport {
	return &netConnTransport{Conn: conn}
}

func (t *netConnTransport) LocalAddr() net.Addr {
	return t.Conn.LocalAddr()
}

func (t *netConnTransport) RemoteAddr() net.Addr {
	return t.Conn.RemoteAddr()
}

// DialFunc is a function that dials a specific address.
type DialFunc func(ctx context.Context, addr string) (Transport, error)

// ResolvingDialer wraps a dial function with name resolution.
// It resolves the target to addresses and picks one to dial.
type ResolvingDialer struct {
	target   string
	resolver resolver.Resolver
	picker   resolver.Picker
	dialFunc DialFunc
}

// resolvingConfig holds configuration for ResolvingDialer.
type resolvingConfig struct {
	picker resolver.Picker
}

func defaultResolvingConfig() *resolvingConfig {
	return &resolvingConfig{
		picker: &resolver.RoundRobinPicker{},
	}
}

// ResolvingOption configures a ResolvingDialer.
type ResolvingOption func(*resolvingConfig)

// WithPicker sets the address picker for the dialer.
func WithPicker(p resolver.Picker) ResolvingOption {
	return func(c *resolvingConfig) {
		c.picker = p
	}
}

// NewResolvingDialer creates a dialer with name resolution.
// The target is parsed according to the scheme://authority/endpoint format.
// If no scheme is specified, "passthrough" is used.
//
// Example targets:
//   - "dns:///myservice.namespace:8080"
//   - "passthrough:///localhost:8080"
//   - "localhost:8080" (uses passthrough)
func NewResolvingDialer(target string, dialFunc DialFunc, opts ...ResolvingOption) (*ResolvingDialer, error) {
	cfg := defaultResolvingConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	// Parse target
	t, err := resolver.Parse(target)
	if err != nil {
		return nil, fmt.Errorf("parse target: %w", err)
	}

	// Get builder for scheme
	b, ok := resolver.Get(t.Scheme)
	if !ok {
		return nil, fmt.Errorf("unknown resolver scheme: %s", t.Scheme)
	}

	// Build resolver
	r, err := b.Build(t, resolver.BuildOptions{})
	if err != nil {
		return nil, fmt.Errorf("build resolver: %w", err)
	}

	return &ResolvingDialer{
		target:   target,
		resolver: r,
		picker:   cfg.picker,
		dialFunc: dialFunc,
	}, nil
}

// Dial resolves the target and connects to a resolved address.
func (d *ResolvingDialer) Dial(ctx context.Context) (Transport, error) {
	// Resolve target to addresses
	addrs, err := d.resolver.Resolve(ctx)
	if err != nil {
		return nil, fmt.Errorf("resolve %q: %w", d.target, err)
	}

	// Pick an address
	addr, err := d.picker.Pick(addrs)
	if err != nil {
		return nil, fmt.Errorf("pick address: %w", err)
	}

	// Dial the address
	return d.dialFunc(ctx, addr.Addr)
}

// Close releases resources held by the dialer.
func (d *ResolvingDialer) Close() error {
	return d.resolver.Close()
}
