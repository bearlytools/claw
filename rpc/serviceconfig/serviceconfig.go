// Package serviceconfig provides per-method configuration for RPC calls.
// This allows setting default timeouts and other options on a per-service
// or per-method basis without modifying call sites.
package serviceconfig

import (
	"strings"
	"time"
)

// MethodConfig configures behavior for matching methods.
type MethodConfig struct {
	// Timeout is the default timeout for calls to this method.
	// Zero means no default timeout (use context deadline only).
	// This timeout is only applied if the context does not already have a deadline.
	Timeout time.Duration

	// WaitForReady, if true, causes calls to block until the connection
	// is ready rather than failing immediately.
	WaitForReady bool
}

// Config holds configuration for RPC services.
// It maps method patterns to their configuration.
type Config struct {
	// methods maps method patterns to their configuration.
	// Patterns are matched in order of specificity:
	//   1. "pkg/service/method" - exact match
	//   2. "pkg/service/*" - all methods in service
	//   3. "pkg/*/*" - all methods in package
	//   4. "*/*/*" - global default
	methods map[string]MethodConfig
}

// New creates a new empty service config.
func New() *Config {
	return &Config{
		methods: make(map[string]MethodConfig),
	}
}

// SetMethodConfig sets the configuration for a method pattern.
// Pattern format: "pkg/service/method", "pkg/service/*", "pkg/*/*", or "*/*/*"
func (c *Config) SetMethodConfig(pattern string, cfg MethodConfig) *Config {
	c.methods[pattern] = cfg
	return c
}

// SetTimeout is a convenience method to set just the timeout for a pattern.
func (c *Config) SetTimeout(pattern string, timeout time.Duration) *Config {
	cfg := c.methods[pattern]
	cfg.Timeout = timeout
	c.methods[pattern] = cfg
	return c
}

// SetWaitForReady is a convenience method to set wait-for-ready for a pattern.
func (c *Config) SetWaitForReady(pattern string, wait bool) *Config {
	cfg := c.methods[pattern]
	cfg.WaitForReady = wait
	c.methods[pattern] = cfg
	return c
}

// GetMethodConfig returns the configuration for a specific method.
// It tries to match in order of specificity:
//  1. Exact match: "pkg/service/method"
//  2. Service wildcard: "pkg/service/*"
//  3. Package wildcard: "pkg/*/*"
//  4. Global wildcard: "*/*/*"
//
// Returns the matched config and true if found, or zero config and false if not.
func (c *Config) GetMethodConfig(pkg, service, method string) (MethodConfig, bool) {
	if c == nil || len(c.methods) == 0 {
		return MethodConfig{}, false
	}

	// Try exact match first.
	exact := pkg + "/" + service + "/" + method
	if cfg, ok := c.methods[exact]; ok {
		return cfg, true
	}

	// Try service wildcard.
	servicePattern := pkg + "/" + service + "/*"
	if cfg, ok := c.methods[servicePattern]; ok {
		return cfg, true
	}

	// Try package wildcard.
	pkgPattern := pkg + "/*/*"
	if cfg, ok := c.methods[pkgPattern]; ok {
		return cfg, true
	}

	// Try global wildcard.
	if cfg, ok := c.methods["*/*/*"]; ok {
		return cfg, true
	}

	return MethodConfig{}, false
}

// GetTimeout returns the timeout for a specific method.
// Returns 0 if no timeout is configured.
func (c *Config) GetTimeout(pkg, service, method string) time.Duration {
	cfg, ok := c.GetMethodConfig(pkg, service, method)
	if !ok {
		return 0
	}
	return cfg.Timeout
}

// GetWaitForReady returns the wait-for-ready setting for a specific method.
func (c *Config) GetWaitForReady(pkg, service, method string) bool {
	cfg, ok := c.GetMethodConfig(pkg, service, method)
	if !ok {
		return false
	}
	return cfg.WaitForReady
}

// ParsePattern parses a method pattern into its components.
// Returns pkg, service, method, and whether the parse was successful.
func ParsePattern(pattern string) (pkg, service, method string, ok bool) {
	parts := strings.Split(pattern, "/")
	if len(parts) != 3 {
		return "", "", "", false
	}
	return parts[0], parts[1], parts[2], true
}

// Builder provides a fluent interface for building service configs.
type Builder struct {
	config *Config
}

// NewBuilder creates a new config builder.
func NewBuilder() *Builder {
	return &Builder{config: New()}
}

// WithTimeout adds a timeout for a pattern.
func (b *Builder) WithTimeout(pattern string, timeout time.Duration) *Builder {
	b.config.SetTimeout(pattern, timeout)
	return b
}

// WithWaitForReady sets wait-for-ready for a pattern.
func (b *Builder) WithWaitForReady(pattern string, wait bool) *Builder {
	b.config.SetWaitForReady(pattern, wait)
	return b
}

// WithMethodConfig adds a full method config for a pattern.
func (b *Builder) WithMethodConfig(pattern string, cfg MethodConfig) *Builder {
	b.config.SetMethodConfig(pattern, cfg)
	return b
}

// WithDefaultTimeout sets a global default timeout for all methods.
func (b *Builder) WithDefaultTimeout(timeout time.Duration) *Builder {
	b.config.SetTimeout("*/*/*", timeout)
	return b
}

// Build returns the completed config.
func (b *Builder) Build() *Config {
	return b.config
}
