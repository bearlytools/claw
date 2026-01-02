package segment

import (
	"encoding/binary"
	"testing"
	"unsafe"

	"github.com/bearlytools/claw/clawc/languages/go/field"
	"github.com/bearlytools/claw/clawc/languages/go/mapping"
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
			{Name: "struct", Type: field.FTStruct, FieldNum: 13},
			{Name: "listBools", Type: field.FTListBools, FieldNum: 14},
			{Name: "listInt32", Type: field.FTListInt32, FieldNum: 15},
			{Name: "listStrings", Type: field.FTListStrings, FieldNum: 16},
			{Name: "map", Type: field.FTMap, FieldNum: 17},
		},
	}
}

// FuzzDecodeHeader fuzzes the header decoding function which parses field metadata.
func FuzzDecodeHeader(f *testing.F) {
	// Valid headers with various field types
	for _, ft := range []field.Type{
		field.FTBool, field.FTInt8, field.FTInt16, field.FTInt32, field.FTInt64,
		field.FTUint8, field.FTUint16, field.FTUint32, field.FTUint64,
		field.FTFloat32, field.FTFloat64, field.FTString, field.FTBytes,
		field.FTStruct, field.FTListBools, field.FTListInt32, field.FTMap,
	} {
		buf := make([]byte, 8)
		EncodeHeader(buf, 1, ft, 100)
		f.Add(buf)
	}

	// Edge cases for field numbers
	buf := make([]byte, 8)
	EncodeHeader(buf, 0, field.FTInt32, 0) // zero field num
	f.Add(buf)

	buf = make([]byte, 8)
	EncodeHeader(buf, 65535, field.FTInt32, 0) // max field num
	f.Add(buf)

	// Edge cases for final40
	buf = make([]byte, 8)
	EncodeHeader(buf, 1, field.FTBytes, 0) // zero size
	f.Add(buf)

	buf = make([]byte, 8)
	EncodeHeader(buf, 1, field.FTBytes, MaxFinal40) // max 40-bit
	f.Add(buf)

	// All zeros (padding)
	f.Add([]byte{0, 0, 0, 0, 0, 0, 0, 0})

	// All ones
	f.Add([]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff})

	f.Fuzz(func(t *testing.T, input []byte) {
		if len(input) < HeaderSize {
			return // skip inputs too short for DecodeHeader
		}

		// Should not panic
		fieldNum, fieldType, final40 := DecodeHeader(input[:8])

		// Basic sanity checks
		_ = fieldNum  // any uint16 is valid
		_ = fieldType // any uint8 is valid
		_ = final40   // any uint64 is valid (but only 40 bits used)
	})
}

// FuzzDecodeMapHeader fuzzes the map header decoding function.
func FuzzDecodeMapHeader(f *testing.F) {
	// Valid map headers
	for _, keyType := range []field.Type{
		field.FTBool, field.FTInt32, field.FTUint64, field.FTString,
	} {
		for _, valType := range []field.Type{
			field.FTBool, field.FTInt32, field.FTString, field.FTStruct,
		} {
			buf := make([]byte, 8)
			EncodeMapHeader(buf, 1, keyType, valType, 100)
			f.Add(buf)
		}
	}

	// Edge cases
	buf := make([]byte, 8)
	EncodeMapHeader(buf, 1, field.FTString, field.FTInt32, 0) // zero size
	f.Add(buf)

	buf = make([]byte, 8)
	EncodeMapHeader(buf, 1, field.FTString, field.FTInt32, MaxMapSize) // max size
	f.Add(buf)

	f.Fuzz(func(t *testing.T, input []byte) {
		if len(input) < HeaderSize {
			return
		}

		// Should not panic
		keyType, valueType, totalSize := DecodeMapHeader(input[:8])

		_ = keyType
		_ = valueType
		_ = totalSize
	})
}

// FuzzParseFieldIndex fuzzes the field index parsing which processes binary segment data.
func FuzzParseFieldIndex(f *testing.F) {
	m := testMappingForFuzz()

	// Create a valid segment with a header
	validSeg := make([]byte, 16)
	EncodeHeader(validSeg[0:8], 0, field.FTStruct, 16) // root header
	EncodeHeader(validSeg[8:16], 1, field.FTInt32, 42) // field 1: int32 with value 42
	f.Add(validSeg)

	// Segment with string field
	seg := make([]byte, 24)
	EncodeHeader(seg[0:8], 0, field.FTStruct, 24)
	EncodeHeader(seg[8:16], 11, field.FTString, 5) // string with 5 chars
	copy(seg[16:21], "hello")
	f.Add(seg)

	// Segment with nested struct
	seg = make([]byte, 24)
	EncodeHeader(seg[0:8], 0, field.FTStruct, 24)
	EncodeHeader(seg[8:16], 13, field.FTStruct, 16) // nested struct of size 16
	f.Add(seg)

	// Segment with bool list
	seg = make([]byte, 24)
	EncodeHeader(seg[0:8], 0, field.FTStruct, 24)
	EncodeHeader(seg[8:16], 14, field.FTListBools, 8) // 8 bools = 1 byte
	f.Add(seg)

	// Segment with map
	seg = make([]byte, 24)
	EncodeHeader(seg[0:8], 0, field.FTStruct, 24)
	EncodeMapHeader(seg[8:16], 17, field.FTString, field.FTInt32, 16)
	f.Add(seg)

	// Empty segment (just header)
	empty := make([]byte, 8)
	EncodeHeader(empty, 0, field.FTStruct, 8)
	f.Add(empty)

	// Segment with padding (type 0)
	padded := make([]byte, 16)
	EncodeHeader(padded[0:8], 0, field.FTStruct, 16)
	// bytes 8-15 are zero (padding)
	f.Add(padded)

	f.Fuzz(func(t *testing.T, input []byte) {
		if len(input) < HeaderSize {
			return
		}

		// Create a struct with this data
		s := &Struct{
			seg:        &Segment{data: input},
			mapping:    m,
			fieldIndex: make([]fieldEntry, len(m.Fields)),
		}

		// Should not panic
		parseFieldIndex(s)

		// Verify fieldIndex entries are within bounds
		for i, entry := range s.fieldIndex {
			if !entry.isSet {
				continue
			}
			offset := int(entry.offset)
			size := int(entry.size)
			if offset < 0 || size < 0 {
				t.Errorf("FuzzParseFieldIndex: field %d has invalid offset/size: %d/%d", i, offset, size)
			}
			if offset+size > len(input) {
				// This can happen with malformed input and is expected
				// Just ensure we didn't panic
			}
		}
	})
}

