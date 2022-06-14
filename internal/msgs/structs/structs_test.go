package structs

import (
	"math"
	"testing"

	"github.com/bearlytools/claw/internal/field"
	"github.com/bearlytools/claw/internal/mapping"
)

func TestBasicEncodeDecodeStruct(t *testing.T) {
	msg0Mapping := mapping.Map{
		&mapping.FieldDesc{Name: "Bool", Type: field.FTBool},
		&mapping.FieldDesc{Name: "Int8", Type: field.FTInt8},
		&mapping.FieldDesc{Name: "Int16", Type: field.FTInt16},
		&mapping.FieldDesc{Name: "Int32", Type: field.FTInt32},
		&mapping.FieldDesc{Name: "Int64", Type: field.FTInt64},
		&mapping.FieldDesc{Name: "Uint8", Type: field.FTUint8},
		&mapping.FieldDesc{Name: "Uint16", Type: field.FTUint16},
		&mapping.FieldDesc{Name: "Uint32", Type: field.FTUint32},
		&mapping.FieldDesc{Name: "Uint64", Type: field.FTUint64},
		&mapping.FieldDesc{Name: "Float32", Type: field.FTFloat32},
		&mapping.FieldDesc{Name: "Float64", Type: field.FTFloat64},
	}
	// 8 * 8 + 16 * 3 = 112

	msg0Factory := New(msg0Mapping)
	root := msg0Factory(0)

	// Test zero value of bool field.
	gotBool, err := GetBool(root, 1)
	if err != nil {
		t.Fatalf("TestBasicEncodeDecodeStruct(initial setup): unexpected error: %s", err)
	}
	if gotBool {
		t.Fatalf("TestBasicEncodeDecodeStruct(initial setup): non-set bool field is true")
	}

	// Set bool field.
	if err := SetBool(root, 1, true); err != nil {
		t.Fatalf("TestBasicEncodeDecodeStruct(initial setup): unexpected error: %s", err)
	}
	// Test bool field.
	gotBool, err = GetBool(root, 1)
	if err != nil {
		t.Fatalf("TestBasicEncodeDecodeStruct(initial setup): unexpected error: %s", err)
	}
	if !gotBool {
		t.Fatalf("TestBasicEncodeDecodeStruct(initial setup): set bool field is false")
	}

	// Test zero value of int8 field.
	gotInt8, err := GetNumber[int8](root, 2)
	if err != nil {
		t.Fatalf("TestBasicEncodeDecodeStruct(initial setup): unexpected error: %s", err)
	}
	if gotInt8 != 0 {
		t.Fatalf("TestBasicEncodeDecodeStruct(initial setup): int8 field is %d", gotInt8)
	}
	// Set int8 field.
	if err := SetNumber(root, 2, int8(-1)); err != nil {
		t.Fatalf("TestBasicEncodeDecodeStruct(initial setup): unexpected error: %s", err)
	}
	// Test int8 field.
	gotInt8, err = GetNumber[int8](root, 2)
	if err != nil {
		t.Fatalf("TestBasicEncodeDecodeStruct(initial setup): unexpected error: %s", err)
	}
	if gotInt8 != -1 {
		t.Fatalf("TestBasicEncodeDecodeStruct(initial setup): int8 field, got %d, want -1", gotInt8)
	}

	// Test zero value of int16 field.
	gotInt16, err := GetNumber[int16](root, 3)
	if err != nil {
		t.Fatalf("TestBasicEncodeDecodeStruct(initial setup): unexpected error: %s", err)
	}
	if gotInt16 != 0 {
		t.Fatalf("TestBasicEncodeDecodeStruct(initial setup): int16 field is %d", gotInt16)
	}
	// Set int16 field.
	if err := SetNumber(root, 3, int16(-2)); err != nil {
		t.Fatalf("TestBasicEncodeDecodeStruct(initial setup): unexpected error: %s", err)
	}
	// Test int16 field.
	gotInt16, err = GetNumber[int16](root, 3)
	if err != nil {
		t.Fatalf("TestBasicEncodeDecodeStruct(initial setup): unexpected error: %s", err)
	}
	if gotInt16 != -2 {
		t.Fatalf("TestBasicEncodeDecodeStruct(initial setup): int16 field, got %d, want -2", gotInt16)
	}

	// Test zero value of int32 field.
	gotInt32, err := GetNumber[int32](root, 4)
	if err != nil {
		t.Fatalf("TestBasicEncodeDecodeStruct(initial setup): unexpected error: %s", err)
	}
	if gotInt32 != 0 {
		t.Fatalf("TestBasicEncodeDecodeStruct(initial setup): int32 field is %d", gotInt32)
	}
	// Set int32 field.
	if err := SetNumber(root, 4, int32(-3)); err != nil {
		t.Fatalf("TestBasicEncodeDecodeStruct(initial setup): unexpected error: %s", err)
	}
	// Test int32 field.
	gotInt32, err = GetNumber[int32](root, 4)
	if err != nil {
		t.Fatalf("TestBasicEncodeDecodeStruct(initial setup): unexpected error: %s", err)
	}
	if gotInt32 != -3 {
		t.Fatalf("TestBasicEncodeDecodeStruct(initial setup): int32 field, got %d, want -3", gotInt32)
	}

	// Test zero value of int64 field.
	gotInt64, err := GetNumber[int64](root, 5)
	if err != nil {
		t.Fatalf("TestBasicEncodeDecodeStruct(initial setup): unexpected error: %s", err)
	}
	if gotInt64 != 0 {
		t.Fatalf("TestBasicEncodeDecodeStruct(initial setup): int64 field is %d", gotInt64)
	}
	// Set int64 field.
	if err := SetNumber(root, 5, int64(-4)); err != nil {
		t.Fatalf("TestBasicEncodeDecodeStruct(initial setup): unexpected error: %s", err)
	}
	// Test int64 field.
	gotInt64, err = GetNumber[int64](root, 5)
	if err != nil {
		t.Fatalf("TestBasicEncodeDecodeStruct(initial setup): unexpected error: %s", err)
	}
	if gotInt64 != -4 {
		t.Fatalf("TestBasicEncodeDecodeStruct(initial setup): int64 field, got %d, want -4", gotInt64)
	}

	// Test zero value of uint8 field.
	gotUint8, err := GetNumber[uint8](root, 6)
	if err != nil {
		t.Fatalf("TestBasicEncodeDecodeStruct(initial setup): unexpected error: %s", err)
	}
	if gotUint8 != 0 {
		t.Fatalf("TestBasicEncodeDecodeStruct(initial setup): uint8 field is %d", gotUint8)
	}
	// Set uint8 field.
	if err := SetNumber(root, 6, uint8(1)); err != nil {
		t.Fatalf("TestBasicEncodeDecodeStruct(initial setup): unexpected error: %s", err)
	}
	// Test uint8 field.
	gotUint8, err = GetNumber[uint8](root, 6)
	if err != nil {
		t.Fatalf("TestBasicEncodeDecodeStruct(initial setup): unexpected error: %s", err)
	}
	if gotUint8 != 1 {
		t.Fatalf("TestBasicEncodeDecodeStruct(initial setup): uint8 field, got %d, want 1", gotUint8)
	}

	// Test zero value of uint16 field.
	gotUint16, err := GetNumber[uint16](root, 7)
	if err != nil {
		t.Fatalf("TestBasicEncodeDecodeStruct(initial setup): unexpected error: %s", err)
	}
	if gotUint16 != 0 {
		t.Fatalf("TestBasicEncodeDecodeStruct(initial setup): uint16 field is %d", gotUint16)
	}
	// Set uint16 field.
	if err := SetNumber(root, 7, uint16(2)); err != nil {
		t.Fatalf("TestBasicEncodeDecodeStruct(initial setup): unexpected error: %s", err)
	}
	// Test uint16 field.
	gotUint16, err = GetNumber[uint16](root, 7)
	if err != nil {
		t.Fatalf("TestBasicEncodeDecodeStruct(initial setup): unexpected error: %s", err)
	}
	if gotUint16 != 2 {
		t.Fatalf("TestBasicEncodeDecodeStruct(initial setup): uint16 field, got %d, want 2", gotUint16)
	}

	// Test zero value of uint32 field.
	gotUint32, err := GetNumber[uint32](root, 8)
	if err != nil {
		t.Fatalf("TestBasicEncodeDecodeStruct(initial setup): unexpected error: %s", err)
	}
	if gotUint32 != 0 {
		t.Fatalf("TestBasicEncodeDecodeStruct(initial setup): uint32 field is %d", gotUint32)
	}
	// Set uint32 field.
	if err := SetNumber(root, 8, uint32(3)); err != nil {
		t.Fatalf("TestBasicEncodeDecodeStruct(initial setup): unexpected error: %s", err)
	}
	// Test uint32 field.
	gotUint32, err = GetNumber[uint32](root, 8)
	if err != nil {
		t.Fatalf("TestBasicEncodeDecodeStruct(initial setup): unexpected error: %s", err)
	}
	if gotUint32 != 3 {
		t.Fatalf("TestBasicEncodeDecodeStruct(initial setup): uint32 field, got %d, want 3", gotUint32)
	}

	// Test zero value of uint64 field.
	gotUint64, err := GetNumber[uint64](root, 9)
	if err != nil {
		t.Fatalf("TestBasicEncodeDecodeStruct(initial setup): unexpected error: %s", err)
	}
	if gotUint64 != 0 {
		t.Fatalf("TestBasicEncodeDecodeStruct(initial setup): uint64 field is %d", gotUint64)
	}
	// Set uint64 field.
	if err := SetNumber(root, 9, uint64(4)); err != nil {
		t.Fatalf("TestBasicEncodeDecodeStruct(initial setup): unexpected error: %s", err)
	}
	// Test uint64 field.
	gotUint64, err = GetNumber[uint64](root, 9)
	if err != nil {
		t.Fatalf("TestBasicEncodeDecodeStruct(initial setup): unexpected error: %s", err)
	}
	if gotUint64 != 4 {
		t.Fatalf("TestBasicEncodeDecodeStruct(initial setup): uint64 field, got %d, want 4", gotUint64)
	}

	// Test zero value of float32 field.
	gotFloat32, err := GetNumber[float32](root, 10)
	if err != nil {
		t.Fatalf("TestBasicEncodeDecodeStruct(initial setup): unexpected error: %s", err)
	}
	if gotFloat32 != 0 {
		t.Fatalf("TestBasicEncodeDecodeStruct(initial setup): float32 field is %v", gotFloat32)
	}
	// Set float32 field.
	if err := SetNumber(root, 10, float32(1.2)); err != nil {
		t.Fatalf("TestBasicEncodeDecodeStruct(initial setup): unexpected error: %s", err)
	}
	// Test float32 field.
	gotFloat32, err = GetNumber[float32](root, 10)
	if err != nil {
		t.Fatalf("TestBasicEncodeDecodeStruct(initial setup): unexpected error: %s", err)
	}
	if gotFloat32 != 1.2 {
		t.Fatalf("TestBasicEncodeDecodeStruct(initial setup): float32 field, got %v, want 1.2", gotFloat32)
	}

	// Test zero value of float64 field.
	gotFloat64, err := GetNumber[float64](root, 11)
	if err != nil {
		t.Fatalf("TestBasicEncodeDecodeStruct(initial setup): unexpected error: %s", err)
	}
	if gotFloat64 != 0 {
		t.Fatalf("TestBasicEncodeDecodeStruct(initial setup): float64 field is %v", gotFloat64)
	}
	// Set float64 field.
	if err := SetNumber(root, 11, float64(1.2)); err != nil {
		t.Fatalf("TestBasicEncodeDecodeStruct(initial setup): unexpected error: %s", err)
	}
	// Test float64 field.
	gotFloat64, err = GetNumber[float64](root, 11)
	if err != nil {
		t.Fatalf("TestBasicEncodeDecodeStruct(initial setup): unexpected error: %s", err)
	}
	if gotFloat64 != 1.2 {
		t.Fatalf("TestBasicEncodeDecodeStruct(initial setup): float64 field, got %v, want 1.2", gotFloat64)
	}

	if *root.total != 112 {
		t.Fatalf("TestBasicEncodeDecodeStruct(initial setup): .total after setting up bool + numeric fields was %d, want %d", *root.total, 112)
	}
}

