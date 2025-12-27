package codec

import (
	"bytes"
	"testing"
	"unsafe"

	"github.com/bearlytools/claw/clawc/languages/go/field"
	"github.com/bearlytools/claw/clawc/languages/go/mapping"
	"github.com/bearlytools/claw/clawc/languages/go/structs/header"
)

func TestEncoderForType(t *testing.T) {
	tests := []struct {
		name string
		ft   field.Type
	}{
		{name: "Success: FTBool returns encodeScalar32", ft: field.FTBool},
		{name: "Success: FTInt8 returns encodeScalar32", ft: field.FTInt8},
		{name: "Success: FTInt16 returns encodeScalar32", ft: field.FTInt16},
		{name: "Success: FTInt32 returns encodeScalar32", ft: field.FTInt32},
		{name: "Success: FTUint8 returns encodeScalar32", ft: field.FTUint8},
		{name: "Success: FTUint16 returns encodeScalar32", ft: field.FTUint16},
		{name: "Success: FTUint32 returns encodeScalar32", ft: field.FTUint32},
		{name: "Success: FTFloat32 returns encodeScalar32", ft: field.FTFloat32},
		{name: "Success: FTInt64 returns encodeScalar64", ft: field.FTInt64},
		{name: "Success: FTUint64 returns encodeScalar64", ft: field.FTUint64},
		{name: "Success: FTFloat64 returns encodeScalar64", ft: field.FTFloat64},
		{name: "Success: FTString returns encodeBytes", ft: field.FTString},
		{name: "Success: FTBytes returns encodeBytes", ft: field.FTBytes},
		{name: "Success: FTStruct returns encodeStruct", ft: field.FTStruct},
		{name: "Success: FTListBools returns encodeListBools", ft: field.FTListBools},
		{name: "Success: FTListInt8 returns encodeListInt8", ft: field.FTListInt8},
		{name: "Success: FTListUint8 returns encodeListUint8", ft: field.FTListUint8},
		{name: "Success: FTListInt16 returns encodeListInt16", ft: field.FTListInt16},
		{name: "Success: FTListUint16 returns encodeListUint16", ft: field.FTListUint16},
		{name: "Success: FTListInt32 returns encodeListInt32", ft: field.FTListInt32},
		{name: "Success: FTListUint32 returns encodeListUint32", ft: field.FTListUint32},
		{name: "Success: FTListFloat32 returns encodeListFloat32", ft: field.FTListFloat32},
		{name: "Success: FTListInt64 returns encodeListInt64", ft: field.FTListInt64},
		{name: "Success: FTListUint64 returns encodeListUint64", ft: field.FTListUint64},
		{name: "Success: FTListFloat64 returns encodeListFloat64", ft: field.FTListFloat64},
		{name: "Success: FTListBytes returns encodeListBytes", ft: field.FTListBytes},
		{name: "Success: FTListStrings returns encodeListStrings", ft: field.FTListStrings},
		{name: "Success: FTListStructs returns encodeListStructs", ft: field.FTListStructs},
		{name: "Success: unknown type returns encodeUnsupported", ft: field.Type(255)},
	}

	for _, test := range tests {
		enc := encoderForType(test.ft)
		if enc == nil {
			t.Errorf("[TestEncoderForType](%s): got nil encoder, want non-nil", test.name)
			continue
		}
	}
}