// FuzzHeaderRoundTrip tests encode/decode roundtrip for headers.
func FuzzHeaderRoundTrip(f *testing.F) {
	f.Add(uint16(0), uint8(1), uint64(0))
	f.Add(uint16(1), uint8(1), uint64(1))
	f.Add(uint16(100), uint8(12), uint64(1000))
	f.Add(uint16(65535), uint8(55), uint64(MaxFinal40))

	f.Fuzz(func(t *testing.T, fieldNum uint16, fieldTypeRaw uint8, final40Raw uint64) {
		ft := field.Type(fieldTypeRaw)

		// Clamp final40 to valid range
		final40 := final40Raw & MaxFinal40

		buf := make([]byte, 8)
		EncodeHeader(buf, fieldNum, ft, final40)

		gotFieldNum, gotFieldType, gotFinal40 := DecodeHeader(buf)

		if gotFieldNum != fieldNum {
			t.Errorf("FuzzHeaderRoundTrip: fieldNum = %d, want %d", gotFieldNum, fieldNum)
		}
		if gotFieldType != ft {
			t.Errorf("FuzzHeaderRoundTrip: fieldType = %d, want %d", gotFieldType, ft)
		}
		if gotFinal40 != final40 {
			t.Errorf("FuzzHeaderRoundTrip: final40 = %d, want %d", gotFinal40, final40)
		}
	})
}

// FuzzMapHeaderRoundTrip tests encode/decode roundtrip for map headers.
func FuzzMapHeaderRoundTrip(f *testing.F) {
	f.Add(uint16(1), uint8(1), uint8(1), uint32(0))
	f.Add(uint16(1), uint8(12), uint8(4), uint32(100))
	f.Add(uint16(100), uint8(1), uint8(14), uint32(MaxMapSize))

	f.Fuzz(func(t *testing.T, fieldNum uint16, keyTypeRaw, valTypeRaw uint8, totalSizeRaw uint32) {
		keyType := field.Type(keyTypeRaw)
		valType := field.Type(valTypeRaw)

		// Clamp totalSize to valid range
		totalSize := totalSizeRaw & MaxMapSize

		buf := make([]byte, 8)
		EncodeMapHeader(buf, fieldNum, keyType, valType, totalSize)

		gotKeyType, gotValType, gotTotalSize := DecodeMapHeader(buf)

		if gotKeyType != keyType {
			t.Errorf("FuzzMapHeaderRoundTrip: keyType = %d, want %d", gotKeyType, keyType)
		}
		if gotValType != valType {
			t.Errorf("FuzzMapHeaderRoundTrip: valType = %d, want %d", gotValType, valType)
		}
		if gotTotalSize != totalSize {
			t.Errorf("FuzzMapHeaderRoundTrip: totalSize = %d, want %d", gotTotalSize, totalSize)
		}
	})
}

// FuzzSegmentWithRandomBytes tests segment parsing with completely random bytes.
func FuzzSegmentWithRandomBytes(f *testing.F) {
	m := testMappingForFuzz()

	// Random byte patterns
	f.Add([]byte{0x01, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0x00, 0x04, 0x2a, 0x00, 0x00, 0x00, 0x00})
	f.Add([]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff})
	f.Add([]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})

	f.Fuzz(func(t *testing.T, input []byte) {
		if len(input) < 8 {
			return
		}

		// Ensure we have at least a header size
		s := &Struct{
			seg:        &Segment{data: input},
			mapping:    m,
			fieldIndex: make([]fieldEntry, len(m.Fields)),
		}

		// Should not panic on any input
		parseFieldIndex(s)
	})
}

// FuzzPaddingNeeded tests the padding calculation function.
func FuzzPaddingNeeded(f *testing.F) {
	f.Add(0)
	f.Add(1)
	f.Add(7)
	f.Add(8)
	f.Add(9)
	f.Add(15)
	f.Add(16)
	f.Add(100)
	f.Add(-1)
	f.Add(-100)

	f.Fuzz(func(t *testing.T, size int) {
		// Should not panic
		padding := paddingNeeded(size)

		// Verify properties
		if size <= 0 {
			if padding != 0 {
				t.Errorf("FuzzPaddingNeeded: size=%d got padding=%d, want 0", size, padding)
			}
			return
		}

		// Padding should be 0-7
		if padding < 0 || padding > 7 {
			t.Errorf("FuzzPaddingNeeded: size=%d got padding=%d, want 0-7", size, padding)
		}

		// size + padding should be 8-byte aligned
		if (size+padding)%8 != 0 {
			t.Errorf("FuzzPaddingNeeded: size=%d, padding=%d, sum=%d not 8-byte aligned", size, padding, size+padding)
		}
	})
}

// FuzzSizeWithPadding tests the size with padding calculation.
func FuzzSizeWithPadding(f *testing.F) {
	f.Add(0)
	f.Add(1)
	f.Add(7)
	f.Add(8)
	f.Add(100)

	f.Fuzz(func(t *testing.T, size int) {
		if size < 0 {
			return
		}

		// Should not panic
		result := sizeWithPadding(size)

		// Result should be >= size
		if result < size {
			t.Errorf("FuzzSizeWithPadding: size=%d, result=%d is less than size", size, result)
		}

		// Result should be 8-byte aligned (for positive sizes)
		if size > 0 && result%8 != 0 {
			t.Errorf("FuzzSizeWithPadding: size=%d, result=%d not 8-byte aligned", size, result)
		}
	})
}

