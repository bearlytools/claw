package structs

import (
	"bytes"
	"fmt"
	"log"
	"math"
	"reflect"
	"testing"

	"github.com/bearlytools/claw/languages/go/field"
	"github.com/bearlytools/claw/languages/go/mapping"
	"golang.org/x/exp/constraints"
)

func TestGenericHeader(t *testing.T) {
	h := NewGenericHeader()
	h.SetFieldNum(1)
	h.SetFieldType(field.FTListUint16)
	h.SetFinal40(3)

	if h.FieldNum() != 1 {
		t.Fatalf("TestGenericHeader(FieldNum()): got %d, want %d", h.FieldNum(), 1)
	}
	if h.FieldType() != field.FTListUint16 {
		t.Fatalf("TestGenericHeader(FieldType()): got %d, want %d", h.FieldType(), field.FTListUint16)
	}
	if h.Final40() != 3 {
		t.Fatalf("TestGenericHeader(Final40()): got %d, want %d", h.Final40(), 3)
	}

	// Make sure it clears the old bits.
	h.SetFinal40(240)
	if h.Final40() != 240 {
		t.Fatalf("TestGenericHeader(Final40()): got %d, want %d", h.Final40(), 240)
	}

	h.SetFieldType(8)
	if h.FieldType() != 8 {
		t.Fatalf("TestGenericHeader(First16()): got %d, want %d", h.FieldNum(), 1)
	}

	h.SetFieldNum(16)
	if h.FieldNum() != 16 {
		t.Fatalf("TestGenericHeader(Next8()): got %d, want %d", h.FieldType(), 2)
	}

	// Make sure changing the Next8 and First16 did not any values.
	if h.FieldType() != 8 {
		t.Fatalf("TestGenericHeader(First16()): got %d, want %d", h.FieldNum(), 1)
	}
	if h.Final40() != 240 {
		t.Fatalf("TestGenericHeader(Final40()): got %d, want %d", h.Final40(), 240)
	}

}

