package patch

import (
	"encoding/binary"
	"testing"

	"github.com/bearlytools/claw/clawc/languages/go/field"
	"github.com/bearlytools/claw/clawc/languages/go/mapping"
	"github.com/bearlytools/claw/clawc/languages/go/segment"
	"github.com/bearlytools/claw/languages/go/patch/msgs"
)

// testMappingForFuzz creates a mapping with various field types for fuzzing.
func testMappingForFuzz() *mapping.Map {
	return &mapping.Map{
		Fields: []*mapping.FieldDescr{
			{Name: "bool", Type: field.FTBool, FieldNum: 0},
			{Name: "int8", Type: field.FTInt8, FieldNum: 1},
			{Name: "int16", Type: field.FTInt16, FieldNum: 2},
			{Name: "int32", Type: field.FTInt32, FieldNum: 3},
			{Name: "int64", Type: field.FTInt64, FieldNum: 4},
			{Name: "uint8", Type: field.FTUint8, FieldNum: 5},
			{Name: "uint16", Type: field.FTUint16, FieldNum: 6},
			{Name: "uint32", Type: field.FTUint32, FieldNum: 7},
			{Name: "uint64", Type: field.FTUint64, FieldNum: 8},
			{Name: "float32", Type: field.FTFloat32, FieldNum: 9},
			{Name: "float64", Type: field.FTFloat64, FieldNum: 10},
			{Name: "string", Type: field.FTString, FieldNum: 11},
			{Name: "bytes", Type: field.FTBytes, FieldNum: 12},
			{Name: "listBools", Type: field.FTListBools, FieldNum: 13},
			{Name: "listInt32", Type: field.FTListInt32, FieldNum: 14},
			{Name: "listStrings", Type: field.FTListStrings, FieldNum: 15},
		},
	}
}

// FuzzApplySet fuzzes the applySet function with various field types and data.
func FuzzApplySet(f *testing.F) {
	m := testMappingForFuzz()

	// Bool field (1 byte)
	f.Add(uint16(0), uint8(field.FTBool), []byte{0})
	f.Add(uint16(0), uint8(field.FTBool), []byte{1})

	// Int8 field (1 byte)
	f.Add(uint16(1), uint8(field.FTInt8), []byte{127})
	f.Add(uint16(1), uint8(field.FTInt8), []byte{0x80}) // -128

	// Int16 field (2 bytes)
	f.Add(uint16(2), uint8(field.FTInt16), []byte{0x00, 0x00})
	f.Add(uint16(2), uint8(field.FTInt16), []byte{0xff, 0x7f}) // 32767

	// Int32 field (4 bytes)
	f.Add(uint16(3), uint8(field.FTInt32), []byte{0x00, 0x00, 0x00, 0x00})
	f.Add(uint16(3), uint8(field.FTInt32), []byte{0xff, 0xff, 0xff, 0x7f}) // max int32

	// Int64 field (8 bytes)
	f.Add(uint16(4), uint8(field.FTInt64), []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})

	// String field
	f.Add(uint16(11), uint8(field.FTString), []byte("hello"))
	f.Add(uint16(11), uint8(field.FTString), []byte(""))

	// Bytes field
	f.Add(uint16(12), uint8(field.FTBytes), []byte{0xde, 0xad, 0xbe, 0xef})

	// Edge cases: too short data
	f.Add(uint16(2), uint8(field.FTInt16), []byte{0x00}) // int16 needs 2 bytes
	f.Add(uint16(3), uint8(field.FTInt32), []byte{0x00}) // int32 needs 4 bytes
	f.Add(uint16(4), uint8(field.FTInt64), []byte{0x00}) // int64 needs 8 bytes

	// Empty data
	f.Add(uint16(0), uint8(field.FTBool), []byte{})
	f.Add(uint16(3), uint8(field.FTInt32), []byte{})

	f.Fuzz(func(t *testing.T, fieldNum uint16, fieldTypeRaw uint8, data []byte) {
		ctx := t.Context()

		// Only fuzz valid field numbers
		if int(fieldNum) >= len(m.Fields) {
			return
		}

		// Create a struct to apply the set operation
		s := segment.New(ctx, m)
		fd := m.Fields[fieldNum]

		// Should not panic
		_ = applySet(ctx, s, fd, fieldNum, data)
	})
}