func TestScanSizerForType(t *testing.T) {
	tests := []struct {
		name string
		ft   field.Type
	}{
		{name: "Success: FTBool returns scanSizeScalar8", ft: field.FTBool},
		{name: "Success: FTInt8 returns scanSizeScalar8", ft: field.FTInt8},
		{name: "Success: FTInt16 returns scanSizeScalar8", ft: field.FTInt16},
		{name: "Success: FTInt32 returns scanSizeScalar8", ft: field.FTInt32},
		{name: "Success: FTUint8 returns scanSizeScalar8", ft: field.FTUint8},
		{name: "Success: FTUint16 returns scanSizeScalar8", ft: field.FTUint16},
		{name: "Success: FTUint32 returns scanSizeScalar8", ft: field.FTUint32},
		{name: "Success: FTFloat32 returns scanSizeScalar8", ft: field.FTFloat32},
		{name: "Success: FTInt64 returns scanSizeScalar16", ft: field.FTInt64},
		{name: "Success: FTUint64 returns scanSizeScalar16", ft: field.FTUint64},
		{name: "Success: FTFloat64 returns scanSizeScalar16", ft: field.FTFloat64},
		{name: "Success: FTString returns scanSizeBytes", ft: field.FTString},
		{name: "Success: FTBytes returns scanSizeBytes", ft: field.FTBytes},
		{name: "Success: FTStruct returns scanSizeStruct", ft: field.FTStruct},
		{name: "Success: FTListBools returns scanSizeListBools", ft: field.FTListBools},
		{name: "Success: FTListInt8 returns scanSizeListInt8", ft: field.FTListInt8},
		{name: "Success: FTListUint8 returns scanSizeListInt8", ft: field.FTListUint8},
		{name: "Success: FTListInt16 returns scanSizeListInt16", ft: field.FTListInt16},
		{name: "Success: FTListUint16 returns scanSizeListInt16", ft: field.FTListUint16},
		{name: "Success: FTListInt32 returns scanSizeListInt32", ft: field.FTListInt32},
		{name: "Success: FTListUint32 returns scanSizeListInt32", ft: field.FTListUint32},
		{name: "Success: FTListFloat32 returns scanSizeListInt32", ft: field.FTListFloat32},
		{name: "Success: FTListInt64 returns scanSizeListInt64", ft: field.FTListInt64},
		{name: "Success: FTListUint64 returns scanSizeListInt64", ft: field.FTListUint64},
		{name: "Success: FTListFloat64 returns scanSizeListInt64", ft: field.FTListFloat64},
		{name: "Success: FTListBytes returns scanSizeListBytes", ft: field.FTListBytes},
		{name: "Success: FTListStrings returns scanSizeListBytes", ft: field.FTListStrings},
		{name: "Success: FTListStructs returns scanSizeListStructs", ft: field.FTListStructs},
		{name: "Success: unknown type returns scanSizeUnknown", ft: field.Type(255)},
	}

	for _, test := range tests {
		scanner := scanSizerForType(test.ft)
		if scanner == nil {
			t.Errorf("[TestScanSizerForType](%s): got nil scanner, want non-nil", test.name)
			continue
		}
	}
}

func TestScanSizeScalar8(t *testing.T) {
	hdr := header.New()
	hdr.SetFieldNum(0)
	hdr.SetFieldType(field.FTBool)
	hdr.SetFinal40(1)

	size := scanSizeScalar8(nil, hdr)
	if size != 8 {
		t.Errorf("[TestScanSizeScalar8]: got %d, want 8", size)
	}
}

func TestScanSizeScalar16(t *testing.T) {
	hdr := header.New()
	hdr.SetFieldNum(0)
	hdr.SetFieldType(field.FTInt64)
	hdr.SetFinal40(0)

	size := scanSizeScalar16(nil, hdr)
	if size != 16 {
		t.Errorf("[TestScanSizeScalar16]: got %d, want 16", size)
	}
}

func TestScanSizeBytes(t *testing.T) {
	tests := []struct {
		name     string
		dataSize uint64
		wantSize uint32
	}{
		{name: "Success: empty string", dataSize: 0, wantSize: 8},
		{name: "Success: 1 byte string (needs 7 padding)", dataSize: 1, wantSize: 16},
		{name: "Success: 5 byte string (needs 3 padding)", dataSize: 5, wantSize: 16},
		{name: "Success: 8 byte string (no padding)", dataSize: 8, wantSize: 16},
		{name: "Success: 9 byte string (needs 7 padding)", dataSize: 9, wantSize: 24},
		{name: "Success: 16 byte string (no padding)", dataSize: 16, wantSize: 24},
	}

	for _, test := range tests {
		hdr := header.New()
		hdr.SetFieldNum(0)
		hdr.SetFieldType(field.FTString)
		hdr.SetFinal40(test.dataSize)

		size := scanSizeBytes(nil, hdr)
		if size != test.wantSize {
			t.Errorf("[TestScanSizeBytes](%s): got %d, want %d", test.name, size, test.wantSize)
		}
	}
}