// TestBasicEncodeDecodeStruct is a more involved version of decode_test.go/TestDecodeStruct().
// If this tests stops failing, that one probably does too.  It is easier to debug that one.
func TestBasicEncodeDecodeStruct(t *testing.T) {
	msg1Mapping := &mapping.Map{
		Fields: []*mapping.FieldDescr{
			{Name: "Bool", Type: field.FTBool},
		},
	}
	msg0Mapping := &mapping.Map{
		Fields: []*mapping.FieldDescr{
			{Name: "Bool", Type: field.FTBool}, // 1
			{Name: "Int8", Type: field.FTInt8},
			{Name: "Int16", Type: field.FTInt16},
			{Name: "Int32", Type: field.FTInt32},
			{Name: "Int64", Type: field.FTInt64}, // 5
			{Name: "Uint8", Type: field.FTUint8},
			{Name: "Uint16", Type: field.FTUint16},
			{Name: "Uint32", Type: field.FTUint32},
			{Name: "Uint64", Type: field.FTUint64},
			{Name: "Float32", Type: field.FTFloat32},                            // 10
			{Name: "Float64", Type: field.FTFloat64},                            // 11
			{Name: "Bytes", Type: field.FTBytes},                                // 12
			{Name: "Msg1", Type: field.FTStruct, Mapping: msg1Mapping},          // 13
			{Name: "ListMsg1", Type: field.FTListStructs, Mapping: msg1Mapping}, // 14
			{Name: "ListNumber", Type: field.FTListUint8},                       // 15
			{Name: "ListBytes", Type: field.FTListBytes},                        // 16
		},
	}
	// Number      |   Size
	// 8               8 bytes
	// 3           |   16 bytes
	// ========================
	// Total: 112 bytes

	// Bytes Field  | Size
	// 1            | 19 (header + data)
	// ========================
	// Total with padding: 24

	// Total: 136

	root := New(0, msg0Mapping)
	root.XXXSetNoZeroTypeCompression()

	/////////////////////
	// Start Scalars
	/////////////////////

	// Test zero value of bool field.
	gotBool, err := GetBool(root, 0)
	if err != nil {
		t.Fatalf("TestBasicEncodeDecodeStruct(initial setup): unexpected error: %s", err)
	}
	if gotBool {
		t.Fatalf("TestBasicEncodeDecodeStruct(initial setup): non-set bool field is true")
	}

	// Set bool field.
	if err := SetBool(root, 0, true); err != nil {
		t.Fatalf("TestBasicEncodeDecodeStruct(initial setup): unexpected error: %s", err)
	}
	// Test bool field.
	gotBool, err = GetBool(root, 0)
	if err != nil {
		t.Fatalf("TestBasicEncodeDecodeStruct(initial setup): unexpected error: %s", err)
	}
	if !gotBool {
		t.Fatalf("TestBasicEncodeDecodeStruct(initial setup): set bool field is false")
	}

	// Test zero value of int8 field.
	gotInt8, err := GetNumber[int8](root, 1)
	if err != nil {
		t.Fatalf("TestBasicEncodeDecodeStruct(initial setup): unexpected error: %s", err)
	}
	if gotInt8 != 0 {
		t.Fatalf("TestBasicEncodeDecodeStruct(initial setup): int8 field is %d", gotInt8)
	}
	// Set int8 field.
	if err := SetNumber(root, 1, int8(-1)); err != nil {
		t.Fatalf("TestBasicEncodeDecodeStruct(initial setup): unexpected error: %s", err)
	}
	// Test int8 field.
	gotInt8, err = GetNumber[int8](root, 1)
	if err != nil {
		t.Fatalf("TestBasicEncodeDecodeStruct(initial setup): unexpected error: %s", err)
	}
	if gotInt8 != -1 {
		t.Fatalf("TestBasicEncodeDecodeStruct(initial setup): int8 field, got %d, want -1", gotInt8)
	}

	// Test zero value of int16 field.
	gotInt16, err := GetNumber[int16](root, 2)
	if err != nil {
		t.Fatalf("TestBasicEncodeDecodeStruct(initial setup): unexpected error: %s", err)
	}
	if gotInt16 != 0 {
		t.Fatalf("TestBasicEncodeDecodeStruct(initial setup): int16 field is %d", gotInt16)
	}
	// Set int16 field.
	if err := SetNumber(root, 2, int16(-2)); err != nil {
		t.Fatalf("TestBasicEncodeDecodeStruct(initial setup): unexpected error: %s", err)
	}
	// Test int16 field.
	gotInt16, err = GetNumber[int16](root, 2)
	if err != nil {
		t.Fatalf("TestBasicEncodeDecodeStruct(initial setup): unexpected error: %s", err)
	}
	if gotInt16 != -2 {
		t.Fatalf("TestBasicEncodeDecodeStruct(initial setup): int16 field, got %d, want -2", gotInt16)
	}

	// Test zero value of int32 field.
	gotInt32, err := GetNumber[int32](root, 3)
	if err != nil {
		t.Fatalf("TestBasicEncodeDecodeStruct(initial setup): unexpected error: %s", err)
	}
	if gotInt32 != 0 {
		t.Fatalf("TestBasicEncodeDecodeStruct(initial setup): int32 field is %d", gotInt32)
	}
	// Set int32 field.
	if err := SetNumber(root, 3, int32(-3)); err != nil {
		t.Fatalf("TestBasicEncodeDecodeStruct(initial setup): unexpected error: %s", err)
	}
	// Test int32 field.
	gotInt32, err = GetNumber[int32](root, 3)
	if err != nil {
		t.Fatalf("TestBasicEncodeDecodeStruct(initial setup): unexpected error: %s", err)
	}
	if gotInt32 != -3 {
		t.Fatalf("TestBasicEncodeDecodeStruct(initial setup): int32 field, got %d, want -3", gotInt32)
	}

	// Test zero value of int64 field.
	gotInt64, err := GetNumber[int64](root, 4)
	if err != nil {
		t.Fatalf("TestBasicEncodeDecodeStruct(initial setup): unexpected error: %s", err)
	}
	if gotInt64 != 0 {
		t.Fatalf("TestBasicEncodeDecodeStruct(initial setup): int64 field is %d", gotInt64)
	}
	// Set int64 field.
	if err := SetNumber(root, 4, int64(-4)); err != nil {
		t.Fatalf("TestBasicEncodeDecodeStruct(initial setup): unexpected error: %s", err)
	}
	// Test int64 field.
	gotInt64, err = GetNumber[int64](root, 4)
	if err != nil {
		t.Fatalf("TestBasicEncodeDecodeStruct(initial setup): unexpected error: %s", err)
	}
	if gotInt64 != -4 {
		t.Fatalf("TestBasicEncodeDecodeStruct(initial setup): int64 field, got %d, want -4", gotInt64)
	}

	// Test zero value of uint8 field.
	gotUint8, err := GetNumber[uint8](root, 5)
	if err != nil {
		t.Fatalf("TestBasicEncodeDecodeStruct(initial setup): unexpected error: %s", err)
	}
	if gotUint8 != 0 {
		t.Fatalf("TestBasicEncodeDecodeStruct(initial setup): uint8 field is %d", gotUint8)
	}
	// Set uint8 field.
	if err := SetNumber(root, 5, uint8(1)); err != nil {
		t.Fatalf("TestBasicEncodeDecodeStruct(initial setup): unexpected error: %s", err)
	}
	// Test uint8 field.
	gotUint8, err = GetNumber[uint8](root, 5)
	if err != nil {
		t.Fatalf("TestBasicEncodeDecodeStruct(initial setup): unexpected error: %s", err)
	}
	if gotUint8 != 1 {
		t.Fatalf("TestBasicEncodeDecodeStruct(initial setup): uint8 field, got %d, want 1", gotUint8)
	}

	// Test zero value of uint16 field.
	gotUint16, err := GetNumber[uint16](root, 6)
	if err != nil {
		t.Fatalf("TestBasicEncodeDecodeStruct(initial setup): unexpected error: %s", err)
	}
	if gotUint16 != 0 {
		t.Fatalf("TestBasicEncodeDecodeStruct(initial setup): uint16 field is %d", gotUint16)
	}
	// Set uint16 field.
	if err := SetNumber(root, 6, uint16(2)); err != nil {
		t.Fatalf("TestBasicEncodeDecodeStruct(initial setup): unexpected error: %s", err)
	}
	// Test uint16 field.
	gotUint16, err = GetNumber[uint16](root, 6)
	if err != nil {
		t.Fatalf("TestBasicEncodeDecodeStruct(initial setup): unexpected error: %s", err)
	}
	if gotUint16 != 2 {
		t.Fatalf("TestBasicEncodeDecodeStruct(initial setup): uint16 field, got %d, want 2", gotUint16)
	}

	// Test zero value of uint32 field.
	gotUint32, err := GetNumber[uint32](root, 7)
	if err != nil {
		t.Fatalf("TestBasicEncodeDecodeStruct(initial setup): unexpected error: %s", err)
	}
	if gotUint32 != 0 {
		t.Fatalf("TestBasicEncodeDecodeStruct(initial setup): uint32 field is %d", gotUint32)
	}
	// Set uint32 field.
	if err := SetNumber(root, 7, uint32(3)); err != nil {
		t.Fatalf("TestBasicEncodeDecodeStruct(initial setup): unexpected error: %s", err)
	}
	// Test uint32 field.
	gotUint32, err = GetNumber[uint32](root, 7)
	if err != nil {
		t.Fatalf("TestBasicEncodeDecodeStruct(initial setup): unexpected error: %s", err)
	}
	if gotUint32 != 3 {
		t.Fatalf("TestBasicEncodeDecodeStruct(initial setup): uint32 field, got %d, want 3", gotUint32)
	}

	// Test zero value of uint64 field.
	gotUint64, err := GetNumber[uint64](root, 8)
	if err != nil {
		t.Fatalf("TestBasicEncodeDecodeStruct(initial setup): unexpected error: %s", err)
	}
	if gotUint64 != 0 {
		t.Fatalf("TestBasicEncodeDecodeStruct(initial setup): uint64 field is %d", gotUint64)
	}
	// Set uint64 field.
	if err := SetNumber(root, 8, uint64(4)); err != nil {
		t.Fatalf("TestBasicEncodeDecodeStruct(initial setup): unexpected error: %s", err)
	}
	// Test uint64 field.
	gotUint64, err = GetNumber[uint64](root, 8)
	if err != nil {
		t.Fatalf("TestBasicEncodeDecodeStruct(initial setup): unexpected error: %s", err)
	}
	if gotUint64 != 4 {
		t.Fatalf("TestBasicEncodeDecodeStruct(initial setup): uint64 field, got %d, want 4", gotUint64)
	}

	// Test zero value of float32 field.
	gotFloat32, err := GetNumber[float32](root, 9)
	if err != nil {
		t.Fatalf("TestBasicEncodeDecodeStruct(initial setup): unexpected error: %s", err)
	}
	if gotFloat32 != 0 {
		t.Fatalf("TestBasicEncodeDecodeStruct(initial setup): float32 field is %v", gotFloat32)
	}
	// Set float32 field.
	if err := SetNumber(root, 9, float32(1.2)); err != nil {
		t.Fatalf("TestBasicEncodeDecodeStruct(initial setup): unexpected error: %s", err)
	}
	// Test float32 field.
	gotFloat32, err = GetNumber[float32](root, 9)
	if err != nil {
		t.Fatalf("TestBasicEncodeDecodeStruct(initial setup): unexpected error: %s", err)
	}
	if gotFloat32 != 1.2 {
		t.Fatalf("TestBasicEncodeDecodeStruct(initial setup): float32 field, got %v, want 1.2", gotFloat32)
	}

	// Test zero value of float64 field.
	gotFloat64, err := GetNumber[float64](root, 10)
	if err != nil {
		t.Fatalf("TestBasicEncodeDecodeStruct(initial setup): unexpected error: %s", err)
	}
	if gotFloat64 != 0 {
		t.Fatalf("TestBasicEncodeDecodeStruct(initial setup): float64 field is %v", gotFloat64)
	}
	// Set float64 field.
	if err := SetNumber(root, 10, float64(1.2)); err != nil {
		t.Fatalf("TestBasicEncodeDecodeStruct(initial setup): unexpected error: %s", err)
	}
	// Test float64 field.
	gotFloat64, err = GetNumber[float64](root, 10)
	if err != nil {
		t.Fatalf("TestBasicEncodeDecodeStruct(initial setup): unexpected error: %s", err)
	}
	if gotFloat64 != 1.2 {
		t.Fatalf("TestBasicEncodeDecodeStruct(initial setup): float64 field, got %v, want 1.2", gotFloat64)
	}

	var totalWithScalars int64 = 120 // Scalar sizes + 8 byte hedaer for Struct
	if *root.structTotal != totalWithScalars {
		t.Fatalf("TestBasicEncodeDecodeStruct(initial setup): .total after setting up bool + numeric fields was %d, want %d", *root.structTotal, totalWithScalars)
	}

	if err := marshalCheck(root, int(totalWithScalars)); err != nil {
		t.Fatalf("TestBasicEncodeDecodeStruct(encoding after adding scalar fields): %s", err)
	}

	/////////////////////
	// End Scalars
	/////////////////////

	/////////////////////
	// Start Bytes
	/////////////////////

	// Test zero value of Bytes field.
	getBytes, err := GetBytes(root, 11)
	if err != nil {
		t.Fatalf("TestBasicEncodeDecodeStruct(initial setup): unexpected error: %s", err)
	}
	if getBytes != nil {
		t.Fatalf("TestBasicEncodeDecodeStruct(initial setup): bytes field is %v", getBytes)
	}

	// Add byte field.
	strData := "Hello World"
	err = SetBytes(root, 11, []byte(strData), false)
	if err != nil {
		t.Fatalf("TestBasicEncodeDecodeStruct(initial setup): unexpected error: %s", err)
	}
	getBytes, err = GetBytes(root, 11)
	if err != nil {
		t.Fatalf("TestBasicEncodeDecodeStruct(initial setup): unexpected error: %s", err)
	}
	if !bytes.Equal(*getBytes, []byte(strData)) {
		t.Fatalf("TestBasicEncodeDecodeStruct(initial setup): want empty bytes field: %v, got: %v", strData, string(*getBytes))
	}

	log.Println("before total: ", totalWithScalars)
	log.Println("we are adding: ", 8+SizeWithPadding(len(strData)))
	totalWithBytes := totalWithScalars + 8 + int64(SizeWithPadding(len(strData)))
	log.Println("totalWithBytes: ", totalWithBytes)
	if *root.structTotal != totalWithBytes {
		t.Fatalf("TestBasicEncodeDecodeStruct(initial setup): .total after adding bytes field was %d, want %d", *root.structTotal, totalWithBytes)
	}

	if *root.structTotal%8 != 0 {
		t.Fatalf("TestBasicEncodeDecodeStruct(initial setup): structTotal(%d) is not divisible by 8", *root.structTotal)
	}

	if err := marshalCheck(root, int(totalWithBytes)); err != nil {
		t.Fatalf("TestBasicEncodeDecodeStruct(encoding after adding bytes field): %s", err)
	}

	/////////////////////
	// End Bytes
	/////////////////////

	////////////////////
	// Start Struct
	////////////////////
	sub := New(13, msg1Mapping)
	if err := SetStruct(root, 12, sub); err != nil {
		panic(err)
	}
	totalWithStruct := totalWithBytes + 8
	if *root.structTotal != totalWithStruct {
		t.Fatalf("TestBasicEncodeDecodeStruct(adding Struct): root.Struct total was %d, want %d", *root.structTotal, totalWithStruct)
	}

	if err = SetBool(sub, 0, true); err != nil {
		t.Fatalf("TestBasicEncodeDecodeStruct(sub Struct): root.Struct[13], unexpected error on SetBool(): %s", err)
	}
	sub, err = GetStruct(root, 12)
	if err != nil {
		t.Fatalf("TestBasicEncodeDecodeStruct(get sub Struct): unexpected error on GetStruct(): %s", err)
	}
	gotBool, err = GetBool(sub, 0)
	if err != nil {
		t.Fatalf("TestBasicEncodeDecodeStruct(sub Struct): root.Struct[13], unexpected error on GotBool(): %s", err)
	}
	totalWithStruct += 8 // Additional space for Bool value.
	if !gotBool {
		t.Fatalf("TestBasicEncodeDecodeStruct(sub Struct): root.Struct[13], got %v, want %v", gotBool, true)
	}
	if *root.structTotal != totalWithStruct {
		t.Fatalf("TestBasicEncodeDecodeStruct(adding Struct+Bool value): root.Struct total was %d, want %d", *root.structTotal, totalWithStruct)
	}

	if err := marshalCheck(root, totalWithStruct); err != nil {
		t.Fatalf("TestBasicEncodeDecodeStruct(encoding after adding sub Struct): %s", err)
	}

	////////////////////
	// End Struct
	////////////////////

	////////////////////
	// Start List Struct
	////////////////////
	structs := []*Struct{
		New(0, msg1Mapping),
		New(0, msg1Mapping),
	}

	if err := AppendListStruct(root, 13, structs...); err != nil {
		t.Fatalf("TestBasicEncodeDecodeStruct(adding ListStruct): AddListStruct() had error: %s", err)
	}

	totalWithListStruct := totalWithStruct + 8 + 16 // ListStruct header + two Struct headers
	if *root.structTotal != totalWithListStruct {
		t.Fatalf("TestBasicEncodeDecodeStruct(adding ListStruct): root.Struct total was %d, want %d", *root.structTotal, totalWithListStruct)
	}

	if err = SetBool(structs[1], 0, true); err != nil {
		t.Fatalf("TestBasicEncodeDecodeStruct(ListStruct): root.Struct[14], unexpected error on SetBool(structs[1]...): %s", err)
	}

	listStruct, err := GetListStruct(root, 13)
	if err != nil {
		t.Fatalf("TestBasicEncodeDecodeStruct(ListStruct): GetListStruct() had error: %s", err)
	}
	gotBool, err = GetBool(listStruct.Get(1), 0)
	if err != nil {
		t.Fatalf("TestBasicEncodeDecodeStruct(ListStruct): root.Struct[14][1], unexpected error on GotBool(): %s", err)
	}
	totalWithListStruct += 8 // Additional space for Bool value.
	if !gotBool {
		t.Fatalf("TestBasicEncodeDecodeStruct(ListStruct): root.Struct[14][1], got %v, want %v", gotBool, true)
	}
	if *root.structTotal != totalWithListStruct {
		t.Fatalf("TestBasicEncodeDecodeStruct(adding ListStruct+Bool value): root.Struct total was %d, want %d", *root.structTotal, totalWithListStruct)
	}
	if err := marshalCheck(root, totalWithListStruct); err != nil {
		t.Fatalf("TestBasicEncodeDecodeStruct(encoding after adding list Struct): %s", err)
	}

	////////////////////
	// End List Struct
	////////////////////

	////////////////////
	// Start List Number
	////////////////////

	nums := NewNumbers[uint8]()
	if err = SetListNumber(root, 14, nums); err != nil {
		t.Fatalf("TestBasicEncodeDecodeStruct(encoding list of numbers): %s", err)
	}

	totalWithListNumber := totalWithListStruct + 8
	if *root.structTotal != totalWithListNumber {
		t.Fatalf("TestBasicEncodeDecodeStruct(adding ListNumber): root.Struct total was %d, want %d", *root.structTotal, totalWithListNumber)
	}

	nums.Append(1, 2, 3, 4, 5, 6, 7, 8, 9)
	totalWithListNumber += 16 // Requires 16 bytes to hold 9 uint8 values
	if *root.structTotal != totalWithListNumber {
		t.Fatalf("TestBasicEncodeDecodeStruct(appending to ListNumber): root.Struct total was %d, want %d", *root.structTotal, totalWithListNumber)
	}

	if err := marshalCheck(root, totalWithListNumber); err != nil {
		t.Fatalf("TestBasicEncodeDecodeStruct(encoding after adding ListNumber): %s", err)
	}

	////////////////////
	// End List Number
	////////////////////

	////////////////////
	// Start List Bytes
	////////////////////

	bytesList := NewBytes()
	if err := SetListBytes(root, 15, bytesList); err != nil {
		t.Fatalf("TestBasicEncodeDecodeStruct(encoding list of bytes): %s", err)
	}
	totalWithListBytes := totalWithListNumber + 8

	if *root.structTotal != totalWithListBytes {
		t.Fatalf("TestBasicEncodeDecodeStruct(adding Listbytes): root.Struct total was %d, want %d", *root.structTotal, totalWithListBytes)
	}

	if err := bytesList.Append([]byte("what"), []byte("ever")); err != nil {
		t.Fatalf("TestBasicEncodeDecodeStruct(append Listbytes): error: %s", err)
	}

	totalWithListBytes += 16 // 2 * content(4 bytes each) + two entry headers(4 bytes)
	if *root.structTotal != totalWithListBytes {
		t.Fatalf("TestBasicEncodeDecodeStruct(appending to Listbytes): root.Struct total was %d, want %d", *root.structTotal, totalWithListBytes)
	}
	if err := marshalCheck(root, totalWithListBytes); err != nil {
		t.Fatalf("TestBasicEncodeDecodeStruct(encoding after adding Listbytes): %s", err)
	}

	////////////////////
	// End List Bytes
	////////////////////

	////////////////////
	// Start Decode
	////////////////////

	log.Println("======================================================")
	buff := new(bytes.Buffer)
	written, _ := root.Marshal(buff) // We just marshalled, so no error
	log.Println("encoder says it wrote: ", written)
	cp := New(0, msg0Mapping)
	log.Println("new root is: ", *cp.structTotal)
	if _, err := cp.unmarshal(buff); err != nil {
		panic(err)
	}
	if *cp.structTotal != int64(written) {
		t.Fatalf("TestBasicEncodeDecodeStruct(decode message): cp.Struct total was %d, want %d", *cp.structTotal, written)
	}

	if err := compareStruct(root, cp); err != nil {
		t.Fatalf("TestBasicEncodeDecodeStruct(decode message): %s", err)
	}
}

