// Package runtime provides runtime helpers for the Claw file.
package runtime

import (
	"fmt"
	"log"

	"github.com/bearlytools/claw/clawc/languages/go/reflect/internal/interfaces"
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
