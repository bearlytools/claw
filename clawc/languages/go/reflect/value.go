package reflect

import (
	"fmt"

	"github.com/bearlytools/claw/clawc/languages/go/reflect/internal/interfaces"
	"github.com/bearlytools/claw/clawc/languages/go/reflect/internal/value"
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
func ValueOfEnum[N ~uint8 | ~uint16](v N, enumGroup EnumGroup) Value {
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
		return value.NewListBools(t)
	case []int8:
		return value.NewListNumbers(t)
	case []int16:
		return value.NewListNumbers(t)
	case []int32:
		return value.NewListNumbers(t)
	case []int64:
		return value.NewListNumbers(t)
	case []uint8:
		return value.NewListNumbers(t)
	case []uint16:
		return value.NewListNumbers(t)
	case []uint32:
		return value.NewListNumbers(t)
	case []uint64:
		return value.NewListNumbers(t)
	case []float32:
		return value.NewListNumbers(t)
	case []float64:
		return value.NewListNumbers(t)
	case []string:
		return value.NewListStrings(t)
	case [][]byte:
		return value.NewListBytes(t)
	}
	panic(fmt.Sprintf("%T is not supported", v))
}