func TestGetBool(t *testing.T) {
	m := mapping.Map{
		&mapping.FieldDesc{
			Type: field.FTBool,
		},
		&mapping.FieldDesc{
			Type: field.FTFloat32,
		},
		&mapping.FieldDesc{
			Type: field.FTBool,
		},
		&mapping.FieldDesc{
			Type: field.FTBool,
		},
	}

	s := &Struct{
		mapping: m,
		fields: [][]byte{
			nil,
			nil,
			nil,
			nil,
		},
		total: new(int64),
	}

	if err := SetBool(s, 3, true); err != nil {
		panic(err)
	}
	if err := SetBool(s, 4, false); err != nil {
		panic(err)
	}

	tests := []struct {
		desc     string
		s        *Struct
		fieldNum uint16
		want     bool
		err      bool
	}{
		{
			desc:     "Error: fieldNum is 0",
			s:        s,
			fieldNum: 0,
			err:      true,
		},
		{
			desc:     "Error: fieldNum is greater that possible fields",
			s:        s,
			fieldNum: 5,
			err:      true,
		},
		{
			desc:     "Error: fieldNum is not a bool",
			s:        s,
			fieldNum: 2, // FTFloat32
			err:      true,
		},
		{
			desc:     "Error: fieldNum is not a bool",
			s:        s,
			fieldNum: 2, // FTFloat32
			err:      true,
		},
		{
			desc:     "fieldNum that has a nil value and should return false",
			s:        s,
			fieldNum: 1,
			want:     false,
		},
		{
			desc:     "fieldNum that is set to true",
			s:        s,
			fieldNum: 3,
			want:     true,
		},
		{
			desc:     "fieldNum that is set to false",
			s:        s,
			fieldNum: 4,
			want:     false,
		},
	}
	for _, test := range tests {
		got, err := GetBool(test.s, test.fieldNum)
		switch {
		case err == nil && test.err:
			t.Errorf("TestGetBool(%s): got err == nil, want err != nil", test.desc)
			continue
		case err != nil && !test.err:
			t.Errorf("TestGetBool(%s): got err == %s, want err == nil", test.desc, err)
			continue
		case err != nil:
			continue
		}

		if got != test.want {
			t.Errorf("TestGetBool(%s): got %v, want %v", test.desc, got, test.want)
		}
	}
}

