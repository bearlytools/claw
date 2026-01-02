package conversions

import (
	"testing"
)

// FuzzBytesToNumUint8 fuzzes the BytesToNum function for uint8.
func FuzzBytesToNumUint8(f *testing.F) {
	f.Add([]byte{0})
	f.Add([]byte{127})
	f.Add([]byte{255})

	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) != 1 {
			return
		}

		// Should not panic
		ptr := BytesToNum[uint8](data)
		if *ptr != data[0] {
			t.Errorf("FuzzBytesToNumUint8: got %d, want %d", *ptr, data[0])
		}
	})
}

// FuzzBytesToNumInt8 fuzzes the BytesToNum function for int8.
func FuzzBytesToNumInt8(f *testing.F) {
	f.Add([]byte{0})
	f.Add([]byte{127})
	f.Add([]byte{128}) // -128

	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) != 1 {
			return
		}

		// Should not panic
		ptr := BytesToNum[int8](data)
		expected := int8(data[0])
		if *ptr != expected {
			t.Errorf("FuzzBytesToNumInt8: got %d, want %d", *ptr, expected)
		}
	})
}

// FuzzBytesToNumUint16 fuzzes the BytesToNum function for uint16.
func FuzzBytesToNumUint16(f *testing.F) {
	f.Add([]byte{0, 0})
	f.Add([]byte{255, 255})
	f.Add([]byte{1, 0})
	f.Add([]byte{0, 1})

	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) != 2 {
			return
		}

		// Should not panic
		ptr := BytesToNum[uint16](data)

		// Verify the value is accessible
		_ = *ptr
	})
}

// FuzzBytesToNumInt16 fuzzes the BytesToNum function for int16.
func FuzzBytesToNumInt16(f *testing.F) {
	f.Add([]byte{0, 0})
	f.Add([]byte{255, 127}) // max int16
	f.Add([]byte{0, 128})   // min int16

	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) != 2 {
			return
		}

		// Should not panic
		ptr := BytesToNum[int16](data)
		_ = *ptr
	})
}

// FuzzBytesToNumUint32 fuzzes the BytesToNum function for uint32.
func FuzzBytesToNumUint32(f *testing.F) {
	f.Add([]byte{0, 0, 0, 0})
	f.Add([]byte{255, 255, 255, 255})
	f.Add([]byte{1, 0, 0, 0})

	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) != 4 {
			return
		}

		// Should not panic
		ptr := BytesToNum[uint32](data)
		_ = *ptr
	})
}

// FuzzBytesToNumInt32 fuzzes the BytesToNum function for int32.
func FuzzBytesToNumInt32(f *testing.F) {
	f.Add([]byte{0, 0, 0, 0})
	f.Add([]byte{255, 255, 255, 127}) // max int32
	f.Add([]byte{0, 0, 0, 128})       // min int32

	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) != 4 {
			return
		}

		// Should not panic
		ptr := BytesToNum[int32](data)
		_ = *ptr
	})
}

// FuzzBytesToNumUint64 fuzzes the BytesToNum function for uint64.
func FuzzBytesToNumUint64(f *testing.F) {
	f.Add([]byte{0, 0, 0, 0, 0, 0, 0, 0})
	f.Add([]byte{255, 255, 255, 255, 255, 255, 255, 255})
	f.Add([]byte{1, 0, 0, 0, 0, 0, 0, 0})

	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) != 8 {
			return
		}

		// Should not panic
		ptr := BytesToNum[uint64](data)
		_ = *ptr
	})
}

// FuzzBytesToNumInt64 fuzzes the BytesToNum function for int64.
func FuzzBytesToNumInt64(f *testing.F) {
	f.Add([]byte{0, 0, 0, 0, 0, 0, 0, 0})
	f.Add([]byte{255, 255, 255, 255, 255, 255, 255, 127}) // max int64
	f.Add([]byte{0, 0, 0, 0, 0, 0, 0, 128})               // min int64

	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) != 8 {
			return
		}

		// Should not panic
		ptr := BytesToNum[int64](data)
		_ = *ptr
	})
}

// FuzzNumToBytesRoundTrip fuzzes the NumToBytes/BytesToNum roundtrip for uint32.
func FuzzNumToBytesRoundTrip(f *testing.F) {
	f.Add(uint32(0))
	f.Add(uint32(1))
	f.Add(uint32(0xFFFFFFFF))
	f.Add(uint32(0x12345678))

	f.Fuzz(func(t *testing.T, val uint32) {
		// Get bytes from the value
		b := NumToBytes(&val)
		if len(b) != 4 {
			t.Errorf("FuzzNumToBytesRoundTrip: expected 4 bytes, got %d", len(b))
			return
		}

		// Convert back
		ptr := BytesToNum[uint32](b)
		if *ptr != val {
			t.Errorf("FuzzNumToBytesRoundTrip: roundtrip failed: got %d, want %d", *ptr, val)
		}
	})
}

// FuzzByteSlice2String fuzzes the ByteSlice2String function.
func FuzzByteSlice2String(f *testing.F) {
	f.Add([]byte{})
	f.Add([]byte("hello"))
	f.Add([]byte("hello\x00world"))
	f.Add([]byte{0, 1, 2, 3, 4, 5})
	f.Add([]byte("unicode: 日本語"))

	f.Fuzz(func(t *testing.T, data []byte) {
		// Make a copy since ByteSlice2String says the original is no longer safe to use
		dataCopy := make([]byte, len(data))
		copy(dataCopy, data)

		// Should not panic
		s := ByteSlice2String(dataCopy)

		// Verify length matches
		if len(s) != len(data) {
			t.Errorf("FuzzByteSlice2String: length mismatch: got %d, want %d", len(s), len(data))
		}
	})
}

// FuzzUnsafeGetBytes fuzzes the UnsafeGetBytes function.
func FuzzUnsafeGetBytes(f *testing.F) {
	f.Add("")
	f.Add("hello")
	f.Add("hello\x00world")
	f.Add("unicode: 日本語")
	f.Add("a")
	f.Add(string(make([]byte, 1000)))

	f.Fuzz(func(t *testing.T, s string) {
		// Should not panic
		b := UnsafeGetBytes(s)

		// Empty string should return nil
		if s == "" {
			if b != nil {
				t.Errorf("FuzzUnsafeGetBytes: expected nil for empty string, got %v", b)
			}
			return
		}

		// Verify length matches
		if len(b) != len(s) {
			t.Errorf("FuzzUnsafeGetBytes: length mismatch: got %d, want %d", len(b), len(s))
		}

		// Verify bytes match
		for i := 0; i < len(s); i++ {
			if b[i] != s[i] {
				t.Errorf("FuzzUnsafeGetBytes: byte mismatch at %d: got %d, want %d", i, b[i], s[i])
				break
			}
		}
	})
}

// FuzzStringBytesRoundTrip fuzzes the roundtrip of ByteSlice2String and UnsafeGetBytes.
func FuzzStringBytesRoundTrip(f *testing.F) {
	f.Add("hello")
	f.Add("")
	f.Add("test string with spaces")
	f.Add("日本語")

	f.Fuzz(func(t *testing.T, original string) {
		if original == "" {
			return // Skip empty strings since UnsafeGetBytes returns nil
		}

		// Get bytes from string
		b := UnsafeGetBytes(original)

		// Make a copy since we can't modify the original
		bCopy := make([]byte, len(b))
		copy(bCopy, b)

		// Convert back to string
		s := ByteSlice2String(bCopy)

		// Should be equal
		if s != original {
			t.Errorf("FuzzStringBytesRoundTrip: roundtrip failed: got %q, want %q", s, original)
		}
	})
}
