package value

import (
	"bytes"
	"fmt"

	"github.com/bearlytools/claw/clawc/internal/pragma"
	"github.com/bearlytools/claw/clawc/languages/go/field"
	"github.com/bearlytools/claw/clawc/languages/go/reflect/internal/interfaces"
	"github.com/bearlytools/claw/clawc/languages/go/segment"
)

type doNotImplement pragma.DoNotImplement

// Value represents a read-only Claw value. This can be used to retrieve a value or
// set a value.
type Value struct {
	// ft is the field type
	ft field.Type

	// hasScalar indicates if this Value contains a scalar value
	hasScalar bool

	// Scalar value storage (only one is used based on ft)
	boolVal  bool
	intVal   int64
	uintVal  uint64
	floatVal float64

	// Non-scalar storage
	bytesVal  []byte
	stringVal string

	// Enum support
	isEnum    bool
	enumGroup interfaces.EnumGroup

	// Complex types
	list    interfaces.List
	aStruct interfaces.Struct
}

// Bool returns the boolean value stored in Value. If Value is not a bool type, this will panic.
func (v Value) Bool() bool {
	if v.ft != field.FTBool {
		panic("Value is not Bool, was " + v.ft.String())
	}
	return v.boolVal
}

// Bytes returns the Bytes value stored in Value. If Value is not a Bytes type, this will panic.
func (v Value) Bytes() []byte {
	if v.ft != field.FTBytes {
		panic("Value is not Bytes, was " + v.ft.String())
	}
	return v.bytesVal
}

// Enum returns the enumerated value stored in Value. If Value is not an Enum type, this will panic.
func (v Value) Enum() interfaces.Enum {
	if !v.isEnum {
		panic("Enum() called on non enum value")
	}
	return v.enumGroup.ByValue(uint16(v.uintVal))
}

// Float returns the Float value stored in Value. If Value is not a Float type, this will panic.
func (v Value) Float() float64 {
	switch v.ft {
	case field.FTFloat32, field.FTFloat64:
		return v.floatVal
	}
	panic("field type was not for a float32 or float64, was " + v.ft.String())
}

// Int returns the integer value stored in Value. If Value is not an integer type, this will panic.
func (v Value) Int() int64 {
	switch v.ft {
	case field.FTInt8, field.FTInt16, field.FTInt32, field.FTInt64:
		return v.intVal
	}
	panic("field type was not for an int8, int16, int32 or int64, was " + v.ft.String())
}

// Any decodes the value into the any type. If the value isn't valid, this panics.
// Here are a list of the decodes:
//
//   - Enumerators will decode into the interfaces.Enum type
//   - Bool will decode to the bool type
//   - Numbers(int*, uint*, float*) decode into their Go equivalent
//   - String into the string type
//   - Bytes into the []byte type
//   - Struct into reflect.Struct
//   - List of bools into []bool
//   - List of Numbers into []<go number type>
//   - List of String into []string
//   - List of Bytes into [][]byte
//   - List of Struct into a []reflect.Struct
func (v Value) Any() any {
	if !v.hasScalar && v.aStruct == nil && v.enumGroup == nil && v.list == nil && v.bytesVal == nil && v.stringVal == "" {
		return nil
	}

	if v.isEnum {
		return v.Enum()
	}

	if v.aStruct != nil {
		return v.aStruct
	}

	switch v.ft {
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
		return v.stringVal
	case field.FTBytes:
		return v.Bytes()
	case field.FTListBools:
		return boolSliceFromValue(v)
	case field.FTListInt8:
		return numberSliceFromValue[int8](v)
	case field.FTListInt16:
		return numberSliceFromValue[int16](v)
	case field.FTListInt32:
		return numberSliceFromValue[int32](v)
	case field.FTListInt64:
		return numberSliceFromValue[int64](v)
	case field.FTListUint8:
		return numberSliceFromValue[uint8](v)
	case field.FTListUint16:
		return numberSliceFromValue[uint16](v)
	case field.FTListUint32:
		return numberSliceFromValue[uint32](v)
	case field.FTListUint64:
		return numberSliceFromValue[uint64](v)
	case field.FTListFloat32:
		return numberSliceFromValue[float32](v)
	case field.FTListFloat64:
		return numberSliceFromValue[float64](v)
	case field.FTListBytes:
		return bytesSliceFromValue(v)
	case field.FTListStrings:
		return stringSliceFromValue(v)
	case field.FTListStructs:
		return structSliceFromValue(v)
	}

	panic(fmt.Sprintf("unsupported type %v", v.ft))
}

