// Package compress provides compression support for the Claw RPC system.
// It includes built-in compressors for gzip, snappy, and zstd, and supports
// custom compressor registration.
package compress

import (
	"fmt"

	"github.com/bearlytools/claw/rpc/internal/msgs"
	"github.com/gostdlib/base/concurrency/sync"
)

// Compressor defines the interface for compression algorithms.
type Compressor interface {
	// Compress compresses data. Returns compressed data or error.
	Compress(data []byte) ([]byte, error)

	// Decompress decompresses data. Returns original data or error.
	Decompress(data []byte) ([]byte, error)

	// Type returns the compression type for the wire protocol.
	Type() msgs.Compression
}

var (
	registry   = map[msgs.Compression]Compressor{}
	registryMu sync.RWMutex
)

// Register adds a compressor to the registry. This can be used to register
// custom compressors or override built-in compressors. Thread-safe.
func Register(c Compressor) {
	registryMu.Lock()
	defer registryMu.Unlock()
	registry[c.Type()] = c
}

// Get returns the compressor for the given type, or nil if not found.
func Get(t msgs.Compression) Compressor {
	registryMu.RLock()
	defer registryMu.RUnlock()
	return registry[t]
}

// Compress compresses data using the specified algorithm.
// Returns original data unchanged if type is CmpNone.
// Returns an error if the compressor is not registered.
func Compress(t msgs.Compression, data []byte) ([]byte, error) {
	if t == msgs.CmpNone {
		return data, nil
	}
	if len(data) == 0 {
		return data, nil
	}
	c := Get(t)
	if c == nil {
		return nil, fmt.Errorf("compressor not registered for type %d", t)
	}
	return c.Compress(data)
}

// Decompress decompresses data using the specified algorithm.
// Returns original data unchanged if type is CmpNone.
// Returns an error if the compressor is not registered.
func Decompress(t msgs.Compression, data []byte) ([]byte, error) {
	if t == msgs.CmpNone {
		return data, nil
	}
	if len(data) == 0 {
		return data, nil
	}
	c := Get(t)
	if c == nil {
		return nil, fmt.Errorf("compressor not registered for type %d", t)
	}
	return c.Decompress(data)
}

func init() {
	Register(&Gzip{})
	Register(&Snappy{})
	Register(&Zstd{})
}
