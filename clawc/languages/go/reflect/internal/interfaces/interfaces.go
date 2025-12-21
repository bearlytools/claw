package interfaces

import (
	"github.com/bearlytools/claw/clawc/internal/pragma"
	"github.com/bearlytools/claw/clawc/languages/go/field"
	"golang.org/x/exp/constraints"
)

type doNotImplement pragma.DoNotImplement

// Number represents all int, uint and float types.
type Number interface {
	constraints.Integer | constraints.Float
}

// PackageDescr is used to describe a claw package and its contents.
type PackageDescr interface {
	// PackageName returns the name of the package.
	PackageName() string
	// FullPath returns the full path of the package.
	FullPath() string
	// Imports is a list of imported claw files.
	Imports() []PackageDescr
	// Enums is a list of the Enum declarations.
	Enums() EnumGroups
	// Structs is a list of the top-level message declarations.
	Structs() StructDescrs

	doNotImplement
}

// EnumGroup describes a single set of enum values defined in a claw package.
type EnumGroup interface {
	// Name is the name of the EnumGroup.
	Name() string
	// Len reports the number of enum values.
	// TODO(jdoak): Change this to be uint16
	Len() int
	// Get returns the ith EnumValue. It panics if out of bounds.
	Get(i uint16) Enum
	// ByName returns the EnumValue for an enum named s.
	// It returns nil if not found.
	ByName(s string) Enum
	// ByValue gets the Enum by its value.
	ByValue(i uint16) Enum
	// Size returns the size in bits of the enumerator.
	Size() uint8 // Either 8 or 16

	doNotImplement
}

// EnumGroups describes enum groups in a package.
type EnumGroups interface {
	// Len reports the number of enum types.
	Len() int
	// Get returns the ith EnumDescriptor. It panics if out of bounds.
	Get(i int) EnumGroup
	// ByName returns the EnumDescriptor for an enum named s.
	// It returns nil if not found.
	ByName(s string) EnumGroup

	doNotImplement
}

// Enum describes an enumerated value.
type Enum interface {
	// Name returns the name of the Enum value.
	Name() string
	// Number returns the enum number value.
	Number() uint16
	// Size is the size in bits that the enum is in. This is either 8 or 16.
	Size() uint8

	doNotImplement
}

// TODO(jdoak): Remove this?
// Enum is the refection interface for a concrete enum value.
/*
type Enum interface {
	Descriptor() EnumValueDescr
	// Number returns the number value of the enum. This value could be a sized for
	// uint8 or uint16, to determine the enumerator size, use .Size().
	Number() uint16
	// String returns the string representation of the enumerator.
	String() string
	// Size returns the size in bits of the enumerator.
	Size() uint8 // Either 8 or 16

	doNotImplement
}
*/

// List provides access to one of Claw's list types from the int family, uint family,
// lists of bytes/string or list of structs.
type List interface {
	// Type returns the list's type.
	Type() field.Type

	// Len reports the number of entries in the List.
	// Get, Set, and Truncate panic with out of bound indexes.
	Len() int

	// Get retrieves the value at the given index.
	// It never returns an invalid value.
	Get(int) Value

	// Set stores a value for the given index. If Value is not a valid element
	// type for this list, this will panic.
	Set(int, Value)

	// Append appends the provided Value to the end of the list. If the Value
	// is not a valid element type for this list, this will panic.
	Append(Value)

	// New returns a newly allocated and mutable empty Struct value. This can
	// only be used if the List represents a list of Struct values.
	New() Struct

	doNotImplement
}