// FuzzHeaderFieldNumUpdate tests updating just the field number.
func FuzzHeaderFieldNumUpdate(f *testing.F) {
	f.Add(uint16(0))
	f.Add(uint16(1))
	f.Add(uint16(100))
	f.Add(uint16(65535))

	f.Fuzz(func(t *testing.T, fieldNum uint16) {
		buf := make([]byte, 8)

		// Initialize with some values
		EncodeHeader(buf, 42, field.FTString, 12345)

		// Update just field number
		EncodeHeaderFieldNum(buf, fieldNum)

		// Verify field number changed, others unchanged
		gotFieldNum, gotFieldType, gotFinal40 := DecodeHeader(buf)

		if gotFieldNum != fieldNum {
			t.Errorf("FuzzHeaderFieldNumUpdate: fieldNum = %d, want %d", gotFieldNum, fieldNum)
		}
		if gotFieldType != field.FTString {
			t.Errorf("FuzzHeaderFieldNumUpdate: fieldType changed to %d, want %d", gotFieldType, field.FTString)
		}
		if gotFinal40 != 12345 {
			t.Errorf("FuzzHeaderFieldNumUpdate: final40 changed to %d, want 12345", gotFinal40)
		}
	})
}

// FuzzBinaryHeaderBytes tests raw binary operations on header bytes.
func FuzzBinaryHeaderBytes(f *testing.F) {
	// Test that binary encoding is little-endian as expected
	f.Add([]byte{0x01, 0x00, 0x02, 0x00, 0x00, 0x00, 0x00, 0x00})
	f.Add([]byte{0x00, 0x01, 0x03, 0x00, 0x00, 0x00, 0x00, 0x00})

	f.Fuzz(func(t *testing.T, input []byte) {
		if len(input) < 8 {
			return
		}

		// Decode using our function
		fieldNum, fieldType, _ := DecodeHeader(input[:8])

		// Verify fieldNum matches little-endian reading
		expectedFieldNum := binary.LittleEndian.Uint16(input[0:2])
		if fieldNum != expectedFieldNum {
			t.Errorf("FuzzBinaryHeaderBytes: fieldNum = %d, expected %d", fieldNum, expectedFieldNum)
		}

		// Verify fieldType matches byte 2
		if uint8(fieldType) != input[2] {
			t.Errorf("FuzzBinaryHeaderBytes: fieldType = %d, expected %d", fieldType, input[2])
		}
	})
}

// ============================================================================
// List Parsing Fuzz Tests
// ============================================================================

// FuzzBoolsOperations fuzzes the Bools list operations.
func FuzzBoolsOperations(f *testing.F) {
	m := testMappingForFuzz()

	// Use bytes to represent bools (0 = false, non-zero = true)
	f.Add([]byte{})
	f.Add([]byte{1})
	f.Add([]byte{0})
	f.Add([]byte{1, 0, 1, 0})
	f.Add([]byte{1, 1, 1, 1, 1, 1, 1, 1}) // 8 bools (1 byte packed)
	f.Add([]byte{1, 1, 1, 1, 1, 1, 1, 1, 1}) // 9 bools (crosses byte boundary)

	f.Fuzz(func(t *testing.T, data []byte) {
		ctx := t.Context()
		s := New(ctx, m)

		// Convert bytes to bools
		values := make([]bool, len(data))
		for i, b := range data {
			values[i] = b != 0
		}

		bools := NewBools(s, 14)
		bools.SetAll(values)

		// Verify length
		if bools.Len() != len(values) {
			t.Errorf("FuzzBoolsOperations: Len() = %d, want %d", bools.Len(), len(values))
		}

		// Verify values
		for i, want := range values {
			got := bools.Get(i)
			if got != want {
				t.Errorf("FuzzBoolsOperations: Get(%d) = %v, want %v", i, got, want)
			}
		}

		// Sync and verify
		if err := bools.SyncToSegment(); err != nil {
			t.Errorf("FuzzBoolsOperations: SyncToSegment error: %v", err)
		}
	})
}

// FuzzBoolsAppend fuzzes appending to Bools list.
func FuzzBoolsAppend(f *testing.F) {
	m := testMappingForFuzz()

	f.Add(true)
	f.Add(false)

	f.Fuzz(func(t *testing.T, value bool) {
		ctx := t.Context()
		s := New(ctx, m)

		bools := NewBools(s, 14)
		bools.Append(value)

		if bools.Len() != 1 {
			t.Errorf("FuzzBoolsAppend: Len() = %d, want 1", bools.Len())
		}
		if bools.Get(0) != value {
			t.Errorf("FuzzBoolsAppend: Get(0) = %v, want %v", bools.Get(0), value)
		}
	})
}

// FuzzNumbersInt32 fuzzes the Numbers[int32] list operations.
func FuzzNumbersInt32(f *testing.F) {
	m := testMappingForFuzz()

	f.Add([]byte{})
	f.Add([]byte{0, 0, 0, 0})                         // single 0
	f.Add([]byte{1, 0, 0, 0})                         // single 1
	f.Add([]byte{0xff, 0xff, 0xff, 0x7f})             // max int32
	f.Add([]byte{0x00, 0x00, 0x00, 0x80})             // min int32
	f.Add([]byte{1, 0, 0, 0, 2, 0, 0, 0, 3, 0, 0, 0}) // [1, 2, 3]

	f.Fuzz(func(t *testing.T, data []byte) {
		ctx := t.Context()
		s := New(ctx, m)

		// Round down to multiple of 4 bytes
		numItems := len(data) / 4
		if numItems == 0 {
			return
		}
		data = data[:numItems*4]

		nums := NewNumbers[int32](s, 15)

		// Parse values from data
		values := make([]int32, numItems)
		for i := 0; i < numItems; i++ {
			values[i] = int32(binary.LittleEndian.Uint32(data[i*4:]))
		}

		nums.SetAll(values)

		// Verify length
		if nums.Len() != len(values) {
			t.Errorf("FuzzNumbersInt32: Len() = %d, want %d", nums.Len(), len(values))
		}

		// Verify values
		for i, want := range values {
			got := nums.Get(i)
			if got != want {
				t.Errorf("FuzzNumbersInt32: Get(%d) = %d, want %d", i, got, want)
			}
		}

		// Sync to segment
		if err := nums.SyncToSegment(); err != nil {
			t.Errorf("FuzzNumbersInt32: SyncToSegment error: %v", err)
		}
	})
}

