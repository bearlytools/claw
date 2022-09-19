package value

import (
	"fmt"
	"math"
	"unsafe"

	"github.com/bearlytools/claw/internal/binary"
	"github.com/bearlytools/claw/internal/bits"
	"github.com/bearlytools/claw/internal/conversions"
	"github.com/bearlytools/claw/languages/go/field"
	"github.com/bearlytools/claw/languages/go/reflect/internal/interfaces"
	"github.com/bearlytools/claw/languages/go/structs"
)

const maxDataSize = 1099511627775

// ValueOfBool returns a Value that represents bool.
func ValueOfBool(v bool) Value {
	h := structs.NewGenericHeader()
	h.SetFieldType(field.FTBool)
	n := conversions.BytesToNum[uint64](h)
	*n = bits.SetBit(*n, 24, v)
	return Value{h: h}
}

// ValueOfBytes returns a Value that represents []byte. Cannot be nil.
func ValueOfBytes(v []byte) Value {
	return valueOfBytes(v, false)
}

// ValueOfString returns a Value that represents a string. A string cannot be empty.
func ValueOfString(v string) Value {
	return valueOfBytes(conversions.UnsafeGetBytes(v), true)
}

// valueOfBytes returns a value representing a []byte. v cannot be nil or have 0 length
// or this will panic. Also, cannot be larger than size 1099511627775.
func valueOfBytes(v []byte, isString bool) Value {
	if len(v) == 0 {
		panic("cannot encode an empty Bytes value")
	}

	if len(v) > maxDataSize {
		panic("cannot set a String or Byte field to size > 1099511627775")
	}

	ftype := field.FTBytes
	if isString {
		ftype = field.FTString
	}

	h := structs.NewGenericHeader()
	h.SetFieldType(ftype)
	h.SetFinal40(uint64(len(v)))

	return Value{h: h, ptr: unsafe.Pointer(&v)}
}

// ValueOfEnum returns a Value that represents an enumerator.
func ValueOfEnum[N ~uint8 | ~uint16](v N, enumGroup interfaces.EnumGroup) Value {
	e := numberValue(v)
	e.isEnum = true
	e.enumGroup = enumGroup
	return e
}

// value.ValueOfNumber will return Value representing a number type.
func ValueOfNumber[N interfaces.Number](v N) Value {
	return numberValue(v)
}

func numberValue[N interfaces.Number](v N) Value {
	size := 0
	ft := field.FTUnknown
	isFloat := false
	switch any(v).(type) {
	case uint8:
		size = 8
		ft = field.FTUint8
	case int8:
		size = 8
		ft = field.FTInt8
	case uint16:
		size = 16
		ft = field.FTUint16
	case int16:
		size = 16
		ft = field.FTInt16
	case uint32:
		size = 32
		ft = field.FTUint32
	case int32:
		size = 32
		ft = field.FTInt32
	case uint64:
		size = 64
		ft = field.FTUint64
	case int64:
		size = 64
		ft = field.FTInt64
	case float32:
		size = 32
		ft = field.FTFloat32
		isFloat = true
	case float64:
		size = 64
		ft = field.FTFloat64
		isFloat = true
	default:
		panic(fmt.Sprintf("unsupported number type: %T", v))
	}

	// Convert value to uint64.
	var i uint64
	if isFloat {
		if size == 64 {
			i = math.Float64bits(float64(v))
		} else {
			i = uint64(math.Float32bits(float32(v)))
		}
	} else {
		i = uint64(v)
	}

	h := structs.NewGenericHeader()
	h.SetFieldType(ft)
	if size == 64 {
		b := make([]byte, 8)
		binary.Put(b, i)
		return Value{h: h, ptr: unsafe.Pointer(&b)}
	}
	h.SetFinal40(i)
	return Value{h: h}
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
		aStruct: v,
	}
}
