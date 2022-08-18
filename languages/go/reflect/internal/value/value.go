package value

import (
	"math"
	"unsafe"

	"github.com/bearlytools/claw/internal/binary"
	"github.com/bearlytools/claw/internal/bits"
	"github.com/bearlytools/claw/languages/go/field"
	"github.com/bearlytools/claw/languages/go/internal/pragma"
	"github.com/bearlytools/claw/languages/go/structs"
	"github.com/bearlytools/claw/languages/go/structs/header"
)

type doNotImplement pragma.DoNotImplement

var boolMask = bits.Mask[uint64](24, 25)

// Value represents a read-only Claw value. This can be used to retrieve a value or
// set a value.
type Value struct {
	// h works like normal for all non-list and non-aStruct types. In those cases,
	// list and aStruct should be referenced.
	h      header.Generic
	ptr    unsafe.Pointer
	isEnum bool

	list    List
	aStruct Struct
}

// Bool returns the boolean value stored in Value. If Value is not a bool type, this will panic.
func (v Value) Bool() bool {
	if v.h == nil {
		panic("Value is empty value")
	}
	if v.h.FieldType() != field.FTBool {
		panic("Value is not Bool, was " + v.h.FieldType().String())
	}
	i := binary.Get[uint64](v.h)
	return bits.GetValue[uint64, uint8](i, boolMask, 24) == 1
}

// Bytes returns the Bytes value stored in Value. If Value is not a Bytes type, this will panic.
func (v Value) Bytes() []byte {
	if v.h == nil {
		panic("Value is empty value")
	}
	if v.h.FieldType() != field.FTBytes {
		panic("Value is not Bytes, was " + v.h.FieldType().String())
	}

	if v.ptr == nil { // Set, but value is empty
		return nil
	}

	return *(*[]byte)(v.ptr)
}

// Enum returns the enumerated value stored in Value. If Value is not an Enum type, this will panic.
func (v Value) Enum() Enum {
	panic("not implemented")
}

// Float returns the Float value stored in Value. If Value is not a Float type, this will panic.
func (v Value) Float() float64 {
	if v.h == nil {
		panic("Value is empty value")
	}

	switch v.h.FieldType() {
	case field.FTFloat32:
		f, err := getNumber[float32](v, true)
		if err != nil {
			panic(err)
		}
		return float64(f)
	case field.FTFloat64:
		f, err := getNumber[float64](v, true)
		if err != nil {
			panic(err)
		}
		return f
	}
	panic("field type was not for a float32 or float64, was " + v.h.FieldType().String())
}

// Int returns the integer value stored in Value. If Value is not an integer type, this will panic.
func (v Value) Int() int64 {
	if v.h == nil {
		panic("Value is empty value")
	}

	switch v.h.FieldType() {
	case field.FTInt8:
		i, err := getNumber[int8](v, false)
		if err != nil {
			panic(err)
		}
		return int64(i)
	case field.FTInt16:
		i, err := getNumber[int16](v, false)
		if err != nil {
			panic(err)
		}
		return int64(i)
	case field.FTInt32:
		i, err := getNumber[int32](v, false)
		if err != nil {
			panic(err)
		}
		return int64(i)
	case field.FTInt64:
		i, err := getNumber[int64](v, false)
		if err != nil {
			panic(err)
		}
		return i
	}
	panic("field type was not for a int8, int16, int32 or int64, was " + v.h.FieldType().String())
}

// Any decodes the value into the any type. If the value isn't valid, this panics.
func (v Value) Any() any {
	panic("not all types implemented yet")
	if v.h == nil {
		return nil
	}

	if v.isEnum {
		return v.Enum()
	}

	switch v.h.FieldType() {
	case field.FTBool:
		return v.Bool()
	case field.FTInt8:
		return int8(v.Int())
	case field.FTInt16:
		return int16(v.Int())
	case field.FTInt32:
		return int32(v.Int())
	case field.FTInt64:
		return v.Int()
	case field.FTUint8:
		return uint8(v.Uint())
	case field.FTUint16:
		return uint16(v.Uint())
	case field.FTUint32:
		return uint32(v.Uint())
	case field.FTUint64:
		return v.Uint()
	case field.FTFloat32:
		return float32(v.Float())
	case field.FTFloat64:
		return v.Float()
	case field.FTString:
		return v.String()
	case field.FTBytes:
		return v.Bytes()
	default:
		panic("eh")
	}
}

// String returns the string value stored in Value. If Value is not a string type, this will panic.
func (v Value) String() string {
	if v.h == nil {
		panic("Value is empty value")
	}

	x := (*[]byte)(v.ptr)
	return string(*x)
}

// Uint returns the unsigned integer value stored in Value. If Value is not an unsigned integer type, this will panic.
func (v Value) Uint() uint64 {
	if v.h == nil {
		panic("Value is empty value")
	}

	switch v.h.FieldType() {
	case field.FTUint8:
		i, err := getNumber[uint8](v, false)
		if err != nil {
			panic(err)
		}
		return uint64(i)
	case field.FTUint16:
		i, err := getNumber[uint16](v, false)
		if err != nil {
			panic(err)
		}
		return uint64(i)
	case field.FTUint32:
		i, err := getNumber[uint32](v, false)
		if err != nil {
			panic(err)
		}
		return uint64(i)
	case field.FTUint64:
		i, err := getNumber[uint64](v, false)
		if err != nil {
			panic(err)
		}
		return i
	}
	panic("field type was not for a uint8, uint16, uint32 or uint64, was " + v.h.FieldType().String())
}

// List returns the List value stored in Value. If Value is not some list type, this will panic.
func (v Value) List() List {
	if v.list == nil {
		panic("type is not a list type")
	}
	return v.list
}

// Struct returns the Struct value stored in Value. If Value is not a Struct type, this will panic.
func (v Value) Struct() Struct {
	if v.aStruct == nil {
		panic("type is not a struct type")
	}

	return v.aStruct
}

// getNumber gets a number value at fieldNum.
func getNumber[N Number](v Value, isFloat bool) (N, error) {
	if v.ptr == nil {
		b := v.h[3:8]
		if isFloat {
			i := binary.Get[uint32](b)
			return N(math.Float32frombits(uint32(i))), nil
		}
		return N(binary.Get[uint32](b)), nil
	}
	b := *(*[]byte)(v.ptr)
	if isFloat {
		i := binary.Get[uint64](b)
		return N(math.Float64frombits(uint64(i))), nil
	}
	return N(binary.Get[uint64](b)), nil
}

// XXXNewStruct wraps our internal *structs.Struct objects in the reflect.Struct type.
// This is used in our generated code to implement the ClawStruct() method.
func XXXNewStruct(v *structs.Struct) Struct {
	descr := NewStructDescrImpl(v.Map())
	return StructImpl{s: v, descr: descr}
}