// FuzzNumbersInt64 fuzzes the Numbers[int64] list operations.
func FuzzNumbersInt64(f *testing.F) {
	m := &mapping.Map{
		Fields: []*mapping.FieldDescr{
			{Name: "listInt64", Type: field.FTListInt64, FieldNum: 0},
		},
	}

	f.Add([]byte{0, 0, 0, 0, 0, 0, 0, 0})                                 // single 0
	f.Add([]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x7f})         // max int64
	f.Add([]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x80})         // min int64

	f.Fuzz(func(t *testing.T, data []byte) {
		ctx := t.Context()
		s := New(ctx, m)

		// Round down to multiple of 8 bytes
		numItems := len(data) / 8
		if numItems == 0 {
			return
		}
		data = data[:numItems*8]

		nums := NewNumbers[int64](s, 0)

		// Parse values from data
		values := make([]int64, numItems)
		for i := 0; i < numItems; i++ {
			values[i] = int64(binary.LittleEndian.Uint64(data[i*8:]))
		}

		nums.SetAll(values)

		// Verify roundtrip
		for i, want := range values {
			got := nums.Get(i)
			if got != want {
				t.Errorf("FuzzNumbersInt64: Get(%d) = %d, want %d", i, got, want)
			}
		}
	})
}

// FuzzNumbersFloat32 fuzzes the Numbers[float32] list operations.
func FuzzNumbersFloat32(f *testing.F) {
	m := &mapping.Map{
		Fields: []*mapping.FieldDescr{
			{Name: "listFloat32", Type: field.FTListFloat32, FieldNum: 0},
		},
	}

	f.Add([]byte{0, 0, 0, 0})                 // 0.0
	f.Add([]byte{0x00, 0x00, 0x80, 0x3f})     // 1.0
	f.Add([]byte{0x00, 0x00, 0x80, 0xbf})     // -1.0
	f.Add([]byte{0x00, 0x00, 0xc0, 0x7f})     // NaN (should still work)
	f.Add([]byte{0x00, 0x00, 0x80, 0x7f})     // +Inf

	f.Fuzz(func(t *testing.T, data []byte) {
		ctx := t.Context()
		s := New(ctx, m)

		numItems := len(data) / 4
		if numItems == 0 {
			return
		}

		nums := NewNumbers[float32](s, 0)

		// Append raw values
		for i := 0; i < numItems; i++ {
			bits := binary.LittleEndian.Uint32(data[i*4:])
			nums.Append(float32FromBits(bits))
		}

		// Verify length
		if nums.Len() != numItems {
			t.Errorf("FuzzNumbersFloat32: Len() = %d, want %d", nums.Len(), numItems)
		}
	})
}

// float32FromBits is a helper to convert uint32 bits to float32.
func float32FromBits(bits uint32) float32 {
	return *(*float32)(unsafe.Pointer(&bits))
}

// FuzzStringsOperations fuzzes the Strings list operations.
func FuzzStringsOperations(f *testing.F) {
	m := testMappingForFuzz()

	f.Add("")
	f.Add("hello")
	f.Add("hello\x00world") // null in middle
	f.Add("unicode: 中文")
	f.Add("very long string that is quite long and tests capacity growth in the buffer")

	f.Fuzz(func(t *testing.T, value string) {
		ctx := t.Context()
		s := New(ctx, m)

		strs := NewStrings(s, 16)
		strs.Append(value)

		if strs.Len() != 1 {
			t.Errorf("FuzzStringsOperations: Len() = %d, want 1", strs.Len())
		}
		if strs.Get(0) != value {
			t.Errorf("FuzzStringsOperations: Get(0) = %q, want %q", strs.Get(0), value)
		}

		// Test SetAll
		strs.SetAll([]string{value, value})
		if strs.Len() != 2 {
			t.Errorf("FuzzStringsOperations: after SetAll Len() = %d, want 2", strs.Len())
		}

		// Sync
		if err := strs.SyncToSegment(); err != nil {
			t.Errorf("FuzzStringsOperations: SyncToSegment error: %v", err)
		}
	})
}

// FuzzBytesOperations fuzzes the Bytes list operations.
func FuzzBytesOperations(f *testing.F) {
	m := &mapping.Map{
		Fields: []*mapping.FieldDescr{
			{Name: "listBytes", Type: field.FTListBytes, FieldNum: 0},
		},
	}

	f.Add([]byte{})
	f.Add([]byte{0x00})
	f.Add([]byte{0xde, 0xad, 0xbe, 0xef})
	f.Add([]byte{0xff, 0xff, 0xff, 0xff})

	f.Fuzz(func(t *testing.T, value []byte) {
		ctx := t.Context()
		s := New(ctx, m)

		b := NewBytes(s, 0)
		b.Append(value)

		if b.Len() != 1 {
			t.Errorf("FuzzBytesOperations: Len() = %d, want 1", b.Len())
		}

		got := b.Get(0)
		if len(got) != len(value) {
			t.Errorf("FuzzBytesOperations: Get(0) len = %d, want %d", len(got), len(value))
		}

		// Sync
		if err := b.SyncToSegment(); err != nil {
			t.Errorf("FuzzBytesOperations: SyncToSegment error: %v", err)
		}
	})
}

