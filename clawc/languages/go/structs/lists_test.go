package structs

import (
	"fmt"
	"math"
	"testing"

	"github.com/bearlytools/claw/clawc/internal/bits"
	"github.com/bearlytools/claw/clawc/languages/go/conversions"
	"github.com/bearlytools/claw/clawc/languages/go/field"
	"github.com/bearlytools/claw/clawc/languages/go/mapping"
)

func TestBoolGetSetAppendRange(t *testing.T) {
	// Sets our header to message type 15, field number 2 and 20 entries.
	b := []byte{15, 0, 2, 20, 0, 0, 0, 0}
	s := New(0, &mapping.Map{Fields: []*mapping.FieldDescr{{Type: field.FTListBools, FieldNum: 0}}})

	// This sets up the first 20 entries to be set to true, everything else is false.
	b = append(b, bits.Mask[uint8](0, 8), bits.Mask[uint8](0, 8), bits.Mask[uint8](0, 4), 0, 0, 0, 0, 0)
	_, list, err := NewBoolsFromBytes(&b, s)
	if err != nil {
		panic(err)
	}

	values := map[int]bool{}
	for i := 0; i < 20; i++ {
		values[i] = true
	}

	testValues := func(numEntries int) error {
		for i := 0; i < numEntries; i++ {
			if list.Get(i) != values[i] {
				return fmt.Errorf("index(%d) got %v, want %v", i, list.Get(i), values[i])
			}
		}
		return nil
	}

	// Make sure all the first 20 values are true.
	if err := testValues(20); err != nil {
		t.Fatalf("TestBoolGetSetAppendRange(initial data test): %s", err)
	}

	// Let's set index 3 and index 15 values to false.
	list.Set(3, false)
	list.Set(15, false)
	values[3] = false
	values[15] = false

	if err := testValues(20); err != nil {
		t.Fatalf("TestBoolGetSetAppendRange(change initial data test): %s", err)
	}

	// Let's add values that use the rest of the space already allocated.
	sl := []bool{}
	for i := 0; i < 44; i++ {
		sl = append(sl, true)
		values[20+i] = true
	}
	list.Append(sl...)

	if err := testValues(64); err != nil {
		t.Fatalf("TestBoolGetSetAppendRange(fill out 64 values): %s", err)
	}
	if len(list.data) != 16 {
		t.Fatalf("TestBoolGetSetAppendRange(fill out 64 values): Append() extended the data when it was not required")
	}

	// Do an Append() that extends our data.
	list.Append(false)
	values[64] = false
	if err := testValues(65); err != nil {
		t.Fatalf("TestBoolGetSetAppendRange(extend our data): %s", err)
	}
	if len(list.data) != 24 {
		t.Fatalf("TestBoolGetSetAppendRange(extend our data): Append() should have be len == 24, was %d", len(list.data))
	}

	var i int
	for got := range list.Range(1, list.Len()-2) {
		i++
		if got != values[i] {
			t.Fatalf("TestBoolGetSetAppendRange(Range): index %d, got %v, want %v", i, got, values[i])
		}
	}
	if i != list.Len()-3 {
		t.Fatalf("TestBoolGetSetAppendRange(Range): found %d items, want %d items", i, list.Len()-3)
	}

	if s.structTotal.Load() != 32 { // 24 for the Bool, 8 for the Struct header
		t.Fatalf("TestBoolGetSetAppendRange(total count): internal 'total' counter, got %d bytes, want %d bytes", s.structTotal.Load(), 32)
	}
}

