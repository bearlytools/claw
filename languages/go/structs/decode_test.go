package structs

import (
	"bytes"
	"fmt"
	"log"
	"math"
	"reflect"
	"testing"

	"github.com/bearlytools/claw/internal/binary"
	"github.com/bearlytools/claw/internal/bits"
	"github.com/bearlytools/claw/internal/conversions"
	"github.com/bearlytools/claw/languages/go/field"
	"github.com/bearlytools/claw/languages/go/mapping"
)

func TestDecodeBool(t *testing.T) {
	m := &mapping.Map{
		Fields: []*mapping.FieldDescr{
			{
				Name: "bool",
				Type: field.FTBool,
			},
		},
	}

	// Sets up "h" to be a bool with the value of true.
	h := NewGenericHeader()
	h.SetFieldType(field.FTBool)
	n := conversions.BytesToNum[uint64](h)
	*n = bits.SetBit(*n, 24, true)

	tests := []struct {
		desc     string
		buf      []byte
		fieldNum uint16
		want     bool
		err      bool
	}{
		{
			desc:     "Error: len(buffer) is < 8",
			buf:      []byte{0, 0, 0, 0, 0, 0, 0}, // 7 in length
			fieldNum: 0,
			err:      true,
		},
		{
			desc:     "Success: set to true",
			buf:      h,
			fieldNum: 0,
			want:     true,
		},
	}

	for _, test := range tests {
		wantHeader := make([]byte, len(test.buf))
		copy(wantHeader, test.buf)

		s := New(0, m)
		err := s.decodeBool(&test.buf, test.fieldNum)
		switch {
		case err == nil && test.err:
			t.Errorf("TestDecodeBool(%s): got err == nil, want err != nil", test.desc)
			continue
		case err != nil && !test.err:
			t.Errorf("TestDecodeBool(%s): got err == %s, want err == nil", test.desc, err)
			continue
		case err != nil:
			continue
		}
		got, err := GetBool(s, test.fieldNum)
		if err != nil {
			panic(err)
		}
		if got != test.want {
			t.Errorf("TestDecodeBool(%s): got %v, want %v", test.desc, got, test.want)
		}
		if *s.structTotal != 16 {
			t.Errorf("TestDecodeBool(%s): structTotal: got %v, want %v", test.desc, s.structTotal, 16)
		}
		f := s.fields[test.fieldNum]
		if !bytes.Equal(f.Header, wantHeader) {
			t.Errorf("TestDecodeBool(%s): field.header value: got %v, want %v", test.desc, f.Header, wantHeader)
		}
		if len(test.buf) != 0 {
			t.Errorf("TestDecodeBool(%s): did not advance the buffer correctly", test.desc)
		}
	}
}

func numFieldInBytes[N Number](value N, dataMap *mapping.Map) []byte {
	s := New(0, dataMap)
	if err := SetNumber(s, 0, value); err != nil {
		panic(err)
	}
	var b []byte
	if s.fields[0].Ptr == nil {
		b = make([]byte, len(s.fields[0].Header))
		copy(b, s.fields[0].Header)
	} else {
		ptr := (*[]byte)(s.fields[0].Ptr)
		b = make([]byte, len(s.fields[0].Header)+len(*ptr))
		copy(b, s.fields[0].Header)
		copy(b[8:], *ptr)
	}
	return b
}

