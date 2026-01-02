// Package context provides RPC-specific context utilities.
// It uses private key types to prevent collisions with other packages.
package context

import (
	"net"

	"github.com/gostdlib/base/context"
)

// remoteAddrKey is a private type used as a context key for the remote address.
type remoteAddrKey struct{}

// RemoteAddr retrieves the remote address from context.
// Returns nil if not set.
func RemoteAddr(ctx context.Context) net.Addr {
	addr, _ := ctx.Value(remoteAddrKey{}).(net.Addr)
	return addr
}

// WithRemoteAddr returns a context with the remote address attached.
func WithRemoteAddr(ctx context.Context, addr net.Addr) context.Context {
	return context.WithValue(ctx, remoteAddrKey{}, addr)
}
