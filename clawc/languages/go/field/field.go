// Package field details field types used by the Claw format.
package field

import "fmt"

//go:generate stringer -type=Type -linecomment

// Type represents the type of data that is held in a byte field.
type Type uint8

const (
	FTUnknown Type = 0  // Unknown
	FTBool    Type = 1  // bool
	FTInt8    Type = 2  // int8
	FTInt16   Type = 3  // int16
	FTInt32   Type = 4  // int32
	FTInt64   Type = 5  // int64
	FTUint8   Type = 6  // uint8
	FTUint16  Type = 7  // uint16
	FTUint32  Type = 8  // uint32
	FTUint64  Type = 9  // uint64
	FTFloat32 Type = 10 // float32
	FTFloat64 Type = 11 // float64
	FTString  Type = 12 // string
	FTBytes   Type = 13 // bytes
	FTStruct  Type = 14 // struct
	// Reserve 15 to 40
	FTListBools   Type = 41 // []bool
	FTListInt8    Type = 42 // []int8
	FTListInt16   Type = 43 // []int16
	FTListInt32   Type = 44 // []int32
	FTListInt64   Type = 45 // []int64
	FTListUint8   Type = 46 // []uint8
	FTListUint16  Type = 47 // []uint16
	FTListUint32  Type = 48 // []uint32
	FTListUint64  Type = 49 // []uint64
	FTListFloat32 Type = 50 // []float32
	FTListFloat64 Type = 51 // []float64
	FTListBytes   Type = 52 // []bytes
	FTListStrings Type = 53 // []string
	FTListStructs Type = 54 // []structs
	// Reserve 55 to 79
)

// IsList determines if a Type represents a list of entries.
func IsList(ft Type) bool {
	if ft > 14 && ft < 29 {
		return true
	}
	return false
}

// NumberTypes is a list of field types that represent a number.
var NumberTypes = []Type{
	FTInt8,
	FTInt16,
	FTInt32,
	FTInt64,
	FTUint8,
	FTUint16,
	FTUint32,
	FTUint64,
	FTFloat32,
	FTFloat64,
}

// ListTypes is a list of field types that represent a list.
var ListTypes = []Type{
	FTListBools,
	FTListInt8,
	FTListInt16,
	FTListInt32,
	FTListInt64,
	FTListUint8,
	FTListUint16,
	FTListUint32,
	FTListUint64,
	FTListFloat32,
	FTListFloat64,
	FTListBytes,
	FTListStrings,
	FTListStructs,
}

// NumericListTypes is a list of field types that represent a number.
var NumericListTypes = []Type{
	FTListInt8,
	FTListInt16,
	FTListInt32,
	FTListInt64,
	FTListUint8,
	FTListUint16,
	FTListUint32,
	FTListUint64,
	FTListFloat32,
	FTListFloat64,
}

// TypeToString returns the type as a string WITHOUT the leading "FT".
func TypeToString(t Type) string {
	return t.String()[2:]
}

// GoType will return the Go string representation of a type.
// So a FTListUint64 will return "[]uint64".  If the type isn't
// based on a basic Go type, this will panic. So you can't do FTListStruct.
func GoType(t Type) string {
	switch t {
	case FTBool:
		return "bool"
	case FTInt8:
		return "int8"
	case FTInt16:
		return "int16"
	case FTInt32:
		return "int32"
	case FTInt64:
		return "int64"
	case FTUint8:
		return "uint8"
	case FTUint16:
		return "uint16"
	case FTUint32:
		return "uint32"
	case FTUint64:
		return "uint64"
	case FTFloat32:
		return "float32"
	case FTFloat64:
		return "float64"
	case FTString:
		return "string"
	case FTBytes:
		return "[]bytes"
	case FTListBools:
		return "[]bool"
	case FTListInt8:
		return "[]int8"
	case FTListInt16:
		return "[]int16"
	case FTListInt32:
		return "[]int32"
	case FTListInt64:
		return "[]int64"
	case FTListUint8:
		return "[]uint8"
	case FTListUint16:
		return "[]uint16"
	case FTListUint32:
		return "[]uint32"
	case FTListUint64:
		return "[]uint64"
	case FTListFloat32:
		return "[]float32"
	case FTListFloat64:
		return "[]float64"
	case FTListBytes:
		return "[][]byte"
	case FTListStrings:
		return "[]string"
	}
	panic(fmt.Sprintf("unsupported type: %T", t))
}
