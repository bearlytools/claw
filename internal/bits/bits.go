package bits

import (
	"fmt"
	"log"
	"math/bits"
	"strings"

	"github.com/bearlytools/claw/internal/conversions"
	"golang.org/x/exp/constraints"
)

type PtrUnsigned interface {
	*uint8 | *uint16 | *uint32 | *uint64
}

// SetValue stores "val" in unsigned number "store" starting at bit "start" and
// ending at bit "end" (exclusive). If start >= end, this panics.
func SetValue[I, U constraints.Unsigned](val I, store U, start, end uint64) U {
	if start >= end {
		panic("start cannot be > end")
	}

	c := U(val) << start

	return store | c
}

// SetValue stores "val" in "store" starting at bit "start" and
// ending at bit "end" (exclusive). Any error condition in this causes a panic, for example:
// start >= end, store cannot contain start and end, store isn't big enough for val, ...
func SetValueBytes[I constraints.Unsigned](val I, store []byte, start, end uint64) {
	log.Printf("setting[%d:%d] to value: %d", start, end, val)
	if start >= end {
		panic("start cannot be > end")
	}

	l := len(store)
	if l > 8 {
		panic("SetValueBytes() cannot receive a len(store) > 8, as 8 bytes stores our maximum integer size, 64 bits")
	}
	if start/64 > uint64(l) {
		panic(fmt.Sprintf("SetValueBytes() cannot store a value with start %d and len(store) == %d", start, l))
	}
	if end/64 > uint64(l) {
		panic(fmt.Sprintf("SetValueBytes() cannot store a value with end %d and len(store) == %d", end, l))
	}

	switch any(val).(type) {
	case uint8:
		if l < 1 {
			panic("SetValueBytes() can only store a uint8 value if len(store) == 1")
		}
		switch l {
		case 1:
			u := conversions.BytesToNum[uint8](store)
			log.Println("our number: ", *u)
			c := val << start
			*u = *u | uint8(c)
			log.Println("our number after: ", *u)
		case 2:
			u := conversions.BytesToNum[uint16](store)
			c := val << start
			*u = *u | uint16(c)
		case 4:
			u := conversions.BytesToNum[uint32](store)
			c := val << start
			*u = *u | uint32(c)
		case 8:
			u := conversions.BytesToNum[uint64](store)
			c := val << start
			*u = *u | uint64(c)
		default:
			panic("SetValueBytes[uint8]() must receive a len(store) == 1 | 2 | 4 | 8")
		}
	case uint16:
		l := len(store)
		if l < 2 {
			panic("SetValueBytes() can only store a uint16 value if len(store) == 2")
		}
		switch l {
		case 2:
			u := conversions.BytesToNum[uint16](store)
			c := val << start
			*u = *u | uint16(c)
		case 4:
			u := conversions.BytesToNum[uint32](store)
			c := val << start
			*u = *u | uint32(c)
		case 8:
			u := conversions.BytesToNum[uint64](store)
			c := val << start
			*u = *u | uint64(c)
		default:
			panic("SetValueBytes[uint16]() must receive a len(store) == 2 | 4 | 8")
		}
	case uint32:
		l := len(store)
		if l < 4 {
			panic("SetValueBytes() can only store a uint32 value if len(store) == 4")
		}
		switch l {
		case 4:
			u := conversions.BytesToNum[uint32](store)
			c := val << start
			*u = *u | uint32(c)
		case 8:
			u := conversions.BytesToNum[uint64](store)
			c := val << start
			*u = *u | uint64(c)
		default:
			panic("SetValueBytes[uint32]() must receive a len(store) == 4 | 8")
		}
	case uint64:
		l := len(store)
		if l < 8 {
			panic("SetValueBytes() can only store a uint64 value if len(store) == 8")
		}
		switch l {
		case 8:
			u := conversions.BytesToNum[uint64](store)
			c := val << start
			*u = *u | uint64(c)
		default:
			panic("SetValueBytes[uint64]() must receive a len(store) == 4 | 8")
		}
	default:
		panic("you've gone'n done it now!")
	}
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

// ClearBits clears all bits from "from" until "to".
func ClearBits[U constraints.Unsigned](store U, from, to uint8) U {
	for i := from; i < to; i++ {
		store = ClearBit(store, i)
	}
	return store
}

// Mask creates a mask for setting, getting and clearing a set of bits.
// start is the bit location you wish to start at and end is the bit you wish to end at (exclusive).
// Index starts at 0.  So mask(1, 4) will create a mask that includes bits at location 1 to 3.
// If start >= end, this will panic.
func Mask[U constraints.Unsigned](start, end uint64) U {
	return U(setBits(uint(0), start, end))
}

// SetBits sets all bits to 1 from start (inclusive) to end(exclusive).
// This is not particularly fast, so best to use at init time. If this is not a number,
// aka it is a uintptr, this will panic. If start >= end, this will panic.
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
		panic(fmt.Sprintf("n must be of type uint8/uint16/uint32/uint64, was %T", n))
	}

	if start >= end {
		panic("start cannot be >= end")
	}
	if end > size {
		panic(fmt.Sprintf("end cannot be %d, as that is the largest amount of bits in an %d bit number", end, size))
	}

	var r uint
	for x := start; x < end; x++ {
		c := (uint(1) << x)
		r = r | c
	}

	return n | I(r)
}

func BytesInBinary(bs []byte) string {
	buff := strings.Builder{}
	for _, n := range bs {
		buff.WriteString(fmt.Sprintf("% 08b", n))
	}
	return buff.String()
}
