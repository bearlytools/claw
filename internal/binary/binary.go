// Package binary replaces the encoding/binary package in the standard library for little endian encoding using generics.
package binary

import (
	"fmt"

	"golang.org/x/exp/constraints"
)

// Get gets any Uint size from a []byte slice.
func Get[T constraints.Integer](b []byte) T {
	_ = b[len(b)-1] // bounds check hint to compiler; see golang.org/issue/14808

	var r T // This is only used for type detction.
	switch any(r).(type) {
	case int8:
		return T(int8(b[0]))
	case int16:
		return T(int16(uint16(b[0]) | uint16(b[1])<<8))
	case int32:
		return T(int32(uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 | uint32(b[3])<<24))
	case int64:
		return T(int64(uint64(b[0]) | uint64(b[1])<<8 | uint64(b[2])<<16 | uint64(b[3])<<24 |
                uint64(b[4])<<32 | uint64(b[5])<<40 | uint64(b[6])<<48 | uint64(b[7])<<56))
	case uint8:
		return T(uint8(b[0]))
	case uint16:
		return T(uint16(b[0]) | uint16(b[1])<<8)
	case uint32:
		return T(uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 | uint32(b[3])<<24)
	case uint64:
		return T(uint64(b[0]) | uint64(b[1])<<8 | uint64(b[2])<<16 | uint64(b[3])<<24 |
			uint64(b[4])<<32 | uint64(b[5])<<40 | uint64(b[6])<<48 | uint64(b[7])<<56)
	}
	panic(fmt.Sprintf("unsupported type that passed the type constraint %T", r))
}

// Put puts any Uint size into a []byte slice.
func Put[T constraints.Integer](b []byte, v T) {
	_ = b[len(b)-1] // bounds check hint to compiler; see golang.org/issue/14808

	switch any(v).(type) {
	case int8:
		v = T(uint8(v))
	case int16:
		v = T(uint16(v))
	case int32:
		v = T(uint32(v))
	case int64:
		v = T(uint64(v))
	}

	switch any(v).(type) {
	case uint8:
		b[0] = byte(v)
	case uint16:
		b[0] = byte(v)
		b[1] = byte(v >> 8)
		return
	case uint32:
		b[0] = byte(v)
		b[1] = byte(v >> 8)
		b[2] = byte(v >> 16)
		b[3] = byte(v >> 24)
		return
	}
	b[0] = byte(v)
	b[1] = byte(v >> 8)
	b[2] = byte(v >> 16)
	b[3] = byte(v >> 24)
	b[4] = byte(v >> 32)
	b[5] = byte(v >> 40)
	b[6] = byte(v >> 48)
	b[7] = byte(v >> 56)
}