func TestNumberGetSetAppendRange(t *testing.T) {
	// Sets our header to message type 16, field number 3 and 7 entries.
	b := []byte{16, 0, 3, 7, 0, 0, 0, 0}

	s := New(0, &mapping.Map{Fields: []*mapping.FieldDescr{{Type: field.FTListInt8, FieldNum: 0}}})

	values := map[int]uint8{
		0: 5,
		1: 10,
		2: 15,
		3: 20,
		4: 25,
		5: 30,
		6: 35,
	}

	// This sets up the first 7 entries.
	for i := 0; i < len(values); i++ {
		b = append(b, values[i])
	}
	b = append(b, 0) // Padding
	list, err := NewNumbersFromBytes[uint8](&b, s)
	if err != nil {
		panic(err)
	}

	testValues := func() error {
		for i := 0; i < len(values); i++ {
			if list.Get(i) != values[i] {
				return fmt.Errorf("index(%d) got %v, want %v", i, list.Get(i), values[i])
			}
		}
		return nil
	}

	// Make sure everythiung is right.
	if err := testValues(); err != nil {
		t.Fatalf("TestNumberGetSetAppendRange(test initial setup): %s", err)
	}

	// Change a value.
	list.Set(3, 80)
	values[3] = 80
	if err := testValues(); err != nil {
		t.Fatalf("TestNumberGetSetAppendRange(test set value): %s", err)
	}

	// Append a single value, which should fit in existing space.
	list.Append(45)
	values[len(values)] = 45
	if err := testValues(); err != nil {
		t.Fatalf("TestNumberGetSetAppendRange(test append within size): %s", err)
	}
	if len(list.data) != 16 {
		t.Fatalf("TestNumberGetSetAppendRange(test append within size): expected buffer size incorrect, got %d, want %d", len(list.data), 16)
	}

	// Append several values which requires new space.
	toAppend := []uint8{50, 5, 60, 65, 70}
	for _, v := range toAppend {
		values[len(values)] = v
	}

	list.Append(toAppend...)

	if err := testValues(); err != nil {
		t.Fatalf("TestNumberGetSetAppendRange(test append without enough space): %s", err)
	}

	var i int
	for got := range list.Range(1, list.Len()-2) {
		i++
		if got != values[i] {
			t.Fatalf("TestNumberGetSetAppendRange(Range): index %d, got %d, want %d", i, got, values[i])
		}
	}
	if i != list.Len()-3 {
		t.Fatalf("TestNumberGetSetAppendRange(Range): found %d items, want %d items", i, list.Len()-3)
	}

	if s.structTotal.Load() != 32 { // 24 for the Number, 8 for the Struct header
		t.Fatalf("TestNumberGetSetAppendRange(total count): internal 'total' counter, got %d bytes, want %d bytes", s.structTotal.Load(), 32)
	}
}

func TestNumberFloat(t *testing.T) {
	tests := []struct {
		desc                 string
		h                    GenericHeader
		values, appendValues []float64
		err                  bool
	}{
		{
			h: func() GenericHeader {
				h := NewGenericHeader()
				h.SetFieldNum(3)
				h.SetFieldType(field.FTFloat64)
				return h
			}(),
			values:       []float64{3.2, 2.8, 5.2, 0},
			appendValues: []float64{18.2},
		},
	}

	for _, test := range tests {
		test.h.SetFinal40(uint64(len(test.values)))
		for _, v := range test.values {
			u := math.Float64bits(v)
			b := conversions.NumToBytes(&u)
			test.h = append(test.h, b...)
		}

		s := New(0, &mapping.Map{Fields: []*mapping.FieldDescr{{Type: field.FTFloat64, FieldNum: 0}}})
		b := []byte(test.h)
		list, err := NewNumbersFromBytes[float64](&b, s)
		switch {
		case err == nil && test.err:
			t.Errorf("TestNumberFloat(%s): got err == nil, want err != nil", test.desc)
			continue
		case err != nil && !test.err:
			t.Errorf("TestNumberFloat(%s): got err == %s, want err == nil", test.desc, err)
			continue
		case err != nil:
			continue
		}
		list.Append(test.appendValues...)

		i := 0
		want := append(test.values, test.appendValues...)
		for got := range list.Range(0, list.Len()) {
			if got != want[i] {
				t.Fatalf("TestNumberFloat: index %d, got %v, want %v", i, got, want[i])
			}
			i++
		}
	}
}

