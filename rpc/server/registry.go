package server

import (
	"errors"
	"fmt"

	"github.com/gostdlib/base/concurrency/sync"

	"github.com/bearlytools/claw/rpc/internal/msgs"
)

// ErrHandlerExists is returned when trying to register a handler that already exists.
var ErrHandlerExists = errors.New("handler already registered")

// Registry manages handler registration for RPC services.
type Registry struct {
	handlers map[string]Handler
	mu       sync.RWMutex
}

// NewRegistry creates a new handler registry.
func NewRegistry() *Registry {
	return &Registry{
		handlers: make(map[string]Handler),
	}
}

// Register registers a handler for a specific package/service/call combination.
func (r *Registry) Register(pkg, service, call string, h Handler) error {
	key := makeKey(pkg, service, call)

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.handlers[key]; exists {
		return fmt.Errorf("%w: %s", ErrHandlerExists, key)
	}

	r.handlers[key] = h
	return nil
}

// Lookup finds a handler for the given package/service/call.
func (r *Registry) Lookup(pkg, service, call string) (Handler, bool) {
	key := makeKey(pkg, service, call)

	r.mu.RLock()
	defer r.mu.RUnlock()

	h, ok := r.handlers[key]
	return h, ok
}

// LookupByDescr finds a handler using a Descr message.
func (r *Registry) LookupByDescr(descr msgs.Descr) (Handler, bool) {
	return r.Lookup(descr.Package(), descr.Service(), descr.Call())
}

func makeKey(pkg, service, call string) string {
	return pkg + "/" + service + "/" + call
}
