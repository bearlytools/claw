package structs

import (
	"bytes"
	"testing"
	"unsafe"

	"github.com/bearlytools/claw/clawc/internal/bits"
	"github.com/bearlytools/claw/clawc/languages/go/conversions"
	"github.com/bearlytools/claw/clawc/languages/go/field"
	"github.com/bearlytools/claw/clawc/languages/go/mapping"
)

func TestLazyDecoderForType(t *testing.T) {
	tests := []struct {
		name string
		ft   field.Type
	}{
		{name: "Success: FTBool returns lazyDecodeScalar8", ft: field.FTBool},
		{name: "Success: FTInt8 returns lazyDecodeScalar8", ft: field.FTInt8},
		{name: "Success: FTInt16 returns lazyDecodeScalar8", ft: field.FTInt16},
		{name: "Success: FTInt32 returns lazyDecodeScalar8", ft: field.FTInt32},
		{name: "Success: FTUint8 returns lazyDecodeScalar8", ft: field.FTUint8},
		{name: "Success: FTUint16 returns lazyDecodeScalar8", ft: field.FTUint16},
		{name: "Success: FTUint32 returns lazyDecodeScalar8", ft: field.FTUint32},
		{name: "Success: FTFloat32 returns lazyDecodeScalar8", ft: field.FTFloat32},
		{name: "Success: FTInt64 returns lazyDecodeScalar64", ft: field.FTInt64},
		{name: "Success: FTUint64 returns lazyDecodeScalar64", ft: field.FTUint64},
		{name: "Success: FTFloat64 returns lazyDecodeScalar64", ft: field.FTFloat64},
		{name: "Success: FTString returns lazyDecodeBytes", ft: field.FTString},
		{name: "Success: FTBytes returns lazyDecodeBytes", ft: field.FTBytes},
		{name: "Success: FTStruct returns lazyDecodeStruct", ft: field.FTStruct},
		{name: "Success: FTListBools returns lazyDecodeListBools", ft: field.FTListBools},
		{name: "Success: FTListInt8 returns lazyDecodeListInt8", ft: field.FTListInt8},
		{name: "Success: FTListInt16 returns lazyDecodeListInt16", ft: field.FTListInt16},
		{name: "Success: FTListInt32 returns lazyDecodeListInt32", ft: field.FTListInt32},
		{name: "Success: FTListInt64 returns lazyDecodeListInt64", ft: field.FTListInt64},
		{name: "Success: FTListUint8 returns lazyDecodeListUint8", ft: field.FTListUint8},
		{name: "Success: FTListUint16 returns lazyDecodeListUint16", ft: field.FTListUint16},
		{name: "Success: FTListUint32 returns lazyDecodeListUint32", ft: field.FTListUint32},
		{name: "Success: FTListUint64 returns lazyDecodeListUint64", ft: field.FTListUint64},
		{name: "Success: FTListFloat32 returns lazyDecodeListFloat32", ft: field.FTListFloat32},
		{name: "Success: FTListFloat64 returns lazyDecodeListFloat64", ft: field.FTListFloat64},
		{name: "Success: FTListBytes returns lazyDecodeListBytes", ft: field.FTListBytes},
		{name: "Success: FTListStrings returns lazyDecodeListBytes", ft: field.FTListStrings},
		{name: "Success: FTListStructs returns lazyDecodeListStructs", ft: field.FTListStructs},
		{name: "Success: unknown type returns lazyDecodeNoop", ft: field.Type(255)},
	}

	for _, test := range tests {
		decoder := lazyDecoderForType(test.ft)
		if decoder == nil {
			t.Errorf("[TestLazyDecoderForType](%s): got nil decoder, want non-nil", test.name)
		}
	}
}

func TestRegisterLazyDecoders(t *testing.T) {
	m := &mapping.Map{
		Fields: []*mapping.FieldDescr{
			{Name: "Bool", Type: field.FTBool},
			{Name: "Int64", Type: field.FTInt64},
			{Name: "String", Type: field.FTString},
			{Name: "ListInt32", Type: field.FTListInt32},
		},
	}

	registerLazyDecoders(m)

	if len(m.LazyDecoders) != 4 {
		t.Errorf("[TestRegisterLazyDecoders]: got %d decoders, want 4", len(m.LazyDecoders))
	}
	for i, dec := range m.LazyDecoders {
		if dec == nil {
			t.Errorf("[TestRegisterLazyDecoders]: decoder %d is nil", i)
		}
	}
}