// FuzzApplyClear fuzzes the applyClear function.
func FuzzApplyClear(f *testing.F) {
	m := testMappingForFuzz()

	f.Add(uint16(0))  // bool
	f.Add(uint16(3))  // int32
	f.Add(uint16(11)) // string
	f.Add(uint16(12)) // bytes
	f.Add(uint16(100)) // out of bounds

	f.Fuzz(func(t *testing.T, fieldNum uint16) {
		ctx := t.Context()

		s := segment.New(ctx, m)

		var fd *mapping.FieldDescr
		if int(fieldNum) < len(m.Fields) {
			fd = m.Fields[fieldNum]
		}

		// Should not panic
		_ = applyClear(s, fd, fieldNum)
	})
}

// FuzzApplyListReplaceBools fuzzes the bool list replacement.
func FuzzApplyListReplaceBools(f *testing.F) {
	f.Add([]byte{})
	f.Add([]byte{0})
	f.Add([]byte{1})
	f.Add([]byte{0, 1, 0, 1})
	f.Add([]byte{1, 1, 1, 1, 1, 1, 1, 1}) // 8 bools

	f.Fuzz(func(t *testing.T, data []byte) {
		ctx := t.Context()
		m := testMappingForFuzz()
		s := segment.New(ctx, m)

		// Should not panic
		_ = applyListReplaceBools(s, 13, data)
	})
}

// FuzzApplyListReplaceBytes fuzzes the bytes list replacement.
func FuzzApplyListReplaceBytes(f *testing.F) {
	// Valid format: [count:4][len1:4][data1...][len2:4][data2...]...

	// Empty list
	buf := make([]byte, 4)
	binary.LittleEndian.PutUint32(buf, 0)
	f.Add(buf)

	// Single item
	buf = make([]byte, 12)
	binary.LittleEndian.PutUint32(buf[0:], 1)  // count
	binary.LittleEndian.PutUint32(buf[4:], 4)  // len
	copy(buf[8:], []byte("test"))
	f.Add(buf)

	// Two items
	buf = make([]byte, 21)
	binary.LittleEndian.PutUint32(buf[0:], 2)    // count
	binary.LittleEndian.PutUint32(buf[4:], 5)    // len1
	copy(buf[8:], []byte("hello"))
	binary.LittleEndian.PutUint32(buf[13:], 4)   // len2
	copy(buf[17:], []byte("test"))
	f.Add(buf)

	// Edge cases
	f.Add([]byte{})                 // empty
	f.Add([]byte{0x01, 0x02, 0x03}) // too short for count
	f.Add([]byte{0x01, 0x00, 0x00, 0x00}) // count=1 but no item data

	// Truncated item
	buf = make([]byte, 8)
	binary.LittleEndian.PutUint32(buf[0:], 1)    // count=1
	binary.LittleEndian.PutUint32(buf[4:], 100)  // len=100 (but no data)
	f.Add(buf)

	f.Fuzz(func(t *testing.T, data []byte) {
		ctx := t.Context()
		m := testMappingForFuzz()
		s := segment.New(ctx, m)

		// Should not panic - may return error for malformed data
		_ = applyListReplaceBytes(s, 15, data, field.FTListStrings)
	})
}