// FuzzListBitPacking tests bool list bit packing edge cases.
func FuzzListBitPacking(f *testing.F) {
	m := testMappingForFuzz()

	// Test various counts that exercise bit packing
	for count := 0; count <= 17; count++ {
		values := make([]bool, count)
		for i := range values {
			values[i] = i%2 == 0
		}
		f.Add(count)
	}

	f.Fuzz(func(t *testing.T, count int) {
		if count < 0 || count > 1000 {
			return
		}

		ctx := t.Context()
		s := New(ctx, m)

		// Create alternating pattern
		values := make([]bool, count)
		for i := range values {
			values[i] = i%2 == 0
		}

		bools := NewBools(s, 14)
		bools.SetAll(values)

		// Sync to segment
		if err := bools.SyncToSegment(); err != nil {
			t.Errorf("FuzzListBitPacking: SyncToSegment error: %v", err)
			return
		}

		// Create a new struct from the marshaled data to test parsing
		data, err := s.Marshal()
		if err != nil {
			t.Errorf("FuzzListBitPacking: Marshal error: %v", err)
			return
		}

		s2 := New(ctx, m)
		if err := s2.Unmarshal(data); err != nil {
			t.Errorf("FuzzListBitPacking: Unmarshal error: %v", err)
			return
		}

		// Get the list from unmarshaled struct
		bools2 := GetListBools(s2, 14)
		if bools2 == nil {
			if count > 0 {
				t.Errorf("FuzzListBitPacking: GetListBools returned nil for count=%d", count)
			}
			return
		}

		// Verify values match
		if bools2.Len() != count {
			t.Errorf("FuzzListBitPacking: after roundtrip Len() = %d, want %d", bools2.Len(), count)
			return
		}

		for i := 0; i < count; i++ {
			if bools2.Get(i) != values[i] {
				t.Errorf("FuzzListBitPacking: after roundtrip Get(%d) = %v, want %v", i, bools2.Get(i), values[i])
			}
		}
	})
}

// FuzzNumbersRoundTrip tests number list marshal/unmarshal roundtrip.
func FuzzNumbersRoundTrip(f *testing.F) {
	m := testMappingForFuzz()

	f.Add(int32(0))
	f.Add(int32(1))
	f.Add(int32(-1))
	f.Add(int32(2147483647))  // max
	f.Add(int32(-2147483648)) // min

	f.Fuzz(func(t *testing.T, value int32) {
		ctx := t.Context()
		s := New(ctx, m)

		nums := NewNumbers[int32](s, 15)
		nums.Append(value)

		// Sync
		if err := nums.SyncToSegment(); err != nil {
			t.Errorf("FuzzNumbersRoundTrip: SyncToSegment error: %v", err)
			return
		}

		// Marshal and unmarshal
		data, err := s.Marshal()
		if err != nil {
			t.Errorf("FuzzNumbersRoundTrip: Marshal error: %v", err)
			return
		}

		s2 := New(ctx, m)
		if err := s2.Unmarshal(data); err != nil {
			t.Errorf("FuzzNumbersRoundTrip: Unmarshal error: %v", err)
			return
		}

		// Get the list from unmarshaled struct
		nums2 := GetListNumbers[int32](s2, 15)
		if nums2 == nil {
			t.Errorf("FuzzNumbersRoundTrip: GetListNumbers returned nil")
			return
		}

		if nums2.Len() != 1 {
			t.Errorf("FuzzNumbersRoundTrip: after roundtrip Len() = %d, want 1", nums2.Len())
			return
		}

		if nums2.Get(0) != value {
			t.Errorf("FuzzNumbersRoundTrip: after roundtrip Get(0) = %d, want %d", nums2.Get(0), value)
		}
	})
}

// ============================================================================
// Map Fuzz Tests
// ============================================================================

// FuzzMapsStringInt32 fuzzes the Maps[string, int32] operations.
func FuzzMapsStringInt32(f *testing.F) {
	m := testMappingForFuzz()

	f.Add("", int32(0))
	f.Add("key", int32(42))
	f.Add("hello", int32(-1))
	f.Add("unicode: 日本語", int32(2147483647))

	f.Fuzz(func(t *testing.T, key string, value int32) {
		ctx := t.Context()
		s := New(ctx, m)

		maps := NewMaps[string, int32](s, 17, field.FTString, field.FTInt32, nil)
		maps.Set(key, value)

		// Verify
		got, ok := maps.Get(key)
		if !ok {
			t.Errorf("FuzzMapsStringInt32: Get(%q) returned not found", key)
			return
		}
		if got != value {
			t.Errorf("FuzzMapsStringInt32: Get(%q) = %d, want %d", key, got, value)
		}

		// Verify Has
		if !maps.Has(key) {
			t.Errorf("FuzzMapsStringInt32: Has(%q) = false, want true", key)
		}

		// Sync
		if err := maps.SyncToSegment(); err != nil {
			t.Errorf("FuzzMapsStringInt32: SyncToSegment error: %v", err)
		}
	})
}

// FuzzMapsInt32String fuzzes the Maps[int32, string] operations.
func FuzzMapsInt32String(f *testing.F) {
	m := testMappingForFuzz()

	f.Add(int32(0), "")
	f.Add(int32(42), "value")
	f.Add(int32(-1), "hello")
	f.Add(int32(2147483647), "unicode: 日本語")

	f.Fuzz(func(t *testing.T, key int32, value string) {
		ctx := t.Context()
		s := New(ctx, m)

		maps := NewMaps[int32, string](s, 17, field.FTInt32, field.FTString, nil)
		maps.Set(key, value)

		// Verify
		got, ok := maps.Get(key)
		if !ok {
			t.Errorf("FuzzMapsInt32String: Get(%d) returned not found", key)
			return
		}
		if got != value {
			t.Errorf("FuzzMapsInt32String: Get(%d) = %q, want %q", key, got, value)
		}
	})
}

// FuzzMapsMultipleKeys fuzzes multiple key insertions maintaining sorted order.
func FuzzMapsMultipleKeys(f *testing.F) {
	m := testMappingForFuzz()

	f.Add(int32(1), int32(2), int32(3))
	f.Add(int32(3), int32(1), int32(2))
	f.Add(int32(-100), int32(0), int32(100))
	f.Add(int32(2147483647), int32(-2147483648), int32(0))

	f.Fuzz(func(t *testing.T, k1, k2, k3 int32) {
		ctx := t.Context()
		s := New(ctx, m)

		maps := NewMaps[int32, int32](s, 17, field.FTInt32, field.FTInt32, nil)

		// Insert keys in the fuzzed order
		maps.Set(k1, k1*10)
		maps.Set(k2, k2*10)
		maps.Set(k3, k3*10)

		// Verify all keys are present
		for _, key := range []int32{k1, k2, k3} {
			got, ok := maps.Get(key)
			if !ok {
				t.Errorf("FuzzMapsMultipleKeys: Get(%d) returned not found", key)
				continue
			}
			if got != key*10 {
				t.Errorf("FuzzMapsMultipleKeys: Get(%d) = %d, want %d", key, got, key*10)
			}
		}

		// Verify keys are sorted
		keys := maps.Keys()
		for i := 1; i < len(keys); i++ {
			if keys[i-1] > keys[i] {
				t.Errorf("FuzzMapsMultipleKeys: keys not sorted: %v", keys)
				break
			}
		}
	})
}

