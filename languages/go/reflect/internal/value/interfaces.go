package value

import (
	"github.com/bearlytools/claw/internal/field"
	"github.com/bearlytools/claw/languages/go/structs"
	"golang.org/x/exp/constraints"
)

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
	Structs() []StructDescr

	doNotImplement
}

// EnumGroup describes a single set of enum values defined in a claw package.
type EnumGroup interface {
	// Name is the name of the EnumGroup.
	Name() string
	// Len reports the number of enum values.
	Len() int
	// Get returns the ith EnumValue. It panics if out of bounds.
	Get(i int) EnumValueDescr
	// ByName returns the EnumValue for an enum named s.
	// It returns nil if not found.
	ByName(s string) EnumValueDescr
	// ByValue gets the Enum by its value.
	ByValue(i int) EnumValueDescr
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

// EnumValueDescr describes an enumerated value.
type EnumValueDescr interface {
	// Name returns the name of the Enum value.
	Name() string
	// Number returns the enum number value.
	Number() uint16

	doNotImplement
}

// Enum is the refection interface for a concrete enum value.
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

type Struct interface {
	doNotImplement

	// Descriptor returns message descriptor, which contains only the protobuf
	// type information for the message.
	Descriptor() StructDescr

	// New returns a newly allocated and mutable empty Struct.
	New() Struct

	// Range iterates over every populated field in an undefined order,
	// calling f for each field descriptor and value encountered.
	// Range returns immediately if f returns false.
	// While iterating, mutating operations may only be performed
	// on the current field descriptor.
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

	realType() *structs.Struct
}

// StructDescrs gives access to the descriptions of a package's struct objects.
type StructDescrs interface {
	// Len reports the number of messages.
	Len() int
	// Get returns the ith StructDescr. It panics if out of bounds.
	Get(i int) StructDescr
	// ByName returns the StructDescr for a Struct named s.
	// It returns nil if not found.
	ByName(s string) StructDescr

	doNotImplement
}

// StructDescr describes a claw struct object.
type StructDescr interface {
	// StructName is the name of the struct.
	StructName() string
	// Package will be return name package name this struct was defined in.
	Package() string
	// FullPath will return the full path of the package as used in Go import statements.
	FullPath() string
	// Fields will return a list of field descriptions.
	Fields() []FieldDescr

	doNotImplement
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
