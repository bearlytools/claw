package binary

import (
	"bytes"
	"testing"
)

// FuzzGetInt8 fuzzes the int8 get function.
func FuzzGetInt8(f *testing.F) {
	f.Add([]byte{0})
	f.Add([]byte{127})
	f.Add([]byte{128})
	f.Add([]byte{255})
	f.Add([]byte{1})

	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) < 1 {
			return
		}
		// Should not panic
		result := GetInt8(data)

		// Verify round-trip
		out := make([]byte, 1)
		PutInt8(out, result)
		if out[0] != data[0] {
			t.Errorf("FuzzGetInt8: round-trip failed: got %d, want %d", out[0], data[0])
		}
	})
}

// FuzzGetInt16 fuzzes the int16 get function.
func FuzzGetInt16(f *testing.F) {
	f.Add([]byte{0, 0})
	f.Add([]byte{255, 127})
	f.Add([]byte{0, 128})
	f.Add([]byte{255, 255})
	f.Add([]byte{1, 0})

	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) < 2 {
			return
		}
		// Should not panic
		result := GetInt16(data)

		// Verify round-trip
		out := make([]byte, 2)
		PutInt16(out, result)
		if !bytes.Equal(out, data[:2]) {
			t.Errorf("FuzzGetInt16: round-trip failed")
		}
	})
}

// FuzzGetInt32 fuzzes the int32 get function.
func FuzzGetInt32(f *testing.F) {
	f.Add([]byte{0, 0, 0, 0})
	f.Add([]byte{255, 255, 255, 127})
	f.Add([]byte{0, 0, 0, 128})
	f.Add([]byte{255, 255, 255, 255})
	f.Add([]byte{1, 0, 0, 0})

	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) < 4 {
			return
		}
		// Should not panic
		result := GetInt32(data)

		// Verify round-trip
		out := make([]byte, 4)
		PutInt32(out, result)
		if !bytes.Equal(out, data[:4]) {
			t.Errorf("FuzzGetInt32: round-trip failed")
		}
	})
}

// FuzzGetInt64 fuzzes the int64 get function.
func FuzzGetInt64(f *testing.F) {
	f.Add([]byte{0, 0, 0, 0, 0, 0, 0, 0})
	f.Add([]byte{255, 255, 255, 255, 255, 255, 255, 127})
	f.Add([]byte{0, 0, 0, 0, 0, 0, 0, 128})
	f.Add([]byte{255, 255, 255, 255, 255, 255, 255, 255})
	f.Add([]byte{1, 0, 0, 0, 0, 0, 0, 0})

	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) < 8 {
			return
		}
		// Should not panic
		result := GetInt64(data)

		// Verify round-trip
		out := make([]byte, 8)
		PutInt64(out, result)
		if !bytes.Equal(out, data[:8]) {
			t.Errorf("FuzzGetInt64: round-trip failed")
		}
	})
}

// FuzzGetUint8 fuzzes the uint8 get function.
func FuzzGetUint8(f *testing.F) {
	f.Add([]byte{0})
	f.Add([]byte{127})
	f.Add([]byte{128})
	f.Add([]byte{255})

	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) < 1 {
			return
		}
		// Should not panic
		result := GetUint8(data)

		// Verify round-trip
		out := make([]byte, 1)
		PutUint8(out, result)
		if out[0] != data[0] {
			t.Errorf("FuzzGetUint8: round-trip failed: got %d, want %d", out[0], data[0])
		}
	})
}

// FuzzGetUint16 fuzzes the uint16 get function.
func FuzzGetUint16(f *testing.F) {
	f.Add([]byte{0, 0})
	f.Add([]byte{255, 255})
	f.Add([]byte{0, 128})
	f.Add([]byte{1, 0})

	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) < 2 {
			return
		}
		// Should not panic
		result := GetUint16(data)

		// Verify round-trip
		out := make([]byte, 2)
		PutUint16(out, result)
		if !bytes.Equal(out, data[:2]) {
			t.Errorf("FuzzGetUint16: round-trip failed")
		}
	})
}

// FuzzGetUint32 fuzzes the uint32 get function.
func FuzzGetUint32(f *testing.F) {
	f.Add([]byte{0, 0, 0, 0})
	f.Add([]byte{255, 255, 255, 255})
	f.Add([]byte{0, 0, 0, 128})
	f.Add([]byte{1, 0, 0, 0})

	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) < 4 {
			return
		}
		// Should not panic
		result := GetUint32(data)

		// Verify round-trip
		out := make([]byte, 4)
		PutUint32(out, result)
		if !bytes.Equal(out, data[:4]) {
			t.Errorf("FuzzGetUint32: round-trip failed")
		}
	})
}

// FuzzGetUint64 fuzzes the uint64 get function.
func FuzzGetUint64(f *testing.F) {
	f.Add([]byte{0, 0, 0, 0, 0, 0, 0, 0})
	f.Add([]byte{255, 255, 255, 255, 255, 255, 255, 255})
	f.Add([]byte{0, 0, 0, 0, 0, 0, 0, 128})
	f.Add([]byte{1, 0, 0, 0, 0, 0, 0, 0})

	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) < 8 {
			return
		}
		// Should not panic
		result := GetUint64(data)

		// Verify round-trip
		out := make([]byte, 8)
		PutUint64(out, result)
		if !bytes.Equal(out, data[:8]) {
			t.Errorf("FuzzGetUint64: round-trip failed")
		}
	})
}