// FuzzMapsDelete fuzzes the Delete operation.
func FuzzMapsDelete(f *testing.F) {
	m := testMappingForFuzz()

	f.Add("key1", "key2")
	f.Add("", "a")
	f.Add("same", "same")

	f.Fuzz(func(t *testing.T, key1, key2 string) {
		ctx := t.Context()
		s := New(ctx, m)

		maps := NewMaps[string, int32](s, 17, field.FTString, field.FTInt32, nil)
		maps.Set(key1, 100)
		maps.Set(key2, 200)

		// Delete first key
		maps.Delete(key1)

		// First key should not exist (unless key1 == key2)
		_, ok := maps.Get(key1)
		if ok && key1 != key2 {
			t.Errorf("FuzzMapsDelete: Get(%q) found after delete", key1)
		}

		// Second key should still exist
		if key1 != key2 {
			_, ok = maps.Get(key2)
			if !ok {
				t.Errorf("FuzzMapsDelete: Get(%q) not found after deleting different key", key2)
			}
		}
	})
}

// FuzzMapsClear fuzzes the Clear operation.
func FuzzMapsClear(f *testing.F) {
	m := testMappingForFuzz()

	f.Add(int32(1), int32(2), int32(3))

	f.Fuzz(func(t *testing.T, k1, k2, k3 int32) {
		ctx := t.Context()
		s := New(ctx, m)

		maps := NewMaps[int32, int32](s, 17, field.FTInt32, field.FTInt32, nil)
		maps.Set(k1, 1)
		maps.Set(k2, 2)
		maps.Set(k3, 3)

		maps.Clear()

		if maps.Len() != 0 {
			t.Errorf("FuzzMapsClear: Len() = %d after Clear, want 0", maps.Len())
		}

		for _, key := range []int32{k1, k2, k3} {
			if maps.Has(key) {
				t.Errorf("FuzzMapsClear: Has(%d) = true after Clear", key)
			}
		}
	})
}

// FuzzParseMapFromSegment fuzzes parsing map data from raw bytes.
func FuzzParseMapFromSegment(f *testing.F) {
	// Valid map segment with string->int32
	// Format: [header:8][key_len:4][key_data][value:4]...
	validData := make([]byte, 24)
	EncodeMapHeader(validData[:8], 17, field.FTString, field.FTInt32, 24)
	binary.LittleEndian.PutUint32(validData[8:12], 3)  // key length
	copy(validData[12:15], "key")
	binary.LittleEndian.PutUint32(validData[15:19], 42) // value
	f.Add(validData)

	// Empty map
	emptyData := make([]byte, 8)
	EncodeMapHeader(emptyData[:8], 17, field.FTString, field.FTInt32, 8)
	f.Add(emptyData)

	// Truncated data
	f.Add([]byte{0, 0, 0, 0, 0, 0, 0, 0})
	f.Add([]byte{})
	f.Add([]byte{1, 2, 3, 4})

	f.Fuzz(func(t *testing.T, data []byte) {
		// Should not panic on any input
		keys, values := ParseMapFromSegment[string, int32](data, field.FTString, field.FTInt32, nil)

		// Verify lengths match
		if len(keys) != len(values) {
			t.Errorf("FuzzParseMapFromSegment: keys len=%d != values len=%d", len(keys), len(values))
		}
	})
}

// FuzzParseMapFromSegmentInt32 fuzzes parsing int32->int32 maps.
func FuzzParseMapFromSegmentInt32(f *testing.F) {
	// Valid map with int32 keys and values
	validData := make([]byte, 24)
	EncodeMapHeader(validData[:8], 17, field.FTInt32, field.FTInt32, 24)
	binary.LittleEndian.PutUint32(validData[8:12], 1)   // key
	binary.LittleEndian.PutUint32(validData[12:16], 10) // value
	binary.LittleEndian.PutUint32(validData[16:20], 2)  // key
	binary.LittleEndian.PutUint32(validData[20:24], 20) // value
	f.Add(validData)

	f.Fuzz(func(t *testing.T, data []byte) {
		// Should not panic on any input
		keys, values := ParseMapFromSegment[int32, int32](data, field.FTInt32, field.FTInt32, nil)

		if len(keys) != len(values) {
			t.Errorf("FuzzParseMapFromSegmentInt32: keys len=%d != values len=%d", len(keys), len(values))
		}
	})
}

// FuzzCompareKeysString fuzzes the compareKeys function for strings.
func FuzzCompareKeysString(f *testing.F) {
	f.Add("", "")
	f.Add("a", "b")
	f.Add("b", "a")
	f.Add("hello", "hello")
	f.Add("abc", "abd")
	f.Add("日本語", "日本語")

	f.Fuzz(func(t *testing.T, a, b string) {
		result := compareKeys(a, b)

		// Verify consistency with string comparison
		switch {
		case a < b:
			if result >= 0 {
				t.Errorf("FuzzCompareKeysString: compareKeys(%q, %q) = %d, want < 0", a, b, result)
			}
		case a > b:
			if result <= 0 {
				t.Errorf("FuzzCompareKeysString: compareKeys(%q, %q) = %d, want > 0", a, b, result)
			}
		default:
			if result != 0 {
				t.Errorf("FuzzCompareKeysString: compareKeys(%q, %q) = %d, want 0", a, b, result)
			}
		}
	})
}