// FuzzApplyListSet fuzzes the list set operation.
func FuzzApplyListSet(f *testing.F) {
	f.Add(uint16(13), int32(0), []byte{1})     // bool list, index 0
	f.Add(uint16(14), int32(0), []byte{0, 0, 0, 0}) // int32 list, index 0
	f.Add(uint16(13), int32(-1), []byte{1})    // negative index
	f.Add(uint16(13), int32(100), []byte{1})   // out of bounds index
	f.Add(uint16(14), int32(0), []byte{})      // empty data

	f.Fuzz(func(t *testing.T, fieldNum uint16, index int32, data []byte) {
		ctx := t.Context()
		m := testMappingForFuzz()
		s := segment.New(ctx, m)

		var fd *mapping.FieldDescr
		if int(fieldNum) < len(m.Fields) {
			fd = m.Fields[fieldNum]
		}

		// Should not panic - may return error for invalid operations
		_ = applyListSet(ctx, s, fd, fieldNum, index, data)
	})
}

// FuzzApplyListInsert fuzzes the list insert operation.
func FuzzApplyListInsert(f *testing.F) {
	f.Add(uint16(13), int32(0), []byte{1})     // bool list, index 0
	f.Add(uint16(14), int32(0), []byte{0, 0, 0, 0}) // int32 list
	f.Add(uint16(13), int32(-1), []byte{1})    // negative index
	f.Add(uint16(13), int32(0), []byte{})      // empty data

	f.Fuzz(func(t *testing.T, fieldNum uint16, index int32, data []byte) {
		ctx := t.Context()
		m := testMappingForFuzz()
		s := segment.New(ctx, m)

		var fd *mapping.FieldDescr
		if int(fieldNum) < len(m.Fields) {
			fd = m.Fields[fieldNum]
		}

		// Should not panic
		_ = applyListInsert(ctx, s, fd, fieldNum, index, data)
	})
}

// FuzzApplyListRemove fuzzes the list remove operation.
func FuzzApplyListRemove(f *testing.F) {
	f.Add(uint16(13), int32(0))   // bool list, index 0
	f.Add(uint16(14), int32(0))   // int32 list
	f.Add(uint16(13), int32(-1))  // negative index
	f.Add(uint16(13), int32(100)) // out of bounds

	f.Fuzz(func(t *testing.T, fieldNum uint16, index int32) {
		ctx := t.Context()
		m := testMappingForFuzz()
		s := segment.New(ctx, m)

		var fd *mapping.FieldDescr
		if int(fieldNum) < len(m.Fields) {
			fd = m.Fields[fieldNum]
		}

		// Should not panic
		_ = applyListRemove(ctx, s, fd, fieldNum, index)
	})
}

// FuzzPatchUnmarshal fuzzes the patch unmarshaling with random bytes.
func FuzzPatchUnmarshal(f *testing.F) {
	ctx := f.Context()

	// Create a valid patch
	p := msgs.NewPatch(ctx)
	p.SetVersion(PatchVersion)
	data, _ := p.Marshal()
	f.Add(data)

	// Empty data
	f.Add([]byte{})

	// Random bytes
	f.Add([]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})
	f.Add([]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff})

	f.Fuzz(func(t *testing.T, data []byte) {
		p := msgs.NewPatch(t.Context())

		// Should not panic on any input
		_ = p.Unmarshal(data)
	})
}

// FuzzOpUnmarshal fuzzes the operation unmarshaling.
func FuzzOpUnmarshal(f *testing.F) {
	ctx := f.Context()

	// Create valid ops
	op := msgs.NewOp(ctx)
	op.SetFieldNum(1)
	op.SetType(msgs.Set)
	op.SetData([]byte{0x42})
	data, _ := op.Marshal()
	f.Add(data)

	// List operation
	op = msgs.NewOp(ctx)
	op.SetFieldNum(10)
	op.SetType(msgs.ListSet)
	op.SetIndex(5)
	op.SetData([]byte{0x01, 0x02, 0x03, 0x04})
	data, _ = op.Marshal()
	f.Add(data)

	// Empty
	f.Add([]byte{})

	f.Fuzz(func(t *testing.T, data []byte) {
		op := msgs.NewOp(t.Context())

		// Should not panic
		_ = op.Unmarshal(data)
	})
}

