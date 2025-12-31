package segment

import (
	"bytes"
	"testing"

	"github.com/bearlytools/claw/clawc/languages/go/field"
	"github.com/bearlytools/claw/clawc/languages/go/mapping"
)

// testMapping creates a simple mapping for testing.
func testMapping(numFields int) *mapping.Map {
	fields := make([]*mapping.FieldDescr, numFields)
	for i := 0; i < numFields; i++ {
		fields[i] = &mapping.FieldDescr{
			Name: "field",
			Type: field.FTString,
		}
	}
	return &mapping.Map{
		Fields: fields,
	}
}

func TestSegmentBasicOperations(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "Success: insert and read data"},
	}

	for _, test := range tests {
		seg := NewSegment(64)

		// Initial size should be 8 (header)
		if seg.Len() != 8 {
			t.Errorf("TestSegmentBasicOperations(%s): initial len = %d, want 8", test.name, seg.Len())
		}

		// Append some data
		seg.Append([]byte{1, 2, 3, 4})
		if seg.Len() != 12 {
			t.Errorf("TestSegmentBasicOperations(%s): after append len = %d, want 12", test.name, seg.Len())
		}

		// Insert at middle
		seg.InsertAt(10, []byte{0xAA, 0xBB})
		if seg.Len() != 14 {
			t.Errorf("TestSegmentBasicOperations(%s): after insert len = %d, want 14", test.name, seg.Len())
		}

		// Check data
		expected := []byte{0, 0, 0, 0, 0, 0, 0, 0, 1, 2, 0xAA, 0xBB, 3, 4}
		if !bytes.Equal(seg.data, expected) {
			t.Errorf("TestSegmentBasicOperations(%s): data = %v, want %v", test.name, seg.data, expected)
		}

		// Remove some data
		seg.RemoveAt(10, 2)
		if seg.Len() != 12 {
			t.Errorf("TestSegmentBasicOperations(%s): after remove len = %d, want 12", test.name, seg.Len())
		}
	}
}

func TestHeaderEncodeDecode(t *testing.T) {
	tests := []struct {
		name      string
		fieldNum  uint16
		fieldType field.Type
		final40   uint64
	}{
		{name: "Success: bool field", fieldNum: 1, fieldType: field.FTBool, final40: 1},
		{name: "Success: string field", fieldNum: 5, fieldType: field.FTString, final40: 100},
		{name: "Success: large final40", fieldNum: 100, fieldType: field.FTBytes, final40: 1099511627775}, // max 40-bit value
	}

	for _, test := range tests {
		buf := make([]byte, 8)
		EncodeHeader(buf, test.fieldNum, test.fieldType, test.final40)

		fn, ft, f40 := DecodeHeader(buf)

		if fn != test.fieldNum {
			t.Errorf("TestHeaderEncodeDecode(%s): fieldNum = %d, want %d", test.name, fn, test.fieldNum)
		}
		if ft != test.fieldType {
			t.Errorf("TestHeaderEncodeDecode(%s): fieldType = %d, want %d", test.name, ft, test.fieldType)
		}
		if f40 != test.final40 {
			t.Errorf("TestHeaderEncodeDecode(%s): final40 = %d, want %d", test.name, f40, test.final40)
		}
	}
}

func TestStructScalarFields(t *testing.T) {
	ctx := t.Context()
	m := testMapping(10)
	s := New(ctx, m)

	// Set some scalar fields
	SetInt32(s, 0, 42)
	SetInt32(s, 2, 100)
	SetInt32(s, 1, 50)

	// Verify fields are present
	if !s.HasField(0) {
		t.Error("TestStructScalarFields: field 0 should be present")
	}
	if !s.HasField(1) {
		t.Error("TestStructScalarFields: field 1 should be present")
	}
	if !s.HasField(2) {
		t.Error("TestStructScalarFields: field 2 should be present")
	}
	if s.HasField(3) {
		t.Error("TestStructScalarFields: field 3 should NOT be present")
	}

	// Verify values can be read back
	if GetInt32(s, 0) != 42 {
		t.Errorf("TestStructScalarFields: field 0 = %d, want 42", GetInt32(s, 0))
	}
	if GetInt32(s, 1) != 50 {
		t.Errorf("TestStructScalarFields: field 1 = %d, want 50", GetInt32(s, 1))
	}
	if GetInt32(s, 2) != 100 {
		t.Errorf("TestStructScalarFields: field 2 = %d, want 100", GetInt32(s, 2))
	}

	// Verify field ordering (should be 0, 1, 2)
	offset0, _ := s.FieldOffset(0)
	offset1, _ := s.FieldOffset(1)
	offset2, _ := s.FieldOffset(2)

	if offset0 >= offset1 || offset1 >= offset2 {
		t.Errorf("TestStructScalarFields: fields not in order: offsets = %d, %d, %d", offset0, offset1, offset2)
	}
}

