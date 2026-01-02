package bits

import (
	"testing"
)

// FuzzSetGetValue fuzzes the SetValue/GetValue round-trip.
func FuzzSetGetValue(f *testing.F) {
	// (value, start, end)
	f.Add(uint64(0), uint64(0), uint64(8))
	f.Add(uint64(255), uint64(0), uint64(8))
	f.Add(uint64(15), uint64(4), uint64(8))
	f.Add(uint64(1), uint64(0), uint64(1))
	f.Add(uint64(1), uint64(7), uint64(8))
	f.Add(uint64(0xFFFF), uint64(0), uint64(16))
	f.Add(uint64(0xF), uint64(28), uint64(32))

	f.Fuzz(func(t *testing.T, val, start, end uint64) {
		// Ensure valid range
		if start >= end || end > 64 {
			return
		}
		// Ensure value fits in the range
		width := end - start
		maxVal := uint64(1)<<width - 1
		if val > maxVal {
			val = val & maxVal
		}

		// Should not panic
		store := SetValue(val, uint64(0), start, end)

		// Create mask and retrieve value
		mask := Mask[uint64](start, end)
		retrieved := GetValue[uint64, uint64](store, mask, start)

		if retrieved != val {
			t.Errorf("FuzzSetGetValue: round-trip failed: got %d, want %d (start=%d, end=%d)", retrieved, val, start, end)
		}
	})
}

// FuzzSetGetBit fuzzes the SetBit/GetBit functions.
func FuzzSetGetBit(f *testing.F) {
	f.Add(uint8(0), uint8(0), true)
	f.Add(uint8(0), uint8(7), true)
	f.Add(uint8(255), uint8(0), false)
	f.Add(uint8(255), uint8(7), false)
	f.Add(uint8(0), uint8(3), true)
	f.Add(uint8(128), uint8(7), false)

	f.Fuzz(func(t *testing.T, store, pos uint8, val bool) {
		// Limit position to valid range
		if pos > 7 {
			return
		}

		// Should not panic
		newStore := SetBit(store, pos, val)
		retrieved := GetBit(newStore, pos)

		if retrieved != val {
			t.Errorf("FuzzSetGetBit: round-trip failed: got %v, want %v (store=%d, pos=%d)", retrieved, val, store, pos)
		}
	})
}

// FuzzSetGetBit16 fuzzes SetBit/GetBit with uint16.
func FuzzSetGetBit16(f *testing.F) {
	f.Add(uint16(0), uint8(0), true)
	f.Add(uint16(0), uint8(15), true)
	f.Add(uint16(65535), uint8(0), false)
	f.Add(uint16(65535), uint8(15), false)

	f.Fuzz(func(t *testing.T, store uint16, pos uint8, val bool) {
		if pos > 15 {
			return
		}

		newStore := SetBit(store, pos, val)
		retrieved := GetBit(newStore, pos)

		if retrieved != val {
			t.Errorf("FuzzSetGetBit16: round-trip failed: got %v, want %v", retrieved, val)
		}
	})
}

// FuzzSetGetBit32 fuzzes SetBit/GetBit with uint32.
func FuzzSetGetBit32(f *testing.F) {
	f.Add(uint32(0), uint8(0), true)
	f.Add(uint32(0), uint8(31), true)
	f.Add(uint32(0xFFFFFFFF), uint8(0), false)

	f.Fuzz(func(t *testing.T, store uint32, pos uint8, val bool) {
		if pos > 31 {
			return
		}

		newStore := SetBit(store, pos, val)
		retrieved := GetBit(newStore, pos)

		if retrieved != val {
			t.Errorf("FuzzSetGetBit32: round-trip failed: got %v, want %v", retrieved, val)
		}
	})
}

// FuzzSetGetBit64 fuzzes SetBit/GetBit with uint64.
func FuzzSetGetBit64(f *testing.F) {
	f.Add(uint64(0), uint8(0), true)
	f.Add(uint64(0), uint8(63), true)
	f.Add(uint64(0xFFFFFFFFFFFFFFFF), uint8(0), false)

	f.Fuzz(func(t *testing.T, store uint64, pos uint8, val bool) {
		if pos > 63 {
			return
		}

		newStore := SetBit(store, pos, val)
		retrieved := GetBit(newStore, pos)

		if retrieved != val {
			t.Errorf("FuzzSetGetBit64: round-trip failed: got %v, want %v", retrieved, val)
		}
	})
}

// FuzzClearBit fuzzes the ClearBit function.
func FuzzClearBit(f *testing.F) {
	f.Add(uint8(255), uint8(0))
	f.Add(uint8(255), uint8(7))
	f.Add(uint8(128), uint8(7))
	f.Add(uint8(1), uint8(0))

	f.Fuzz(func(t *testing.T, store, pos uint8) {
		if pos > 7 {
			return
		}

		// Should not panic
		cleared := ClearBit(store, pos)

		// Verify bit is cleared
		if GetBit(cleared, pos) {
			t.Errorf("FuzzClearBit: bit still set after clearing (store=%d, pos=%d)", store, pos)
		}
	})
}

// FuzzClearBits fuzzes the ClearBits function.
func FuzzClearBits(f *testing.F) {
	f.Add(uint8(255), uint8(0), uint8(4))
	f.Add(uint8(255), uint8(0), uint8(8))
	f.Add(uint8(255), uint8(4), uint8(8))
	f.Add(uint8(0xF0), uint8(4), uint8(8))

	f.Fuzz(func(t *testing.T, store, from, to uint8) {
		if from >= to || to > 8 {
			return
		}

		// Should not panic
		cleared := ClearBits(store, from, to)

		// Verify bits are cleared in range
		for pos := from; pos < to; pos++ {
			if GetBit(cleared, pos) {
				t.Errorf("FuzzClearBits: bit %d still set after clearing (store=%d, from=%d, to=%d)", pos, store, from, to)
			}
		}
	})
}

