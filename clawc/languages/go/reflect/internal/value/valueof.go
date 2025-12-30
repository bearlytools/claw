package value

import (
	"github.com/bearlytools/claw/clawc/languages/go/field"
	"github.com/bearlytools/claw/clawc/languages/go/reflect/internal/interfaces"
)

// ValueOfBool returns a Value that represents bool.
func ValueOfBool(v bool) Value {
	return Value{
		ft:        field.FTBool,
		boolVal:   v,
		hasScalar: true,
	}
}

// ValueOfBytes returns a Value that represents []byte. Can be nil.
func ValueOfBytes(v []byte) Value {
	return Value{
		ft:       field.FTBytes,
		bytesVal: v,
	}
}

// ValueOfString returns a Value that represents a string.
func ValueOfString(v string) Value {
	return Value{
		ft:        field.FTString,
		stringVal: v,
	}
}

// ValueOfEnum returns a Value that represents an enumerator.
func ValueOfEnum[N ~uint8 | ~uint16](v N, enumGroup interfaces.EnumGroup) Value {
	val := Value{
		isEnum:    true,
		enumGroup: enumGroup,
		uintVal:   uint64(v),
		hasScalar: true,
	}
	// Set the field type based on the enum size
	switch any(v).(type) {
	case uint8:
		val.ft = field.FTUint8
	case uint16:
		val.ft = field.FTUint16
	}
	return val
}

// ValueOfNumber returns a Value representing a number type.
func ValueOfNumber[N interfaces.Number](v N) Value {
	val := Value{hasScalar: true}

	switch any(v).(type) {
	case int8:
		val.ft = field.FTInt8
		val.intVal = int64(v)
	case int16:
		val.ft = field.FTInt16
		val.intVal = int64(v)
	case int32:
		val.ft = field.FTInt32
		val.intVal = int64(v)
	case int64:
		val.ft = field.FTInt64
		val.intVal = int64(v)
	case uint8:
		val.ft = field.FTUint8
		val.uintVal = uint64(v)
	case uint16:
		val.ft = field.FTUint16
		val.uintVal = uint64(v)
	case uint32:
		val.ft = field.FTUint32
		val.uintVal = uint64(v)
	case uint64:
		val.ft = field.FTUint64
		val.uintVal = uint64(v)
	case float32:
		val.ft = field.FTFloat32
		val.floatVal = float64(v)
	case float64:
		val.ft = field.FTFloat64
		val.floatVal = float64(v)
	}
	return val
}

// ValueOfList returns a Value that represents List.
func ValueOfList(v interfaces.List) Value {
	return Value{
		list: v,
	}
}

// ValueOfStruct returns a Value that represents a Struct.
func ValueOfStruct(v interfaces.Struct) Value {
	if v == nil {
		panic("v cannot be nil")
	}
	return Value{
		ft:      field.FTStruct,
		aStruct: v,
	}
}