func TestScanSizeStruct(t *testing.T) {
	tests := []struct {
		name       string
		structSize uint64
		wantSize   uint32
	}{
		{name: "Success: empty struct", structSize: 8, wantSize: 8},
		{name: "Success: struct with one field", structSize: 16, wantSize: 16},
		{name: "Success: larger struct", structSize: 128, wantSize: 128},
	}

	for _, test := range tests {
		hdr := header.New()
		hdr.SetFieldNum(0)
		hdr.SetFieldType(field.FTStruct)
		hdr.SetFinal40(test.structSize)

		size := scanSizeStruct(nil, hdr)
		if size != test.wantSize {
			t.Errorf("[TestScanSizeStruct](%s): got %d, want %d", test.name, size, test.wantSize)
		}
	}
}

func TestScanSizeListBools(t *testing.T) {
	// Formula: wordsNeeded = (items / 64) + 1
	// Size = 8 (header) + wordsNeeded * 8
	tests := []struct {
		name     string
		numItems uint64
		wantSize uint32
	}{
		{name: "Success: empty list (1 word)", numItems: 0, wantSize: 16},
		{name: "Success: 1 bool (1 word)", numItems: 1, wantSize: 16},
		{name: "Success: 63 bools (1 word)", numItems: 63, wantSize: 16},
		{name: "Success: 64 bools (2 words)", numItems: 64, wantSize: 24},
		{name: "Success: 65 bools (2 words)", numItems: 65, wantSize: 24},
		{name: "Success: 127 bools (2 words)", numItems: 127, wantSize: 24},
		{name: "Success: 128 bools (3 words)", numItems: 128, wantSize: 32},
		{name: "Success: 129 bools (3 words)", numItems: 129, wantSize: 32},
	}

	for _, test := range tests {
		hdr := header.New()
		hdr.SetFieldNum(0)
		hdr.SetFieldType(field.FTListBools)
		hdr.SetFinal40(test.numItems)

		size := scanSizeListBools(nil, hdr)
		if size != test.wantSize {
			t.Errorf("[TestScanSizeListBools](%s): got %d, want %d", test.name, size, test.wantSize)
		}
	}
}

func TestScanSizeListInt8(t *testing.T) {
	tests := []struct {
		name     string
		numItems uint64
		wantSize uint32
	}{
		{name: "Success: empty list", numItems: 0, wantSize: 8},
		{name: "Success: 1 item (needs 7 padding)", numItems: 1, wantSize: 16},
		{name: "Success: 8 items (no padding)", numItems: 8, wantSize: 16},
		{name: "Success: 9 items (needs 7 padding)", numItems: 9, wantSize: 24},
		{name: "Success: 16 items (no padding)", numItems: 16, wantSize: 24},
	}

	for _, test := range tests {
		hdr := header.New()
		hdr.SetFieldNum(0)
		hdr.SetFieldType(field.FTListInt8)
		hdr.SetFinal40(test.numItems)

		size := scanSizeListInt8(nil, hdr)
		if size != test.wantSize {
			t.Errorf("[TestScanSizeListInt8](%s): got %d, want %d", test.name, size, test.wantSize)
		}
	}
}

func TestScanSizeListInt16(t *testing.T) {
	tests := []struct {
		name     string
		numItems uint64
		wantSize uint32
	}{
		{name: "Success: empty list", numItems: 0, wantSize: 8},
		{name: "Success: 1 item (2 bytes, needs 6 padding)", numItems: 1, wantSize: 16},
		{name: "Success: 4 items (8 bytes, no padding)", numItems: 4, wantSize: 16},
		{name: "Success: 5 items (10 bytes, needs 6 padding)", numItems: 5, wantSize: 24},
	}

	for _, test := range tests {
		hdr := header.New()
		hdr.SetFieldNum(0)
		hdr.SetFieldType(field.FTListInt16)
		hdr.SetFinal40(test.numItems)

		size := scanSizeListInt16(nil, hdr)
		if size != test.wantSize {
			t.Errorf("[TestScanSizeListInt16](%s): got %d, want %d", test.name, size, test.wantSize)
		}
	}
}

func TestScanSizeListInt32(t *testing.T) {
	tests := []struct {
		name     string
		numItems uint64
		wantSize uint32
	}{
		{name: "Success: empty list", numItems: 0, wantSize: 8},
		{name: "Success: 1 item (4 bytes, needs 4 padding)", numItems: 1, wantSize: 16},
		{name: "Success: 2 items (8 bytes, no padding)", numItems: 2, wantSize: 16},
		{name: "Success: 3 items (12 bytes, needs 4 padding)", numItems: 3, wantSize: 24},
	}

	for _, test := range tests {
		hdr := header.New()
		hdr.SetFieldNum(0)
		hdr.SetFieldType(field.FTListInt32)
		hdr.SetFinal40(test.numItems)

		size := scanSizeListInt32(nil, hdr)
		if size != test.wantSize {
			t.Errorf("[TestScanSizeListInt32](%s): got %d, want %d", test.name, size, test.wantSize)
		}
	}
}