func TestDecodeNum(t *testing.T) {
	mappings := make([]*mapping.Map, len(field.NumberTypes))
	encoded := make([][]byte, len(mappings))
	want := make([]any, len(mappings))
	size := make([]int8, len(mappings))

	for i, ft := range field.NumberTypes {
		mappings[i] = &mapping.Map{
			Fields: []*mapping.FieldDescr{
				{Type: ft},
			},
		}
		switch ft {
		case field.FTUint8:
			want[i] = uint8(math.MaxUint8)
			encoded[i] = numFieldInBytes[uint8](math.MaxUint8, mappings[i])
			size[i] = 8
		case field.FTUint16:
			want[i] = uint16(math.MaxUint16)
			encoded[i] = numFieldInBytes[uint16](math.MaxUint16, mappings[i])
			size[i] = 16
		case field.FTUint32:
			want[i] = uint32(math.MaxUint32)
			encoded[i] = numFieldInBytes[uint32](math.MaxUint32, mappings[i])
			size[i] = 32
		case field.FTUint64:
			want[i] = uint64(math.MaxUint64)
			encoded[i] = numFieldInBytes[uint64](math.MaxUint64, mappings[i])
			size[i] = 64
		case field.FTInt8:
			want[i] = int8(math.MaxInt8)
			encoded[i] = numFieldInBytes[int8](math.MaxInt8, mappings[i])
			size[i] = 8
		case field.FTInt16:
			want[i] = int16(math.MinInt16)
			encoded[i] = numFieldInBytes[int16](math.MinInt16, mappings[i]) // Tests negative number
			size[i] = 16
		case field.FTInt32:
			want[i] = int32(math.MaxInt32)
			encoded[i] = numFieldInBytes[int32](math.MaxInt32, mappings[i])
			size[i] = 32
		case field.FTInt64:
			want[i] = int64(math.MaxInt64)
			encoded[i] = numFieldInBytes[int64](math.MaxInt64, mappings[i])
			size[i] = 64
		case field.FTFloat32:
			want[i] = float32(math.MaxFloat32)
			encoded[i] = numFieldInBytes[float32](math.MaxFloat32, mappings[i])
			size[i] = 32
		case field.FTFloat64:
			want[i] = float64(math.SmallestNonzeroFloat64)
			encoded[i] = numFieldInBytes(math.SmallestNonzeroFloat64, mappings[i])
			size[i] = 64
		}
	}

	for i, mapping := range mappings {
		s := New(0, mapping)
		err := s.decodeNum(&encoded[i], 0, size[i])
		if err != nil {
			t.Errorf("TestDecodeNum: could not decode type %v: %s", mapping.Fields[0].Type, err)
			continue
		}
		switch mapping.Fields[0].Type {
		case field.FTUint8:
			got, err := GetNumber[uint8](s, 0)
			if err != nil {
				panic(err)
			}
			if !reflect.DeepEqual(got, want[i]) {
				t.Errorf("TestDecodeNum(uint8): got %d, want %d", got, want[i])
			}
			if *s.structTotal != 16 {
				t.Errorf("TestDecodeNum(uint8): structTotal: got %d, want %d", *s.structTotal, 16)
			}
			if s.fields[0].Header.FieldType() != field.FTUint8 {
				t.Errorf("TestDecodeNum(uint8): fieldNum: got %v", field.Type(s.fields[0].Header.FieldType()))
			}
			if s.fields[0].Header.FieldNum() != 0 {
				t.Errorf("TestDecodeNum(uint8): fieldNum: got %d, want %d", s.fields[0].Header.FieldNum(), 0)
			}
		case field.FTUint16:
			got, err := GetNumber[uint16](s, 0)
			if err != nil {
				panic(err)
			}
			if !reflect.DeepEqual(got, want[i]) {
				t.Errorf("TestDecodeNum(uint16): got %d, want %d", got, want[i])
			}
			if *s.structTotal != 16 {
				t.Errorf("TestDecodeNum(uint16): structTotal: got %d, want %d", *s.structTotal, 16)
			}
			if s.fields[0].Header.FieldType() != field.FTUint16 {
				t.Errorf("TestDecodeNum(uint16): fieldNum: got %v", field.Type(s.fields[0].Header.FieldType()))
			}
			if s.fields[0].Header.FieldNum() != 0 {
				t.Errorf("TestDecodeNum(uint16): fieldNum: got %d, want %d", s.fields[0].Header.FieldNum(), 0)
			}
		case field.FTUint32:
			got, err := GetNumber[uint32](s, 0)
			if err != nil {
				panic(err)
			}
			if !reflect.DeepEqual(got, want[i]) {
				t.Errorf("TestDecodeNum(uint32): got %d, want %d", got, want[i])
			}
			if *s.structTotal != 16 {
				t.Errorf("TestDecodeNum(uint32): structTotal: got %d, want %d", *s.structTotal, 16)
			}
			if s.fields[0].Header.FieldType() != field.FTUint32 {
				t.Errorf("TestDecodeNum(uint32): fieldNum: got %v", field.Type(s.fields[0].Header.FieldType()))
			}
			if s.fields[0].Header.FieldNum() != 0 {
				t.Errorf("TestDecodeNum(uint32): fieldNum: got %d, want %d", s.fields[0].Header.FieldNum(), 0)
			}
		case field.FTUint64:
			got, err := GetNumber[uint64](s, 0)
			if err != nil {
				panic(err)
			}
			if !reflect.DeepEqual(got, want[i]) {
				t.Errorf("TestDecodeNum(uint64): got %d, want %d", got, want[i])
			}
			if *s.structTotal != 24 {
				t.Errorf("TestDecodeNum(uint64): structTotal: got %d, want %d", *s.structTotal, 24)
			}
			if s.fields[0].Header.FieldType() != field.FTUint64 {
				t.Errorf("TestDecodeNum(uint64): fieldNum: got %v", field.Type(s.fields[0].Header.FieldType()))
			}
			if s.fields[0].Header.FieldNum() != 0 {
				t.Errorf("TestDecodeNum(uint64): fieldNum: got %d, want %d", s.fields[0].Header.FieldNum(), 0)
			}
		case field.FTInt8:
			got, err := GetNumber[int8](s, 0)
			if err != nil {
				panic(err)
			}
			if !reflect.DeepEqual(got, want[i]) {
				t.Errorf("TestDecodeNum(int8): got %d, want %d", got, want[i])
			}
			if *s.structTotal != 16 {
				t.Errorf("TestDecodeNum(int8): structTotal: got %d, want %d", *s.structTotal, 16)
			}
			if s.fields[0].Header.FieldType() != field.FTInt8 {
				t.Errorf("TestDecodeNum(int8): fieldNum: got %v", field.Type(s.fields[0].Header.FieldType()))
			}
			if s.fields[0].Header.FieldNum() != 0 {
				t.Errorf("TestDecodeNum(int8): fieldNum: got %d, want %d", s.fields[0].Header.FieldNum(), 0)
			}
		case field.FTInt16:
			got, err := GetNumber[int16](s, 0)
			if err != nil {
				panic(err)
			}
			if !reflect.DeepEqual(got, want[i]) {
				t.Errorf("TestDecodeNum(int16): got %d, want %d", got, want[i])
			}
			if *s.structTotal != 16 {
				t.Errorf("TestDecodeNum(int16): structTotal: got %d, want %d", *s.structTotal, 16)
			}
			if s.fields[0].Header.FieldType() != field.FTInt16 {
				t.Errorf("TestDecodeNum(int16): fieldNum: got %v", field.Type(s.fields[0].Header.FieldType()))
			}
			if s.fields[0].Header.FieldNum() != 0 {
				t.Errorf("TestDecodeNum(int16): fieldNum: got %d, want %d", s.fields[0].Header.FieldNum(), 0)
			}
		case field.FTInt32:
			got, err := GetNumber[int32](s, 0)
			if err != nil {
				panic(err)
			}
			if !reflect.DeepEqual(got, want[i]) {
				t.Errorf("TestDecodeNum(int32): got %d, want %d", got, want[i])
			}
			if *s.structTotal != 16 {
				t.Errorf("TestDecodeNum(int32): structTotal: got %d, want %d", *s.structTotal, 16)
			}
			if s.fields[0].Header.FieldType() != field.FTInt32 {
				t.Errorf("TestDecodeNum(int32): fieldNum: got %v", field.Type(s.fields[0].Header.FieldType()))
			}
			if s.fields[0].Header.FieldNum() != 0 {
				t.Errorf("TestDecodeNum(int32): fieldNum: got %d, want %d", s.fields[0].Header.FieldNum(), 0)
			}
		case field.FTInt64:
			got, err := GetNumber[int64](s, 0)
			if err != nil {
				panic(err)
			}
			if !reflect.DeepEqual(got, want[i]) {
				t.Errorf("TestDecodeNum(int64): got %d, want %d", got, want[i])
			}
			if *s.structTotal != 24 {
				t.Errorf("TestDecodeNum(int64): structTotal: got %d, want %d", *s.structTotal, 24)
			}
			if s.fields[0].Header.FieldType() != field.FTInt64 {
				t.Errorf("TestDecodeNum(int64): fieldNum: got %v", field.Type(s.fields[0].Header.FieldType()))
			}
			if s.fields[0].Header.FieldNum() != 0 {
				t.Errorf("TestDecodeNum(int64): fieldNum: got %d, want %d", s.fields[0].Header.FieldNum(), 0)
			}
		case field.FTFloat32:
			got, err := GetNumber[float32](s, 0)
			if err != nil {
				panic(err)
			}
			if !reflect.DeepEqual(got, want[i]) {
				t.Errorf("TestDecodeNum(float32): got %v, want %v", got, want[i])
			}
			if *s.structTotal != 16 {
				t.Errorf("TestDecodeNum(float32): structTotal: got %d, want %d", *s.structTotal, 16)
			}
			if s.fields[0].Header.FieldType() != field.FTFloat32 {
				t.Errorf("TestDecodeNum(float32): fieldNum: got %v", field.Type(s.fields[0].Header.FieldType()))
			}
			if s.fields[0].Header.FieldNum() != 0 {
				t.Errorf("TestDecodeNum(float32): fieldNum: got %d, want %d", s.fields[0].Header.FieldNum(), 0)
			}
		case field.FTFloat64:
			got, err := GetNumber[float64](s, 0)
			if err != nil {
				panic(err)
			}
			if !reflect.DeepEqual(got, want[i]) {
				t.Errorf("TestDecodeNum(float64): got %v, want %v", got, want[i])
			}
			if *s.structTotal != 24 {
				t.Errorf("TestDecodeNum(float64): structTotal: got %d, want %d", *s.structTotal, 24)
			}
			if s.fields[0].Header.FieldType() != field.FTFloat64 {
				t.Errorf("TestDecodeNum(float64): fieldNum: got %v", field.Type(s.fields[0].Header.FieldType()))
			}
			if s.fields[0].Header.FieldNum() != 0 {
				t.Errorf("TestDecodeNum(float64): fieldNum: got %d, want %d", s.fields[0].Header.FieldNum(), 0)
			}
		}
		if len(encoded[i]) != 0 {
			t.Errorf("TestDecodeNum(%v): did not advance the buffer correctly", mapping.Fields[0].Type)
		}
	}

	errTests := []struct {
		desc     string
		buf      []byte
		fieldNum uint16
		size     int8
	}{
		{
			desc:     "Error: size 8 with len(buffer) is < 8",
			buf:      make([]byte, 7),
			fieldNum: 1,
			size:     8,
		},
		{
			desc:     "Error: size 16 with len(buffer) is < 8",
			buf:      make([]byte, 7),
			fieldNum: 1,
			size:     16,
		},
		{
			desc:     "Error: size 32 with len(buffer) is < 8",
			buf:      make([]byte, 7),
			fieldNum: 1,
			size:     32,
		},
		{
			desc:     "Error: size 64 with len(buffer) is < 16",
			buf:      make([]byte, 15),
			fieldNum: 1,
			size:     64,
		},
		{
			desc:     "Error: size is not 8, 16, 32, 64",
			buf:      make([]byte, 16),
			fieldNum: 1,
			size:     1,
		},
	}

	for _, test := range errTests {
		s := New(0, mappings[0])
		err := s.decodeNum(&test.buf, test.fieldNum, test.size)

		if err == nil {
			t.Errorf("TestDecodeNum(%s): got err == nil, want err != nil", test.desc)
			continue
		}
	}
}

