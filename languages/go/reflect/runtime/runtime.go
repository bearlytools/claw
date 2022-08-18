// Package runtime provides runtime helpers for the Claw file.
package runtime

import (
	"fmt"

	"github.com/bearlytools/claw/languages/go/reflect/internal/value"
)

var registry map[string]value.PackageDescr

// RegisterPackage registers a PackageDescr for the runtime.
func RegisterPackage(descr value.PackageDescr) {
	if _, ok := registry[descr.FullPath()]; ok {
		panic(fmt.Sprintf("cannot register %q twice", descr.FullPath()))
	}
	registry[descr.FullPath()] = descr
}

// PackageDescr returns the PackageDescr for the path.
func PackageDescr(path string) value.PackageDescr {
	return registry[path]
}
