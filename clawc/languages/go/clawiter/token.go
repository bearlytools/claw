// Package clawiter provides types for streaming iteration over Claw structs.
// This enables serialization to various formats (JSON, XML, etc.) without
// format-specific code in each struct.
package clawiter

import (
	"math"
	"unsafe"

	"github.com/bearlytools/claw/clawc/languages/go/field"
)

// TokenKind represents the type of token in the walk stream.
type TokenKind uint8

const (
	TokenStructStart TokenKind = iota // Beginning of a struct
	TokenStructEnd                    // End of a struct
	TokenField                        // A field (scalar or announces complex type)
	TokenListStart                    // Beginning of a list
	TokenListEnd                      // End of a list
	TokenMapStart                     // Beginning of a map
	TokenMapEnd                       // End of a map
	TokenMapEntry                     // A key-value pair in a map
)

// Token represents a single event in the walk stream.
type Token struct {
	// Kind is the type of token.
	Kind TokenKind
	// Name is the struct name (for Start/End) or field name (for Field).
	Name string
	// Type is the field type (for TokenField and TokenListStart).
	Type field.Type

	// data stores scalar values inline (all fit in 64 bits).
	data uint64
	// Bytes stores string and byte slice data. Use String() for zero-copy string conversion.
	Bytes []byte

	// IsEnum indicates if this is an enum value.
	IsEnum bool
	// EnumGroup is the name of the enum group (e.g., "Type").
	EnumGroup string
	// EnumName is the string name of the enum value (e.g., "Car").
	EnumName string

	// StructName is the struct type name for FTStruct/FTListStructs.
	StructName string
	// IsNil indicates the struct or list is nil/empty. No Start/End tokens follow.
	IsNil bool

	// Len is the list length (for TokenListStart).
	Len int

	// Map-related fields (for TokenMapStart and TokenMapEntry)
	// KeyType is the type of map keys.
	KeyType field.Type
	// ValueType is the type of map values.
	ValueType field.Type
	// Key holds the map key value (uses same encoding as data/Bytes).
	Key uint64
	// KeyBytes holds string/bytes map keys.
	KeyBytes []byte
}

// Bool returns the boolean value. Only valid when Type == FTBool.
func (t Token) Bool() bool { return t.data != 0 }

// Int8 returns the int8 value. Only valid when Type == FTInt8.
func (t Token) Int8() int8 { return int8(t.data) }

// Int16 returns the int16 value. Only valid when Type == FTInt16.
func (t Token) Int16() int16 { return int16(t.data) }

// Int32 returns the int32 value. Only valid when Type == FTInt32.
func (t Token) Int32() int32 { return int32(t.data) }

// Int64 returns the int64 value. Only valid when Type == FTInt64.
func (t Token) Int64() int64 { return int64(t.data) }

// Uint8 returns the uint8 value. Only valid when Type == FTUint8.
func (t Token) Uint8() uint8 { return uint8(t.data) }

// Uint16 returns the uint16 value. Only valid when Type == FTUint16.
func (t Token) Uint16() uint16 { return uint16(t.data) }

// Uint32 returns the uint32 value. Only valid when Type == FTUint32.
func (t Token) Uint32() uint32 { return uint32(t.data) }

// Uint64 returns the uint64 value. Only valid when Type == FTUint64.
func (t Token) Uint64() uint64 { return t.data }

// Float32 returns the float32 value. Only valid when Type == FTFloat32.
func (t Token) Float32() float32 { return math.Float32frombits(uint32(t.data)) }

// Float64 returns the float64 value. Only valid when Type == FTFloat64.
func (t Token) Float64() float64 { return math.Float64frombits(t.data) }

// String returns the string value using unsafe.String for zero-copy conversion.
// Only valid when Type == FTString.
func (t Token) String() string {
	if len(t.Bytes) == 0 {
		return ""
	}
	return unsafe.String(&t.Bytes[0], len(t.Bytes))
}

// SetBool sets the boolean value in the token.
func (t *Token) SetBool(v bool) {
	if v {
		t.data = 1
	} else {
		t.data = 0
	}
}

// SetInt8 sets the int8 value in the token.
func (t *Token) SetInt8(v int8) { t.data = uint64(v) }

// SetInt16 sets the int16 value in the token.
func (t *Token) SetInt16(v int16) { t.data = uint64(v) }