func bytesFieldInBytes(value []byte, dataMap *mapping.Map) []byte {
	s := New(0, dataMap)
	if err := SetBytes(s, 0, value, false); err != nil {
		panic(err)
	}
	var b []byte
	ptr := (*[]byte)(s.fields[0].Ptr)
	dataSize := len(*ptr)
	b = make([]byte, 8+dataSize+PaddingNeeded(dataSize))
	copy(b, s.fields[0].Header)
	copy(b[8:], *ptr)
	return b
}

func TestDecodeBytes(t *testing.T) {
	m := &mapping.Map{
		Fields: []*mapping.FieldDescr{
			{
				Name: "bytes",
				Type: field.FTBytes,
			},
		},
	}

	sizeZeroHeader := NewGenericHeader()
	sizeZeroHeader.SetFieldNum(0)
	sizeZeroHeader.SetFieldType(field.FTBytes)

	tests := []struct {
		desc     string
		buf      []byte
		fieldNum uint16
		want     []byte
		err      bool
	}{
		{
			desc:     "Error: Not enough buffer for header",
			buf:      make([]byte, 7),
			fieldNum: 1,
			err:      true,
		},
		{
			desc:     "Error: Struct size is 0",
			buf:      sizeZeroHeader,
			fieldNum: 1,
			err:      true,
		},
		{
			desc:     "Error: Not enough padding",
			buf:      bytesFieldInBytes([]byte("1234567"), m)[0:7], // Remove 1 from the padding
			fieldNum: 1,
			err:      true,
		},
		{
			desc:     "Encode with padding",
			fieldNum: 1,
			want:     []byte("1234567"),
		},
	}

	for _, test := range tests {
		if test.buf == nil {
			test.buf = bytesFieldInBytes(test.want, m)
		}
		s := New(0, m)

		err := s.decodeBytes(&test.buf, 0)
		switch {
		case err == nil && test.err:
			t.Errorf("TestDecodeBytes(%s): got err == nil, want err != nil", test.desc)
			continue
		case err != nil && !test.err:
			t.Errorf("TestDecodeBytes(%s): got err == %s, want err == nil", test.desc, err)
			continue
		case err != nil:
			continue
		}

		got, err := GetBytes(s, 0)
		if err != nil {
			t.Errorf("TestDecodeBytes(%s): unexpected error: %s", test.desc, err)
			continue
		}
		if !bytes.Equal(*got, test.want) {
			t.Errorf("TestDecodeBytes(%s): got %q, want %q", test.desc, string(*got), string(test.want))
			continue
		}
		totalWant := int64(16 + SizeWithPadding(len(test.want))) // structSize(8) + header(8) + datasize(7) + padding(1)
		if *s.structTotal != totalWant {
			t.Errorf("TestDecodeBytes(%s): structTotal: got %d, want %d", test.desc, *s.structTotal, totalWant)
			continue
		}
		if s.fields[0].Header.FieldType() != field.FTBytes {
			t.Errorf("TestDecodeBytes(%s): fieldNum: got %v", test.desc, field.Type(s.fields[0].Header.FieldType()))
		}
		if s.fields[0].Header.FieldNum() != 0 {
			t.Errorf("TestDecodeBytes(%s): fieldNum: got %d, want %d", test.desc, s.fields[0].Header.FieldNum(), 0)
		}
	}
}

func boolListInBytes(howMany uint64) []byte {
	h := NewGenericHeader()
	h.SetFieldNum(0)
	h.SetFieldType(field.FTListBools)
	h.SetFinal40(howMany)

	wordsNeeded := (howMany / 64) + 1
	d := make([]byte, 8*wordsNeeded)
	val := false
	for i := 0; i < int(howMany); i++ {
		if i%2 == 0 {
			val = true
		} else {
			val = false
		}

		where := i / 8 // What byte do we find our bool bit in
		n := conversions.BytesToNum[uint64](d[where : where+1])
		*n = bits.SetBit(*n, uint8(i%8), val) // i%8 because we are chaning a single bit in a byte
	}
	return append(h, d...)
}