func TestBytes(t *testing.T) {
	// Sets our header to message type 20, field number 5 and 1 entry.
	h := NewGenericHeader()
	h.SetFieldType(field.FTBytes)
	h.SetFieldNum(5)

	s := New(0, &mapping.Map{Fields: []*mapping.FieldDescr{{Type: field.FTBytes, FieldNum: 0}}})

	values := []string{
		"hello", // len 5
	}

	// This sets up the first entry.
	for i := 0; i < len(values); i++ {
		h = append(h, []byte{5, 0, 0, 0}...) // 32 bit entry header
		h = append(h, []byte(values[i])...)
	} // 17 bytes now - list header(8) + entry header(4) + data(5)
	h = append(h, 0, 0, 0, 0, 0, 0, 0) // 7 bytes of Padding
	h.SetFinal40(1)                    // number of items
	b := []byte(h)
	if len(b) != 24 {
		t.Fatalf("TestBytes(test initial setup): message to read in was %d bytes, expected %d bytes", len(b), 24)
	}

	list, err := NewBytesFromBytes(&b, s)
	if err != nil {
		panic(err)
	}

	testValues := func() error {
		for i := 0; i < len(values); i++ {
			if string(list.Get(i)) != values[i] {
				return fmt.Errorf("index(%d) got %v, want %v", i, string(list.Get(i)), values[i])
			}
		}
		return nil
	}

	// Make sure everything is right.
	if err := testValues(); err != nil {
		t.Fatalf("TestBytes(test initial setup): %s", err)
	}
	if list.dataSize.Load() != 9 {
		t.Fatalf("TestBytes(test initial setup): dataSize was %d, want %d", list.dataSize.Load(), 9)
	}
	if list.padding.Load() != 7 {
		t.Fatalf("TestBytes(test initial setup): padding was %d, want %d", list.padding.Load(), 7)
	}

	// Append a few values.
	values = append(values, "I", "must", "be", "going")
	for _, v := range values[1:] {
		list.Append([]byte(v))
	} // At this point, our list is 45 bytes without padding, which rounds to 48

	if err := testValues(); err != nil {
		t.Fatalf("TestBytes(test after Append): %s", err)
	}

	want := values[1 : len(values)-1]
	i := 0
	for got := range list.Range(1, list.Len()-1) {
		if string(got) != want[i] {
			t.Fatalf("TestBytes(Range): index %d, got %s, want %s", i, got, want[i])
		}
		i++
	}
	if i != len(want) {
		t.Fatalf("TestBytes(Range): only found %d entries, want %d entries", i, len(want))
	}

	size := 0
	for _, v := range list.data {
		size += len(v)
	}

	if s.structTotal.Load() != 56 { // 48 for the Bytes, 8 for the Struct header
		t.Fatalf("TestBytes(total count): internal 'total' counter, got %d bytes, want %d bytes", s.structTotal.Load(), 56)
	}
}

func TestBytesSlice(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		values []string
	}{
		{
			name:   "Success: empty Bytes returns nil",
			values: nil,
		},
		{
			name:   "Success: single entry",
			values: []string{"hello"},
		},
		{
			name:   "Success: multiple entries",
			values: []string{"hello", "world", "test"},
		},
		{
			name:   "Success: entries with varying lengths",
			values: []string{"a", "ab", "abc", "abcd", "abcde"},
		},
		{
			name:   "Success: entries with empty string",
			values: []string{"hello", "", "world"},
		},
		{
			name:   "Success: large entry",
			values: []string{"this is a much longer string that spans many bytes"},
		},
	}

	for _, test := range tests {
		s := New(0, &mapping.Map{Fields: []*mapping.FieldDescr{{Type: field.FTListBytes, FieldNum: 0}}})
		b := NewBytes()

		for _, v := range test.values {
			b.Append([]byte(v))
		}

		got := b.Slice()

		if test.values == nil {
			if got != nil {
				t.Errorf("[TestBytesSlice] %s: got non-nil slice for empty Bytes", test.name)
			}
			continue
		}

		if len(got) != len(test.values) {
			t.Errorf("[TestBytesSlice] %s: len(got) = %d, want %d", test.name, len(got), len(test.values))
			continue
		}

		for i, want := range test.values {
			if string(got[i]) != want {
				t.Errorf("[TestBytesSlice] %s: got[%d] = %q, want %q", test.name, i, string(got[i]), want)
			}
		}

		// Suppress unused variable warning
		_ = s
	}
}

func TestBytesSliceIndependence(t *testing.T) {
	t.Parallel()

	// Test that modifying the returned slice doesn't affect the original Bytes
	b := NewBytes()
	b.Append([]byte("hello"))
	b.Append([]byte("world"))

	slice := b.Slice()

	// Modify the slice
	slice[0][0] = 'X'

	// Original should be unchanged
	if string(b.Get(0)) != "hello" {
		t.Errorf("[TestBytesSliceIndependence]: modifying slice affected original, got %q, want %q", string(b.Get(0)), "hello")
	}
}

func TestNumbersPoolReuse(t *testing.T) {
	t.Parallel()

	// Create multiple Numbers instances to exercise pool
	for i := 0; i < 10; i++ {
		n := NewNumbers[int32]()
		n.Append(int32(i))
		n.Append(int32(i * 2))

		if n.Len() != 2 {
			t.Errorf("[TestNumbersPoolReuse] iteration %d: Len() = %d, want 2", i, n.Len())
		}
		if n.Get(0) != int32(i) {
			t.Errorf("[TestNumbersPoolReuse] iteration %d: Get(0) = %d, want %d", i, n.Get(0), i)
		}
		if n.Get(1) != int32(i*2) {
			t.Errorf("[TestNumbersPoolReuse] iteration %d: Get(1) = %d, want %d", i, n.Get(1), i*2)
		}
	}
}