func TestStructStringFields(t *testing.T) {
	ctx := t.Context()
	m := testMapping(5)
	s := New(ctx, m)

	// Set string fields
	SetStringAsBytes(s, 0, "hello")
	SetStringAsBytes(s, 1, "world")

	// Verify fields are present
	if !s.HasField(0) {
		t.Error("TestStructStringFields: field 0 should be present")
	}
	if !s.HasField(1) {
		t.Error("TestStructStringFields: field 1 should be present")
	}

	// Verify values
	if GetString(s, 0) != "hello" {
		t.Errorf("TestStructStringFields: field 0 = %q, want %q", GetString(s, 0), "hello")
	}
	if GetString(s, 1) != "world" {
		t.Errorf("TestStructStringFields: field 1 = %q, want %q", GetString(s, 1), "world")
	}

	// Verify size includes padding (8-byte alignment)
	// "hello" = 5 bytes + 3 padding = 8 bytes + 8 header = 16 bytes
	_, size0 := s.FieldOffset(0)
	if size0 != 16 {
		t.Errorf("TestStructStringFields: field 0 size = %d, want 16", size0)
	}
}

func TestStructMarshal(t *testing.T) {
	ctx := t.Context()
	m := testMapping(5)
	s := New(ctx, m)

	SetInt32(s, 0, 42)
	SetStringAsBytes(s, 1, "test")

	// Marshal to bytes
	data, err := s.Marshal()
	if err != nil {
		t.Fatalf("TestStructMarshal: Marshal failed: %v", err)
	}

	// Verify size is 8-byte aligned
	if len(data)%8 != 0 {
		t.Errorf("TestStructMarshal: marshal size %d is not 8-byte aligned", len(data))
	}

	// Marshal to writer
	var buf bytes.Buffer
	s2 := New(ctx, m)
	SetInt32(s2, 0, 42)
	SetStringAsBytes(s2, 1, "test")

	n, err := s2.MarshalWriter(&buf)
	if err != nil {
		t.Fatalf("TestStructMarshal: Marshal failed: %v", err)
	}

	if n != len(data) {
		t.Errorf("TestStructMarshal: Marshal wrote %d bytes, want %d", n, len(data))
	}
}

func TestSparseEncoding(t *testing.T) {
	ctx := t.Context()
	m := testMapping(10)
	s := New(ctx, m)

	// Set a zero value - should not be stored
	SetInt32(s, 0, 0)
	if s.HasField(0) {
		t.Error("TestSparseEncoding: zero value should not create field")
	}

	// Set a non-zero value
	SetInt32(s, 0, 1)
	if !s.HasField(0) {
		t.Error("TestSparseEncoding: non-zero value should create field")
	}

	// Set back to zero - should remove field
	SetInt32(s, 0, 0)
	if s.HasField(0) {
		t.Error("TestSparseEncoding: zero value should remove field")
	}
}

func TestFieldReplacement(t *testing.T) {
	ctx := t.Context()
	m := testMapping(5)
	s := New(ctx, m)

	// Set initial value
	SetStringAsBytes(s, 0, "short")
	initialSize := s.Size()

	// Replace with same-size value
	SetStringAsBytes(s, 0, "hello") // same length
	if s.Size() != initialSize {
		t.Errorf("TestFieldReplacement: size changed for same-length replace: %d vs %d", s.Size(), initialSize)
	}

	// Replace with longer value
	SetStringAsBytes(s, 0, "this is a longer string")
	if s.Size() <= initialSize {
		t.Error("TestFieldReplacement: size should increase for longer string")
	}
	longerSize := s.Size()

	// Replace with shorter value
	SetStringAsBytes(s, 0, "x")
	if s.Size() >= longerSize {
		t.Error("TestFieldReplacement: size should decrease for shorter string")
	}

	// Verify value is correct
	if GetString(s, 0) != "x" {
		t.Errorf("TestFieldReplacement: value = %q, want %q", GetString(s, 0), "x")
	}
}

func TestNumbersList(t *testing.T) {
	ctx := t.Context()
	m := testMapping(5)
	s := New(ctx, m)

	// Create a list of int32
	list := NewNumbers[int32](s, 0)

	// Append values
	list.Append(1, 2, 3, 4, 5)

	if list.Len() != 5 {
		t.Errorf("TestNumbersList: len = %d, want 5", list.Len())
	}

	// Get values
	for i := 0; i < 5; i++ {
		if list.Get(i) != int32(i+1) {
			t.Errorf("TestNumbersList: Get(%d) = %d, want %d", i, list.Get(i), i+1)
		}
	}

	// Sync to segment
	if err := list.SyncToSegment(); err != nil {
		t.Fatalf("TestNumbersList: SyncToSegment failed: %v", err)
	}

	// Field should now be present
	if !s.HasField(0) {
		t.Error("TestNumbersList: field should be present after sync")
	}
}

