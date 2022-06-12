package claw

import (
	"reflect"
	"unsafe"

	"github.com/bearlytools/claw/internal/field"
)

const (
	_1MiB = 1048576
)

// FieldType represents the type of data that is held in a byte field.
type FieldType = field.Type

const (
	FTUnknown    = field.FTUnknown
	FTBool       = field.FTBool
	FTInt8       = field.FTInt8
	FTInt16      = field.FTInt16
	FTInt32      = field.FTInt32
	FTInt64      = field.FTInt64
	FTUint8      = field.FTUint8
	FTUint16     = field.FTUint16
	FTUint32     = field.FTUint32
	FTUint64     = field.FTUint64
	FTFloat32    = field.FTFloat32
	FTFloat64    = field.FTFloat64
	FTString     = field.FTString
	FTBytes      = field.FTBytes
	FTStruct     = field.FTStruct
	FTListBool   = field.FTListBool
	FTList8      = field.FTList8
	FTList16     = field.FTList16
	FTList32     = field.FTList32
	FTList64     = field.FTList64
	FTListBytes  = field.FTListBytes
	FTListStruct = field.FTListStruct
)

func byteSlice2String(bs []byte) string {
	return *(*string)(unsafe.Pointer(&bs))
}

func unsafeGetBytes(s string) []byte {
	return (*[0x7fff0000]byte)(unsafe.Pointer(
		(*reflect.StringHeader)(unsafe.Pointer(&s)).Data),
	)[:len(s):len(s)]
}
