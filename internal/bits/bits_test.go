package bits

import (
	"math"
	"testing"
)

func TestSetValue(t *testing.T) {
	storeStart := uint8(1)
	// We start at using bit 1 (we index at 0), so that we can set bit 0 to 1
	// (meaning our storage number starts at value 1). That way we can make sure
	// we are only retrieving the values we think we are.
	for start := uint64(1); start < 8; start++ {
		for end := start + 1; end < 8; end++ {
			maxBits := end - start
			maxValue := uint8(math.Pow(2, float64(maxBits)))
			for val := uint8(0); val < maxValue; val++ {
				store := SetValue(val, storeStart, start, end)
				bitMask := Mask[uint8](start, end)
				got := GetValue[uint8, uint8](store, bitMask, start)

				if got != val {
					t.Fatalf("TestSetValue(start: %d, end: %d, val: %d): got %d, want %d", start, end, val, got, val)
				}
			}
		}
	}
}

func TestSetBits(t *testing.T) {
	// Tests we can set all bits that we expect.
	for start := 0; start < 8; start++ {
		for end := start + 1; end < 8; end++ {
			got := setBits(uint8(0), uint64(start), uint64(end))
			var want uint8
			if end-start == 1 {
				want = 1 << start
			} else {
				for x := start; x < end; x++ {
					want += 1 << x
				}
			}

			if got != want {
				t.Fatalf("TestSetBits(start: %d, end: %d): got %d, want %d", start, end, got, want)
			}
		}
	}

	// Test we can ignore existing bits.
	// 10000001 start = 65
	// 10111101 end  = 125
	got := setBits(uint8(65), uint64(2), uint64(6))
	if got != 125 {
		t.Fatalf("TestSetBits(num: %d, start: %d, end: %d): got %d, want %d", 65, 2, 6, got, 125)
	}
}

func TestGetSetBit(t *testing.T) {
	for i := uint8(0); i < 8; i++ {
		var store uint8

		store = SetBit(store, i, true)
		if !GetBit(store, i) {
			t.Fatalf("TestGetSetBit(set bit %d): got false, want true", i)
		}
		if store != 1<<i {
			t.Fatalf("TestGetSetBit(set bit %d): store value was %d, expected %d", i, store, 1<<i)
		}
	}

	for i := uint8(0); i < 8; i++ {
		var store uint8 = 255

		store = SetBit(store, i, false)
		if GetBit(store, i) {
			t.Fatalf("TestGetSetBit(set bit %d): got true, want false", i)
		}
	}
}
