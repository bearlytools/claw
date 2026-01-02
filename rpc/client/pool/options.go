// Package pool provides a load-balanced connection pool for RPC clients.
// It manages multiple connections to backend addresses with health checking
// and automatic reconnection.
package pool

import (
	"time"

	"github.com/bearlytools/claw/rpc/client"
	"github.com/bearlytools/claw/rpc/transport/resolver"
)

// config holds configuration for Pool.
type config struct {
	// balancer is the connection selection strategy.
	balancer BalancerPicker

	// healthCheckInterval is how often to health check SubConns.
	// Zero disables health checking.
	healthCheckInterval time.Duration

	// healthCheckTimeout is the timeout for health check calls.
	healthCheckTimeout time.Duration

	// clientOpts are options to pass to each SubConn's client.Conn.
	clientOpts []client.Option

	// resolver is the address resolver. If nil, parsed from target.
	resolver resolver.Resolver

	// minConns is the minimum number of ready connections to maintain.
	// Pool will attempt to keep at least this many connections ready.
	minConns int

	// enableHealthCheck enables health checking of SubConns.
	enableHealthCheck bool
}

func defaultConfig() *config {
	return &config{
		balancer:            &RoundRobinBalancer{},
		healthCheckInterval: 30 * time.Second,
		healthCheckTimeout:  5 * time.Second,
		minConns:            1,
		enableHealthCheck:   true,
	}
}

// Option configures a Pool.
type Option func(*config)

// WithBalancer sets the connection selection strategy.
// Default is RoundRobinBalancer.
func WithBalancer(b BalancerPicker) Option {
	return func(c *config) {
		if b != nil {
			c.balancer = b
		}
	}
}

// WithHealthCheckInterval sets how often to health check SubConns.
// Default is 30 seconds. Set to zero to disable health checking.
func WithHealthCheckInterval(d time.Duration) Option {
	return func(c *config) {
		c.healthCheckInterval = d
		c.enableHealthCheck = d > 0
	}
}

// WithHealthCheckTimeout sets the timeout for health check calls.
// Default is 5 seconds.
func WithHealthCheckTimeout(d time.Duration) Option {
	return func(c *config) {
		c.healthCheckTimeout = d
	}
}

// WithClientOptions sets options to pass to each SubConn's client.Conn.
func WithClientOptions(opts ...client.Option) Option {
	return func(c *config) {
		c.clientOpts = append(c.clientOpts, opts...)
	}
}

// WithResolver sets a custom resolver for address discovery.
// If not set, the target string is parsed to create a resolver.
func WithResolver(r resolver.Resolver) Option {
	return func(c *config) {
		c.resolver = r
	}
}

// WithMinConnections sets the minimum number of ready connections to maintain.
// Default is 1.
func WithMinConnections(n int) Option {
	return func(c *config) {
		if n > 0 {
			c.minConns = n
		}
	}
}

// WithHealthCheckDisabled disables health checking of SubConns.
// By default, health checking is enabled.
func WithHealthCheckDisabled() Option {
	return func(c *config) {
		c.enableHealthCheck = false
	}
}