func TestScanSizeListInt64(t *testing.T) {
	tests := []struct {
		name     string
		numItems uint64
		wantSize uint32
	}{
		{name: "Success: empty list", numItems: 0, wantSize: 8},
		{name: "Success: 1 item (8 bytes, no padding)", numItems: 1, wantSize: 16},
		{name: "Success: 2 items (16 bytes, no padding)", numItems: 2, wantSize: 24},
		{name: "Success: 3 items (24 bytes, no padding)", numItems: 3, wantSize: 32},
	}

	for _, test := range tests {
		hdr := header.New()
		hdr.SetFieldNum(0)
		hdr.SetFieldType(field.FTListInt64)
		hdr.SetFinal40(test.numItems)

		size := scanSizeListInt64(nil, hdr)
		if size != test.wantSize {
			t.Errorf("[TestScanSizeListInt64](%s): got %d, want %d", test.name, size, test.wantSize)
		}
	}
}

func TestScanSizeUnknown(t *testing.T) {
	hdr := header.New()
	size := scanSizeUnknown(nil, hdr)
	if size != 0 {
		t.Errorf("[TestScanSizeUnknown]: got %d, want 0", size)
	}
}

func TestEncodeScalar32(t *testing.T) {
	// Zero-value compression is always on, so zero values are skipped
	tests := []struct {
		name  string
		value uint64
		wantN int
	}{
		{name: "Success: non-zero value", value: 42, wantN: 8},
		{name: "Success: zero value skips write", value: 0, wantN: 0},
	}

	for _, test := range tests {
		hdr := header.New()
		hdr.SetFieldNum(0)
		hdr.SetFieldType(field.FTInt32)
		hdr.SetFinal40(test.value)

		var buf bytes.Buffer
		n, err := encodeScalar32(&buf, hdr, nil, nil)
		if err != nil {
			t.Errorf("[TestEncodeScalar32](%s): unexpected error: %v", test.name, err)
			continue
		}
		if n != test.wantN {
			t.Errorf("[TestEncodeScalar32](%s): got %d bytes, want %d", test.name, n, test.wantN)
		}
	}
}

func TestEncodeScalar64(t *testing.T) {
	// Zero-value compression is always on, so zero values are skipped
	tests := []struct {
		name  string
		data  []byte
		wantN int
	}{
		{name: "Success: non-zero value", data: []byte{1, 2, 3, 4, 5, 6, 7, 8}, wantN: 16},
		{name: "Success: nil pointer skips write", data: nil, wantN: 0},
		{name: "Success: all zeros skips write", data: []byte{0, 0, 0, 0, 0, 0, 0, 0}, wantN: 0},
		{name: "Success: non-zero writes", data: []byte{1, 0, 0, 0, 0, 0, 0, 0}, wantN: 16},
	}

	for _, test := range tests {
		hdr := header.New()
		hdr.SetFieldNum(0)
		hdr.SetFieldType(field.FTInt64)
		hdr.SetFinal40(0)

		var buf bytes.Buffer
		var ptr unsafe.Pointer
		if test.data != nil {
			ptr = unsafe.Pointer(&test.data)
		}
		n, err := encodeScalar64(&buf, hdr, ptr, nil)
		if err != nil {
			t.Errorf("[TestEncodeScalar64](%s): unexpected error: %v", test.name, err)
			continue
		}
		if n != test.wantN {
			t.Errorf("[TestEncodeScalar64](%s): got %d bytes, want %d", test.name, n, test.wantN)
		}
	}
}