// FuzzClearBits16 fuzzes ClearBits with uint16.
func FuzzClearBits16(f *testing.F) {
	f.Add(uint16(0xFFFF), uint8(0), uint8(8))
	f.Add(uint16(0xFFFF), uint8(8), uint8(16))
	f.Add(uint16(0xFF00), uint8(8), uint8(16))

	f.Fuzz(func(t *testing.T, store uint16, from, to uint8) {
		if from >= to || to > 16 {
			return
		}

		cleared := ClearBits(store, from, to)

		for pos := from; pos < to; pos++ {
			if GetBit(cleared, pos) {
				t.Errorf("FuzzClearBits16: bit %d still set after clearing", pos)
			}
		}
	})
}

// FuzzClearBits64 fuzzes ClearBits with uint64.
func FuzzClearBits64(f *testing.F) {
	f.Add(uint64(0xFFFFFFFFFFFFFFFF), uint8(0), uint8(32))
	f.Add(uint64(0xFFFFFFFFFFFFFFFF), uint8(32), uint8(64))

	f.Fuzz(func(t *testing.T, store uint64, from, to uint8) {
		if from >= to || to > 64 {
			return
		}

		cleared := ClearBits(store, from, to)

		for pos := from; pos < to; pos++ {
			if GetBit(cleared, pos) {
				t.Errorf("FuzzClearBits64: bit %d still set after clearing", pos)
			}
		}
	})
}

// FuzzMask fuzzes the Mask function.
func FuzzMask(f *testing.F) {
	f.Add(uint64(0), uint64(8))
	f.Add(uint64(0), uint64(1))
	f.Add(uint64(4), uint64(8))
	f.Add(uint64(0), uint64(64))
	f.Add(uint64(32), uint64(64))

	f.Fuzz(func(t *testing.T, start, end uint64) {
		if start >= end || end > 64 {
			return
		}

		// Should not panic
		mask := Mask[uint64](start, end)

		// Verify mask has correct bits set
		for i := uint64(0); i < 64; i++ {
			bitSet := (mask & (1 << i)) != 0
			shouldBeSet := i >= start && i < end
			if bitSet != shouldBeSet {
				t.Errorf("FuzzMask: bit %d: got %v, want %v (start=%d, end=%d)", i, bitSet, shouldBeSet, start, end)
			}
		}
	})
}

// FuzzMask8 fuzzes Mask with uint8.
func FuzzMask8(f *testing.F) {
	f.Add(uint64(0), uint64(4))
	f.Add(uint64(0), uint64(8))
	f.Add(uint64(4), uint64(8))

	f.Fuzz(func(t *testing.T, start, end uint64) {
		if start >= end || end > 8 {
			return
		}

		mask := Mask[uint8](start, end)

		for i := uint64(0); i < 8; i++ {
			bitSet := (mask & (1 << i)) != 0
			shouldBeSet := i >= start && i < end
			if bitSet != shouldBeSet {
				t.Errorf("FuzzMask8: bit %d: got %v, want %v", i, bitSet, shouldBeSet)
			}
		}
	})
}

// FuzzClearBytes fuzzes the ClearBytes function.
func FuzzClearBytes(f *testing.F) {
	f.Add([]byte{1, 2, 3, 4, 5}, uint8(0), uint8(3))
	f.Add([]byte{255, 255, 255, 255}, uint8(1), uint8(3))
	f.Add([]byte{1, 2, 3, 4, 5, 6, 7, 8}, uint8(0), uint8(8))

	f.Fuzz(func(t *testing.T, data []byte, from, to uint8) {
		if len(data) == 0 {
			return
		}
		if from >= to || to > uint8(len(data)) {
			return
		}

		// Make a copy to test
		dataCopy := make([]byte, len(data))
		copy(dataCopy, data)

		// Should not panic
		ClearBytes(dataCopy, from, to)

		// Verify bytes are cleared in range
		for i := from; i < to; i++ {
			if dataCopy[i] != 0 {
				t.Errorf("FuzzClearBytes: byte %d not cleared: got %d, want 0", i, dataCopy[i])
			}
		}

		// Verify bytes outside range are unchanged
		for i := uint8(0); i < from; i++ {
			if dataCopy[i] != data[i] {
				t.Errorf("FuzzClearBytes: byte %d changed: got %d, want %d", i, dataCopy[i], data[i])
			}
		}
		for i := to; i < uint8(len(data)); i++ {
			if dataCopy[i] != data[i] {
				t.Errorf("FuzzClearBytes: byte %d changed: got %d, want %d", i, dataCopy[i], data[i])
			}
		}
	})
}

// FuzzBytesInBinary fuzzes the BytesInBinary function.
func FuzzBytesInBinary(f *testing.F) {
	f.Add([]byte{})
	f.Add([]byte{0})
	f.Add([]byte{255})
	f.Add([]byte{0, 255})
	f.Add([]byte{1, 2, 3, 4})
	f.Add([]byte{0xAA, 0x55, 0xFF, 0x00})

	f.Fuzz(func(t *testing.T, data []byte) {
		// Should not panic
		result := BytesInBinary(data)

		// Verify result contains only valid characters (space, 0, 1)
		for _, r := range result {
			if r != ' ' && r != '0' && r != '1' {
				t.Errorf("FuzzBytesInBinary: unexpected character %q in result", r)
			}
		}
	})
}