func TestDecodeListBool(t *testing.T) {
	m := &mapping.Map{
		Fields: []*mapping.FieldDescr{
			{
				Name: "listBool",
				Type: field.FTListBools,
			},
		},
	}

	tests := []struct {
		desc     string
		listData []byte
		want     int
		err      bool
	}{
		{
			desc:     "Error: < 8 bytes",
			listData: make([]byte, 7),
			err:      true,
		},
		{
			desc: "Error: header has wrong type",
			listData: func() []byte {
				h := NewGenericHeader()
				h.SetFieldNum(0)
				h.SetFieldType(field.FTBool)
				h.SetFinal40(1)
				return h
			}(),
			err: true,
		},
		{
			desc:     "Error: Header but no data",
			listData: boolListInBytes(0),
			err:      true,
		},
		{
			desc:     "Success",
			want:     65,
			listData: boolListInBytes(65),
		},
	}

	for _, test := range tests {
		s := New(0, m)
		dataSize := len(test.listData) // Must record before we change the []byte slice

		err := s.decodeListBool(&test.listData, 0)
		switch {
		case err == nil && test.err:
			t.Errorf("TestDecodeListBool(%s): got err == nil, want err != nil", test.desc)
			continue
		case err != nil && !test.err:
			t.Errorf("TestDecodeListBool(%s): got err == %s, want err == nil", test.desc, err)
			continue
		case err != nil:
			continue
		}

		lb, err := GetListBool(s, 0)
		if err != nil {
			panic(err)
		}
		if lb.Len() != test.want {
			t.Errorf("TestDecodeListBool(%s): Len(): got %d, want %d", test.desc, lb.Len(), test.want)
			continue
		}

		for i := 0; i < test.want; i++ {
			got := lb.Get(i)
			want := false
			if i%2 == 0 {
				want = true
			}
			if got != want {
				t.Errorf("TestDecodeListBool(%s): entry[%d]: got %v, want %v", test.desc, i, got, want)
				continue
			}
		}

		if int(*s.structTotal) != 8+dataSize { // structHeader(8) + listHeaderAndData(dataSize)
			t.Errorf("TestDecodeListBool(%s): structTotal: got %d, want %d)", test.desc, *s.structTotal, 8+dataSize)
		}
		if len(test.listData) > 0 {
			t.Errorf("TestDecodeListBool(%s): after decode []byte buffer had len %d, but expected 0", test.desc, len(test.listData))
		}
	}
}

func TestDecodeListBytes(t *testing.T) {
	m := &mapping.Map{
		Fields: []*mapping.FieldDescr{
			{
				Name: "listBytes",
				Type: field.FTListBytes,
			},
		},
	}

	entryData := [][]byte{[]byte("make the madness stop"), []byte("because I'm tired")}

	tests := []struct {
		desc     string
		listData []byte
		want     [][]byte
		err      bool
	}{
		{
			desc:     "Error: < 8 bytes",
			listData: make([]byte, 7),
			err:      true,
		},
		{
			desc: "Error: header has wrong type",
			listData: func() []byte {
				h := NewGenericHeader()
				h.SetFieldType(field.FTBytes)
				h.SetFinal40(1)
				return h
			}(),
			err: true,
		},
		{
			desc:     "Error: Header but no data",
			listData: boolListInBytes(0),
			err:      true,
		},
		{
			desc: "Success",
			want: entryData,
			listData: func() []byte {
				// Put in header.
				h := NewGenericHeader()
				h.SetFieldType(field.FTListBytes)
				h.SetFinal40(2)
				// Put in the entry header + data
				entry0Header := make([]byte, 4)
				binary.Put(entry0Header, uint32(len(entryData[0])))
				h = append(h, entry0Header...)
				h = append(h, entryData[0]...)
				// Put in the entry header + data
				entry1Header := make([]byte, 4)
				binary.Put(entry1Header, uint32(len(entryData[1])))
				h = append(h, entry1Header...)
				h = append(h, entryData[1]...)
				// Add in padding
				h = append(h, Padding(PaddingNeeded(len(h)))...)
				return h
			}(),
		},
	}

	for _, test := range tests {
		s := New(0, m)
		dataSize := len(test.listData) // Must record before we change the slice

		err := s.decodeListBytes(&test.listData, 0)
		switch {
		case err == nil && test.err:
			t.Errorf("TestDecodeListBytes(%s): got err == nil, want err != nil", test.desc)
			continue
		case err != nil && !test.err:
			t.Errorf("TestDecodeListBytes(%s): got err == %s, want err == nil", test.desc, err)
			continue
		case err != nil:
			continue
		}

		lb, err := GetListBytes(s, 0)
		if err != nil {
			panic(err)
		}
		if lb.Len() != len(test.want) {
			t.Errorf("TestDecodeListBytes(%s): Len(): got %d, want %d", test.desc, lb.Len(), test.want)
			continue
		}

		for i := 0; i < len(test.want); i++ {
			got := lb.Get(i)
			if !bytes.Equal(got, test.want[i]) {
				t.Errorf("TestDecodeListBytes(%s): entry[%d]: got %q, want %q", test.desc, i, string(got), string(test.want[i]))
				continue
			}
		}

		if int(*s.structTotal) != 8+dataSize { // structHeader(8) + listHeaderAndData(dataSize)
			t.Errorf("TestDecodeListBytes(%s): structTotal: got %d, want %d)", test.desc, *s.structTotal, 8+dataSize)
		}
		if len(test.listData) > 0 {
			t.Errorf("TestDecodeListBytes(%s): after decode []byte buffer had len %d, but expected 0", test.desc, len(test.listData))
		}
	}
}

