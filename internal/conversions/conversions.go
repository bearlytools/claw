// Package conversions is a set of unsafe conversions from one type to another. Such as converting
// some number to its slice representation or a slice representation
package conversions

import (
	"fmt"
	"reflect"
	"unsafe"
)

// FixedIntegers are integer types that don't vary in size.
type FixedIntegers interface {
	~uint8 | ~uint16 | ~uint32 | ~uint64 |
		~int8 | ~int16 | ~int32 | ~int64
}

// NumToBytes returns the underlying storage that value's integer points at.
// This is a pointer to an integer, because otherwise we'd have to make an
// allocation and at that point it would be a useless exercise.
// Mainly exists to reconvert after a BytesToNum() call.
func NumToBytes[N FixedIntegers](value *N) []byte {
	switch any(value).(type) {
	case *uint8, *int8:
		b := (*[1]byte)(unsafe.Pointer(value))
		return b[:]
	case *uint16, *int16:
		b := (*[2]byte)(unsafe.Pointer(value))
		return b[:]
	case *uint32, *int32:
		b := (*[4]byte)(unsafe.Pointer(value))
		return b[:]
	case *uint64, *int64:
		b := (*[8]byte)(unsafe.Pointer(value))
		return b[:]
	default:
		panic(fmt.Sprintf("unsupported type: %T", *value))
	}
}

// BytesToNum returns a pointer to an integer that uses []byte as the
// underlying storage. value must have the correct length for the type of
// integer you wish to covert to or it will likely panic (or worse).
func BytesToNum[N FixedIntegers](value []byte) *N {
	switch len(value) {
	case 1:
		ptr := (*N)(unsafe.Pointer((*[1]byte)(value)))
		return ptr
	case 2:
		ptr := (*N)(unsafe.Pointer((*[2]byte)(value)))
		return ptr
	case 4:
		ptr := (*N)(unsafe.Pointer((*[4]byte)(value)))
		return ptr
	case 8:
		ptr := (*N)(unsafe.Pointer((*[8]byte)(value)))
		return ptr
	default:
		panic("value was invalid")
	}
}

// ByteSlice2String coverts bs to a string. It is no longer safe to use bs after this.
// This prevents having to make a copy of bs.
func ByteSlice2String(bs []byte) string {
	return *(*string)(unsafe.Pointer(&bs))
}

// UnsafeGetBytes retrieves the underlying []byte held in string "s" without doing
// a copy. Do not modify the []byte or suffer the consequences.
func UnsafeGetBytes(s string) []byte {
	if s == "" {
		return nil
	}
	return (*[0x7fff0000]byte)(unsafe.Pointer(
		(*reflect.StringHeader)(unsafe.Pointer(&s)).Data),
	)[:len(s):len(s)]
}