func TestLazyDecodeScalar8(t *testing.T) {
	m := &mapping.Map{
		Fields: []*mapping.FieldDescr{
			{Name: "Bool", Type: field.FTBool},
		},
	}

	// Create a header with bool value true
	h := NewGenericHeader()
	h.SetFieldNum(0)
	h.SetFieldType(field.FTBool)
	n := conversions.BytesToNum[uint64](h)
	*n = bits.SetBit(*n, 24, true)

	s := New(0, m)
	desc := m.Fields[0]

	lazyDecodeScalar8(unsafe.Pointer(s), 0, h, desc)

	if s.fields[0].Header == nil {
		t.Error("[TestLazyDecodeScalar8]: Header is nil after decode")
		return
	}
	if !bytes.Equal(s.fields[0].Header, h) {
		t.Errorf("[TestLazyDecodeScalar8]: Header mismatch, got %v, want %v", s.fields[0].Header, h)
	}

	// Verify we can read the value
	got, err := GetBool(s, 0)
	if err != nil {
		t.Errorf("[TestLazyDecodeScalar8]: GetBool error: %v", err)
		return
	}
	if !got {
		t.Error("[TestLazyDecodeScalar8]: expected true, got false")
	}
}

func TestLazyDecodeScalar64(t *testing.T) {
	m := &mapping.Map{
		Fields: []*mapping.FieldDescr{
			{Name: "Int64", Type: field.FTInt64},
		},
	}

	h := NewGenericHeader()
	h.SetFieldNum(0)
	h.SetFieldType(field.FTInt64)

	// Add 8 bytes of data (value = 12345678901234)
	data := make([]byte, 16)
	copy(data[:8], h)
	val := int64(12345678901234)
	data[8] = byte(val)
	data[9] = byte(val >> 8)
	data[10] = byte(val >> 16)
	data[11] = byte(val >> 24)
	data[12] = byte(val >> 32)
	data[13] = byte(val >> 40)
	data[14] = byte(val >> 48)
	data[15] = byte(val >> 56)

	s := New(0, m)
	desc := m.Fields[0]

	lazyDecodeScalar64(unsafe.Pointer(s), 0, data, desc)

	if s.fields[0].Header == nil {
		t.Error("[TestLazyDecodeScalar64]: Header is nil after decode")
		return
	}
	if s.fields[0].Ptr == nil {
		t.Error("[TestLazyDecodeScalar64]: Ptr is nil after decode")
		return
	}

	// Verify header is correct
	if !bytes.Equal(s.fields[0].Header[:8], h) {
		t.Errorf("[TestLazyDecodeScalar64]: Header mismatch")
	}
}

func TestLazyDecodeBytes(t *testing.T) {
	m := &mapping.Map{
		Fields: []*mapping.FieldDescr{
			{Name: "String", Type: field.FTString},
		},
	}

	testString := "hello world"

	h := NewGenericHeader()
	h.SetFieldNum(0)
	h.SetFieldType(field.FTString)
	h.SetFinal40(uint64(len(testString)))

	// Create data: header + string + padding
	paddingNeeded := PaddingNeeded(len(testString))
	data := make([]byte, 8+len(testString)+paddingNeeded)
	copy(data[:8], h)
	copy(data[8:], testString)

	s := New(0, m)
	desc := m.Fields[0]

	lazyDecodeBytes(unsafe.Pointer(s), 0, data, desc)

	if s.fields[0].Header == nil {
		t.Error("[TestLazyDecodeBytes]: Header is nil after decode")
		return
	}
	if s.fields[0].Ptr == nil {
		t.Error("[TestLazyDecodeBytes]: Ptr is nil after decode")
		return
	}

	got, err := GetBytes(s, 0)
	if err != nil {
		t.Errorf("[TestLazyDecodeBytes]: GetBytes error: %v", err)
		return
	}
	if string(*got) != testString {
		t.Errorf("[TestLazyDecodeBytes]: got %q, want %q", string(*got), testString)
	}
}

func TestLazyDecodeBytesEmpty(t *testing.T) {
	m := &mapping.Map{
		Fields: []*mapping.FieldDescr{
			{Name: "String", Type: field.FTString},
		},
	}

	// Create header with size 0
	h := NewGenericHeader()
	h.SetFieldNum(0)
	h.SetFieldType(field.FTString)
	h.SetFinal40(0)

	data := make([]byte, 8)
	copy(data[:8], h)

	s := New(0, m)
	desc := m.Fields[0]

	lazyDecodeBytes(unsafe.Pointer(s), 0, data, desc)

	if s.fields[0].Header == nil {
		t.Error("[TestLazyDecodeBytesEmpty]: Header is nil after decode")
		return
	}
	// Ptr should be nil for empty string
	if s.fields[0].Ptr != nil {
		t.Error("[TestLazyDecodeBytesEmpty]: Ptr should be nil for empty string")
	}
}

