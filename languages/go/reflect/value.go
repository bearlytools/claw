package reflect

import (
	"fmt"

	"github.com/bearlytools/claw/internal/conversions"
	"github.com/bearlytools/claw/languages/go/reflect/internal/interfaces"
	"github.com/bearlytools/claw/languages/go/reflect/internal/value"
	"github.com/bearlytools/claw/languages/go/structs"
)

// Number represents all int, uint and float types.
type Number interfaces.Number

// Value holds a Claw value type for use in reflection.
type Value = interfaces.Value

// ValueOfBool will return the value v as a Value type.
func ValueOfBool(v bool) Value {
	return value.ValueOfBool(v)
}

// ValueOfBytes will return the value v as a Value type.
func ValueOfBytes(v []byte) Value {
	return value.ValueOfBytes(v)
}

// ValueOfString will return the value v as a Value type.
func ValueOfString(v string) Value {
	return value.ValueOfString(v)
}

// ValueOfEnum will return the value v as a Value type.
func ValueOfEnum[N uint8 | uint16](v N, enumGroup EnumGroup) Value {
	return value.ValueOfEnum(v, enumGroup)
}

// ValueOfNumber will return the value v as a Value type.
func ValueOfNumber[N Number](v N) Value {
	return value.ValueOfNumber(v)
}

// ValueOfList will return the value v as a Value type.
func ValueOfList(v List) Value {
	return value.ValueOfList(v)
}

// ValueOfStruct will return the value v as a Value type.
func ValueOfStruct(v Struct) Value {
	return value.ValueOfStruct(v)
}

// ListFrom will create a List type from the following types:
// []bool, []int*, []uint*, []float*, []string, [][]byte.
// If it is not one of these, it will panic.
func ListFrom(v any) List {
	switch t := v.(type) {
	case []bool:
		b := structs.NewBools(0)
		b.Append(t...)
		return value.NewListBools(b)
	case []int8:
		n := structs.NewNumbers[int8]()
		n.Append(t...)
		return value.NewListNumbers(n)
	case []int16:
		n := structs.NewNumbers[int16]()
		n.Append(t...)
		return value.NewListNumbers(n)
	case []int32:
		n := structs.NewNumbers[int32]()
		n.Append(t...)
		return value.NewListNumbers(n)
	case []int64:
		n := structs.NewNumbers[int64]()
		n.Append(t...)
		return value.NewListNumbers(n)
	case []uint8:
		n := structs.NewNumbers[uint8]()
		n.Append(t...)
		return value.NewListNumbers(n)
	case []uint16:
		n := structs.NewNumbers[uint16]()
		n.Append(t...)
		return value.NewListNumbers(n)
	case []uint32:
		n := structs.NewNumbers[uint32]()
		n.Append(t...)
		return value.NewListNumbers(n)
	case []uint64:
		n := structs.NewNumbers[uint64]()
		n.Append(t...)
		return value.NewListNumbers(n)
	case []float32:
		n := structs.NewNumbers[float32]()
		n.Append(t...)
		return value.NewListNumbers(n)
	case []float64:
		n := structs.NewNumbers[float64]()
		n.Append(t...)
		return value.NewListNumbers(n)
	case []string:
		b := structs.NewBytes()
		l := make([][]byte, 0, len(t))
		for _, s := range t {
			l = append(l, conversions.UnsafeGetBytes(s))
		}
		b.Append(l...)
		return value.NewListStrings(b)
	case [][]byte:
		b := structs.NewBytes()
		b.Append(t...)
		return value.NewListBytes(b)
	}
	panic(fmt.Sprintf("%T is not supported", v))
}
