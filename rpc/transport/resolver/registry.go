package resolver

import (
	"github.com/gostdlib/base/concurrency/sync"
)

var (
	builders = make(map[string]Builder)
	mu       sync.RWMutex
)

// Register registers a resolver builder for the given scheme.
// If a builder is already registered for the scheme, it is replaced.
// This is typically called from init() functions in resolver packages.
func Register(b Builder) {
	mu.Lock()
	defer mu.Unlock()
	builders[b.Scheme()] = b
}

// Get returns the builder for the given scheme.
// Returns nil and false if no builder is registered for the scheme.
func Get(scheme string) (Builder, bool) {
	mu.RLock()
	defer mu.RUnlock()
	b, ok := builders[scheme]
	return b, ok
}

// Schemes returns all registered scheme names.
func Schemes() []string {
	mu.RLock()
	defer mu.RUnlock()
	schemes := make([]string, 0, len(builders))
	for scheme := range builders {
		schemes = append(schemes, scheme)
	}
	return schemes
}