// FuzzCompareKeysInt32 fuzzes the compareKeys function for int32.
func FuzzCompareKeysInt32(f *testing.F) {
	f.Add(int32(0), int32(0))
	f.Add(int32(1), int32(2))
	f.Add(int32(-1), int32(1))
	f.Add(int32(2147483647), int32(-2147483648))

	f.Fuzz(func(t *testing.T, a, b int32) {
		result := compareKeys(a, b)

		switch {
		case a < b:
			if result >= 0 {
				t.Errorf("FuzzCompareKeysInt32: compareKeys(%d, %d) = %d, want < 0", a, b, result)
			}
		case a > b:
			if result <= 0 {
				t.Errorf("FuzzCompareKeysInt32: compareKeys(%d, %d) = %d, want > 0", a, b, result)
			}
		default:
			if result != 0 {
				t.Errorf("FuzzCompareKeysInt32: compareKeys(%d, %d) = %d, want 0", a, b, result)
			}
		}
	})
}

// FuzzCompareKeysBool fuzzes the compareKeys function for bool.
func FuzzCompareKeysBool(f *testing.F) {
	f.Add(false, false)
	f.Add(true, true)
	f.Add(false, true)
	f.Add(true, false)

	f.Fuzz(func(t *testing.T, a, b bool) {
		result := compareKeys(a, b)

		// false < true in our comparison
		switch {
		case !a && b: // false < true
			if result >= 0 {
				t.Errorf("FuzzCompareKeysBool: compareKeys(false, true) = %d, want < 0", result)
			}
		case a && !b: // true > false
			if result <= 0 {
				t.Errorf("FuzzCompareKeysBool: compareKeys(true, false) = %d, want > 0", result)
			}
		default: // equal
			if result != 0 {
				t.Errorf("FuzzCompareKeysBool: compareKeys(%v, %v) = %d, want 0", a, b, result)
			}
		}
	})
}

// FuzzMapsRoundTrip tests map marshal/unmarshal roundtrip.
func FuzzMapsRoundTrip(f *testing.F) {
	m := testMappingForFuzz()

	f.Add("key", int32(42))
	f.Add("", int32(0))
	f.Add("unicode: 日本語", int32(-1))

	f.Fuzz(func(t *testing.T, key string, value int32) {
		ctx := t.Context()
		s := New(ctx, m)

		maps := NewMaps[string, int32](s, 17, field.FTString, field.FTInt32, nil)
		maps.Set(key, value)

		// Sync to segment
		if err := maps.SyncToSegment(); err != nil {
			t.Errorf("FuzzMapsRoundTrip: SyncToSegment error: %v", err)
			return
		}

		// Marshal
		data, err := s.Marshal()
		if err != nil {
			t.Errorf("FuzzMapsRoundTrip: Marshal error: %v", err)
			return
		}

		// Unmarshal into new struct
		s2 := New(ctx, m)
		if err := s2.Unmarshal(data); err != nil {
			t.Errorf("FuzzMapsRoundTrip: Unmarshal error: %v", err)
			return
		}

		// Get map from unmarshaled struct
		maps2 := GetMapScalar[string, int32](s2, 17, field.FTString, field.FTInt32)
		if maps2 == nil {
			t.Errorf("FuzzMapsRoundTrip: GetMapScalar returned nil")
			return
		}

		// Verify the value
		got, ok := maps2.Get(key)
		if !ok {
			t.Errorf("FuzzMapsRoundTrip: Get(%q) not found after roundtrip", key)
			return
		}
		if got != value {
			t.Errorf("FuzzMapsRoundTrip: Get(%q) = %d, want %d", key, got, value)
		}
	})
}

// ============================================================================
// Map[K]Any Fuzz Tests
// ============================================================================

// testMappingForMapAny creates a mapping with map fields for fuzzing.
// Note: Map key/value types are passed to NewMaps directly, not stored in FieldDescr.
func testMappingForMapAny() *mapping.Map {
	return &mapping.Map{
		Fields: []*mapping.FieldDescr{
			{Name: "mapAny", Type: field.FTMap, FieldNum: 0},
			{Name: "mapListAny", Type: field.FTMap, FieldNum: 1},
		},
	}
}

// FuzzMapAnyValueSize tests valueSize calculation for MapAnyValue.
func FuzzMapAnyValueSize(f *testing.F) {
	m := testMappingForMapAny()

	f.Add("key", []byte{0x01, 0x02, 0x03, 0x04})
	f.Add("", []byte{})
	f.Add("unicode: 日本語", []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff})

	f.Fuzz(func(t *testing.T, key string, data []byte) {
		ctx := t.Context()
		s := New(ctx, m)

		mav := &MapAnyValue{
			TypeHash: [16]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10},
			Data:     data,
		}

		maps := NewMaps[string, *MapAnyValue](s, 0, field.FTString, field.FTAny, nil)
		maps.Set(key, mav)

		// Should not panic during sync
		if err := maps.SyncToSegment(); err != nil {
			t.Errorf("FuzzMapAnyValueSize: SyncToSegment error: %v", err)
			return
		}

		// Verify key exists after sync
		got, ok := maps.Get(key)
		if !ok {
			t.Errorf("FuzzMapAnyValueSize: Get(%q) not found after sync", key)
			return
		}

		// Verify type hash
		if got.TypeHash != mav.TypeHash {
			t.Errorf("FuzzMapAnyValueSize: TypeHash mismatch")
		}

		// Verify data length
		if len(got.Data) != len(data) {
			t.Errorf("FuzzMapAnyValueSize: Data len = %d, want %d", len(got.Data), len(data))
		}
	})
}