func TestEncodeBytes(t *testing.T) {
	// Zero-value compression is always on, so empty data is skipped
	tests := []struct {
		name  string
		data  []byte
		wantN int
	}{
		{name: "Success: non-empty data", data: []byte("hello"), wantN: 16},
		{name: "Success: empty skips", data: nil, wantN: 0},
		{name: "Success: 8-byte aligned data", data: []byte("12345678"), wantN: 16},
	}

	for _, test := range tests {
		hdr := header.New()
		hdr.SetFieldNum(0)
		hdr.SetFieldType(field.FTBytes)
		if test.data != nil {
			hdr.SetFinal40(uint64(len(test.data)))
		} else {
			hdr.SetFinal40(0)
		}

		var buf bytes.Buffer
		var ptr unsafe.Pointer
		if test.data != nil {
			ptr = unsafe.Pointer(&test.data)
		}
		n, err := encodeBytes(&buf, hdr, ptr, nil)
		if err != nil {
			t.Errorf("[TestEncodeBytes](%s): unexpected error: %v", test.name, err)
			continue
		}
		if n != test.wantN {
			t.Errorf("[TestEncodeBytes](%s): got %d bytes, want %d", test.name, n, test.wantN)
		}
	}
}

func TestEncodeUnsupported(t *testing.T) {
	hdr := header.New()
	desc := &mapping.FieldDescr{Type: field.Type(255)}

	var buf bytes.Buffer
	_, err := encodeUnsupported(&buf, hdr, nil, desc)
	if err == nil {
		t.Error("[TestEncodeUnsupported]: expected error, got nil")
	}
}

func TestRegisterEncoders(t *testing.T) {
	m := &mapping.Map{
		Fields: []*mapping.FieldDescr{
			{Name: "Bool", Type: field.FTBool},
			{Name: "Int64", Type: field.FTInt64},
			{Name: "String", Type: field.FTString},
		},
	}

	registerEncoders(m)

	if len(m.Encoders) != 3 {
		t.Errorf("[TestRegisterEncoders]: got %d encoders, want 3", len(m.Encoders))
	}
	for i, enc := range m.Encoders {
		if enc == nil {
			t.Errorf("[TestRegisterEncoders]: encoder %d is nil", i)
		}
	}
}

func TestRegisterScanSizers(t *testing.T) {
	m := &mapping.Map{
		Fields: []*mapping.FieldDescr{
			{Name: "Bool", Type: field.FTBool},
			{Name: "Int64", Type: field.FTInt64},
			{Name: "ListBytes", Type: field.FTListBytes},
		},
	}

	registerScanSizers(m)

	if len(m.ScanSizers) != 3 {
		t.Errorf("[TestRegisterScanSizers]: got %d scanners, want 3", len(m.ScanSizers))
	}
	for i, scanner := range m.ScanSizers {
		if scanner == nil {
			t.Errorf("[TestRegisterScanSizers]: scanner %d is nil", i)
		}
	}
}

func TestScanSizeListBytesWithData(t *testing.T) {
	tests := []struct {
		name     string
		numItems uint64
		data     []byte
		wantSize uint32
	}{
		{
			name:     "Success: empty list",
			numItems: 0,
			data:     make([]byte, 8),
			wantSize: 8,
		},
		{
			name:     "Success: insufficient data returns partial size",
			numItems: 1,
			data:     make([]byte, 4), // less than 8 bytes
			wantSize: 0,
		},
	}

	for _, test := range tests {
		hdr := header.New()
		hdr.SetFieldNum(0)
		hdr.SetFieldType(field.FTListBytes)
		hdr.SetFinal40(test.numItems)

		size := scanSizeListBytes(test.data, hdr)
		if size != test.wantSize {
			t.Errorf("[TestScanSizeListBytesWithData](%s): got %d, want %d", test.name, size, test.wantSize)
		}
	}
}

func TestScanSizeListStructsWithData(t *testing.T) {
	tests := []struct {
		name     string
		numItems uint64
		data     []byte
		wantSize uint32
	}{
		{
			name:     "Success: empty list",
			numItems: 0,
			data:     make([]byte, 8),
			wantSize: 8,
		},
		{
			name:     "Success: insufficient data returns zero",
			numItems: 1,
			data:     make([]byte, 4),
			wantSize: 0,
		},
	}

	for _, test := range tests {
		hdr := header.New()
		hdr.SetFieldNum(0)
		hdr.SetFieldType(field.FTListStructs)
		hdr.SetFinal40(test.numItems)

		size := scanSizeListStructs(test.data, hdr)
		if size != test.wantSize {
			t.Errorf("[TestScanSizeListStructsWithData](%s): got %d, want %d", test.name, size, test.wantSize)
		}
	}
}
