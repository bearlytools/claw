package reflect

import (
	"github.com/bearlytools/claw/languages/go/reflect/internal/value"
)

// Number represents all int, uint and float types.
type Number value.Number

// Value holds a Claw value type for use in reflection.
type Value = value.Value

// ValueOfBool will return the value v as a Value type.
func ValueOfBool(v bool) Value {
	return ValueOfBool(v)
}

// ValueOfBytes will return the value v as a Value type.
func ValueOfBytes(v []byte) Value {
	return ValueOfBytes(v)
}

// ValueOfString will return the value v as a Value type.
func ValueOfString(v string) Value {
	return ValueOfString(v)
}

// ValueOfEnum will return the value v as a Value type.
func ValueOfEnum[N uint8 | uint16](v N) Value {
	return ValueOfEnum[N](v)
}

// ValueOfNumber will return the value v as a Value type.
func ValueOfNumber[N Number](v N) Value {
	return ValueOfNumber[N](v)
}

/*
func ValueOfList(v List) Value {

}
func ValueOfStruct(v Struct) Value {

}
*/
