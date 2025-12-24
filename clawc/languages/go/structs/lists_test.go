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
	if list.dataSize != 9 {
		t.Fatalf("TestBytes(test initial setup): dataSize was %d, want %d", list.dataSize, 9)
	}
	if list.padding != 7 {
		t.Fatalf("TestBytes(test initial setup): padding was %d, want %d", list.padding, 7)
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
