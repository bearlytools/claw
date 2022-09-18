package reflect

import (
	"github.com/bearlytools/claw/languages/go/mapping"
	"github.com/bearlytools/claw/languages/go/reflect/enums"
	"github.com/bearlytools/claw/languages/go/reflect/internal/interfaces"
	"github.com/bearlytools/claw/languages/go/reflect/internal/value"
	"github.com/bearlytools/claw/languages/go/structs"
)

// ClawStruct indicates that the type represents a Claw Struct.
type ClawStruct interface {
	// ClawReflect returns the reflect.Struct for a Claw Struct.
	ClawReflect() Struct
}

// PackageDescr is used to describe a claw package and its contents.
type PackageDescr = interfaces.PackageDescr

// Struct represents a concrete value of a Claw Struct.
type Struct = interfaces.Struct

// StructDescrs gives access to the descriptions of a package's Claw Structs.
type StructDescrs = interfaces.StructDescrs

// StructDescr is a descriptor of a Claw Struct.
type StructDescr = interfaces.StructDescr

// FieldDescr is a descriptor of a Claw Struct field.
type FieldDescr = interfaces.FieldDescr

// EnumGroup describes a single set of enum values defined in a Claw package.
type EnumGroup = interfaces.EnumGroup

// EnumValueDescr is a descriptor for an enumerated value.
type EnumValueDescr = interfaces.Enum

// Enum is the refection interface for a concrete enum value.
type Enum = interfaces.Enum

// EnumGroups describes enum groups in a package.
type EnumGroups = interfaces.EnumGroups

// List provides access to one of Claw's list types from the int family, uint family,
// lists of bytes/string or list of structs.
type List = interfaces.List

type XXXPackageDescrImpl = value.PackageDescrImpl
type XXXEnumGroupsImpl = enums.EnumGroupsImpl
type XXXEnumGroupImpl = enums.EnumGroupImpl
type XXXEnumValueDescrImpl = enums.EnumImpl
type XXXStructDescrsImpl = value.StructDescrsImpl
type XXXStructDescrImpl = value.StructDescrImpl
type XXXFieldDescrImpl = value.FieldDescrImpl

func XXXNewStruct(v *structs.Struct) Struct {
	return value.XXXNewStruct(v)
}

/*
func XXXNewStructDescrsImpl(structs []StructDescr) StructDescrs {
	return value.StructDescrsImpl{Descrs: structs}
}
*/

func XXXNewStructDescrImpl(m *mapping.Map) StructDescr {
	return value.NewStructDescrImpl(m)
}