func TestSetNumber(t *testing.T) {
	// This is going to only handle cases not handled in GetNumber()
	m := mapping.Map{
		&mapping.FieldDesc{
			Type: field.FTFloat32,
		},
		&mapping.FieldDesc{
			Type: field.FTFloat64,
		},
	}

	s := &Struct{
		mapping: m,
		fields: [][]byte{
			nil,
			nil,
		},
		total: new(int64),
	}

	if err := SetNumber[float32](s, 1, float32(8.7)); err != nil {
		panic(err)
	}

	if err := SetNumber[float64](s, 2, math.MaxFloat64); err != nil {
		panic(err)
	}

	gotFloat32, err := GetNumber[float32](s, 1)
	if err != nil {
		panic(err)
	}
	if gotFloat32 != 8.7 {
		t.Fatalf("TestSetNumber(float32): got %v, want 8.7", gotFloat32)
	}

	gotFloat64, err := GetNumber[float64](s, 2)
	if err != nil {
		panic(err)
	}
	if gotFloat64 != math.MaxFloat64 {
		t.Fatalf("TestSetNumber(float64): got %v, want 8.7", gotFloat64)
	}
}

func TestGetNumber(t *testing.T) {
	m := mapping.Map{
		&mapping.FieldDesc{
			Type: field.FTUint8,
		},
		&mapping.FieldDesc{
			Type: field.FTBool,
		},
		&mapping.FieldDesc{
			Type: field.FTInt8,
		},
		&mapping.FieldDesc{
			Type: field.FTUint64,
		},
		&mapping.FieldDesc{
			Type: field.FTFloat32,
		},
	}

	s := &Struct{
		mapping: m,
		fields: [][]byte{
			nil,
			nil,
			nil,
			nil,
			nil,
		},
		total: new(int64),
	}

	if err := SetNumber[int8](s, 3, 10); err != nil {
		panic(err)
	}
	if err := SetNumber[uint64](s, 4, uint64(math.MaxUint32)+1); err != nil {
		panic(err)
	}
	if err := SetNumber[float32](s, 5, 3.2); err != nil {
		panic(err)
	}

	tests := []struct {
		desc     string
		s        *Struct
		fieldNum uint16
		want     any
		err      bool
	}{
		{
			desc:     "Error: fieldNum is 0",
			s:        s,
			fieldNum: 0,
			err:      true,
		},
		{
			desc:     "Error: fieldNum is greater that possible fields",
			s:        s,
			fieldNum: 30,
			err:      true,
		},
		{
			desc:     "Error: fieldNum is not a number",
			s:        s,
			fieldNum: 2, // FTBool
			err:      true,
		},
		{
			desc:     "fieldNum that has a nil value and should return 0",
			s:        s,
			fieldNum: 1,
			want:     uint8(0),
		},
		{
			desc:     "fieldNum that is set to 10",
			s:        s,
			fieldNum: 3,
			want:     int8(10),
		},
		{
			desc:     "fieldNum that is set to math.MaxUint32+1",
			s:        s,
			fieldNum: 4,
			want:     uint64(math.MaxUint32) + 1,
		},
		{
			desc:     "fieldNum that is set to a float",
			s:        s,
			fieldNum: 5,
			want:     float32(3.2),
		},
	}
	for _, test := range tests {
		var got any
		var err error

		// We can't switch on types for either field 0 or fields not in our mapping.Map, but
		// we still want to test our error conditions.
		if test.fieldNum < 1 || test.fieldNum-1 > uint16(len(m)) {
			got, err = GetNumber[uint8](test.s, test.fieldNum)
		} else { // Any other tests
			switch m[test.fieldNum-1].Type {
			case field.FTUint8:
				got, err = GetNumber[uint8](test.s, test.fieldNum)
			case field.FTUint16:
				got, err = GetNumber[uint16](test.s, test.fieldNum)
			case field.FTUint64:
				got, err = GetNumber[uint64](test.s, test.fieldNum)
			case field.FTInt8:
				got, err = GetNumber[int8](test.s, test.fieldNum)
			case field.FTFloat32:
				got, err = GetNumber[float32](test.s, test.fieldNum)
			case field.FTBool: // So we can test that we get an error on a bad field type
				got, err = GetNumber[uint64](test.s, test.fieldNum)
			default:
				panic("wtf")
			}
		}

		switch {
		case err == nil && test.err:
			t.Errorf("TestGetNumber(%s): got err == nil, want err != nil", test.desc)
			continue
		case err != nil && !test.err:
			t.Errorf("TestGetNumber(%s): got err == %s, want err == nil", test.desc, err)
			continue
		case err != nil:
			continue
		}

		if got != test.want {
			t.Errorf("TestGetNumber(%s): got %v, want %v", test.desc, got, test.want)
		}
	}
}

