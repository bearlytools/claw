package structs

import (
	"bytes"
	"math"
	"reflect"
	"testing"

	"github.com/bearlytools/claw/internal/binary"
	"github.com/bearlytools/claw/internal/bits"
	"github.com/bearlytools/claw/internal/field"
	"github.com/bearlytools/claw/internal/mapping"
)

func TestDecodeBool(t *testing.T) {
	m := mapping.Map{
		0: &mapping.FieldDesc{
			Name:   "bool",
			GoName: "Bool",
			Type:   field.FTBool,
		},
	}
	var valTrue uint64

	valTrue = bits.SetValue(uint16(1), valTrue, 0, 16)
	valTrue = bits.SetValue(uint8(field.FTBool), valTrue, 16, 24)
	valTrue = bits.SetBit(valTrue, 24, true)
	bufTrue := make([]byte, 8)
	binary.Put(bufTrue, valTrue)

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
			fieldNum: 1,
			err:      true,
		},
		{
			desc:     "Error: fieldNum is > than len(struct.fields)",
			buf:      make([]byte, 8),
			fieldNum: 2,
			err:      true,
		},
		{
			desc:     "Success: set to true",
			buf:      bufTrue,
			fieldNum: 1,
			want:     true,
		},
	}

	for _, test := range tests {
		wantHeader := make([]byte, len(test.buf))
		copy(wantHeader, test.buf)

		s := New(0, m, nil)
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
		f := s.fields[test.fieldNum-1]
		if !bytes.Equal(f.header, wantHeader) {
			t.Errorf("TestDecodeBool(%s): field.header value: got %v, want %v", test.desc, f.header, wantHeader)
		}
		if len(test.buf) != 0 {
			t.Errorf("TestDecodeBool(%s): did not advance the buffer correctly", test.desc)
		}
	}
}

func numFieldInBytes[N Numbers](value N, dataMap mapping.Map) []byte {
	s := New(0, dataMap, nil)
	if err := SetNumber(s, 1, value); err != nil {
		panic(err)
	}
	var b []byte
	if s.fields[0].ptr == nil {
		b = make([]byte, len(s.fields[0].header))
		copy(b, s.fields[0].header)
	} else {
		ptr := (*[]byte)(s.fields[0].ptr)
		b = make([]byte, len(s.fields[0].header)+len(*ptr))
		copy(b, s.fields[0].header)
		copy(b[8:], *ptr)
	}
	return b
}

