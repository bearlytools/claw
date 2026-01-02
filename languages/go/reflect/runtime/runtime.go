// Package runtime provides runtime helpers for the Claw file.
package runtime

import (
	"fmt"
	"log"

	"github.com/bearlytools/claw/clawc/languages/go/clawiter"
	"github.com/bearlytools/claw/languages/go/reflect/internal/interfaces"
	"github.com/gostdlib/base/context"
)

var registry = map[string]interfaces.PackageDescr{}

// RegisterPackage registers a PackageDescr for the runtime.
func RegisterPackage(descr interfaces.PackageDescr) {
	log.Println("runtime registered: ", descr.FullPath())
	if _, ok := registry[descr.FullPath()]; ok {
		panic(fmt.Sprintf("cannot register %q twice", descr.FullPath()))
	}
	registry[descr.FullPath()] = descr
}

// PackageDescr returns the PackageDescr for the path.
func PackageDescr(path string) interfaces.PackageDescr {
	log.Println("runtime fetch: ", path)
	return registry[path]
}

// TypeEntry holds information about a registered type for hash-based lookup.
type TypeEntry struct {
	// Name is the struct name (e.g., "Container").
	Name string
	// FullPath is the full import path (e.g., "github.com/example/pkg").
	FullPath string
	// New creates a new empty instance of this type.
	New func(ctx context.Context) AnyType
}

// AnyType is the interface that types must implement to be stored in Any fields.
// This is satisfied by all generated Claw struct types.
type AnyType interface {
	// XXXTypeHash returns the 16-byte SHAKE128 hash identifying this type.
	XXXTypeHash() [16]byte
	// Walk iterates over all fields in the struct.
	Walk(ctx context.Context, yield clawiter.YieldToken, opts ...clawiter.WalkOption)
	// Unmarshal deserializes data into the struct.
	Unmarshal(data []byte) error
	// Marshal serializes the struct to bytes.
	Marshal() ([]byte, error)
}

// typeHashRegistry maps 16-byte type hashes to their TypeEntry.
var typeHashRegistry = map[[16]byte]TypeEntry{}

// RegisterTypeHash registers a type by its hash for Any field decoding.
func RegisterTypeHash(hash [16]byte, entry TypeEntry) {
	if _, ok := typeHashRegistry[hash]; ok {
		// Already registered, skip (can happen with multiple imports)
		return
	}
	typeHashRegistry[hash] = entry
}

// LookupTypeHash returns the TypeEntry for a given hash, or nil if not found.
func LookupTypeHash(hash [16]byte) *TypeEntry {
	entry, ok := typeHashRegistry[hash]
	if !ok {
		return nil
	}
	return &entry
}