func TestNumberToDescCheck(t *testing.T) {
	tests := []struct {
		n           any
		desc        mapping.FieldDesc
		wantSize    uint8
		wantIsFloat bool
		wantErr     bool
	}{
		{uint8(1), mapping.FieldDesc{Type: field.FTUint8}, 8, false, false},
		{uint16(1), mapping.FieldDesc{Type: field.FTUint16}, 16, false, false},
		{uint32(1), mapping.FieldDesc{Type: field.FTUint32}, 32, false, false},
		{uint64(1), mapping.FieldDesc{Type: field.FTUint64}, 64, false, false},
		{int8(1), mapping.FieldDesc{Type: field.FTInt8}, 8, false, false},
		{int16(1), mapping.FieldDesc{Type: field.FTInt16}, 16, false, false},
		{int32(1), mapping.FieldDesc{Type: field.FTInt32}, 32, false, false},
		{int64(1), mapping.FieldDesc{Type: field.FTInt64}, 64, false, false},
		{float32(1), mapping.FieldDesc{Type: field.FTFloat32}, 32, true, false},
		{float64(1), mapping.FieldDesc{Type: field.FTFloat64}, 64, true, false},
		// Cause an error.
		{uint8(1), mapping.FieldDesc{Type: field.FTUint16}, 8, false, true},
	}

	for _, test := range tests {
		var gotSize uint8
		var gotIsFloat bool
		var err error
		switch test.n.(type) {
		case uint8:
			gotSize, gotIsFloat, err = numberToDescCheck[uint8](&test.desc)
		case uint16:
			gotSize, gotIsFloat, err = numberToDescCheck[uint16](&test.desc)
		case uint32:
			gotSize, gotIsFloat, err = numberToDescCheck[uint32](&test.desc)
		case uint64:
			gotSize, gotIsFloat, err = numberToDescCheck[uint64](&test.desc)
		case int8:
			gotSize, gotIsFloat, err = numberToDescCheck[int8](&test.desc)
		case int16:
			gotSize, gotIsFloat, err = numberToDescCheck[int16](&test.desc)
		case int32:
			gotSize, gotIsFloat, err = numberToDescCheck[int32](&test.desc)
		case int64:
			gotSize, gotIsFloat, err = numberToDescCheck[int64](&test.desc)
		case float32:
			gotSize, gotIsFloat, err = numberToDescCheck[float32](&test.desc)
		case float64:
			gotSize, gotIsFloat, err = numberToDescCheck[float64](&test.desc)
		default:
			panic("wtf")
		}
		switch {
		case err == nil && test.wantErr:
			t.Errorf("TestNumberToDescCheck(%T): got err == nil, want err != nil", test.n)
			continue
		case err != nil && !test.wantErr:
			t.Errorf("TestNumberToDescCheck(%T): got err == %s, want err == nil", test.n, err)
			continue
		case err != nil:
			continue
		}

		if gotSize != test.wantSize {
			t.Errorf("TestNumberToDescCheck(%T): size: got %v, want %v", test.n, gotSize, test.wantSize)
		}
		if gotIsFloat != test.wantIsFloat {
			t.Errorf("TestNumberToDescCheck(%T): isFloat: got %v, want %v", test.n, gotIsFloat, test.wantIsFloat)
		}
	}
}