func TestStringsList(t *testing.T) {
	ctx := t.Context()
	m := testMapping(5)
	s := New(ctx, m)

	list := NewStrings(s, 0)
	list.Append("hello", "world", "foo")

	if list.Len() != 3 {
		t.Errorf("TestStringsList: len = %d, want 3", list.Len())
	}

	if list.Get(0) != "hello" {
		t.Errorf("TestStringsList: Get(0) = %q, want %q", list.Get(0), "hello")
	}

	if err := list.SyncToSegment(); err != nil {
		t.Fatalf("TestStringsList: SyncToSegment failed: %v", err)
	}

	if !s.HasField(0) {
		t.Error("TestStringsList: field should be present after sync")
	}
}

func TestBoolsList(t *testing.T) {
	ctx := t.Context()
	m := testMapping(5)
	s := New(ctx, m)

	list := NewBools(s, 0)
	list.Append(true, false, true, true, false, false, true, false, true)

	if list.Len() != 9 {
		t.Errorf("TestBoolsList: len = %d, want 9", list.Len())
	}

	expected := []bool{true, false, true, true, false, false, true, false, true}
	for i, want := range expected {
		if list.Get(i) != want {
			t.Errorf("TestBoolsList: Get(%d) = %v, want %v", i, list.Get(i), want)
		}
	}

	// Test Set
	list.Set(1, true)
	if list.Get(1) != true {
		t.Errorf("TestBoolsList: after Set(1, true), Get(1) = %v, want true", list.Get(1))
	}

	// Test Slice
	slice := list.Slice()
	if len(slice) != 9 {
		t.Errorf("TestBoolsList: Slice len = %d, want 9", len(slice))
	}

	// Test iterators
	count := 0
	for range list.All() {
		count++
	}
	if count != 9 {
		t.Errorf("TestBoolsList: All() count = %d, want 9", count)
	}

	// Sync to segment
	if err := list.SyncToSegment(); err != nil {
		t.Fatalf("TestBoolsList: SyncToSegment failed: %v", err)
	}

	if !s.HasField(0) {
		t.Error("TestBoolsList: field should be present after sync")
	}
}

func TestBytesList(t *testing.T) {
	ctx := t.Context()
	m := testMapping(5)
	s := New(ctx, m)

	list := NewBytes(s, 0)
	list.Append([]byte("hello"), []byte("world"), []byte("foo"))

	if list.Len() != 3 {
		t.Errorf("TestBytesList: len = %d, want 3", list.Len())
	}

	if !bytes.Equal(list.Get(0), []byte("hello")) {
		t.Errorf("TestBytesList: Get(0) = %q, want %q", list.Get(0), "hello")
	}

	// Test Set
	list.Set(1, []byte("WORLD"))
	if !bytes.Equal(list.Get(1), []byte("WORLD")) {
		t.Errorf("TestBytesList: after Set, Get(1) = %q, want %q", list.Get(1), "WORLD")
	}

	// Test Slice
	slice := list.Slice()
	if len(slice) != 3 {
		t.Errorf("TestBytesList: Slice len = %d, want 3", len(slice))
	}

	// Test iterators
	count := 0
	for range list.All() {
		count++
	}
	if count != 3 {
		t.Errorf("TestBytesList: All() count = %d, want 3", count)
	}

	// Sync to segment
	if err := list.SyncToSegment(); err != nil {
		t.Fatalf("TestBytesList: SyncToSegment failed: %v", err)
	}

	if !s.HasField(0) {
		t.Error("TestBytesList: field should be present after sync")
	}
}

func TestStructsList(t *testing.T) {
	ctx := t.Context()
	m := testMapping(5)
	s := New(ctx, m)

	innerMapping := testMapping(3)
	list := NewStructs(ctx, s, 0, innerMapping)

	// Create and append items
	item1 := list.NewItem()
	SetInt32(item1, 0, 100)

	item2 := list.NewItem()
	SetInt32(item2, 0, 200)

	list.Append(item1, item2)

	if list.Len() != 2 {
		t.Errorf("TestStructsList: len = %d, want 2", list.Len())
	}

	// Test Get
	got1 := list.Get(0)
	if GetInt32(got1, 0) != 100 {
		t.Errorf("TestStructsList: Get(0).field0 = %d, want 100", GetInt32(got1, 0))
	}

	got2 := list.Get(1)
	if GetInt32(got2, 0) != 200 {
		t.Errorf("TestStructsList: Get(1).field0 = %d, want 200", GetInt32(got2, 0))
	}

	// Test iterators
	count := 0
	for range list.All() {
		count++
	}
	if count != 2 {
		t.Errorf("TestStructsList: All() count = %d, want 2", count)
	}

	// Sync to segment
	if err := list.SyncToSegment(); err != nil {
		t.Fatalf("TestStructsList: SyncToSegment failed: %v", err)
	}

	if !s.HasField(0) {
		t.Error("TestStructsList: field should be present after sync")
	}
}