func TestDecodeListNum(t *testing.T) {
	mappings := make([]*mapping.Map, len(field.NumberTypes))
	encoded := make([][]byte, len(mappings))
	want := make([]any, len(mappings))
	sizeInBytes := make([]int8, len(mappings))

	for i, ft := range field.NumericListTypes {
		switch ft {
		case field.FTListUint8:
			mappings[i] = &mapping.Map{
				Fields: []*mapping.FieldDescr{
					{Type: ft},
				},
			}
			vals := []uint8{math.MaxUint8, 0, 1, 2, 3, 4, 5, 6, 9, 10} // store 10 values
			want[i] = vals

			n := NewNumbers[uint8]()
			n.Append(vals...)
			encoded[i] = n.Encode()
			// Encode() only encodes what is there. FieldNum is only assigned when attaching directly
			// to a Struct with SetListNumbers(). So we fake it.
			GenericHeader(encoded[i][:8]).SetFieldNum(0)
			if len(encoded[i]) != 24 {
				panic(fmt.Sprintf("encoded %d bytes, but expected %d", len(encoded[i]), 24))
			}

			sizeInBytes[i] = 1
		case field.FTListUint16:
			mappings[i] = &mapping.Map{
				Fields: []*mapping.FieldDescr{
					{Type: ft},
				},
			}
			vals := []uint16{math.MaxUint16, 0, 1, 2, 3} // store 5 values
			want[i] = vals

			n := NewNumbers[uint16]()
			n.Append(vals...)
			encoded[i] = n.Encode()
			// Encode() only encodes what is there. FieldNum is only assigned when attaching directly
			// to a Struct with SetListNumbers(). So we fake it.
			GenericHeader(encoded[i][:8]).SetFieldNum(0)

			sizeInBytes[i] = 2
		case field.FTListUint32:
			mappings[i] = &mapping.Map{
				Fields: []*mapping.FieldDescr{
					{Type: ft},
				},
			}
			vals := []uint32{math.MaxUint32, 0, 1} // store 3 values
			want[i] = vals

			n := NewNumbers[uint32]()
			n.Append(vals...)
			encoded[i] = n.Encode()
			// Encode() only encodes what is there. FieldNum is only assigned when attaching directly
			// to a Struct with SetListNumbers(). So we fake it.
			GenericHeader(encoded[i][:8]).SetFieldNum(0)

			sizeInBytes[i] = 4
		case field.FTListUint64:
			mappings[i] = &mapping.Map{
				Fields: []*mapping.FieldDescr{
					{Type: ft},
				},
			}
			vals := []uint64{math.MaxUint64, 0} // store 2 values
			want[i] = vals

			n := NewNumbers[uint64]()
			n.Append(vals...)
			encoded[i] = n.Encode()
			// Encode() only encodes what is there. FieldNum is only assigned when attaching directly
			// to a Struct with SetListNumbers(). So we fake it.
			GenericHeader(encoded[i][:8]).SetFieldNum(0)
			sizeInBytes[i] = 8
		case field.FTListInt8:
			mappings[i] = &mapping.Map{
				Fields: []*mapping.FieldDescr{
					{Type: ft},
				},
			}
			vals := []int8{math.MaxInt8, math.MinInt8, 1, 2, 3, 4, 5, 6, 9, 10} // store 10 values
			want[i] = vals

			n := NewNumbers[int8]()
			n.Append(vals...)
			encoded[i] = n.Encode()
			// Encode() only encodes what is there. FieldNum is only assigned when attaching directly
			// to a Struct with SetListNumbers(). So we fake it.
			GenericHeader(encoded[i][:8]).SetFieldNum(0)
			if len(encoded[i]) != 24 {
				panic(fmt.Sprintf("encoded %d bytes, but expected %d", len(encoded[i]), 24))
			}

			sizeInBytes[i] = 1
		case field.FTListInt16:
			mappings[i] = &mapping.Map{
				Fields: []*mapping.FieldDescr{
					{Type: ft},
				},
			}
			vals := []int16{math.MaxInt8, math.MinInt16, 1, 2, 3} // store 5 values
			want[i] = vals

			n := NewNumbers[int16]()
			n.Append(vals...)
			encoded[i] = n.Encode()
			// Encode() only encodes what is there. FieldNum is only assigned when attaching directly
			// to a Struct with SetListNumbers(). So we fake it.
			GenericHeader(encoded[i][:8]).SetFieldNum(0)

			sizeInBytes[i] = 2
		case field.FTListInt32:
			mappings[i] = &mapping.Map{
				Fields: []*mapping.FieldDescr{
					{Type: ft},
				},
			}
			vals := []int32{math.MaxInt32, math.MinInt32, 1} // store 3 values
			want[i] = vals

			n := NewNumbers[int32]()
			n.Append(vals...)
			encoded[i] = n.Encode()
			// Encode() only encodes what is there. FieldNum is only assigned when attaching directly
			// to a Struct with SetListNumbers(). So we fake it.
			GenericHeader(encoded[i][:8]).SetFieldNum(0)

			sizeInBytes[i] = 4
		case field.FTListInt64:
			mappings[i] = &mapping.Map{
				Fields: []*mapping.FieldDescr{
					{Type: ft},
				},
			}
			vals := []int64{math.MaxInt64, math.MinInt64} // store 2 values
			want[i] = vals

			n := NewNumbers[int64]()
			n.Append(vals...)
			encoded[i] = n.Encode()
			// Encode() only encodes what is there. FieldNum is only assigned when attaching directly
			// to a Struct with SetListNumbers(). So we fake it.
			GenericHeader(encoded[i][:8]).SetFieldNum(0)

			sizeInBytes[i] = 8
		case field.FTListFloat32:
			mappings[i] = &mapping.Map{
				Fields: []*mapping.FieldDescr{
					{Type: ft},
				},
			}
			vals := []float32{math.MaxFloat32, math.SmallestNonzeroFloat32, 1.1} // store 3 values
			want[i] = vals

			n := NewNumbers[float32]()
			n.Append(vals...)
			encoded[i] = n.Encode()
			// Encode() only encodes what is there. FieldNum is only assigned when attaching directly
			// to a Struct with SetListNumbers(). So we fake it.
			GenericHeader(encoded[i][:8]).SetFieldNum(0)

			sizeInBytes[i] = 4
		case field.FTListFloat64:
			mappings[i] = &mapping.Map{
				Fields: []*mapping.FieldDescr{
					{Type: ft},
				},
			}
			vals := []float64{math.MaxFloat32, math.SmallestNonzeroFloat64} // store 2 values
			want[i] = vals

			n := NewNumbers[float64]()
			n.Append(vals...)
			encoded[i] = n.Encode()
			// Encode() only encodes what is there. FieldNum is only assigned when attaching directly
			// to a Struct with SetListNumbers(). So we fake it.
			GenericHeader(encoded[i][:8]).SetFieldNum(0)

			sizeInBytes[i] = 8
		}
	}

	for i, mapping := range mappings {
		s := New(0, mapping)
		err := s.decodeListNumber(&encoded[i], 0)
		if err != nil {
			t.Errorf("TestDecodeListNum: could not decode type %v: %s", mapping.Fields[0].Type, err)
			continue
		}
		switch mapping.Fields[0].Type {
		case field.FTListInt8:
			wantList := want[i].([]int8)
			gotList, err := GetListNumber[int8](s, 0)
			if err != nil {
				panic(err)
			}
			for x, want := range wantList {
				if want != gotList.Get(x) {
					t.Errorf("TestDecodeListNum([]int8): item[%d]: got %d, want %d", x, gotList.Get(x), want)
				}
			}

			// StructHeader(8) + ListHeader(8) + Data(16 bytes to hold 10 values)
			wantSize := int64(8 + 8 + (8 * wordsRequiredToStore(len(wantList), int(sizeInBytes[i]))))
			if *s.structTotal != wantSize {
				t.Errorf("TestDecodeListNum([]int8): structTotal: got %d, want %d", *s.structTotal, wantSize)
			}
			if s.fields[0].Header.FieldType() != field.FTListInt8 {
				t.Errorf("TestDecodeListNum([]int8]): field type: got %v", field.Type(s.fields[0].Header.FieldType()))
			}
			if s.fields[0].Header.FieldNum() != 0 {
				t.Errorf("TestDecodeListNum([]int8): fieldNum: got %d, want %d", s.fields[0].Header.FieldNum(), 0)
			}
		case field.FTListUint8:
			wantList := want[i].([]uint8)
			gotList, err := GetListNumber[uint8](s, 0)
			if err != nil {
				panic(err)
			}
			for x, want := range wantList {
				if want != gotList.Get(x) {
					t.Errorf("TestDecodeListNum([]uint8): item[%d]: got %d, want %d", x, gotList.Get(x), want)
				}
			}

			wantSize := int64(8 + 8 + (8 * wordsRequiredToStore(len(wantList), int(sizeInBytes[i]))))
			if *s.structTotal != wantSize {
				t.Errorf("TestDecodeListNum([]uint8): structTotal: got %d, want %d", *s.structTotal, wantSize)
			}
			if s.fields[0].Header.FieldType() != field.FTListUint8 {
				t.Errorf("TestDecodeListNum([]uint8): field type: got %v", field.Type(s.fields[0].Header.FieldType()))
			}
			if s.fields[0].Header.FieldNum() != 0 {
				t.Errorf("TestDecodeListNum([]uint8): fieldNum: got %d, want %d", s.fields[0].Header.FieldNum(), 0)
			}
		case field.FTListInt16:
			wantList := want[i].([]int16)
			gotList, err := GetListNumber[int16](s, 0)
			if err != nil {
				panic(err)
			}
			for x, want := range wantList {
				if want != gotList.Get(x) {
					t.Errorf("TestDecodeListNum([]int16): item[%d]: got %d, want %d", x, gotList.Get(x), want)
				}
			}
			wantSize := int64(8 + 8 + (8 * wordsRequiredToStore(len(wantList), int(sizeInBytes[i]))))
			if *s.structTotal != wantSize {
				t.Errorf("TestDecodeListNum([]int16): structTotal: got %d, want %d", *s.structTotal, wantSize)
			}
			if s.fields[0].Header.FieldType() != field.FTListInt16 {
				t.Errorf("TestDecodeListNum([]int16): field type: got %v", field.Type(s.fields[0].Header.FieldType()))
			}
			if s.fields[0].Header.FieldNum() != 0 {
				t.Errorf("TestDecodeListNum([]int16): fieldNum: got %d, want %d", s.fields[0].Header.FieldNum(), 0)
			}
		case field.FTListUint16:
			wantList := want[i].([]uint16)
			gotList, err := GetListNumber[uint16](s, 0)
			if err != nil {
				panic(err)
			}
			for x, want := range wantList {
				if want != gotList.Get(x) {
					t.Errorf("TestDecodeListNum([]uint16): item[%d]: got %d, want %d", x, gotList.Get(x), want)
				}
			}
			wantSize := int64(8 + 8 + (8 * wordsRequiredToStore(len(wantList), int(sizeInBytes[i]))))
			if *s.structTotal != wantSize {
				t.Errorf("TestDecodeListNum([]uint16): structTotal: got %d, want %d", *s.structTotal, wantSize)
			}
			if s.fields[0].Header.FieldType() != field.FTListUint16 {
				t.Errorf("TestDecodeListNum([]uint16): field type: got %v", field.Type(s.fields[0].Header.FieldType()))
			}
			if s.fields[0].Header.FieldNum() != 0 {
				t.Errorf("TestDecodeListNum([]uint16): fieldNum: got %d, want %d", s.fields[0].Header.FieldNum(), 0)
			}

		case field.FTListInt32:
			wantList := want[i].([]int32)
			gotList, err := GetListNumber[int32](s, 0)
			if err != nil {
				panic(err)
			}
			for x, want := range wantList {
				if want != gotList.Get(x) {
					t.Errorf("TestDecodeListNum([]int32): item[%d]: got %d, want %d", x, gotList.Get(x), want)
				}
			}
			wantSize := int64(8 + 8 + (8 * wordsRequiredToStore(len(wantList), int(sizeInBytes[i]))))
			if *s.structTotal != wantSize {
				t.Errorf("TestDecodeListNum([]int32): structTotal: got %d, want %d", *s.structTotal, wantSize)
			}
			if s.fields[0].Header.FieldType() != field.FTListInt32 {
				t.Errorf("TestDecodeListNum([]int32): field type: got %v", field.Type(s.fields[0].Header.FieldType()))
			}
			if s.fields[0].Header.FieldNum() != 0 {
				t.Errorf("TestDecodeListNum([]int32): fieldNum: got %d, want %d", s.fields[0].Header.FieldNum(), 0)
			}
		case field.FTListUint32:
			wantList := want[i].([]uint32)
			gotList, err := GetListNumber[uint32](s, 0)
			if err != nil {
				panic(err)
			}
			for x, want := range wantList {
				if want != gotList.Get(x) {
					t.Errorf("TestDecodeListNum([]uint32): item[%d]: got %d, want %d", x, gotList.Get(x), want)
				}
			}
			wantSize := int64(8 + 8 + (8 * wordsRequiredToStore(len(wantList), int(sizeInBytes[i]))))
			if *s.structTotal != wantSize {
				t.Errorf("TestDecodeListNum([]uint32): structTotal: got %d, want %d", *s.structTotal, wantSize)
			}
			if s.fields[0].Header.FieldType() != field.FTListUint32 {
				t.Errorf("TestDecodeListNum([]uint32): field type: got %v", field.Type(s.fields[0].Header.FieldType()))
			}
			if s.fields[0].Header.FieldNum() != 0 {
				t.Errorf("TestDecodeListNum([]uint32): fieldNum: got %d, want %d", s.fields[0].Header.FieldNum(), 0)
			}
		case field.FTListFloat32:
			wantList := want[i].([]float32)
			gotList, err := GetListNumber[float32](s, 0)
			if err != nil {
				panic(err)
			}
			for x, want := range wantList {
				if want != gotList.Get(x) {
					t.Errorf("TestDecodeListNum([]float32): item[%d]: got %v, want %v", x, gotList.Get(x), want)
				}
			}
			wantSize := int64(8 + 8 + (8 * wordsRequiredToStore(len(wantList), int(sizeInBytes[i]))))
			if *s.structTotal != wantSize {
				t.Errorf("TestDecodeListNum([]float32): structTotal: got %d, want %d", *s.structTotal, wantSize)
			}
			if s.fields[0].Header.FieldType() != field.FTListFloat32 {
				t.Errorf("TestDecodeListNum([]float32): field type: got %v", field.Type(s.fields[0].Header.FieldType()))
			}
			if s.fields[0].Header.FieldNum() != 0 {
				t.Errorf("TestDecodeListNum([]float32): fieldNum: got %d, want %d", s.fields[0].Header.FieldNum(), 0)
			}
		case field.FTListInt64:
			wantList := want[i].([]int64)
			gotList, err := GetListNumber[int64](s, 0)
			if err != nil {
				panic(err)
			}
			for x, want := range wantList {
				if want != gotList.Get(x) {
					t.Errorf("TestDecodeListNum([]int64): item[%d]: got %d, want %d", x, gotList.Get(x), want)
				}
			}
			wantSize := int64(8 + 8 + (8 * wordsRequiredToStore(len(wantList), int(sizeInBytes[i]))))
			if *s.structTotal != wantSize {
				t.Errorf("TestDecodeListNum([]int64): structTotal: got %d, want %d", *s.structTotal, wantSize)
			}
			if s.fields[0].Header.FieldType() != field.FTListInt64 {
				t.Errorf("TestDecodeListNum([]int64): field type: got %v", field.Type(s.fields[0].Header.FieldType()))
			}
			if s.fields[0].Header.FieldNum() != 0 {
				t.Errorf("TestDecodeListNum([]int64): fieldNum: got %d, want %d", s.fields[0].Header.FieldNum(), 0)
			}
		case field.FTListUint64:
			wantList := want[i].([]uint64)
			gotList, err := GetListNumber[uint64](s, 0)
			if err != nil {
				panic(err)
			}
			for x, want := range wantList {
				if want != gotList.Get(x) {
					t.Errorf("TestDecodeListNum([]uint64): item[%d]: got %d, want %d", x, gotList.Get(x), want)
				}
			}
			wantSize := int64(8 + 8 + (8 * wordsRequiredToStore(len(wantList), int(sizeInBytes[i]))))
			if *s.structTotal != wantSize {
				t.Errorf("TestDecodeListNum([]uint64): structTotal: got %d, want %d", *s.structTotal, wantSize)
			}
			if s.fields[0].Header.FieldType() != field.FTListUint64 {
				t.Errorf("TestDecodeListNum([]uint64): field type: got %v", field.Type(s.fields[0].Header.FieldType()))
			}
			if s.fields[0].Header.FieldNum() != 0 {
				t.Errorf("TestDecodeListNum([]uint64): fieldNum: got %d, want %d", s.fields[0].Header.FieldNum(), 0)
			}
		case field.FTListFloat64:
			wantList := want[i].([]float64)
			gotList, err := GetListNumber[float64](s, 0)
			if err != nil {
				panic(err)
			}
			for x, want := range wantList {
				if want != gotList.Get(x) {
					t.Errorf("TestDecodeListNum([]float64): item[%d]: got %v, want %v", x, gotList.Get(x), want)
				}
			}
			wantSize := int64(8 + 8 + (8 * wordsRequiredToStore(len(wantList), int(sizeInBytes[i]))))
			if *s.structTotal != wantSize {
				t.Errorf("TestDecodeListNum([]float64): structTotal: got %d, want %d", *s.structTotal, wantSize)
			}
			if s.fields[0].Header.FieldType() != field.FTListFloat64 {
				t.Errorf("TestDecodeListNum([]float64): field type: got %v", field.Type(s.fields[0].Header.FieldType()))
			}
			if s.fields[0].Header.FieldNum() != 0 {
				t.Errorf("TestDecodeListNum([]float64): fieldNum: got %d, want %d", s.fields[0].Header.FieldNum(), 0)
			}
		}
		if len(encoded[i]) != 0 {
			t.Errorf("TestDecodeNum(%v): did not advance the buffer correctly, had %d bytes left", mapping.Fields[0].Type, len(encoded[i]))
		}
	}
}