func TestLazyDecodeNoop(t *testing.T) {
	m := &mapping.Map{
		Fields: []*mapping.FieldDescr{
			{Name: "Unknown", Type: field.Type(255)},
		},
	}

	data := make([]byte, 8)
	s := New(0, m)
	desc := m.Fields[0]

	// Should not panic
	lazyDecodeNoop(unsafe.Pointer(s), 0, data, desc)

	// Fields should remain unmodified
	if s.fields[0].Header != nil {
		t.Error("[TestLazyDecodeNoop]: Header should be nil")
	}
	if s.fields[0].Ptr != nil {
		t.Error("[TestLazyDecodeNoop]: Ptr should be nil")
	}
}

func TestLazyDecodeListNumbers(t *testing.T) {
	tests := []struct {
		name       string
		ft         field.Type
		values     []int32
		decodeFunc func(unsafe.Pointer, uint16, []byte, *mapping.FieldDescr)
	}{
		{
			name:       "Success: ListInt32",
			ft:         field.FTListInt32,
			values:     []int32{1, 2, 3},
			decodeFunc: lazyDecodeListInt32,
		},
	}

	for _, test := range tests {
		m := &mapping.Map{
			Fields: []*mapping.FieldDescr{
				{Name: "List", Type: test.ft},
			},
		}

		// Create encoded list data
		nums := NewNumbers[int32]()
		nums.Append(test.values...)
		encoded := nums.Encode()
		GenericHeader(encoded[:8]).SetFieldNum(0)

		s := New(0, m)
		desc := m.Fields[0]

		test.decodeFunc(unsafe.Pointer(s), 0, encoded, desc)

		if s.fields[0].Header == nil {
			t.Errorf("[TestLazyDecodeListNumbers](%s): Header is nil", test.name)
			continue
		}
		if s.fields[0].Ptr == nil {
			t.Errorf("[TestLazyDecodeListNumbers](%s): Ptr is nil", test.name)
			continue
		}

		// Verify we can read the list
		list, err := GetListNumber[int32](s, 0)
		if err != nil {
			t.Errorf("[TestLazyDecodeListNumbers](%s): GetListNumber error: %v", test.name, err)
			continue
		}
		if list.Len() != len(test.values) {
			t.Errorf("[TestLazyDecodeListNumbers](%s): Len got %d, want %d", test.name, list.Len(), len(test.values))
			continue
		}
		for i, want := range test.values {
			if list.Get(i) != want {
				t.Errorf("[TestLazyDecodeListNumbers](%s): item[%d] got %d, want %d", test.name, i, list.Get(i), want)
			}
		}
	}
}

func TestLazyDecodeListBools(t *testing.T) {
	m := &mapping.Map{
		Fields: []*mapping.FieldDescr{
			{Name: "ListBools", Type: field.FTListBools},
		},
	}

	bools := NewBools(0)
	bools.Append(true, false, true, false, true)
	encoded := bools.Encode()
	GenericHeader(encoded[:8]).SetFieldNum(0)

	s := New(0, m)
	desc := m.Fields[0]

	lazyDecodeListBools(unsafe.Pointer(s), 0, encoded, desc)

	if s.fields[0].Header == nil {
		t.Error("[TestLazyDecodeListBools]: Header is nil")
		return
	}
	if s.fields[0].Ptr == nil {
		t.Error("[TestLazyDecodeListBools]: Ptr is nil")
		return
	}

	list, err := GetListBool(s, 0)
	if err != nil {
		t.Errorf("[TestLazyDecodeListBools]: GetListBool error: %v", err)
		return
	}
	if list.Len() != 5 {
		t.Errorf("[TestLazyDecodeListBools]: Len got %d, want 5", list.Len())
		return
	}

	expected := []bool{true, false, true, false, true}
	for i, want := range expected {
		if list.Get(i) != want {
			t.Errorf("[TestLazyDecodeListBools]: item[%d] got %v, want %v", i, list.Get(i), want)
		}
	}
}

