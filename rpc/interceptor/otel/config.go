// Package otel provides OpenTelemetry tracing and metrics interceptors for RPC servers and clients.
package otel

import (
	"net"
	"strings"

	"go.opentelemetry.io/otel/metric"

	"github.com/bearlytools/claw/rpc/internal/msgs"
)

// Config configures the OpenTelemetry interceptors.
type Config struct {
	// EnableTracing enables distributed tracing. Default is true.
	EnableTracing bool

	// EnableMetrics enables metrics collection. Default is true.
	EnableMetrics bool

	// MeterProvider for metrics. If nil, uses context.Meter().
	MeterProvider metric.MeterProvider

	// RecordPayloadSize records request/response sizes in metrics. Default is true.
	RecordPayloadSize bool

	// TraceRules defines custom rules for always-trace scenarios.
	// These rules are evaluated AFTER the OTEL sampler decision.
	// If any rule matches, the request is traced regardless of sampler.
	TraceRules *TraceRules
}

// DefaultConfig returns a Config with default values.
func DefaultConfig() Config {
	return Config{
		EnableTracing:     true,
		EnableMetrics:     true,
		RecordPayloadSize: true,
	}
}

// TraceRules defines conditions for always-trace scenarios.
// If any condition matches, the request is traced.
type TraceRules struct {
	// IPRanges are CIDR blocks that should always be traced.
	// Example: ["10.0.0.0/8", "192.168.1.0/24"]
	IPRanges []string

	// Metadata specifies key/value pairs that trigger tracing.
	// If a request contains a matching metadata key/value, it's traced.
	// Use "*" as value to match any value for that key.
	Metadata map[string]string

	// Methods are specific RPC methods to always trace.
	// Format: "package/service/method" or just "method"
	Methods []string

	// Parsed CIDR networks (populated by compile)
	cidrs []*net.IPNet
}

// compile parses the CIDR strings into net.IPNet for efficient matching.
func (r *TraceRules) compile() error {
	if r == nil {
		return nil
	}

	r.cidrs = make([]*net.IPNet, 0, len(r.IPRanges))
	for _, cidr := range r.IPRanges {
		_, network, err := net.ParseCIDR(cidr)
		if err != nil {
			return err
		}
		r.cidrs = append(r.cidrs, network)
	}
	return nil
}

// matchesIP checks if the given IP address matches any of the configured CIDR ranges.
func (r *TraceRules) matchesIP(ipStr string) bool {
	if r == nil || len(r.cidrs) == 0 {
		return false
	}

	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}

	for _, cidr := range r.cidrs {
		if cidr.Contains(ip) {
			return true
		}
	}
	return false
}

// matchesMetadata checks if any of the request metadata matches the configured rules.
func (r *TraceRules) matchesMetadata(metadata []msgs.Metadata) bool {
	if r == nil || len(r.Metadata) == 0 {
		return false
	}

	for _, md := range metadata {
		key := md.Key()
		if want, ok := r.Metadata[key]; ok {
			if want == "*" || want == string(md.Value()) {
				return true
			}
		}
	}
	return false
}

// matchesMethod checks if the given method matches any of the configured methods.
func (r *TraceRules) matchesMethod(fullMethod string) bool {
	if r == nil || len(r.Methods) == 0 {
		return false
	}

	for _, method := range r.Methods {
		if method == fullMethod {
			return true
		}
		// Also check if the method is a suffix match (e.g., "Login" matches "auth/Login")
		if strings.HasSuffix(fullMethod, "/"+method) {
			return true
		}
	}
	return false
}

// ShouldTrace returns true if any trace rule matches the given request info.
func (r *TraceRules) ShouldTrace(ip, method string, metadata []msgs.Metadata) bool {
	if r == nil {
		return false
	}

	return r.matchesIP(ip) || r.matchesMethod(method) || r.matchesMetadata(metadata)
}

// TODO: LoadConfig will be implemented once clawtext supports the Config struct.
// For now, use NewConfig() with programmatic configuration.
