package resolver

import (
	"time"
)

// BuildOptions configures resolver creation.
type BuildOptions struct {
	// DialTimeout is the timeout for resolver connections (e.g., to DNS server).
	// Zero means no timeout.
	DialTimeout time.Duration
}

// buildConfig holds internal configuration for resolver building.
type buildConfig struct {
	dialTimeout time.Duration
	picker      Picker
}

func defaultBuildConfig() *buildConfig {
	return &buildConfig{
		dialTimeout: 10 * time.Second,
		picker:      &RoundRobinPicker{},
	}
}

// BuildOption configures resolver building.
type BuildOption func(*buildConfig)

// WithDialTimeout sets the timeout for resolver connections.
func WithDialTimeout(d time.Duration) BuildOption {
	return func(c *buildConfig) {
		c.dialTimeout = d
	}
}

// WithPicker sets the address picker for the resolver.
func WithPicker(p Picker) BuildOption {
	return func(c *buildConfig) {
		c.picker = p
	}
}