func TestDecodeNum(t *testing.T) {
	mappings := make([]mapping.Map, len(field.NumberTypes))
	encoded := make([][]byte, len(mappings))
	want := make([]any, len(mappings))
	size := make([]int8, len(mappings))

	for i, ft := range field.NumberTypes {
		mappings[i] = mapping.Map{0: &mapping.FieldDesc{Type: ft}}
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
			encoded[i] = numFieldInBytes[float64](math.SmallestNonzeroFloat64, mappings[i])
			size[i] = 64
		}
	}

	for i, mapping := range mappings {
		s := New(0, mapping, nil)
		err := s.decodeNum(&encoded[i], 1, size[i])
		if err != nil {
			t.Errorf("TestDecodeNum: could not decode type %v: %s", mapping[0].Type, err)
			continue
		}
		switch mapping[0].Type {
		case field.FTUint8:
			got, err := GetNumber[uint8](s, 1)
			if err != nil {
				panic(err)
			}
			if !reflect.DeepEqual(got, want[i]) {
				t.Errorf("TestDecodeNum(uint8): got %d, want %d", got, want[i])
			}
			if *s.structTotal != 16 {
				t.Errorf("TestDecodeNum(uint8): structTotal: got %d, want %d", *s.structTotal, 16)
			}
			if s.fields[0].header.Next8() != uint8(field.FTUint8) {
				t.Errorf("TestDecodeNum(uint8): fieldNum: got %v", field.Type(s.fields[0].header.Next8()))
			}
			if s.fields[0].header.First16() != 1 {
				t.Errorf("TestDecodeNum(uint8): fieldNum: got %d, want %d", s.fields[0].header.First16(), 1)
			}
		case field.FTUint16:
			got, err := GetNumber[uint16](s, 1)
			if err != nil {
				panic(err)
			}
			if !reflect.DeepEqual(got, want[i]) {
				t.Errorf("TestDecodeNum(uint16): got %d, want %d", got, want[i])
			}
			if *s.structTotal != 16 {
				t.Errorf("TestDecodeNum(uint16): structTotal: got %d, want %d", *s.structTotal, 16)
			}
			if s.fields[0].header.Next8() != uint8(field.FTUint16) {
				t.Errorf("TestDecodeNum(uint16): fieldNum: got %v", field.Type(s.fields[0].header.Next8()))
			}
			if s.fields[0].header.First16() != 1 {
				t.Errorf("TestDecodeNum(uint16): fieldNum: got %d, want %d", s.fields[0].header.First16(), 1)
			}
		case field.FTUint32:
			got, err := GetNumber[uint32](s, 1)
			if err != nil {
				panic(err)
			}
			if !reflect.DeepEqual(got, want[i]) {
				t.Errorf("TestDecodeNum(uint32): got %d, want %d", got, want[i])
			}
			if *s.structTotal != 16 {
				t.Errorf("TestDecodeNum(uint32): structTotal: got %d, want %d", *s.structTotal, 16)
			}
			if s.fields[0].header.Next8() != uint8(field.FTUint32) {
				t.Errorf("TestDecodeNum(uint32): fieldNum: got %v", field.Type(s.fields[0].header.Next8()))
			}
			if s.fields[0].header.First16() != 1 {
				t.Errorf("TestDecodeNum(uint32): fieldNum: got %d, want %d", s.fields[0].header.First16(), 1)
			}
		case field.FTUint64:
			got, err := GetNumber[uint64](s, 1)
			if err != nil {
				panic(err)
			}
			if !reflect.DeepEqual(got, want[i]) {
				t.Errorf("TestDecodeNum(uint64): got %d, want %d", got, want[i])
			}
			if *s.structTotal != 24 {
				t.Errorf("TestDecodeNum(uint64): structTotal: got %d, want %d", *s.structTotal, 24)
			}
			if s.fields[0].header.Next8() != uint8(field.FTUint64) {
				t.Errorf("TestDecodeNum(uint64): fieldNum: got %v", field.Type(s.fields[0].header.Next8()))
			}
			if s.fields[0].header.First16() != 1 {
				t.Errorf("TestDecodeNum(uint64): fieldNum: got %d, want %d", s.fields[0].header.First16(), 1)
			}
		case field.FTInt8:
			got, err := GetNumber[int8](s, 1)
			if err != nil {
				panic(err)
			}
			if !reflect.DeepEqual(got, want[i]) {
				t.Errorf("TestDecodeNum(int8): got %d, want %d", got, want[i])
			}
			if *s.structTotal != 16 {
				t.Errorf("TestDecodeNum(int8): structTotal: got %d, want %d", *s.structTotal, 16)
			}
			if s.fields[0].header.Next8() != uint8(field.FTInt8) {
				t.Errorf("TestDecodeNum(int8): fieldNum: got %v", field.Type(s.fields[0].header.Next8()))
			}
			if s.fields[0].header.First16() != 1 {
				t.Errorf("TestDecodeNum(int8): fieldNum: got %d, want %d", s.fields[0].header.First16(), 1)
			}
		case field.FTInt16:
			got, err := GetNumber[int16](s, 1)
			if err != nil {
				panic(err)
			}
			if !reflect.DeepEqual(got, want[i]) {
				t.Errorf("TestDecodeNum(int16): got %d, want %d", got, want[i])
			}
			if *s.structTotal != 16 {
				t.Errorf("TestDecodeNum(int16): structTotal: got %d, want %d", *s.structTotal, 16)
			}
			if s.fields[0].header.Next8() != uint8(field.FTInt16) {
				t.Errorf("TestDecodeNum(int16): fieldNum: got %v", field.Type(s.fields[0].header.Next8()))
			}
			if s.fields[0].header.First16() != 1 {
				t.Errorf("TestDecodeNum(int16): fieldNum: got %d, want %d", s.fields[0].header.First16(), 1)
			}
		case field.FTInt32:
			got, err := GetNumber[int32](s, 1)
			if err != nil {
				panic(err)
			}
			if !reflect.DeepEqual(got, want[i]) {
				t.Errorf("TestDecodeNum(int32): got %d, want %d", got, want[i])
			}
			if *s.structTotal != 16 {
				t.Errorf("TestDecodeNum(int32): structTotal: got %d, want %d", *s.structTotal, 16)
			}
			if s.fields[0].header.Next8() != uint8(field.FTInt32) {
				t.Errorf("TestDecodeNum(int32): fieldNum: got %v", field.Type(s.fields[0].header.Next8()))
			}
			if s.fields[0].header.First16() != 1 {
				t.Errorf("TestDecodeNum(int32): fieldNum: got %d, want %d", s.fields[0].header.First16(), 1)
			}
		case field.FTInt64:
			got, err := GetNumber[int64](s, 1)
			if err != nil {
				panic(err)
			}
			if !reflect.DeepEqual(got, want[i]) {
				t.Errorf("TestDecodeNum(int64): got %d, want %d", got, want[i])
			}
			if *s.structTotal != 24 {
				t.Errorf("TestDecodeNum(int64): structTotal: got %d, want %d", *s.structTotal, 24)
			}
			if s.fields[0].header.Next8() != uint8(field.FTInt64) {
				t.Errorf("TestDecodeNum(int64): fieldNum: got %v", field.Type(s.fields[0].header.Next8()))
			}
			if s.fields[0].header.First16() != 1 {
				t.Errorf("TestDecodeNum(int64): fieldNum: got %d, want %d", s.fields[0].header.First16(), 1)
			}
		case field.FTFloat32:
			got, err := GetNumber[float32](s, 1)
			if err != nil {
				panic(err)
			}
			if !reflect.DeepEqual(got, want[i]) {
				t.Errorf("TestDecodeNum(float32): got %v, want %v", got, want[i])
			}
			if *s.structTotal != 16 {
				t.Errorf("TestDecodeNum(float32): structTotal: got %d, want %d", *s.structTotal, 16)
			}
			if s.fields[0].header.Next8() != uint8(field.FTFloat32) {
				t.Errorf("TestDecodeNum(float32): fieldNum: got %v", field.Type(s.fields[0].header.Next8()))
			}
			if s.fields[0].header.First16() != 1 {
				t.Errorf("TestDecodeNum(float32): fieldNum: got %d, want %d", s.fields[0].header.First16(), 1)
			}
		case field.FTFloat64:
			got, err := GetNumber[float64](s, 1)
			if err != nil {
				panic(err)
			}
			if !reflect.DeepEqual(got, want[i]) {
				t.Errorf("TestDecodeNum(float64): got %v, want %v", got, want[i])
			}
			if *s.structTotal != 24 {
				t.Errorf("TestDecodeNum(float64): structTotal: got %d, want %d", *s.structTotal, 24)
			}
			if s.fields[0].header.Next8() != uint8(field.FTFloat64) {
				t.Errorf("TestDecodeNum(float64): fieldNum: got %v", field.Type(s.fields[0].header.Next8()))
			}
			if s.fields[0].header.First16() != 1 {
				t.Errorf("TestDecodeNum(float64): fieldNum: got %d, want %d", s.fields[0].header.First16(), 1)
			}
		}
		if len(encoded[i]) != 0 {
			t.Errorf("TestDecodeNum(%v): did not advance the buffer correctly", mapping[0].Type)
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
		s := New(0, mappings[0], nil)
		err := s.decodeNum(&test.buf, test.fieldNum, test.size)

		if err == nil {
			t.Errorf("TestDecodeNum(%s): got err == nil, want err != nil", test.desc)
			continue
		}
	}
}
