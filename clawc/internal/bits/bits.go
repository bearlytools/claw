// Package bits provides bit manipulation utilities. This is not a replacement for math/bits.
package bits

import (
	"fmt"
	"math/bits"
	"strings"

	"golang.org/x/exp/constraints"
)

type PtrUnsigned interface {
	*uint8 | *uint16 | *uint32 | *uint64
}

// SetValue stores "val" in unsigned number "store" starting at bit "start" and
// ending at bit "end" (exclusive). If start >= end, this panics.
// This clears the existing bits in the range before setting the new value.
func SetValue[I, U constraints.Unsigned](val I, store U, start, end uint64) U {
	if start >= end {
		panic("start cannot be > end")
	}

	// Clear the existing bits in the range first
	store = ClearBits(store, uint8(start), uint8(end))

	// Then set the new value
	c := U(val) << start

	return store | c
}

// GetValue retrieves a value we stored with setValue. store is the unsigned number we
// stored the value in. bitMask is the mask to apply to retrieve the value. start tells
// us the starting position we stored in (we need to shift the number this many bits).
// So if you did something like: storage := setValue(uint8(0), uint64(0), 17, 24)
// You would retrieve with: getValue(storage, aMask, 17)
// Where "mask" would be generated with aMask := mask(uint32(17), uint32(24))
func GetValue[U, U1 constraints.Unsigned](store U, bitMask U, start uint64) U1 {
	return U1((store & bitMask) >> start)
}

// GetBit gets a single bit value from "store" in position "pos". true if set, false if not.
func GetBit[U constraints.Unsigned](store U, pos uint8) bool {
	switch any(store).(type) {
	case uint8:
		if pos > 7 {
			panic(fmt.Sprintf("can't GetBit() a uint8 position %d", pos))
		}
	case uint16:
		if pos > 15 {
			panic(fmt.Sprintf("can't GetBit() a uint16 position %d", pos))
		}
	case uint32:
		if pos > 31 {
			panic(fmt.Sprintf("can't GetBit() a uint32 position %d", pos))
		}
	case uint64:
		if pos > 63 {
			panic(fmt.Sprintf("can't GetBit() a uint64 position %d", pos))
		}
	}
	return store&(1<<pos) != 0
}

// SetBit sets a single bit in "store" at position "pos" to value "val". If val is true,
// the bit is set to 1, if false, it is set to 0.
func SetBit[U constraints.Unsigned](store U, pos uint8, val bool) U {
	switch any(store).(type) {
	case uint8:
		if pos > 7 {
			panic(fmt.Sprintf("can't SetBit() a uint8 position %d", pos))
		}
	case uint16:
		if pos > 15 {
			panic(fmt.Sprintf("can't SetBit() a uint16 position %d", pos))
		}
	case uint32:
		if pos > 31 {
			panic(fmt.Sprintf("can't SetBit() a uint32 position %d", pos))
		}
	case uint64:
		if pos > 63 {
			panic(fmt.Sprintf("can't SetBit() a uint64 position %d", pos))
		}
	}
	if val {
		return store | (1 << pos)
	}

	return store & ^(1 << pos)
}

// ClearBit clears the bit at pos in store.
func ClearBit[U constraints.Unsigned](store U, pos uint8) U {
	store &^= (1 << pos)
	return store
}

// ClearBytes clears all the bytes at position from to position to.
func ClearBytes(b []byte, from, to uint8) {
	for i := from; i < to; i++ {
		b[from] = 0
	}
}

// ClearBits clears all bits from "from" until "to".
func ClearBits[U constraints.Unsigned](store U, from, to uint8) U {
	if from >= to {
		return store
	}

	width := to - from

	var m uint64
	if width == 64 {
		// Avoid shifting by 64 (illegal in Go)
		m = ^uint64(0)
	} else {
		m = (uint64(1)<<width - 1) << from
	}

	// Clear those bits
	return store &^ U(m)
}

// Mask creates a mask for setting, getting and clearing a set of bits.
// start is the bit location you wish to start at and end is the bit you wish to end at (exclusive).
// Index starts at 0.  So mask(1, 4) will create a mask that includes bits at location 1 to 3.
// If start >= end, this will panic.
func Mask[U constraints.Unsigned](start, end uint64) U {
	return U(setBits(uint(0), start, end))
}

// setBits sets all bits to 1 from start (inclusive) to end(exclusive).
// If this is not a number, aka it is a uintptr, this will panic. If start >= end, this will panic.
func setBits[I constraints.Unsigned](n I, start, end uint64) I {
	var size uint64
	switch any(n).(type) {
	case uint:
		size = bits.UintSize
	case uint8:
		size = 8
	case uint16:
		size = 16
	case uint32:
		size = 32
	case uint64:
		size = 64
	default:
		panic(fmt.Sprintf("n must be uint8/16/32/64, got %T", n))
	}

	if start >= end {
		panic("start cannot be >= end")
	}
	if end > size {
		panic(fmt.Sprintf("end %d exceeds width %d", end, size))
	}

	// Width of range
	width := end - start

	// Construct mask:
	// (1<<width)-1 produces width 1-bits
	// then shift left by start
	var mask uint64
	if width == 64 {
		// Special case: shifting by 64 is illegal
		mask = ^uint64(0)
	} else {
		mask = (uint64(1)<<width - 1) << start
	}

	// Apply mask
	return n | I(mask)
}

func BytesInBinary(bs []byte) string {
	buff := strings.Builder{}
	for _, n := range bs {
		buff.WriteString(fmt.Sprintf("% 08b", n))
	}
	return buff.String()
}