func TestNumbersPoolAllTypes(t *testing.T) {
	t.Parallel()

	// Test that pool works for all number types
	tests := []struct {
		name string
		fn   func()
	}{
		{"int8", func() {
			n := NewNumbers[int8]()
			n.Append(1)
			if n.Get(0) != 1 {
				t.Errorf("[TestNumbersPoolAllTypes] int8: got %d, want 1", n.Get(0))
			}
		}},
		{"int16", func() {
			n := NewNumbers[int16]()
			n.Append(1000)
			if n.Get(0) != 1000 {
				t.Errorf("[TestNumbersPoolAllTypes] int16: got %d, want 1000", n.Get(0))
			}
		}},
		{"int32", func() {
			n := NewNumbers[int32]()
			n.Append(100000)
			if n.Get(0) != 100000 {
				t.Errorf("[TestNumbersPoolAllTypes] int32: got %d, want 100000", n.Get(0))
			}
		}},
		{"int64", func() {
			n := NewNumbers[int64]()
			n.Append(10000000000)
			if n.Get(0) != 10000000000 {
				t.Errorf("[TestNumbersPoolAllTypes] int64: got %d, want 10000000000", n.Get(0))
			}
		}},
		{"uint8", func() {
			n := NewNumbers[uint8]()
			n.Append(255)
			if n.Get(0) != 255 {
				t.Errorf("[TestNumbersPoolAllTypes] uint8: got %d, want 255", n.Get(0))
			}
		}},
		{"uint16", func() {
			n := NewNumbers[uint16]()
			n.Append(65535)
			if n.Get(0) != 65535 {
				t.Errorf("[TestNumbersPoolAllTypes] uint16: got %d, want 65535", n.Get(0))
			}
		}},
		{"uint32", func() {
			n := NewNumbers[uint32]()
			n.Append(4294967295)
			if n.Get(0) != 4294967295 {
				t.Errorf("[TestNumbersPoolAllTypes] uint32: got %d, want 4294967295", n.Get(0))
			}
		}},
		{"uint64", func() {
			n := NewNumbers[uint64]()
			n.Append(18446744073709551615)
			if n.Get(0) != 18446744073709551615 {
				t.Errorf("[TestNumbersPoolAllTypes] uint64: got %d, want 18446744073709551615", n.Get(0))
			}
		}},
		{"float32", func() {
			n := NewNumbers[float32]()
			n.Append(3.14)
			if n.Get(0) != 3.14 {
				t.Errorf("[TestNumbersPoolAllTypes] float32: got %f, want 3.14", n.Get(0))
			}
		}},
		{"float64", func() {
			n := NewNumbers[float64]()
			n.Append(3.14159265359)
			if n.Get(0) != 3.14159265359 {
				t.Errorf("[TestNumbersPoolAllTypes] float64: got %f, want 3.14159265359", n.Get(0))
			}
		}},
	}

	for _, test := range tests {
		test.fn()
	}
}

func TestStructsListSizeTracking(t *testing.T) {
	t.Parallel()

	testMapping := &mapping.Map{
		Name: "Inner",
		Fields: []*mapping.FieldDescr{
			{Name: "Value", Type: field.FTInt32, FieldNum: 0},
		},
	}
	testMapping.Init()

	outerMapping := &mapping.Map{
		Name: "Outer",
		Fields: []*mapping.FieldDescr{
			{Name: "List", Type: field.FTListStructs, FieldNum: 0, Mapping: testMapping},
		},
	}
	outerMapping.Init()

	// Create a Structs list and add items
	list := NewStructs(testMapping)

	// Add first struct
	s1 := New(0, testMapping)
	MustSetNumber(s1, 0, int32(100))
	list.Append(s1)

	size1 := list.size.Load()
	if size1 == 0 {
		t.Errorf("[TestStructsListSizeTracking]: size should be > 0 after first Append")
	}

	// Add second struct
	s2 := New(0, testMapping)
	MustSetNumber(s2, 0, int32(200))
	list.Append(s2)

	size2 := list.size.Load()
	if size2 <= size1 {
		t.Errorf("[TestStructsListSizeTracking]: size should increase after second Append, got %d <= %d", size2, size1)
	}

	// Verify we can read back the values
	if list.Len() != 2 {
		t.Errorf("[TestStructsListSizeTracking]: Len() = %d, want 2", list.Len())
	}

	got1, _ := GetNumber[int32](list.Get(0), 0)
	if got1 != 100 {
		t.Errorf("[TestStructsListSizeTracking]: first struct value = %d, want 100", got1)
	}

	got2, _ := GetNumber[int32](list.Get(1), 0)
	if got2 != 200 {
		t.Errorf("[TestStructsListSizeTracking]: second struct value = %d, want 200", got2)
	}
}