func TestDecodeListStruct(t *testing.T) {
	lmsgMapping := &mapping.Map{
		Fields: []*mapping.FieldDescr{
			{Name: "Bool", Type: field.FTBool}, // 1
		},
	}

	msg0Mapping := &mapping.Map{
		Fields: []*mapping.FieldDescr{
			{Name: "ListStructs", Type: field.FTListStructs, Mapping: lmsgMapping},
		},
	}

	msg0Mapping.MustValidate()

	ls0 := New(0, lmsgMapping)
	MustSetBool(ls0, 0, true)       // 16 bytes
	ls1 := New(0, lmsgMapping)      // 8 bytes
	expectedTotal := 8 + 8 + 8 + 16 // s0Header(8) + list header(8) + ls0(16) + ls1(8)

	s0 := New(0, msg0Mapping)
	s0.XXXSetNoZeroTypeCompression()
	MustAppendListStruct(s0, 0, ls0, ls1)

	buff := &bytes.Buffer{}
	written, err := s0.Marshal(buff)
	if err != nil {
		panic(err)
	}
	if *s0.structTotal != int64(written) {
		t.Fatalf("TestDecodeListStruct: s0 had structTotal %d, but encoded it was %d", *s0.structTotal, written)
	}
	if written != expectedTotal {
		t.Fatalf("TestDecodeListStruct: encoding: wrote %d, expected %d", written, expectedTotal)
	}
}

