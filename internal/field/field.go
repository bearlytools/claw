package field

//go:generate stringer -type=Type

// Type represents the type of data that is held in a byte field.
type Type uint8

const (
	FTUnknown Type = 0
	FTBool    Type = 1
	FTInt8    Type = 2
	FTInt16   Type = 3
	FTInt32   Type = 4
	FTInt64   Type = 5
	FTUint8   Type = 6
	FTUint16  Type = 7
	FTUint32  Type = 8
	FTUint64  Type = 9
	FTFloat32 Type = 10
	FTFloat64 Type = 11
	FTString  Type = 12
	FTBytes   Type = 13
	FTStruct  Type = 14
	// Reserve 15 to 40
	FTListBools   Type = 41
	FTListInt8    Type = 42
	FTListInt16   Type = 43
	FTListInt32   Type = 44
	FTListInt64   Type = 45
	FTListUint8   Type = 46
	FTListUint16  Type = 47
	FTListUint32  Type = 48
	FTListUint64  Type = 49
	FTListFloat32 Type = 50
	FTListFloat64 Type = 51
	FTListBytes   Type = 52
	FTListStrings Type = 53
	FTListStructs Type = 54
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