// String returns the string value stored in Value. String returns the string v's underlying
// value, as a string. String is a special case because of Go's String method convention.
// Unlike the other getters, it does not panic if v's Kind is not String. Instead, it returns
// a string of the form "<T value>" where T is v's type. The fmt package treats Values
// specially. It does not call their String method implicitly but instead prints the
// concrete values they hold.
func (v Value) String() string {
	if v.ft == field.FTUnknown && !v.hasScalar && v.aStruct == nil && v.list == nil {
		return "<invalid Value>"
	}

	// If it is an enumerator, print the enumerator name.
	if v.isEnum {
		return v.Enum().Name()
	}

	// String type returns directly
	if v.ft == field.FTString {
		return v.stringVal
	}

	// There are two types that require special attention, Struct and []Struct.
	switch v.ft {
	case field.FTStruct:
		buff := bytes.Buffer{}
		buff.WriteRune('{')
		start := true
		v.aStruct.Range(
			func(fd interfaces.FieldDescr, v interfaces.Value) bool {
				if !start {
					buff.WriteString(", ")
				}
				buff.WriteString(fmt.Sprintf("%s: %s", fd.Name(), v.String()))
				start = false
				return true
			},
		)
		buff.WriteRune('}')
		return buff.String()
	case field.FTListStructs:
		buff := bytes.Buffer{}
		buff.WriteRune('[')

		for i := 0; i < v.list.Len(); i++ {
			if i != 0 {
				buff.WriteString(", ")
			}
			buff.WriteString(v.list.Get(i).String())
		}

		buff.WriteRune(']')
		return buff.String()
	}

	// For everything else, convert to the Go type and let Go's fmt.Sprint() handle the
	// string conversion.
	return fmt.Sprint(v.Any())
}

// Uint returns the unsigned integer value stored in Value. If Value is not an unsigned integer type, this will panic.
func (v Value) Uint() uint64 {
	switch v.ft {
	case field.FTUint8, field.FTUint16, field.FTUint32, field.FTUint64:
		return v.uintVal
	}
	panic("field type was not for a uint8, uint16, uint32 or uint64, was " + v.ft.String())
}

// List returns the List value stored in Value. If Value is not some list type, this will panic.
func (v Value) List() interfaces.List {
	if v.list == nil {
		panic("type is not a list type")
	}
	return v.list
}

// Struct returns the Struct value stored in Value. If Value is not a Struct type, this will panic.
func (v Value) Struct() interfaces.Struct {
	if v.aStruct == nil {
		panic("type is not a struct type")
	}

	return v.aStruct
}

func boolSliceFromValue(v Value) []bool {
	if v.list.Len() == 0 {
		return nil
	}

	x := make([]bool, v.list.Len())
	for i := 0; i < v.list.Len(); i++ {
		x[i] = v.list.Get(i).Bool()
	}
	return x
}

func numberSliceFromValue[N interfaces.Number](v Value) []N {
	var a N

	x := make([]N, v.list.Len())

	for i := 0; i < v.list.Len(); i++ {
		switch any(a).(type) {
		case int8, int16, int32, int64:
			x[i] = N(v.list.Get(i).Int())
		case uint8, uint16, uint32, uint64:
			x[i] = N(v.list.Get(i).Uint())
		case float32, float64:
			x[i] = N(v.list.Get(i).Float())
		default:
			panic(fmt.Sprintf("unsupported type %T", a))
		}
	}
	return x
}

func bytesSliceFromValue(v Value) [][]byte {
	if v.list.Len() == 0 {
		return nil
	}

	x := make([][]byte, v.list.Len())
	for i := 0; i < v.list.Len(); i++ {
		x[i] = v.list.Get(i).Bytes()
	}
	return x
}

func stringSliceFromValue(v Value) []string {
	if v.list.Len() == 0 {
		return nil
	}

	x := make([]string, v.list.Len())
	for i := 0; i < v.list.Len(); i++ {
		x[i] = v.list.Get(i).String()
	}
	return x
}

func structSliceFromValue(v Value) []interfaces.Struct {
	if v.list.Len() == 0 {
		return nil
	}

	x := make([]interfaces.Struct, v.list.Len())
	for i := 0; i < v.list.Len(); i++ {
		x[i] = v.list.Get(i).Struct()
	}
	return x
}

// XXXNewStruct wraps our internal *segment.Struct objects in the reflect.Struct type.
// This is used in our generated code to implement the ClawStruct() method.
func XXXNewStruct(v *segment.Struct, descr interfaces.StructDescr) interfaces.Struct {
	return StructImpl{s: v, descr: descr}
}