// FuzzApplyOpWithDepth fuzzes the operation application with depth tracking.
func FuzzApplyOpWithDepth(f *testing.F) {
	f.Add(uint16(0), uint8(msgs.Set), int32(-1), []byte{1}, 0)
	f.Add(uint16(3), uint8(msgs.Set), int32(-1), []byte{0, 0, 0, 0}, 0)
	f.Add(uint16(0), uint8(msgs.Clear), int32(-1), []byte{}, 0)
	f.Add(uint16(13), uint8(msgs.ListReplace), int32(-1), []byte{0, 1, 0, 1}, 0)
	f.Add(uint16(13), uint8(msgs.ListSet), int32(0), []byte{1}, 0)
	f.Add(uint16(13), uint8(msgs.ListInsert), int32(0), []byte{1}, 0)
	f.Add(uint16(13), uint8(msgs.ListRemove), int32(0), []byte{}, 0)

	// Deep nesting
	f.Add(uint16(0), uint8(msgs.StructPatch), int32(-1), []byte{}, 99)
	f.Add(uint16(0), uint8(msgs.StructPatch), int32(-1), []byte{}, 100) // at limit

	// Unknown operation type
	f.Add(uint16(0), uint8(255), int32(-1), []byte{}, 0)

	f.Fuzz(func(t *testing.T, fieldNum uint16, opTypeRaw uint8, index int32, data []byte, depth int) {
		ctx := t.Context()
		m := testMappingForFuzz()
		s := segment.New(ctx, m)

		op := msgs.NewOp(ctx)
		op.SetFieldNum(fieldNum)
		op.SetType(msgs.OpType(opTypeRaw))
		op.SetIndex(index)
		op.SetData(data)

		// Should not panic
		_ = applyOpWithDepth(ctx, s, m, op, depth)
	})
}

// FuzzEncodingRoundTrip tests encoding functions roundtrip.
func FuzzEncodingRoundTrip(f *testing.F) {
	f.Add(int16(0))
	f.Add(int16(1))
	f.Add(int16(-1))
	f.Add(int16(32767))
	f.Add(int16(-32768))

	f.Fuzz(func(t *testing.T, value int16) {
		buf := make([]byte, 2)
		binary.LittleEndian.PutUint16(buf, uint16(value))
		decoded := decodeInt16(buf)

		if decoded != value {
			t.Errorf("FuzzEncodingRoundTrip: int16 roundtrip failed: got %d, want %d", decoded, value)
		}
	})
}

// FuzzDecodeInt32 fuzzes the int32 decoding.
func FuzzDecodeInt32(f *testing.F) {
	f.Add([]byte{0, 0, 0, 0})
	f.Add([]byte{0xff, 0xff, 0xff, 0x7f}) // max int32
	f.Add([]byte{0x00, 0x00, 0x00, 0x80}) // min int32
	f.Add([]byte{0x01, 0x02, 0x03, 0x04})

	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) < 4 {
			return
		}

		// Should not panic
		decoded := decodeInt32(data)

		// Verify roundtrip
		buf := make([]byte, 4)
		binary.LittleEndian.PutUint32(buf, uint32(decoded))
		if decodeInt32(buf) != decoded {
			t.Errorf("FuzzDecodeInt32: roundtrip failed")
		}
	})
}

// FuzzDecodeInt64 fuzzes the int64 decoding.
func FuzzDecodeInt64(f *testing.F) {
	f.Add([]byte{0, 0, 0, 0, 0, 0, 0, 0})
	f.Add([]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x7f}) // max int64
	f.Add([]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x80}) // min int64

	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) < 8 {
			return
		}

		// Should not panic
		decoded := decodeInt64(data)

		// Verify roundtrip
		buf := make([]byte, 8)
		binary.LittleEndian.PutUint64(buf, uint64(decoded))
		if decodeInt64(buf) != decoded {
			t.Errorf("FuzzDecodeInt64: roundtrip failed")
		}
	})
}