func compareStruct(a, b *Struct) error {
	if !reflect.DeepEqual(a.mapping, b.mapping) {
		return fmt.Errorf("a and b don't have the same mapping, so they cannot be the same")
	}
	if len(a.fields) != len(b.fields) {
		return fmt.Errorf("a and b don't have the same number of fields, so they cannot be the same")
	}
	if len(a.fields) != len(a.mapping.Fields) {
		return fmt.Errorf("a has fields length %d, mapping has %d, malformed Struct", len(a.fields), len(a.mapping.Fields))
	}
	if len(b.fields) != len(b.mapping.Fields) {
		return fmt.Errorf("b has fields length %d, mapping has %d, malformed Struct", len(a.fields), len(a.mapping.Fields))
	}
	for i := 0; i < len(a.fields); i++ {
		fieldNum := uint16(i)
		switch a.mapping.Fields[i].Type {
		case field.FTBool:
			v0 := MustGetBool(a, fieldNum)
			v1 := MustGetBool(b, fieldNum)
			if v0 != v1 {
				return fmt.Errorf("%d field: diff: a was %v, b was %v", fieldNum, v0, v1)
			}
		case field.FTInt8:
			v0 := MustGetNumber[int8](a, fieldNum)
			v1 := MustGetNumber[int8](b, fieldNum)
			if v0 != v1 {
				return fmt.Errorf("%d field: diff: a was %v, b was %v", fieldNum, v0, v1)
			}
		case field.FTInt16:
			v0 := MustGetNumber[int16](a, fieldNum)
			v1 := MustGetNumber[int16](b, fieldNum)
			if v0 != v1 {
				return fmt.Errorf("%d field: diff: a was %v, b was %v", fieldNum, v0, v1)
			}
		case field.FTInt32:
			v0 := MustGetNumber[int32](a, fieldNum)
			v1 := MustGetNumber[int32](b, fieldNum)
			if v0 != v1 {
				return fmt.Errorf("%d field: diff: a was %v, b was %v", fieldNum, v0, v1)
			}
		case field.FTInt64:
			v0 := MustGetNumber[int64](a, fieldNum)
			v1 := MustGetNumber[int64](b, fieldNum)
			if v0 != v1 {
				return fmt.Errorf("%d field: diff: a was %v, b was %v", fieldNum, v0, v1)
			}
		case field.FTUint8:
			v0 := MustGetNumber[uint8](a, fieldNum)
			v1 := MustGetNumber[uint8](b, fieldNum)
			if v0 != v1 {
				return fmt.Errorf("%d field: diff: a was %v, b was %v", fieldNum, v0, v1)
			}
		case field.FTUint16:
			v0 := MustGetNumber[uint16](a, fieldNum)
			v1 := MustGetNumber[uint16](b, fieldNum)
			if v0 != v1 {
				return fmt.Errorf("%d field: diff: a was %v, b was %v", fieldNum, v0, v1)
			}
		case field.FTUint32:
			v0 := MustGetNumber[uint32](a, fieldNum)
			v1 := MustGetNumber[uint32](b, fieldNum)
			if v0 != v1 {
				return fmt.Errorf("%d field: diff: a was %v, b was %v", fieldNum, v0, v1)
			}
		case field.FTUint64:
			v0 := MustGetNumber[uint64](a, fieldNum)
			v1 := MustGetNumber[uint64](b, fieldNum)
			if v0 != v1 {
				return fmt.Errorf("%d field: diff: a was %v, b was %v", fieldNum, v0, v1)
			}
		case field.FTFloat32:
			v0 := MustGetNumber[float32](a, fieldNum)
			v1 := MustGetNumber[float32](b, fieldNum)
			if v0 != v1 {
				return fmt.Errorf("%d field: diff: a was %v, b was %v", fieldNum, v0, v1)
			}
		case field.FTFloat64:
			v0 := MustGetNumber[float64](a, fieldNum)
			v1 := MustGetNumber[float64](b, fieldNum)
			if v0 != v1 {
				return fmt.Errorf("%d field: diff: a was %v, b was %v", fieldNum, v0, v1)
			}
		case field.FTString, field.FTBytes:
			v0 := MustGetBytes(a, fieldNum)
			v1 := MustGetBytes(b, fieldNum)
			if !bytes.Equal(*v0, *v1) {
				return fmt.Errorf("%d field: diff: a was %v, b was %v", fieldNum, v0, v1)
			}
		case field.FTStruct:
			v0 := MustGetStruct(a, fieldNum)
			v1 := MustGetStruct(b, fieldNum)
			if err := compareStruct(v0, v1); err != nil {
				return fmt.Errorf("%d.%w", i, err)
			}
		case field.FTListBools:
			v0 := MustGetListBool(a, fieldNum)
			v1 := MustGetListBool(b, fieldNum)
			for x := 0; x < v0.Len(); x++ {
				if v0.Get(x) != v1.Get(x) {
					return fmt.Errorf("%d field, item %d: a was %v, b was %v", fieldNum, x, v0.Get(x), v1.Get(x))
				}
			}
		case field.FTListInt8:
			v0 := MustGetListNumber[int8](a, fieldNum)
			v1 := MustGetListNumber[int8](b, fieldNum)
			for x := 0; x < v0.Len(); x++ {
				if v0.Get(x) != v1.Get(x) {
					return fmt.Errorf("%d field, item %d: a was %v, b was %v", fieldNum, x, v0.Get(x), v1.Get(x))
				}
			}
		case field.FTListUint8:
			v0 := MustGetListNumber[uint8](a, fieldNum)
			v1 := MustGetListNumber[uint8](b, fieldNum)
			for x := 0; x < v0.Len(); x++ {
				if v0.Get(x) != v1.Get(x) {
					return fmt.Errorf("%d field, item %d: a was %v, b was %v", fieldNum, x, v0.Get(x), v1.Get(x))
				}
			}
		case field.FTListInt16:
			v0 := MustGetListNumber[int16](a, fieldNum)
			v1 := MustGetListNumber[int16](b, fieldNum)
			for x := 0; x < v0.Len(); x++ {
				if v0.Get(x) != v1.Get(x) {
					return fmt.Errorf("%d field, item %d: a was %v, b was %v", fieldNum, x, v0.Get(x), v1.Get(x))
				}
			}
		case field.FTListUint16:
			v0 := MustGetListNumber[uint16](a, fieldNum)
			v1 := MustGetListNumber[uint16](b, fieldNum)
			for x := 0; x < v0.Len(); x++ {
				if v0.Get(x) != v1.Get(x) {
					return fmt.Errorf("%d field, item %d: a was %v, b was %v", fieldNum, x, v0.Get(x), v1.Get(x))
				}
			}
		case field.FTListInt32:
			v0 := MustGetListNumber[int32](a, fieldNum)
			v1 := MustGetListNumber[int32](b, fieldNum)
			for x := 0; x < v0.Len(); x++ {
				if v0.Get(x) != v1.Get(x) {
					return fmt.Errorf("%d field, item %d: a was %v, b was %v", fieldNum, x, v0.Get(x), v1.Get(x))
				}
			}
		case field.FTListUint32:
			v0 := MustGetListNumber[uint32](a, fieldNum)
			v1 := MustGetListNumber[uint32](b, fieldNum)
			for x := 0; x < v0.Len(); x++ {
				if v0.Get(x) != v1.Get(x) {
					return fmt.Errorf("%d field, item %d: a was %v, b was %v", fieldNum, x, v0.Get(x), v1.Get(x))
				}
			}
		case field.FTListFloat32:
			v0 := MustGetListNumber[float32](a, fieldNum)
			v1 := MustGetListNumber[float32](b, fieldNum)
			for x := 0; x < v0.Len(); x++ {
				if v0.Get(x) != v1.Get(x) {
					return fmt.Errorf("%d field, item %d: a was %v, b was %v", fieldNum, x, v0.Get(x), v1.Get(x))
				}
			}
		case field.FTListInt64:
			v0 := MustGetListNumber[int64](a, fieldNum)
			v1 := MustGetListNumber[int64](b, fieldNum)
			for x := 0; x < v0.Len(); x++ {
				if v0.Get(x) != v1.Get(x) {
					return fmt.Errorf("%d field, item %d: a was %v, b was %v", fieldNum, x, v0.Get(x), v1.Get(x))
				}
			}
		case field.FTListUint64:
			v0 := MustGetListNumber[uint64](a, fieldNum)
			v1 := MustGetListNumber[uint64](b, fieldNum)
			for x := 0; x < v0.Len(); x++ {
				if v0.Get(x) != v1.Get(x) {
					return fmt.Errorf("%d field, item %d: a was %v, b was %v", fieldNum, x, v0.Get(x), v1.Get(x))
				}
			}
		case field.FTListFloat64:
			v0 := MustGetListNumber[float64](a, fieldNum)
			v1 := MustGetListNumber[float64](b, fieldNum)
			for x := 0; x < v0.Len(); x++ {
				if v0.Get(x) != v1.Get(x) {
					return fmt.Errorf("%d field, item %d: a was %v, b was %v", fieldNum, x, v0.Get(x), v1.Get(x))
				}
			}
		case field.FTListBytes:
			v0 := MustGetListBytes(a, fieldNum)
			v1 := MustGetListBytes(b, fieldNum)
			for x := 0; x < v0.Len(); x++ {
				b0 := v0.Get(x)
				b1 := v1.Get(x)
				if !bytes.Equal(b0, b1) {
					return fmt.Errorf("%d field, item %d: a was %v, b was %v", fieldNum, x, b0, b1)
				}
			}
		case field.FTListStructs:
			v0 := MustGetListStruct(a, fieldNum)
			v1 := MustGetListStruct(b, fieldNum)
			for x := 0; x < v0.Len(); x++ {
				s0 := v0.Get(x)
				s1 := v1.Get(x)
				if err := compareStruct(s0, s1); err != nil {
					return fmt.Errorf("%d field, item %d, a was %v, b was %v", fieldNum, x, s0, s1)
				}
			}
		default:
			return fmt.Errorf("%d field has unknown type: %v", i, a.mapping.Fields[i].Type)
		}
	}
	return nil
}

