package dns

import (
	"time"
)

// Option configures a DNS resolver.
type Option func(*config)

// WithDefaultPort sets the default port used when the endpoint doesn't include one.
// Default is "443".
func WithDefaultPort(port string) Option {
	return func(c *config) {
		c.defaultPort = port
	}
}

// WithSRV enables SRV record lookups before falling back to A/AAAA records.
// The service and proto parameters correspond to the SRV record format:
// _service._proto.name
//
// Example: WithSRV("grpc", "tcp") looks up _grpc._tcp.endpoint
func WithSRV(service, proto string) Option {
	return func(c *config) {
		c.srvService = service
		c.srvProto = proto
		c.useSRV = true
	}
}

// WithResolveTimeout sets the timeout for DNS resolution.
// Default is 10 seconds.
func WithResolveTimeout(d time.Duration) Option {
	return func(c *config) {
		c.resolveTimeout = d
	}
}
