// Package ratelimit provides rate limiting interceptors for RPC servers.
// It uses a token bucket algorithm to limit request rates per key.
package ratelimit

import (
	"errors"
	"time"

	"github.com/gostdlib/base/concurrency/sync"
	"github.com/gostdlib/base/context"

	"github.com/bearlytools/claw/rpc/interceptor"
	"github.com/bearlytools/claw/rpc/internal/msgs"
)

// ErrRateLimited is returned when a request is rate limited.
var ErrRateLimited = errors.New("rate limited")

// KeyFunc extracts a rate limiting key from RPC info.
// Different requests with the same key share rate limits.
type KeyFunc func(info interface{}) string

// ByMethod returns a KeyFunc that limits by method name.
// Format: "package/service/method"
func ByMethod() KeyFunc {
	return func(info interface{}) string {
		switch i := info.(type) {
		case *interceptor.UnaryServerInfo:
			return i.Package + "/" + i.Service + "/" + i.Method
		case *interceptor.StreamServerInfo:
			return i.Package + "/" + i.Service + "/" + i.Method
		}
		return "unknown"
	}
}

// ByClient returns a KeyFunc that limits by a metadata key value.
// Use this to limit by client ID, API key, or similar identifier.
func ByClient(metadataKey string) KeyFunc {
	return func(info interface{}) string {
		switch i := info.(type) {
		case *interceptor.UnaryServerInfo:
			return findMetadataValue(i.Metadata, metadataKey)
		case *interceptor.StreamServerInfo:
			return findMetadataValue(i.Metadata, metadataKey)
		}
		return "unknown"
	}
}

// ByMethodAndClient returns a KeyFunc that limits by both method and client.
// Format: "package/service/method:clientValue"
func ByMethodAndClient(metadataKey string) KeyFunc {
	return func(info interface{}) string {
		var method, client string
		switch i := info.(type) {
		case *interceptor.UnaryServerInfo:
			method = i.Package + "/" + i.Service + "/" + i.Method
			client = findMetadataValue(i.Metadata, metadataKey)
		case *interceptor.StreamServerInfo:
			method = i.Package + "/" + i.Service + "/" + i.Method
			client = findMetadataValue(i.Metadata, metadataKey)
		default:
			return "unknown"
		}
		return method + ":" + client
	}
}

// findMetadataValue finds a value in the metadata list by key.
func findMetadataValue(metadata []msgs.Metadata, key string) string {
	for _, m := range metadata {
		if m.Key() == key {
			return string(m.Value())
		}
	}
	return ""
}

// Config configures a rate limiter.
type Config struct {
	// Rate is the number of requests allowed per second.
	Rate float64

	// Burst is the maximum number of requests that can be made at once.
	Burst int

	// KeyFunc extracts the rate limiting key from RPC info.
	// If nil, all requests share a single rate limit.
	KeyFunc KeyFunc
}

// bucket represents a token bucket for rate limiting.
type bucket struct {
	tokens     float64
	lastUpdate time.Time
}

// Limiter implements rate limiting using the token bucket algorithm.
type Limiter struct {
	rate    float64
	burst   int
	keyFunc KeyFunc

	mu      sync.Mutex
	buckets map[string]*bucket
}

// New creates a new rate limiter with the given configuration.
func New(cfg Config) *Limiter {
	if cfg.Rate <= 0 {
		cfg.Rate = 100 // Default: 100 req/sec
	}
	if cfg.Burst <= 0 {
		cfg.Burst = 10 // Default: burst of 10
	}
	if cfg.KeyFunc == nil {
		cfg.KeyFunc = func(info interface{}) string { return "" }
	}

	return &Limiter{
		rate:    cfg.Rate,
		burst:   cfg.Burst,
		keyFunc: cfg.KeyFunc,
		buckets: make(map[string]*bucket),
	}
}

// allow checks if a request with the given key is allowed.
// Returns true if allowed, false if rate limited.
func (l *Limiter) allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	b, ok := l.buckets[key]
	if !ok {
		b = &bucket{
			tokens:     float64(l.burst),
			lastUpdate: now,
		}
		l.buckets[key] = b
	}

	// Add tokens based on elapsed time.
	elapsed := now.Sub(b.lastUpdate).Seconds()
	b.tokens += elapsed * l.rate
	if b.tokens > float64(l.burst) {
		b.tokens = float64(l.burst)
	}
	b.lastUpdate = now

	// Try to consume a token.
	if b.tokens >= 1 {
		b.tokens--
		return true
	}
	return false
}

// UnaryServerInterceptor returns an interceptor that rate limits unary RPCs.
func (l *Limiter) UnaryServerInterceptor() interceptor.UnaryServerInterceptor {
	return func(ctx context.Context, req []byte, info *interceptor.UnaryServerInfo, handler interceptor.UnaryHandler) ([]byte, error) {
		key := l.keyFunc(info)
		if !l.allow(key) {
			return nil, ErrRateLimited
		}
		return handler(ctx, req)
	}
}

// StreamServerInterceptor returns an interceptor that rate limits stream RPCs.
// Note: This limits the initial stream creation, not individual messages.
func (l *Limiter) StreamServerInterceptor() interceptor.StreamServerInterceptor {
	return func(ctx context.Context, stream interceptor.ServerStream, info *interceptor.StreamServerInfo, handler interceptor.StreamHandler) error {
		key := l.keyFunc(info)
		if !l.allow(key) {
			return ErrRateLimited
		}
		return handler(ctx, stream)
	}
}

// Cleanup removes rate limit entries that haven't been used for the given duration.
// Call this periodically to prevent memory growth from many unique keys.
func (l *Limiter) Cleanup(maxAge time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	cutoff := time.Now().Add(-maxAge)
	for key, b := range l.buckets {
		if b.lastUpdate.Before(cutoff) {
			delete(l.buckets, key)
		}
	}
}

// Stats returns the number of tracked keys.
func (l *Limiter) Stats() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.buckets)
}