// FuzzDecodeFloat32 fuzzes the float32 decoding.
func FuzzDecodeFloat32(f *testing.F) {
	f.Add([]byte{0, 0, 0, 0})          // 0.0
	f.Add([]byte{0x00, 0x00, 0x80, 0x3f}) // 1.0
	f.Add([]byte{0x00, 0x00, 0x80, 0xbf}) // -1.0

	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) < 4 {
			return
		}

		// Should not panic
		_ = decodeFloat32(data)
	})
}

// FuzzDecodeFloat64 fuzzes the float64 decoding.
func FuzzDecodeFloat64(f *testing.F) {
	f.Add([]byte{0, 0, 0, 0, 0, 0, 0, 0})                         // 0.0
	f.Add([]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xf0, 0x3f}) // 1.0
	f.Add([]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xf0, 0xbf}) // -1.0

	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) < 8 {
			return
		}

		// Should not panic
		_ = decodeFloat64(data)
	})
}

// ============================================================================
// Encoding roundtrip fuzz tests
// ============================================================================

// FuzzEncodeDecodeInt16 fuzzes the int16 encode/decode roundtrip.
func FuzzEncodeDecodeInt16(f *testing.F) {
	f.Add(int16(0))
	f.Add(int16(1))
	f.Add(int16(-1))
	f.Add(int16(32767))
	f.Add(int16(-32768))

	f.Fuzz(func(t *testing.T, val int16) {
		encoded := encodeInt16(val)
		if len(encoded) != 2 {
			t.Errorf("FuzzEncodeDecodeInt16: expected 2 bytes, got %d", len(encoded))
			return
		}
		decoded := decodeInt16(encoded)
		if decoded != val {
			t.Errorf("FuzzEncodeDecodeInt16: roundtrip failed: got %d, want %d", decoded, val)
		}
	})
}

// FuzzEncodeDecodeUint16 fuzzes the uint16 encode/decode roundtrip.
func FuzzEncodeDecodeUint16(f *testing.F) {
	f.Add(uint16(0))
	f.Add(uint16(1))
	f.Add(uint16(32767))
	f.Add(uint16(65535))

	f.Fuzz(func(t *testing.T, val uint16) {
		encoded := encodeUint16(val)
		if len(encoded) != 2 {
			t.Errorf("FuzzEncodeDecodeUint16: expected 2 bytes, got %d", len(encoded))
			return
		}
		decoded := decodeUint16(encoded)
		if decoded != val {
			t.Errorf("FuzzEncodeDecodeUint16: roundtrip failed: got %d, want %d", decoded, val)
		}
	})
}

// FuzzEncodeDecodeInt32 fuzzes the int32 encode/decode roundtrip.
func FuzzEncodeDecodeInt32(f *testing.F) {
	f.Add(int32(0))
	f.Add(int32(1))
	f.Add(int32(-1))
	f.Add(int32(2147483647))
	f.Add(int32(-2147483648))

	f.Fuzz(func(t *testing.T, val int32) {
		encoded := encodeInt32(val)
		if len(encoded) != 4 {
			t.Errorf("FuzzEncodeDecodeInt32: expected 4 bytes, got %d", len(encoded))
			return
		}
		decoded := decodeInt32(encoded)
		if decoded != val {
			t.Errorf("FuzzEncodeDecodeInt32: roundtrip failed: got %d, want %d", decoded, val)
		}
	})
}