// Struct represents a Struct
type Struct interface {
	doNotImplement

	// Descriptor returns message descriptor, which contains only the Claw
	// type information for the message.
	Descriptor() StructDescr

	// New returns a newly allocated and mutable empty Struct.
	New() Struct

	// Range iterates over every populated field in an undefined order,
	// calling f for each field descriptor and value encountered.
	// Range returns immediately if f returns false.
	// While iterating, mutating operations may only be performed
	// on the current field descriptor. If the Value is nil, that means the
	// field's value was not set.
	Range(f func(FieldDescr, Value) bool)

	// Has reports whether a field is populated. This always works for list type fields
	// and Struct fields. With scalar values, this can be interpreted in two ways. If
	// NoZeroValueCompression is on, then this will report if the value has been set or not.
	// If it hasn't, this will report true for all scalar values, string and bytes types, as
	// there is no way to determine if the zero value was set.
	Has(FieldDescr) bool

	// Clear clears the field such that a subsequent Has call reports false.
	//
	// Clearing an extension field clears both the extension type and value
	// associated with the given field number.
	//
	// Clear is a mutating operation and unsafe for concurrent use.
	Clear(FieldDescr)

	// Get retrieves the value for a field.
	//
	// For unpopulated scalars, it returns the default value, where
	// the default value of a bytes scalar is guaranteed to be a copy.
	// For unpopulated composite types, it returns an empty, read-only view
	// of the value; to obtain a mutable reference, use Mutable.
	Get(FieldDescr) Value

	// Set stores the value for a field.
	//
	// For a field belonging to a oneof, it implicitly clears any other field
	// that may be currently set within the same oneof.
	// For extension fields, it implicitly stores the provided ExtensionType.
	// When setting a composite type, it is unspecified whether the stored value
	// aliases the source's memory in any way. If the composite value is an
	// empty, read-only value, then it panics.
	//
	// Set is a mutating operation and unsafe for concurrent use.
	Set(FieldDescr, Value)

	// NewField returns a new value that is assignable to the field
	// for the given descriptor. For scalars, this returns the default value.
	// For lists and Structs, this returns a new, empty, mutable value.
	NewField(FieldDescr) Value
}

// StructDescrs gives access to the descriptions of a package's struct objects.
type StructDescrs interface {
	// Len reports the number of messages.
	Len() int
	// Get returns the ith StructDescr. It panics if out of bounds.
	Get(i int) StructDescr
	// ByName returns the StructDescr for a Struct named s.
	// It returns nil if not found.
	ByName(name string) StructDescr

	doNotImplement
}

// StructDescr describes a claw struct object.
type StructDescr interface {
	doNotImplement

	// StructName is the name of the struct.
	StructName() string
	// Package will be return name package name this struct was defined in.
	Package() string
	// FullPath will return the full path of the package as used in Go import statements.
	FullPath() string
	// Fields will return a list of field descriptions.
	Fields() []FieldDescr
	// FieldDescrByName returns the FieldDescr by the name of the field. If the field
	// is not found, this will be nil.
	FieldDescrByName(name string) FieldDescr
	// FieldDescrByIndex returns the FieldDescr by index. If the index is out of bounds this
	// will panic.
	FieldDescrByIndex(index int) FieldDescr
	// New creates a new empty Struct described by this StructDescr.
	New() Struct
}

// FieldDescr describes a field in a claw Struct.
type FieldDescr interface {
	// Name is the name of the field as described in the .claw file.
	Name() string
	// Type is the type of field.
	Type() field.Type
	// FieldNum is the field number inside the Struct.
	FieldNum() uint16
	// IsEnum indicates if this field is an enumerator.
	IsEnum() bool
	// EnumGroup returns the EnumGroup associated with this field.
	// This is only valid if IsEnum() is true.
	EnumGroup() EnumGroup
	// ItemType returns the name of a Struct if the field is a list of Struct values.
	// If not, this panics.
	ItemType() string
}

// Value represents a read-only Claw value. This can be used to retrieve a value or
// set a value.
type Value interface {
	// Bool returns the boolean value stored in Value. If Value is not a bool type, this will panic.
	Bool() bool
	// Bytes returns the Bytes value stored in Value. If Value is not a Bytes type, this will panic.
	Bytes() []byte
	// Enum returns the enumerated value stored in Value. If Value is not an Enum type, this will panic.
	Enum() Enum
	// Float returns the Float value stored in Value. If Value is not a Float type, this will panic.
	Float() float64
	// Int returns the integer value stored in Value. If Value is not an integer type, this will panic.
	Int() int64
	// Any decodes the value into the any type. If the value isn't valid, this panics.
	Any() any
	// String returns the string value stored in Value. If Value is not a string type, this will panic.
	String() string
	// Uint returns the unsigned integer value stored in Value. If Value is not an unsigned integer type, this will panic.
	Uint() uint64
	// List returns the List value stored in Value. If Value is not some list type, this will panic.
	List() List
	// Struct returns the Struct value stored in Value. If Value is not a Struct type, this will panic.
	Struct() Struct
}