// FuzzGetBufferInt8 fuzzes the buffer int8 read function.
func FuzzGetBufferInt8(f *testing.F) {
	f.Add([]byte{0})
	f.Add([]byte{127})
	f.Add([]byte{128})
	f.Add([]byte{255})

	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) < 1 {
			return
		}
		// Should not panic
		result, err := GetBufferInt8(bytes.NewReader(data))
		if err != nil {
			return
		}

		// Verify value
		expected := GetInt8(data)
		if result != expected {
			t.Errorf("FuzzGetBufferInt8: got %d, want %d", result, expected)
		}
	})
}

// FuzzGetBufferUint16 fuzzes the buffer uint16 read function.
func FuzzGetBufferUint16(f *testing.F) {
	f.Add([]byte{0, 0})
	f.Add([]byte{255, 255})
	f.Add([]byte{1, 0})

	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) < 2 {
			return
		}
		// Should not panic
		result, err := GetBufferUint16(bytes.NewReader(data))
		if err != nil {
			return
		}

		// Verify value
		expected := GetUint16(data)
		if result != expected {
			t.Errorf("FuzzGetBufferUint16: got %d, want %d", result, expected)
		}
	})
}

// FuzzGetBufferUint32 fuzzes the buffer uint32 read function.
func FuzzGetBufferUint32(f *testing.F) {
	f.Add([]byte{0, 0, 0, 0})
	f.Add([]byte{255, 255, 255, 255})
	f.Add([]byte{1, 0, 0, 0})

	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) < 4 {
			return
		}
		// Should not panic
		result, err := GetBufferUint32(bytes.NewReader(data))
		if err != nil {
			return
		}

		// Verify value
		expected := GetUint32(data)
		if result != expected {
			t.Errorf("FuzzGetBufferUint32: got %d, want %d", result, expected)
		}
	})
}

// FuzzGetBufferUint64 fuzzes the buffer uint64 read function.
func FuzzGetBufferUint64(f *testing.F) {
	f.Add([]byte{0, 0, 0, 0, 0, 0, 0, 0})
	f.Add([]byte{255, 255, 255, 255, 255, 255, 255, 255})
	f.Add([]byte{1, 0, 0, 0, 0, 0, 0, 0})

	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) < 8 {
			return
		}
		// Should not panic
		result, err := GetBufferUint64(bytes.NewReader(data))
		if err != nil {
			return
		}

		// Verify value
		expected := GetUint64(data)
		if result != expected {
			t.Errorf("FuzzGetBufferUint64: got %d, want %d", result, expected)
		}
	})
}

// FuzzPutBufferInt32 fuzzes the buffer int32 write function.
func FuzzPutBufferInt32(f *testing.F) {
	f.Add(int32(0))
	f.Add(int32(1))
	f.Add(int32(-1))
	f.Add(int32(2147483647))
	f.Add(int32(-2147483648))

	f.Fuzz(func(t *testing.T, val int32) {
		var buf bytes.Buffer
		// Should not panic
		err := PutBufferInt32(&buf, val)
		if err != nil {
			t.Errorf("FuzzPutBufferInt32: unexpected error: %v", err)
			return
		}

		// Verify round-trip
		result := GetInt32(buf.Bytes())
		if result != val {
			t.Errorf("FuzzPutBufferInt32: round-trip failed: got %d, want %d", result, val)
		}
	})
}

// FuzzPutBufferInt64 fuzzes the buffer int64 write function.
func FuzzPutBufferInt64(f *testing.F) {
	f.Add(int64(0))
	f.Add(int64(1))
	f.Add(int64(-1))
	f.Add(int64(9223372036854775807))
	f.Add(int64(-9223372036854775808))

	f.Fuzz(func(t *testing.T, val int64) {
		var buf bytes.Buffer
		// Should not panic
		err := PutBufferInt64(&buf, val)
		if err != nil {
			t.Errorf("FuzzPutBufferInt64: unexpected error: %v", err)
			return
		}

		// Verify round-trip
		result := GetInt64(buf.Bytes())
		if result != val {
			t.Errorf("FuzzPutBufferInt64: round-trip failed: got %d, want %d", result, val)
		}
	})
}

// FuzzGenericGetPut fuzzes the generic Get/Put functions with various integer types.
func FuzzGenericGetPut(f *testing.F) {
	f.Add([]byte{1, 2, 3, 4, 5, 6, 7, 8})
	f.Add([]byte{0, 0, 0, 0, 0, 0, 0, 0})
	f.Add([]byte{255, 255, 255, 255, 255, 255, 255, 255})

	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) < 8 {
			return
		}

		// Test Get[uint8]
		v8 := Get[uint8](data)
		out8 := make([]byte, 1)
		Put(out8, v8)
		if out8[0] != data[0] {
			t.Errorf("FuzzGenericGetPut: uint8 mismatch")
		}

		// Test Get[uint16]
		v16 := Get[uint16](data)
		out16 := make([]byte, 2)
		Put(out16, v16)
		if !bytes.Equal(out16, data[:2]) {
			t.Errorf("FuzzGenericGetPut: uint16 mismatch")
		}

		// Test Get[uint32]
		v32 := Get[uint32](data)
		out32 := make([]byte, 4)
		Put(out32, v32)
		if !bytes.Equal(out32, data[:4]) {
			t.Errorf("FuzzGenericGetPut: uint32 mismatch")
		}

		// Test Get[uint64]
		v64 := Get[uint64](data)
		out64 := make([]byte, 8)
		Put(out64, v64)
		if !bytes.Equal(out64, data[:8]) {
			t.Errorf("FuzzGenericGetPut: uint64 mismatch")
		}
	})
}