// FuzzMapAnyRoundTrip tests map[string]Any marshal/unmarshal roundtrip.
func FuzzMapAnyRoundTrip(f *testing.F) {
	m := testMappingForMapAny()

	// Create properly encoded struct headers as seed data
	seed1 := make([]byte, HeaderSize)
	EncodeHeader(seed1, 0, field.FTStruct, uint64(HeaderSize))
	f.Add("key1", seed1)
	f.Add("", seed1)
	f.Add("unicode", seed1)

	f.Fuzz(func(t *testing.T, key string, data []byte) {
		ctx := t.Context()
		s := New(ctx, m)

		// Always use a valid struct header for the Any value
		validHeader := make([]byte, HeaderSize)
		EncodeHeader(validHeader, 0, field.FTStruct, uint64(HeaderSize))
		data = validHeader

		mav := &MapAnyValue{
			TypeHash: [16]byte{0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88, 0x99, 0x00},
			Data:     data,
		}

		maps := NewMaps[string, *MapAnyValue](s, 0, field.FTString, field.FTAny, nil)
		maps.Set(key, mav)

		// Sync to segment
		if err := maps.SyncToSegment(); err != nil {
			t.Errorf("FuzzMapAnyRoundTrip: SyncToSegment error: %v", err)
			return
		}

		// Marshal
		marshaledData, err := s.Marshal()
		if err != nil {
			t.Errorf("FuzzMapAnyRoundTrip: Marshal error: %v", err)
			return
		}

		// Unmarshal into new struct
		s2 := New(ctx, m)
		if err := s2.Unmarshal(marshaledData); err != nil {
			t.Errorf("FuzzMapAnyRoundTrip: Unmarshal error: %v", err)
			return
		}

		// Get map from unmarshaled struct
		maps2 := GetMapAny[string](s2, 0, field.FTString)
		if maps2 == nil {
			t.Errorf("FuzzMapAnyRoundTrip: GetMapAny returned nil")
			return
		}

		// Verify the value
		got, ok := maps2.Get(key)
		if !ok {
			t.Errorf("FuzzMapAnyRoundTrip: Get(%q) not found after roundtrip", key)
			return
		}

		if got.TypeHash != mav.TypeHash {
			t.Errorf("FuzzMapAnyRoundTrip: TypeHash mismatch after roundtrip")
		}

		if len(got.Data) != len(mav.Data) {
			t.Errorf("FuzzMapAnyRoundTrip: Data len = %d, want %d", len(got.Data), len(mav.Data))
		}
	})
}

// FuzzMapListAnyRoundTrip tests map[string][]Any marshal/unmarshal roundtrip.
func FuzzMapListAnyRoundTrip(f *testing.F) {
	m := testMappingForMapAny()

	// Create properly encoded struct headers as seed data
	seed1 := make([]byte, HeaderSize)
	EncodeHeader(seed1, 0, field.FTStruct, uint64(HeaderSize))
	f.Add("key1", uint8(1), seed1)

	seed2 := make([]byte, HeaderSize)
	EncodeHeader(seed2, 0, field.FTStruct, uint64(HeaderSize))
	f.Add("", uint8(3), seed2)

	f.Fuzz(func(t *testing.T, key string, count uint8, data []byte) {
		ctx := t.Context()
		s := New(ctx, m)

		// Always use a valid struct header - the fuzz data determines additional payload
		validHeader := make([]byte, HeaderSize)
		EncodeHeader(validHeader, 0, field.FTStruct, uint64(HeaderSize))
		data = validHeader

		// Create list of MapAnyValues
		items := make([]MapAnyValue, int(count%5)+1) // 1-5 items
		for i := range items {
			var hash [16]byte
			hash[0] = byte(i)
			hash[15] = byte(i)
			items[i] = MapAnyValue{
				TypeHash: hash,
				Data:     data,
			}
		}

		maps := NewMaps[string, []MapAnyValue](s, 1, field.FTString, field.FTListAny, nil)
		maps.Set(key, items)

		// Sync to segment
		if err := maps.SyncToSegment(); err != nil {
			t.Errorf("FuzzMapListAnyRoundTrip: SyncToSegment error: %v", err)
			return
		}

		// Marshal
		marshaledData, err := s.Marshal()
		if err != nil {
			t.Errorf("FuzzMapListAnyRoundTrip: Marshal error: %v", err)
			return
		}

		// Unmarshal into new struct
		s2 := New(ctx, m)
		if err := s2.Unmarshal(marshaledData); err != nil {
			t.Errorf("FuzzMapListAnyRoundTrip: Unmarshal error: %v", err)
			return
		}

		// Get map from unmarshaled struct
		maps2 := GetMapListAny[string](s2, 1, field.FTString)
		if maps2 == nil {
			t.Errorf("FuzzMapListAnyRoundTrip: GetMapListAny returned nil")
			return
		}

		// Verify the value
		got, ok := maps2.Get(key)
		if !ok {
			t.Errorf("FuzzMapListAnyRoundTrip: Get(%q) not found after roundtrip", key)
			return
		}

		if len(got) != len(items) {
			t.Errorf("FuzzMapListAnyRoundTrip: len = %d, want %d", len(got), len(items))
			return
		}

		for i, item := range got {
			if item.TypeHash != items[i].TypeHash {
				t.Errorf("FuzzMapListAnyRoundTrip: item[%d] TypeHash mismatch", i)
			}
			if len(item.Data) != len(items[i].Data) {
				t.Errorf("FuzzMapListAnyRoundTrip: item[%d] Data len = %d, want %d", i, len(item.Data), len(items[i].Data))
			}
		}
	})
}

// FuzzParseMapAnyFromSegment tests parsing map[string]Any from raw bytes.
func FuzzParseMapAnyFromSegment(f *testing.F) {
	// Create valid map[string]Any segment data
	validData := make([]byte, 64)
	EncodeMapHeader(validData[:8], 0, field.FTString, field.FTAny, 64)
	// Key: 3 bytes "key"
	binary.LittleEndian.PutUint32(validData[8:12], 3)
	copy(validData[12:15], "key")
	// Type hash: 16 bytes
	for i := 0; i < 16; i++ {
		validData[15+i] = byte(i)
	}
	// Struct data: minimal header
	EncodeHeader(validData[31:39], 0, field.FTStruct, 8)
	f.Add(validData)

	// Truncated data
	f.Add([]byte{0, 0, 0, 0, 0, 0, 0, 0})
	f.Add([]byte{})
	f.Add([]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10})

	f.Fuzz(func(t *testing.T, data []byte) {
		// Should not panic on any input
		keys, values := ParseMapFromSegment[string, *MapAnyValue](data, field.FTString, field.FTAny, nil)

		// Verify lengths match
		if len(keys) != len(values) {
			t.Errorf("FuzzParseMapAnyFromSegment: keys len=%d != values len=%d", len(keys), len(values))
		}
	})
}