func TestLazyDecodeStruct(t *testing.T) {
	innerMapping := &mapping.Map{
		Fields: []*mapping.FieldDescr{
			{Name: "Bool", Type: field.FTBool},
		},
	}

	outerMapping := &mapping.Map{
		Fields: []*mapping.FieldDescr{
			{Name: "Inner", Type: field.FTStruct, Mapping: innerMapping},
		},
	}

	inner := New(0, innerMapping)
	inner.XXXSetIsSetEnabled()
	MustSetBool(inner, 0, true)

	var buf bytes.Buffer
	_, err := inner.Marshal(&buf)
	if err != nil {
		t.Fatalf("[TestLazyDecodeStruct]: Marshal error: %v", err)
	}
	encoded := buf.Bytes()

	outer := New(0, outerMapping)
	outer.XXXSetIsSetEnabled() // Enable IsSet to properly decode IsSet-enabled data
	desc := outerMapping.Fields[0]

	lazyDecodeStruct(unsafe.Pointer(outer), 0, encoded, desc)

	if outer.fields[0].Header == nil {
		t.Error("[TestLazyDecodeStruct]: Header is nil")
		return
	}
	if outer.fields[0].Ptr == nil {
		t.Error("[TestLazyDecodeStruct]: Ptr is nil")
		return
	}

	decoded, err := GetStruct(outer, 0)
	if err != nil {
		t.Errorf("[TestLazyDecodeStruct]: GetStruct error: %v", err)
		return
	}

	boolVal, err := GetBool(decoded, 0)
	if err != nil {
		t.Errorf("[TestLazyDecodeStruct]: GetBool error: %v", err)
		return
	}
	if !boolVal {
		t.Error("[TestLazyDecodeStruct]: expected bool true, got false")
	}
}

func TestLazyDecodeStructSelfReferential(t *testing.T) {
	selfRefMapping := &mapping.Map{
		Fields: []*mapping.FieldDescr{
			{Name: "Bool", Type: field.FTBool},
			{Name: "Child", Type: field.FTStruct, Mapping: nil}, // Self-referential
		},
	}
	selfRefMapping.Fields[1].Mapping = selfRefMapping

	child := New(1, selfRefMapping)
	child.XXXSetIsSetEnabled()
	MustSetBool(child, 0, true)

	var buf bytes.Buffer
	_, err := child.Marshal(&buf)
	if err != nil {
		t.Fatalf("[TestLazyDecodeStructSelfReferential]: Marshal error: %v", err)
	}
	encoded := buf.Bytes()

	parent := New(0, selfRefMapping)
	parent.XXXSetIsSetEnabled() // Enable IsSet to properly decode IsSet-enabled data
	desc := &mapping.FieldDescr{
		Name:    "Child",
		Type:    field.FTStruct,
		Mapping: nil, // Simulate self-referential
	}

	lazyDecodeStruct(unsafe.Pointer(parent), 1, encoded, desc)

	if parent.fields[1].Ptr == nil {
		t.Error("[TestLazyDecodeStructSelfReferential]: Ptr is nil")
		return
	}
}

func TestLazyDecodeListBytes(t *testing.T) {
	m := &mapping.Map{
		Fields: []*mapping.FieldDescr{
			{Name: "ListBytes", Type: field.FTListBytes},
		},
	}

	bytesList := NewBytes()
	bytesList.Append([]byte("hello"), []byte("world"))

	var buf bytes.Buffer
	_, err := bytesList.Encode(&buf)
	if err != nil {
		t.Fatalf("[TestLazyDecodeListBytes]: Encode error: %v", err)
	}
	encoded := buf.Bytes()
	GenericHeader(encoded[:8]).SetFieldNum(0)

	s := New(0, m)
	desc := m.Fields[0]

	lazyDecodeListBytes(unsafe.Pointer(s), 0, encoded, desc)

	if s.fields[0].Header == nil {
		t.Error("[TestLazyDecodeListBytes]: Header is nil")
		return
	}
	if s.fields[0].Ptr == nil {
		t.Error("[TestLazyDecodeListBytes]: Ptr is nil")
		return
	}

	list, err := GetListBytes(s, 0)
	if err != nil {
		t.Errorf("[TestLazyDecodeListBytes]: GetListBytes error: %v", err)
		return
	}
	if list.Len() != 2 {
		t.Errorf("[TestLazyDecodeListBytes]: Len got %d, want 2", list.Len())
		return
	}
	if string(list.Get(0)) != "hello" {
		t.Errorf("[TestLazyDecodeListBytes]: item[0] got %q, want %q", string(list.Get(0)), "hello")
	}
	if string(list.Get(1)) != "world" {
		t.Errorf("[TestLazyDecodeListBytes]: item[1] got %q, want %q", string(list.Get(1)), "world")
	}
}

