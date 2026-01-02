// Package resolver provides name resolution for RPC transports.
// It enables service discovery through DNS, static addresses, and custom resolvers.
package resolver

import (
	"github.com/gostdlib/base/context"
)

// Address represents a resolved network address.
type Address struct {
	// Addr is the network address (e.g., "192.168.1.1:8080" or "/var/run/app.sock").
	Addr string

	// Weight is used for weighted load balancing. Higher weight means more traffic.
	// Zero is treated as default weight (1).
	Weight uint32

	// Priority is used for priority-based selection. Lower value means higher priority.
	// Zero is the highest priority.
	Priority uint32

	// Attributes holds arbitrary metadata about this address.
	// Examples: datacenter, zone, version labels.
	Attributes map[string]any
}

// Target represents a parsed target string.
// Format: scheme://authority/endpoint
type Target struct {
	// Scheme identifies the resolver to use (e.g., "dns", "passthrough").
	Scheme string

	// Authority is an optional component, typically used for custom resolver configuration.
	// For DNS, this could be a custom DNS server address.
	Authority string

	// Endpoint is the service name or address to resolve.
	Endpoint string
}

// Resolver resolves service names to network addresses.
type Resolver interface {
	// Resolve returns addresses for the configured target.
	// The returned slice may contain multiple addresses for load balancing.
	// Implementations should respect context cancellation and deadlines.
	Resolve(ctx context.Context) ([]Address, error)

	// Close releases any resources held by the resolver.
	// After Close is called, Resolve should not be called.
	Close() error
}

// Builder creates Resolver instances for a specific scheme.
// Builders are registered globally and looked up by scheme.
type Builder interface {
	// Scheme returns the scheme this builder handles (e.g., "dns", "passthrough").
	// Must be lowercase and match RFC 3986 scheme syntax.
	Scheme() string

	// Build creates a new resolver for the given target.
	// The target's Scheme field will match this builder's Scheme().
	Build(target Target, opts BuildOptions) (Resolver, error)
}
