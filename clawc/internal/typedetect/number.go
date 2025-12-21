// Package typedetect provides utilities for detecting type characteristics at runtime
// using unsafe operations for performance.
package typedetect

import (
	"unsafe"

	"golang.org/x/exp/constraints"
)

// Number represents all int, uint and float types.
type Number interface {
	constraints.Integer | constraints.Float
}

// IsSignedInteger returns true if T is a signed integer type.
func IsSignedInteger[T Number]() bool {
	var t T
	
	// Set the high bit - if T is signed, this will be negative; if unsigned, positive
	switch unsafe.Sizeof(t) {
	case 1:
		*(*uint8)(unsafe.Pointer(&t)) = 0x80 // high bit set
		return t < 0
	case 2:
		*(*uint16)(unsafe.Pointer(&t)) = 0x8000 // high bit set
		return t < 0
	case 4:
		*(*uint32)(unsafe.Pointer(&t)) = 0x80000000 // high bit set
		return t < 0
	case 8:
		*(*uint64)(unsafe.Pointer(&t)) = 0x8000000000000000 // high bit set
		return t < 0
	default:
		return false
	}
}

// IsFloat returns true if T is a floating point type.
func IsFloat[T Number]() bool {
	// Use NaN property: NaN != NaN only for floats
	switch unsafe.Sizeof(T(0)) {
	case 4:
		nanBits := uint32(0x7FC00000) // float32 NaN
		nan := *(*T)(unsafe.Pointer(&nanBits))
		return nan != nan
	case 8:
		nanBits := uint64(0x7FF8000000000000) // float64 NaN
		nan := *(*T)(unsafe.Pointer(&nanBits))
		return nan != nan
	default:
		return false
	}
}