func TestLazyDecodeListStructs(t *testing.T) {
	innerMapping := &mapping.Map{
		Fields: []*mapping.FieldDescr{
			{Name: "Bool", Type: field.FTBool},
		},
	}

	outerMapping := &mapping.Map{
		Fields: []*mapping.FieldDescr{
			{Name: "ListStructs", Type: field.FTListStructs, Mapping: innerMapping},
		},
	}

	s1 := New(0, innerMapping)
	s1.XXXSetIsSetEnabled()
	MustSetBool(s1, 0, true)

	s2 := New(0, innerMapping)
	s2.XXXSetIsSetEnabled()
	MustSetBool(s2, 0, false)

	parent := New(0, outerMapping)
	parent.XXXSetIsSetEnabled()
	MustAppendListStruct(parent, 0, s1, s2)

	var buf bytes.Buffer
	_, err := parent.Marshal(&buf)
	if err != nil {
		t.Fatalf("[TestLazyDecodeListStructs]: Marshal error: %v", err)
	}
	fullEncoded := buf.Bytes()

	listData := fullEncoded[8:] // Skip the struct header

	newParent := New(0, outerMapping)
	newParent.XXXSetIsSetEnabled() // Enable IsSet to properly decode IsSet-enabled data
	desc := outerMapping.Fields[0]

	lazyDecodeListStructs(unsafe.Pointer(newParent), 0, listData, desc)

	if newParent.fields[0].Header == nil {
		t.Error("[TestLazyDecodeListStructs]: Header is nil")
		return
	}
	if newParent.fields[0].Ptr == nil {
		t.Error("[TestLazyDecodeListStructs]: Ptr is nil")
		return
	}

	list, err := GetListStruct(newParent, 0)
	if err != nil {
		t.Errorf("[TestLazyDecodeListStructs]: GetListStruct error: %v", err)
		return
	}
	if list.Len() != 2 {
		t.Errorf("[TestLazyDecodeListStructs]: Len got %d, want 2", list.Len())
		return
	}

	item0 := list.Get(0)
	b0, err := GetBool(item0, 0)
	if err != nil {
		t.Errorf("[TestLazyDecodeListStructs]: GetBool item0 error: %v", err)
		return
	}
	if !b0 {
		t.Error("[TestLazyDecodeListStructs]: item[0].Bool expected true, got false")
	}

	item1 := list.Get(1)
	b1, err := GetBool(item1, 0)
	if err != nil {
		t.Errorf("[TestLazyDecodeListStructs]: GetBool item1 error: %v", err)
		return
	}
	if b1 {
		t.Error("[TestLazyDecodeListStructs]: item[1].Bool expected false, got true")
	}
}

func TestLazyDecodeAllNumberTypes(t *testing.T) {
	numberTests := []struct {
		name       string
		ft         field.Type
		decodeFunc func(unsafe.Pointer, uint16, []byte, *mapping.FieldDescr)
	}{
		{name: "ListInt8", ft: field.FTListInt8, decodeFunc: lazyDecodeListInt8},
		{name: "ListInt16", ft: field.FTListInt16, decodeFunc: lazyDecodeListInt16},
		{name: "ListInt32", ft: field.FTListInt32, decodeFunc: lazyDecodeListInt32},
		{name: "ListInt64", ft: field.FTListInt64, decodeFunc: lazyDecodeListInt64},
		{name: "ListUint8", ft: field.FTListUint8, decodeFunc: lazyDecodeListUint8},
		{name: "ListUint16", ft: field.FTListUint16, decodeFunc: lazyDecodeListUint16},
		{name: "ListUint32", ft: field.FTListUint32, decodeFunc: lazyDecodeListUint32},
		{name: "ListUint64", ft: field.FTListUint64, decodeFunc: lazyDecodeListUint64},
		{name: "ListFloat32", ft: field.FTListFloat32, decodeFunc: lazyDecodeListFloat32},
		{name: "ListFloat64", ft: field.FTListFloat64, decodeFunc: lazyDecodeListFloat64},
	}

	for _, test := range numberTests {
		m := &mapping.Map{
			Fields: []*mapping.FieldDescr{
				{Name: "List", Type: test.ft},
			},
		}

		h := NewGenericHeader()
		h.SetFieldNum(0)
		h.SetFieldType(test.ft)
		h.SetFinal40(0) // Empty list

		data := make([]byte, 8)
		copy(data, h)

		s := New(0, m)
		desc := m.Fields[0]

		test.decodeFunc(unsafe.Pointer(s), 0, data, desc)

		if s.fields[0].Header == nil {
			t.Errorf("[TestLazyDecodeAllNumberTypes](%s): Header is nil", test.name)
		}
	}
}