// SetInt32 sets the int32 value in the token.
func (t *Token) SetInt32(v int32) { t.data = uint64(v) }

// SetInt64 sets the int64 value in the token.
func (t *Token) SetInt64(v int64) { t.data = uint64(v) }

// SetUint8 sets the uint8 value in the token.
func (t *Token) SetUint8(v uint8) { t.data = uint64(v) }

// SetUint16 sets the uint16 value in the token.
func (t *Token) SetUint16(v uint16) { t.data = uint64(v) }

// SetUint32 sets the uint32 value in the token.
func (t *Token) SetUint32(v uint32) { t.data = uint64(v) }

// SetUint64 sets the uint64 value in the token.
func (t *Token) SetUint64(v uint64) { t.data = v }

// SetFloat32 sets the float32 value in the token.
func (t *Token) SetFloat32(v float32) { t.data = uint64(math.Float32bits(v)) }

// SetFloat64 sets the float64 value in the token.
func (t *Token) SetFloat64(v float64) { t.data = math.Float64bits(v) }

// Map key accessor methods

// KeyBool returns the boolean key value.
func (t Token) KeyBool() bool { return t.Key != 0 }

// KeyInt8 returns the int8 key value.
func (t Token) KeyInt8() int8 { return int8(t.Key) }

// KeyInt16 returns the int16 key value.
func (t Token) KeyInt16() int16 { return int16(t.Key) }

// KeyInt32 returns the int32 key value.
func (t Token) KeyInt32() int32 { return int32(t.Key) }

// KeyInt64 returns the int64 key value.
func (t Token) KeyInt64() int64 { return int64(t.Key) }

// KeyUint8 returns the uint8 key value.
func (t Token) KeyUint8() uint8 { return uint8(t.Key) }

// KeyUint16 returns the uint16 key value.
func (t Token) KeyUint16() uint16 { return uint16(t.Key) }

// KeyUint32 returns the uint32 key value.
func (t Token) KeyUint32() uint32 { return uint32(t.Key) }

// KeyUint64 returns the uint64 key value.
func (t Token) KeyUint64() uint64 { return t.Key }

// KeyFloat32 returns the float32 key value.
func (t Token) KeyFloat32() float32 { return math.Float32frombits(uint32(t.Key)) }

// KeyFloat64 returns the float64 key value.
func (t Token) KeyFloat64() float64 { return math.Float64frombits(t.Key) }

// KeyString returns the string key value.
func (t Token) KeyString() string {
	if len(t.KeyBytes) == 0 {
		return ""
	}
	return unsafe.String(&t.KeyBytes[0], len(t.KeyBytes))
}

// SetKeyBool sets a boolean key.
func (t *Token) SetKeyBool(v bool) {
	if v {
		t.Key = 1
	} else {
		t.Key = 0
	}
}

// SetKeyInt8 sets an int8 key.
func (t *Token) SetKeyInt8(v int8) { t.Key = uint64(v) }

// SetKeyInt16 sets an int16 key.
func (t *Token) SetKeyInt16(v int16) { t.Key = uint64(v) }

// SetKeyInt32 sets an int32 key.
func (t *Token) SetKeyInt32(v int32) { t.Key = uint64(v) }

// SetKeyInt64 sets an int64 key.
func (t *Token) SetKeyInt64(v int64) { t.Key = uint64(v) }

// SetKeyUint8 sets a uint8 key.
func (t *Token) SetKeyUint8(v uint8) { t.Key = uint64(v) }

// SetKeyUint16 sets a uint16 key.
func (t *Token) SetKeyUint16(v uint16) { t.Key = uint64(v) }

// SetKeyUint32 sets a uint32 key.
func (t *Token) SetKeyUint32(v uint32) { t.Key = uint64(v) }

// SetKeyUint64 sets a uint64 key.
func (t *Token) SetKeyUint64(v uint64) { t.Key = v }

// SetKeyFloat32 sets a float32 key.
func (t *Token) SetKeyFloat32(v float32) { t.Key = uint64(math.Float32bits(v)) }

// SetKeyFloat64 sets a float64 key.
func (t *Token) SetKeyFloat64(v float64) { t.Key = math.Float64bits(v) }

// YieldToken is the callback type for Walk iteration.
// Returns false to stop iteration early.
type YieldToken func(Token) bool

// Walker is a function that walks tokens and yields them to a callback.
// This is the type accepted by Ingest.
type Walker func(YieldToken)
