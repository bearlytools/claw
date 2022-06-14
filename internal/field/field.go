package field

//go:generate stringer -type=Type

// Type represents the type of data that is held in a byte field.
type Type uint8

const (
	FTUnknown    Type = 0
	FTBool       Type = 1
	FTInt8       Type = 2
	FTInt16      Type = 3
	FTInt32      Type = 4
	FTInt64      Type = 5
	FTUint8      Type = 6
	FTUint16     Type = 7
	FTUint32     Type = 8
	FTUint64     Type = 9
	FTFloat32    Type = 10
	FTFloat64    Type = 11
	FTString     Type = 12
	FTBytes      Type = 13
	FTStruct     Type = 14
	FTListBool   Type = 15
	FTList8      Type = 16
	FTList16     Type = 17
	FTList32     Type = 18
	FTList64     Type = 19
	FTListBytes  Type = 20
	FTListStruct Type = 21
)

// IsList determines if a Type represents a list of entries.
func IsList(ft Type) bool {
	switch ft {
	case FTListBool, FTList8, FTList16, FTList32, FTList64, FTListBytes, FTListStruct:
		return true
	}
	return false
}