func marshalCheck[I constraints.Integer](msg *Struct, wantWritten I) error {
	buff := new(bytes.Buffer)
	written, err := msg.Marshal(buff)
	log.Println("marshalCheck says we wrote: ", written)
	if err != nil {
		return err
	}

	if written != int(wantWritten) {
		return fmt.Errorf("wrote %d bytes, but total was %d", written, wantWritten)
	}
	return nil
}

func TestGetBool(t *testing.T) {
	m := &mapping.Map{
		Fields: []*mapping.FieldDescr{
			&mapping.FieldDescr{
				Type: field.FTBool,
			},
			&mapping.FieldDescr{
				Type: field.FTFloat32,
			},
			&mapping.FieldDescr{
				Type: field.FTBool,
			},
			&mapping.FieldDescr{
				Type: field.FTBool,
			},
		},
	}

	s := New(0, m)

	if err := SetBool(s, 2, true); err != nil {
		panic(err)
	}
	if err := SetBool(s, 3, false); err != nil {
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
			desc:     "Error: fieldNum is greater that possible fields",
			s:        s,
			fieldNum: 4,
			err:      true,
		},
		{
			desc:     "Error: fieldNum is not a bool",
			s:        s,
			fieldNum: 1, // FTFloat32
			err:      true,
		},
		{
			desc:     "fieldNum that has a nil value and should return false",
			s:        s,
			fieldNum: 0,
			want:     false,
		},
		{
			desc:     "fieldNum that is set to true",
			s:        s,
			fieldNum: 2,
			want:     true,
		},
		{
			desc:     "fieldNum that is set to false",
			s:        s,
			fieldNum: 3,
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
	m := &mapping.Map{
		Fields: []*mapping.FieldDescr{
			&mapping.FieldDescr{
				Type: field.FTFloat32,
			},
			&mapping.FieldDescr{
				Type: field.FTFloat64,
			},
		},
	}

	s := New(0, m)

	if err := SetNumber(s, 0, float32(8.7)); err != nil {
		panic(err)
	}

	if err := SetNumber(s, 1, math.MaxFloat64); err != nil {
		panic(err)
	}

	gotFloat32, err := GetNumber[float32](s, 0)
	if err != nil {
		panic(err)
	}
	if gotFloat32 != 8.7 {
		t.Fatalf("TestSetNumber(float32): got %v, want 8.7", gotFloat32)
	}

	gotFloat64, err := GetNumber[float64](s, 1)
	if err != nil {
		panic(err)
	}
	if gotFloat64 != math.MaxFloat64 {
		t.Fatalf("TestSetNumber(float64): got %v, want 8.7", gotFloat64)
	}
}

func TestGetNumber(t *testing.T) {
	m := &mapping.Map{
		Fields: []*mapping.FieldDescr{
			&mapping.FieldDescr{
				Type: field.FTUint8,
			},
			&mapping.FieldDescr{
				Type: field.FTBool,
			},
			&mapping.FieldDescr{
				Type: field.FTInt8,
			},
			&mapping.FieldDescr{
				Type: field.FTUint64,
			},
			&mapping.FieldDescr{
				Type: field.FTFloat32,
			},
		},
	}

	s := New(0, m)

	if err := SetNumber[int8](s, 2, 10); err != nil {
		panic(err)
	}
	if err := SetNumber(s, 3, uint64(math.MaxUint32)+1); err != nil {
		panic(err)
	}
	if err := SetNumber[float32](s, 4, 3.2); err != nil {
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
			desc:     "Error: fieldNum is greater that possible fields",
			s:        s,
			fieldNum: 29,
			err:      true,
		},
		{
			desc:     "Error: fieldNum is not a number",
			s:        s,
			fieldNum: 1, // FTBool
			err:      true,
		},
		{
			desc:     "fieldNum that has a nil value and should return 0",
			s:        s,
			fieldNum: 0,
			want:     uint8(0),
		},
		{
			desc:     "fieldNum that is set to 10",
			s:        s,
			fieldNum: 2,
			want:     int8(10),
		},
		{
			desc:     "fieldNum that is set to math.MaxUint32+1",
			s:        s,
			fieldNum: 3,
			want:     uint64(math.MaxUint32) + 1,
		},
		{
			desc:     "fieldNum that is set to a float",
			s:        s,
			fieldNum: 4,
			want:     float32(3.2),
		},
	}
	for _, test := range tests {
		var got any
		var err error

		// We can't switch on types for either field 0 or fields not in our mapping.Map, but
		// we still want to test our error conditions.
		if test.fieldNum >= uint16(len(m.Fields)) {
			got, err = GetNumber[uint8](test.s, test.fieldNum)
		} else { // Any other tests
			switch m.Fields[test.fieldNum].Type {
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
		desc        mapping.FieldDescr
		wantSize    uint8
		wantIsFloat bool
		wantErr     bool
	}{
		{uint8(1), mapping.FieldDescr{Type: field.FTUint8}, 8, false, false},
		{uint16(1), mapping.FieldDescr{Type: field.FTUint16}, 16, false, false},
		{uint32(1), mapping.FieldDescr{Type: field.FTUint32}, 32, false, false},
		{uint64(1), mapping.FieldDescr{Type: field.FTUint64}, 64, false, false},
		{int8(1), mapping.FieldDescr{Type: field.FTInt8}, 8, false, false},
		{int16(1), mapping.FieldDescr{Type: field.FTInt16}, 16, false, false},
		{int32(1), mapping.FieldDescr{Type: field.FTInt32}, 32, false, false},
		{int64(1), mapping.FieldDescr{Type: field.FTInt64}, 64, false, false},
		{float32(1), mapping.FieldDescr{Type: field.FTFloat32}, 32, true, false},
		{float64(1), mapping.FieldDescr{Type: field.FTFloat64}, 64, true, false},
		// Cause an error.
		{uint8(1), mapping.FieldDescr{Type: field.FTUint16}, 8, false, true},
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