// FuzzEncodeDecodeUint32 fuzzes the uint32 encode/decode roundtrip.
func FuzzEncodeDecodeUint32(f *testing.F) {
	f.Add(uint32(0))
	f.Add(uint32(1))
	f.Add(uint32(2147483647))
	f.Add(uint32(4294967295))

	f.Fuzz(func(t *testing.T, val uint32) {
		encoded := encodeUint32(val)
		if len(encoded) != 4 {
			t.Errorf("FuzzEncodeDecodeUint32: expected 4 bytes, got %d", len(encoded))
			return
		}
		decoded := decodeUint32(encoded)
		if decoded != val {
			t.Errorf("FuzzEncodeDecodeUint32: roundtrip failed: got %d, want %d", decoded, val)
		}
	})
}

// FuzzEncodeDecodeInt64 fuzzes the int64 encode/decode roundtrip.
func FuzzEncodeDecodeInt64(f *testing.F) {
	f.Add(int64(0))
	f.Add(int64(1))
	f.Add(int64(-1))
	f.Add(int64(9223372036854775807))
	f.Add(int64(-9223372036854775808))

	f.Fuzz(func(t *testing.T, val int64) {
		encoded := encodeInt64(val)
		if len(encoded) != 8 {
			t.Errorf("FuzzEncodeDecodeInt64: expected 8 bytes, got %d", len(encoded))
			return
		}
		decoded := decodeInt64(encoded)
		if decoded != val {
			t.Errorf("FuzzEncodeDecodeInt64: roundtrip failed: got %d, want %d", decoded, val)
		}
	})
}

// FuzzEncodeDecodeUint64 fuzzes the uint64 encode/decode roundtrip.
func FuzzEncodeDecodeUint64(f *testing.F) {
	f.Add(uint64(0))
	f.Add(uint64(1))
	f.Add(uint64(9223372036854775807))
	f.Add(uint64(18446744073709551615))

	f.Fuzz(func(t *testing.T, val uint64) {
		encoded := encodeUint64(val)
		if len(encoded) != 8 {
			t.Errorf("FuzzEncodeDecodeUint64: expected 8 bytes, got %d", len(encoded))
			return
		}
		decoded := decodeUint64(encoded)
		if decoded != val {
			t.Errorf("FuzzEncodeDecodeUint64: roundtrip failed: got %d, want %d", decoded, val)
		}
	})
}

// FuzzEncodeDecodeFloat32 fuzzes the float32 encode/decode roundtrip.
func FuzzEncodeDecodeFloat32(f *testing.F) {
	f.Add(float32(0.0))
	f.Add(float32(1.0))
	f.Add(float32(-1.0))
	f.Add(float32(3.14159))
	f.Add(float32(1e38))
	f.Add(float32(-1e38))

	f.Fuzz(func(t *testing.T, val float32) {
		encoded := encodeFloat32(val)
		if len(encoded) != 4 {
			t.Errorf("FuzzEncodeDecodeFloat32: expected 4 bytes, got %d", len(encoded))
			return
		}
		decoded := decodeFloat32(encoded)

		// Special handling for NaN
		if val != val { // NaN check
			if decoded == decoded { // should also be NaN
				t.Errorf("FuzzEncodeDecodeFloat32: NaN roundtrip failed")
			}
		} else if decoded != val {
			t.Errorf("FuzzEncodeDecodeFloat32: roundtrip failed: got %v, want %v", decoded, val)
		}
	})
}

// FuzzEncodeDecodeFloat64 fuzzes the float64 encode/decode roundtrip.
func FuzzEncodeDecodeFloat64(f *testing.F) {
	f.Add(float64(0.0))
	f.Add(float64(1.0))
	f.Add(float64(-1.0))
	f.Add(float64(3.14159265358979))
	f.Add(float64(1e308))
	f.Add(float64(-1e308))

	f.Fuzz(func(t *testing.T, val float64) {
		encoded := encodeFloat64(val)
		if len(encoded) != 8 {
			t.Errorf("FuzzEncodeDecodeFloat64: expected 8 bytes, got %d", len(encoded))
			return
		}
		decoded := decodeFloat64(encoded)

		// Special handling for NaN
		if val != val { // NaN check
			if decoded == decoded { // should also be NaN
				t.Errorf("FuzzEncodeDecodeFloat64: NaN roundtrip failed")
			}
		} else if decoded != val {
			t.Errorf("FuzzEncodeDecodeFloat64: roundtrip failed: got %v, want %v", decoded, val)
		}
	})
}

