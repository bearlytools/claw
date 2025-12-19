// Package transport provides transport abstractions for RPC connections.
package transport

import (
	"io"
	"net"

	"github.com/gostdlib/base/context"
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
