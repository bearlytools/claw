package reflect

import (
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

// StructDescrs gives access to the descriptions of a package's struct objects.
type StructDescrs = interfaces.StructDescr

// EnumGroup describes a single set of enum values defined in a claw package.
type EnumGroup = interfaces.EnumGroup

// EnumValueDescr describes an enumerated value.
type EnumValueDescr = interfaces.Enum

// Enum is the refection interface for a concrete enum value.
type Enum = interfaces.Enum

// EnumGroups describes enum groups in a package.
type EnumGroups = interfaces.EnumGroups

// List provides access to one of Claw's list types from the int family, uint family,
// lists of bytes/string or list of structs.
type List = interfaces.List

type XXXPackageDescrImpl = value.PackageDescrImpl
type XXXEnumGroupsImpl = value.EnumGroupsImpl
type XXXEnumGroupImpl = value.EnumGroupImpl
type XXXEnumValueDescrImpl = value.EnumImpl
type XXXStructDescrImpl = value.StructDescrImpl
type XXXFieldDescrImpl = value.FieldDescrImpl

func XXXNewStruct(v *structs.Struct) Struct {
	return value.XXXNewStruct(v)
}