// FuzzApplyListReplaceNumbers fuzzes number list replacement.
func FuzzApplyListReplaceNumbers(f *testing.F) {
	// int32 list replacement data (4 bytes per item)
	f.Add([]byte{})
	f.Add([]byte{0, 0, 0, 0})
	f.Add([]byte{1, 0, 0, 0})
	f.Add([]byte{0xff, 0xff, 0xff, 0x7f}) // max int32
	f.Add([]byte{0x00, 0x00, 0x00, 0x80}) // min int32
	f.Add([]byte{1, 0, 0, 0, 2, 0, 0, 0, 3, 0, 0, 0}) // [1, 2, 3]

	f.Fuzz(func(t *testing.T, data []byte) {
		ctx := t.Context()
		m := testMappingForFuzz()
		s := segment.New(ctx, m)

		// Should not panic
		_ = applyListReplaceNumbers(s, 14, data, field.FTListInt32)
	})
}

// FuzzApplyListStructPatch fuzzes the list struct patch operation.
func FuzzApplyListStructPatch(f *testing.F) {
	ctx := f.Context()

	// Create a valid patch
	p := msgs.NewPatch(ctx)
	p.SetVersion(PatchVersion)
	data, _ := p.Marshal()
	f.Add(int32(0), data)

	// Empty patch data
	f.Add(int32(0), []byte{})

	// Random bytes
	f.Add(int32(0), []byte{0xff, 0xff, 0xff, 0xff})

	// Negative index
	f.Add(int32(-1), data)

	f.Fuzz(func(t *testing.T, index int32, patchData []byte) {
		ctx := t.Context()

		// Create a mapping with a nested struct list
		nestedMapping := &mapping.Map{
			Name: "Nested",
			Fields: []*mapping.FieldDescr{
				{Name: "value", Type: field.FTInt32, FieldNum: 0},
			},
		}
		m := &mapping.Map{
			Name: "Parent",
			Fields: []*mapping.FieldDescr{
				{Name: "structs", Type: field.FTListStructs, FieldNum: 0, Mapping: nestedMapping},
			},
		}

		s := segment.New(ctx, m)

		// Should not panic
		_ = applyListStructPatch(ctx, s, m.Fields[0], uint16(0), index, patchData, 0)
	})
}

// FuzzApplyStructPatch fuzzes the struct patch operation.
func FuzzApplyStructPatch(f *testing.F) {
	ctx := f.Context()

	// Create a valid patch
	p := msgs.NewPatch(ctx)
	p.SetVersion(PatchVersion)
	data, _ := p.Marshal()
	f.Add(data)

	// Empty data
	f.Add([]byte{})

	// Random bytes
	f.Add([]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})
	f.Add([]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff})

	f.Fuzz(func(t *testing.T, patchData []byte) {
		ctx := t.Context()

		// Create a mapping with a nested struct
		nestedMapping := &mapping.Map{
			Name: "Nested",
			Fields: []*mapping.FieldDescr{
				{Name: "value", Type: field.FTInt32, FieldNum: 0},
			},
		}
		m := &mapping.Map{
			Name: "Parent",
			Fields: []*mapping.FieldDescr{
				{Name: "nested", Type: field.FTStruct, FieldNum: 0, Mapping: nestedMapping},
			},
		}

		s := segment.New(ctx, m)

		// Should not panic - may return error for invalid data
		_ = applyStructPatch(ctx, s, m.Fields[0], 0, patchData, 0)
	})
}