func TestDecodeStruct(t *testing.T) {
	lmsgMapping := &mapping.Map{
		Fields: []*mapping.FieldDescr{
			{Name: "Bool", Type: field.FTBool}, // 0
		},
	}

	msg1Mapping := &mapping.Map{
		Fields: []*mapping.FieldDescr{
			{Name: "Bool", Type: field.FTBool}, // 0
			{Name: "Int8", Type: field.FTInt8},
			{Name: "Int16", Type: field.FTInt16},
			{Name: "Int32", Type: field.FTInt32},
			{Name: "Int64", Type: field.FTInt64}, // 4
			{Name: "Uint8", Type: field.FTUint8},
			{Name: "Uint16", Type: field.FTUint16},
			{Name: "Uint32", Type: field.FTUint32},
			{Name: "Uint64", Type: field.FTUint64},
			{Name: "Float32", Type: field.FTFloat32},                               // 9
			{Name: "Float64", Type: field.FTFloat64},                               // 10
			{Name: "Bytes", Type: field.FTBytes},                                   // 11
			{Name: "ListNumber", Type: field.FTListUint8},                          // 12
			{Name: "ListBytes", Type: field.FTListBytes},                           // 13
			{Name: "ListStructs", Type: field.FTListStructs, Mapping: lmsgMapping}, // 14
		},
	}
	msg0Mapping := &mapping.Map{
		Fields: []*mapping.FieldDescr{
			{Name: "Struct", Type: field.FTStruct, Mapping: msg1Mapping},
		},
	}

	msg0Mapping.MustValidate()

	ls0 := New(0, lmsgMapping)
	ls0.XXXSetNoZeroTypeCompression()
	MustSetBool(ls0, 0, true)  // 16 bytes
	ls1 := New(0, lmsgMapping) // 8 bytes
	ls1.XXXSetNoZeroTypeCompression()

	numList := NewNumbers[uint8]()
	numList.Append(0, 1, 2, 3, 4, 5, 6, 7, 8) // 24

	bytesList := NewBytes()
	bytesList.Append([]byte("what"), []byte("ever")) // 24 = header(8) + entry header(4) + entry(4) + entry header(4) + entry(4)

	s0 := New(0, msg0Mapping)
	s0.XXXSetNoZeroTypeCompression()
	s1 := New(1, msg1Mapping) // 8 bytes
	s1.XXXSetNoZeroTypeCompression()
	MustSetBool(s1, 0, true)                     // 16 bytes
	MustSetNumber(s1, 1, int8(1))                // 24 bytes
	MustSetNumber(s1, 2, int16(1))               // 32 bytes
	MustSetNumber(s1, 3, int32(1))               // 40 bytes
	MustSetNumber(s1, 4, int64(1))               // 56 bytes
	MustSetNumber(s1, 5, uint8(1))               // 64 bytes
	MustSetNumber(s1, 6, uint16(1))              // 72 bytes
	MustSetNumber(s1, 7, uint32(1))              // 80 bytes
	MustSetNumber(s1, 8, uint64(1))              // 96 bytes
	MustSetNumber(s1, 9, float32(1.1))           // 104 bytes
	MustSetNumber(s1, 10, float64(1.1))          // 120 bytes
	MustSetBytes(s1, 11, []byte("Hello"), false) // 136 bytes
	MustSetListNumber(s1, 12, numList)           // 160 bytes
	MustSetListBytes(s1, 13, bytesList)          // 184 bytes
	MustAppendListStruct(s1, 14, ls0, ls1)       // 216 bytes = list header(8) + ls0(16) + ls1(8)

	// Total for both structs: 216 + 8
	expectedTotal := 224
	if err := SetStruct(s0, 0, s1); err != nil {
		panic(err)
	}
	log.Println("header: ", s0.header)
	log.Println("header: ", s1.header)

	buff := &bytes.Buffer{}
	written, err := s0.Marshal(buff)
	if err != nil {
		panic(err)
	}
	if *s0.structTotal != int64(written) {
		t.Fatalf("TestDecodeStruct: s0 had structTotal %d, but encoded it was %d", *s0.structTotal, written)
	}
	if written != expectedTotal {
		t.Fatalf("TestDecodeStruct: encoding: wrote %d, expected %d", written, expectedTotal)
	}

	cp := New(0, msg0Mapping)
	read, err := cp.Unmarshal(buff)
	if err != nil {
		panic(err)
	}
	if read != int(*cp.structTotal) {
		t.Fatalf("TestDecodeStruct: decoding: unmarshal() returned %d bytes read, but .structTotal was %d", read, *cp.structTotal)
	}
	if read != expectedTotal {
		t.Fatalf("TestDecodeStruct: decoding: unmarshal() returned %d bytes, expected %d bytes", read, expectedTotal)
	}

	sub := MustGetStruct(cp, 0)

	log.Println("before call Struct is: ", sub)
	if MustGetBool(sub, 0) != true {
		t.Fatalf("TestDecodeStruct: decoding: msg0.msg1[1] != true")
	}
	if MustGetNumber[int8](sub, 1) != 1 {
		t.Fatalf("TestDecodeStruct: decoding: msg0.msg1[2] != 1")
	}
	if MustGetNumber[int16](sub, 2) != 1 {
		t.Fatalf("TestDecodeStruct: decoding: msg0.msg1[3] != 1")
	}
	if MustGetNumber[int32](sub, 3) != 1 {
		t.Fatalf("TestDecodeStruct: decoding: msg0.msg1[4] != 1")
	}
	if MustGetNumber[int64](sub, 4) != 1 {
		t.Fatalf("TestDecodeStruct: decoding: msg0.msg1[5] != 1")
	}
	if MustGetNumber[uint8](sub, 5) != 1 {
		t.Fatalf("TestDecodeStruct: decoding: msg0.msg1[6] != 1")
	}
	if MustGetNumber[uint16](sub, 6) != 1 {
		t.Fatalf("TestDecodeStruct: decoding: msg0.msg1[7] != 1")
	}
	if MustGetNumber[uint32](sub, 7) != 1 {
		t.Fatalf("TestDecodeStruct: decoding: msg0.msg1[8] != 1")
	}
	if MustGetNumber[uint64](sub, 8) != 1 {
		t.Fatalf("TestDecodeStruct: decoding: msg0.msg1[9] != 1")
	}
	if MustGetNumber[float32](sub, 9) != 1.1 {
		t.Fatalf("TestDecodeStruct: decoding: msg0.msg1[10] != 1.1")
	}
	if MustGetNumber[float64](sub, 10) != 1.1 {
		t.Fatalf("TestDecodeStruct: decoding: msg0.msg1[11] != 1.1")
	}
	if !bytes.Equal(*MustGetBytes(sub, 11), []byte("Hello")) {
		t.Fatalf("TestDecodeStruct: decoding: msg0.msg1[10]: got %q, want %q", *MustGetBytes(sub, 12), []byte("hello"))
	}
	f12 := MustGetListNumber[uint8](sub, 12)
	for i := 0; i < numList.Len(); i++ {
		if numList.Get(i) != f12.Get(i) {
			t.Fatalf("TestDecodeStruct: decoding: msg0.msg1[13]: number index %d: got %d, want %d", i, f12.Get(i), numList.Get(i))
		}
	}
	f13 := MustGetListBytes(sub, 13)
	for i := 0; i < f13.Len(); i++ {
		if !bytes.Equal(f13.Get(i), bytesList.Get(i)) {
			t.Fatalf("TestDecodeStruct: decoding: msg0.msg1[14]: bytes list index %d: got %s, want %s", i, string(f13.Get(i)), string(bytesList.Get(i)))
		}
	}
}
